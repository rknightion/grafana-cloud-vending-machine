package main

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/google/go-cmp/cmp"
)

func TestAssistantTermsAcceptanceGatesChildrenUntilObservedTrue(t *testing.T) {
	claim := assistantGovernanceDocument(map[string]any{
		"termsAcceptance": map[string]any{"accepted": true},
		"rules": []any{map[string]any{
			"name": "inert-guidance", "profile": "inert-guidance", "scope": "tenant",
		}},
		"mcpServers": []any{map[string]any{
			"name": "inert-tools", "scope": "tenant",
			"configuration": map[string]any{"url": "https://mcp.example.com/"},
		}},
	})
	readyStack := requiredStackResource("teamdemo01", "grafana-vending", "True")

	withoutAcceptance := runAssistantWithInput(t, claim, nil, readyStack)
	assertNoAssistantFatal(t, withoutAcceptance)
	if got, want := assistantResourceKinds(withoutAcceptance), []string{"TermsAcceptance"}; cmp.Diff(want, got) != "" {
		t.Fatalf("unaccepted assistant resources differ (-want +got):\n%s", cmp.Diff(want, got))
	}

	observed := map[string]*fnv1.Resource{
		"terms": observedResource(`{"status":{"atProvider":{"accepted":true}}}`),
	}
	withAcceptance := runAssistantWithInput(t, claim, observed, readyStack)
	assertNoAssistantFatal(t, withAcceptance)
	got := assistantResourceKinds(withAcceptance)
	sort.Strings(got)
	want := []string{"McpServer", "Rule", "TermsAcceptance"}
	sort.Strings(want)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("accepted assistant resources differ (-want +got):\n%s", diff)
	}

	rule := assistantResourceByKind(t, withAcceptance, "Rule")
	if got := nestedMap(t, rule, "spec", "forProvider")["ruleContent"]; got != "Use only the inert example guidance." {
		t.Fatalf("ruleContent = %v, want platform profile content", got)
	}
}

func TestAssistantWithdrawalRemovesChildrenEvenWhileAcceptanceIsObserved(t *testing.T) {
	claim := assistantGovernanceDocument(map[string]any{
		"termsAcceptance": map[string]any{"accepted": false},
		"rules": []any{map[string]any{
			"name": "inert-guidance", "profile": "inert-guidance", "scope": "tenant",
		}},
		"mcpServers": []any{map[string]any{
			"name": "inert-tools", "scope": "tenant",
		}},
	})
	observed := map[string]*fnv1.Resource{
		"terms": observedResource(`{"status":{"atProvider":{"accepted":true}}}`),
		"rule-" + stableResourceSuffix("inert-guidance"):    observedResource(`{"status":{"atProvider":{"id":"rule-id"}}}`),
		"mcp-server-" + stableResourceSuffix("inert-tools"): observedResource(`{"status":{"atProvider":{"id":"mcp-id"}}}`),
	}
	rsp := runAssistantWithInput(t, claim, observed, requiredStackResource("teamdemo01", "grafana-vending", "True"))
	assertNoAssistantFatal(t, rsp)
	if got, want := assistantResourceKinds(rsp), []string{"TermsAcceptance"}; cmp.Diff(want, got) != "" {
		t.Fatalf("withdrawn assistant resources differ (-want +got):\n%s", cmp.Diff(want, got))
	}
	terms := assistantResourceByKind(t, rsp, "TermsAcceptance")
	if got := nestedMap(t, terms, "spec", "forProvider")["accepted"]; got != false {
		t.Fatalf("withdrawn terms accepted = %v, want false", got)
	}
}

func TestAssistantObservedWithdrawalRemovesPreviouslyObservedChildren(t *testing.T) {
	claim := assistantGovernanceDocument(map[string]any{
		"termsAcceptance": map[string]any{"accepted": true},
		"rules": []any{map[string]any{
			"name": "inert-guidance", "profile": "inert-guidance", "scope": "tenant",
		}},
		"mcpServers": []any{map[string]any{
			"name": "inert-tools", "scope": "tenant",
		}},
	})
	observed := map[string]*fnv1.Resource{
		"terms": observedResource(`{"status":{"atProvider":{"accepted":false}}}`),
		"rule-" + stableResourceSuffix("inert-guidance"):    observedResource(`{"status":{"atProvider":{"id":"rule-id"}}}`),
		"mcp-server-" + stableResourceSuffix("inert-tools"): observedResource(`{"status":{"atProvider":{"id":"mcp-id"}}}`),
	}
	rsp := runAssistantWithInput(t, claim, observed, requiredStackResource("teamdemo01", "grafana-vending", "True"))
	assertNoAssistantFatal(t, rsp)
	if got, want := assistantResourceKinds(rsp), []string{"TermsAcceptance"}; cmp.Diff(want, got) != "" {
		t.Fatalf("observed withdrawal resources differ (-want +got):\n%s", cmp.Diff(want, got))
	}
}

func TestAssistantMCPApprovalDefaultsToAlwaysAsk(t *testing.T) {
	claim := assistantGovernanceDocument(map[string]any{
		"termsAcceptance": map[string]any{"accepted": true},
		"mcpServers": []any{map[string]any{
			"name": "inert-tools", "scope": "tenant",
			"configuration": map[string]any{
				"url":                  "https://mcp.example.com/",
				"toolPreferences":      map[string]any{"read": "enabled", "write": "enabled"},
				"toolApprovalPolicies": map[string]any{"read": "auto_approve"},
			},
		}},
	})
	observed := map[string]*fnv1.Resource{
		"terms": observedResource(`{"status":{"atProvider":{"accepted":true}}}`),
	}
	rsp := runAssistantWithInput(t, claim, observed, requiredStackResource("teamdemo01", "grafana-vending", "True"))
	assertNoAssistantFatal(t, rsp)
	mcp := assistantResourceByKind(t, rsp, "McpServer")
	configuration := nestedMap(t, mcp, "spec", "forProvider", "configuration")
	want := map[string]any{"read": "auto_approve", "write": "always_ask"}
	if diff := cmp.Diff(want, configuration["toolApprovalPolicies"]); diff != "" {
		t.Fatalf("tool approval policies differ (-want +got):\n%s", diff)
	}
}

func TestAssistantMCPConfigurationMustBeAnObject(t *testing.T) {
	claim := assistantGovernanceDocument(map[string]any{
		"termsAcceptance": map[string]any{"accepted": true},
		"mcpServers": []any{map[string]any{
			"name": "inert-tools", "scope": "tenant", "configuration": "not-an-object",
		}},
	})
	observed := map[string]*fnv1.Resource{
		"terms": observedResource(`{"status":{"atProvider":{"accepted":true}}}`),
	}
	rsp := runAssistantWithInput(t, claim, observed, requiredStackResource("teamdemo01", "grafana-vending", "True"))
	if fatal := fatalResult(rsp); !strings.Contains(fatal, "mcpServers[0].configuration must be an object") {
		t.Fatalf("malformed MCP configuration fatal result = %q, want object validation error", fatal)
	}
}

func assistantGovernanceDocument(fields map[string]any) string {
	return mustJSON(map[string]any{
		"apiVersion": "platform.example.org/v1beta1",
		"kind":       "GrafanaAssistantGovernance",
		"metadata":   map[string]any{"name": "assistant-governance", "namespace": "grafana-vending"},
		"spec":       mergeAssistantSpec(fields),
	})
}

func mergeAssistantSpec(fields map[string]any) map[string]any {
	spec := map[string]any{"stackRef": map[string]any{"name": "teamdemo01"}}
	for key, value := range fields {
		spec[key] = value
	}
	return spec
}

func assistantResourceKinds(rsp *fnv1.RunFunctionResponse) []string {
	got := make([]string, 0, len(rsp.GetDesired().GetResources()))
	for _, desired := range rsp.GetDesired().GetResources() {
		got = append(got, desired.GetResource().GetFields()["kind"].GetStringValue())
	}
	sort.Strings(got)
	return got
}

func assistantResourceByKind(t *testing.T, rsp *fnv1.RunFunctionResponse, kind string) map[string]any {
	t.Helper()
	var found map[string]any
	for _, desired := range rsp.GetDesired().GetResources() {
		object := desired.GetResource().AsMap()
		if object["kind"] == kind {
			if found != nil {
				t.Fatalf("multiple desired %s resources", kind)
			}
			found = object
		}
	}
	if found == nil {
		t.Fatalf("desired resource kind %q was not rendered", kind)
	}
	return found
}

func assertNoAssistantFatal(t *testing.T, rsp *fnv1.RunFunctionResponse) {
	t.Helper()
	if fatal := fatalResult(rsp); fatal != "" {
		t.Fatalf("RunFunction returned a fatal result: %s", fatal)
	}
}

func runAssistantWithInput(t *testing.T, claim string, observed map[string]*fnv1.Resource, required *fnv1.Resources) *fnv1.RunFunctionResponse {
	t.Helper()
	req := &fnv1.RunFunctionRequest{
		Meta: &fnv1.RequestMeta{Capabilities: requiredResourceCapabilities()},
		Observed: &fnv1.State{
			Composite: &fnv1.Resource{Resource: resource.MustStructJSON(claim)},
			Resources: observed,
		},
		Input:             resource.MustStructJSON(`{"apiVersion":"platform.example.org/v1beta1","kind":"GrafanaAssistantGovernanceConfig","spec":{"ruleProfiles":[{"name":"inert-guidance","ruleContent":"Use only the inert example guidance."}]}}`),
		RequiredResources: map[string]*fnv1.Resources{"referenced-stack": required},
	}
	rsp, err := (&Function{log: logging.NewNopLogger()}).RunFunction(context.Background(), req)
	if err != nil {
		t.Fatalf("RunFunction returned an error: %v", err)
	}
	return rsp
}
