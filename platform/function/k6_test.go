package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
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

func TestK6ProjectRendersLoadTestsAndSchedulesAfterPlatformCaps(t *testing.T) {
	observed := k6BootstrapObserved()
	claim := k6ProjectDocument(map[string]any{
		"usage":            "development",
		"allowedLoadZones": []any{"private-zone-example-a", "private-zone-example-b"},
		"loadTests": []any{map[string]any{
			"name": "smoke", "workload": k6HTTPWorkload("https://service.example.invalid/health"), "vus": float64(2),
			"browserVus": float64(0), "durationSeconds": float64(60), "loadZones": []any{"private-zone-example-b", "private-zone-example-a"},
		}},
		"schedules": []any{map[string]any{
			"name": "nightly", "loadTest": "smoke", "starts": "2030-01-01T00:00:00Z",
			"recurrenceRule": map[string]any{"frequency": "DAILY", "interval": float64(1), "count": float64(7)},
		}},
	})
	firstDesired, err := renderK6Project(claim, observed, k6Config())
	if err != nil {
		t.Fatalf("first renderK6Project returned an error: %v", err)
	}
	if got, want := len(firstDesired), 7; got != want {
		t.Fatalf("first render produced %d resources, want %d before dynamic children", got, want)
	}
	for name, value := range k6CurrentPlatformCapObservations(t, firstDesired) {
		observed[name] = value
	}
	observed["load-test-smoke"] = observedComposed(`{"status":{"atProvider":{"id":"501"}}}`)
	observed["schedule-nightly"] = observedComposed(`{"status":{"atProvider":{"id":"601"}}}`)
	desired, err := renderK6Project(claim, observed, k6Config())
	if err != nil {
		t.Fatalf("second renderK6Project returned an error: %v", err)
	}
	if got, want := len(desired), 9; got != want {
		t.Fatalf("rendered %d resources, want %d including one load test and one schedule", got, want)
	}
	loadTest := k6Desired(t, desired, "load-test-smoke")
	if got, want := loadTest["kind"], "LoadTest"; got != want {
		t.Fatalf("load test kind = %v, want %s", got, want)
	}
	if got, want := nestedMap(t, loadTest, "metadata")["name"], "example-k6-project-load-test-smoke"; got != want {
		t.Fatalf("load test metadata.name = %v, want %s", got, want)
	}
	if got, want := nestedMap(t, loadTest, "metadata", "annotations")["crossplane.io/external-name"], "501"; got != want {
		t.Fatalf("load test external name = %v, want observed ID %s", got, want)
	}
	if got, want := nestedMap(t, loadTest, "spec")["managementPolicies"], k6DynamicManagementPolicies; !cmp.Equal(got, want) {
		t.Fatalf("load test management policies differ (-want +got):\n%s", cmp.Diff(want, got))
	}
	loadTestParameters := nestedMap(t, loadTest, "spec", "forProvider")
	wantLoadTestParameters := map[string]any{
		"projectId": "42", "name": "smoke",
		"script": "import http from \"k6/http\";\n\nexport const options = {\n  scenarios: {\n    default: {\n      executor: \"constant-vus\",\n      vus: 2,\n      duration: \"60s\",\n      gracefulStop: \"0s\",\n    },\n  },\n  cloud: {\n    distribution: {\n      \"private-zone-example-a\": { loadZone: \"private-zone-example-a\", percent: 50 },\n      \"private-zone-example-b\": { loadZone: \"private-zone-example-b\", percent: 50 },\n    },\n  },\n};\n\nexport default function () {\n  http.get(\"https://service.example.invalid/health\");\n}\n",
	}
	if diff := cmp.Diff(wantLoadTestParameters, loadTestParameters); diff != "" {
		t.Fatalf("load test parameters differ (-want +got):\n%s", diff)
	}

	schedule := k6Desired(t, desired, "schedule-nightly")
	if got, want := schedule["kind"], "Schedule"; got != want {
		t.Fatalf("schedule kind = %v, want %s", got, want)
	}
	if got, want := nestedMap(t, schedule, "metadata")["name"], "example-k6-project-schedule-nightly"; got != want {
		t.Fatalf("schedule metadata.name = %v, want %s", got, want)
	}
	if got, want := nestedMap(t, schedule, "metadata", "annotations")["crossplane.io/external-name"], "601"; got != want {
		t.Fatalf("schedule external name = %v, want observed ID %s", got, want)
	}
	if got, want := nestedMap(t, schedule, "spec")["managementPolicies"], k6DynamicManagementPolicies; !cmp.Equal(got, want) {
		t.Fatalf("schedule management policies differ (-want +got):\n%s", cmp.Diff(want, got))
	}
	wantScheduleParameters := map[string]any{
		"loadTestId": "501", "starts": "2030-01-01T00:00:00Z",
		"recurrenceRule": map[string]any{"frequency": "DAILY", "interval": float64(1), "count": float64(7)},
	}
	if diff := cmp.Diff(wantScheduleParameters, nestedMap(t, schedule, "spec", "forProvider")); diff != "" {
		t.Fatalf("schedule parameters differ (-want +got):\n%s", diff)
	}
}

func TestK6ProjectWaitsForObservedPlatformCapsBeforeDynamicChildren(t *testing.T) {
	claim := k6ProjectDocument(map[string]any{
		"usage": "development",
		"loadTests": []any{map[string]any{
			"name": "smoke", "workload": k6HTTPWorkload("https://service.example.invalid/health"), "vus": float64(1),
			"durationSeconds": float64(60), "loadZones": []any{},
		}},
		"schedules": []any{},
	})
	observed := map[resource.Name]resource.ObservedComposed{
		"installation":         observedComposed(`{"status":{"conditions":[{"type":"Ready","status":"True"}]}}`),
		"provider-credentials": observedComposed(`{"status":{"conditions":[{"type":"Ready","status":"True"}]}}`),
		"project":              observedComposed(`{"status":{"atProvider":{"id":"42"}}}`),
	}
	desired, err := renderK6Project(claim, observed, k6Config())
	if err != nil {
		t.Fatalf("renderK6Project returned an error: %v", err)
	}
	if _, ok := desired["load-test-smoke"]; ok {
		t.Fatal("load test rendered before ProjectLimits and ProjectAllowedLoadZones were Ready")
	}
}

func TestK6ProjectRequiresCurrentMatchingPlatformCapsBeforeDynamicChildren(t *testing.T) {
	claim := k6ProjectDocument(map[string]any{
		"usage": "development",
		"loadTests": []any{map[string]any{
			"name": "smoke", "workload": k6HTTPWorkload("https://service.example.invalid/health"), "vus": float64(1),
			"durationSeconds": float64(60), "loadZones": []any{},
		}},
	})
	firstObserved := k6BootstrapObserved()
	firstDesired, err := renderK6Project(claim, firstObserved, k6Config())
	if err != nil {
		t.Fatalf("first renderK6Project returned an error: %v", err)
	}

	t.Run("current caps admit dynamic children", func(t *testing.T) {
		observed := k6BootstrapObserved()
		for name, value := range k6CurrentPlatformCapObservations(t, firstDesired) {
			observed[name] = value
		}
		desired, renderErr := renderK6Project(claim, observed, k6Config())
		if renderErr != nil {
			t.Fatalf("renderK6Project returned an error: %v", renderErr)
		}
		k6Desired(t, desired, "load-test-smoke")
	})

	t.Run("stale generation with Ready condition waits", func(t *testing.T) {
		observed := k6BootstrapObserved()
		caps := k6CurrentPlatformCapObservations(t, firstDesired)
		stale := caps["limits"].Resource
		metadata := stale.Object["metadata"].(map[string]any)
		metadata["generation"] = int64(2)
		caps["limits"] = resource.ObservedComposed{Resource: stale}
		for name, value := range caps {
			observed[name] = value
		}
		desired, renderErr := renderK6Project(claim, observed, k6Config())
		if renderErr != nil {
			t.Fatalf("renderK6Project returned an error: %v", renderErr)
		}
		if _, ok := desired["load-test-smoke"]; ok {
			t.Fatal("load test rendered from an observation with a stale generation")
		}
	})

	t.Run("spec mismatch with Ready condition waits", func(t *testing.T) {
		observed := k6BootstrapObserved()
		caps := k6CurrentPlatformCapObservations(t, firstDesired)
		mismatched := caps["limits"].Resource
		spec := mismatched.Object["spec"].(map[string]any)
		forProvider := spec["forProvider"].(map[string]any)
		forProvider["vuMaxPerTest"] = float64(9)
		for name, value := range caps {
			observed[name] = value
		}
		desired, renderErr := renderK6Project(claim, observed, k6Config())
		if renderErr != nil {
			t.Fatalf("renderK6Project returned an error: %v", renderErr)
		}
		if _, ok := desired["load-test-smoke"]; ok {
			t.Fatal("load test rendered from an observation whose spec does not match desired")
		}
	})
}

func TestK6ProjectWaitsForObservedLoadTestIDBeforeSchedule(t *testing.T) {
	claim := k6ProjectDocument(map[string]any{
		"usage": "development",
		"loadTests": []any{map[string]any{
			"name": "smoke", "workload": k6HTTPWorkload("https://service.example.invalid/health"), "vus": float64(1),
			"durationSeconds": float64(60), "loadZones": []any{},
		}},
		"schedules": []any{map[string]any{
			"name": "nightly", "loadTest": "smoke", "starts": "2030-01-01T00:00:00Z",
		}},
	})
	observed := k6BootstrapObserved()
	firstDesired, err := renderK6Project(claim, observed, k6Config())
	if err != nil {
		t.Fatalf("first renderK6Project returned an error: %v", err)
	}
	for name, value := range k6CurrentPlatformCapObservations(t, firstDesired) {
		observed[name] = value
	}
	desired, err := renderK6Project(claim, observed, k6Config())
	if err != nil {
		t.Fatalf("second renderK6Project returned an error: %v", err)
	}
	k6Desired(t, desired, "load-test-smoke")
	if _, ok := desired["schedule-nightly"]; ok {
		t.Fatal("schedule rendered before provider-assigned LoadTest ID was observed")
	}
}

func TestK6ProjectRejectsDynamicChildrenOutsidePlatformCaps(t *testing.T) {
	cases := []struct {
		name       string
		addition   map[string]any
		expectText string
	}{
		{
			name: "regular VUs", addition: map[string]any{"vus": float64(11)},
			expectText: "load test \"over-cap\" requests 11 VUs; platform maximum is 10",
		},
		{
			name: "browser VUs", addition: map[string]any{"browserVus": float64(3)},
			expectText: "loadTests[0].browserVus is unsupported; generated browser workloads are not available",
		},
		{
			name: "duration", addition: map[string]any{"durationSeconds": float64(601)},
			expectText: "load test \"over-cap\" requests 601 seconds; platform maximum is 600",
		},
		{
			name: "load zone", addition: map[string]any{"loadZones": []any{"private-zone-example-b"}},
			expectText: "load test \"over-cap\" uses load zone \"private-zone-example-b\" outside the project's allowed load zones",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			loadTest := map[string]any{
				"name": "over-cap", "workload": k6HTTPWorkload("https://service.example.invalid/health"), "vus": float64(1),
				"durationSeconds": float64(60), "loadZones": []any{},
			}
			for key, value := range tc.addition {
				loadTest[key] = value
			}
			_, err := renderK6Project(k6ProjectDocument(map[string]any{
				"usage": "development", "loadTests": []any{loadTest},
			}), nil, k6Config())
			if err == nil || !strings.Contains(err.Error(), tc.expectText) {
				t.Fatalf("dynamic cap error = %v, want %q", err, tc.expectText)
			}
		})
	}
}

func TestK6ProjectRequiresBoundedStructuredHTTPWorkloads(t *testing.T) {
	cases := []struct {
		name       string
		mutate     func(map[string]any)
		expectText string
	}{
		{
			name: "raw script", mutate: func(loadTest map[string]any) {
				loadTest["script"] = "export default function () {}"
			}, expectText: "loadTests[0].script is unsupported; use workload.httpGet.url",
		},
		{
			name: "missing workload", mutate: func(loadTest map[string]any) {
				delete(loadTest, "workload")
			}, expectText: "loadTests[0].workload must be supplied",
		},
		{
			name: "non HTTPS URL", mutate: func(loadTest map[string]any) {
				loadTest["workload"] = k6HTTPWorkload("http://service.example.invalid/health")
			}, expectText: "loadTests[0].workload.httpGet.url must be a non-empty HTTPS URL without credentials",
		},
		{
			name: "URL credentials", mutate: func(loadTest map[string]any) {
				loadTest["workload"] = k6HTTPWorkload("https://user:password@service.example.invalid/health")
			}, expectText: "loadTests[0].workload.httpGet.url must be a non-empty HTTPS URL without credentials",
		},
		{
			name: "fractional VUs", mutate: func(loadTest map[string]any) {
				loadTest["vus"] = float64(1.5)
			}, expectText: "loadTests[0].vus must be an integer",
		},
		{
			name: "fractional duration", mutate: func(loadTest map[string]any) {
				loadTest["durationSeconds"] = float64(60.5)
			}, expectText: "loadTests[0].durationSeconds must be an integer",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			loadTest := map[string]any{
				"name": "bounded", "workload": k6HTTPWorkload("https://service.example.invalid/health"), "vus": float64(1),
				"durationSeconds": float64(60), "loadZones": []any{},
			}
			tc.mutate(loadTest)
			_, err := renderK6Project(k6ProjectDocument(map[string]any{
				"usage": "development", "loadTests": []any{loadTest},
			}), nil, k6Config())
			if err == nil || !strings.Contains(err.Error(), tc.expectText) {
				t.Fatalf("bounded workload error = %v, want %q", err, tc.expectText)
			}
		})
	}
}

func TestK6ProjectRejectsUsageMismatchForDynamicChildren(t *testing.T) {
	_, err := renderK6Project(k6ProjectDocument(map[string]any{
		"usage": "production",
		"loadTests": []any{map[string]any{
			"name": "smoke", "workload": k6HTTPWorkload("https://service.example.invalid/health"), "vus": float64(1),
			"durationSeconds": float64(60), "loadZones": []any{},
		}},
	}), nil, k6Config())
	if err == nil || !strings.Contains(err.Error(), `spec.usage "production" does not match trusted referenced stack usage "development"`) {
		t.Fatalf("usage mismatch error = %v", err)
	}
}

func TestK6ProjectDynamicChildrenAreDeleteManaged(t *testing.T) {
	claim := k6ProjectDocument(map[string]any{
		"usage": "development",
		"loadTests": []any{map[string]any{
			"name": "smoke", "workload": k6HTTPWorkload("https://service.example.invalid/health"), "vus": float64(1),
			"durationSeconds": float64(60), "loadZones": []any{},
		}},
		"schedules": []any{map[string]any{
			"name": "nightly", "loadTest": "smoke", "starts": "2030-01-01T00:00:00Z",
		}},
	})
	observed := k6BootstrapObserved()
	firstDesired, err := renderK6Project(claim, observed, k6Config())
	if err != nil {
		t.Fatalf("first renderK6Project returned an error: %v", err)
	}
	for name, value := range k6CurrentPlatformCapObservations(t, firstDesired) {
		observed[name] = value
	}
	observed["load-test-smoke"] = observedComposed(`{"status":{"atProvider":{"id":"501"}}}`)
	observed["schedule-nightly"] = observedComposed(`{"status":{"atProvider":{"id":"601"}}}`)
	desired, err := renderK6Project(claim, observed, k6Config())
	if err != nil {
		t.Fatalf("second renderK6Project returned an error: %v", err)
	}
	wantDynamicPolicies := []any{"Create", "Observe", "Update", "Delete", "LateInitialize"}
	for _, name := range []resource.Name{"load-test-smoke", "schedule-nightly"} {
		policies := nestedMap(t, k6Desired(t, desired, name), "spec")["managementPolicies"]
		if diff := cmp.Diff(wantDynamicPolicies, policies); diff != "" {
			t.Errorf("%s management policies differ (-want +got):\n%s", name, diff)
		}
	}
	projectPolicies := nestedMap(t, k6Desired(t, desired, "project"), "spec")["managementPolicies"]
	if diff := cmp.Diff(managementPolicies, projectPolicies); diff != "" {
		t.Fatalf("retained project management policies differ (-want +got):\n%s", diff)
	}

	withdrawn := k6ProjectDocument(map[string]any{"usage": "development"})
	withdrawnDesired, err := renderK6Project(withdrawn, observed, k6Config())
	if err != nil {
		t.Fatalf("renderK6Project after dynamic withdrawal returned an error: %v", err)
	}
	for _, name := range []resource.Name{"load-test-smoke", "schedule-nightly"} {
		if _, ok := withdrawnDesired[name]; ok {
			t.Errorf("withdrawn dynamic child %q remained in desired state", name)
		}
	}
	if _, ok := withdrawnDesired["project"]; !ok {
		t.Fatal("retained project disappeared when its dynamic children were withdrawn")
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

func k6BootstrapObserved() map[resource.Name]resource.ObservedComposed {
	return map[resource.Name]resource.ObservedComposed{
		"installation":         observedComposed(`{"status":{"conditions":[{"type":"Ready","status":"True"}]}}`),
		"provider-credentials": observedComposed(`{"status":{"conditions":[{"type":"Ready","status":"True"}]}}`),
		"project":              observedComposed(`{"status":{"atProvider":{"id":"42"}}}`),
	}
}

func k6CurrentPlatformCapObservations(t *testing.T, desired map[resource.Name]*resource.DesiredComposed) map[resource.Name]resource.ObservedComposed {
	t.Helper()
	return map[resource.Name]resource.ObservedComposed{
		"limits":             k6ObservedReadyForDesired(t, desired["limits"]),
		"allowed-load-zones": k6ObservedReadyForDesired(t, desired["allowed-load-zones"]),
	}
}

func k6ObservedReadyForDesired(t *testing.T, desired *resource.DesiredComposed) resource.ObservedComposed {
	t.Helper()
	if desired == nil || desired.Resource == nil {
		t.Fatal("cannot build an observed fixture without the desired resource")
	}
	observed := desired.Resource.DeepCopy()
	object := observed.UnstructuredContent()
	metadata, ok := object["metadata"].(map[string]any)
	if !ok {
		t.Fatal("desired fixture has no metadata object")
	}
	metadata["generation"] = int64(1)
	object["status"] = map[string]any{
		"conditions": []any{
			map[string]any{"type": "Synced", "status": "True", "observedGeneration": int64(1)},
			map[string]any{"type": "Ready", "status": "True"},
		},
	}
	observed.SetUnstructuredContent(object)
	return resource.ObservedComposed{Resource: observed}
}

func k6HTTPWorkload(workloadURL string) map[string]any {
	return map[string]any{"httpGet": map[string]any{"url": workloadURL}}
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

func TestK6LoadTestCapAdmissionUsesTheCompositionProfile(t *testing.T) {
	if !admissionHarnessImplemented {
		t.Skip("admission harness pre-pass: implementation pending")
	}
	env := &admissionEnv{paths: []string{"../apis/k6-v1beta1.yaml"}}
	if err := env.Start(t); err != nil {
		t.Fatalf("start admission environment: %v", err)
	}
	t.Cleanup(func() {
		if err := env.Stop(); err != nil {
			t.Errorf("stop admission environment: %v", err)
		}
	})
	ctx := context.Background()
	allowed := k6AdmissionRequest("k6-cap-allowed", 10)
	if err := applyK6AdmissionEventually(ctx, env, allowed, true, ""); err != nil {
		t.Fatalf("profile-sized load test was refused: %v", err)
	}
	rejected := k6AdmissionRequest("k6-cap-rejected", 11)
	const capMessage = "load test regular VUs exceed the platform ProjectLimits cap"
	err := applyK6AdmissionEventually(ctx, env, rejected, false, capMessage)
	if err == nil || !strings.Contains(err.Error(), "load test regular VUs exceed the platform ProjectLimits cap") {
		t.Fatalf("over-cap load test admission error = %v", err)
	}
	t.Logf("API server rejection output: %v", err)

	rawScript := k6AdmissionRequest("k6-raw-script", 1)
	rawLoadTest := rawScript.Object["spec"].(map[string]any)["loadTests"].([]any)[0].(map[string]any)
	rawLoadTest["script"] = "export default function () {}"
	const scriptMessage = "script is unsupported; use workload.httpGet.url"
	err = applyK6AdmissionEventually(ctx, env, rawScript, false, scriptMessage)
	if err == nil || !strings.Contains(err.Error(), scriptMessage) {
		t.Fatalf("raw script admission error = %v", err)
	}
	t.Logf("API server raw script rejection output: %v", err)

	nullScript := k6AdmissionRequest("k6-null-script", 1)
	nullLoadTest := nullScript.Object["spec"].(map[string]any)["loadTests"].([]any)[0].(map[string]any)
	nullLoadTest["script"] = nil
	if err := applyK6AdmissionEventually(ctx, env, nullScript, true, ""); err != nil {
		t.Fatalf("nullable script compatibility value was refused: %v", err)
	}
	t.Log("API server accepted null compatibility script value")
}

func TestK6LoadTestCapAdmissionNegativeControlWeakensAndRestoresProfile(t *testing.T) {
	if !admissionHarnessImplemented {
		t.Skip("admission harness pre-pass: implementation pending")
	}
	env := &admissionEnv{paths: []string{"../apis/k6-v1beta1.yaml"}}
	if err := env.Start(t); err != nil {
		t.Fatalf("start admission environment: %v", err)
	}
	t.Cleanup(func() {
		if err := env.Stop(); err != nil {
			t.Errorf("stop admission environment: %v", err)
		}
	})
	ctx := context.Background()
	const expectedMessage = "load test regular VUs exceed the platform ProjectLimits cap"
	baseline := k6AdmissionRequest("k6-cap-negative-baseline", 11)
	baselineErr := applyK6AdmissionEventually(ctx, env, baseline, false, expectedMessage)
	if baselineErr == nil || !strings.Contains(baselineErr.Error(), expectedMessage) {
		t.Fatalf("negative-control baseline did not reject with its own message: %v", baselineErr)
	}
	t.Logf("negative control baseline output: %v", baselineErr)

	source, err := os.ReadFile("../apis/k6-v1beta1.yaml")
	if err != nil {
		t.Fatalf("read negative-control source XRD: %v", err)
	}
	weakened := bytes.Replace(source, []byte("vuMaxPerTest: 10"), []byte("vuMaxPerTest: 100"), 1)
	if bytes.Equal(weakened, source) {
		t.Fatal("negative-control dynamic profile cap was not found")
	}
	scratchPath := filepath.Join(t.TempDir(), "k6-v1beta1.yaml")
	if err := os.WriteFile(scratchPath, weakened, 0o600); err != nil {
		t.Fatalf("write weakened scratch XRD: %v", err)
	}
	weakenedCRD, err := crdFromXRD(scratchPath)
	if err != nil {
		t.Fatalf("derive weakened scratch CRD: %v", err)
	}
	if err := env.Apply(ctx, weakenedCRD); err != nil {
		t.Fatalf("apply weakened scratch CRD: %v", err)
	}
	weakenedComposition := k6CompositionFromSource(t, scratchPath)
	if err := env.Apply(ctx, weakenedComposition); err != nil {
		t.Fatalf("apply weakened Composition: %v", err)
	}
	weakenedErr := applyK6AdmissionEventually(ctx, env, k6AdmissionRequest("k6-cap-negative-weakened", 11), true, "")
	if weakenedErr != nil {
		t.Fatalf("negative control weakened output: %v", weakenedErr)
	}
	t.Log("negative control weakened output: admitted")

	originalCRD, err := crdFromXRD("../apis/k6-v1beta1.yaml")
	if err != nil {
		t.Fatalf("derive original CRD for restore: %v", err)
	}
	if err := env.Apply(ctx, originalCRD); err != nil {
		t.Fatalf("restore original CRD: %v", err)
	}
	originalComposition := k6CompositionFromSource(t, "../apis/k6-v1beta1.yaml")
	if err := env.Apply(ctx, originalComposition); err != nil {
		t.Fatalf("restore original Composition: %v", err)
	}
	restoredErr := applyK6AdmissionEventually(ctx, env, k6AdmissionRequest("k6-cap-negative-restored", 11), false, expectedMessage)
	if restoredErr == nil || !strings.Contains(restoredErr.Error(), expectedMessage) {
		t.Fatalf("negative control restored output did not reject with its own message: %v", restoredErr)
	}
	t.Logf("negative control restored output: %v", restoredErr)
}

func k6AdmissionRequest(name string, vus int) *unstructured.Unstructured {
	return requestObject("GrafanaK6Project", name, map[string]any{
		"stackRef":         map[string]any{"name": "stack001"},
		"grafanaUser":      "example-user",
		"usage":            "development",
		"allowedLoadZones": []any{"private-zone-example-a"},
		"loadTests": []any{map[string]any{
			"name": "smoke", "workload": k6HTTPWorkload("https://service.example.invalid/health"), "vus": float64(vus),
			"durationSeconds": float64(60), "loadZones": []any{},
		}},
	})
}

func applyK6AdmissionEventually(ctx context.Context, env *admissionEnv, object *unstructured.Unstructured, wantAdmitted bool, expectedRefusal string) error {
	deadline := time.Now().Add(10 * time.Second)
	var last error
	for time.Now().Before(deadline) {
		candidate := object.DeepCopy()
		err := env.Apply(ctx, candidate)
		if wantAdmitted && err == nil {
			return nil
		}
		if !wantAdmitted && err != nil && (expectedRefusal == "" || strings.Contains(err.Error(), expectedRefusal)) {
			return err
		}
		last = err
		time.Sleep(100 * time.Millisecond)
	}
	return last
}

func k6CompositionFromSource(t *testing.T, path string) *unstructured.Unstructured {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Composition source: %v", err)
	}
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(raw), 4096)
	for {
		object := &unstructured.Unstructured{}
		err := decoder.Decode(&object.Object)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("decode Composition source: %v", err)
		}
		if object.GetAPIVersion() == "apiextensions.crossplane.io/v1" && object.GetKind() == "Composition" {
			return object
		}
	}
	t.Fatal("Composition document was not found")
	return nil
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
