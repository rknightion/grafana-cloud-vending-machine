package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestAlertingRoutingRendersOneAuthoritativePolicyAndOrdinaryRules(t *testing.T) {
	rsp := runAlertingRouting(t, alertingRoutingDocument(nil))
	policyCount := 0
	for logicalName, resource := range rsp.GetDesired().GetResources() {
		kind, _ := resource.GetResource().AsMap()["kind"].(string)
		if kind == "NotificationPolicy" {
			policyCount++
			if logicalName != "notification-policy" {
				t.Fatalf("unexpected NotificationPolicy writer %q", logicalName)
			}
		}
		if kind == "RoutingtreeV1Beta1" {
			t.Fatalf("unexpected second routing-tree writer %q", logicalName)
		}
	}
	if policyCount != 1 {
		t.Fatalf("NotificationPolicy writers = %d, want exactly one", policyCount)
	}

	policy := desiredResource(t, rsp, "notification-policy")
	if got, want := desiredExternalName(t, policy), "notification-policy"; got != want {
		t.Fatalf("notification policy external name = %q, want %q", got, want)
	}
	parameters := nestedMap(t, policy, "spec", "forProvider")
	if got, want := parameters["contactPoint"], "teamdemo01-routing-operations"; got != want {
		t.Fatalf("default policy contact point = %v, want %q", got, want)
	}
	if got, want := parameters["groupBy"], []any{"alertname", "grafana_folder"}; gotValueDiff(got, want) {
		t.Fatalf("policy groupBy = %#v, want %#v", got, want)
	}
	policies, ok := parameters["policy"].([]any)
	if !ok || len(policies) != 1 {
		t.Fatalf("policy nodes = %#v, want one route", parameters["policy"])
	}
	child := policies[0].(map[string]any)
	if got, want := child["contactPoint"], "teamdemo01-routing-audit"; got != want {
		t.Fatalf("child policy contact point = %v, want %q", got, want)
	}
	if _, found := child["matchers"]; found {
		t.Fatalf("provider policy used request key matchers instead of schema key matcher: %#v", child)
	}
	if got, want := child["matcher"], []any{map[string]any{"label": "category", "match": "=", "value": "security"}}; gotValueDiff(got, want) {
		t.Fatalf("provider policy matcher = %#v, want %#v", got, want)
	}

	contact := desiredResource(t, rsp, "contact-point-operations")
	if got, want := desiredExternalName(t, contact), "teamdemo01-routing-operations"; got != want {
		t.Fatalf("contact point external name = %q, want %q", got, want)
	}
	ruleGroup := desiredResource(t, rsp, "rule-group-policy-health")
	if got, want := desiredExternalName(t, ruleGroup), "routing-folder:teamdemo01-routing-policy-health"; got != want {
		t.Fatalf("rule group external name = %q, want %q", got, want)
	}
	rules := nestedMap(t, ruleGroup, "spec", "forProvider")["rule"].([]any)
	if _, bypass := rules[0].(map[string]any)["notificationSettings"]; bypass {
		t.Fatalf("routing rule must not bypass the policy tree: %#v", rules[0])
	}
}

func TestAlertingRoutingSendsAnEmptyPolicyListForUnvendedNodes(t *testing.T) {
	claim := alertingRoutingDocument(map[string]any{
		"contactPoints": []any{map[string]any{
			"name": "operations", "email": map[string]any{"addresses": []any{"alerts@example.invalid"}},
		}},
		"defaultContactPoint": "operations",
		"routes":              []any{},
		"ruleGroups":          []any{},
	})
	policy := nestedMap(t, desiredResource(t, runAlertingRouting(t, claim), "notification-policy"), "spec", "forProvider")
	if got, want := policy["policy"], []any{}; gotValueDiff(got, want) {
		t.Fatalf("unvended policy nodes = %#v, want authoritative empty list", got)
	}
}

func TestAlertingRoutingRejectsUnknownRouteAndDirectRouteBypass(t *testing.T) {
	for name, additions := range map[string]map[string]any{
		"missing required interval": {
			"ruleGroups": []any{map[string]any{
				"name": "policy-health", "folderUid": "routing-folder", "rules": []any{map[string]any{
					"name": "Policy health", "uid": "policy-health", "condition": "A", "data": []any{map[string]any{"refId": "A"}},
				}},
			}},
		},
		"unknown default contact point": {"defaultContactPoint": "missing"},
		"unknown route contact point": {
			"routes": []any{map[string]any{
				"name": "unknown", "contactPoint": "missing", "matchers": []any{map[string]any{"label": "category", "match": "=", "value": "security"}},
			}},
		},
		"direct route bypass": {
			"ruleGroups": []any{map[string]any{
				"name": "policy-health", "folderUid": "routing-folder", "rules": []any{map[string]any{
					"name": "Policy health", "uid": "policy-health", "condition": "A", "data": []any{map[string]any{"refId": "A"}},
					"notificationSettings": map[string]any{"contactPoint": "operations"},
				}},
			}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			rsp := callFunctionWithRequiredResources(t, alertingRoutingDocument(additions), nil, requiredStackResource("teamdemo01", "grafana-vending", "True"), requiredResourceCapabilities())
			if fatal := fatalResult(rsp); fatal == "" {
				t.Fatal("RunFunction accepted an invalid routing request")
			}
		})
	}
}

func TestAlertingRoutingAdmissionAndUnroutableContactPointNegativeControl(t *testing.T) {
	ctx := context.Background()
	env := &admissionEnv{paths: []string{"../apis/alerting-routing-v1beta1.yaml"}}
	if err := env.Start(t); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() {
		if err := env.Stop(); err != nil {
			t.Errorf("Stop() error = %v", err)
		}
	})

	allowed := alertingRoutingRequest("teamdemo01")
	if err := env.Apply(ctx, allowed); err != nil {
		t.Fatalf("allowed control was refused: %v", err)
	}
	t.Log("allowed control output: admitted")
	contacts, found, err := unstructured.NestedSlice(allowed.Object, "spec", "contactPoints")
	if err != nil || !found {
		t.Fatalf("read contactPoints for allowed update: found=%t error=%v", found, err)
	}
	contacts[0].(map[string]any)["email"].(map[string]any)["subject"] = "Updated subject"
	if err := unstructured.SetNestedSlice(allowed.Object, contacts, "spec", "contactPoints"); err != nil {
		t.Fatalf("set contactPoints for allowed update: %v", err)
	}
	if err := env.Apply(ctx, allowed); err != nil {
		t.Fatalf("allowed update was refused: %v", err)
	}
	t.Log("allowed update output: admitted")

	rejectedUpdate := alertingRoutingRequest("teamdemo01")
	if err := env.Apply(ctx, rejectedUpdate); err != nil {
		t.Fatalf("recreate test object for rejected update: %v", err)
	}
	routes, found, err := unstructured.NestedSlice(rejectedUpdate.Object, "spec", "routes")
	if err != nil || !found {
		t.Fatalf("read routes for rejected update: found=%t error=%v", found, err)
	}
	routes[0].(map[string]any)["contactPoint"] = "operations"
	if err := unstructured.SetNestedSlice(rejectedUpdate.Object, routes, "spec", "routes"); err != nil {
		t.Fatalf("set routes for rejected update: %v", err)
	}
	const expectedMessage = "defaultContactPoint must be vended and every vended contact point must be the default route or referenced by a policy route"
	if err := env.Apply(ctx, rejectedUpdate); err == nil || !strings.Contains(err.Error(), expectedMessage) {
		t.Fatalf("unroutable update error = %v, want own message %q", err, expectedMessage)
	} else {
		t.Logf("unroutable update output: %v", err)
	}

	runAlertingRoutingUnroutableNegativeControl(ctx, env, t, expectedMessage)
}

func TestAlertingRoutingAdmissionRejectsRouteToUndeclaredContactPoint(t *testing.T) {
	ctx := context.Background()
	env := &admissionEnv{paths: []string{"../apis/alerting-routing-v1beta1.yaml"}}
	if err := env.Start(t); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() {
		if err := env.Stop(); err != nil {
			t.Errorf("Stop() error = %v", err)
		}
	})

	allowed := alertingRoutingRequest("routeguard01")
	if err := env.Apply(ctx, allowed); err != nil {
		t.Fatalf("allowed control was refused: %v", err)
	}
	t.Log("route reference allowed control output: admitted")
	contacts, found, err := unstructured.NestedSlice(allowed.Object, "spec", "contactPoints")
	if err != nil || !found {
		t.Fatalf("read contactPoints for allowed update: found=%t error=%v", found, err)
	}
	contacts[0].(map[string]any)["email"].(map[string]any)["subject"] = "Updated subject"
	if err := unstructured.SetNestedSlice(allowed.Object, contacts, "spec", "contactPoints"); err != nil {
		t.Fatalf("set contactPoints for allowed update: %v", err)
	}
	if err := env.Apply(ctx, allowed); err != nil {
		t.Fatalf("allowed update was refused: %v", err)
	}
	t.Log("route reference allowed update output: admitted")

	const expectedMessage = "every policy route must reference a vended contact point"
	baselineErr := env.Apply(ctx, alertingRoutingUndeclaredRouteRequest("routeguard02"))
	if baselineErr == nil || !strings.Contains(baselineErr.Error(), expectedMessage) {
		t.Fatalf("undeclared-route create error = %v, want own message %q", baselineErr, expectedMessage)
	}
	t.Logf("route reference baseline create output: %v", baselineErr)

	rejectedUpdate := alertingRoutingRequest("routeguard01")
	if err := env.Apply(ctx, rejectedUpdate); err != nil {
		t.Fatalf("recreate test object for rejected update: %v", err)
	}
	updateContacts, found, err := unstructured.NestedSlice(rejectedUpdate.Object, "spec", "contactPoints")
	if err != nil || !found || len(updateContacts) != 2 {
		t.Fatalf("read contactPoints for rejected update: found=%t len=%d error=%v", found, len(updateContacts), err)
	}
	if err := unstructured.SetNestedSlice(rejectedUpdate.Object, updateContacts[:1], "spec", "contactPoints"); err != nil {
		t.Fatalf("set contactPoints for rejected update: %v", err)
	}
	routes, found, err := unstructured.NestedSlice(rejectedUpdate.Object, "spec", "routes")
	if err != nil || !found {
		t.Fatalf("read routes for rejected update: found=%t error=%v", found, err)
	}
	routes[0].(map[string]any)["contactPoint"] = "missing"
	if err := unstructured.SetNestedSlice(rejectedUpdate.Object, routes, "spec", "routes"); err != nil {
		t.Fatalf("set routes for rejected update: %v", err)
	}
	updateErr := env.Apply(ctx, rejectedUpdate)
	if updateErr == nil || !strings.Contains(updateErr.Error(), expectedMessage) {
		t.Fatalf("undeclared-route update error = %v, want own message %q", updateErr, expectedMessage)
	}
	t.Logf("route reference update output: %v", updateErr)

	runAlertingRoutingRouteReferenceNegativeControl(ctx, env, t, expectedMessage)
}

func runAlertingRoutingRouteReferenceNegativeControl(ctx context.Context, env *admissionEnv, t *testing.T, expectedMessage string) {
	t.Helper()
	const (
		xrdPath      = "../apis/alerting-routing-v1beta1.yaml"
		originalRule = "- rule: self.spec.routes.all(route, self.spec.contactPoints.exists(contactPoint, contactPoint.name == route.contactPoint))"
		weakenedRule = "- rule: \"true\""
	)
	source, err := os.ReadFile(xrdPath)
	if err != nil {
		t.Fatalf("read route-reference source XRD: %v", err)
	}
	if got := strings.Count(string(source), originalRule); got != 1 {
		t.Fatalf("route-reference source contains %d copies of %q", got, originalRule)
	}
	scratchPath := filepath.Join(t.TempDir(), "alerting-routing-v1beta1.yaml")
	weakened := strings.Replace(string(source), originalRule, weakenedRule, 1)
	if err := os.WriteFile(scratchPath, []byte(weakened), 0o600); err != nil {
		t.Fatalf("write weakened route-reference XRD: %v", err)
	}
	weakenedCRD, err := crdFromXRD(scratchPath)
	if err != nil {
		t.Fatalf("derive weakened route-reference CRD: %v", err)
	}
	if err := env.Apply(ctx, weakenedCRD); err != nil {
		t.Fatalf("apply weakened route-reference CRD: %v", err)
	}

	weakenedErr := env.Apply(ctx, alertingRoutingUndeclaredRouteRequest("routeguard03"))
	if weakenedErr != nil {
		t.Fatalf("route-reference weakened output: %v", weakenedErr)
	}
	t.Log("route reference weakened output: admitted")

	originalCRD, err := crdFromXRD(xrdPath)
	if err != nil {
		t.Fatalf("derive original route-reference CRD for restore: %v", err)
	}
	if err := env.Apply(ctx, originalCRD); err != nil {
		t.Fatalf("restore route-reference CRD: %v", err)
	}
	restoredErr := env.Apply(ctx, alertingRoutingUndeclaredRouteRequest("routeguard04"))
	if restoredErr == nil || !strings.Contains(restoredErr.Error(), expectedMessage) {
		t.Fatalf("route-reference restored output did not reject with its own message: %v", restoredErr)
	}
	t.Logf("route reference restored output: %v", restoredErr)
}

func runAlertingRoutingUnroutableNegativeControl(ctx context.Context, env *admissionEnv, t *testing.T, expectedMessage string) {
	t.Helper()
	const (
		xrdPath      = "../apis/alerting-routing-v1beta1.yaml"
		originalRule = "- rule: self.spec.contactPoints.exists(contactPoint, contactPoint.name == self.spec.defaultContactPoint) && self.spec.contactPoints.all(contactPoint, contactPoint.name == self.spec.defaultContactPoint || self.spec.routes.exists(route, route.contactPoint == contactPoint.name))"
		weakenedRule = "- rule: \"true\""
	)

	baselineErr := env.Apply(ctx, alertingRoutingUnroutableRequest("unrouted01"))
	if baselineErr == nil || !strings.Contains(baselineErr.Error(), expectedMessage) {
		t.Fatalf("negative-control baseline did not reject unroutable contact point with its own message: %v", baselineErr)
	}
	t.Logf("negative control baseline output: %v", baselineErr)

	source, err := os.ReadFile(xrdPath)
	if err != nil {
		t.Fatalf("read negative-control source XRD: %v", err)
	}
	if got := strings.Count(string(source), originalRule); got != 1 {
		t.Fatalf("negative-control source contains %d copies of %q", got, originalRule)
	}
	scratchPath := filepath.Join(t.TempDir(), "alerting-routing-v1beta1.yaml")
	weakened := strings.Replace(string(source), originalRule, weakenedRule, 1)
	if err := os.WriteFile(scratchPath, []byte(weakened), 0o600); err != nil {
		t.Fatalf("write weakened scratch XRD: %v", err)
	}
	weakenedCRD, err := crdFromXRD(scratchPath)
	if err != nil {
		t.Fatalf("derive weakened scratch CRD: %v", err)
	}
	if err := env.Apply(ctx, weakenedCRD); err != nil {
		t.Fatalf("apply weakened scratch CRD: %v", err)
	}

	weakenedErr := env.Apply(ctx, alertingRoutingUnroutableRequest("unrouted02"))
	if weakenedErr != nil {
		t.Fatalf("negative control weakened output: %v", weakenedErr)
	}
	t.Log("negative control weakened output: admitted")

	originalCRD, err := crdFromXRD(xrdPath)
	if err != nil {
		t.Fatalf("derive original CRD for restore: %v", err)
	}
	if err := env.Apply(ctx, originalCRD); err != nil {
		t.Fatalf("restore original CRD: %v", err)
	}
	restoredErr := env.Apply(ctx, alertingRoutingUnroutableRequest("unrouted03"))
	if restoredErr == nil || !strings.Contains(restoredErr.Error(), expectedMessage) {
		t.Fatalf("negative control restored output did not reject unroutable contact point with its own message: %v", restoredErr)
	}
	t.Logf("negative control restored output: %v", restoredErr)
}

func runAlertingRouting(t *testing.T, claim string) *fnv1.RunFunctionResponse {
	t.Helper()
	rsp := callFunctionWithRequiredResources(t, claim, nil, requiredStackResource("teamdemo01", "grafana-vending", "True"), requiredResourceCapabilities())
	if fatal := fatalResult(rsp); fatal != "" {
		t.Fatalf("RunFunction returned a fatal result: %s", fatal)
	}
	return rsp
}

func alertingRoutingDocument(additions map[string]any) string {
	spec := map[string]any{
		"stackRef": map[string]any{"name": "teamdemo01"},
		"contactPoints": []any{
			map[string]any{"name": "operations", "email": map[string]any{"addresses": []any{"alerts@example.invalid"}, "subject": "Example alert"}},
			map[string]any{"name": "audit", "email": map[string]any{"addresses": []any{"audit@example.invalid"}}},
		},
		"defaultContactPoint": "operations",
		"routes": []any{map[string]any{
			"name": "audit-security", "contactPoint": "audit", "matchers": []any{map[string]any{"label": "category", "match": "=", "value": "security"}}, "groupWait": "30s",
		}},
		"ruleGroups": []any{map[string]any{
			"name": "policy-health", "folderUid": "routing-folder", "intervalSeconds": 60,
			"rules": []any{map[string]any{
				"name": "Notification policy health", "uid": "notification-policy-health", "condition": "A",
				"data":        []any{map[string]any{"refId": "A", "queryType": "", "datasourceUid": "-100", "model": "{\"expression\":\"1\",\"refId\":\"A\",\"type\":\"math\"}", "relativeTimeRange": []any{map[string]any{"from": 0, "to": 0}}}},
				"noDataState": "NoData", "execErrState": "Alerting", "labels": map[string]any{"category": "security"},
			}},
		}},
	}
	for key, value := range additions {
		spec[key] = value
	}
	return mustJSON(map[string]any{
		"apiVersion": "platform.example.org/v1beta1", "kind": "GrafanaAlertingRouting",
		"metadata": map[string]any{"name": "teamdemo01", "namespace": "grafana-vending"}, "spec": spec,
	})
}

func alertingRoutingRequest(name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "platform.example.org/v1beta1", "kind": "GrafanaAlertingRouting",
		"metadata": map[string]any{"name": name, "namespace": "default"},
		"spec": map[string]any{
			"stackRef": map[string]any{"name": name},
			"contactPoints": []any{
				map[string]any{"name": "operations", "email": map[string]any{"addresses": []any{"alerts@example.invalid"}}},
				map[string]any{"name": "audit", "email": map[string]any{"addresses": []any{"audit@example.invalid"}}},
			},
			"defaultContactPoint": "operations",
			"routes": []any{map[string]any{
				"name": "audit-security", "contactPoint": "audit", "matchers": []any{map[string]any{"label": "category", "match": "=", "value": "security"}},
			}},
		},
	}}
}

func alertingRoutingUnroutableRequest(name string) *unstructured.Unstructured {
	obj := alertingRoutingRequest(name)
	contacts, _, _ := unstructured.NestedSlice(obj.Object, "spec", "contactPoints")
	contacts = append(contacts, map[string]any{"name": "unroutable", "email": map[string]any{"addresses": []any{"unroutable@example.invalid"}}})
	_ = unstructured.SetNestedSlice(obj.Object, contacts, "spec", "contactPoints")
	return obj
}

func alertingRoutingUndeclaredRouteRequest(name string) *unstructured.Unstructured {
	obj := alertingRoutingRequest(name)
	contacts := []any{map[string]any{"name": "operations", "email": map[string]any{"addresses": []any{"alerts@example.invalid"}}}}
	_ = unstructured.SetNestedSlice(obj.Object, contacts, "spec", "contactPoints")
	routes, _, _ := unstructured.NestedSlice(obj.Object, "spec", "routes")
	routes[0].(map[string]any)["contactPoint"] = "missing"
	_ = unstructured.SetNestedSlice(obj.Object, routes, "spec", "routes")
	return obj
}
