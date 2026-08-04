package main

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
)

func TestMinimalStackRendersBaseline(t *testing.T) {
	rsp := runStack(t, minimalStack(), nil)

	got := make([]string, 0, len(rsp.GetDesired().GetResources()))
	for name, desired := range rsp.GetDesired().GetResources() {
		got = append(got, name+"="+desired.GetResource().GetFields()["kind"].GetStringValue())
	}
	sort.Strings(got)

	want := []string{
		"billing-dashboard=Dashboard",
		"billing-folder=Folder",
		"credentials=PushSecret",
		"endpoints-dashboard=Dashboard",
		"endpoints-folder=Folder",
		"homepage-dashboard=Dashboard",
		"homepage-folder=Folder",
		"instance-credentials=ExternalSecret",
		"organization-preferences=OrganizationPreferences",
		"provider-config=ProviderConfig",
		"stack=Stack",
		"stack-service-account=StackServiceAccount",
		"telemetry-access-policy=AccessPolicy",
		"telemetry-credentials=PushSecret",
	}
	sort.Strings(want)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("minimal stack desired resources differ (-want +got):\n%s", diff)
	}
}

func TestTelemetryAccessPolicyIsStackScopedAndLeastPrivilege(t *testing.T) {
	rsp := runStack(t, minimalStack(), nil)
	policy := desiredResource(t, rsp, "telemetry-access-policy")
	spec := nestedMap(t, policy, "spec")
	forProvider := nestedMap(t, spec, "forProvider")

	want := map[string]any{
		"displayName": "Telemetry publisher for teamdemo01",
		"name":        "teamdemo01-telemetry-publisher",
		"realm": []any{map[string]any{
			"stackRef": map[string]any{"name": "teamdemo01"},
			"type":     "stack",
		}},
		"region": "prod-us-central-0",
		"scopes": []any{"stacks:read", "metrics:write", "logs:write", "traces:write"},
	}
	if diff := cmp.Diff(want, forProvider); diff != "" {
		t.Fatalf("telemetry access policy differs (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(managementPolicies, spec["managementPolicies"]); diff != "" {
		t.Fatalf("telemetry access policy management policies differ (-want +got):\n%s", diff)
	}
}

func TestTelemetryRotatingTokenWaitsForPolicyIDAndPublishesToAWS(t *testing.T) {
	baseline := runStack(t, minimalStack(), nil)
	if _, ok := baseline.GetDesired().GetResources()["telemetry-token"]; ok {
		t.Fatal("telemetry rotating token was rendered before Grafana assigned an access-policy ID")
	}

	observed := map[string]*fnv1.Resource{
		"telemetry-access-policy": observedResource(`{
			"apiVersion":"cloud.grafana.m.crossplane.io/v1alpha1",
			"kind":"AccessPolicy",
			"metadata":{"name":"teamdemo01-telemetry-publisher","namespace":"grafana-vending"},
			"status":{"atProvider":{"policyId":"policy-12345"}}
		}`),
	}
	rsp := runStack(t, minimalStack(), observed)
	token := desiredResource(t, rsp, "telemetry-token")
	forProvider := nestedMap(t, token, "spec", "forProvider")
	wantToken := map[string]any{
		"accessPolicyId":      "policy-12345",
		"deleteOnDestroy":     false,
		"displayName":         "Telemetry publisher token for teamdemo01",
		"earlyRotationWindow": "168h",
		"expireAfter":         "720h",
		"namePrefix":          "teamdemo01-telemetry-",
		"region":              "prod-us-central-0",
	}
	if diff := cmp.Diff(wantToken, forProvider); diff != "" {
		t.Fatalf("telemetry rotating token differs (-want +got):\n%s", diff)
	}

	push := desiredResource(t, rsp, "telemetry-credentials")
	pushSpec := nestedMap(t, push, "spec")
	document := nestedMap(t, pushSpec, "template", "data")["telemetry.json"].(string)
	for _, fragment := range []string{
		`"stack_slug":"teamdemo01"`,
		`"stack_region":"prod-us-central-0"`,
		`"access_policy_name":"teamdemo01-telemetry-publisher"`,
		`"access_policy_token":`,
		`index . "attribute.token"`,
	} {
		if !strings.Contains(document, fragment) {
			t.Errorf("telemetry output document does not contain %q: %s", fragment, document)
		}
	}
	matches := pushSpec["data"].([]any)
	remote := nestedMap(t, matches[0].(map[string]any), "match", "remoteRef")
	if got, want := remote["remoteKey"], "/platform/grafana-cloud/stacks/prod-us-central-0/production/teamdemo01/telemetry-publisher"; got != want {
		t.Fatalf("telemetry output path = %v, want %s", got, want)
	}
}

func TestTelemetryAccessCanBeDisabled(t *testing.T) {
	claim := stackDocument(map[string]any{"telemetryAccess": map[string]any{"enabled": false}})
	rsp := runStack(t, claim, nil)
	for _, name := range []string{"telemetry-access-policy", "telemetry-token", "telemetry-credentials"} {
		if _, ok := rsp.GetDesired().GetResources()[name]; ok {
			t.Fatalf("disabled telemetry access rendered %s", name)
		}
	}
}

func TestPluginInstallationsAreOptionalAndStackScoped(t *testing.T) {
	if _, ok := runStack(t, minimalStack(), nil).GetDesired().GetResources()["plugin-clock-panel"]; ok {
		t.Fatal("plugin was rendered without an explicit request")
	}

	claim := stackDocument(map[string]any{"plugins": []any{
		map[string]any{"slug": "grafana-clock-panel", "version": "2.1.3"},
		map[string]any{"slug": "grafana-piechart-panel"},
	}})
	rsp := runStack(t, claim, nil)
	clock := desiredResource(t, rsp, "plugin-grafana-clock-panel")
	wantClock := map[string]any{
		"cloudStackRef": map[string]any{"name": "teamdemo01"},
		"slug":          "grafana-clock-panel",
		"version":       "2.1.3",
	}
	if diff := cmp.Diff(wantClock, nestedMap(t, clock, "spec", "forProvider")); diff != "" {
		t.Fatalf("plugin installation differs (-want +got):\n%s", diff)
	}
	pie := desiredResource(t, rsp, "plugin-grafana-piechart-panel")
	if got := nestedMap(t, pie, "spec", "forProvider")["version"]; got != "latest" {
		t.Fatalf("default plugin version = %v, want latest", got)
	}
}

func TestUnknownCompositeKindIsRejected(t *testing.T) {
	claim := strings.Replace(minimalStack(), `"kind":"GrafanaCloudStackRequest"`, `"kind":"Unknown"`, 1)
	rsp := callFunction(t, claim, nil, "")

	if fatal := fatalResult(rsp); !strings.Contains(fatal, "unsupported composite kind") {
		t.Fatalf("unknown kind returned fatal result %q", fatal)
	}
}

func TestRotatingTokenWaitsForServiceAccountID(t *testing.T) {
	rsp := runStack(t, minimalStack(), nil)
	if _, ok := rsp.GetDesired().GetResources()["stack-token"]; ok {
		t.Fatal("rotating token was rendered before Grafana assigned a service-account ID")
	}
}

func TestRotatingTokenUsesObservedServiceAccountID(t *testing.T) {
	observed := map[string]*fnv1.Resource{
		"stack": observedResource(`{
			"apiVersion":"cloud.grafana.m.crossplane.io/v1alpha1",
			"kind":"Stack",
			"metadata":{"name":"teamdemo01","namespace":"grafana-vending"},
			"status":{"atProvider":{"id":"12345","slug":"teamdemo01","url":"https://teamdemo01.grafana.net"}}
		}`),
		"stack-service-account": observedResource(`{
			"apiVersion":"cloud.grafana.m.crossplane.io/v1alpha1",
			"kind":"StackServiceAccount",
			"metadata":{"name":"teamdemo01-admin","namespace":"grafana-vending"},
			"status":{"atProvider":{"id":"67890"}}
		}`),
	}
	rsp := runStack(t, minimalStack(), observed)

	token := desiredResource(t, rsp, "stack-token")
	forProvider := nestedMap(t, token, "spec", "forProvider")
	want := map[string]any{
		"namePrefix":                 "grafana-vending-",
		"secondsToLive":              float64(2592000),
		"earlyRotationWindowSeconds": float64(604800),
		"deleteOnDestroy":            false,
		"stackSlug":                  "teamdemo01",
		"serviceAccountId":           "67890",
	}
	if diff := cmp.Diff(want, forProvider); diff != "" {
		t.Fatalf("rotating token parameters differ (-want +got):\n%s", diff)
	}
}

func TestObservedReadyResourcesAreMarkedReady(t *testing.T) {
	observed := map[string]*fnv1.Resource{
		"stack": observedResource(`{
			"apiVersion":"cloud.grafana.m.crossplane.io/v1alpha1",
			"kind":"Stack",
			"metadata":{"name":"teamdemo01","namespace":"grafana-vending"},
			"status":{"conditions":[{"type":"Ready","status":"True","reason":"Available"}]}
		}`),
	}
	rsp := runStack(t, minimalStack(), observed)

	if got := rsp.GetDesired().GetResources()["stack"].GetReady(); got != fnv1.Ready_READY_TRUE {
		t.Fatalf("observed ready stack readiness = %s, want READY_TRUE", got)
	}
	if got := rsp.GetDesired().GetResources()["stack-service-account"].GetReady(); got != fnv1.Ready_READY_UNSPECIFIED {
		t.Fatalf("unobserved service account readiness = %s, want READY_UNSPECIFIED", got)
	}
}

func TestCredentialChainBuildsStructuredOutputWithoutLiteralToken(t *testing.T) {
	rsp := runStack(t, minimalStack(), nil)

	push := desiredResource(t, rsp, "credentials")
	pushSpec := nestedMap(t, push, "spec")
	data := nestedMap(t, pushSpec, "template", "data")
	document, _ := data["vending.json"].(string)
	for _, fragment := range []string{
		`"stack_name":"Example Stack 01"`,
		`"stack_slug":"teamdemo01"`,
		`"stack_url":"https://teamdemo01.grafana.net"`,
		`"stack_region":"prod-us-central-0"`,
		`"usage":"production"`,
		`"change_reference":""`,
		`"configuration_item_reference":""`,
		`"stack_service_account_token":`,
		`"telemetry_access_policy_secret_path":"/platform/grafana-cloud/stacks/prod-us-central-0/production/teamdemo01/telemetry-publisher"`,
		`index . "attribute.key"`,
	} {
		if !strings.Contains(document, fragment) {
			t.Errorf("output document does not contain %q: %s", fragment, document)
		}
	}
	if strings.Contains(document, "gl"+"c_") {
		t.Fatal("output document contains a literal Grafana credential")
	}

	matches, _ := pushSpec["data"].([]any)
	match := nestedMap(t, matches[0].(map[string]any), "match", "remoteRef")
	if got, want := match["remoteKey"], "/platform/grafana-cloud/stacks/prod-us-central-0/production/teamdemo01"; got != want {
		t.Fatalf("remote output path = %v, want %s", got, want)
	}

	externalSecret := desiredResource(t, rsp, "instance-credentials")
	targetData := nestedMap(t, externalSecret, "spec", "target", "template", "data")
	if got := targetData["credentials"]; got != `{"auth":{{ .stackServiceAccountToken | toJson }},"url":{{ .stackURL | toJson }}}` {
		t.Fatalf("ProviderConfig credential template = %v", got)
	}

	providerConfig := desiredResource(t, rsp, "provider-config")
	secretRef := nestedMap(t, providerConfig, "spec", "credentials", "secretRef")
	wantRef := map[string]any{"key": "credentials", "name": "teamdemo01-provider-credentials", "namespace": "grafana-vending"}
	if diff := cmp.Diff(wantRef, secretRef); diff != "" {
		t.Fatalf("ProviderConfig secret reference differs (-want +got):\n%s", diff)
	}
}

func TestPlatformSettingsArePortableAndApplied(t *testing.T) {
	input := mustJSON(map[string]any{
		"apiVersion": "platform.example.org/v1beta1",
		"kind":       "GrafanaVendingConfig",
		"spec": map[string]any{
			"organizationProviderConfigName": "organization-provider",
			"outputSecretPrefix":             "/example/platform/grafana/stacks",
			"secretStoreRef": map[string]any{
				"name": "central-secret-store",
				"kind": "ClusterSecretStore",
			},
		},
	})
	rsp := runStackWithInput(t, minimalStack(), nil, input)

	stackProvider := nestedMap(t, desiredResource(t, rsp, "stack"), "spec", "providerConfigRef")
	if got := stackProvider["name"]; got != "organization-provider" {
		t.Fatalf("organization ProviderConfig = %v, want organization-provider", got)
	}

	pushSpec := nestedMap(t, desiredResource(t, rsp, "credentials"), "spec")
	storeRef := pushSpec["secretStoreRefs"].([]any)[0].(map[string]any)
	if diff := cmp.Diff(map[string]any{"name": "central-secret-store", "kind": "ClusterSecretStore"}, storeRef); diff != "" {
		t.Fatalf("PushSecret store reference differs (-want +got):\n%s", diff)
	}
	remote := nestedMap(t, pushSpec["data"].([]any)[0].(map[string]any), "match", "remoteRef")
	if got, want := remote["remoteKey"], "/example/platform/grafana/stacks/prod-us-central-0/production/teamdemo01"; got != want {
		t.Fatalf("configured output path = %v, want %s", got, want)
	}

	externalSecretStore := nestedMap(t, desiredResource(t, rsp, "instance-credentials"), "spec", "secretStoreRef")
	if diff := cmp.Diff(map[string]any{"name": "central-secret-store", "kind": "ClusterSecretStore"}, externalSecretStore); diff != "" {
		t.Fatalf("ExternalSecret store reference differs (-want +got):\n%s", diff)
	}

	status := nestedMap(t, rsp.GetDesired().GetComposite().GetResource().AsMap(), "status")
	if got := status["outputSecretPath"]; got != "/example/platform/grafana/stacks/prod-us-central-0/production/teamdemo01" {
		t.Fatalf("status output path = %v", got)
	}
}

func TestDashboardAndHomeReconciliationModes(t *testing.T) {
	tests := map[string]struct {
		claim     string
		field     string
		wantOwner string
	}{
		"create only is the safe default": {
			claim: minimalStack(), field: "configJson", wantOwner: "initProvider",
		},
		"enforced dashboard content": {
			claim: stackDocument(map[string]any{
				"reconciliation": map[string]any{"dashboards": "enforced", "homePreference": "enforced"},
			}),
			field: "configJson", wantOwner: "forProvider",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			rsp := runStack(t, tc.claim, nil)
			dashboardSpec := nestedMap(t, desiredResource(t, rsp, "homepage-dashboard"), "spec")
			if _, ok := nestedMap(t, dashboardSpec, tc.wantOwner)[tc.field]; !ok {
				t.Fatalf("%s.%s is absent", tc.wantOwner, tc.field)
			}
			other := "forProvider"
			if tc.wantOwner == other {
				other = "initProvider"
			}
			if values, ok := dashboardSpec[other].(map[string]any); ok {
				if _, exists := values[tc.field]; exists {
					t.Fatalf("%s.%s is present; field has two reconciliation owners", other, tc.field)
				}
			}

			preferencesSpec := nestedMap(t, desiredResource(t, rsp, "organization-preferences"), "spec")
			wantPreferenceOwner := "initProvider"
			if strings.Contains(tc.claim, `"homePreference":"enforced"`) {
				wantPreferenceOwner = "forProvider"
			}
			if _, ok := nestedMap(t, preferencesSpec, wantPreferenceOwner)["homeDashboardUid"]; !ok {
				t.Fatalf("%s.homeDashboardUid is absent", wantPreferenceOwner)
			}
		})
	}
}

func TestSSOReconciliationModesUseStableIdentity(t *testing.T) {
	for _, mode := range []string{"enforced", "createOnly", "observeOnly", "disabled"} {
		t.Run(mode, func(t *testing.T) {
			claim := stackDocument(map[string]any{"sso": map[string]any{"mode": mode, "profile": "corporate"}})
			rsp := runStackWithInput(t, claim, nil, platformConfig())
			sso, exists := rsp.GetDesired().GetResources()["sso"]
			if mode == "disabled" {
				if exists {
					t.Fatal("disabled SSO rendered a managed resource")
				}
				return
			}
			if !exists {
				t.Fatal("SSO managed resource was not rendered")
			}
			object := sso.GetResource().AsMap()
			annotations := nestedMap(t, object, "metadata", "annotations")
			if got := annotations["crossplane.io/external-name"]; got != "generic_oauth" {
				t.Fatalf("SSO external identity = %v", got)
			}
			spec := nestedMap(t, object, "spec")
			policies := spec["managementPolicies"].([]any)
			if mode == "observeOnly" {
				if diff := cmp.Diff([]any{"Observe"}, policies); diff != "" {
					t.Fatalf("observe-only policies differ (-want +got):\n%s", diff)
				}
				if _, ok := spec["initProvider"]; ok {
					t.Fatal("observe-only SSO contains initProvider")
				}
				return
			}

			owner := "forProvider"
			if mode == "createOnly" {
				owner = "initProvider"
			}
			settings, ok := nestedMap(t, spec, owner)["oauth2Settings"]
			if !ok {
				t.Fatalf("%s.oauth2Settings is absent", owner)
			}
			if strings.Contains(mustJSON(settings), "literal-client-secret") {
				t.Fatal("SSO resource contains a literal client secret")
			}
		})
	}
}

func TestMonthlyReportIsOptionalValidatedAndDeterministic(t *testing.T) {
	if _, ok := runStack(t, minimalStack(), nil).GetDesired().GetResources()["monthly-report"]; ok {
		t.Fatal("monthly report is enabled by default")
	}

	claim := stackDocument(map[string]any{
		"monthlyReport": map[string]any{
			"enabled": true, "recipients": []any{"platform@example.com", "owner@example.com"}, "replyTo": "platform@example.com",
		},
	})
	first := desiredResource(t, runStack(t, claim, nil), "monthly-report")
	second := desiredResource(t, runStack(t, claim, nil), "monthly-report")
	if diff := cmp.Diff(first, second); diff != "" {
		t.Fatalf("monthly report is not deterministic (-first +second):\n%s", diff)
	}
	schedule := nestedMap(t, first, "spec", "forProvider")["schedule"].([]any)[0].(map[string]any)
	if got := schedule["frequency"]; got != "monthly" {
		t.Fatalf("report frequency = %v", got)
	}
	if start, _ := schedule["startTime"].(string); !strings.HasPrefix(start, "2020-01-01T") {
		t.Fatalf("report start time is not a stable epoch slot: %q", start)
	}

	invalid := stackDocument(map[string]any{"monthlyReport": map[string]any{"enabled": true}})
	rsp := callFunction(t, invalid, nil, "")
	if fatal := fatalResult(rsp); !strings.Contains(fatal, "monthlyReport") {
		t.Fatalf("invalid report settings returned fatal result %q", fatal)
	}
}

func TestIncidentIntegrationIsOptionalAndSecretBacked(t *testing.T) {
	baseline := runStack(t, minimalStack(), nil)
	for name := range baseline.GetDesired().GetResources() {
		if strings.HasPrefix(name, "incident-") {
			t.Fatalf("incident resource %q is enabled by default", name)
		}
	}

	claim := stackDocument(map[string]any{"incidentIntegration": map[string]any{"enabled": true, "profile": "incident-relay"}})
	rsp := runStackWithInput(t, claim, nil, platformConfig())
	want := []string{
		"incident-oncall-test-firing", "incident-oncall-test-resolved",
		"incident-oncall-production-firing", "incident-oncall-production-resolved",
		"incident-alerting-test", "incident-alerting-production",
	}
	for _, name := range want {
		resource := desiredResource(t, rsp, name)
		serialized := mustJSON(resource)
		if !strings.Contains(serialized, "authorization") || !strings.Contains(serialized, "SecretRef") {
			t.Errorf("%s has no Secret-backed authorization: %s", name, serialized)
		}
		if strings.Contains(serialized, "literal-relay-token") {
			t.Errorf("%s contains a literal relay credential", name)
		}
	}
}

func TestCustomRoleBindingRendersTeamRoleAndAssignment(t *testing.T) {
	claim := `{
		"apiVersion":"platform.example.org/v1beta1",
		"kind":"GrafanaCustomRoleBinding",
		"metadata":{"name":"example-editor","namespace":"grafana-vending"},
		"spec":{
			"stackRef":{"name":"teamdemo01"},
			"team":{"name":"Example Editors","groups":["idp-example-editors"]},
			"role":{"name":"example-editor","uid":"example-editor","description":"Example editor access","permissions":[{"action":"dashboards:read","scope":"folders:*"}]}
		}
	}`
	rsp := runStack(t, claim, nil)
	got := map[string]string{}
	for name, resource := range rsp.GetDesired().GetResources() {
		got[name] = resource.GetResource().GetFields()["kind"].GetStringValue()
	}
	want := map[string]string{"team": "Team", "role": "Role", "role-assignment": "RoleAssignment"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("custom role resources differ (-want +got):\n%s", diff)
	}
	assignment := desiredResource(t, rsp, "role-assignment")
	params := nestedMap(t, assignment, "spec", "forProvider")
	if _, ok := params["roleRef"]; !ok {
		t.Fatal("role assignment has no roleRef")
	}
	if _, ok := params["teamRefs"]; !ok {
		t.Fatal("role assignment has no teamRefs")
	}
}

func TestStackStatusPublishesSafeObservedFields(t *testing.T) {
	observed := map[string]*fnv1.Resource{
		"stack": observedResource(`{
			"apiVersion":"cloud.grafana.m.crossplane.io/v1alpha1","kind":"Stack",
			"status":{"atProvider":{"id":"12345","url":"https://teamdemo01.grafana.net"}}
		}`),
	}
	rsp := runStack(t, minimalStack(), observed)
	status := nestedMap(t, rsp.GetDesired().GetComposite().GetResource().AsMap(), "status")
	want := map[string]any{
		"outputSecretPath":    "/platform/grafana-cloud/stacks/prod-us-central-0/production/teamdemo01",
		"telemetrySecretPath": "/platform/grafana-cloud/stacks/prod-us-central-0/production/teamdemo01/telemetry-publisher",
		"stack":               map[string]any{"id": "12345", "url": "https://teamdemo01.grafana.net"},
	}
	if diff := cmp.Diff(want, status); diff != "" {
		t.Fatalf("stack status differs (-want +got):\n%s", diff)
	}
}

func TestStackStatusDoesNotEchoObservedCompositeFields(t *testing.T) {
	claim := strings.Replace(
		minimalStack(),
		`"metadata":{`,
		`"metadata":{"managedFields":[{"manager":"argocd-controller","operation":"Apply"}],`,
		1,
	)
	rsp := runStack(t, claim, nil)
	desired := rsp.GetDesired().GetComposite().GetResource().AsMap()

	if _, ok := desired["metadata"]; ok {
		t.Fatalf("desired composite echoed observed metadata: %s", mustJSON(desired["metadata"]))
	}
	if _, ok := desired["spec"]; ok {
		t.Fatalf("desired composite echoed observed spec: %s", mustJSON(desired["spec"]))
	}
	if _, ok := desired["status"]; !ok {
		t.Fatal("desired composite does not contain status")
	}
}

func minimalStack() string {
	return stackDocument(nil)
}

func stackDocument(additions map[string]any) string {
	spec := map[string]any{
		"displayName": "Example Stack 01",
		"slug":        "teamdemo01",
		"region":      "prod-us-central-0",
		"usage":       "production",
	}
	for key, value := range additions {
		spec[key] = value
	}
	document := map[string]any{
		"apiVersion": "platform.example.org/v1beta1",
		"kind":       "GrafanaCloudStackRequest",
		"metadata":   map[string]any{"name": "teamdemo01", "namespace": "grafana-vending"},
		"spec":       spec,
	}
	b, _ := json.Marshal(document)
	return string(b)
}

func platformConfig() string {
	return `{
		"apiVersion":"platform.example.org/v1beta1",
		"kind":"GrafanaVendingConfig",
		"spec":{
			"ssoProfiles":[{
				"name":"corporate","providerName":"generic_oauth",
				"oauth2Settings":{
					"name":"Example SSO","enabled":true,"allowSignUp":true,
					"authUrl":"https://identity.example.com/oauth2/authorize",
					"tokenUrl":"https://identity.example.com/oauth2/token",
					"apiUrl":"https://identity.example.com/oauth2/userinfo",
					"clientId":"grafana-example","scopes":"openid profile email",
					"emailAttributeName":"email","groupsAttributePath":"groups",
					"roleAttributePath":"contains(groups[*], 'grafana-admins') && 'Admin' || 'Viewer'",
					"roleAttributeStrict":true,
					"clientSecretSecretRef":{"name":"grafana-sso-corporate","key":"clientSecret"}
				}
			}],
			"incidentProfiles":[{
				"name":"incident-relay","url":"https://incident-relay.example.com/grafana/events",
				"authorizationSecretRef":{"name":"grafana-incident-relay","key":"authorization"}
			}]
		}
	}`
}

func runStack(t *testing.T, composite string, observed map[string]*fnv1.Resource) *fnv1.RunFunctionResponse {
	t.Helper()
	return runStackWithInput(t, composite, observed, "")
}

func runStackWithInput(t *testing.T, composite string, observed map[string]*fnv1.Resource, input string) *fnv1.RunFunctionResponse {
	t.Helper()
	rsp := callFunction(t, composite, observed, input)
	if fatal := fatalResult(rsp); fatal != "" {
		t.Fatalf("RunFunction returned a fatal result: %s", fatal)
	}
	return rsp
}

func callFunction(t *testing.T, composite string, observed map[string]*fnv1.Resource, input string) *fnv1.RunFunctionResponse {
	t.Helper()
	req := &fnv1.RunFunctionRequest{
		Observed: &fnv1.State{
			Composite: &fnv1.Resource{Resource: resource.MustStructJSON(composite)},
			Resources: observed,
		},
	}
	if input != "" {
		req.Input = resource.MustStructJSON(input)
	}
	rsp, err := (&Function{log: logging.NewNopLogger()}).RunFunction(context.Background(), req)
	if err != nil {
		t.Fatalf("RunFunction returned an error: %v", err)
	}
	return rsp
}

func observedResource(document string) *fnv1.Resource {
	return &fnv1.Resource{Resource: resource.MustStructJSON(document)}
}

func desiredResource(t *testing.T, rsp *fnv1.RunFunctionResponse, name string) map[string]any {
	t.Helper()
	r, ok := rsp.GetDesired().GetResources()[name]
	if !ok {
		t.Fatalf("desired resource %q was not rendered", name)
	}
	return r.GetResource().AsMap()
}

func nestedMap(t *testing.T, object map[string]any, fields ...string) map[string]any {
	t.Helper()
	current := object
	for _, field := range fields {
		next, ok := current[field].(map[string]any)
		if !ok {
			t.Fatalf("field %q in path %v is %T, want object", field, fields, current[field])
		}
		current = next
	}
	return current
}

func fatalResult(rsp *fnv1.RunFunctionResponse) string {
	for _, result := range rsp.GetResults() {
		if result.GetSeverity() == fnv1.Severity_SEVERITY_FATAL {
			return result.GetMessage()
		}
	}
	return ""
}

func mustJSON(value any) string {
	b, _ := json.Marshal(value)
	return string(b)
}
