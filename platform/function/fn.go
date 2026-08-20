package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
	"github.com/crossplane/function-sdk-go/resource/composite"
	"github.com/crossplane/function-sdk-go/response"
	corev1 "k8s.io/api/core/v1"
)

// Function renders the managed resources for the Grafana vending APIs.
type Function struct {
	fnv1.UnimplementedFunctionRunnerServiceServer

	log logging.Logger
}

// RunFunction renders desired composed resources from a platform claim.
func (f *Function) RunFunction(_ context.Context, req *fnv1.RunFunctionRequest) (*fnv1.RunFunctionResponse, error) {
	rsp := response.To(req, response.DefaultTTL)

	xr, err := request.GetObservedCompositeResource(req)
	if err != nil {
		response.Fatal(rsp, errors.Wrap(err, "cannot read observed composite resource"))
		return rsp, nil
	}

	observed, err := request.GetObservedComposedResources(req)
	if err != nil {
		response.Fatal(rsp, errors.Wrap(err, "cannot read observed composed resources"))
		return rsp, nil
	}

	config := map[string]any{}
	if req.GetInput() != nil {
		config = req.GetInput().AsMap()
	}

	content := xr.Resource.UnstructuredContent()
	kind, _ := content["kind"].(string)
	var desired map[resource.Name]*resource.DesiredComposed
	switch kind {
	case "GrafanaCloudStackRequest":
		desired, err = renderStack(content, observed, config)
		if err == nil {
			if err = response.SetDesiredCompositeResource(rsp, desiredStackStatus(content, observed, config)); err != nil {
				err = errors.Wrap(err, "cannot set desired composite status")
			}
		}
	case "GrafanaCustomRoleBinding":
		desired, err = renderRoleBinding(content)
	case "GrafanaTeamAccess":
		desired, err = renderTeamAccess(content, observed)
	case "GrafanaContentAccessPolicy":
		desired, err = renderContentAccessPolicy(content)
	default:
		err = errors.Errorf("unsupported composite kind %q", kind)
	}
	if err != nil {
		response.Fatal(rsp, err)
		return rsp, nil
	}
	markObservedResourcesReady(desired, observed)
	if err := response.SetDesiredComposedResources(rsp, desired); err != nil {
		response.Fatal(rsp, errors.Wrap(err, "cannot set desired composed resources"))
	}

	return rsp, nil
}

func markObservedResourcesReady(desired map[resource.Name]*resource.DesiredComposed, observed map[resource.Name]resource.ObservedComposed) {
	for name, desiredResource := range desired {
		if !observedReady(observed, name) {
			continue
		}
		desiredResource.Ready = resource.ReadyTrue
	}
}

func observedExists(observed map[resource.Name]resource.ObservedComposed, name resource.Name) bool {
	r, ok := observed[name]
	return ok && r.Resource != nil
}

func observedReady(observed map[resource.Name]resource.ObservedComposed, name resource.Name) bool {
	r, ok := observed[name]
	if !ok || r.Resource == nil {
		return false
	}
	return r.Resource.GetCondition(xpv2.TypeReady).Status == corev1.ConditionTrue
}

// admitStaged merges a staged group into the desired set once its precondition
// holds, and keeps any member that already exists regardless. The second rule
// is the important one: a desired resource that disappears is deleted, so a
// stack that briefly reports not-Ready must not take its own content with it.
func admitStaged(desired, staged map[resource.Name]*resource.DesiredComposed, observed map[resource.Name]resource.ObservedComposed, admit bool) {
	for name, r := range staged {
		if admit || observedExists(observed, name) {
			desired[name] = r
		}
	}
}

var managementPolicies = []any{"Create", "Observe", "Update", "LateInitialize"}

func renderStack(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	metadata, _ := xr["metadata"].(map[string]any)
	spec, _ := xr["spec"].(map[string]any)

	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	displayName, _ := spec["displayName"].(string)
	slug, _ := spec["slug"].(string)
	region, _ := spec["region"].(string)
	usage, _ := spec["usage"].(string)
	profile := stringValue(spec, "profile", "standard")
	if name == "" || namespace == "" || displayName == "" || slug == "" || region == "" || usage == "" {
		return nil, errors.New("stack claim must set metadata name and namespace plus displayName, slug, region, and usage")
	}
	if name != slug {
		return nil, errors.New("stack claim metadata.name must match spec.slug")
	}
	settings := configuredPlatformSettings(config)

	reconciliation, _ := spec["reconciliation"].(map[string]any)
	dashboardMode := stringValue(reconciliation, "dashboards", "createOnly")
	homeMode := stringValue(reconciliation, "homePreference", "createOnly")
	if !oneOf(dashboardMode, "enforced", "createOnly") {
		return nil, errors.Errorf("unsupported dashboard reconciliation mode %q", dashboardMode)
	}
	if !oneOf(homeMode, "enforced", "createOnly") {
		return nil, errors.Errorf("unsupported home preference reconciliation mode %q", homeMode)
	}

	stackDetailsSecret := slug + "-stack-details"
	tokenSecret := slug + "-token"
	credentialSecret := slug + "-provider-credentials"
	providerConfig := slug
	outputPath := fmt.Sprintf("%s/%s/%s/%s", settings.outputSecretPrefix, region, usage, slug)
	telemetryOutputPath := ""
	if telemetryAccessEnabled(spec) {
		telemetryOutputPath = outputPath + "/telemetry-publisher"
	}

	// The foundation goes to the organization credential and can be created
	// immediately. Stack-local resources cannot: the baseline content and every
	// optional domain reach the stack through the per-stack ProviderConfig, whose
	// credential Secret does not exist until the admin token has been minted and
	// published, and plugin installation calls the stack's own API. Rendering
	// them before the stack serves traffic does not fail the vend, because
	// Crossplane retries, but it does emit an error per resource per reconcile:
	// a healthy first vend then produces tens of warning events, a real failure
	// is indistinguishable from that noise, and the churn can trip the
	// composite's watch circuit breaker. So each group waits for the condition it
	// actually depends on.
	desired := map[resource.Name]*resource.DesiredComposed{}
	whenStackServes := map[resource.Name]*resource.DesiredComposed{}
	whenCredentialsPublished := map[resource.Name]*resource.DesiredComposed{}
	desired["stack"] = newDesired("cloud.grafana.m.crossplane.io/v1alpha1", "Stack", namespace, slug,
		map[string]any{"crossplane.io/external-name": slug},
		map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider": map[string]any{
				"name": displayName, "slug": slug, "regionSlug": region,
				"deleteProtection": true, "waitForReadiness": true, "waitForReadinessTimeout": "10m0s",
				"labels": map[string]any{"provisioner": "crossplane", "usage": usage, "profile": profile},
			},
			"providerConfigRef":          map[string]any{"kind": "ProviderConfig", "name": settings.organizationProviderConfigName},
			"writeConnectionSecretToRef": map[string]any{"name": stackDetailsSecret},
		})
	desired["stack-service-account"] = newDesired("cloud.grafana.m.crossplane.io/v1alpha1", "StackServiceAccount", namespace, slug+"-admin", nil,
		map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider": map[string]any{
				"cloudStackRef": map[string]any{"name": slug},
				"name":          "Grafana vending controller", "role": "Admin", "isDisabled": false,
			},
			"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": settings.organizationProviderConfigName},
		})
	if serviceAccountID := observedString(observed, "stack-service-account", "status.atProvider.id"); serviceAccountID != "" {
		desired["stack-token"] = newDesired("cloud.grafana.m.crossplane.io/v1alpha1", "StackServiceAccountRotatingToken", namespace, slug+"-admin", nil,
			map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider": map[string]any{
					"namePrefix": "grafana-vending-", "secondsToLive": 2592000,
					"earlyRotationWindowSeconds": 604800, "deleteOnDestroy": false,
					"stackSlug": slug, "serviceAccountId": serviceAccountID,
				},
				"providerConfigRef":          map[string]any{"kind": "ProviderConfig", "name": settings.organizationProviderConfigName},
				"writeConnectionSecretToRef": map[string]any{"name": tokenSecret},
			})
	}
	desired["provider-config"] = newDesired("grafana.m.crossplane.io/v1beta1", "ProviderConfig", namespace, providerConfig, nil,
		map[string]any{
			"credentials": map[string]any{
				"source":    "Secret",
				"secretRef": map[string]any{"name": credentialSecret, "namespace": namespace, "key": "credentials"},
			},
			"stackSecretRef": map[string]any{"name": stackDetailsSecret, "namespace": namespace},
		})

	baselineEnabled := true
	if baseline, ok := spec["baselineDashboards"].(map[string]any); ok {
		if enabled, exists := baseline["enabled"].(bool); exists {
			baselineEnabled = enabled
		}
	}
	if baselineEnabled {
		for _, dashboard := range []struct {
			name, title, uid, description string
		}{
			{name: "billing", title: "Billing and usage", uid: "vending-billing", description: "Generic billing and usage overview."},
			{name: "endpoints", title: "Telemetry endpoints", uid: "vending-endpoints", description: "Generic telemetry endpoint reference."},
			{name: "homepage", title: "Stack home", uid: "vending-home", description: "This stack is continuously reconciled from Git through Crossplane."},
		} {
			folderName := dashboard.name + "-folder"
			dashboardName := dashboard.name + "-dashboard"
			whenCredentialsPublished[resource.Name(folderName)] = newDesired("oss.grafana.m.crossplane.io/v1alpha1", "Folder", namespace, slug+"-"+dashboard.name,
				map[string]any{"crossplane.io/external-name": dashboard.uid + "-folder"},
				map[string]any{
					"managementPolicies": managementPolicies,
					"forProvider":        map[string]any{"title": dashboard.title, "uid": dashboard.uid + "-folder"},
					"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": providerConfig},
				})
			dashboardSpec := map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider": map[string]any{
					"folderRef": map[string]any{"name": slug + "-" + dashboard.name}, "overwrite": true,
				},
				"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": providerConfig},
			}
			ownedParameters(dashboardSpec, dashboardMode)["configJson"] = dashboardJSON(dashboard.title, dashboard.uid, dashboard.description)
			whenCredentialsPublished[resource.Name(dashboardName)] = newDesired("oss.grafana.m.crossplane.io/v1alpha1", "Dashboard", namespace, slug+"-"+dashboard.name,
				map[string]any{"crossplane.io/external-name": dashboard.uid}, dashboardSpec)
		}

		preferencesSpec := map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider":        map[string]any{},
			"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": providerConfig},
		}
		ownedParameters(preferencesSpec, homeMode)["homeDashboardUid"] = "vending-home"
		whenCredentialsPublished["organization-preferences"] = newDesired("oss.grafana.m.crossplane.io/v1alpha1", "OrganizationPreferences", namespace, slug+"-preferences", nil, preferencesSpec)
	}

	changeReference, _ := spec["changeReference"].(string)
	configurationItemReference, _ := spec["configurationItemReference"].(string)
	outputDocument := fmt.Sprintf(`{{ $token := index . "attribute.key" | toString }}{"stack_name":%q,"stack_slug":%q,"stack_url":%q,"stack_region":%q,"usage":%q,"change_reference":%q,"configuration_item_reference":%q,"stack_service_account_token":{{ $token | toJson }},"telemetry_access_policy_secret_path":%q}`, displayName, slug, "https://"+slug+".grafana.net", region, usage, changeReference, configurationItemReference, telemetryOutputPath)
	desired["credentials"] = newDesired("external-secrets.io/v1alpha1", "PushSecret", namespace, slug+"-credentials", nil,
		map[string]any{
			"refreshInterval": "1h", "updatePolicy": "Replace", "deletionPolicy": "None",
			"secretStoreRefs": []any{settings.secretStoreReference()},
			"selector":        map[string]any{"secret": map[string]any{"name": tokenSecret}},
			"template": map[string]any{
				"engineVersion": "v2", "mergePolicy": "Replace",
				"data": map[string]any{"vending.json": outputDocument},
			},
			"data": []any{map[string]any{
				"match": map[string]any{
					"secretKey": "vending.json",
					"remoteRef": map[string]any{"remoteKey": outputPath},
				},
				"metadata": map[string]any{
					"apiVersion": "kubernetes.external-secrets.io/v1alpha1", "kind": "PushSecretMetadata",
					"spec": map[string]any{"secretPushFormat": "string"},
				},
			}},
		})
	desired["instance-credentials"] = newDesired("external-secrets.io/v1", "ExternalSecret", namespace, credentialSecret, nil,
		map[string]any{
			"refreshInterval": "1h",
			"secretStoreRef":  settings.secretStoreReference(),
			"target": map[string]any{
				"name": credentialSecret, "creationPolicy": "Owner", "deletionPolicy": "Retain",
				"template": map[string]any{
					"engineVersion": "v2",
					"data":          map[string]any{"credentials": `{"auth":{{ .stackServiceAccountToken | toJson }},"url":{{ .stackURL | toJson }}}`},
				},
			},
			"data": []any{
				map[string]any{"secretKey": "stackServiceAccountToken", "remoteRef": map[string]any{"key": outputPath, "property": "stack_service_account_token"}},
				map[string]any{"secretKey": "stackURL", "remoteRef": map[string]any{"key": outputPath, "property": "stack_url"}},
			},
		})

	if err := addTelemetryAccess(desired, observed, namespace, slug, region, telemetryOutputPath, spec, settings); err != nil {
		return nil, err
	}
	if err := addPluginInstallations(whenStackServes, namespace, slug, spec, settings.organizationProviderConfigName); err != nil {
		return nil, err
	}

	if err := addSSO(whenCredentialsPublished, namespace, providerConfig, spec, config); err != nil {
		return nil, err
	}
	if err := addMonthlyReport(whenCredentialsPublished, namespace, providerConfig, slug, spec, baselineEnabled); err != nil {
		return nil, err
	}
	if err := addIncidentIntegration(whenCredentialsPublished, namespace, providerConfig, slug, spec, config); err != nil {
		return nil, err
	}

	stackServes := observedReady(observed, "stack")
	admitStaged(desired, whenStackServes, observed, stackServes)
	admitStaged(desired, whenCredentialsPublished, observed, stackServes && observedReady(observed, "instance-credentials"))

	return desired, nil
}

func desiredStackStatus(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) *resource.Composite {
	spec, _ := xr["spec"].(map[string]any)
	slug, _ := spec["slug"].(string)
	region, _ := spec["region"].(string)
	usage, _ := spec["usage"].(string)
	settings := configuredPlatformSettings(config)
	status := map[string]any{
		"outputSecretPath": fmt.Sprintf("%s/%s/%s/%s", settings.outputSecretPrefix, region, usage, slug),
	}
	if telemetryAccessEnabled(spec) {
		status["telemetrySecretPath"] = fmt.Sprintf("%s/%s/%s/%s/telemetry-publisher", settings.outputSecretPrefix, region, usage, slug)
	}
	stack := map[string]any{}
	if id := observedString(observed, "stack", "status.atProvider.id"); id != "" {
		stack["id"] = id
	}
	if url := observedString(observed, "stack", "status.atProvider.url"); url != "" {
		stack["url"] = url
	}
	if len(stack) > 0 {
		status["stack"] = stack
	}
	desired := &resource.Composite{Resource: composite.New()}
	desired.Resource.SetUnstructuredContent(map[string]any{"status": status})
	return desired
}

type platformSettings struct {
	organizationProviderConfigName string
	outputSecretPrefix             string
	secretStoreName                string
	secretStoreKind                string
}

func configuredPlatformSettings(config map[string]any) platformSettings {
	spec, _ := config["spec"].(map[string]any)
	store, _ := spec["secretStoreRef"].(map[string]any)
	return platformSettings{
		organizationProviderConfigName: stringValue(spec, "organizationProviderConfigName", "grafana-cloud-org"),
		outputSecretPrefix:             stringValue(spec, "outputSecretPrefix", "/platform/grafana-cloud/stacks"),
		secretStoreName:                stringValue(store, "name", "grafana-vending-secrets"),
		secretStoreKind:                stringValue(store, "kind", "SecretStore"),
	}
}

func (s platformSettings) secretStoreReference() map[string]any {
	return map[string]any{"name": s.secretStoreName, "kind": s.secretStoreKind}
}

func observedString(observed map[resource.Name]resource.ObservedComposed, name resource.Name, fieldPath string) string {
	r, ok := observed[name]
	if !ok || r.Resource == nil {
		return ""
	}
	v, err := r.Resource.GetString(fieldPath)
	if err != nil {
		return ""
	}
	return v
}

func observedIntegerString(observed map[resource.Name]resource.ObservedComposed, name resource.Name, fieldPath string) string {
	r, ok := observed[name]
	if !ok || r.Resource == nil {
		return ""
	}
	v, err := r.Resource.GetInteger(fieldPath)
	if err != nil {
		return ""
	}
	return strconv.FormatInt(v, 10)
}

func newDesired(apiVersion, kind, namespace, name string, annotations, spec map[string]any) *resource.DesiredComposed {
	r := composed.New()
	metadata := map[string]any{"name": name, "namespace": namespace}
	if len(annotations) > 0 {
		metadata["annotations"] = annotations
	}
	r.SetUnstructuredContent(map[string]any{
		"apiVersion": apiVersion,
		"kind":       kind,
		"metadata":   metadata,
		"spec":       spec,
	})
	return &resource.DesiredComposed{Resource: r}
}

func dashboardJSON(title, uid, description string) string {
	dashboard := map[string]any{
		"title": title, "uid": uid, "schemaVersion": 41, "timezone": "browser",
		"tags": []string{"crossplane", "grafana-vending"},
		"panels": []any{map[string]any{
			"type": "text", "title": title, "gridPos": map[string]any{"h": 6, "w": 24, "x": 0, "y": 0},
			"options": map[string]any{"mode": "markdown", "content": description},
		}},
		"time": map[string]any{"from": "now-6h", "to": "now"},
	}
	b, _ := json.Marshal(dashboard)
	return string(b)
}
