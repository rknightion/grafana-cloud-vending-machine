package main

import (
	"github.com/crossplane/function-sdk-go/resource"
	"testing"
	"time"
)

func TestSMWholeSetOwnsExternalCheckDeletion(t *testing.T) {
	desired, err := renderSyntheticMonitoring(syntheticMonitoringClaim(), syntheticMonitoringVerifiedObserved("12345"), syntheticMonitoringPlatformConfig(true))
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for name, child := range desired {
		if child.Resource.GetKind() != "Check" {
			continue
		}
		count++
		spec := child.Resource.UnstructuredContent()["spec"].(map[string]any)
		all := false
		for _, policy := range spec["managementPolicies"].([]any) {
			if policy == "*" || policy == "Delete" {
				all = true
			}
		}
		if !all {
			t.Fatalf("%s would remain active after removal from the whole check set", name)
		}
	}
	if count < 2 {
		t.Fatal("did not exercise verifier and team checks")
	}
}

func TestExpiryWarningOwnsExternalRemoval(t *testing.T) {
	desired := map[resource.Name]*resource.DesiredComposed{"incident-alerting-production": newDesired("alerting.grafana.m.crossplane.io/v1alpha1", "ContactPoint", "grafana-vending", "contact", nil, map[string]any{"forProvider": map[string]any{"name": "existing-contact"}})}
	xr := map[string]any{"metadata": map[string]any{"namespace": "grafana-vending"}, "spec": map[string]any{"slug": "example"}}
	if err := addExpiryWarning(desired, xr, expiryPolicy{warningFolder: "existing-folder"}, time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	spec := desired["expiry-warning"].Resource.UnstructuredContent()["spec"].(map[string]any)
	for _, policy := range spec["managementPolicies"].([]any) {
		if policy == "*" || policy == "Delete" {
			return
		}
	}
	t.Fatal("temporary warning would remain firing after expiry")
}
