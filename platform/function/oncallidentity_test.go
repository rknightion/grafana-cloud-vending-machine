package main

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
	"google.golang.org/protobuf/proto"
)

// A trace observes only preceding desired outputs. Journal writes are persisted
// separately, so a returned phase cannot become authority in its own invocation.
func TestOnCallIdentityMigrationTrace(t *testing.T) {
	for _, mode := range []string{"forward", "committed-opt-out", "closed-admission", "unresolved-absence", "reverse", "cancellation"} {
		t.Run(mode, func(t *testing.T) {
			xr := onCallTestXR()
			xr["apiVersion"] = "platform.example.org/v1beta1"
			xr["kind"] = "GrafanaOnCall"
			meta := xr["metadata"].(map[string]any)
			meta["uid"] = "owner-fixture"
			meta["generation"] = int64(1)
			observed := identityBaseline(t, xr)
			xr["spec"].(map[string]any)["integrationType"] = "grafana_alerting"
			if mode == "cancellation" {
				xr["spec"].(map[string]any)["route"] = map[string]any{"match": "all", "channelId": "C12345678"}
			}
			req := &fnv1.RunFunctionRequest{RequiredResources: map[string]*fnv1.Resources{referencedStackRequirement: requiredStackResource("stackdemo", "grafana-vending", "True")}}
			var previousPhase string
			changed, cancelled, reversed := false, false, false
			for step := 0; step < 40; step++ {
				req.Observed = &fnv1.State{Composite: &fnv1.Resource{Resource: resource.MustStructJSON(mustJSON(xr))}, Resources: map[string]*fnv1.Resource{}}
				for key, obj := range observed {
					req.Observed.Resources[string(key)] = &fnv1.Resource{Resource: resource.MustStructJSON(mustJSON(obj.Resource.UnstructuredContent()))}
				}
				if mode == "closed-admission" && previousPhase == "Prepared" {
					req.RequiredResources[referencedStackRequirement] = requiredStackResource("stackdemo", "grafana-vending", "False")
					req.Desired = nil
				}
				rsp, err := (&Function{}).RunFunction(context.Background(), req)
				if err != nil || fatalResult(rsp) != "" {
					t.Fatalf("trace step %d failed: %v %s", step, err, fatalResult(rsp))
				}
				status := rsp.GetDesired().GetComposite().GetResource().AsMap()["status"].(map[string]any)
				journal, ok := status["onCallIdentity"].(map[string]any)
				if !ok {
					t.Fatal("opt-in must return a durable journal before creating successor children")
				}
				trans, _ := journal["transition"].(map[string]any)
				phase, _ := trans["phase"].(string)
				children := rsp.GetDesired().GetResources()
				if mode == "cancellation" && trans["disposition"] == "cancellation" {
					for _, result := range rsp.GetResults() {
						if result.GetMessage() == "OnCall Slack destination is not confirmed by current provider readback" {
							t.Fatal("cancellation warned about the discarded target instead of the schedule-only keep pair")
						}
					}
				}
				if previousPhase == "Prepared" && mode != "closed-admission" && mode != "cancellation" {
					for _, key := range []string{"integration", "catch-all-route", "integration-grafana-alerting", "catch-all-route-grafana-alerting"} {
						if children[key] == nil {
							t.Fatal("first returned retirement phase withdrew a pair before observed authority")
						}
					}
				}
				if mode == "closed-admission" && previousPhase == "Prepared" {
					if children["catch-all-route"] == nil || phase != previousPhase {
						t.Fatal("closed admission failed to preserve predecessor and journal")
					}
					return
				}
				if mode == "committed-opt-out" && phase == "RetiringRoute" && previousPhase == phase && !changed {
					delete(xr["spec"].(map[string]any), "integrationType")
					meta["generation"] = int64(2)
					changed = true
				}
				if mode == "cancellation" && phase == "Building" && children["integration-grafana-alerting"] != nil && !cancelled {
					delete(xr["spec"].(map[string]any), "integrationType")
					delete(xr["spec"].(map[string]any)["route"].(map[string]any), "channelId")
					meta["generation"] = int64(2)
					cancelled = true
				}
				if phase == "RetiringRoute" && children["catch-all-route"] == nil && mode == "unresolved-absence" {
					// Missing composed observation without a resolved named lookup is not absence.
					delete(observed, "catch-all-route")
					xr["status"] = status
					req.Desired = rsp.Desired
					if previousPhase == "RetiringRoute" {
						if trans["phase"] != "RetiringRoute" || children["integration"] == nil {
							t.Fatal("unresolved named absence retired Integration")
						}
						return
					}
					previousPhase = phase
					continue
				}
				// Persist status, then observe only emitted desired resources on the next turn.
				xr["status"] = status
				next := map[resource.Name]*resource.DesiredComposed{}
				for key, obj := range children {
					c := newDesired("", "", "", "", nil, nil)
					c.Resource.SetUnstructuredContent(obj.Resource.AsMap())
					next[resource.Name(key)] = c
				}
				observed = identityObserve(t, xr, next, observed)
				for key, selector := range rsp.GetRequirements().GetResources() {
					if key == referencedStackRequirement {
						continue
					}
					found := false
					for _, o := range observed {
						m := o.Resource.UnstructuredContent()["metadata"].(map[string]any)
						if m["name"] == selector.GetMatchName() {
							found = true
						}
					}
					if !found {
						req.RequiredResources[key] = nil
					}
				}
				// Keep stale incoming losing entries throughout to test SDK merge pruning.
				req.Desired = proto.Clone(rsp.Desired).(*fnv1.State)
				if req.Desired.Resources == nil {
					req.Desired.Resources = map[string]*fnv1.Resource{}
				}
				for _, key := range []string{"integration", "catch-all-route"} {
					if req.Desired.Resources[key] == nil {
						req.Desired.Resources[key] = &fnv1.Resource{Resource: resource.MustStructJSON(mustJSON(identityBaseline(t, onCallTestXR())[resource.Name(key)].Resource.UnstructuredContent()))}
					}
				}
				if trans == nil {
					active, _ := journal["activeType"].(string)
					if mode == "reverse" && active == "grafana_alerting" && !reversed {
						delete(xr["spec"].(map[string]any), "integrationType")
						meta["generation"] = int64(2)
						reversed = true
						previousPhase = ""
						continue
					}
					want := "grafana_alerting"
					if mode == "reverse" || mode == "cancellation" {
						want = "inbound_email"
					}
					if active == want && (mode != "committed-opt-out" || changed) {
						if want == "grafana_alerting" && (children["integration"] != nil || children["catch-all-route"] != nil) {
							t.Fatal("stable retirement resurrected stale incoming children")
						}
						if mode == "reverse" {
							a := children["integration"].Resource.AsMap()["metadata"].(map[string]any)["annotations"].(map[string]any)
							if a["crossplane.io/external-name"] != "fixture-integration" {
								t.Fatal("reverse migration did not adopt inventoried released identity")
							}
						}
						return
					}
				}
				previousPhase = phase
			}
			t.Fatal("bounded lifecycle did not complete")
		})
	}
}

func identityBaseline(t *testing.T, xr map[string]any) map[resource.Name]resource.ObservedComposed {
	t.Helper()
	observed := map[resource.Name]resource.ObservedComposed{}
	for stage := 0; stage < 8; stage++ {
		desired, err := renderOnCallLegacy(xr, observed, nil)
		if err != nil {
			t.Fatal(err)
		}
		observed = identityObserve(t, xr, desired, observed)
		if desired["catch-all-route"] != nil {
			return observed
		}
	}
	t.Fatal("baseline did not render a full graph")
	return nil
}
func identityObserve(t *testing.T, xr map[string]any, desired map[resource.Name]*resource.DesiredComposed, old map[resource.Name]resource.ObservedComposed) map[resource.Name]resource.ObservedComposed {
	t.Helper()
	normalized := map[resource.Name]*resource.DesiredComposed{}
	for key, d := range desired {
		obj := newDesired("", "", "", "", nil, nil)
		var content map[string]any
		if err := json.Unmarshal([]byte(mustJSON(d.Resource.UnstructuredContent())), &content); err != nil {
			t.Fatal(err)
		}
		obj.Resource.SetUnstructuredContent(content)
		normalized[key] = obj
	}
	result := currentOnCallReceiverObservations(t, normalized)
	for key, o := range result {
		obj := o.Resource.UnstructuredContent()
		m := obj["metadata"].(map[string]any)
		uid := "uid-" + string(key)
		if prev, ok := old[key]; ok {
			uid = string(prev.Resource.GetUID())
		}
		if uid == "" {
			uid = "uid-" + string(key)
		}
		m["uid"] = uid
		m["ownerReferences"] = []any{map[string]any{"apiVersion": "platform.example.org/v1beta1", "kind": "GrafanaOnCall", "name": xr["metadata"].(map[string]any)["name"], "uid": "owner-fixture", "controller": true}}
		spec := obj["spec"].(map[string]any)
		p, _ := spec["forProvider"].(map[string]any)
		at := map[string]any{}
		if obj["kind"] == "Integration" {
			at["type"] = p["type"]
			at["link"] = "https://hooks.example.invalid/fixture"
			at["inboundEmail"] = "fixture@example.invalid"
		}
		if obj["kind"] == "Route" {
			at["integrationId"] = p["integrationId"]
			at["escalationChainId"] = p["escalationChainId"]
			at["slack"] = p["slack"]
		}
		obj["status"].(map[string]any)["atProvider"] = at
		o.Resource.SetUnstructuredContent(obj)
		result[key] = o
	}
	return result
}

func TestOnCallFreshSlackAndCombinedJoin(t *testing.T) {
	for _, mode := range []string{"fresh-direct", "fresh-reference", "default-slack", "slack-removal", "source-free-opt-out", "reference-change-migration", "stable-responder-addition"} {
		t.Run(mode, func(t *testing.T) {
			xr := onCallTestXR()
			xr["apiVersion"] = "platform.example.org/v1beta1"
			xr["kind"] = "GrafanaOnCall"
			meta := xr["metadata"].(map[string]any)
			meta["uid"] = "owner-fixture"
			meta["generation"] = int64(1)
			spec := xr["spec"].(map[string]any)
			spec["integrationType"] = "grafana_alerting"
			spec["route"] = map[string]any{"channelId": "CABC"}
			if mode == "reference-change-migration" {
				spec["integrationType"] = "inbound_email"
				spec["route"] = map[string]any{"channelRef": map[string]any{"name": "operations"}}
			}
			if mode == "fresh-reference" {
				spec["route"] = map[string]any{"channelRef": map[string]any{"name": "operations"}}
			}
			observed := map[resource.Name]resource.ObservedComposed{}
			if mode == "default-slack" || mode == "slack-removal" {
				delete(spec, "integrationType")
				observed = identityBaseline(t, xr)
			}
			req := &fnv1.RunFunctionRequest{RequiredResources: map[string]*fnv1.Resources{referencedStackRequirement: requiredStackResource("stackdemo", "grafana-vending", "True")}}
			changed := false
			reversed := false
			for step := 0; step < 60; step++ {
				req.Observed = &fnv1.State{Composite: &fnv1.Resource{Resource: resource.MustStructJSON(mustJSON(xr))}, Resources: map[string]*fnv1.Resource{}}
				for key, o := range observed {
					req.Observed.Resources[string(key)] = &fnv1.Resource{Resource: resource.MustStructJSON(mustJSON(o.Resource.UnstructuredContent()))}
				}
				rsp, err := (&Function{}).RunFunction(context.Background(), req)
				if err != nil || fatalResult(rsp) != "" {
					t.Fatalf("step %d failed: %v %s", step, err, fatalResult(rsp))
				}
				status := rsp.GetDesired().GetComposite().GetResource().AsMap()["status"].(map[string]any)
				journal, _ := status["onCallIdentity"].(map[string]any)
				if journal == nil {
					t.Fatal("fresh opt-in has no durable journal")
				}
				xr["status"] = status
				desired := map[resource.Name]*resource.DesiredComposed{}
				for key, child := range rsp.GetDesired().GetResources() {
					d := newDesired("", "", "", "", nil, nil)
					d.Resource.SetUnstructuredContent(child.Resource.AsMap())
					desired[resource.Name(key)] = d
				}
				if mode == "source-free-opt-out" && !changed && desired["integration-grafana-alerting"] != nil {
					delete(spec, "integrationType")
					meta["generation"] = int64(2)
					changed = true
				}
				if mode == "slack-removal" && !changed && status["alertReceiver"] != nil {
					delete(spec["route"].(map[string]any), "channelId")
					meta["generation"] = int64(2)
					changed = true
				}
				if mode == "slack-removal" && changed && status["alertReceiver"] != nil && step > 1 {
					_, rk := onCallKeys("inbound_email")
					old := observed[rk]
					s := old.Resource.UnstructuredContent()["status"].(map[string]any)
					at := s["atProvider"].(map[string]any)
					at["slack"] = []any{map[string]any{"enabled": true, "channelId": "CABC"}}
					old.Resource.UnstructuredContent()["spec"].(map[string]any)["forProvider"].(map[string]any)["slack"] = []any{}
					if _, ready := onCallReceiverStatus(xr, observed, desired); ready {
						t.Fatal("removed Slack destination remained receiver-ready with old enabled readback")
					}
					return
				}
				if mode == "stable-responder-addition" && status["alertReceiver"] != nil && !changed {
					spec["responders"] = append(spec["responders"].([]any), map[string]any{"username": "new-responder"})
					meta["generation"] = int64(2)
					changed = true
				} else if mode == "stable-responder-addition" && changed && status["alertReceiver"] != nil {
					return
				}
				if mode == "reference-change-migration" && status["alertReceiver"] != nil && !changed {
					spec["integrationType"] = "grafana_alerting"
					spec["route"] = map[string]any{"channelRef": map[string]any{"name": "operations-new"}}
					meta["generation"] = int64(2)
					changed = true
				}
				if mode == "reference-change-migration" && changed {
					// A predecessor Always reference must still resolve its original
					// destination while the successor pair is being observed.
					if oldRoute := desired["catch-all-route"]; oldRoute != nil && !reversed {
						lookup := desired["slack-channel"]
						if lookup == nil || lookup.Resource.UnstructuredContent()["spec"].(map[string]any)["forProvider"].(map[string]any)["name"] != "operations" {
							t.Fatal("successor lookup overwrote the predecessor destination")
						}
						slack := oldRoute.Resource.UnstructuredContent()["spec"].(map[string]any)["forProvider"].(map[string]any)["slack"].([]any)[0].(map[string]any)
						if slack["channelId"] != "CABC" || slack["slackChannelRef"].(map[string]any)["name"] != "stackdemo-slack-channel" {
							t.Fatal("predecessor destination changed before retirement")
						}
					}
					if status["alertReceiver"] != nil && journal["activeType"] == "grafana_alerting" && !reversed {
						spec["integrationType"] = "inbound_email"
						spec["route"] = map[string]any{"channelRef": map[string]any{"name": "operations"}}
						meta["generation"] = int64(3)
						reversed = true
					} else if reversed && status["alertReceiver"] != nil && journal["activeType"] == "inbound_email" {
						return
					}
				}
				if status["alertReceiver"] != nil && mode != "slack-removal" && mode != "reference-change-migration" && mode != "stable-responder-addition" {

					if mode == "source-free-opt-out" {
						if journal["activeType"] != "inbound_email" {
							t.Fatal("source-free opt-out did not establish released keep pair")
						}
						return
					}
					if mode == "default-slack" {
						return
					}
					claim := alertingRoutingRequest("stackdemo").Object
					claim["metadata"].(map[string]any)["namespace"] = "grafana-vending"
					cs := claim["spec"].(map[string]any)
					cs["contactPoints"] = []any{map[string]any{"name": "operations", "onCallRef": map[string]any{"name": "stackdemo"}}}
					cs["defaultContactPoint"] = "operations"
					cs["routes"] = []any{map[string]any{"name": "all", "contactPoint": "operations", "matchers": []any{map[string]any{"label": "alertname", "match": "=~", "value": ".*"}}}}
					status["conditions"] = []any{map[string]any{"type": "Ready", "status": "True"}}
					integration := observed["integration-grafana-alerting"].Resource.UnstructuredContent()
					integration["status"].(map[string]any)["atProvider"].(map[string]any)["link"] = "https://hooks.example.invalid/secret-sentinel"
					joinReq := &fnv1.RunFunctionRequest{RequiredResources: map[string]*fnv1.Resources{"oncall-receiver-stackdemo": {Items: []*fnv1.Resource{{Resource: resource.MustStructJSON(mustJSON(xr))}}}, "oncall-integration-stackdemo": {Items: []*fnv1.Resource{{Resource: resource.MustStructJSON(mustJSON(integration))}}}}}
					resolved, ready, e := resolveAlertingReceivers(joinReq, &fnv1.RunFunctionResponse{}, claim)
					if e != nil || !ready {
						t.Fatalf("internal join failed: %v", e)
					}
					routing, e := renderAlertingRouting(resolved, nil, nil)
					if e != nil {
						t.Fatal(e)
					}
					cp := routing["contact-point-operations"].Resource.UnstructuredContent()["spec"].(map[string]any)["forProvider"].(map[string]any)
					if cp["oncall"] == nil || cp["email"] != nil || desired["escalation-chain"] == nil || desired["integration-grafana-alerting"] == nil || routing["notification-policy"] == nil {
						t.Fatal("combined four shapes are not linked")
					}
					for _, surface := range []any{claim, status, journal, rsp.Results} {
						raw, _ := json.Marshal(surface)
						if bytes.Contains(raw, []byte("secret-sentinel")) {
							t.Fatal("bearer link escaped its managed transport boundary")
						}
					}
					return
				}
				observed = identityObserve(t, xr, desired, observed)
				for _, key := range []resource.Name{"slack-channel", "slack-channel-grafana-alerting"} {
					if o, exists := observed[key]; exists {
						id := "CABC"
						if o.Resource.UnstructuredContent()["spec"].(map[string]any)["forProvider"].(map[string]any)["name"] == "operations-new" {
							id = "CNEW"
						}
						o.Resource.UnstructuredContent()["status"].(map[string]any)["atProvider"] = map[string]any{"slackID": id}
					}
				}
				for key, selector := range rsp.GetRequirements().GetResources() {
					if key == referencedStackRequirement {
						continue
					}
					present := false
					for _, o := range observed {
						if o.Resource.GetName() == selector.GetMatchName() {
							present = true
						}
					}
					if !present {
						req.RequiredResources[key] = nil
					}
				}
				req.Desired = proto.Clone(rsp.Desired).(*fnv1.State)
			}
			t.Fatal("fresh/Slack lifecycle failed to publish a usable receiver")
		})
	}
}

func TestOnCallRejectsDisplayNameDestination(t *testing.T) {
	xr := onCallTestXR()
	xr["spec"].(map[string]any)["route"] = map[string]any{"channelId": "operations-channel"}
	if _, err := renderOnCall(xr, onCallObservedResponders(t), nil); err == nil {
		t.Fatal("display-name destination was accepted")
	}
}

func TestRoutingInternalIRMExclusiveAndSecretSafe(t *testing.T) {
	xr := alertingRoutingRequest("stackdemo").Object
	spec := xr["spec"].(map[string]any)
	points := spec["contactPoints"].([]any)
	point := points[0].(map[string]any)
	delete(point, "email")
	point["oncall"] = map[string]any{"url": "https://hooks.example.invalid/sentinel", "oncallIntegrationRef": map[string]any{"name": "stackdemo-grafana-alerting", "policy": map[string]any{"resolution": "Required", "resolve": "Always"}}}
	desired, err := renderAlertingRouting(xr, nil, nil)
	if err != nil {
		t.Fatalf("internal IRM shape refused: %v", err)
	}
	cp := desired[resource.Name("contact-point-"+point["name"].(string))].Resource.UnstructuredContent()["spec"].(map[string]any)["forProvider"].(map[string]any)
	if cp["oncall"] == nil || cp["email"] != nil {
		t.Fatal("IRM block not exclusive")
	}
	payload, _ := json.Marshal(cp)
	if bytes.Contains(payload, []byte("sentinel")) || cp["oncall"].([]any)[0].(map[string]any)["url"] != nil {
		t.Fatal("provider ContactPoint redundantly transports the bearer URL")
	}
	point["oncall"].(map[string]any)["url"] = "https://%sentinel"
	_, err = renderAlertingRouting(xr, nil, nil)
	if err == nil {
		t.Fatal("malformed secret URL accepted")
	}
	b, _ := json.Marshal(err.Error())
	for i := 0; i+8 <= len(b); i++ {
		if string(b[i:i+8]) == "sentinel" {
			t.Fatal("secret-bearing parser error escaped")
		}
	}
}
