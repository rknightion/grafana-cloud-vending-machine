package main

import (
	"strings"
	"testing"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
)

func TestProvisioningRepositoryRejectsBothDashboardOwners(t *testing.T) {
	claim := provisioningRepositoryDocument(map[string]any{
		"dashboard": map[string]any{"uid": "application-overview"},
	})

	rsp := callProvisioning(t, claim)
	if fatal := fatalResult(rsp); !strings.Contains(fatal, "both repository and Crossplane dashboard ownership") {
		t.Fatalf("mixed dashboard ownership returned fatal result %q", fatal)
	}
}

func TestProvisioningRepositoryRejectsCrossplaneDashboardOwnership(t *testing.T) {
	claim := provisioningRepositoryDocument(map[string]any{
		"repository": nil,
		"dashboard":  map[string]any{"uid": "application-overview"},
	})

	rsp := callProvisioning(t, claim)
	if fatal := fatalResult(rsp); !strings.Contains(fatal, "does not render Crossplane Dashboards") {
		t.Fatalf("Crossplane dashboard ownership returned fatal result %q", fatal)
	}
}

func TestProvisioningRepositoryRendersCredentialReferenceWithoutDashboard(t *testing.T) {
	rsp := runProvisioning(t, provisioningRepositoryDocument(nil))
	resources := rsp.GetDesired().GetResources()
	if len(resources) != 1 {
		t.Fatalf("desired resource count = %d, want 1", len(resources))
	}
	repository := desiredResource(t, rsp, "repository")
	if got := repository["kind"]; got != "RepositoryV0Alpha1" {
		t.Fatalf("repository kind = %v, want RepositoryV0Alpha1", got)
	}
	if got := desiredExternalName(t, repository); got != "application-overview" {
		t.Fatalf("repository external name = %q, want application-overview", got)
	}
	forProvider := nestedMap(t, repository, "spec", "forProvider")
	if got := nestedMap(t, forProvider, "spec", "connection")["name"]; got != "shared-github-app" {
		t.Fatalf("repository connection reference = %v, want shared-github-app", got)
	}
	if _, found := forProvider["secure"]; found {
		t.Fatal("repository must not embed credentials in forProvider.secure")
	}
	if got := nestedMap(t, repository, "spec", "providerConfigRef")["name"]; got != "teamdemo01-provider" {
		t.Fatalf("repository ProviderConfig = %v, want teamdemo01-provider", got)
	}
	for name, desired := range resources {
		if kind := desired.GetResource().GetFields()["kind"].GetStringValue(); kind == "Dashboard" {
			t.Fatalf("repository-provisioned subtree rendered Crossplane Dashboard %q", name)
		}
	}
}

func provisioningRepositoryDocument(additions map[string]any) string {
	spec := map[string]any{
		"stackRef": map[string]any{"name": "teamdemo01"},
		"repository": map[string]any{
			"uid": "application-overview", "title": "Application overview", "description": "Git-backed dashboards for the application overview subtree.",
			"url": "https://github.com/example/grafana-dashboards", "branch": "main", "path": "application-overview",
			"connectionRef": map[string]any{"name": "shared-github-app"},
		},
	}
	for key, value := range additions {
		spec[key] = value
	}
	return mustJSON(map[string]any{
		"apiVersion": "platform.example.org/v1beta1", "kind": "GrafanaProvisioningRepository",
		"metadata": map[string]any{"name": "application-overview", "namespace": "grafana-vending"},
		"spec":     spec,
	})
}

func runProvisioning(t *testing.T, composite string) *fnv1.RunFunctionResponse {
	t.Helper()
	rsp := callProvisioning(t, composite)
	if fatal := fatalResult(rsp); fatal != "" {
		t.Fatalf("RunFunction returned a fatal result: %s", fatal)
	}
	return rsp
}

func callProvisioning(t *testing.T, composite string) *fnv1.RunFunctionResponse {
	return callFunctionWithRequiredResources(t, composite, nil, requiredStackResource("teamdemo01", "grafana-vending", "True"), requiredResourceCapabilities())
}
