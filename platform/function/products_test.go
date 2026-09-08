package main

import "testing"

func TestObservabilityProductsRenderOnlyEnabledSingletons(t *testing.T) {
	products := []struct {
		field, name, kind string
	}{
		{field: "applicationObservability", name: "application-observability", kind: "Appo11YconfigV1Alpha1"},
		{field: "kubernetesObservability", name: "kubernetes-observability", kind: "K8So11YconfigV1Alpha1"},
		{field: "databaseObservability", name: "database-observability", kind: "Dbo11YconfigV1Alpha1"},
	}

	for _, tc := range products {
		t.Run(tc.field, func(t *testing.T) {
			flags := map[string]any{
				"applicationObservability": false,
				"kubernetesObservability":  false,
				"databaseObservability":    false,
			}
			flags[tc.field] = true
			rsp := runStack(t, stackDocument(map[string]any{"products": flags}), foundationReadyObserved())
			resource := desiredResource(t, rsp, tc.name)
			if got := resource["apiVersion"]; got != "cloud.grafana.m.crossplane.io/v1alpha1" {
				t.Fatalf("%s apiVersion = %v, want cloud.grafana.m.crossplane.io/v1alpha1", tc.name, got)
			}
			if got := resource["kind"]; got != tc.kind {
				t.Fatalf("%s kind = %v, want %s", tc.name, got, tc.kind)
			}
			if got := desiredExternalName(t, resource); got != "global" {
				t.Fatalf("%s external name = %q, want global", tc.name, got)
			}
			parameters := nestedMap(t, resource, "spec", "forProvider")
			if got := nestedMap(t, parameters, "metadata")["uid"]; got != "global" {
				t.Fatalf("%s metadata uid = %v, want global", tc.name, got)
			}
			if got := nestedMap(t, parameters, "spec")["enabled"]; got != true {
				t.Fatalf("%s enabled = %v, want true", tc.name, got)
			}
			providerRef := nestedMap(t, resource, "spec", "providerConfigRef")
			if got := providerRef["kind"]; got != "ProviderConfig" {
				t.Fatalf("%s provider ref kind = %v, want ProviderConfig", tc.name, got)
			}
			if got := providerRef["name"]; got != "teamdemo01" {
				t.Fatalf("%s provider ref name = %v, want teamdemo01", tc.name, got)
			}
			for _, policy := range nestedMap(t, resource, "spec")["managementPolicies"].([]any) {
				if policy == "Delete" {
					t.Fatalf("%s management policy unexpectedly permits external deletion", tc.name)
				}
			}

			for _, other := range products {
				if other.name == tc.name {
					continue
				}
				if _, ok := rsp.GetDesired().GetResources()[other.name]; ok {
					t.Fatalf("disabled %s rendered a singleton", other.field)
				}
			}
		})
	}
}

func TestObservabilityProductDisableWithdrawsTheSingleton(t *testing.T) {
	observed := foundationReadyObserved()
	for _, tc := range []struct {
		name, kind string
	}{
		{name: "application-observability", kind: "Appo11YconfigV1Alpha1"},
		{name: "kubernetes-observability", kind: "K8So11YconfigV1Alpha1"},
		{name: "database-observability", kind: "Dbo11YconfigV1Alpha1"},
	} {
		observed[tc.name] = observedResource(`{"apiVersion":"cloud.grafana.m.crossplane.io/v1alpha1","kind":"` + tc.kind + `","metadata":{"name":"previously-desired"}}`)
	}
	rsp := runStack(t, stackDocument(map[string]any{"products": map[string]any{
		"applicationObservability": false,
		"kubernetesObservability":  false,
		"databaseObservability":    false,
	}}), observed)
	for _, name := range []string{"application-observability", "kubernetes-observability", "database-observability"} {
		if _, ok := rsp.GetDesired().GetResources()[name]; ok {
			t.Fatalf("disabled product %s remained desired after omission", name)
		}
	}
}
