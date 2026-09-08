package main

import (
	"strings"
	"testing"

	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
)

func TestFleetPipelinesSelectOnlyPlatformOwnedProfiles(t *testing.T) {
	xr := map[string]any{
		"metadata": map[string]any{"name": "example-fleet", "namespace": "grafana-vending"},
		"spec":     map[string]any{"stackRef": map[string]any{"name": "teamdemo01"}, "profile": "standard"},
	}
	desired, err := renderFleetPipelines(xr, nil, fleetConfig())
	if err != nil {
		t.Fatalf("render fleet pipelines: %v", err)
	}
	pipeline := desired["attribution-pipeline"].Resource.UnstructuredContent()
	parameters := nestedMap(t, pipeline, "spec", "forProvider")
	if got := parameters["matchers"]; len(got.([]any)) != 1 {
		t.Fatalf("standard profile matchers = %v, want one platform-owned matcher", got)
	}
	contents, _ := parameters["contents"].(string)
	for _, label := range []string{"team", "cost-centre", "environment"} {
		if !strings.Contains(contents, `key = "`+label+`"`) {
			t.Errorf("attribution pipeline does not enforce %q: %s", label, contents)
		}
	}
	if got := desiredExternalName(t, pipeline); got != "example-fleet-attribution" {
		t.Fatalf("pipeline external name = %q, want deterministic request-derived name", got)
	}

	xr["spec"].(map[string]any)["profile"] = "request-authored"
	if _, err := renderFleetPipelines(xr, nil, fleetConfig()); err == nil || !strings.Contains(err.Error(), "not configured by the platform") {
		t.Fatalf("unconfigured profile error = %v, want platform configuration rejection", err)
	}
}

func TestFleetCredentialWaitsForObservedIDsAndNeverEmbedsToken(t *testing.T) {
	desired := map[resource.Name]*resource.DesiredComposed{}
	if err := addFleetAccess(desired, nil, "grafana-vending", "teamdemo01", "prod-us-central-0", "/platform/grafana-cloud/stacks/example-primary/production/teamdemo01", "standard", fleetTestSettings(), "organization-provider", false); err != nil {
		t.Fatal(err)
	}
	if _, ok := desired["fleet-management-token"]; ok {
		t.Fatal("fleet token rendered before the access-policy ID was observed")
	}

	observed := map[resource.Name]resource.ObservedComposed{
		"fleet-management-access-policy": fleetObserved(`{"status":{"atProvider":{"policyId":"policy-12345"}}}`),
	}
	if err := addFleetAccess(desired, observed, "grafana-vending", "teamdemo01", "prod-us-central-0", "/platform/grafana-cloud/stacks/example-primary/production/teamdemo01", "standard", fleetTestSettings(), "organization-provider", false); err != nil {
		t.Fatal(err)
	}
	if _, ok := desired["fleet-management-token"]; !ok {
		t.Fatal("fleet token was not rendered after the access-policy ID was observed")
	}
	if _, ok := desired["fleet-management-credentials"]; ok {
		t.Fatal("fleet credential publication rendered before the stack ID was observed")
	}

	observed["stack"] = fleetObserved(`{"status":{"atProvider":{"id":"stack-12345"}}}`)
	if err := addFleetAccess(desired, observed, "grafana-vending", "teamdemo01", "prod-us-central-0", "/platform/grafana-cloud/stacks/example-primary/production/teamdemo01", "standard", fleetTestSettings(), "organization-provider", false); err != nil {
		t.Fatal(err)
	}
	publication := desired["fleet-management-credentials"].Resource.UnstructuredContent()
	document := nestedMap(t, publication, "spec", "template", "data")["fleet-management.json"].(string)
	if !strings.Contains(document, `index . "attribute.token"`) || !strings.Contains(document, `fleet_management_auth`) {
		t.Fatalf("fleet credential document does not derive the credential from the token Secret: %s", document)
	}
	if strings.Contains(document, "credential-value") {
		t.Fatalf("fleet credential document embeds a credential: %s", document)
	}
}

func fleetConfig() map[string]any {
	return map[string]any{"spec": map[string]any{"fleetPipelineProfiles": []any{map[string]any{
		"name": "standard", "matchers": []any{`{environment=~".+"}`},
		"labels": map[string]any{"team": "platform-team", "cost-centre": "platform-cost-centre", "environment": "platform-environment"},
	}}}}
}

func fleetTestSettings() platformSettings {
	return platformSettings{maximumTokenLifetime: "720h", secretStoreName: "grafana-vending-secrets", secretStoreKind: "SecretStore"}
}

func fleetObserved(document string) resource.ObservedComposed {
	value := composed.New()
	value.SetUnstructuredContent(observedResource(document).GetResource().AsMap())
	return resource.ObservedComposed{Resource: value}
}
