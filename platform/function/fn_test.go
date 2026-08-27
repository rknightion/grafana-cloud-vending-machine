package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
)

func TestBaselineResourcesUseDeterministicExternalNames(t *testing.T) {
	rsp := runStack(t, minimalStack(), foundationReadyObserved())

	want := map[string]string{
		"stack":               "teamdemo01",
		"billing-folder":      "vending-billing-folder",
		"billing-dashboard":   "vending-billing",
		"endpoints-folder":    "vending-endpoints-folder",
		"endpoints-dashboard": "vending-endpoints",
		"homepage-folder":     "vending-home-folder",
		"homepage-dashboard":  "vending-home",
	}
	for name, externalName := range want {
		if got := desiredExternalName(t, desiredResource(t, rsp, name)); got != externalName {
			t.Errorf("%s external name = %q, want %q", name, got, externalName)
		}
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

func TestMalformedLifecycleIsRejected(t *testing.T) {
	for _, test := range []struct {
		name  string
		value any
		want  string
	}{
		{name: "object", value: "Delete", want: "spec.lifecycle must be an object"},
		{name: "external resources", value: map[string]any{"externalResources": true}, want: "spec.lifecycle.externalResources must be a string"},
	} {
		t.Run(test.name, func(t *testing.T) {
			claim := stackDocument(map[string]any{"lifecycle": test.value})
			rsp := callFunction(t, claim, nil, "")

			if fatal := fatalResult(rsp); !strings.Contains(fatal, test.want) {
				t.Fatalf("malformed lifecycle returned fatal result %q", fatal)
			}
		})
	}
}

func TestUnlistedUsageIsRejected(t *testing.T) {
	claim := stackDocument(map[string]any{"usage": "preview"})
	rsp := callFunction(t, claim, nil, `{"spec":{"allowedUsages":["development","production"]}}`)

	if fatal := fatalResult(rsp); !strings.Contains(fatal, "usage \"preview\" is not allowed") {
		t.Fatalf("unlisted usage returned fatal result %q", fatal)
	}
}

func TestPlatformAllowedUsageIsAccepted(t *testing.T) {
	claim := stackDocument(map[string]any{"usage": "preview"})
	rsp := callFunction(t, claim, nil, `{"spec":{"allowedUsages":["preview"]}}`)

	if fatal := fatalResult(rsp); fatal != "" {
		t.Fatalf("allowed usage returned fatal result %q", fatal)
	}
}

func TestDeleteLifecycleRequiresAnAuthorizedProfile(t *testing.T) {
	claim := stackDocument(map[string]any{"lifecycle": map[string]any{"externalResources": "Delete"}})
	rsp := callFunction(t, claim, nil, "")

	if fatal := fatalResult(rsp); !strings.Contains(fatal, "request grafana-vending/teamdemo01 with profile \"standard\" is not authorized") {
		t.Fatalf("disallowed delete returned fatal result %q", fatal)
	}
}

func TestDeleteLifecycleAuthorizationIsBoundToRequestIdentity(t *testing.T) {
	claim := stackDocument(map[string]any{"lifecycle": map[string]any{"externalResources": "Delete"}})
	input := `{"spec":{"deletionAuthorizations":[{"namespace":"grafana-vending","name":"teamdemo01","uid":"previous-request-uid","profile":"standard"}]}}`
	rsp := callFunction(t, claim, nil, input)

	if fatal := fatalResult(rsp); !strings.Contains(fatal, "request grafana-vending/teamdemo01 with profile \"standard\" is not authorized") {
		t.Fatalf("wrong-request authorization returned fatal result %q", fatal)
	}
}

func TestAuthorizedDeleteLifecycleProjectsDeletionToExternalResources(t *testing.T) {
	observed := foundationReadyObserved()
	observed["telemetry-access-policy"] = observedResource(`{
		"apiVersion":"cloud.grafana.m.crossplane.io/v1alpha1",
		"kind":"AccessPolicy",
		"metadata":{"name":"teamdemo01-telemetry-publisher","namespace":"grafana-vending"},
		"status":{"atProvider":{"policyId":"policy-12345"}}
	}`)
	claim := stackDocument(map[string]any{"lifecycle": map[string]any{"externalResources": "Delete"}})
	rsp := runStackWithInput(t, claim, observed, deletionAuthorizationInput())

	deletePolicies := []any{"*"}
	for _, name := range []string{"stack", "stack-service-account", "stack-token", "telemetry-access-policy", "telemetry-token"} {
		spec := nestedMap(t, desiredResource(t, rsp, name), "spec")
		if diff := cmp.Diff(deletePolicies, spec["managementPolicies"]); diff != "" {
			t.Errorf("%s management policies differ (-want +got):\n%s", name, diff)
		}
	}
	if got := nestedMap(t, desiredResource(t, rsp, "stack"), "spec", "forProvider")["deleteProtection"]; got != false {
		t.Errorf("stack delete protection = %v, want false", got)
	}
	for _, name := range []string{"stack-token", "telemetry-token"} {
		if got := nestedMap(t, desiredResource(t, rsp, name), "spec", "forProvider")["deleteOnDestroy"]; got != true {
			t.Errorf("%s deleteOnDestroy = %v, want true", name, got)
		}
	}
	for _, name := range []string{"credentials", "telemetry-credentials"} {
		if got := nestedMap(t, desiredResource(t, rsp, name), "spec")["deletionPolicy"]; got != "Delete" {
			t.Errorf("%s deletion policy = %v, want Delete", name, got)
		}
	}
	if got := nestedMap(t, desiredResource(t, rsp, "billing-folder"), "spec")["managementPolicies"]; !cmp.Equal(managementPolicies, got) {
		t.Errorf("stack-local billing-folder management policies = %v, want %v", got, managementPolicies)
	}
}

func TestPushSecretDocumentsCarryTheDeletionGuardTag(t *testing.T) {
	rsp := runStack(t, minimalStack(), foundationReadyObserved())
	want := map[string]any{"grafana-cloud-vending-machine": "managed"}

	for _, name := range []string{"credentials", "telemetry-credentials"} {
		spec := nestedMap(t, desiredResource(t, rsp, name), "spec")
		data, ok := spec["data"].([]any)
		if !ok || len(data) != 1 {
			t.Fatalf("%s data = %T %v, want one entry", name, spec["data"], spec["data"])
		}
		entry, ok := data[0].(map[string]any)
		if !ok {
			t.Fatalf("%s data entry = %T, want object", name, data[0])
		}
		if diff := cmp.Diff(want, nestedMap(t, entry, "metadata", "spec", "tags")); diff != "" {
			t.Errorf("%s deletion guard tags differ (-want +got):\n%s", name, diff)
		}
	}
}

func TestAuthorizedDeleteLifecycleWaitsForObservedProtectionToBeDisabled(t *testing.T) {
	claim := stackDocument(map[string]any{"lifecycle": map[string]any{"externalResources": "Delete"}})
	input := deletionAuthorizationInput()

	untilObserved := runStackWithInput(t, claim, nil, input)
	status := nestedMap(t, untilObserved.GetDesired().GetComposite().GetResource().AsMap(), "status")
	if got := status["deletionArmed"]; got != true {
		t.Errorf("deletion armed = %v, want true", got)
	}
	if got := status["deletionReady"]; got != false {
		t.Errorf("deletion ready without observed protection = %v, want false", got)
	}

	protected := map[string]*fnv1.Resource{
		"stack": observedResource(`{
			"apiVersion":"cloud.grafana.m.crossplane.io/v1alpha1",
			"kind":"Stack",
			"metadata":{"name":"teamdemo01","namespace":"grafana-vending"},
			"status":{"atProvider":{"deleteProtection":true}}
		}`),
	}
	stillProtected := runStackWithInput(t, claim, protected, input)
	status = nestedMap(t, stillProtected.GetDesired().GetComposite().GetResource().AsMap(), "status")
	if got := status["deletionReady"]; got != false {
		t.Errorf("deletion ready while observed protection remains enabled = %v, want false", got)
	}

	observed := map[string]*fnv1.Resource{
		"stack": observedResource(`{
			"apiVersion":"cloud.grafana.m.crossplane.io/v1alpha1",
			"kind":"Stack",
			"metadata":{"name":"teamdemo01","namespace":"grafana-vending"},
			"status":{"atProvider":{"deleteProtection":false}}
		}`),
	}
	notPrepared := runStackWithInput(t, claim, observed, input)
	status = nestedMap(t, notPrepared.GetDesired().GetComposite().GetResource().AsMap(), "status")
	if got := status["deletionReady"]; got != false {
		t.Errorf("deletion ready before PushSecrets are prepared = %v, want false", got)
	}

	observed["credentials"] = observedResource(`{
		"apiVersion":"external-secrets.io/v1alpha1",
		"kind":"PushSecret",
		"metadata":{"name":"teamdemo01-credentials","namespace":"grafana-vending","generation":2},
		"spec":{"deletionPolicy":"Delete"},
		"status":{
			"syncedResourceVersion":"2-example",
			"syncedPushSecrets":{"SecretStore/grafana-vending-secrets":{"vending.json":{}}},
			"conditions":[{"type":"Ready","status":"True","reason":"Synced"}]
		}
	}`)
	observed["telemetry-credentials"] = deletionPreparedPushSecret("teamdemo01-telemetry-credentials")
	notFinalized := runStackWithInput(t, claim, observed, input)
	status = nestedMap(t, notFinalized.GetDesired().GetComposite().GetResource().AsMap(), "status")
	if got := status["deletionReady"]; got != false {
		t.Errorf("deletion ready before ESO finalizer is installed = %v, want false", got)
	}

	observed["credentials"] = deletionPreparedPushSecret("teamdemo01-credentials")
	ready := runStackWithInput(t, claim, observed, input)
	status = nestedMap(t, ready.GetDesired().GetComposite().GetResource().AsMap(), "status")
	if got := status["deletionReady"]; got != false {
		t.Errorf("deletion ready before rotating tokens are deletion-managed = %v, want false", got)
	}

	observed["stack-token"] = deletionPreparedToken("StackServiceAccountRotatingToken", "teamdemo01-admin", "serviceAccountId", "67890")
	observed["telemetry-token"] = deletionPreparedToken("AccessPolicyRotatingToken", "teamdemo01-telemetry-publisher", "accessPolicyId", "policy-12345")
	ready = runStackWithInput(t, claim, observed, input)
	status = nestedMap(t, ready.GetDesired().GetComposite().GetResource().AsMap(), "status")
	if got := status["deletionReady"]; got != true {
		t.Errorf("deletion ready after Stack and PushSecrets are prepared = %v, want true", got)
	}
}

func TestObservedRotatingTokensSurviveTransientParentIDLoss(t *testing.T) {
	observed := foundationReadyObserved()
	observed["stack-service-account"] = observedResource(`{
		"apiVersion":"cloud.grafana.m.crossplane.io/v1alpha1",
		"kind":"StackServiceAccount",
		"metadata":{"name":"teamdemo01-admin","namespace":"grafana-vending"}
	}`)
	observed["stack-token"] = deletionPreparedToken("StackServiceAccountRotatingToken", "teamdemo01-admin", "serviceAccountId", "67890")
	observed["telemetry-access-policy"] = observedResource(`{
		"apiVersion":"cloud.grafana.m.crossplane.io/v1alpha1",
		"kind":"AccessPolicy",
		"metadata":{"name":"teamdemo01-telemetry-publisher","namespace":"grafana-vending"}
	}`)
	observed["telemetry-token"] = deletionPreparedToken("AccessPolicyRotatingToken", "teamdemo01-telemetry-publisher", "accessPolicyId", "policy-12345")

	rsp := runStack(t, minimalStack(), observed)
	if got := nestedMap(t, desiredResource(t, rsp, "stack-token"), "spec", "forProvider")["serviceAccountId"]; got != "67890" {
		t.Errorf("retained administrator token service account ID = %v, want 67890", got)
	}
	if got := nestedMap(t, desiredResource(t, rsp, "telemetry-token"), "spec", "forProvider")["accessPolicyId"]; got != "policy-12345" {
		t.Errorf("retained telemetry token access policy ID = %v, want policy-12345", got)
	}
}

func deletionAuthorizationInput() string {
	return `{"spec":{"deletionAuthorizations":[{"namespace":"grafana-vending","name":"teamdemo01","uid":"example-request-uid","profile":"standard"}]}}`
}

func deletionPreparedPushSecret(name string) *fnv1.Resource {
	return observedResource(fmt.Sprintf(`{
		"apiVersion":"external-secrets.io/v1alpha1",
		"kind":"PushSecret",
		"metadata":{
			"name":%q,
			"namespace":"grafana-vending",
			"generation":2,
			"finalizers":["pushsecret.externalsecrets.io/finalizer"]
		},
		"spec":{"deletionPolicy":"Delete"},
		"status":{
			"syncedResourceVersion":"2-example",
			"syncedPushSecrets":{"SecretStore/grafana-vending-secrets":{"vending.json":{}}},
			"conditions":[{"type":"Ready","status":"True","reason":"Synced"}]
		}
	}`, name))
}

func deletionPreparedToken(kind, name, idField, id string) *fnv1.Resource {
	return observedResource(mustJSON(map[string]any{
		"apiVersion": "cloud.grafana.m.crossplane.io/v1alpha1",
		"kind":       kind,
		"metadata":   map[string]any{"name": name, "namespace": "grafana-vending"},
		"spec": map[string]any{
			"managementPolicies": []any{"*"},
			"forProvider": map[string]any{
				idField:           id,
				"deleteOnDestroy": true,
			},
		},
	}))
}

func TestOmittedAndRetainLifecycleKeepExternalResourcesRetained(t *testing.T) {
	for _, test := range []struct {
		name      string
		lifecycle map[string]any
	}{
		{name: "omitted"},
		{name: "retain", lifecycle: map[string]any{"externalResources": "Retain"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			additions := map[string]any{}
			if test.lifecycle != nil {
				additions["lifecycle"] = test.lifecycle
			}
			rsp := runStack(t, stackDocument(additions), foundationReadyObserved())

			stack := nestedMap(t, desiredResource(t, rsp, "stack"), "spec")
			if diff := cmp.Diff(managementPolicies, stack["managementPolicies"]); diff != "" {
				t.Errorf("stack management policies differ (-want +got):\n%s", diff)
			}
			if got := nestedMap(t, stack, "forProvider")["deleteProtection"]; got != true {
				t.Errorf("stack delete protection = %v, want true", got)
			}
			if got := nestedMap(t, desiredResource(t, rsp, "stack-token"), "spec", "forProvider")["deleteOnDestroy"]; got != false {
				t.Errorf("stack token deleteOnDestroy = %v, want false", got)
			}
			if got := nestedMap(t, desiredResource(t, rsp, "credentials"), "spec")["deletionPolicy"]; got != "None" {
				t.Errorf("credentials deletion policy = %v, want None", got)
			}
			status := nestedMap(t, rsp.GetDesired().GetComposite().GetResource().AsMap(), "status")
			for _, field := range []string{"deletionArmed", "deletionReady"} {
				if got := status[field]; got != false {
					t.Errorf("retained lifecycle status.%s = %v, want false", field, got)
				}
			}
		})
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
	rsp := runStack(t, claim, foundationReadyObserved())
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
			rsp := runStack(t, tc.claim, foundationReadyObserved())
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
			rsp := runStackWithInput(t, claim, foundationReadyObserved(), platformConfig())
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

func TestOAuthSSORejectsLiteralClientSecretWithoutSecretReference(t *testing.T) {
	claim := stackDocument(map[string]any{"sso": map[string]any{"mode": "enforced", "profile": "unsafe"}})
	input := `{
		"apiVersion":"platform.example.org/v1beta1","kind":"GrafanaVendingConfig",
		"spec":{"ssoProfiles":[{
			"name":"unsafe","providerName":"generic_oauth",
			"oauth2Settings":{"clientId":"example","clientSecret":"literal-client-secret"}
		}]}
	}`
	rsp := callFunction(t, claim, nil, input)
	if fatal := fatalResult(rsp); !strings.Contains(fatal, "clientSecretSecretRef") {
		t.Fatalf("literal-only OAuth secret returned fatal result %q", fatal)
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
	first := desiredResource(t, runStack(t, claim, foundationReadyObserved()), "monthly-report")
	second := desiredResource(t, runStack(t, claim, foundationReadyObserved()), "monthly-report")
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
	rsp := runStackWithInput(t, claim, foundationReadyObserved(), platformConfig())
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
	webhookSpec := nestedMap(t, desiredResource(t, rsp, "incident-oncall-test-firing"), "spec")
	if _, ok := nestedMap(t, webhookSpec, "initProvider")["data"]; !ok {
		t.Fatal("default incident webhook does not preserve later data-template edits")
	}
	if _, ok := nestedMap(t, webhookSpec, "forProvider")["data"]; ok {
		t.Fatal("default incident webhook has two data-template owners")
	}
	contactPoint := nestedMap(t, desiredResource(t, rsp, "incident-alerting-test"), "spec", "forProvider")
	if _, ok := contactPoint["webhook"]; !ok {
		t.Fatal("alerting contact-point payload is not enforced")
	}

	enforcedClaim := stackDocument(map[string]any{"incidentIntegration": map[string]any{
		"enabled": true, "profile": "incident-relay", "templateMode": "enforced",
	}})
	enforced := runStackWithInput(t, enforcedClaim, foundationReadyObserved(), platformConfig())
	enforcedSpec := nestedMap(t, desiredResource(t, enforced, "incident-oncall-test-firing"), "spec")
	if _, ok := nestedMap(t, enforcedSpec, "forProvider")["data"]; !ok {
		t.Fatal("enforced incident webhook does not own its data template")
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
	rsp := runAccess(t, claim, nil)
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
	role := desiredResource(t, rsp, "role")
	if got := desiredExternalName(t, role); got != "example-editor" {
		t.Fatalf("role external name = %q, want example-editor", got)
	}
	roleParams := nestedMap(t, role, "spec", "forProvider")
	if got := roleParams["autoIncrementVersion"]; got != false {
		t.Fatalf("role autoIncrementVersion = %v, want false", got)
	}
	if _, ok := roleParams["version"]; ok {
		t.Fatal("role owns the server-managed version field")
	}
	if got := desiredExternalName(t, assignment); got != "example-editor" {
		t.Fatalf("role assignment external name = %q, want example-editor", got)
	}
}

func TestTeamAccessRendersMembershipPreferencesAndAdditiveRoleAssignments(t *testing.T) {
	claim := `{
		"apiVersion":"platform.example.org/v1beta1",
		"kind":"GrafanaTeamAccess",
		"metadata":{"name":"example-editors","namespace":"grafana-vending"},
		"spec":{
			"stackRef":{"name":"teamdemo01"},
			"team":{
				"name":"Example Editors","email":"editors@example.com",
				"members":["engineer@example.com"],"externalGroups":["idp-example-editors"],
				"ignoreExternallySyncedMembers":true,
				"preferences":{"homeDashboardUid":"vending-home","theme":"system","timezone":"browser","weekStart":"monday"}
			},
			"customRoles":[{
				"name":"example-alert-editor","uid":"example-alert-editor","displayName":"Example alert editor",
				"permissions":[{"action":"alert.rules:read","scope":"folders:*"},{"action":"alert.rules:write","scope":"folders:*"}]
			}],
			"fixedRoleUids":["fixed_C2x8IxkiBc1KZVjyYH775T9jNMQ"]
		}
	}`
	customSuffix := stableResourceSuffix("example-alert-editor")
	fixedSuffix := stableResourceSuffix("fixed_C2x8IxkiBc1KZVjyYH775T9jNMQ")
	initial := runAccess(t, claim, nil)
	for _, name := range []string{"custom-role-assignment-" + customSuffix, "fixed-role-assignment-" + fixedSuffix} {
		if _, ok := initial.GetDesired().GetResources()[name]; ok {
			t.Fatalf("%s was rendered before Grafana assigned the Team ID", name)
		}
	}

	observed := map[string]*fnv1.Resource{
		"team": observedResource(`{
			"apiVersion":"oss.grafana.m.crossplane.io/v1alpha1",
			"kind":"Team",
			"status":{"atProvider":{"teamId":42}}
		}`),
	}
	rsp := runAccess(t, claim, observed)
	got := map[string]string{}
	for name, desired := range rsp.GetDesired().GetResources() {
		got[name] = desired.GetResource().GetFields()["kind"].GetStringValue()
	}
	want := map[string]string{
		"team":                                   "Team",
		"custom-role-" + customSuffix:            "Role",
		"custom-role-assignment-" + customSuffix: "RoleAssignmentItem",
		"fixed-role-assignment-" + fixedSuffix:   "RoleAssignmentItem",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("team access resources differ (-want +got):\n%s", diff)
	}

	team := nestedMap(t, desiredResource(t, rsp, "team"), "spec", "forProvider")
	if got := team["members"].([]any); len(got) != 1 || got[0] != "engineer@example.com" {
		t.Fatalf("direct team members = %v", got)
	}
	teamSync := team["teamSync"].([]any)[0].(map[string]any)
	if got := teamSync["groups"].([]any); len(got) != 1 || got[0] != "idp-example-editors" {
		t.Fatalf("team sync groups = %v", got)
	}
	preferences := team["preferences"].([]any)[0].(map[string]any)
	if got := preferences["homeDashboardUid"]; got != "vending-home" {
		t.Fatalf("team home dashboard = %v", got)
	}

	customAssignment := nestedMap(t, desiredResource(t, rsp, "custom-role-assignment-"+customSuffix), "spec", "forProvider")
	if _, ok := customAssignment["roleRef"]; !ok {
		t.Fatal("custom role assignment has no roleRef")
	}
	if got := customAssignment["teamId"]; got != "42" {
		t.Fatalf("custom role assignment team ID = %v, want 42", got)
	}
	if _, ok := customAssignment["teamRef"]; ok {
		t.Fatal("custom role assignment uses an org-qualified Team reference")
	}
	customRole := desiredResource(t, rsp, "custom-role-"+customSuffix)
	if got := desiredExternalName(t, customRole); got != "example-alert-editor" {
		t.Fatalf("custom role external name = %q, want example-alert-editor", got)
	}
	customRoleParams := nestedMap(t, customRole, "spec", "forProvider")
	if got := customRoleParams["autoIncrementVersion"]; got != false {
		t.Fatalf("custom role autoIncrementVersion = %v, want false", got)
	}
	if _, ok := customRoleParams["version"]; ok {
		t.Fatal("custom role owns the server-managed version field")
	}
	customAssignmentResource := desiredResource(t, rsp, "custom-role-assignment-"+customSuffix)
	if got := desiredExternalName(t, customAssignmentResource); got != "example-alert-editor:team:42" {
		t.Fatalf("custom role assignment external name = %q, want example-alert-editor:team:42", got)
	}
	fixedAssignmentResource := desiredResource(t, rsp, "fixed-role-assignment-"+fixedSuffix)
	fixedAssignment := nestedMap(t, fixedAssignmentResource, "spec", "forProvider")
	if got := fixedAssignment["roleUid"]; got != "fixed_C2x8IxkiBc1KZVjyYH775T9jNMQ" {
		t.Fatalf("fixed role UID = %v", got)
	}
	if got := fixedAssignment["teamId"]; got != "42" {
		t.Fatalf("fixed role assignment team ID = %v, want 42", got)
	}
	if got := desiredExternalName(t, fixedAssignmentResource); got != "fixed_C2x8IxkiBc1KZVjyYH775T9jNMQ:team:42" {
		t.Fatalf("fixed role assignment external name = %q", got)
	}
	if _, ok := team["admins"]; ok {
		t.Fatal("team owns admins although the request declared no administrators")
	}
}

// The provider gained a separate admins set, and members now means ordinary
// membership only. Absent admins leaves administrators created outside Crossplane
// alone; an explicitly empty admins demotes them. So an unset request field must
// not render the key at all, and a set one must render exactly what was asked for,
// empty list included.
func TestTeamAccessOwnsAdministratorsOnlyWhenTheRequestDeclaresThem(t *testing.T) {
	claim := func(administrators any) string {
		team := map[string]any{"name": "Example Editors", "members": []any{"engineer@example.com"}}
		if administrators != nil {
			team["administrators"] = administrators
		}
		return mustJSON(map[string]any{
			"apiVersion": "platform.example.org/v1beta1", "kind": "GrafanaTeamAccess",
			"metadata": map[string]any{"name": "example-editors", "namespace": "grafana-vending"},
			"spec": map[string]any{
				"stackRef": map[string]any{"name": "teamdemo01"},
				"team":     team,
			},
		})
	}

	declared := nestedMap(t, desiredResource(t, runAccess(t, claim([]any{"lead@example.com"}), nil), "team"), "spec", "forProvider")
	if got := declared["admins"].([]any); len(got) != 1 || got[0] != "lead@example.com" {
		t.Fatalf("team admins = %v", got)
	}
	if got := declared["members"].([]any); len(got) != 1 || got[0] != "engineer@example.com" {
		t.Fatalf("team members = %v, want the ordinary member only", got)
	}

	emptied := nestedMap(t, desiredResource(t, runAccess(t, claim([]any{}), nil), "team"), "spec", "forProvider")
	admins, ok := emptied["admins"].([]any)
	if !ok || len(admins) != 0 {
		t.Fatalf("explicitly empty administrators must render an empty admins set, got %v", emptied["admins"])
	}

	omitted := nestedMap(t, desiredResource(t, runAccess(t, claim(nil), nil), "team"), "spec", "forProvider")
	if _, ok := omitted["admins"]; ok {
		t.Fatal("omitted administrators must not render admins, which would demote administrators set outside Crossplane")
	}
}

func TestTeamAccessRoleResourceIdentityDoesNotDependOnListOrder(t *testing.T) {
	roleA := map[string]any{"name": "Role A", "uid": "role-a", "permissions": []any{map[string]any{"action": "dashboards:read"}}}
	roleB := map[string]any{"name": "Role B", "uid": "role-b", "permissions": []any{map[string]any{"action": "folders:read"}}}
	document := func(roles []any, fixed []any) string {
		return mustJSON(map[string]any{
			"apiVersion": "platform.example.org/v1beta1", "kind": "GrafanaTeamAccess",
			"metadata": map[string]any{"name": "stable-team", "namespace": "grafana-vending"},
			"spec": map[string]any{
				"stackRef":    map[string]any{"name": "teamdemo01"},
				"team":        map[string]any{"name": "Stable Team"},
				"customRoles": roles, "fixedRoleUids": fixed,
			},
		})
	}
	observed := map[string]*fnv1.Resource{
		"team": observedResource(`{"status":{"atProvider":{"teamId":42}}}`),
	}
	first := runAccess(t, document([]any{roleA, roleB}, []any{"fixed_C2x8IxkiBc1KZVjyYH775T9jNMQ", "fixed_Sgr67JTOhjQGFlzYRahOe45TdWM"}), observed)
	second := runAccess(t, document([]any{roleB, roleA}, []any{"fixed_Sgr67JTOhjQGFlzYRahOe45TdWM", "fixed_C2x8IxkiBc1KZVjyYH775T9jNMQ"}), observed)
	serialized := func(rsp *fnv1.RunFunctionResponse) map[string]string {
		result := map[string]string{}
		for name, desired := range rsp.GetDesired().GetResources() {
			result[name] = mustJSON(desired.GetResource().AsMap())
		}
		return result
	}
	if diff := cmp.Diff(serialized(first), serialized(second)); diff != "" {
		t.Fatalf("role resource identity depends on list order (-first +second):\n%s", diff)
	}
}

func TestContentAccessPolicyOwnsWholeFolderOrDashboardACL(t *testing.T) {
	claim := `{
		"apiVersion":"platform.example.org/v1beta1",
		"kind":"GrafanaContentAccessPolicy",
		"metadata":{"name":"billing-access","namespace":"grafana-vending"},
		"spec":{
			"stackRef":{"name":"teamdemo01"},
			"target":{"kind":"Folder","ref":{"name":"teamdemo01-billing"}},
			"permissions":[
				{"basicRole":"Viewer","permission":"View"},
				{"teamRef":{"name":"example-editors-team"},"permission":"Edit"}
			]
		}
	}`
	rsp := runAccess(t, claim, nil)
	policy := desiredResource(t, rsp, "access-policy")
	if got := policy["kind"]; got != "FolderPermission" {
		t.Fatalf("content access kind = %v", got)
	}
	params := nestedMap(t, policy, "spec", "forProvider")
	if got := nestedMap(t, params, "folderRef")["name"]; got != "teamdemo01-billing" {
		t.Fatalf("folder reference = %v", got)
	}
	permissions := params["permissions"].([]any)
	if got := permissions[0].(map[string]any)["role"]; got != "Viewer" {
		t.Fatalf("basic role permission = %v", got)
	}
	if got := nestedMap(t, permissions[1].(map[string]any), "teamRef")["name"]; got != "example-editors-team" {
		t.Fatalf("team permission reference = %v", got)
	}

	invalid := strings.Replace(claim, `"basicRole":"Viewer"`, `"basicRole":"Viewer","userId":"42"`, 1)
	if fatal := fatalResult(callFunction(t, invalid, nil, "")); !strings.Contains(fatal, "exactly one") {
		t.Fatalf("ambiguous ACL actor returned fatal result %q", fatal)
	}

	directUID := strings.Replace(claim, `"kind":"Folder","ref":{"name":"teamdemo01-billing"}`, `"kind":"Dashboard","ref":{"uid":"vending-home"}`, 1)
	directPolicy := desiredResource(t, runAccess(t, directUID, nil), "access-policy")
	if got := desiredExternalName(t, directPolicy); got != "vending-home" {
		t.Fatalf("direct-UID content policy external name = %q, want vending-home", got)
	}
}

func TestAccessClaimsRequestTheirReferencedStack(t *testing.T) {
	for kind, claim := range accessClaims() {
		t.Run(kind, func(t *testing.T) {
			rsp := callFunctionWithRequiredResources(t, claim, nil, nil, nil)

			selector, ok := rsp.GetRequirements().GetResources()["referenced-stack"]
			if !ok {
				t.Fatal("access claim did not request the referenced stack")
			}
			if got, want := selector.GetApiVersion(), "platform.example.org/v1beta1"; got != want {
				t.Fatalf("referenced stack API version = %q, want %q", got, want)
			}
			if got, want := selector.GetKind(), "GrafanaCloudStackRequest"; got != want {
				t.Fatalf("referenced stack kind = %q, want %q", got, want)
			}
			if got, want := selector.GetMatchName(), "teamdemo01"; got != want {
				t.Fatalf("referenced stack name = %q, want %q", got, want)
			}
			if got, want := selector.GetNamespace(), "grafana-vending"; got != want {
				t.Fatalf("referenced stack namespace = %q, want %q", got, want)
			}
			if got := len(rsp.GetDesired().GetResources()); got != 0 {
				t.Fatalf("unresolved referenced stack rendered %d access children", got)
			}
			if got := rsp.GetDesired().GetComposite().GetReady(); got != fnv1.Ready_READY_FALSE {
				t.Fatalf("unresolved referenced stack readiness = %s, want READY_FALSE", got)
			}
		})
	}
}

func TestReferencedStackSelectorPreservesTheAccessAPIVersion(t *testing.T) {
	claim := strings.Replace(accessClaims()["GrafanaTeamAccess"], "platform.example.org/v1beta1", "custom.example.io/v1alpha1", 1)
	rsp := callFunctionWithRequiredResources(t, claim, nil, nil, nil)

	selector := rsp.GetRequirements().GetResources()["referenced-stack"]
	if got, want := selector.GetApiVersion(), "custom.example.io/v1alpha1"; got != want {
		t.Fatalf("referenced stack API version = %q, want access API version %q", got, want)
	}
}

func TestAccessClaimsAdmitOnlyWhenReferencedStackIsReady(t *testing.T) {
	for kind, claim := range accessClaims() {
		t.Run(kind, func(t *testing.T) {
			unready := callFunctionWithRequiredResources(t, claim, nil, requiredStackResource("teamdemo01", "grafana-vending", "False"), requiredResourceCapabilities())
			if got := len(unready.GetDesired().GetResources()); got != 0 {
				t.Fatalf("not-ready referenced stack rendered %d access children", got)
			}
			if got := unready.GetDesired().GetComposite().GetReady(); got != fnv1.Ready_READY_FALSE {
				t.Fatalf("not-ready referenced stack readiness = %s, want READY_FALSE", got)
			}

			ready := callFunctionWithRequiredResources(t, claim, nil, requiredStackResource("teamdemo01", "grafana-vending", "True"), requiredResourceCapabilities())
			if got := len(ready.GetDesired().GetResources()); got == 0 {
				t.Fatal("ready referenced stack rendered no access children")
			}
			if got := ready.GetDesired().GetComposite().GetReady(); got != fnv1.Ready_READY_UNSPECIFIED {
				t.Fatalf("ready referenced stack readiness = %s, want READY_UNSPECIFIED", got)
			}
		})
	}
}

func TestAccessClaimsCloseAdmissionWhenReferencedStackIsDecommissioning(t *testing.T) {
	claim := accessClaims()["GrafanaTeamAccess"]
	for name, required := range map[string]*fnv1.Resources{
		"armed":       requiredStackResourceState("teamdemo01", "grafana-vending", "True", true, false),
		"terminating": requiredStackResourceState("teamdemo01", "grafana-vending", "True", false, true),
	} {
		t.Run(name, func(t *testing.T) {
			rsp := callFunctionWithRequiredResources(t, claim, nil, required, requiredResourceCapabilities())
			if got := len(rsp.GetDesired().GetResources()); got != 0 {
				t.Fatalf("decommissioning referenced stack rendered %d new access children", got)
			}
			if got := rsp.GetDesired().GetComposite().GetReady(); got != fnv1.Ready_READY_FALSE {
				t.Fatalf("decommissioning referenced stack readiness = %s, want READY_FALSE", got)
			}
		})
	}
}

func TestAccessAdmissionIsOneWay(t *testing.T) {
	claim := accessClaims()["GrafanaTeamAccess"]
	observed := map[string]*fnv1.Resource{
		"team": observedResource(`{"apiVersion":"oss.grafana.m.crossplane.io/v1alpha1","kind":"Team","metadata":{"name":"example-editors-team","namespace":"grafana-vending"}}`),
	}
	rsp := callFunctionWithRequiredResources(t, claim, observed, requiredStackResource("teamdemo01", "grafana-vending", "False"), requiredResourceCapabilities())
	if _, ok := rsp.GetDesired().GetResources()["team"]; !ok {
		t.Fatal("observed access child was removed while referenced stack was not ready")
	}
	if got := len(rsp.GetDesired().GetResources()); got != 1 {
		t.Fatalf("not-ready access claim retained %d resources, want only the observed configured child", got)
	}
}

func TestAccessAdmissionRejectsAmbiguousOrMismatchedReferencedStack(t *testing.T) {
	claim := accessClaims()["GrafanaTeamAccess"]
	tests := map[string]*fnv1.Resources{
		"multiple results": {
			Items: []*fnv1.Resource{
				requiredStackResource("teamdemo01", "grafana-vending", "True").GetItems()[0],
				requiredStackResource("teamdemo01", "grafana-vending", "True").GetItems()[0],
			},
		},
		"identity mismatch": {
			Items: []*fnv1.Resource{requiredStackResource("other-stack", "grafana-vending", "True").GetItems()[0]},
		},
	}
	for name, required := range tests {
		t.Run(name, func(t *testing.T) {
			rsp := callFunctionWithRequiredResources(t, claim, nil, required, requiredResourceCapabilities())
			if fatal := fatalResult(rsp); !strings.Contains(fatal, "referenced stack") {
				t.Fatalf("invalid referenced stack returned fatal result %q", fatal)
			}
		})
	}
}

func TestAccessAdmissionRejectsAdvertisedMissingRequiredResourceCapability(t *testing.T) {
	claim := accessClaims()["GrafanaTeamAccess"]
	rsp := callFunctionWithRequiredResources(t, claim, nil, nil, []fnv1.Capability{fnv1.Capability_CAPABILITY_CAPABILITIES})
	if fatal := fatalResult(rsp); !strings.Contains(fatal, "CAPABILITY_REQUIRED_RESOURCES") {
		t.Fatalf("missing required-resource capability returned fatal result %q", fatal)
	}
}

func TestSAMLSSOProfileRendersSAMLSettings(t *testing.T) {
	claim := stackDocument(map[string]any{"sso": map[string]any{"mode": "enforced", "profile": "example-saml"}})
	input := `{
		"apiVersion":"platform.example.org/v1beta1","kind":"GrafanaVendingConfig",
		"spec":{"ssoProfiles":[{
			"name":"example-saml","providerName":"saml",
			"samlSettings":{
				"name":"Example SAML","enabled":true,"allowSignUp":true,
				"idpMetadataUrl":"https://identity.example.com/saml/metadata",
				"assertionAttributeEmail":"email","assertionAttributeRole":"role",
				"roleValuesAdmin":"Admin","roleValuesEditor":"Editor","roleValuesViewer":"Viewer"
			}
		}]}
	}`
	rsp := runStackWithInput(t, claim, foundationReadyObserved(), input)
	params := nestedMap(t, desiredResource(t, rsp, "sso"), "spec", "forProvider")
	if _, ok := params["samlSettings"]; !ok {
		t.Fatal("SAML profile did not render samlSettings")
	}
	if _, ok := params["oauth2Settings"]; ok {
		t.Fatal("SAML profile rendered oauth2Settings")
	}
	if got := params["providerName"]; got != "saml" {
		t.Fatalf("SAML provider name = %v", got)
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
		"deletionArmed":       false,
		"deletionReady":       false,
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
		"metadata":   map[string]any{"name": "teamdemo01", "namespace": "grafana-vending", "uid": "example-request-uid"},
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

func runAccess(t *testing.T, composite string, observed map[string]*fnv1.Resource) *fnv1.RunFunctionResponse {
	t.Helper()
	rsp := callFunctionWithRequiredResources(t, composite, observed, requiredStackResource("teamdemo01", "grafana-vending", "True"), requiredResourceCapabilities())
	if fatal := fatalResult(rsp); fatal != "" {
		t.Fatalf("RunFunction returned a fatal result: %s", fatal)
	}
	return rsp
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

func callFunctionWithRequiredResources(t *testing.T, composite string, observed map[string]*fnv1.Resource, required *fnv1.Resources, capabilities []fnv1.Capability) *fnv1.RunFunctionResponse {
	t.Helper()
	req := &fnv1.RunFunctionRequest{
		Meta: &fnv1.RequestMeta{Capabilities: capabilities},
		Observed: &fnv1.State{
			Composite: &fnv1.Resource{Resource: resource.MustStructJSON(composite)},
			Resources: observed,
		},
	}
	if required != nil {
		req.RequiredResources = map[string]*fnv1.Resources{"referenced-stack": required}
	}
	rsp, err := (&Function{log: logging.NewNopLogger()}).RunFunction(context.Background(), req)
	if err != nil {
		t.Fatalf("RunFunction returned an error: %v", err)
	}
	return rsp
}

func requiredResourceCapabilities() []fnv1.Capability {
	return []fnv1.Capability{
		fnv1.Capability_CAPABILITY_CAPABILITIES,
		fnv1.Capability_CAPABILITY_REQUIRED_RESOURCES,
	}
}

func requiredStackResource(name, namespace, readyStatus string) *fnv1.Resources {
	return requiredStackResourceState(name, namespace, readyStatus, false, false)
}

func requiredStackResourceState(name, namespace, readyStatus string, deletionArmed, terminating bool) *fnv1.Resources {
	conditions := []any{}
	if readyStatus != "" {
		conditions = append(conditions, map[string]any{"type": "Ready", "status": readyStatus})
	}
	metadata := map[string]any{"name": name, "namespace": namespace}
	if terminating {
		metadata["deletionTimestamp"] = "2026-08-20T12:00:00Z"
	}
	return &fnv1.Resources{Items: []*fnv1.Resource{{Resource: resource.MustStructJSON(mustJSON(map[string]any{
		"apiVersion": "platform.example.org/v1beta1",
		"kind":       "GrafanaCloudStackRequest",
		"metadata":   metadata,
		"status":     map[string]any{"conditions": conditions, "deletionArmed": deletionArmed},
	}))}}}
}

func accessClaims() map[string]string {
	return map[string]string{
		"GrafanaCustomRoleBinding": `{
			"apiVersion":"platform.example.org/v1beta1",
			"kind":"GrafanaCustomRoleBinding",
			"metadata":{"name":"example-editor","namespace":"grafana-vending"},
			"spec":{
				"stackRef":{"name":"teamdemo01"},
				"team":{"name":"Example Editors","groups":["idp-example-editors"]},
				"role":{"name":"example-editor","uid":"example-editor","permissions":[{"action":"dashboards:read","scope":"folders:*"}]}
			}
		}`,
		"GrafanaTeamAccess": `{
			"apiVersion":"platform.example.org/v1beta1",
			"kind":"GrafanaTeamAccess",
			"metadata":{"name":"example-editors","namespace":"grafana-vending"},
			"spec":{
				"stackRef":{"name":"teamdemo01"},
				"team":{"name":"Example Editors"}
			}
		}`,
		"GrafanaContentAccessPolicy": `{
			"apiVersion":"platform.example.org/v1beta1",
			"kind":"GrafanaContentAccessPolicy",
			"metadata":{"name":"billing-access","namespace":"grafana-vending"},
			"spec":{
				"stackRef":{"name":"teamdemo01"},
				"target":{"kind":"Folder","ref":{"name":"teamdemo01-billing"}},
				"permissions":[{"basicRole":"Viewer","permission":"View"}]
			}
		}`,
	}
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

func desiredExternalName(t *testing.T, object map[string]any) string {
	t.Helper()
	annotations := nestedMap(t, object, "metadata", "annotations")
	value, _ := annotations["crossplane.io/external-name"].(string)
	return value
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

func TestFirstPassRendersOnlyTheStackFoundation(t *testing.T) {
	rsp := runStack(t, minimalStack(), nil)

	got := make([]string, 0, len(rsp.GetDesired().GetResources()))
	for name := range rsp.GetDesired().GetResources() {
		got = append(got, name)
	}
	sort.Strings(got)

	want := []string{
		"credentials",
		"instance-credentials",
		"provider-config",
		"stack",
		"stack-service-account",
		"telemetry-access-policy",
		"telemetry-credentials",
	}
	sort.Strings(want)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("first-pass desired resources differ (-want +got):\n%s", diff)
	}
}

func TestStackLocalResourcesAppearOnceTheStackServesAndCredentialsArePublished(t *testing.T) {
	rsp := runStack(t, minimalStack(), foundationReadyObserved())

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
		"stack-token=StackServiceAccountRotatingToken",
		"telemetry-access-policy=AccessPolicy",
		"telemetry-credentials=PushSecret",
	}
	sort.Strings(want)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("staged desired resources differ (-want +got):\n%s", diff)
	}
}

func TestStackLocalResourcesAreNeverWithdrawnOnceObserved(t *testing.T) {
	// A desired resource that disappears is deleted, so an already-created
	// folder must survive the stack reporting not-Ready.
	observed := map[string]*fnv1.Resource{
		"billing-folder": observedResource(`{
			"apiVersion":"oss.grafana.m.crossplane.io/v1alpha1",
			"kind":"Folder",
			"metadata":{"name":"teamdemo01-billing","namespace":"grafana-vending"}
		}`),
	}
	rsp := runStack(t, minimalStack(), observed)

	resources := rsp.GetDesired().GetResources()
	if _, ok := resources["billing-folder"]; !ok {
		t.Fatal("an already-observed folder was withdrawn from the desired set")
	}
	if _, ok := resources["endpoints-folder"]; ok {
		t.Fatal("an unobserved folder was rendered before the stack was serving")
	}
}

func TestPluginInstallationWaitsOnlyForTheStackToServe(t *testing.T) {
	observed := map[string]*fnv1.Resource{
		"stack": observedResource(`{
			"apiVersion":"cloud.grafana.m.crossplane.io/v1alpha1",
			"kind":"Stack",
			"metadata":{"name":"teamdemo01","namespace":"grafana-vending"},
			"status":{"conditions":[{"type":"Ready","status":"True","reason":"Available"}]}
		}`),
	}
	claim := stackDocument(map[string]any{"plugins": []any{
		map[string]any{"slug": "grafana-clock-panel", "version": "2.1.3"},
	}})
	rsp := runStack(t, claim, observed)

	resources := rsp.GetDesired().GetResources()
	if _, ok := resources["plugin-grafana-clock-panel"]; !ok {
		t.Fatal("plugin installation was withheld after the stack began serving")
	}
	if _, ok := resources["billing-folder"]; ok {
		t.Fatal("baseline content was rendered before the per-stack credential was published")
	}
}

func foundationReadyObserved() map[string]*fnv1.Resource {
	return map[string]*fnv1.Resource{
		"stack": observedResource(`{
			"apiVersion":"cloud.grafana.m.crossplane.io/v1alpha1",
			"kind":"Stack",
			"metadata":{"name":"teamdemo01","namespace":"grafana-vending"},
			"status":{
				"atProvider":{"id":"12345","slug":"teamdemo01","url":"https://teamdemo01.grafana.net"},
				"conditions":[{"type":"Ready","status":"True","reason":"Available"}]
			}
		}`),
		"stack-service-account": observedResource(`{
			"apiVersion":"cloud.grafana.m.crossplane.io/v1alpha1",
			"kind":"StackServiceAccount",
			"metadata":{"name":"teamdemo01-admin","namespace":"grafana-vending"},
			"status":{
				"atProvider":{"id":"67890"},
				"conditions":[{"type":"Ready","status":"True","reason":"Available"}]
			}
		}`),
		"instance-credentials": observedResource(`{
			"apiVersion":"external-secrets.io/v1",
			"kind":"ExternalSecret",
			"metadata":{"name":"teamdemo01-provider-credentials","namespace":"grafana-vending"},
			"status":{"conditions":[{"type":"Ready","status":"True","reason":"SecretSynced"}]}
		}`),
	}
}
