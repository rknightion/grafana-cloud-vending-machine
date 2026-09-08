package main

import (
	"strings"
	"testing"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
)

func TestAlertingBundleRequiresExplicitProvenance(t *testing.T) {
	claim := alertingBundleDocument(map[string]any{"provenance": ""})
	rsp := callFunctionWithRequiredResources(t, claim, nil, requiredStackResource("teamdemo01", "grafana-vending", "True"), requiredResourceCapabilities())
	if fatal := fatalResult(rsp); !strings.Contains(fatal, "provenance") {
		t.Fatalf("missing provenance fatal result = %q", fatal)
	}
}

func TestAlertingBundleProvenanceSelectsOwnership(t *testing.T) {
	for _, tc := range []struct {
		provenance string
		owner      string
		value      bool
	}{
		{provenance: "enforced", owner: "forProvider", value: false},
		{provenance: "createOnly", owner: "initProvider", value: true},
	} {
		t.Run(tc.provenance, func(t *testing.T) {
			rsp := runAlertingBundle(t, alertingBundleDocument(map[string]any{"provenance": tc.provenance}))
			spec := nestedMap(t, desiredResource(t, rsp, "contact-point-operations"), "spec")
			parameters := nestedMap(t, spec, tc.owner)
			if got, ok := parameters["disableProvenance"].(bool); !ok || got != tc.value {
				t.Fatalf("%s disableProvenance = %v, want %v", tc.owner, parameters["disableProvenance"], tc.value)
			}
			if tc.provenance == "createOnly" {
				forProvider, ok := spec["forProvider"].(map[string]any)
				if !ok || len(forProvider) != 0 {
					t.Fatalf("createOnly forProvider = %#v, want an owned empty map", spec["forProvider"])
				}
			} else if _, ok := spec["initProvider"]; ok {
				t.Fatal("enforced mode also rendered initProvider")
			}
		})
	}
}

func TestAlertingBundleUsesSimplifiedRoutingWithoutPolicy(t *testing.T) {
	rsp := runAlertingBundle(t, alertingBundleDocument(nil))
	resources := rsp.GetDesired().GetResources()
	for name, resource := range resources {
		if resource.GetResource().AsMap()["kind"] == "NotificationPolicy" || resource.GetResource().AsMap()["kind"] == "RoutingtreeV1Beta1" {
			t.Fatalf("unexpected whole-tree routing resource %q", name)
		}
	}
	rule := nestedMap(t, desiredResource(t, rsp, "rule-group-primary"), "spec", "forProvider")["rule"].([]any)[0].(map[string]any)
	routing := rule["notificationSettings"].([]any)[0].(map[string]any)
	if got, want := routing["contactPoint"], "example-alerts-operations"; got != want {
		t.Fatalf("simplified routing contact point = %v, want %s", got, want)
	}
}

func TestAlertingBundleHasOneOwnerPerWholeSetObject(t *testing.T) {
	rsp := runAlertingBundle(t, alertingBundleDocument(nil))
	seen := map[string]string{}
	for logicalName, resource := range rsp.GetDesired().GetResources() {
		object := resource.GetResource().AsMap()
		kind, _ := object["kind"].(string)
		if !oneOf(kind, "RuleGroup", "ContactPoint", "MuteTiming", "MessageTemplate", "InhibitionruleV1Beta1") {
			continue
		}
		externalName := desiredExternalName(t, object)
		if owner, exists := seen[kind+":"+externalName]; exists {
			t.Fatalf("%s and %s both own %s %q", owner, logicalName, kind, externalName)
		}
		seen[kind+":"+externalName] = logicalName
	}
	if len(seen) != 5 {
		t.Fatalf("whole-set object owners = %d, want 5", len(seen))
	}
}

func TestAlertingBundleRejectsDuplicateOwnedNames(t *testing.T) {
	tests := map[string]map[string]any{
		"message template": {"messageTemplates": []any{
			map[string]any{"name": "duplicate", "template": "first"},
			map[string]any{"name": "duplicate", "template": "second"},
		}},
		"inhibition rule": {"inhibitionRules": []any{
			map[string]any{"uid": "duplicate"},
			map[string]any{"uid": "duplicate"},
		}},
		"rule group": {"ruleGroups": []any{
			map[string]any{"name": "duplicate", "folderUid": "alerts", "rules": []any{alertingTestRule()}},
			map[string]any{"name": "duplicate", "folderUid": "alerts", "rules": []any{alertingTestRule()}},
		}},
	}
	for name, addition := range tests {
		t.Run(name, func(t *testing.T) {
			claim := alertingBundleDocument(addition)
			rsp := callFunctionWithRequiredResources(t, claim, nil, requiredStackResource("teamdemo01", "grafana-vending", "True"), requiredResourceCapabilities())
			if fatal := fatalResult(rsp); !strings.Contains(fatal, "duplicate") {
				t.Fatalf("duplicate owner returned fatal result %q", fatal)
			}
		})
	}
}

func alertingTestRule() map[string]any {
	return map[string]any{
		"name": "Alerting bundle health", "uid": "alerting-bundle-health", "condition": "B",
		"data":                 []any{map[string]any{"refId": "A"}},
		"notificationSettings": map[string]any{"contactPoint": "operations"},
	}
}

func runAlertingBundle(t *testing.T, claim string) *fnv1.RunFunctionResponse {
	t.Helper()
	rsp := callFunctionWithRequiredResources(t, claim, nil, requiredStackResource("teamdemo01", "grafana-vending", "True"), requiredResourceCapabilities())
	if fatal := fatalResult(rsp); fatal != "" {
		t.Fatalf("RunFunction returned a fatal result: %s", fatal)
	}
	return rsp
}

func alertingBundleDocument(additions map[string]any) string {
	spec := map[string]any{
		"stackRef":   map[string]any{"name": "teamdemo01"},
		"provenance": "enforced",
		"contactPoints": []any{map[string]any{
			"name":  "operations",
			"email": map[string]any{"addresses": []any{"alerts@example.invalid"}},
		}},
		"muteTimings": []any{map[string]any{
			"name":      "maintenance",
			"intervals": []any{map[string]any{"weekdays": []any{"saturday", "sunday"}}},
		}},
		"messageTemplates": []any{map[string]any{
			"name": "default-message", "template": "{{ define \"bundle.message\" }}example{{ end }}",
		}},
		"inhibitionRules": []any{map[string]any{
			"uid":            "suppress-warning-while-critical",
			"sourceMatchers": []any{map[string]any{"label": "severity", "type": "=", "value": "critical"}},
			"targetMatchers": []any{map[string]any{"label": "severity", "type": "=", "value": "warning"}},
		}},
		"ruleGroups": []any{map[string]any{
			"name": "primary", "folderUid": "vending-alerts", "intervalSeconds": 60,
			"rules": []any{map[string]any{
				"name": "Alerting bundle health", "uid": "alerting-bundle-health", "condition": "B",
				"data":        []any{map[string]any{"refId": "A", "queryType": "", "datasourceUid": "-100", "model": "{\"expression\":\"1\",\"refId\":\"A\",\"type\":\"math\"}", "relativeTimeRange": []any{map[string]any{"from": 0, "to": 0}}}},
				"noDataState": "NoData", "execErrState": "Alerting", "labels": map[string]any{"severity": "warning"},
				"notificationSettings": map[string]any{"contactPoint": "operations", "groupBy": []any{"alertname", "grafana_folder"}, "groupWait": "30s", "muteTimings": []any{"maintenance"}},
			}},
		}},
	}
	for key, value := range additions {
		spec[key] = value
	}
	return mustJSON(map[string]any{
		"apiVersion": "platform.example.org/v1beta1", "kind": "GrafanaAlertingBundle",
		"metadata": map[string]any{"name": "example-alerts", "namespace": "grafana-vending"}, "spec": spec,
	})
}
