package main

import (
	"context"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
	"strings"
	"testing"
)

func TestSurfaceStackContextRejectsInputSpoofAndBindsObservedIdentity(t *testing.T) {
	xr := requestObject("GrafanaFrontendObservability", "faro", map[string]any{"stackRef": map[string]any{"name": "stack001"}}).Object
	config := frontendObservabilityConfig()
	config["referencedStack"] = map[string]any{"stackID": "forged"}
	config["spec"].(map[string]any)["organizations"] = []any{map[string]any{"name": "example-primary", "providerConfigName": "example-provider", "allowedRegions": []any{"prod-us-central-0"}, "allowedUsages": []any{"development"}}}
	req := &fnv1.RunFunctionRequest{}
	rsp := &fnv1.RunFunctionResponse{}
	got, ready, err := observedSurfaceStackConfig(req, rsp, xr, config)
	if err != nil || ready || got["referencedStack"] != nil {
		t.Fatalf("unobserved input trusted: %v %t %v", got, ready, err)
	}
	if rsp.GetRequirements().GetResources()[referencedStackRequirement].GetMatchName() != "stack001" {
		t.Fatal("missing identity-bound selector")
	}
	required := requiredStackResource("stack001", "default", "True")
	obj := required.Items[0].Resource.AsMap()
	status := obj["status"].(map[string]any)
	status["stack"] = map[string]any{"id": "123"}
	obj["spec"] = map[string]any{"usage": "development", "organization": "example-primary", "region": "prod-us-central-0"}
	required.Items[0].Resource = resource.MustStructJSON(mustJSON(obj))
	req.RequiredResources = map[string]*fnv1.Resources{referencedStackRequirement: required}
	got, ready, err = observedSurfaceStackConfig(req, rsp, xr, config)
	if err != nil || !ready {
		t.Fatalf("observed identity refused: ready=%t error=%v", ready, err)
	}
	stack := got["referencedStack"].(map[string]any)
	if stack["stackID"] != "123" || stack["providerConfigName"] != "stack001" || stack["usage"] != "development" {
		t.Fatalf("wrong trusted context: %v", stack)
	}
	obj["metadata"].(map[string]any)["namespace"] = "wrong"
	required.Items[0].Resource = resource.MustStructJSON(mustJSON(obj))
	got, ready, err = observedSurfaceStackConfig(req, rsp, xr, config)
	if err == nil || ready || got["referencedStack"] != nil {
		t.Fatal("cross-namespace identity was trusted")
	}
}

func TestNewStackSurfacesRejectASecondOwnerAndStackMove(t *testing.T) {
	e := &admissionEnv{paths: []string{"../apis/service-accounts-v1beta1.yaml", "../apis/frontend-observability-v1beta1.yaml", "../apis/ml-v1beta1.yaml"}}
	if err := e.Start(t); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := e.Stop(); err != nil {
			t.Error(err)
		}
	})
	ctx := context.Background()
	for _, kind := range []string{"GrafanaServiceAccounts", "GrafanaFrontendObservability", "GrafanaML"} {
		t.Run(kind, func(t *testing.T) {
			spec := map[string]any{"stackRef": map[string]any{"name": "stack001"}, "profile": "standard"}
			if kind == "GrafanaServiceAccounts" {
				spec["profile"] = "ci"
				spec["accounts"] = []any{map[string]any{"name": "runner"}}
			}
			obj := requestObject(kind, "stack001", spec)
			if err := e.Apply(ctx, obj); err != nil {
				t.Fatalf("singleton allowed control: %v", err)
			}
			duplicate := obj.DeepCopy()
			duplicate.SetName("secondowner")
			duplicate.SetResourceVersion("")
			duplicate.SetUID("")
			if err := e.Apply(ctx, duplicate); err == nil || !strings.Contains(err.Error(), "one composite owns this stack surface") {
				t.Fatalf("duplicate owner: %v", err)
			} else {
				t.Logf("API-SERVER second owner refused: %v", err)
			}
			obj.Object["spec"].(map[string]any)["stackRef"] = map[string]any{"name": "different"}
			if err := e.Apply(ctx, obj); err == nil || !strings.Contains(err.Error(), "stackRef is immutable") {
				t.Fatalf("stack move: %v", err)
			} else {
				t.Logf("API-SERVER stack move refused: %v", err)
			}
		})
	}
}

func TestCloudSurfaceUsesObservedOrganizationCredentialReference(t *testing.T) {
	xr := requestObject("GrafanaCloudIntegrations", "stack001", map[string]any{"stackRef": map[string]any{"name": "stack001"}}).Object
	config := map[string]any{"spec": map[string]any{"organizations": []any{map[string]any{"name": "example-primary", "providerConfigName": "example-provider", "allowedRegions": []any{"prod-us-central-0"}, "allowedUsages": []any{"development"}}}}}
	required := requiredStackResource("stack001", "default", "True")
	obj := required.Items[0].Resource.AsMap()
	obj["spec"] = map[string]any{"usage": "development", "organization": "example-primary", "region": "prod-us-central-0"}
	obj["status"].(map[string]any)["stack"] = map[string]any{"id": "123"}
	required.Items[0].Resource = resource.MustStructJSON(mustJSON(obj))
	req := &fnv1.RunFunctionRequest{RequiredResources: map[string]*fnv1.Resources{referencedStackRequirement: required}}
	if _, ready, err := observedSurfaceStackConfig(req, &fnv1.RunFunctionResponse{}, xr, config); err != nil || ready {
		t.Fatalf("missing provider: %t %v", ready, err)
	}
	provider := map[string]any{"apiVersion": "grafana.m.crossplane.io/v1beta1", "kind": "ProviderConfig", "metadata": map[string]any{"name": "example-provider", "namespace": "default"}, "spec": map[string]any{"credentials": map[string]any{"source": "Secret", "secretRef": map[string]any{"name": "custom-org-credentials", "namespace": "default", "key": "json"}}}}
	req.RequiredResources["surface-organization-provider"] = &fnv1.Resources{Items: []*fnv1.Resource{{Resource: resource.MustStructJSON(mustJSON(provider))}}}
	if _, ready, err := observedSurfaceStackConfig(req, &fnv1.RunFunctionResponse{}, xr, config); err != nil || ready {
		t.Fatalf("missing managed Stack endpoint trusted: %t %v", ready, err)
	}
	managed := map[string]any{"apiVersion": "cloud.grafana.m.crossplane.io/v1alpha1", "kind": "Stack", "metadata": map[string]any{"name": "stack001", "namespace": "default", "annotations": map[string]any{"crossplane.io/external-name": "stack001"}}, "status": map[string]any{"conditions": []any{map[string]any{"type": "Ready", "status": "True"}}, "atProvider": map[string]any{"id": "123", "cloudProviderUrl": "https://observed-cloud-provider.example.invalid"}}}
	req.RequiredResources["surface-managed-stack"] = &fnv1.Resources{Items: []*fnv1.Resource{{Resource: resource.MustStructJSON(mustJSON(managed))}}}
	got, ready, err := observedSurfaceStackConfig(req, &fnv1.RunFunctionResponse{}, xr, config)
	if err != nil || !ready {
		t.Fatalf("observed provider: %t %v", ready, err)
	}
	if got["referencedStack"].(map[string]any)["cloudProviderURL"] != "https://observed-cloud-provider.example.invalid" {
		t.Fatal("managed endpoint not retained")
	}
	ref := got["referencedStack"].(map[string]any)["organizationCredentialSecretRef"].(map[string]any)
	if ref["name"] != "custom-org-credentials" || ref["key"] != "json" {
		t.Fatalf("invented credential ref: %v", ref)
	}
	delete(managed["status"].(map[string]any)["atProvider"].(map[string]any), "cloudProviderUrl")
	req.RequiredResources["surface-managed-stack"].Items[0].Resource = resource.MustStructJSON(mustJSON(managed))
	if _, ready, err := observedSurfaceStackConfig(req, &fnv1.RunFunctionResponse{}, xr, config); err != nil || ready {
		t.Fatalf("missing endpoint must wait: ready=%t error=%v", ready, err)
	}
	provider["metadata"].(map[string]any)["namespace"] = "elsewhere"
	req.RequiredResources["surface-organization-provider"].Items[0].Resource = resource.MustStructJSON(mustJSON(provider))
	if _, ready, err := observedSurfaceStackConfig(req, &fnv1.RunFunctionResponse{}, xr, config); err == nil || ready {
		t.Fatal("foreign provider accepted")
	}
}
