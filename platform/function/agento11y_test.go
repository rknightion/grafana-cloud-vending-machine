package main

import (
	"sort"
	"testing"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
)

func TestAgentObservabilityIsOptInAndSeparatesOwnership(t *testing.T) {
	stack := requiredStackResource("replacewithunique03", "example-platform", "True")

	withoutModule := runAgentObservability(t, map[string]any{
		"stackRef": map[string]any{"name": "replacewithunique03"},
	}, nil, stack)
	if got := len(withoutModule.GetDesired().GetResources()); got != 0 {
		t.Fatalf("stack reference alone rendered %d resources, want none", got)
	}

	withModule := runAgentObservability(t, map[string]any{
		"stackRef": map[string]any{"name": "replacewithunique03"},
		"guards": map[string]any{
			"hookRules": []any{map[string]any{
				"ruleId":       "block-example-tools",
				"phase":        "preflight",
				"actionOnFail": "deny",
				"blockedTools": []any{"delete_*"},
			}},
			"ruleActions": []any{map[string]any{
				"name":           "save-failed-evaluations",
				"ruleId":         "score-example-turns",
				"condition":      "all_evaluators_fail",
				"collectionRefs": []any{map[string]any{"name": "failed-evaluations"}},
			}},
		},
		"workload": map[string]any{
			"collections": []any{map[string]any{
				"name":        "failed-evaluations",
				"description": "Example collection; membership is workload-owned.",
			}},
			"evaluators": []any{map[string]any{
				"evaluatorId": "example-quality",
				"kind":        "regex",
				"config":      `{"pattern":"example"}`,
				"outputKeys":  `[{"key":"matched","type":"bool"}]`,
				"version":     "1",
			}},
			"evaluationRules": []any{map[string]any{
				"ruleId":       "score-example-turns",
				"evaluatorIds": []any{"example-quality"},
				"sampleRate":   0.1,
				"selector":     "user_visible_turn",
			}},
		},
	}, map[string]*fnv1.Resource{
		"workload-collection-" + stableResourceSuffix("failed-evaluations"): observedResource(`{"status":{"atProvider":{"id":"collection-example-id"}}}`),
	}, stack)

	wantKinds := map[string]string{
		"guard-hook-" + stableResourceSuffix("block-example-tools"):               "HookRule",
		"guard-action-save-failed-evaluations":                                    "RuleAction",
		"workload-collection-" + stableResourceSuffix("failed-evaluations"):       "Collection",
		"workload-evaluator-" + stableResourceSuffix("example-quality"):           "Evaluator",
		"workload-evaluation-rule-" + stableResourceSuffix("score-example-turns"): "EvaluationRule",
	}
	gotKinds := map[string]string{}
	for name, desired := range withModule.GetDesired().GetResources() {
		gotKinds[name] = desired.GetResource().GetFields()["kind"].GetStringValue()
	}
	if len(gotKinds) != len(wantKinds) {
		t.Fatalf("rendered resources = %v, want exactly %v", sortedResourceKinds(gotKinds), sortedResourceKinds(wantKinds))
	}
	for name, wantKind := range wantKinds {
		if got := gotKinds[name]; got != wantKind {
			t.Errorf("resource %q kind = %q, want %q", name, got, wantKind)
		}
	}

	for name, desired := range withModule.GetDesired().GetResources() {
		providerConfig := nestedMap(t, desired.GetResource().AsMap(), "spec", "providerConfigRef")
		if got, want := providerConfig["name"], "replacewithunique03"; got != want {
			t.Errorf("resource %q ProviderConfig = %v, want %v", name, got, want)
		}
	}
	if got := desiredExternalName(t, desiredResource(t, withModule, "guard-hook-"+stableResourceSuffix("block-example-tools"))); got != "block-example-tools" {
		t.Errorf("HookRule external name = %q, want rule ID", got)
	}
	if got := desiredExternalName(t, desiredResource(t, withModule, "workload-evaluator-"+stableResourceSuffix("example-quality"))); got != "example-quality" {
		t.Errorf("Evaluator external name = %q, want evaluator ID", got)
	}
	if got := desiredExternalName(t, desiredResource(t, withModule, "workload-evaluation-rule-"+stableResourceSuffix("score-example-turns"))); got != "score-example-turns" {
		t.Errorf("EvaluationRule external name = %q, want rule ID", got)
	}
}

func TestAgentObservabilityWaitsForProviderAssignedCollectionID(t *testing.T) {
	spec := map[string]any{
		"stackRef": map[string]any{"name": "replacewithunique03"},
		"guards": map[string]any{
			"ruleActions": []any{map[string]any{
				"name":           "save-failed-evaluations",
				"ruleId":         "score-example-turns",
				"condition":      "all_evaluators_fail",
				"collectionRefs": []any{map[string]any{"name": "failed-evaluations"}},
			}},
		},
		"workload": map[string]any{
			"collections": []any{map[string]any{"name": "failed-evaluations"}},
		},
	}
	stack := requiredStackResource("replacewithunique03", "example-platform", "True")

	first := runAgentObservability(t, spec, nil, stack)
	if _, ok := first.GetDesired().GetResources()["guard-action-save-failed-evaluations"]; ok {
		t.Fatal("RuleAction rendered before its collection ID was observed")
	}
	if _, ok := first.GetDesired().GetResources()["workload-collection-"+stableResourceSuffix("failed-evaluations")]; !ok {
		t.Fatal("Collection was not rendered when declared")
	}

	second := runAgentObservability(t, spec, map[string]*fnv1.Resource{
		"workload-collection-" + stableResourceSuffix("failed-evaluations"): observedResource(`{"status":{"atProvider":{"id":"collection-example-id"}}}`),
	}, stack)
	action := desiredResource(t, second, "guard-action-save-failed-evaluations")
	parameters := nestedMap(t, action, "spec", "forProvider")
	if got, want := parameters["collectionIds"], []any{"collection-example-id"}; gotValueDiff(got, want) {
		t.Errorf("RuleAction collection IDs = %v, want %v", got, want)
	}
}

func runAgentObservability(t *testing.T, spec map[string]any, observed map[string]*fnv1.Resource, required *fnv1.Resources) *fnv1.RunFunctionResponse {
	t.Helper()
	return callFunctionWithRequiredResources(t, agentObservabilityDocument(spec), observed, required, requiredResourceCapabilities())
}

func agentObservabilityDocument(spec map[string]any) string {
	return mustJSON(map[string]any{
		"apiVersion": "platform.example.org/v1beta1",
		"kind":       "GrafanaAgentObservability",
		"metadata": map[string]any{
			"name":      "example-agent-observability",
			"namespace": "example-platform",
		},
		"spec": spec,
	})
}

func sortedResourceKinds(resources map[string]string) []string {
	values := make([]string, 0, len(resources))
	for name, kind := range resources {
		values = append(values, name+"="+kind)
	}
	sort.Strings(values)
	return values
}

func gotValueDiff(got, want any) bool {
	return mustJSON(got) != mustJSON(want)
}
