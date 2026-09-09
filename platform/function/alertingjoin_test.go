package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestAlertingJoinAtAPIServer(t *testing.T) {
	e := &admissionEnv{paths: []string{"../apis/oncall-v1beta1.yaml", "../apis/alerting-routing-v1beta1.yaml"}}
	if err := e.Start(t); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := e.Stop(); err != nil {
			t.Error(err)
		}
	})
	installProviderAdmissionCRDs(t, e)
	ctx := context.Background()
	xr := onCallRequest("teamdemo01")
	if err := e.Apply(ctx, xr); err != nil {
		t.Fatal(err)
	}
	if err := e.client.Get(ctx, client.ObjectKeyFromObject(xr), xr); err != nil {
		t.Fatal(err)
	}
	observed := map[resource.Name]resource.ObservedComposed{}
	var desired map[resource.Name]*resource.DesiredComposed
	for stage := 0; stage < 8; stage++ {
		var err error
		desired, err = renderOnCall(xr.Object, observed, nil)
		if err != nil {
			t.Fatal(err)
		}
		admitProviderChildren(t, e, desired)
		for key, child := range desired {
			if _, exists := observed[key]; exists {
				continue
			}
			obj := &unstructured.Unstructured{}
			if err := json.Unmarshal([]byte(mustJSON(child.Resource.UnstructuredContent())), &obj.Object); err != nil {
				t.Fatal(err)
			}
			if err := e.client.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
				t.Fatal(err)
			}
			annotations := obj.GetAnnotations()
			if annotations == nil {
				annotations = map[string]string{}
			}
			annotations["crossplane.io/external-name"] = "fixture-" + string(key)
			obj.SetAnnotations(annotations)
			if err := e.client.Update(ctx, obj); err != nil {
				t.Fatal(err)
			}
			at := map[string]any{}
			if key == "integration" {
				at["inboundEmail"] = "fixture-integration@example.invalid"
			}
			obj.Object["status"] = map[string]any{"conditions": []any{map[string]any{"type": "Synced", "status": "True", "observedGeneration": obj.GetGeneration(), "reason": "FixtureObservation", "lastTransitionTime": "2026-01-01T00:00:00Z"}, map[string]any{"type": "Ready", "status": "True", "reason": "FixtureObservation", "lastTransitionTime": "2026-01-01T00:00:00Z"}}, "atProvider": at}
			if err := e.client.Status().Update(ctx, obj); err != nil {
				t.Fatal(err)
			}
			if err := e.client.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
				t.Fatal(err)
			}
			c := composed.New()
			c.SetUnstructuredContent(obj.Object)
			observed[key] = resource.ObservedComposed{Resource: c}
			t.Logf("API-SERVER fixture observation %s: %s", key, mustJSON(obj.Object["status"]))
		}
		if desired["catch-all-route"] != nil {
			break
		}
	}
	if desired["catch-all-route"] == nil {
		t.Fatal("full responder graph was never rendered")
	}
	changed := xr.DeepCopy()
	changed.Object["spec"].(map[string]any)["shiftStart"] = "2025-02-01T00:00:00"
	changedDesired, renderErr := renderOnCall(changed.Object, observed, nil)
	if renderErr != nil {
		t.Fatal(renderErr)
	}
	if _, ready := onCallReceiverStatus(changed.Object, observed, changedDesired); ready {
		t.Fatal("stale shift observations published a current receiver")
	}
	t.Log("OBSERVATION changed shift with old Ready children refused before receiver projection")
	status, ready := onCallReceiverStatus(xr.Object, observed, desired)
	if !ready {
		t.Fatalf("observed receiver not ready: %s", mustJSON(status.Resource.UnstructuredContent()))
	}
	routeObject := observed["catch-all-route"].Resource.UnstructuredContent()
	parameters := routeObject["spec"].(map[string]any)["forProvider"].(map[string]any)
	originalIntegration := parameters["integrationId"]
	parameters["integrationId"] = "wrong-integration"
	if _, ready := onCallReceiverStatus(xr.Object, observed, desired); ready {
		t.Fatal("stale route references published receiver")
	}
	t.Log("OBSERVATION stale route integration link refused before receiver projection")
	parameters["integrationId"] = originalIntegration
	routeStatus := routeObject["status"].(map[string]any)["conditions"].([]any)[0].(map[string]any)
	currentGeneration := routeStatus["observedGeneration"]
	routeStatus["observedGeneration"] = int64(0)
	if _, ready := onCallReceiverStatus(xr.Object, observed, desired); ready {
		t.Fatal("stale provider generation published receiver")
	}
	t.Log("OBSERVATION stale provider generation refused before receiver projection")
	routeStatus["observedGeneration"] = currentGeneration
	xr.Object["status"] = status.Resource.UnstructuredContent()["status"]
	xr.Object["status"].(map[string]any)["conditions"] = []any{map[string]any{"type": "Ready", "status": "True", "reason": "FixtureObservation", "lastTransitionTime": "2026-01-01T00:00:00Z"}}
	if err := e.client.Status().Update(ctx, xr); err != nil {
		t.Fatal(err)
	}
	if err := e.client.Get(ctx, client.ObjectKeyFromObject(xr), xr); err != nil {
		t.Fatal(err)
	}
	t.Logf("API-SERVER GrafanaOnCall receiver projection: %s", mustJSON(xr.Object["status"]))
	claim := alertingRoutingRequest("teamdemo01")
	spec := claim.Object["spec"].(map[string]any)
	spec["contactPoints"] = []any{map[string]any{"name": "operations", "onCallRef": map[string]any{"name": "teamdemo01"}}}
	spec["defaultContactPoint"] = "operations"
	spec["routes"] = []any{}
	spec["ruleGroups"] = []any{map[string]any{"name": "policy-health", "folderUid": "fixture-folder", "intervalSeconds": float64(60), "rules": []any{map[string]any{"name": "Policy proof", "condition": "A", "data": []any{map[string]any{"refId": "A", "datasourceUid": "__expr__", "model": `{"expression":"1 > 0","refId":"A","type":"math"}`, "relativeTimeRange": []any{map[string]any{"from": float64(0), "to": float64(0)}}}}}}}}
	if err := e.Apply(ctx, claim); err != nil {
		t.Fatal(err)
	}
	if err := e.client.Get(ctx, client.ObjectKeyFromObject(claim), claim); err != nil {
		t.Fatal(err)
	}
	req := &fnv1.RunFunctionRequest{}
	_, ready, err := resolveAlertingReceivers(req, &fnv1.RunFunctionResponse{}, claim.Object)
	if err != nil || ready {
		t.Fatalf("unobserved reference accepted: %t %v", ready, err)
	}
	req.RequiredResources = map[string]*fnv1.Resources{"oncall-receiver-teamdemo01": {Items: []*fnv1.Resource{{Resource: resource.MustStructJSON(mustJSON(xr.Object))}}}}
	resolved, ready, err := resolveAlertingReceivers(req, &fnv1.RunFunctionResponse{}, claim.Object)
	if err != nil || !ready {
		t.Fatalf("reference resolution: %t %v", ready, err)
	}
	routed, err := renderAlertingRouting(resolved, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	admitProviderChildren(t, e, routed)
	cp := nestedMap(t, routed["contact-point-operations"].Resource.UnstructuredContent(), "spec", "forProvider")
	if got := cp["email"].([]any)[0].(map[string]any)["addresses"].([]any)[0]; got != "fixture-integration@example.invalid" {
		t.Fatalf("observed address lost: %v", got)
	}
	policy := nestedMap(t, routed["notification-policy"].Resource.UnstructuredContent(), "spec", "forProvider")
	if policy["contactPoint"] != cp["name"] {
		t.Fatal("policy misses vended contact point")
	}
	rules := nestedMap(t, routed["rule-group-policy-health"].Resource.UnstructuredContent(), "spec", "forProvider")["rule"].([]any)
	if _, found := rules[0].(map[string]any)["notificationSettings"]; found {
		t.Fatal("ordinary rule bypasses policy")
	}
	chain := nestedMap(t, desired["schedule-escalation"].Resource.UnstructuredContent(), "spec", "forProvider")
	route := nestedMap(t, desired["catch-all-route"].Resource.UnstructuredContent(), "spec", "forProvider")
	if route["integrationId"] != observedExternalName(observed, "integration") || route["escalationChainId"] != chain["escalationChainId"] || chain["notifyOnCallFromSchedule"] != observedExternalName(observed, "schedule") {
		t.Fatal("responder references do not join")
	}
	t.Log("PROVEN configuration chain: ordinary RuleGroup -> NotificationPolicy -> email ContactPoint -> observed inbound-email Integration -> catch-all Route -> EscalationChain -> schedule Escalation -> Schedule -> rotating Shift -> observed Users")
	t.Log("UNPROVEN: fixtures supplied provider observations; no rule evaluation, SMTP delivery or actual page occurred")
	req.Meta = &fnv1.RequestMeta{Capabilities: requiredResourceCapabilities()}
	req.Observed = &fnv1.State{Composite: &fnv1.Resource{Resource: resource.MustStructJSON(mustJSON(claim.Object))}}
	req.RequiredResources[referencedStackRequirement] = requiredStackResource("teamdemo01", claim.GetNamespace(), "True")
	rsp, err := (&Function{log: logging.NewNopLogger()}).RunFunction(ctx, req)
	if err != nil || fatalResult(rsp) != "" {
		t.Fatalf("joined RunFunction failed: %v %s", err, fatalResult(rsp))
	}
	if rsp.GetDesired().GetResources()["contact-point-operations"] == nil {
		t.Fatal("joined receiver is unreachable through RunFunction")
	}
	t.Log("RunFunction consumed observed receiver and emitted the joined ContactPoint, NotificationPolicy and RuleGroup")
	for _, bad := range []string{"stack", "namespace", "generation"} {
		t.Run("reject-"+bad, func(t *testing.T) {
			obj := xr.DeepCopy()
			switch bad {
			case "stack":
				obj.Object["spec"].(map[string]any)["stackRef"] = map[string]any{"name": "wrongstack"}
			case "namespace":
				obj.SetNamespace("wrong")
			case "generation":
				obj.SetGeneration(obj.GetGeneration() + 1)
			}
			req.RequiredResources["oncall-receiver-teamdemo01"].Items[0].Resource = resource.MustStructJSON(mustJSON(obj.Object))
			_, ready, err := resolveAlertingReceivers(req, &fnv1.RunFunctionResponse{}, claim.Object)
			if ready {
				t.Fatalf("stale or foreign context trusted: %s", bad)
			}
			t.Logf("reconcile refused %s: ready=%t error=%v", bad, ready, err)
		})
	}
}

func TestAlertingJoinAdmissionGuards(t *testing.T) {
	e := &admissionEnv{paths: []string{"../apis/alerting-routing-v1beta1.yaml"}}
	if err := e.Start(t); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := e.Stop(); err != nil {
			t.Error(err)
		}
	})
	ctx := context.Background()
	obj := alertingRoutingRequest("receiverchoice")
	if err := e.Apply(ctx, obj); err != nil {
		t.Fatal(err)
	}
	if err := e.client.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
		t.Fatal(err)
	}
	points := obj.Object["spec"].(map[string]any)["contactPoints"].([]any)
	point := points[0].(map[string]any)
	point["onCallRef"] = map[string]any{"name": "receiverchoice"}
	if err := e.Apply(ctx, obj); err == nil || !strings.Contains(err.Error(), "exactly one of email or onCallRef") {
		t.Fatalf("dual destination update: %v", err)
	} else {
		t.Logf("API-SERVER dual destination refused: %v", err)
	}
	delete(point, "email")
	if err := e.Apply(ctx, obj); err != nil {
		t.Fatalf("reference-only control refused: %v", err)
	}
	point = obj.Object["spec"].(map[string]any)["contactPoints"].([]any)[0].(map[string]any)
	delete(point, "onCallRef")
	if err := e.Apply(ctx, obj); err == nil || !strings.Contains(err.Error(), "exactly one of email or onCallRef") {
		t.Fatalf("absent destination: %v", err)
	} else {
		t.Logf("API-SERVER absent destination refused: %v", err)
	}
}
