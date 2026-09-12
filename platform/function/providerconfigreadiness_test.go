package main

import (
	"testing"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
)

func TestObservedProviderConfigCompletesTelemetryOnlyStack(t *testing.T) {
	rsp := runStack(t, telemetryOnlyStackClaim(), telemetryOnlyStackObserved())

	resources := rsp.GetDesired().GetResources()
	wantNames := []string{
		"credentials",
		"fleet-management-access-policy",
		"fleet-management-credentials",
		"fleet-management-token",
		"instance-credentials",
		"provider-config",
		"stack",
		"stack-service-account",
		"stack-token",
		"telemetry-access-policy",
		"telemetry-credentials",
		"telemetry-token",
	}
	if len(resources) != len(wantNames) {
		t.Fatalf("desired resource count = %d, want %d", len(resources), len(wantNames))
	}
	for _, name := range wantNames {
		desired, ok := resources[name]
		if !ok {
			t.Fatalf("desired resource %q was not rendered", name)
		}
		if got := desired.GetReady(); got != fnv1.Ready_READY_TRUE {
			t.Errorf("desired resource %q readiness = %s, want READY_TRUE", name, got)
		}
	}
	for _, name := range []string{
		"billing-dashboard", "billing-folder", "endpoints-dashboard", "endpoints-folder",
		"homepage-dashboard", "homepage-folder", "organization-preferences",
	} {
		if _, ok := resources[name]; ok {
			t.Errorf("telemetry-only stack rendered in-stack resource %q", name)
		}
	}
}

func TestProviderConfigIsNotReadyBeforeObservation(t *testing.T) {
	rsp := runStack(t, telemetryOnlyStackClaim(), foundationReadyObserved())

	providerConfig := rsp.GetDesired().GetResources()["provider-config"]
	if got := providerConfig.GetReady(); got == fnv1.Ready_READY_TRUE {
		t.Fatal("unobserved ProviderConfig was marked READY_TRUE")
	}
	if got := providerConfig.GetReady(); got != fnv1.Ready_READY_UNSPECIFIED {
		t.Fatalf("unobserved ProviderConfig readiness = %s, want READY_UNSPECIFIED", got)
	}
}

func TestObservedProviderConfigDoesNotMaskUnreadyChild(t *testing.T) {
	observed := telemetryOnlyStackObserved()
	observed["stack"] = observedResource(`{
		"apiVersion":"cloud.grafana.m.crossplane.io/v1alpha1",
		"kind":"Stack",
		"metadata":{"name":"teamdemo01","namespace":"grafana-vending"},
		"status":{
			"atProvider":{"id":"12345"},
			"conditions":[{"type":"Ready","status":"False","reason":"Unavailable"}]
		}
	}`)

	rsp := runStack(t, telemetryOnlyStackClaim(), observed)
	resources := rsp.GetDesired().GetResources()
	if got := resources["provider-config"].GetReady(); got != fnv1.Ready_READY_TRUE {
		t.Fatalf("observed ProviderConfig readiness = %s, want READY_TRUE", got)
	}
	if got := resources["stack"].GetReady(); got == fnv1.Ready_READY_TRUE {
		t.Fatal("unready Stack was marked READY_TRUE")
	}
}

func telemetryOnlyStackClaim() string {
	return stackDocument(map[string]any{
		"baselineDashboards": map[string]any{"enabled": false},
	})
}

func telemetryOnlyStackObserved() map[string]*fnv1.Resource {
	observed := foundationReadyObserved()
	observed["provider-config"] = observedResource(`{
		"apiVersion":"grafana.m.crossplane.io/v1beta1",
		"kind":"ProviderConfig",
		"metadata":{"name":"teamdemo01","namespace":"grafana-vending"}
	}`)
	for _, name := range []string{"credentials", "fleet-management-credentials", "fleet-management-token", "stack-token", "telemetry-credentials", "telemetry-token"} {
		observed[name] = observedResource(`{"status":{"conditions":[{"type":"Ready","status":"True"}]}}`)
	}
	observed["fleet-management-access-policy"] = observedResource(`{
		"status":{
			"atProvider":{"policyId":"fleet-policy-12345"},
			"conditions":[{"type":"Ready","status":"True"}]
		}
	}`)
	observed["telemetry-access-policy"] = observedResource(`{
		"status":{
			"atProvider":{"policyId":"telemetry-policy-12345"},
			"conditions":[{"type":"Ready","status":"True"}]
		}
	}`)
	return observed
}
