package main

import "testing"

func TestObservabilityProductsRenderOnlyEnabledSingletons(t *testing.T) {
	claim := stackDocument(map[string]any{"products": map[string]any{
		"applicationObservability": true,
		"kubernetesObservability":  false,
		"databaseObservability":    true,
	}})
	rsp := runStack(t, claim, foundationReadyObserved())

	for _, tc := range []struct {
		name string
		kind string
	}{
		{name: "application-observability", kind: "Appo11YconfigV1Alpha1"},
		{name: "database-observability", kind: "Dbo11YconfigV1Alpha1"},
	} {
		resource := desiredResource(t, rsp, tc.name)
		if got := resource["kind"]; got != tc.kind {
			t.Fatalf("%s kind = %v, want %s", tc.name, got, tc.kind)
		}
		parameters := nestedMap(t, resource, "spec", "forProvider")
		if got := nestedMap(t, parameters, "metadata")["uid"]; got != "global" {
			t.Fatalf("%s metadata uid = %v, want global", tc.name, got)
		}
		if got := nestedMap(t, parameters, "spec")["enabled"]; got != true {
			t.Fatalf("%s enabled = %v, want true", tc.name, got)
		}
	}
	if _, ok := rsp.GetDesired().GetResources()["kubernetes-observability"]; ok {
		t.Fatal("disabled kubernetesObservability rendered a singleton")
	}
}

func TestObservabilityProductDisableWithdrawsTheSingleton(t *testing.T) {
	rsp := runStack(t, stackDocument(map[string]any{"products": map[string]any{
		"applicationObservability": false,
		"kubernetesObservability":  false,
		"databaseObservability":    false,
	}}), foundationReadyObserved())
	for _, name := range []string{"application-observability", "kubernetes-observability", "database-observability"} {
		if _, ok := rsp.GetDesired().GetResources()[name]; ok {
			t.Fatalf("disabled product %s remained desired", name)
		}
	}
}
