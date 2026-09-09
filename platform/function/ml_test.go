package main

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestMLRendersPlatformOwnedHolidayAndContinuouslyRunningResources(t *testing.T) {
	desired, err := renderML(mlClaim(), nil, mlConfig(2))
	if err != nil {
		t.Fatalf("render ML without observed holiday: %v", err)
	}
	if _, found := desired["holiday-example-holiday"]; !found {
		t.Fatal("ML did not render the platform-owned holiday")
	}
	if _, found := desired["outlier-detector-example-outlier-detector"]; !found {
		t.Fatal("ML did not render the platform-owned outlier detector")
	}
	if _, found := desired["job-example-forecast"]; found {
		t.Fatal("ML rendered a job before its provider-assigned holiday ID was observed")
	}
	for name, desiredResource := range desired {
		object := desiredResource.Resource.UnstructuredContent()
		if got, want := object["apiVersion"], mlAPIVersion; got != want {
			t.Errorf("%s apiVersion = %q, want %q", name, got, want)
		}
		metadata := nestedMap(t, object, "metadata")
		if annotations, _ := metadata["annotations"].(map[string]any); annotations != nil && annotations["crossplane.io/external-name"] != nil {
			t.Errorf("%s derived a provider-assigned ML ID as external name: %#v", name, annotations)
		}
	}

	observed := map[resource.Name]resource.ObservedComposed{
		"holiday-example-holiday": mlObserved(`{"status":{"atProvider":{"id":"holiday-provider-id"}}}`),
	}
	desired, err = renderML(mlClaim(), observed, mlConfig(2))
	if err != nil {
		t.Fatalf("render ML with observed holiday: %v", err)
	}
	job, found := desired["job-example-forecast"]
	if !found {
		t.Fatal("ML did not render job after holiday ID observation")
	}
	if holidays, _ := nestedMap(t, job.Resource.UnstructuredContent(), "spec", "forProvider")["holidays"].([]any); !equalAny(holidays, []any{"holiday-provider-id"}) {
		t.Fatalf("job holidays = %#v, want observed holiday ID", holidays)
	}
}

func TestMLProfileCapFailsClosed(t *testing.T) {
	_, err := renderML(mlClaim(), nil, mlConfig(1))
	if err == nil || !strings.Contains(err.Error(), "exceeds maxRunningResources") {
		t.Fatalf("over-cap ML profile error = %v, want cap refusal", err)
	}
}

func TestMLRejectsDuplicateJobsWhileHolidayIDsArePending(t *testing.T) {
	config := mlConfig(3)
	profile := config["spec"].(map[string]any)["mlProfiles"].([]any)[0].(map[string]any)
	jobs := profile["jobs"].([]any)
	profile["jobs"] = append(jobs, jobs[0])

	_, err := renderML(mlClaim(), nil, config)
	if err == nil || !strings.Contains(err.Error(), `duplicate job name "example-forecast"`) {
		t.Fatalf("duplicate jobs awaiting holiday IDs error = %v, want duplicate-name refusal", err)
	}
}

func TestMLAdmissionProfileCapNegativeControl(t *testing.T) {
	source, err := os.ReadFile("../apis/ml-v1beta1.yaml")
	if err != nil {
		t.Fatalf("read ML API source: %v", err)
	}
	limited := strings.Replace(string(source), "maxRunningResources: 2", "maxRunningResources: 1", 1)
	if limited == string(source) {
		t.Fatal("ML negative control did not find the Composition profile cap")
	}
	scratch := filepath.Join(t.TempDir(), "ml-v1beta1.yaml")
	if err := os.WriteFile(scratch, []byte(limited), 0o600); err != nil {
		t.Fatalf("write limited ML admission source: %v", err)
	}

	env := &admissionEnv{paths: []string{scratch}}
	if err := env.Start(t); err != nil {
		t.Fatalf("start isolated ML admission environment: %v", err)
	}
	t.Cleanup(func() { _ = env.Stop() })
	ctx := context.Background()
	baselineErr := waitForMLCapRefusal(ctx, env, t, "mlovercapbaseline")
	t.Logf("ML cap baseline output: %v", baselineErr)

	weakened := mlCapPolicy("true")
	if err := env.Apply(ctx, weakened); err != nil {
		t.Fatalf("weaken ML cap admission policy: %v", err)
	}
	weakenedErr := waitForMLCapAdmission(ctx, env, t, "mlovercapweakened")
	if weakenedErr != nil {
		t.Fatalf("ML cap weakened output: %v", weakenedErr)
	}
	t.Log("ML cap weakened output: admitted")

	if err := env.Apply(ctx, mlCapPolicy(mlProfileCapExpression)); err != nil {
		t.Fatalf("restore ML cap admission policy: %v", err)
	}
	restoredErr := waitForMLCapRefusal(ctx, env, t, "mlovercaprestored")
	t.Logf("ML cap restored output: %v", restoredErr)
}

func waitForMLCapRefusal(ctx context.Context, env *admissionEnv, t *testing.T, prefix string) error {
	t.Helper()
	const message = "selected ML profile exceeds its platform-owned maxRunningResources cap"
	deadline := time.Now().Add(20 * time.Second)
	for attempt := 1; ; attempt++ {
		err := env.Apply(ctx, mlRequest(prefix+strconv.Itoa(attempt)))
		if err != nil && strings.Contains(err.Error(), message) {
			return err
		}
		if time.Now().After(deadline) {
			t.Fatalf("ML cap policy did not become active: final output %v", err)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func waitForMLCapAdmission(ctx context.Context, env *admissionEnv, t *testing.T, prefix string) error {
	t.Helper()
	const denied = "selected ML profile exceeds its platform-owned maxRunningResources cap"
	deadline := time.Now().Add(20 * time.Second)
	for attempt := 1; ; attempt++ {
		err := env.Apply(ctx, mlRequest(prefix+strconv.Itoa(attempt)))
		if err == nil {
			return nil
		}
		if !strings.Contains(err.Error(), denied) {
			t.Fatalf("ML cap weakened output was refused unexpectedly: %v", err)
		}
		if time.Now().After(deadline) {
			return err
		}
		time.Sleep(200 * time.Millisecond)
	}
}

const mlProfileCapExpression = "params.spec.pipeline[0].input.spec.mlProfiles.exists(profile, profile.name == object.spec.profile && size(profile.jobs) + size(profile.outlierDetectors) <= profile.maxRunningResources)"

func mlCapPolicy(expression string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "admissionregistration.k8s.io/v1", "kind": "ValidatingAdmissionPolicy",
		"metadata": map[string]any{"name": "grafana-ml-profile-cap-v1beta1"},
		"spec": map[string]any{
			"failurePolicy": "Fail",
			"paramKind":     map[string]any{"apiVersion": "apiextensions.crossplane.io/v1", "kind": "Composition"},
			"matchConstraints": map[string]any{"resourceRules": []any{map[string]any{
				"apiGroups": []any{"platform.example.org"}, "apiVersions": []any{"v1beta1"}, "operations": []any{"CREATE", "UPDATE"}, "resources": []any{"grafanamls"},
			}}},
			"validations": []any{map[string]any{"expression": expression, "message": "selected ML profile exceeds its platform-owned maxRunningResources cap"}},
		},
	}}
}

func mlRequest(name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "platform.example.org/v1beta1", "kind": "GrafanaML",
		"metadata": map[string]any{"name": name, "namespace": "default"},
		"spec":     map[string]any{"stackRef": map[string]any{"name": name}, "profile": "standard"},
	}}
}

func mlClaim() map[string]any {
	return map[string]any{
		"metadata": map[string]any{"name": "teamdemo01", "namespace": "grafana-vending"},
		"spec":     map[string]any{"stackRef": map[string]any{"name": "teamdemo01"}, "profile": "standard"},
	}
}

func mlConfig(cap int) map[string]any {
	return map[string]any{"referencedStack": map[string]any{"stackID": "12345"}, "spec": map[string]any{
		"mlProfiles": []any{map[string]any{
			"name": "standard", "maxRunningResources": float64(cap),
			"holidays": []any{map[string]any{"name": "example-holiday", "description": "Example holiday", "customPeriods": []any{map[string]any{
				"name": "example-window", "startTime": "2026-01-01T00:00:00Z", "endTime": "2026-01-02T00:00:00Z",
			}}}},
			"jobs":             []any{map[string]any{"name": "example-forecast", "datasourceType": "prometheus", "datasourceUid": "example-prometheus", "metric": "example_forecast", "queryParams": map[string]any{"expr": "up"}, "holidayRefs": []any{"example-holiday"}}},
			"outlierDetectors": []any{map[string]any{"name": "example-outlier-detector", "datasourceType": "prometheus", "datasourceUid": "example-prometheus", "metric": "example_outlier", "queryParams": map[string]any{"expr": "up"}, "algorithm": map[string]any{"name": "mad", "sensitivity": 0.5}}},
		}},
	}}
}

func mlObserved(document string) resource.ObservedComposed {
	value := composed.New()
	value.SetUnstructuredContent(observedResource(document).GetResource().AsMap())
	return resource.ObservedComposed{Resource: value}
}

func equalAny(got, want []any) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
