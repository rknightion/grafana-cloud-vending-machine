package main

import (
	"github.com/crossplane/function-sdk-go/resource"
	"testing"
)

func TestProviderAdmissionNewSurfaces(t *testing.T) {
	e := &admissionEnv{paths: []string{"../apis/stack-v1beta1.yaml"}}
	if err := e.Start(t); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := e.Stop(); err != nil {
			t.Error(err)
		}
	})
	installProviderAdmissionCRDs(t, e)
	cases := []struct {
		name       string
		render     func(map[string]any, map[resource.Name]resource.ObservedComposed, map[string]any) (map[resource.Name]*resource.DesiredComposed, error)
		xr, config map[string]any
		observed   map[resource.Name]resource.ObservedComposed
	}{
		{"cloud", renderCloudIntegrations, cloudIntegrationClaim(), cloudIntegrationConfig(), map[resource.Name]resource.ObservedComposed{"aws-account-example-account": observedComposed(`{"status":{"atProvider":{"resourceId":"provider-assigned-account-id"}}}`)}},
		{"pdc", renderPDC, pdcClaim(), pdcConfig("720h"), map[resource.Name]resource.ObservedComposed{pdcNetworkResourceName("primary"): pdcObserved(`{"status":{"atProvider":{"pdcNetworkId":"provider-network-id"}}}`)}},
		{"faro", renderFrontendObservability, frontendObservabilityClaim(), frontendObservabilityConfig(), nil},
		{"ml", renderML, mlClaim(), mlConfig(2), map[resource.Name]resource.ObservedComposed{"holiday-example-holiday": mlObserved(`{"status":{"atProvider":{"id":"holiday-provider-id"}}}`)}},
		{"service-accounts", renderServiceAccounts, serviceAccountsClaim(), serviceAccountsConfig("720h", "24h"), map[resource.Name]resource.ObservedComposed{
			"service-account-ci-runner": observedComposed(`{"status":{"atProvider":{"id":"101"}}}`),
			"service-account-terraform": observedComposed(`{"status":{"atProvider":{"id":"102"}}}`),
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			children, err := tc.render(tc.xr, tc.observed, tc.config)
			if err != nil {
				t.Fatal(err)
			}
			if len(children) == 0 {
				t.Fatal("empty rendering is not provider admission evidence")
			}
			admitProviderChildren(t, e, children)
		})
	}
	t.Run("k6-new-children", func(t *testing.T) {
		xr := k6ProjectDocument(map[string]any{"usage": "development", "allowedLoadZones": []any{}, "loadTests": []any{map[string]any{"name": "smoke", "workload": k6HTTPWorkload("https://example.invalid/health"), "vus": float64(1), "browserVus": float64(0), "durationSeconds": float64(60), "loadZones": []any{}}}, "schedules": []any{map[string]any{"name": "daily", "loadTest": "smoke", "starts": "2030-01-01T00:00:00Z", "recurrenceRule": map[string]any{"frequency": "DAILY", "interval": float64(1), "count": float64(2)}}}})
		observed := k6BootstrapObserved()
		caps, err := renderK6Project(xr, observed, k6Config())
		if err != nil {
			t.Fatal(err)
		}
		for key, value := range k6CurrentPlatformCapObservations(t, caps) {
			observed[key] = value
		}
		observed["load-test-smoke"] = observedComposed(`{"status":{"atProvider":{"id":"501"}}}`)
		children, err := renderK6Project(xr, observed, k6Config())
		if err != nil {
			t.Fatal(err)
		}
		dynamic := map[resource.Name]*resource.DesiredComposed{}
		for _, key := range []resource.Name{"load-test-smoke", "schedule-daily"} {
			if children[key] == nil {
				t.Fatalf("missing %s", key)
			}
			dynamic[key] = children[key]
		}
		admitProviderChildren(t, e, dynamic)
	})
	t.Run("synthetic-check-alerts", func(t *testing.T) {
		observed := syntheticMonitoringVerifiedObserved("67890")
		observed["check-api-home"] = syntheticMonitoringObserved(`{"status":{"atProvider":{"id":"2468"}}}`)
		children, err := renderSyntheticMonitoring(syntheticMonitoringClaim(), observed, syntheticMonitoringPlatformConfig(true))
		if err != nil {
			t.Fatal(err)
		}
		if children["check-alerts-api-home"] == nil {
			t.Fatal("missing CheckAlerts")
		}
		admitProviderChildren(t, e, map[resource.Name]*resource.DesiredComposed{"check-alerts-api-home": children["check-alerts-api-home"]})
	})

}
