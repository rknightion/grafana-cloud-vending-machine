package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

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
	now func() time.Time
}

type compositeRenderer struct {
	render      func(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) (map[resource.Name]*resource.DesiredComposed, error)
	gateOnStack bool
	implemented bool
}

const reconcileTimeConfigKey = "_reconcileTime"

const resolvedStackProfileConfigKey = "_resolvedStackProfile"

var compositeRenderers = map[string]compositeRenderer{
	"GrafanaAlertingRouting":       {render: renderAlertingRouting, gateOnStack: true, implemented: alertingRoutingRendererImplemented},
	"GrafanaOnCall":                {render: renderOnCall, gateOnStack: true, implemented: onCallRendererImplemented},
	"GrafanaCloudIntegrations":     {render: renderCloudIntegrations, gateOnStack: true, implemented: cloudIntegrationsRendererImplemented},
	"GrafanaPDC":                   {render: renderPDC, gateOnStack: true, implemented: pdcRendererImplemented},
	"GrafanaServiceAccounts":       {render: renderServiceAccounts, gateOnStack: true, implemented: serviceAccountsRendererImplemented},
	"GrafanaFrontendObservability": {render: renderFrontendObservability, gateOnStack: true, implemented: frontendObservabilityRendererImplemented},
	"GrafanaML":                    {render: renderML, gateOnStack: true, implemented: mlRendererImplemented},
	"GrafanaK6Project":             {render: renderK6Project, gateOnStack: true, implemented: k6RendererImplemented},
	"GrafanaSyntheticMonitoring":   {render: renderSyntheticMonitoring, gateOnStack: true, implemented: syntheticMonitoringRendererImplemented},
	"GrafanaStackLadder":           {render: renderStackLadder, gateOnStack: false, implemented: ladderRendererImplemented},
	"GrafanaCloudStackRequest":     {render: renderStack, implemented: true},
	"GrafanaCustomRoleBinding": {
		render: func(xr map[string]any, _ map[resource.Name]resource.ObservedComposed, config map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
			profile, _ := config[resolvedStackProfileConfigKey].(string)
			spec, _ := config["spec"].(map[string]any)
			return renderRoleBindingWithPlatformProfile(xr, profile, stringListValue(spec, "publicDashboardProfiles", nil))
		}, gateOnStack: true, implemented: true,
	},
	"GrafanaTeamAccess": {
		render: func(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
			profile, _ := config[resolvedStackProfileConfigKey].(string)
			spec, _ := config["spec"].(map[string]any)
			return renderTeamAccessWithPlatformProfile(xr, observed, profile, stringListValue(spec, "publicDashboardProfiles", nil))
		}, gateOnStack: true, implemented: true,
	},
	"GrafanaContentAccessPolicy": {
		render: func(xr map[string]any, _ map[resource.Name]resource.ObservedComposed, _ map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
			return renderContentAccessPolicy(xr)
		}, gateOnStack: true, implemented: true,
	},
	"GrafanaStackInventory":         {render: renderStackInventory, gateOnStack: true, implemented: inventoryRendererImplemented},
	"GrafanaFleetPipelines":         {render: renderFleetPipelines, gateOnStack: true, implemented: fleetRendererImplemented},
	"GrafanaAlertingBundle":         {render: renderAlertingBundle, gateOnStack: true, implemented: alertingRendererImplemented},
	"GrafanaAgentObservability":     {render: renderAgentObservability, gateOnStack: true, implemented: agentObservabilityRendererImplemented},
	"GrafanaAssistantGovernance":    {render: renderAssistantGovernance, gateOnStack: true, implemented: assistantRendererImplemented},
	"GrafanaDatasourceAccess":       {render: renderDatasourceAccess, gateOnStack: true, implemented: datasourceAccessRendererImplemented},
	"GrafanaProvisioningRepository": {render: renderProvisioningRepository, gateOnStack: true, implemented: provisioningRendererImplemented},
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

	clock := f.now
	if clock == nil {
		clock = time.Now
	}
	config[reconcileTimeConfigKey] = clock().UTC()

	content := xr.Resource.UnstructuredContent()
	kind, _ := content["kind"].(string)
	renderer, ok := compositeRenderers[kind]
	if !ok {
		response.Fatal(rsp, errors.Errorf("unsupported composite kind %q", kind))
		return rsp, nil
	}
	if !renderer.implemented {
		response.Fatal(rsp, errors.Errorf("composite kind %q is registered but not implemented", kind))
		return rsp, nil
	}
	resolvedConfig := serviceBootstrapConfig(req, rsp, content, rendererConfig(req, content, config))
	productContextReady := true
	if kind == "GrafanaK6Project" {
		_, productContextReady = resolvedConfig["_resolvedK6Stack"]
	}
	if kind == "GrafanaSyntheticMonitoring" {
		_, productContextReady = resolvedConfig["referencedStack"]
	}
	if !productContextReady && len(observed) > 0 {
		response.Fatal(rsp, errors.New("waiting for trusted product bootstrap context; preserving existing composed resources"))
		return rsp, nil
	}
	desired, err := renderer.render(content, observed, resolvedConfig)
	if err == nil {
		err = productWithdrawalError(kind, content, desired, observed)
	}
	if err == nil {
		var status *resource.Composite
		switch kind {
		case "GrafanaCloudStackRequest":
			status = desiredStackStatus(content, observed, config)
		case "GrafanaStackInventory":
			status = desiredStackInventoryStatus(content, observed)
		case "GrafanaStackLadder":
			status = desiredStackLadderStatus(content, observed)
		case "GrafanaSyntheticMonitoring":
			status = desiredSyntheticMonitoringStatus(content, observed, resolvedConfig)
		}
		if status != nil {
			if err = response.SetDesiredCompositeResource(rsp, status); err != nil {
				err = errors.Wrap(err, "cannot set desired composite status")
			}
		}
	}
	if err != nil {
		response.Fatal(rsp, err)
		return rsp, nil
	}
	if renderer.gateOnStack {
		gated, admitted, gateErr := gateAccessResources(req, rsp, content, desired, observed)
		if gateErr != nil {
			response.Fatal(rsp, gateErr)
			return rsp, nil
		}
		desired = gated
		if kind == "GrafanaSyntheticMonitoring" {
			stack, _ := resolvedConfig["referencedStack"].(map[string]any)
			admitted = admitted && syntheticMonitoringInstallationVerified(observed, stringValue(stack, "stackID", ""))
		}
		if err := setAccessCompositeReadiness(rsp, admitted && productContextReady); err != nil {
			response.Fatal(rsp, errors.Wrap(err, "cannot set access composite readiness"))
			return rsp, nil
		}
	}
	if kind == "GrafanaSyntheticMonitoring" {
		pruneSMDesired(rsp, desired)
	}
	if kind == "GrafanaCloudStackRequest" && desired["expiry-warning"] == nil && rsp.GetDesired() != nil {
		delete(rsp.Desired.Resources, "expiry-warning")
	}
	markObservedResourcesReady(desired, observed)
	if err := response.SetDesiredComposedResources(rsp, desired); err != nil {
		response.Fatal(rsp, errors.Wrap(err, "cannot set desired composed resources"))
	}

	return rsp, nil
}

const referencedStackRequirement = "referenced-stack"

// gateAccessResources asks Crossplane to fetch the stack referenced by an
// access claim and admits the claim's children only after that request reports
// Ready=True. The returned map intentionally retains configured children that
// have already been observed while the gate is closed; removing one from the
// desired set would cause a transient stack outage to delete it.
func gateAccessResources(req *fnv1.RunFunctionRequest, rsp *fnv1.RunFunctionResponse, xr map[string]any, staged map[resource.Name]*resource.DesiredComposed, observed map[resource.Name]resource.ObservedComposed) (map[resource.Name]*resource.DesiredComposed, bool, error) {
	admit, err := accessResourcesAdmittedWithRequest(req, rsp, xr)
	if err != nil {
		return nil, false, err
	}
	desired := map[resource.Name]*resource.DesiredComposed{}
	if assistantGovernanceChildrenBlocked(xr, observed) {
		// The normal access gate intentionally retains an already-observed child
		// across a transient stack readiness loss. Assistant terms are a second,
		// stricter gate: an explicit withdrawal or an observed non-acceptance must
		// remove every rule and MCP server immediately, even when those children
		// were present in the previous desired set. Keep only the terms singleton
		// itself, and only when the stack gate permits it or it already exists.
		if terms, ok := staged["terms"]; ok && (admit || observedExists(observed, "terms")) {
			desired["terms"] = terms
		}
		return desired, admit, nil
	}
	admitStaged(desired, staged, observed, admit)
	return desired, admit, nil
}

func assistantGovernanceChildrenBlocked(xr map[string]any, observed map[resource.Name]resource.ObservedComposed) bool {
	if xr["kind"] != "GrafanaAssistantGovernance" {
		return false
	}
	spec, _ := xr["spec"].(map[string]any)
	terms, _ := spec["termsAcceptance"].(map[string]any)
	if terms == nil {
		terms, _ = spec["terms"].(map[string]any)
	}
	accepted, ok := terms["accepted"].(bool)
	if !ok || !accepted {
		return true
	}
	observedAccepted, present := observedBool(observed, "terms", "status", "atProvider", "accepted")
	return !present || !observedAccepted
}

func accessResourcesAdmittedWithRequest(req *fnv1.RunFunctionRequest, rsp *fnv1.RunFunctionResponse, xr map[string]any) (bool, error) {
	if request.AdvertisesCapabilities(req) && !request.HasCapability(req, fnv1.Capability_CAPABILITY_REQUIRED_RESOURCES) {
		return false, errors.New("Crossplane advertises capabilities but does not support CAPABILITY_REQUIRED_RESOURCES")
	}

	stackName := accessStackReference(xr)
	namespace, _ := xr["metadata"].(map[string]any)
	namespaceName, _ := namespace["namespace"].(string)
	apiVersion, err := referencedStackAPIVersion(xr)
	if err != nil {
		return false, err
	}
	if stackName == "" || namespaceName == "" {
		return false, errors.New("access claim must set metadata.namespace and spec.stackRef.name")
	}

	if rsp.Requirements == nil {
		rsp.Requirements = &fnv1.Requirements{}
	}
	if rsp.Requirements.Resources == nil {
		rsp.Requirements.Resources = map[string]*fnv1.ResourceSelector{}
	}
	rsp.Requirements.Resources[referencedStackRequirement] = &fnv1.ResourceSelector{
		ApiVersion: apiVersion,
		Kind:       "GrafanaCloudStackRequest",
		Match:      &fnv1.ResourceSelector_MatchName{MatchName: stackName},
		Namespace:  &namespaceName,
	}

	resources, resolved, err := request.GetRequiredResource(req, referencedStackRequirement)
	if err != nil {
		return false, errors.Wrap(err, "cannot read required referenced stack")
	}
	if !resolved || len(resources) == 0 {
		return false, nil
	}
	if len(resources) != 1 {
		return false, errors.Errorf("referenced stack requirement returned %d resources; expected exactly one", len(resources))
	}
	if resources[0].Resource == nil {
		return false, errors.New("referenced stack requirement returned an empty resource")
	}
	object := resources[0].Resource.UnstructuredContent()
	if !requiredStackIdentityMatches(object, stackName, namespaceName, apiVersion) {
		return false, errors.New("required resource is not the referenced stack")
	}
	stackSpec, _ := object["spec"].(map[string]any)
	spec, _ := xr["spec"].(map[string]any)
	team, _ := spec["team"].(map[string]any)
	groupKey := "externalGroups"
	if xr["kind"] == "GrafanaCustomRoleBinding" {
		groupKey = "groups"
	}
	groups, _ := team[groupKey].([]any)
	if _, scim := stackSpec["scim"]; scim && len(groups) > 0 && (xr["kind"] == "GrafanaTeamAccess" || xr["kind"] == "GrafanaCustomRoleBinding") {
		return false, errors.New("external-group mapping and SCIM group sync are mutually exclusive")
	}
	return requiredStackReady(object), nil
}

func accessStackReference(xr map[string]any) string {
	spec, _ := xr["spec"].(map[string]any)
	stackRef, _ := spec["stackRef"].(map[string]any)
	name, _ := stackRef["name"].(string)
	return name
}

// rendererConfig adds request-derived context that a renderer must not accept
// from the composite itself. In particular, public-dashboard permission is
// selected only by the profile on the validated referenced stack and the
// platform-owned allow-list in the Composition input.
func rendererConfig(req *fnv1.RunFunctionRequest, xr, config map[string]any) map[string]any {
	kind, _ := xr["kind"].(string)
	if kind != "GrafanaCustomRoleBinding" && kind != "GrafanaTeamAccess" {
		return config
	}
	result := make(map[string]any, len(config)+1)
	for key, value := range config {
		result[key] = value
	}
	result[resolvedStackProfileConfigKey] = requiredReferencedStackProfile(req, xr)
	return result
}

func requiredReferencedStackProfile(req *fnv1.RunFunctionRequest, xr map[string]any) string {
	resources, resolved, err := request.GetRequiredResource(req, referencedStackRequirement)
	if err != nil || !resolved || len(resources) != 1 || resources[0].Resource == nil {
		return ""
	}
	metadata, _ := xr["metadata"].(map[string]any)
	name := accessStackReference(xr)
	namespace, _ := metadata["namespace"].(string)
	apiVersion, _ := xr["apiVersion"].(string)
	object := resources[0].Resource.UnstructuredContent()
	if !requiredStackIdentityMatches(object, name, namespace, apiVersion) {
		return ""
	}
	spec, _ := object["spec"].(map[string]any)
	return stringValue(spec, "profile", defaultStackProfile)
}

func referencedStackAPIVersion(xr map[string]any) (string, error) {
	apiVersion, _ := xr["apiVersion"].(string)
	if apiVersion == "" {
		return "", errors.New("access claim apiVersion must be set")
	}
	return apiVersion, nil
}

func requiredStackIdentityMatches(object map[string]any, name, namespace, apiVersion string) bool {
	if object["apiVersion"] != apiVersion || object["kind"] != "GrafanaCloudStackRequest" {
		return false
	}
	metadata, ok := object["metadata"].(map[string]any)
	if !ok {
		return false
	}
	return metadata["name"] == name && metadata["namespace"] == namespace
}

func requiredStackReady(object map[string]any) bool {
	metadata, _ := object["metadata"].(map[string]any)
	if metadata["deletionTimestamp"] != nil {
		return false
	}
	status, ok := object["status"].(map[string]any)
	if !ok {
		return false
	}
	if status["deletionArmed"] == true {
		return false
	}
	conditions, ok := status["conditions"].([]any)
	if !ok {
		return false
	}
	for _, item := range conditions {
		condition, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if condition["type"] == "Ready" && condition["status"] == "True" {
			return true
		}
	}
	return false
}

func setAccessCompositeReadiness(rsp *fnv1.RunFunctionResponse, admitted bool) error {
	ready := resource.ReadyFalse
	if admitted {
		ready = resource.ReadyUnspecified
	}
	compositeResource := rsp.GetDesired().GetComposite()
	if compositeResource == nil || compositeResource.GetResource() == nil {
		return response.SetDesiredCompositeResource(rsp, &resource.Composite{Resource: composite.New(), Ready: ready})
	}
	switch ready {
	case resource.ReadyFalse:
		compositeResource.Ready = fnv1.Ready_READY_FALSE
	case resource.ReadyUnspecified:
		compositeResource.Ready = fnv1.Ready_READY_UNSPECIFIED
	}
	return nil
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

func externalResourcesLifecycle(spec map[string]any) (string, error) {
	lifecycle, exists := spec["lifecycle"]
	if !exists {
		return "Retain", nil
	}
	values, ok := lifecycle.(map[string]any)
	if !ok {
		return "", errors.New("spec.lifecycle must be an object")
	}
	externalResources, exists := values["externalResources"]
	if !exists {
		return "Retain", nil
	}
	mode, ok := externalResources.(string)
	if !ok {
		return "", errors.New("spec.lifecycle.externalResources must be a string")
	}
	if !oneOf(mode, "Retain", "Delete") {
		return "", errors.Errorf("unsupported spec.lifecycle.externalResources %q", mode)
	}
	return mode, nil
}

func renderStack(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	metadata, _ := xr["metadata"].(map[string]any)
	spec, _ := xr["spec"].(map[string]any)
	if _, requested := spec["scim"]; requested {
		return nil, errors.New("SCIM is out of scope; use GrafanaTeamAccess or GrafanaCustomRoleBinding external-group mapping")
	}

	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	displayName, _ := spec["displayName"].(string)
	slug, _ := spec["slug"].(string)
	region, _ := spec["region"].(string)
	usage, _ := spec["usage"].(string)
	organization, _ := spec["organization"].(string)
	profile := stringValue(spec, "profile", "standard")
	requestUID, _ := metadata["uid"].(string)
	if name == "" || namespace == "" || displayName == "" || slug == "" || region == "" || usage == "" || organization == "" {
		return nil, errors.New("stack claim must set metadata name and namespace plus displayName, slug, region, usage, and organization")
	}
	if name != slug {
		return nil, errors.New("stack claim metadata.name must match spec.slug")
	}
	lifecycle, err := externalResourcesLifecycle(spec)
	if err != nil {
		return nil, err
	}
	settings := configuredPlatformSettings(config)
	administratorTokenLifetime, err := boundedTokenLifetime(settings.maximumTokenLifetime, requestedTokenLifetime)
	if err != nil {
		return nil, err
	}
	administratorRotationWindow, err := boundedTokenEarlyRotationWindow(administratorTokenLifetime)
	if err != nil {
		return nil, err
	}
	if !oneOf(usage, settings.allowedUsages...) {
		return nil, errors.Errorf("usage %q is not allowed by platform configuration", usage)
	}
	organizationSettings, err := settings.resolveOrganization(organization, region, usage)
	if err != nil {
		return nil, err
	}
	organizationProviderConfigName := organizationSettings.providerConfigName
	if lifecycle == "Delete" && !settings.deletionAuthorized(namespace, name, requestUID, profile) {
		return nil, errors.Errorf("request %s/%s with profile %q is not authorized to delete external resources", namespace, name, profile)
	}
	deletingExternalResources := lifecycle == "Delete"
	if deletingExternalResources && expiryRendererImplemented {
		deletingExternalResources, err = expiryArmingPermitted(xr, config)
		if err != nil {
			return nil, err
		}
	}
	externalResourcePolicies := managementPolicies
	deleteProtection := true
	deleteOnDestroy := false
	pushSecretDeletionPolicy := "None"
	if deletingExternalResources {
		externalResourcePolicies = []any{"*"}
		deleteProtection = false
		deleteOnDestroy = true
		pushSecretDeletionPolicy = "Delete"
	}

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
	outputPath := fmt.Sprintf("%s/%s/%s/%s", settings.outputSecretPrefix, organization, usage, slug)
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
			"managementPolicies": externalResourcePolicies,
			"forProvider": map[string]any{
				"name": displayName, "slug": slug, "regionSlug": region,
				"deleteProtection": deleteProtection, "waitForReadiness": true, "waitForReadinessTimeout": "10m0s",
				"labels": map[string]any{"provisioner": "crossplane", "usage": usage, "profile": profile},
			},
			"providerConfigRef":          map[string]any{"kind": "ProviderConfig", "name": organizationProviderConfigName},
			"writeConnectionSecretToRef": map[string]any{"name": stackDetailsSecret},
		})
	desired["stack-service-account"] = newDesired("cloud.grafana.m.crossplane.io/v1alpha1", "StackServiceAccount", namespace, slug+"-admin", nil,
		map[string]any{
			"managementPolicies": externalResourcePolicies,
			"forProvider": map[string]any{
				"cloudStackRef": map[string]any{"name": slug},
				"name":          "Grafana vending controller", "role": "Admin", "isDisabled": false,
			},
			"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": organizationProviderConfigName},
		})
	serviceAccountID := observedString(observed, "stack-service-account", "status.atProvider.id")
	if serviceAccountID == "" {
		serviceAccountID = observedString(observed, "stack-token", "spec.forProvider.serviceAccountId")
	}
	if serviceAccountID != "" {
		desired["stack-token"] = newDesired("cloud.grafana.m.crossplane.io/v1alpha1", "StackServiceAccountRotatingToken", namespace, slug+"-admin", nil,
			map[string]any{
				"managementPolicies": externalResourcePolicies,
				"forProvider": map[string]any{
					"namePrefix": "grafana-vending-", "secondsToLive": int64(administratorTokenLifetime / time.Second),
					"earlyRotationWindowSeconds": int64(administratorRotationWindow / time.Second), "deleteOnDestroy": deleteOnDestroy,
					"stackSlug": slug, "serviceAccountId": serviceAccountID,
				},
				"providerConfigRef":          map[string]any{"kind": "ProviderConfig", "name": organizationProviderConfigName},
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
	outputDocument := fmt.Sprintf(`{{ $token := index . "attribute.key" | toString }}{"stack_name":%q,"stack_slug":%q,"stack_url":%q,"stack_region":%q,"organization":%q,"usage":%q,"change_reference":%q,"configuration_item_reference":%q,"stack_service_account_token":{{ $token | toJson }},"telemetry_access_policy_secret_path":%q}`, displayName, slug, "https://"+slug+".grafana.net", region, organization, usage, changeReference, configurationItemReference, telemetryOutputPath)
	desired["credentials"] = newDesired("external-secrets.io/v1alpha1", "PushSecret", namespace, slug+"-credentials", nil,
		map[string]any{
			"refreshInterval": "1h", "updatePolicy": "Replace", "deletionPolicy": pushSecretDeletionPolicy,
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
					"spec": map[string]any{
						"secretPushFormat": "string",
						"tags":             map[string]any{"grafana-cloud-vending-machine": "managed"},
					},
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
					"data":          map[string]any{"credentials": `{"auth":{{ .stackServiceAccountToken | toJson }},"url":{{ .stackURL | toJson }},"fleet_management_auth":{{ .fleetManagementAuth | toJson }}}`},
				},
			},
			"data": []any{
				map[string]any{"secretKey": "stackServiceAccountToken", "remoteRef": map[string]any{"key": outputPath, "property": "stack_service_account_token"}},
				map[string]any{"secretKey": "stackURL", "remoteRef": map[string]any{"key": outputPath, "property": "stack_url"}},
				map[string]any{"secretKey": "fleetManagementAuth", "remoteRef": map[string]any{"key": outputPath + "/fleet-management", "property": "fleet_management_auth"}},
			},
		})

	if err := addFleetAccess(desired, observed, namespace, slug, region, outputPath, profile, settings, organizationProviderConfigName, deletingExternalResources); err != nil {
		return nil, err
	}
	if err := addTelemetryAccess(desired, observed, namespace, slug, region, telemetryOutputPath, spec, settings, organizationProviderConfigName, deletingExternalResources); err != nil {
		return nil, err
	}
	if err := addPluginInstallations(whenStackServes, namespace, slug, spec, organizationProviderConfigName); err != nil {
		return nil, err
	}
	if err := addObservabilityProducts(whenCredentialsPublished, namespace, providerConfig, spec); err != nil {
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

	if sloRendererImplemented {
		if err := addGoldenSLO(whenCredentialsPublished, xr, observed, config); err != nil {
			return nil, err
		}
	} else if platformSpec, ok := config["spec"].(map[string]any); ok {
		if _, configured := platformSpec["goldenSLOProfiles"]; configured {
			return nil, errors.New("golden SLO renderer is not implemented")
		}
	}
	if retentionRendererImplemented {
		if err := addRetentionFanout(whenCredentialsPublished, xr, observed, config); err != nil {
			return nil, err
		}
	} else if _, requested := spec["retention"]; requested {
		return nil, errors.New("retention renderer is not implemented")
	}

	if expiryRendererImplemented {
		if _, requested := spec["expiry"]; requested {
			incident, _ := spec["incidentIntegration"].(map[string]any)
			if incident["enabled"] != true || whenCredentialsPublished["incident-alerting-production"] == nil {
				return nil, errors.New("expiry requires the configured already-vended production incident contact point")
			}
		}
		if err := addStackExpiry(whenCredentialsPublished, xr, observed, config); err != nil {
			return nil, err
		}
	} else if _, requested := spec["expiry"]; requested {
		return nil, errors.New("expiry renderer is not implemented")
	}
	stackServes := observedReady(observed, "stack")
	admitStaged(desired, whenStackServes, observed, stackServes)
	admitStaged(desired, whenCredentialsPublished, observed, stackServes && observedReady(observed, "instance-credentials"))

	return desired, nil
}

func desiredStackStatus(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) *resource.Composite {
	spec, _ := xr["spec"].(map[string]any)
	slug, _ := spec["slug"].(string)
	usage, _ := spec["usage"].(string)
	organization, _ := spec["organization"].(string)
	settings := configuredPlatformSettings(config)
	status := map[string]any{
		"outputSecretPath": fmt.Sprintf("%s/%s/%s/%s", settings.outputSecretPrefix, organization, usage, slug),
	}
	if telemetryAccessEnabled(spec) {
		status["telemetrySecretPath"] = fmt.Sprintf("%s/%s/%s/%s/telemetry-publisher", settings.outputSecretPrefix, organization, usage, slug)
	}
	lifecycle, _ := externalResourcesLifecycle(spec)
	deletionArmed := lifecycle == "Delete"
	if deletionArmed && expiryRendererImplemented {
		permitted, err := expiryArmingPermitted(xr, config)
		deletionArmed = err == nil && permitted
	}
	deleteProtection, protectionObserved := observedBool(observed, "stack", "status", "atProvider", "deleteProtection")
	credentialsPrepared := observedPushSecretDeletionPrepared(observed, "credentials")
	fleetCredentialsPrepared := observedPushSecretDeletionPrepared(observed, "fleet-management-credentials")
	telemetryPrepared := !telemetryAccessEnabled(spec) || observedPushSecretDeletionPrepared(observed, "telemetry-credentials")
	administratorTokenPrepared := observedRotatingTokenDeletionPrepared(observed, "stack-token")
	fleetTokenPrepared := observedRotatingTokenDeletionPrepared(observed, "fleet-management-token")
	telemetryTokenPrepared := !telemetryAccessEnabled(spec) || observedRotatingTokenDeletionPrepared(observed, "telemetry-token")
	status["deletionArmed"] = deletionArmed
	status["deletionReady"] = deletionArmed && protectionObserved && !deleteProtection && administratorTokenPrepared && fleetTokenPrepared && telemetryTokenPrepared && credentialsPrepared && fleetCredentialsPrepared && telemetryPrepared
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
	publishTokenExpiryStatus(status, observed)
	mergeStackExpiryStatus(status, config)
	desired := &resource.Composite{Resource: composite.New()}
	desired.Resource.SetUnstructuredContent(map[string]any{"status": status})
	return desired
}

type platformSettings struct {
	maximumTokenLifetime    string
	tokenUseNetworkProfiles []any
	organizations           []organizationSettings
	outputSecretPrefix      string
	secretStoreName         string
	secretStoreKind         string
	allowedUsages           []string
	deletionAuthorizations  []deletionAuthorization
}

type organizationSettings struct {
	name               string
	providerConfigName string
	allowedRegions     []string
	allowedUsages      []string
}

type deletionAuthorization struct {
	namespace string
	name      string
	uid       string
	profile   string
}

func configuredPlatformSettings(config map[string]any) platformSettings {
	spec, _ := config["spec"].(map[string]any)
	tokenUseNetworkProfiles, _ := spec["tokenUseNetworkProfiles"].([]any)
	store, _ := spec["secretStoreRef"].(map[string]any)
	return platformSettings{
		maximumTokenLifetime:    stringValue(spec, "maximumTokenLifetime", ""),
		tokenUseNetworkProfiles: tokenUseNetworkProfiles,
		organizations:           organizationSettingsList(spec),
		outputSecretPrefix:      stringValue(spec, "outputSecretPrefix", "/platform/grafana-cloud/stacks"),
		secretStoreName:         stringValue(store, "name", "grafana-vending-secrets"),
		secretStoreKind:         stringValue(store, "kind", "SecretStore"),
		allowedUsages:           stringListValue(spec, "allowedUsages", []string{"development", "production"}),
		deletionAuthorizations:  deletionAuthorizationList(spec),
	}
}

func organizationSettingsList(spec map[string]any) []organizationSettings {
	values, _ := spec["organizations"].([]any)
	result := make([]organizationSettings, 0, len(values))
	for _, value := range values {
		item, ok := value.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, organizationSettings{
			name:               stringValue(item, "name", ""),
			providerConfigName: stringValue(item, "providerConfigName", ""),
			allowedRegions:     stringListValue(item, "allowedRegions", nil),
			allowedUsages:      stringListValue(item, "allowedUsages", nil),
		})
	}
	return result
}

func (s platformSettings) resolveOrganization(name, region, usage string) (organizationSettings, error) {
	for _, organization := range s.organizations {
		if organization.name != name {
			continue
		}
		if organization.providerConfigName == "" || len(organization.allowedRegions) == 0 || len(organization.allowedUsages) == 0 {
			return organizationSettings{}, errors.Errorf("organization %q has an incomplete platform registry entry", name)
		}
		if !oneOf(region, organization.allowedRegions...) {
			return organizationSettings{}, errors.Errorf("region %q is not allowed for organization %q", region, name)
		}
		if !oneOf(usage, organization.allowedUsages...) {
			return organizationSettings{}, errors.Errorf("usage %q is not allowed for organization %q", usage, name)
		}
		return organization, nil
	}
	return organizationSettings{}, errors.Errorf("unknown organization %q", name)
}

func deletionAuthorizationList(spec map[string]any) []deletionAuthorization {
	values, _ := spec["deletionAuthorizations"].([]any)
	result := make([]deletionAuthorization, 0, len(values))
	for _, value := range values {
		item, ok := value.(map[string]any)
		if !ok {
			continue
		}
		authorization := deletionAuthorization{
			namespace: stringValue(item, "namespace", ""),
			name:      stringValue(item, "name", ""),
			uid:       stringValue(item, "uid", ""),
			profile:   stringValue(item, "profile", ""),
		}
		if authorization.namespace != "" && authorization.name != "" && authorization.uid != "" && authorization.profile != "" {
			result = append(result, authorization)
		}
	}
	return result
}

func (s platformSettings) deletionAuthorized(namespace, name, uid, profile string) bool {
	for _, authorization := range s.deletionAuthorizations {
		if authorization.namespace == namespace && authorization.name == name && authorization.uid == uid && authorization.profile == profile {
			return true
		}
	}
	return false
}

func stringListValue(object map[string]any, key string, fallback []string) []string {
	values, ok := object[key].([]any)
	if !ok {
		return fallback
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value, ok := value.(string); ok {
			result = append(result, value)
		}
	}
	return result
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

func observedBool(observed map[resource.Name]resource.ObservedComposed, name resource.Name, fieldPath ...string) (bool, bool) {
	r, ok := observed[name]
	if !ok || r.Resource == nil {
		return false, false
	}
	current := r.Resource.UnstructuredContent()
	for _, field := range fieldPath[:len(fieldPath)-1] {
		next, ok := current[field].(map[string]any)
		if !ok {
			return false, false
		}
		current = next
	}
	value, ok := current[fieldPath[len(fieldPath)-1]].(bool)
	return value, ok
}

func observedPushSecretDeletionPrepared(observed map[resource.Name]resource.ObservedComposed, name resource.Name) bool {
	r, ok := observed[name]
	if !ok || r.Resource == nil {
		return false
	}
	current := r.Resource.UnstructuredContent()
	spec, _ := current["spec"].(map[string]any)
	if spec["deletionPolicy"] != "Delete" {
		return false
	}
	metadata, _ := current["metadata"].(map[string]any)
	finalizers, _ := metadata["finalizers"].([]any)
	if !oneOf("pushsecret.externalsecrets.io/finalizer", stringValues(finalizers)...) {
		return false
	}
	generation, err := r.Resource.GetInteger("metadata.generation")
	if err != nil || generation < 1 {
		return false
	}
	status, _ := current["status"].(map[string]any)
	syncedResourceVersion, _ := status["syncedResourceVersion"].(string)
	if !strings.HasPrefix(syncedResourceVersion, strconv.FormatInt(generation, 10)+"-") {
		return false
	}
	syncedPushSecrets, _ := status["syncedPushSecrets"].(map[string]any)
	if len(syncedPushSecrets) == 0 {
		return false
	}
	conditions, _ := status["conditions"].([]any)
	for _, value := range conditions {
		condition, ok := value.(map[string]any)
		if ok && condition["type"] == "Ready" && condition["status"] == "True" {
			return true
		}
	}
	return false
}

func observedRotatingTokenDeletionPrepared(observed map[resource.Name]resource.ObservedComposed, name resource.Name) bool {
	r, ok := observed[name]
	if !ok || r.Resource == nil {
		return false
	}
	current := r.Resource.UnstructuredContent()
	spec, _ := current["spec"].(map[string]any)
	policies, _ := spec["managementPolicies"].([]any)
	if len(policies) != 1 || policies[0] != "*" {
		return false
	}
	forProvider, _ := spec["forProvider"].(map[string]any)
	return forProvider["deleteOnDestroy"] == true
}

func stringValues(values []any) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value, ok := value.(string); ok {
			result = append(result, value)
		}
	}
	return result
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
