package main

import (
	"strings"
	"testing"

	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
)

func TestGoldenSLOUsesUsageProfileAfterDatasourceObservation(t *testing.T) {
	xr := goldenSLOStack("production")
	config := goldenSLOConfig()

	before := map[resource.Name]*resource.DesiredComposed{}
	if err := addGoldenSLO(before, xr, nil, config); err != nil {
		t.Fatalf("addGoldenSLO before datasource observation returned an error: %v", err)
	}
	if _, ok := before[goldenSLOLogicalName]; ok {
		t.Fatal("SLO rendered before its destination datasource was observed")
	}
	datasource := goldenSLODesired(t, before, goldenSLODatasourceLogicalName)
	if got := datasource["kind"]; got != "DataSource" {
		t.Fatalf("destination observer kind = %v, want DataSource", got)
	}
	if got := nestedMap(t, datasource, "spec")["managementPolicies"]; gotValueDiff(got, []any{"Observe"}) {
		t.Fatalf("destination observer management policies = %#v, want Observe", got)
	}

	after := map[resource.Name]*resource.DesiredComposed{}
	observed := map[resource.Name]resource.ObservedComposed{
		goldenSLODatasourceLogicalName: goldenSLOObservedDatasource("observed-prometheus"),
	}
	if err := addGoldenSLO(after, xr, observed, config); err != nil {
		t.Fatalf("addGoldenSLO after datasource observation returned an error: %v", err)
	}
	slo := goldenSLODesired(t, after, goldenSLOLogicalName)
	if got, want := slo["apiVersion"], "slo.grafana.m.crossplane.io/v1alpha1"; got != want {
		t.Fatalf("SLO apiVersion = %v, want %s", got, want)
	}
	if got, want := slo["kind"], "SLO"; got != want {
		t.Fatalf("SLO kind = %v, want %s", got, want)
	}
	spec := nestedMap(t, slo, "spec")
	if _, exists := spec["forProvider"]; !exists {
		t.Fatal("handoff SLO omitted its empty forProvider")
	}
	parameters := nestedMap(t, spec, "initProvider")
	if got, want := parameters["name"], "Production request success"; got != want {
		t.Fatalf("usage-selected SLO name = %v, want %s", got, want)
	}
	if got, want := parameters["destinationDatasource"], []any{map[string]any{"uid": "observed-prometheus"}}; gotValueDiff(got, want) {
		t.Fatalf("destination datasource = %#v, want %#v", got, want)
	}
	query := parameters["query"].([]any)[0].(map[string]any)
	if got, want := query["type"], "ratio"; got != want {
		t.Fatalf("query type = %v, want %s", got, want)
	}
	ratio := query["ratio"].([]any)[0].(map[string]any)
	if got, want := ratio["successMetric"], "example_request_success_total"; got != want {
		t.Fatalf("ratio success metric = %v, want %s", got, want)
	}
	alerting := parameters["alerting"].([]any)[0].(map[string]any)
	for _, ruleType := range []string{"fastburn", "slowburn"} {
		if _, ok := alerting[ruleType]; !ok {
			t.Errorf("alerting omitted %s generated rule configuration", ruleType)
		}
	}
	if containsKey(parameters, "enrichment") {
		t.Fatal("SLO rendered preview alert enrichment")
	}
	if got := desiredExternalName(t, slo); got == "" || got != parameters["uuid"] {
		t.Fatalf("SLO external name = %q, want its deterministic UUID %v", got, parameters["uuid"])
	}
}

func TestGoldenSLORejectsMalformedPlatformProfiles(t *testing.T) {
	xr := goldenSLOStack("production")
	for name, profiles := range map[string]any{
		"string":          "invalid",
		"list":            []any{},
		"profile string":  map[string]any{"production": "invalid"},
		"missing query":   map[string]any{"production": map[string]any{"name": "Example", "datasourceUID": "example-prometheus", "objective": map[string]any{"value": 0.99, "window": "30d"}}},
		"non-ratio query": map[string]any{"production": map[string]any{"name": "Example", "datasourceUID": "example-prometheus", "objective": map[string]any{"value": 0.99, "window": "30d"}, "ratio": "invalid"}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := goldenSLODesiredSet(xr, map[string]any{"spec": map[string]any{"goldenSLOProfiles": profiles}})
			if err == nil || !strings.Contains(err.Error(), "goldenSLOProfiles") {
				t.Fatalf("malformed %s profiles error = %v, want goldenSLOProfiles rejection", name, err)
			}
		})
	}
}

func TestGoldenSLOSkipsUsageWithoutPlatformProfile(t *testing.T) {
	desired, err := goldenSLODesiredSet(goldenSLOStack("development"), goldenSLOConfig())
	if err != nil {
		t.Fatalf("addGoldenSLO returned an error: %v", err)
	}
	if len(desired) != 0 {
		t.Fatalf("usage without a platform profile rendered %#v, want no resources", desired)
	}
}

func goldenSLODesiredSet(xr, config map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	desired := map[resource.Name]*resource.DesiredComposed{}
	return desired, addGoldenSLO(desired, xr, nil, config)
}

func goldenSLOStack(usage string) map[string]any {
	return map[string]any{
		"metadata": map[string]any{"name": "example-stack", "namespace": "grafana-vending"},
		"spec":     map[string]any{"slug": "example-stack", "usage": usage},
	}
}

func goldenSLOConfig() map[string]any {
	return map[string]any{"spec": map[string]any{"goldenSLOProfiles": map[string]any{
		"production": map[string]any{
			"name": "Production request success", "description": "Inert production ratio template.",
			"datasourceUID": "example-prometheus",
			"objective":     map[string]any{"value": 0.99, "window": "30d"},
			"ratio": map[string]any{
				"successMetric": "example_request_success_total", "totalMetric": "example_request_total",
			},
		},
	}}}
}

func goldenSLOObservedDatasource(uid string) resource.ObservedComposed {
	value := composed.New()
	value.SetUnstructuredContent(resource.MustStructJSON(`{"status":{"atProvider":{"uid":"` + uid + `"}}}`).AsMap())
	return resource.ObservedComposed{Resource: value}
}

func goldenSLODesired(t *testing.T, desired map[resource.Name]*resource.DesiredComposed, name resource.Name) map[string]any {
	t.Helper()
	child, ok := desired[name]
	if !ok {
		t.Fatalf("desired resource %q was not rendered", name)
	}
	return child.Resource.UnstructuredContent()
}

func containsKey(value any, key string) bool {
	if object, ok := value.(map[string]any); ok {
		for candidate, nested := range object {
			if candidate == key || containsKey(nested, key) {
				return true
			}
		}
	}
	if values, ok := value.([]any); ok {
		for _, nested := range values {
			if containsKey(nested, key) {
				return true
			}
		}
	}
	return false
}
