package main

import (
	"testing"

	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
	"github.com/google/go-cmp/cmp"
)

func TestK6ProjectRendersBootstrapProjectAndPlatformLimits(t *testing.T) {
	observed := map[resource.Name]resource.ObservedComposed{
		"installation":         observedComposed(`{"status":{"conditions":[{"type":"Ready","status":"True"}]}}`),
		"provider-credentials": observedComposed(`{"status":{"conditions":[{"type":"Ready","status":"True"}]}}`),
		"project":              observedComposed(`{"status":{"atProvider":{"id":"42"}}}`),
	}
	desired, err := renderK6Project(k6ProjectDocument(map[string]any{
		"allowedLoadZones": []any{"private-zone-example-a", "private-zone-example-b"},
		"vuhMaxPerMonth":   float64(999999),
		"vuMaxPerTest":     float64(999999),
	}), observed, k6Config())
	if err != nil {
		t.Fatalf("renderK6Project returned an error: %v", err)
	}

	for name, kind := range map[resource.Name]string{
		"installation":         "Installation",
		"provider-credentials": "ExternalSecret",
		"provider-config":      "ProviderConfig",
		"project":              "Project",
		"limits":               "ProjectLimits",
		"allowed-load-zones":   "ProjectAllowedLoadZones",
		"credentials":          "PushSecret",
	} {
		object := k6Desired(t, desired, name)
		if got := object["kind"]; got != kind {
			t.Errorf("%s kind = %v, want %s", name, got, kind)
		}
	}
	if got, want := len(desired), 7; got != want {
		t.Fatalf("rendered %d resources, want exactly %d without load tests or schedules", got, want)
	}

	installation := k6Desired(t, desired, "installation")
	installationParameters := nestedMap(t, installation, "spec", "forProvider")
	if got, want := installationParameters["grafanaSaTokenSecretRef"], map[string]any{"name": "teamdemo01-token", "key": "attribute.key"}; !cmp.Equal(got, want) {
		t.Fatalf("installation bootstrap secret differs (-want +got):\n%s", cmp.Diff(want, got))
	}
	if got := installationParameters["cloudAccessPolicyTokenSecretRef"]; got != nil {
		t.Fatalf("installation rendered deprecated cloud token field: %v", got)
	}
	if got, want := installationParameters["stackId"], "101"; got != want {
		t.Fatalf("installation stack ID = %v, want %s", got, want)
	}
	if got, want := installationParameters["grafanaUser"], "operator@example.invalid"; got != want {
		t.Fatalf("installation user = %v, want %s", got, want)
	}
	if got, want := nestedMap(t, installation, "spec", "providerConfigRef"), map[string]any{"kind": "ProviderConfig", "name": "grafana-cloud-org-example-primary"}; !cmp.Equal(got, want) {
		t.Fatalf("installation provider configuration differs (-want +got):\n%s", cmp.Diff(want, got))
	}

	providerName := "example-k6-project-k6"
	providerCredentials := k6Desired(t, desired, "provider-credentials")
	if got, want := nestedMap(t, providerCredentials, "spec", "secretStoreRef"), map[string]any{"name": "grafana-vending-secrets", "kind": "SecretStore"}; !cmp.Equal(got, want) {
		t.Fatalf("k6 credential store reference differs (-want +got):\n%s", cmp.Diff(want, got))
	}
	providerTarget := nestedMap(t, providerCredentials, "spec", "target")
	if got, want := providerTarget["name"], "example-k6-project-k6-provider-credentials"; got != want {
		t.Fatalf("k6 credential Secret name = %v, want %s", got, want)
	}
	providerData := nestedMap(t, providerTarget, "template", "data")
	if got, want := providerData["credentials"], `{"k6_access_token":{{ .k6AccessToken | toJson }}}`; got != want {
		t.Fatalf("k6 provider credential document = %v, want %s", got, want)
	}
	externalData := nestedMap(t, providerCredentials, "spec")["data"].([]any)
	remote := nestedMap(t, externalData[0].(map[string]any), "remoteRef")
	if got, want := remote["key"], "/platform/grafana-cloud/stacks/example-primary/development/teamdemo01/k6"; got != want {
		t.Fatalf("k6 credential remote key = %v, want %s", got, want)
	}
	if got, want := remote["property"], "k6_access_token"; got != want {
		t.Fatalf("k6 credential remote property = %v, want %s", got, want)
	}

	providerConfig := k6Desired(t, desired, "provider-config")
	secretRef := nestedMap(t, providerConfig, "spec", "credentials", "secretRef")
	wantSecretRef := map[string]any{"name": "example-k6-project-k6-provider-credentials", "namespace": "grafana-vending", "key": "credentials"}
	if diff := cmp.Diff(wantSecretRef, secretRef); diff != "" {
		t.Fatalf("k6 ProviderConfig secret reference differs (-want +got):\n%s", diff)
	}

	project := k6Desired(t, desired, "project")
	metadata := nestedMap(t, project, "metadata")
	if annotations, ok := metadata["annotations"].(map[string]any); ok {
		if externalName := annotations["crossplane.io/external-name"]; externalName != nil {
			t.Fatalf("project guessed provider-assigned external name: %v", externalName)
		}
	}
	for _, childName := range []resource.Name{"project", "limits", "allowed-load-zones"} {
		provider := nestedMap(t, k6Desired(t, desired, childName), "spec", "providerConfigRef")
		if got, want := provider["name"], providerName; got != want {
			t.Fatalf("%s ProviderConfig = %v, want derived-token ProviderConfig %s", childName, got, want)
		}
		if got := provider["name"]; got == "grafana-cloud-org-example-primary" {
			t.Fatalf("%s reused organization ProviderConfig", childName)
		}
	}
	for _, childName := range []resource.Name{"limits", "allowed-load-zones"} {
		annotations := nestedMap(t, k6Desired(t, desired, childName), "metadata", "annotations")
		if got, want := annotations["crossplane.io/external-name"], "42"; got != want {
			t.Fatalf("%s external name = %v, want observed project ID %s", childName, got, want)
		}
	}

	limits := nestedMap(t, k6Desired(t, desired, "limits"), "spec", "forProvider")
	wantLimits := map[string]any{
		"projectId":           "42",
		"vuhMaxPerMonth":      float64(100),
		"vuMaxPerTest":        float64(10),
		"vuBrowserMaxPerTest": float64(2),
		"durationMaxPerTest":  float64(600),
	}
	if diff := cmp.Diff(wantLimits, limits); diff != "" {
		t.Fatalf("platform-controlled limits differ (-want +got):\n%s", diff)
	}

	zones := nestedMap(t, k6Desired(t, desired, "allowed-load-zones"), "spec", "forProvider")
	if got, want := zones["projectId"], "42"; got != want {
		t.Fatalf("allowed-zone project ID = %v, want %s", got, want)
	}
	if got, want := zones["allowedLoadZones"], []any{"private-zone-example-a", "private-zone-example-b"}; !cmp.Equal(got, want) {
		t.Fatalf("allowed zones differ (-want +got):\n%s", cmp.Diff(want, got))
	}

	push := nestedMap(t, k6Desired(t, desired, "credentials"), "spec")
	if got := nestedMap(t, push, "selector", "secret")["name"]; got != "example-k6-project-k6-token" {
		t.Fatalf("credential selector = %v, want derived k6 token secret", got)
	}
	if got := nestedMap(t, push, "template", "data")["k6.json"]; got == nil {
		t.Fatal("derived k6 credential is not mirrored to the secret store")
	}
	if got, want := nestedMap(t, push, "template", "data")["k6.json"], `{{ $token := index . "attribute.k6_access_token" | toString }}{"k6_access_token":{{ $token | toJson }}}`; got != want {
		t.Fatalf("derived k6 credential mapping = %v, want %s", got, want)
	}
}

func TestK6ProjectWaitsForDerivedCredentialBeforeUsingK6Provider(t *testing.T) {
	observed := map[resource.Name]resource.ObservedComposed{
		"installation": observedComposed(`{"status":{"conditions":[{"type":"Ready","status":"True"}]}}`),
	}
	desired, err := renderK6Project(k6ProjectDocument(nil), observed, k6Config())
	if err != nil {
		t.Fatalf("renderK6Project returned an error: %v", err)
	}
	for _, name := range []resource.Name{"installation", "credentials", "provider-credentials", "provider-config"} {
		k6Desired(t, desired, name)
	}
	for _, name := range []resource.Name{"project", "limits", "allowed-load-zones"} {
		if _, ok := desired[name]; ok {
			t.Fatalf("%s rendered before the derived k6 credential was materialized", name)
		}
	}
}

func TestK6ProjectWaitsForReferencedBootstrapContext(t *testing.T) {
	desired, err := renderK6Project(k6ProjectDocument(nil), nil, map[string]any{"spec": k6Config()["spec"]})
	if err != nil {
		t.Fatalf("renderK6Project returned an error before required resources resolved: %v", err)
	}
	if len(desired) != 0 {
		t.Fatalf("renderK6Project emitted %d resources before trusted stack context", len(desired))
	}
}

func TestK6ProjectWaitsForObservedServiceAccountToken(t *testing.T) {
	config := k6Config()
	stack := config[resolvedK6StackConfigKey].(map[string]any)
	stack["serviceAccountTokenSecret"].(map[string]any)["ready"] = false
	desired, err := renderK6Project(k6ProjectDocument(nil), nil, config)
	if err != nil {
		t.Fatalf("renderK6Project returned an error before the stack token was observed: %v", err)
	}
	if len(desired) != 0 {
		t.Fatalf("renderK6Project emitted %d resources before the stack token was observed", len(desired))
	}
}

func TestK6ProjectUsesObservedStackUsageAndNotARequestOverride(t *testing.T) {
	claim := k6ProjectDocument(map[string]any{"usage": "production"})
	desired, err := renderK6Project(claim, map[resource.Name]resource.ObservedComposed{
		"installation":         observedComposed(`{"status":{"conditions":[{"type":"Ready","status":"True"}]}}`),
		"provider-credentials": observedComposed(`{"status":{"conditions":[{"type":"Ready","status":"True"}]}}`),
		"project":              observedComposed(`{"status":{"atProvider":{"id":"42"}}}`),
	}, k6Config())
	if err != nil {
		t.Fatalf("renderK6Project returned an error: %v", err)
	}
	limits := nestedMap(t, k6Desired(t, desired, "limits"), "spec", "forProvider")
	if got, want := limits["vuhMaxPerMonth"], float64(100); got != want {
		t.Fatalf("request changed the platform-selected usage cap: got %v, want %v", got, want)
	}
}

func TestK6ProjectWaitsForProviderAssignedProjectIDBeforeChildren(t *testing.T) {
	observed := map[resource.Name]resource.ObservedComposed{
		"installation":         observedComposed(`{"status":{"conditions":[{"type":"Ready","status":"True"}]}}`),
		"provider-credentials": observedComposed(`{"status":{"conditions":[{"type":"Ready","status":"True"}]}}`),
	}
	desired, err := renderK6Project(k6ProjectDocument(nil), observed, k6Config())
	if err != nil {
		t.Fatalf("renderK6Project returned an error: %v", err)
	}
	k6Desired(t, desired, "project")
	for _, name := range []resource.Name{"limits", "allowed-load-zones"} {
		if _, ok := desired[name]; ok {
			t.Fatalf("%s rendered before the provider-assigned Project ID was observed", name)
		}
	}
}

func TestK6ProjectRejectsAZoneOutsideItsPlatformProfile(t *testing.T) {
	_, err := renderK6Project(k6ProjectDocument(map[string]any{
		"allowedLoadZones": []any{"unapproved-private-zone"},
	}), nil, k6Config())
	if err == nil {
		t.Fatal("out-of-profile load zone was accepted")
	}
}

func k6ProjectDocument(additions map[string]any) map[string]any {
	spec := map[string]any{"allowedLoadZones": []any{},
		"stackRef":    map[string]any{"name": "teamdemo01"},
		"grafanaUser": "operator@example.invalid",
	}
	for key, value := range additions {
		spec[key] = value
	}
	return map[string]any{
		"apiVersion": "platform.example.org/v1beta1",
		"kind":       "GrafanaK6Project",
		"metadata":   map[string]any{"name": "example-k6-project", "namespace": "grafana-vending"},
		"spec":       spec,
	}
}

func k6Config() map[string]any {
	return map[string]any{
		"spec": map[string]any{"k6LimitProfiles": []any{map[string]any{
			"usage": "development", "vuhMaxPerMonth": float64(100), "vuMaxPerTest": float64(10),
			"vuBrowserMaxPerTest": float64(2), "durationMaxPerTest": float64(600),
			"allowedLoadZones": []any{"private-zone-example-a", "private-zone-example-b"},
		}}},
		"_resolvedK6Stack": map[string]any{
			"usage":                          "development",
			"stackId":                        "101",
			"outputSecretPath":               "/platform/grafana-cloud/stacks/example-primary/development/teamdemo01",
			"organizationProviderConfigName": "grafana-cloud-org-example-primary",
			"serviceAccountTokenSecret": map[string]any{
				"name": "teamdemo01-token", "key": "attribute.key", "ready": true,
			},
		},
	}
}

func k6Desired(t *testing.T, desired map[resource.Name]*resource.DesiredComposed, name resource.Name) map[string]any {
	t.Helper()
	resource, ok := desired[name]
	if !ok {
		t.Fatalf("desired resource %q was not rendered", name)
	}
	return resource.Resource.UnstructuredContent()
}

func observedComposed(document string) resource.ObservedComposed {
	observed := composed.New()
	observed.SetUnstructuredContent(resource.MustStructJSON(document).AsMap())
	return resource.ObservedComposed{Resource: observed}
}

func TestK6LoadZoneIntentMustBeExplicit(t *testing.T) {
	if _, err := requestedK6LoadZones(map[string]any{}, nil); err == nil {
		t.Fatal("omission silently cleared the private-zone allow-list")
	}
	zones, err := requestedK6LoadZones(map[string]any{"allowedLoadZones": []any{}}, nil)
	if err != nil || len(zones) != 0 {
		t.Fatalf("explicit empty set: %v %v", zones, err)
	}
}
