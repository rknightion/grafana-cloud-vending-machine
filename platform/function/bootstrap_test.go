package main

import (
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
	"testing"
)

func TestBootstrapContextRequiresIdentityAndSecret(t *testing.T) {
	xr := map[string]any{"apiVersion": "platform.example.org/v1beta1", "kind": "GrafanaK6Project", "metadata": map[string]any{"name": "load", "namespace": "grafana-vending"}, "spec": map[string]any{"stackRef": map[string]any{"name": "teamdemo01"}}}
	config := map[string]any{"spec": map[string]any{"organizations": []any{map[string]any{"name": "example-primary", "providerConfigName": "example-provider", "allowedRegions": []any{"prod-us-central-0"}, "allowedUsages": []any{"development"}}}}}
	req := &fnv1.RunFunctionRequest{RequiredResources: map[string]*fnv1.Resources{}}
	rsp := &fnv1.RunFunctionResponse{}
	if got := serviceBootstrapConfig(req, rsp, xr, config); got["_resolvedK6Stack"] != nil {
		t.Fatal("admitted absent stack")
	}
	stack := `{"apiVersion":"platform.example.org/v1beta1","kind":"GrafanaCloudStackRequest","metadata":{"name":"teamdemo01","namespace":"grafana-vending","uid":"stack-uid"},"spec":{"usage":"development","organization":"example-primary","region":"prod-us-central-0"},"status":{"stack":{"id":"123"},"outputSecretPath":"/example/stack","conditions":[{"type":"Ready","status":"True"}]}}`
	req.RequiredResources[referencedStackRequirement] = &fnv1.Resources{Items: []*fnv1.Resource{{Resource: resource.MustStructJSON(stack)}}}
	got := serviceBootstrapConfig(req, rsp, xr, config)
	if got["_resolvedK6Stack"] != nil {
		t.Fatal("admitted absent bootstrap secret")
	}
	if rsp.GetRequirements().GetResources()["service-bootstrap-token"].GetMatchName() != "teamdemo01-token" {
		t.Fatal("wrong bootstrap selector")
	}
	req.RequiredResources["service-bootstrap-token"] = &fnv1.Resources{Items: []*fnv1.Resource{{Resource: resource.MustStructJSON(`{"apiVersion":"v1","kind":"Secret","metadata":{"name":"other","namespace":"grafana-vending"},"data":{"attribute.key":"cGxhY2Vob2xkZXI="}}`)}}}
	if got := serviceBootstrapConfig(req, rsp, xr, config); got["_resolvedK6Stack"] != nil {
		t.Fatal("admitted wrong secret")
	}
	req.RequiredResources["service-bootstrap-token"].Items[0].Resource = resource.MustStructJSON(`{"apiVersion":"v1","kind":"Secret","metadata":{"name":"teamdemo01-token","namespace":"grafana-vending"},"data":{"attribute.key":"cGxhY2Vob2xkZXI="}}`)
	got = serviceBootstrapConfig(req, rsp, xr, config)
	ctx, ok := got["_resolvedK6Stack"].(map[string]any)
	if !ok || ctx["usage"] != "development" || ctx["stackId"] != "123" {
		t.Fatalf("context=%v", got)
	}
	ref := ctx["serviceAccountTokenSecret"].(map[string]any)
	if len(ref) != 3 || ref["key"] != "attribute.key" || ref["ready"] != true {
		t.Fatalf("secret ref=%v", ref)
	}
}

func TestBootstrapInterruptionDoesNotWithdrawExistingChildren(t *testing.T) {
	claim := `{"apiVersion":"platform.example.org/v1beta1","kind":"GrafanaK6Project","metadata":{"name":"load","namespace":"grafana-vending"},"spec":{"stackRef":{"name":"teamdemo01"},"grafanaUser":"example-user"}}`
	prior := compositeRenderers["GrafanaK6Project"]
	defer func() { compositeRenderers["GrafanaK6Project"] = prior }()
	current := prior
	current.implemented = true
	compositeRenderers["GrafanaK6Project"] = current
	rsp := callFunctionWithRequiredResources(t, claim, map[string]*fnv1.Resource{"installation": observedResource(`{"apiVersion":"k6.grafana.m.crossplane.io/v1alpha1","kind":"Installation","metadata":{"name":"load-installation","namespace":"grafana-vending"},"spec":{"forProvider":{"stackId":"123"}}}`)}, requiredStackResource("teamdemo01", "grafana-vending", "True"), requiredResourceCapabilities())
	if fatalResult(rsp) == "" {
		t.Fatal("missing trusted bootstrap context must abort reconciliation before withdrawing observed children")
	}
	rsp = callFunctionWithRequiredResources(t, claim, nil, requiredStackResource("teamdemo01", "grafana-vending", "True"), requiredResourceCapabilities())
	if rsp.GetDesired().GetComposite().GetReady() != fnv1.Ready_READY_FALSE {
		t.Fatal("missing bootstrap marked new product ready")
	}
}

func TestProductDependencyInterruptionPreservesChildren(t *testing.T) {
	observed := map[resource.Name]resource.ObservedComposed{"project": {}}
	if err := productWithdrawalError("GrafanaK6Project", nil, nil, observed); err == nil {
		t.Fatal("withdrew existing project")
	}
	xr := map[string]any{"spec": map[string]any{"checks": []any{map[string]any{"name": "home"}}}}
	observed = map[resource.Name]resource.ObservedComposed{"check-home": {}}
	if err := productWithdrawalError("GrafanaSyntheticMonitoring", xr, nil, observed); err == nil {
		t.Fatal("withdrew configured check during verification outage")
	}
	xr["spec"] = map[string]any{"checks": []any{}}
	if err := productWithdrawalError("GrafanaSyntheticMonitoring", xr, nil, observed); err != nil {
		t.Fatal("blocked intentional check removal", err)
	}
}

func TestSMDesiredSetRemovesWithdrawnChecks(t *testing.T) {
	rsp := &fnv1.RunFunctionResponse{Desired: &fnv1.State{Resources: map[string]*fnv1.Resource{"check-old": {}, "synthetic-monitoring-verifier-old": {}, "unrelated": {}}}}
	desired := map[resource.Name]*resource.DesiredComposed{"check-new": {}}
	pruneSMDesired(rsp, desired)
	if _, exists := rsp.Desired.Resources["check-old"]; exists {
		t.Fatal("removed check survived merge")
	}
	if _, exists := rsp.Desired.Resources["synthetic-monitoring-verifier-old"]; exists {
		t.Fatal("obsolete verifier survived merge")
	}
	if _, exists := rsp.Desired.Resources["unrelated"]; !exists {
		t.Fatal("removed unrelated desired resource")
	}
}
