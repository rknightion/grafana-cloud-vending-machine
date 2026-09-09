package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestRenderOnCallStagesProviderAssignedIdentities(t *testing.T) {
	xr := onCallTestXR()
	first, err := renderOnCall(xr, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(first), 2; got != want {
		t.Fatalf("first render resources = %d, want %d responders only", got, want)
	}
	for name, child := range first {
		if child.Resource.GetKind() != "User" {
			t.Fatalf("first render %s kind = %s, want User", name, child.Resource.GetKind())
		}
		if got, want := child.Resource.GetAPIVersion(), "oncall.grafana.o.crossplane.io/v1alpha1"; got != want {
			t.Fatalf("first render %s apiVersion = %s, want %s", name, got, want)
		}
		metadata := child.Resource.UnstructuredContent()["metadata"].(map[string]any)
		if annotations, ok := metadata["annotations"].(map[string]any); ok && annotations["crossplane.io/external-name"] != nil {
			t.Fatalf("%s guessed provider-assigned external name %v", name, annotations["crossplane.io/external-name"])
		}
	}

	observed := onCallObservedResponders(t)
	second, err := renderOnCall(xr, observed, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, found := second["rotating-shift"]; !found {
		t.Fatal("observed responder identities did not unlock rotating shift")
	}
	if _, found := second["schedule"]; found {
		t.Fatal("schedule rendered before the provider assigned the shift identity")
	}

	observed["rotating-shift"] = onCallObserved("shift-id")
	third, err := renderOnCall(xr, observed, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, found := third["schedule"]; !found {
		t.Fatal("observed shift identity did not unlock schedule")
	}
	if _, found := third["escalation-chain"]; found {
		t.Fatal("chain rendered before the provider assigned the schedule identity")
	}
	shift := onCallNestedMap(t, onCallDesired(t, third, "rotating-shift"), "spec", "forProvider")
	if got, want := shift["start"], "2020-01-06T00:00:00"; got != want {
		t.Fatalf("shift start = %v, want configured stable UTC anchor %q", got, want)
	}
	rollingUsers, ok := shift["rollingUsers"].([]any)
	if !ok {
		t.Fatalf("shift rollingUsers = %T, want array", shift["rollingUsers"])
	}
	if got, want := len(rollingUsers), 2; got != want {
		t.Fatalf("rolling user group count = %d, want %d singleton groups", got, want)
	}
	for index, group := range rollingUsers {
		members, ok := group.([]any)
		if !ok || len(members) != 1 {
			t.Fatalf("rollingUsers[%d] = %#v, want exactly one responder", index, group)
		}
	}

	observed["schedule"] = onCallObserved("schedule-id")
	fourth, err := renderOnCall(xr, observed, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, found := fourth["escalation-chain"]; !found {
		t.Fatal("observed schedule identity did not unlock escalation chain")
	}
	if _, found := fourth["integration"]; found {
		t.Fatal("integration rendered before the provider assigned the chain identity")
	}

	observed["escalation-chain"] = onCallObserved("chain-id")
	fifth, err := renderOnCall(xr, observed, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []resource.Name{"schedule-escalation", "integration"} {
		if _, found := fifth[name]; !found {
			t.Fatalf("observed chain identity did not unlock %s", name)
		}
	}
	if _, found := fifth["catch-all-route"]; found {
		t.Fatal("route rendered before the provider assigned the integration identity")
	}

	observed["integration"] = onCallObserved("integration-id")
	observed["schedule-escalation"] = onCallObserved("schedule-escalation-id")
	observed["catch-all-route"] = onCallObserved("catch-all-route-id")
	final, err := renderOnCall(xr, observed, nil)
	if err != nil {
		t.Fatal(err)
	}
	route := onCallDesired(t, final, "catch-all-route")
	parameters := onCallNestedMap(t, route, "spec", "forProvider")
	if got, want := parameters["routingRegex"], ".*"; got != want {
		t.Fatalf("catch-all route regex = %v, want %q", got, want)
	}
	if got, want := parameters["integrationId"], "integration-id"; got != want {
		t.Fatalf("route integration ID = %v, want observed %q", got, want)
	}
	if got, want := parameters["escalationChainId"], "chain-id"; got != want {
		t.Fatalf("route escalation chain ID = %v, want observed %q", got, want)
	}
	escalation := onCallNestedMap(t, onCallDesired(t, final, "schedule-escalation"), "spec", "forProvider")
	if got, want := escalation["notifyOnCallFromSchedule"], "schedule-id"; got != want {
		t.Fatalf("escalation schedule ID = %v, want observed %q", got, want)
	}
	integration := onCallNestedMap(t, onCallDesired(t, final, "integration"), "spec", "forProvider")
	if got, want := integration["type"], "inbound_email"; got != want {
		t.Fatalf("integration type = %v, want %q", got, want)
	}
	for name, wantExternalName := range map[resource.Name]string{
		"rotating-shift":      "shift-id",
		"schedule":            "schedule-id",
		"escalation-chain":    "chain-id",
		"integration":         "integration-id",
		"schedule-escalation": "schedule-escalation-id",
		"catch-all-route":     "catch-all-route-id",
	} {
		child := final[name]
		metadata := child.Resource.UnstructuredContent()["metadata"].(map[string]any)
		annotations, _ := metadata["annotations"].(map[string]any)
		if got := annotations["crossplane.io/external-name"]; got != wantExternalName {
			t.Fatalf("%s external name = %v, want observed %q", name, got, wantExternalName)
		}
	}

	delete(observed, resource.Name("responder-"+stableResourceSuffix("responder-a")))
	if _, err := renderOnCall(xr, observed, nil); err == nil || !strings.Contains(err.Error(), "responder identity was lost") {
		t.Fatalf("lost observed responder error = %v, want fail-closed identity loss", err)
	}
}

func TestOnCallAdmissionRules(t *testing.T) {
	ctx := context.Background()
	env := &admissionEnv{paths: []string{"../apis/oncall-v1beta1.yaml"}}
	if err := env.Start(t); err != nil {
		t.Fatalf("start on-call admission environment: %v", err)
	}
	t.Cleanup(func() {
		if err := env.Stop(); err != nil {
			t.Errorf("stop on-call admission environment: %v", err)
		}
	})

	cases := []struct {
		name, expected string
		allow          func(*testing.T, *unstructured.Unstructured)
		mutate         func(*testing.T, *unstructured.Unstructured)
	}{
		{"schedule with no responder", "on-call schedule must have at least one reachable responder", func(t *testing.T, obj *unstructured.Unstructured) {
			t.Helper()
			setNested([]any{map[string]any{"username": "responder-b"}}, "spec", "responders")(t, obj)
		}, func(t *testing.T, obj *unstructured.Unstructured) {
			t.Helper()
			setNested([]any{}, "spec", "responders")(t, obj)
		}},
		{"chain ends at nobody", "escalation chain final step must reach the vended schedule", func(t *testing.T, obj *unstructured.Unstructured) {
			t.Helper()
			setNested("schedule", "spec", "escalation", "finalStep")(t, obj)
		}, func(t *testing.T, obj *unstructured.Unstructured) {
			t.Helper()
			setNested("nobody", "spec", "escalation", "finalStep")(t, obj)
		}},
		{"chain references unvended schedule", "escalation chain must reference the vended schedule", func(t *testing.T, obj *unstructured.Unstructured) {
			t.Helper()
			setNested("vended", "spec", "escalation", "scheduleRef", "name")(t, obj)
		}, func(t *testing.T, obj *unstructured.Unstructured) {
			t.Helper()
			setNested("outside", "spec", "escalation", "scheduleRef", "name")(t, obj)
		}},
		{"route matches no vended alert", "on-call route must match every vended alert", func(t *testing.T, obj *unstructured.Unstructured) {
			t.Helper()
			setNested("all", "spec", "route", "match")(t, obj)
		}, func(t *testing.T, obj *unstructured.Unstructured) {
			t.Helper()
			setNested("none", "spec", "route", "match")(t, obj)
		}},
	}
	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			obj := onCallRequest("admit" + string(rune('a'+index)))
			requireAdmitted(ctx, env, t, obj, "initial allowed create")
			tc.allow(t, obj)
			requireAdmitted(ctx, env, t, obj, "applicable allowed update")
			tc.mutate(t, obj)
			err := env.Apply(ctx, obj)
			if err == nil || !strings.Contains(err.Error(), tc.expected) {
				t.Fatalf("forbidden update error = %v, want own message %q", err, tc.expected)
			}
			t.Logf("refusal output: %v", err)
		})
	}

	t.Run("one owner per stack and immutable stack reference", func(t *testing.T) {
		mismatch := onCallRequest("oncall-owner")
		setNested("differentstack", "spec", "stackRef", "name")(t, mismatch)
		err := env.Apply(ctx, mismatch)
		if err == nil || !strings.Contains(err.Error(), "metadata.name must match spec.stackRef.name so one composite owns the stack on-call route") {
			t.Fatalf("mismatched stack owner error = %v", err)
		}

		immutable := onCallRequest("oncallimmutable")
		requireAdmitted(ctx, env, t, immutable, "initial stack owner")
		setNested("differentstack", "spec", "stackRef", "name")(t, immutable)
		err = env.Apply(ctx, immutable)
		if err == nil || !strings.Contains(err.Error(), "stackRef.name is immutable") {
			t.Fatalf("mutable stack reference error = %v", err)
		}
	})

	t.Run("invalid calendar date", func(t *testing.T) {
		obj := onCallRequest("badcalendar")
		setNested("2026-99-99T99:99:99", "spec", "shiftStart")(t, obj)
		if err := env.Apply(ctx, obj); err == nil {
			t.Fatal("impossible date admitted")
		} else {
			t.Logf("API-SERVER invalid calendar refused: %v", err)
		}
	})
	for _, shiftStart := range []string{"2020-01-06T00:00:00", "2026-09-07T00:00:00"} {
		t.Run("valid recurring UTC start "+shiftStart, func(t *testing.T) {
			nameSuffix := strings.NewReplacer("-", "", ":", "", "T", "").Replace(shiftStart)
			obj := onCallRequest("oncallstart" + nameSuffix)
			setNested(shiftStart, "spec", "shiftStart")(t, obj)
			requireAdmitted(ctx, env, t, obj, "valid provider-format UTC start")
		})
	}

	for _, control := range []struct {
		name, original, expected string
		mutate                   func(*testing.T, *unstructured.Unstructured)
	}{
		{"final responder control", `rule: "self.escalation.finalStep == 'schedule'"`, "escalation chain final step must reach the vended schedule", func(t *testing.T, obj *unstructured.Unstructured) {
			setNested("nobody", "spec", "escalation", "finalStep")(t, obj)
		}},
		{"vended schedule control", `rule: "self.escalation.scheduleRef.name == 'vended'"`, "escalation chain must reference the vended schedule", func(t *testing.T, obj *unstructured.Unstructured) {
			setNested("outside", "spec", "escalation", "scheduleRef", "name")(t, obj)
		}},
	} {
		t.Run("negative control "+control.name, func(t *testing.T) {
			runOnCallNegativeControl(ctx, t, control.original, control.expected, control.mutate)
		})
	}
}

func TestOnCallProviderCRDAdmission(t *testing.T) {
	env := &admissionEnv{paths: []string{"../apis/oncall-v1beta1.yaml"}}
	if err := env.Start(t); err != nil {
		t.Fatalf("start on-call provider admission environment: %v", err)
	}
	t.Cleanup(func() {
		if err := env.Stop(); err != nil {
			t.Errorf("stop on-call provider admission environment: %v", err)
		}
	})
	installProviderAdmissionCRDs(t, env)

	xr := onCallTestXR()
	observed := map[resource.Name]resource.ObservedComposed{}
	for _, stage := range []struct {
		name    string
		observe map[resource.Name]string
	}{
		{"responder lookup", map[resource.Name]string{
			resource.Name("responder-" + stableResourceSuffix("responder-a")): "user-id-responder-a",
			resource.Name("responder-" + stableResourceSuffix("responder-b")): "user-id-responder-b",
		}},
		{"rotating shift", map[resource.Name]string{"rotating-shift": "shift-id"}},
		{"schedule", map[resource.Name]string{"schedule": "schedule-id"}},
		{"escalation chain", map[resource.Name]string{"escalation-chain": "chain-id"}},
		{"integration", map[resource.Name]string{"integration": "integration-id"}},
	} {
		t.Run(stage.name, func(t *testing.T) {
			desired, err := renderOnCall(xr, observed, nil)
			if err != nil {
				t.Fatal(err)
			}
			admitProviderChildren(t, env, desired)
			for name, externalName := range stage.observe {
				observed[name] = onCallObserved(externalName)
			}
		})
	}

	desired, err := renderOnCall(xr, observed, nil)
	if err != nil {
		t.Fatal(err)
	}
	admitProviderChildren(t, env, desired)
}

func runOnCallNegativeControl(ctx context.Context, t *testing.T, originalRule, expectedMessage string, mutate func(*testing.T, *unstructured.Unstructured)) {
	t.Helper()
	const xrdPath = "../apis/oncall-v1beta1.yaml"
	source, err := os.ReadFile(xrdPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(source), originalRule); got != 1 {
		t.Fatalf("scratch control source contains %d copies of %q", got, originalRule)
	}
	scratchDir := t.TempDir()
	scratchPath := filepath.Join(scratchDir, "oncall-v1beta1.yaml")
	weakened := strings.Replace(string(source), originalRule, `rule: "true"`, 1)
	if err := os.WriteFile(scratchPath, []byte(weakened), 0o600); err != nil {
		t.Fatal(err)
	}
	weakenedEnv := &admissionEnv{paths: []string{scratchPath}}
	if err := weakenedEnv.Start(t); err != nil {
		t.Fatalf("start weakened control plane: %v", err)
	}
	weakenedObj := onCallRequest("weakened")
	mutate(t, weakenedObj)
	if err := weakenedEnv.Apply(ctx, weakenedObj); err != nil {
		t.Fatalf("weakened schema refused the forbidden request: %v", err)
	}
	t.Log("weakened control output: admitted")
	if err := weakenedEnv.Stop(); err != nil {
		t.Fatal(err)
	}

	restoredEnv := &admissionEnv{paths: []string{xrdPath}}
	if err := restoredEnv.Start(t); err != nil {
		t.Fatalf("start restored control plane: %v", err)
	}
	restoredObj := onCallRequest("restored")
	mutate(t, restoredObj)
	err = restoredEnv.Apply(ctx, restoredObj)
	if err == nil || !strings.Contains(err.Error(), expectedMessage) {
		t.Fatalf("restored schema error = %v, want own message %q", err, expectedMessage)
	}
	t.Logf("restored control output: %v", err)
	if err := restoredEnv.Stop(); err != nil {
		t.Fatal(err)
	}
}

func onCallTestXR() map[string]any {
	return map[string]any{
		"metadata": map[string]any{"name": "stackdemo", "namespace": "grafana-vending"},
		"spec": map[string]any{
			"stackRef":   map[string]any{"name": "stackdemo"},
			"responders": []any{map[string]any{"username": "responder-a"}, map[string]any{"username": "responder-b"}},
			"shiftStart": "2020-01-06T00:00:00",
		},
	}
}

func onCallObservedResponders(t *testing.T) map[resource.Name]resource.ObservedComposed {
	t.Helper()
	result := map[resource.Name]resource.ObservedComposed{}
	for _, username := range []string{"responder-a", "responder-b"} {
		key := resource.Name("responder-" + stableResourceSuffix(username))
		result[key] = onCallObserved("user-id-" + username)
	}
	return result
}

func onCallObserved(externalName string) resource.ObservedComposed {
	r := composed.New()
	r.SetUnstructuredContent(map[string]any{"metadata": map[string]any{"annotations": map[string]any{"crossplane.io/external-name": externalName}}})
	return resource.ObservedComposed{Resource: r}
}

func onCallDesired(t *testing.T, desired map[resource.Name]*resource.DesiredComposed, name resource.Name) map[string]any {
	t.Helper()
	child, found := desired[name]
	if !found {
		t.Fatalf("desired child %q was not rendered", name)
	}
	return child.Resource.UnstructuredContent()
}

func onCallNestedMap(t *testing.T, object map[string]any, fields ...string) map[string]any {
	t.Helper()
	current := object
	for _, field := range fields {
		next, ok := current[field].(map[string]any)
		if !ok {
			t.Fatalf("field %s is %T, want object", field, current[field])
		}
		current = next
	}
	return current
}

func onCallRequest(name string) *unstructured.Unstructured {
	return requestObject("GrafanaOnCall", name, map[string]any{
		"stackRef":   map[string]any{"name": name},
		"responders": []any{map[string]any{"username": "responder-a"}},
		"shiftStart": "2020-01-06T00:00:00",
		"escalation": map[string]any{"finalStep": "schedule", "scheduleRef": map[string]any{"name": "vended"}},
		"route":      map[string]any{"match": "all"},
	})
}
