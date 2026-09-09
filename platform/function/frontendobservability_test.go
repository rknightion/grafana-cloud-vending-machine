package main

import (
	"context"
	"os"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestFrontendObservabilityRendersPlatformOwnedAppsAndPinsBrowserCredentialBoundary(t *testing.T) {
	desired, err := renderFrontendObservability(frontendObservabilityClaim(), nil, frontendObservabilityConfig())
	if err != nil {
		t.Fatalf("render frontend observability: %v", err)
	}
	if got, want := len(desired), 1; got != want {
		t.Fatalf("desired resources = %d, want %d", got, want)
	}
	app := desired["app-browser"].Resource.UnstructuredContent()
	if got, want := app["apiVersion"], frontendObservabilityAPIVersion; got != want {
		t.Fatalf("app apiVersion = %q, want %q", got, want)
	}
	if got, want := app["kind"], "App"; got != want {
		t.Fatalf("app kind = %q, want %q", got, want)
	}
	if got := desiredExternalName(t, desired["app-browser"].Resource.UnstructuredContent()); got != "12345:browser" {
		t.Fatalf("app external name = %q, want deterministic stack ID and app name", got)
	}
	parameters := nestedMap(t, app, "spec", "forProvider")
	for _, field := range []string{"allowedOrigins", "extraLogAttributes", "name", "settings", "stackId"} {
		if _, ok := parameters[field]; !ok {
			t.Errorf("App forProvider omitted provider-required field %q", field)
		}
	}
	if got := parameters["stackId"]; got != float64(12345) {
		t.Errorf("app stackId = %#v, want observed stack ID", got)
	}
	providerConfigRef := nestedMap(t, app, "spec", "providerConfigRef")
	if got, want := providerConfigRef["name"], "grafana-cloud-org-example-primary"; got != want {
		t.Fatalf("App providerConfigRef.name = %q, want organization ProviderConfig %q", got, want)
	}
	if got := providerConfigRef["name"]; got == "teamdemo01" {
		t.Fatal("App providerConfigRef used the stack ProviderConfig instead of the organization ProviderConfig")
	}
	if _, found := nestedMap(t, app, "spec")["writeConnectionSecretToRef"]; found {
		t.Fatal("Faro app rendered a Secret publication for a browser-visible collector endpoint")
	}
	if _, found := app["status"]; found {
		t.Fatal("Faro renderer attempted to publish status material")
	}
	encoded := mustJSON(app)
	for _, forbidden := range []string{"appKey", "secret", "credential", "token"} {
		if strings.Contains(strings.ToLower(encoded), strings.ToLower(forbidden)) {
			t.Fatalf("Faro rendered browser-visible app material or provider credentials (%q): %s", forbidden, encoded)
		}
	}

	readmeBytes, err := os.ReadFile("../../examples/catalog/frontend-observability/README.md")
	if err != nil {
		t.Fatalf("read credential-boundary README: %v", err)
	}
	readme := string(readmeBytes)
	for _, sentence := range []string{
		"status.atProvider.collectorEndpoint",
		"contains the Faro app key",
		"intended to be visible in a browser bundle",
		"not accepted from this request, stored in a Kubernetes Secret, or granted privileged Grafana access",
	} {
		if !strings.Contains(readme, sentence) {
			t.Errorf("credential-boundary README omitted %q", sentence)
		}
	}
}

func TestFrontendObservabilityAdmissionRejectsRequestSuppliedAppKeyAndAllowsProfileUpdate(t *testing.T) {
	env := &admissionEnv{paths: []string{"../apis/frontend-observability-v1beta1.yaml"}}
	if err := env.Start(t); err != nil {
		t.Fatalf("start isolated frontend observability admission environment: %v", err)
	}
	t.Cleanup(func() { _ = env.Stop() })
	ctx := context.Background()
	allowed := frontendObservabilityRequest("faroallowed", "standard", "")
	if err := env.Apply(ctx, allowed); err != nil {
		t.Fatalf("allowed profile control was refused: %v", err)
	}
	allowed.Object["spec"].(map[string]any)["profile"] = "regulated"
	if err := env.Apply(ctx, allowed); err != nil {
		t.Fatalf("profile update control was refused: %v", err)
	}

	for _, name := range []string{"farokeycreate", "farokeyupdate"} {
		forbidden := frontendObservabilityRequest(name, "standard", "browser-visible-input")
		if name == "farokeyupdate" {
			forbidden = frontendObservabilityRequest(name, "standard", "")
			if err := env.Apply(ctx, forbidden); err != nil {
				t.Fatalf("update baseline was refused: %v", err)
			}
			forbidden.Object["spec"].(map[string]any)["appKey"] = "browser-visible-input"
		}
		err := env.Apply(ctx, forbidden)
		if err == nil || !strings.Contains(err.Error(), "Faro app keys are browser-visible collector endpoint material and must not be supplied in this request.") {
			t.Fatalf("%s refusal = %v, want its own app-key message", name, err)
		}
		t.Logf("app-key refusal %s: %v", name, err)
	}
}

func frontendObservabilityClaim() map[string]any {
	return map[string]any{
		"metadata": map[string]any{"name": "teamdemo01", "namespace": "grafana-vending"},
		"spec":     map[string]any{"stackRef": map[string]any{"name": "teamdemo01"}, "profile": "standard"},
	}
}

func frontendObservabilityConfig() map[string]any {
	return map[string]any{
		"referencedStack": map[string]any{
			"stackID": "12345", "organizationProviderConfigName": "grafana-cloud-org-example-primary",
		},
		"spec": map[string]any{
			"frontendObservabilityProfiles": []any{
				map[string]any{
					"name": "standard",
					"apps": []any{map[string]any{
						"name": "browser", "allowedOrigins": []any{"https://example.invalid"},
						"extraLogAttributes": map[string]any{"environment": "example"}, "settings": map[string]any{"combineLabData": "0"},
					}},
				},
			},
		},
	}
}

func TestFrontendObservabilityWaitsForOrganizationProviderConfig(t *testing.T) {
	config := frontendObservabilityConfig()
	config["referencedStack"].(map[string]any)["organizationProviderConfigName"] = ""

	_, err := renderFrontendObservability(frontendObservabilityClaim(), nil, config)
	if err == nil || !strings.Contains(err.Error(), "trusted referenced stack context is incomplete") {
		t.Fatalf("render with incomplete trusted context = %v, want trusted-context refusal", err)
	}
}

func frontendObservabilityRequest(name, profile, appKey string) *unstructured.Unstructured {
	spec := map[string]any{"stackRef": map[string]any{"name": name}, "profile": profile}
	if appKey != "" {
		spec["appKey"] = appKey
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "platform.example.org/v1beta1", "kind": "GrafanaFrontendObservability",
		"metadata": map[string]any{"name": name, "namespace": "default"}, "spec": spec,
	}}
}
