package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/crossplane/function-sdk-go/resource"
)

func TestRetentionClassRendersPlatformOwnedDurableFanout(t *testing.T) {
	profile := map[string]any{
		"name":       "long-term",
		"configType": "ALLOY",
		"contents":   `prometheus.remote_write "durable_metrics" {}`,
		"matchers":   []any{`{environment=~".+"}`},
	}
	desired := map[resource.Name]*resource.DesiredComposed{}
	if err := addRetentionFanout(desired, retentionTestXR("long-term"), nil, retentionTestConfig(profile)); err != nil {
		t.Fatalf("addRetentionFanout returned an error: %v", err)
	}

	pipeline, ok := desired["retention-fanout"]
	if !ok {
		t.Fatal("retention fan-out pipeline was not rendered")
	}
	content := pipeline.Resource.UnstructuredContent()
	if got, want := content["apiVersion"], "fleetmanagement.grafana.m.crossplane.io/v1alpha1"; got != want {
		t.Fatalf("pipeline apiVersion = %v, want %s", got, want)
	}
	if got, want := content["kind"], "Pipeline"; got != want {
		t.Fatalf("pipeline kind = %v, want %s", got, want)
	}
	if got := desiredExternalName(t, content); got != "teamdemo01-retention-fanout" {
		t.Fatalf("pipeline external name = %q, want teamdemo01-retention-fanout", got)
	}
	metadata := nestedMap(t, content, "metadata")
	if got, want := metadata["name"], "teamdemo01-retention-fanout"; got != want {
		t.Fatalf("pipeline metadata.name = %v, want %s", got, want)
	}
	if got, want := metadata["namespace"], "grafana-vending"; got != want {
		t.Fatalf("pipeline metadata.namespace = %v, want %s", got, want)
	}

	spec := nestedMap(t, content, "spec")
	if got, want := spec["managementPolicies"], managementPolicies; !reflect.DeepEqual(got, want) {
		t.Fatalf("pipeline managementPolicies = %#v, want %#v", got, want)
	}
	if got, want := nestedMap(t, spec, "providerConfigRef")["name"], "teamdemo01"; got != want {
		t.Fatalf("pipeline ProviderConfig = %v, want %s", got, want)
	}
	forProvider := nestedMap(t, spec, "forProvider")
	wantForProvider := map[string]any{
		"configType":               "ALLOY",
		"contents":                 profile["contents"],
		"enabled":                  true,
		"matchers":                 profile["matchers"],
		"name":                     "teamdemo01-retention-fanout",
		"terraformSourceNamespace": "default",
	}
	if !reflect.DeepEqual(forProvider, wantForProvider) {
		t.Fatalf("pipeline forProvider = %#v, want %#v", forProvider, wantForProvider)
	}
	if _, found := content["retention"]; found {
		t.Fatal("pipeline unexpectedly contains a retention setting")
	}
}

func TestRetentionClassOmittedDoesNotRender(t *testing.T) {
	desired := map[resource.Name]*resource.DesiredComposed{}
	if err := addRetentionFanout(desired, retentionTestXR(""), nil, retentionTestConfig(nil)); err != nil {
		t.Fatalf("omitted retention returned an error: %v", err)
	}
	if len(desired) != 0 {
		t.Fatalf("omitted retention rendered %#v", desired)
	}
}

func TestRetentionClassIsRenderedByStackRequest(t *testing.T) {
	claim := stackDocument(map[string]any{"retention": map[string]any{"class": "long-term"}})
	input := retentionTestConfig(map[string]any{
		"name": "long-term", "contents": "platform-owned pipeline", "matchers": []any{"{environment=~\".+\"}"},
	})
	rsp := runStackWithInput(t, claim, foundationReadyObserved(), mustJSON(input))
	if _, found := rsp.GetDesired().GetResources()["retention-fanout"]; !found {
		t.Fatal("stack request did not render the retention fan-out pipeline")
	}
}

func TestRetentionClassAcceptsKeyedPlatformProfile(t *testing.T) {
	desired := map[resource.Name]*resource.DesiredComposed{}
	config := map[string]any{"spec": map[string]any{"retentionProfiles": map[string]any{
		"long-term": map[string]any{
			"configType": "OTEL", "contents": "platform-owned OTEL pipeline", "matchers": []any{"{environment=~\".+\"}"},
		},
	}}}
	if err := addRetentionFanout(desired, retentionTestXR("long-term"), nil, config); err != nil {
		t.Fatalf("keyed retention profile returned an error: %v", err)
	}
	forProvider := nestedMap(t, desired["retention-fanout"].Resource.UnstructuredContent(), "spec", "forProvider")
	if got, want := forProvider["configType"], "OTEL"; got != want {
		t.Fatalf("keyed retention profile configType = %v, want %s", got, want)
	}
}

func TestRetentionClassRequiresValidPlatformProfile(t *testing.T) {
	tests := []struct {
		name    string
		profile map[string]any
		want    string
	}{
		{name: "unconfigured class", want: "not configured by the platform"},
		{
			name: "missing contents",
			profile: map[string]any{
				"name": "long-term", "matchers": []any{`{environment=~".+"}`},
			},
			want: "must set contents",
		},
		{
			name: "missing matchers",
			profile: map[string]any{
				"name": "long-term", "contents": "platform-owned pipeline",
			},
			want: "must set at least one matcher",
		},
		{
			name: "unsupported config type",
			profile: map[string]any{
				"name": "long-term", "configType": "JSON", "contents": "platform-owned pipeline",
				"matchers": []any{`{environment=~".+"}`},
			},
			want: "configType must be ALLOY or OTEL",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			desired := map[resource.Name]*resource.DesiredComposed{}
			err := addRetentionFanout(desired, retentionTestXR("long-term"), nil, retentionTestConfig(test.profile))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
			if len(desired) != 0 {
				t.Fatalf("invalid retention profile rendered %#v", desired)
			}
		})
	}
}

func TestRetentionClassRejectsUnrecognisedRetentionFields(t *testing.T) {
	xr := retentionTestXR("long-term")
	xr["spec"].(map[string]any)["retention"] = map[string]any{
		"class":  "long-term",
		"period": "365d",
	}
	desired := map[resource.Name]*resource.DesiredComposed{}
	err := addRetentionFanout(desired, xr, nil, retentionTestConfig(map[string]any{
		"name": "long-term", "contents": "platform-owned pipeline", "matchers": []any{"{environment=~\".+\"}"},
	}))
	if err == nil || !strings.Contains(err.Error(), "retention supports only class") {
		t.Fatalf("period field error = %v, want retention field rejection", err)
	}
	if len(desired) != 0 {
		t.Fatalf("retention period field rendered %#v", desired)
	}
}

func retentionTestXR(class string) map[string]any {
	spec := map[string]any{}
	if class != "" {
		spec["retention"] = map[string]any{"class": class}
	}
	spec["slug"] = "teamdemo01"
	return map[string]any{
		"apiVersion": "platform.example.org/v1beta1",
		"kind":       "GrafanaCloudStackRequest",
		"metadata":   map[string]any{"name": "teamdemo01", "namespace": "grafana-vending"},
		"spec":       spec,
	}
}

func retentionTestConfig(profile map[string]any) map[string]any {
	spec := map[string]any{}
	if profile != nil {
		spec["retentionProfiles"] = []any{profile}
	}
	return map[string]any{"spec": spec}
}
