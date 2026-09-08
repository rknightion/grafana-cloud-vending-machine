package main

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
)

func TestTokenSecurityRejectsMissingLifetimeCeiling(t *testing.T) {
	desired := map[resource.Name]*resource.DesiredComposed{}
	err := addTelemetryAccess(
		desired, nil, "grafana-vending", "teamdemo01", "prod-us-central-0",
		"/platform/grafana-cloud/stacks/example-primary/production/teamdemo01/telemetry-publisher",
		map[string]any{"profile": "standard"}, platformSettings{}, "grafana-cloud-org-example-primary", false,
	)
	if err == nil || !strings.Contains(err.Error(), "maximumTokenLifetime") {
		t.Fatalf("missing maximumTokenLifetime error = %v, want a fail-closed configuration error", err)
	}
}

func TestTokenSecurityCapsRequestedLifetime(t *testing.T) {
	desired := map[resource.Name]*resource.DesiredComposed{}
	observed := map[resource.Name]resource.ObservedComposed{
		"telemetry-access-policy": accessObserved(`{"status":{"atProvider":{"policyId":"telemetry-policy-12345"}}}`),
	}
	err := addTelemetryAccess(
		desired, observed, "grafana-vending", "teamdemo01", "prod-us-central-0",
		"/platform/grafana-cloud/stacks/example-primary/production/teamdemo01/telemetry-publisher",
		map[string]any{"profile": "standard", "maximumTokenLifetime": "8760h"},
		platformSettings{maximumTokenLifetime: "336h"}, "grafana-cloud-org-example-primary", false,
	)
	if err != nil {
		t.Fatalf("addTelemetryAccess returned an error: %v", err)
	}
	forProvider := nestedMap(t, desired["telemetry-token"].Resource.UnstructuredContent(), "spec", "forProvider")
	if got, want := forProvider["expireAfter"], "336h"; got != want {
		t.Fatalf("telemetry token expireAfter = %v, want %s", got, want)
	}
}

func TestTokenSecurityOmitsConditionsWithoutSelectedProfile(t *testing.T) {
	desired := map[resource.Name]*resource.DesiredComposed{}
	err := addTelemetryAccess(
		desired,
		nil,
		"grafana-vending",
		"teamdemo01",
		"prod-us-central-0",
		"/platform/grafana-cloud/stacks/example-primary/production/teamdemo01/telemetry-publisher",
		map[string]any{"profile": "isolated"},
		platformSettings{
			maximumTokenLifetime: "720h",
			tokenUseNetworkProfiles: []any{map[string]any{
				"name":           "restricted",
				"allowedSubnets": []any{"192.0.2.0/24"},
			}},
		},
		"grafana-cloud-org-example-primary",
		false,
	)
	if err != nil {
		t.Fatalf("addTelemetryAccess returned an error: %v", err)
	}
	forProvider := nestedMap(t, desired["telemetry-access-policy"].Resource.UnstructuredContent(), "spec", "forProvider")
	if _, exists := forProvider["conditions"]; exists {
		t.Fatalf("unselected token-use network profile emitted conditions: %s", mustJSON(forProvider["conditions"]))
	}
}

func TestTokenSecurityUsesPlatformProfileForAllowedSubnets(t *testing.T) {
	desired := map[resource.Name]*resource.DesiredComposed{}
	err := addTelemetryAccess(
		desired,
		nil,
		"grafana-vending",
		"teamdemo01",
		"prod-us-central-0",
		"/platform/grafana-cloud/stacks/example-primary/production/teamdemo01/telemetry-publisher",
		map[string]any{
			"profile":                "restricted",
			"allowedTokenUseSubnets": []any{"198.51.100.0/24"},
		},
		platformSettings{
			maximumTokenLifetime: "720h",
			tokenUseNetworkProfiles: []any{map[string]any{
				"name":           "restricted",
				"allowedSubnets": []any{"192.0.2.0/24", "2001:db8::/32"},
			}},
		},
		"grafana-cloud-org-example-primary",
		false,
	)
	if err != nil {
		t.Fatalf("addTelemetryAccess returned an error: %v", err)
	}
	forProvider := nestedMap(t, desired["telemetry-access-policy"].Resource.UnstructuredContent(), "spec", "forProvider")
	want := []any{map[string]any{"allowedSubnets": []any{"192.0.2.0/24", "2001:db8::/32"}}}
	if diff := cmp.Diff(want, forProvider["conditions"]); diff != "" {
		t.Fatalf("token-use network conditions differ (-want +got):\n%s", diff)
	}
}

func TestTokenSecurityPublishesObservedExpiries(t *testing.T) {
	observed := map[resource.Name]resource.ObservedComposed{
		"stack-token":            accessObserved(`{"status":{"atProvider":{"expiration":"2030-01-02T03:04:05Z"}}}`),
		"fleet-management-token": accessObserved(`{"status":{"atProvider":{"expiresAt":"2030-02-03T04:05:06Z"}}}`),
		"telemetry-token":        accessObserved(`{"status":{"atProvider":{"expiresAt":"2030-03-04T05:06:07Z"}}}`),
	}
	status := map[string]any{}
	publishTokenExpiryStatus(status, observed)

	want := map[string]any{
		"administrator":      "2030-01-02T03:04:05Z",
		"fleetManagement":    "2030-02-03T04:05:06Z",
		"telemetryPublisher": "2030-03-04T05:06:07Z",
	}
	if diff := cmp.Diff(want, status["tokenExpiries"]); diff != "" {
		t.Fatalf("token expiry status differs (-want +got):\n%s", diff)
	}
}

func accessObserved(document string) resource.ObservedComposed {
	value := composed.New()
	value.SetUnstructuredContent(resource.MustStructJSON(document).AsMap())
	return resource.ObservedComposed{Resource: value}
}

func TestTokenSecurityRejectsExplicitEmptySubnetProfile(t *testing.T) {
	_, err := selectedTokenUseAllowedSubnets([]any{map[string]any{"name": "restricted", "allowedSubnets": []any{}}}, "restricted")
	if err == nil {
		t.Fatal("explicit empty subnet restriction became unrestricted token use")
	}
}
