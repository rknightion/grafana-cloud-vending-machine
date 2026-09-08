package main

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
)

func TestExpiryRejectsUsageWithoutPlatformPolicy(t *testing.T) {
	claim := expiryStackDocument(map[string]any{
		"usage":  "production",
		"expiry": map[string]any{"expiresAt": "2030-01-02T00:00:00Z"},
	})
	rsp := runExpiryStack(t, claim, foundationReadyObserved(), expiryInput(nil), time.Date(2030, time.January, 1, 0, 0, 0, 0, time.UTC))

	if fatal := fatalResult(rsp); !strings.Contains(fatal, "not approved for expiry") {
		t.Fatalf("non-sandbox expiry fatal result = %q", fatal)
	}
}

func TestExpiryExtensionDeterminesEffectiveExpiryAndPreservesDeclaredProvenance(t *testing.T) {
	claim := expiryStackDocument(map[string]any{
		"usage": "sandbox",
		"expiry": map[string]any{
			"expiresAt": "2030-01-02T00:00:00Z",
			"extensions": []any{map[string]any{
				"extendedTo": "2030-01-05T00:00:00Z", "reason": "validation window",
				"requestedBy": "declared-requester", "recordedAt": "2030-01-01T12:00:00Z",
			}},
		},
	})
	rsp := runExpiryStack(t, claim, expiryDeletionPreparedObserved(), expiryInput(nil), time.Date(2030, time.January, 3, 0, 0, 0, 0, time.UTC))
	if fatal := fatalResult(rsp); fatal != "" {
		t.Fatalf("extended expiry fatal result = %q", fatal)
	}

	status := expiryStatus(t, rsp)
	if got, want := status["effectiveExpiresAt"], "2030-01-05T00:00:00Z"; got != want {
		t.Fatalf("effective expiry = %v, want %s", got, want)
	}
	if got := status["expired"]; got != false {
		t.Fatalf("expired = %v, want false", got)
	}
	for _, field := range []string{"deletionArmed", "deletionReady"} {
		if got := status[field]; got != false {
			t.Errorf("extended expiry %s = %v, want false", field, got)
		}
	}
	topLevel := nestedMap(t, rsp.GetDesired().GetComposite().GetResource().AsMap(), "status")
	for _, field := range []string{"deletionArmed", "deletionReady"} {
		if got := topLevel[field]; got != false {
			t.Errorf("extended expiry top-level %s = %v, want false", field, got)
		}
	}
	assertExpiryDeletionFields(t, rsp, false)
	if got, want := status["extensions"], []any{map[string]any{
		"extendedTo": "2030-01-05T00:00:00Z", "reason": "validation window",
		"requestedBy": "declared-requester", "recordedAt": "2030-01-01T12:00:00Z",
	}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("declared extension provenance = %#v, want %#v", got, want)
	}
}

func TestExpiryWarnsThroughExistingIncidentContact(t *testing.T) {
	claim := expiryStackDocument(map[string]any{
		"usage":  "sandbox",
		"expiry": map[string]any{"expiresAt": "2030-01-02T00:00:00Z"},
		"incidentIntegration": map[string]any{
			"enabled": true, "profile": "incident-relay", "templateMode": "enforced",
		},
	})
	rsp := runExpiryStack(t, claim, foundationReadyObserved(), expiryInput(nil), time.Date(2030, time.January, 1, 12, 0, 0, 0, time.UTC))
	if fatal := fatalResult(rsp); fatal != "" {
		t.Fatalf("warning render fatal result = %q", fatal)
	}

	rule := nestedMap(t, desiredResource(t, rsp, "expiry-warning"), "spec", "forProvider")["rule"].([]any)[0].(map[string]any)
	routing := rule["notificationSettings"].([]any)[0].(map[string]any)
	if got, want := routing["contactPoint"], "Incident relay production"; got != want {
		t.Fatalf("expiry warning contact point = %v, want %s", got, want)
	}
	if _, exists := rsp.GetDesired().GetResources()["incident-alerting-production"]; !exists {
		t.Fatal("expiry warning did not use the already-vended incident contact")
	}
}

func TestExpiryDoesNotDeleteWithoutExistingArmedDeletePath(t *testing.T) {
	claim := expiryStackDocument(map[string]any{
		"usage":  "sandbox",
		"expiry": map[string]any{"expiresAt": "2030-01-01T00:00:00Z"},
	})
	rsp := runExpiryStack(t, claim, expiryDeletionPreparedObserved(), expiryInput(nil), time.Date(2030, time.January, 2, 0, 0, 0, 0, time.UTC))
	if fatal := fatalResult(rsp); fatal != "" {
		t.Fatalf("retained expiry fatal result = %q", fatal)
	}
	if _, exists := rsp.GetDesired().GetResources()["stack"]; !exists {
		t.Fatal("expiry removed a retained stack")
	}
	assertExpiryDeletionFields(t, rsp, false)
	status := nestedMap(t, rsp.GetDesired().GetComposite().GetResource().AsMap(), "status")
	for _, field := range []string{"deletionArmed", "deletionReady"} {
		if got := status[field]; got != false {
			t.Errorf("retained expiry top-level %s = %v, want false", field, got)
		}
		if got := expiryStatus(t, rsp)[field]; got != false {
			t.Errorf("retained expiry nested %s = %v, want false", field, got)
		}
	}
}

func TestExpiryReportsReadinessWithoutSkippingReviewedDecommissionStages(t *testing.T) {
	claim := expiryStackDocument(map[string]any{
		"usage":     "sandbox",
		"expiry":    map[string]any{"expiresAt": "2030-01-01T00:00:00Z"},
		"lifecycle": map[string]any{"externalResources": "Delete"},
	})
	input := expiryInput(map[string]any{
		"deletionAuthorizations": []any{map[string]any{
			"namespace": "grafana-vending", "name": "teamdemo01", "uid": "example-request-uid", "profile": "standard",
		}},
	})

	notReady := runExpiryStack(t, claim, foundationReadyObserved(), input, time.Date(2030, time.January, 2, 0, 0, 0, 0, time.UTC))
	if fatal := fatalResult(notReady); fatal != "" {
		t.Fatalf("unready delete fatal result = %q", fatal)
	}
	if _, exists := notReady.GetDesired().GetResources()["stack"]; !exists {
		t.Fatal("expiry removed a stack before the existing deletion path was ready")
	}

	ready := foundationReadyObserved()
	ready["stack"] = observedResource(`{"status":{"atProvider":{"deleteProtection":false}}}`)
	ready["credentials"] = deletionPreparedPushSecret("teamdemo01-credentials")
	ready["fleet-management-credentials"] = deletionPreparedPushSecret("teamdemo01-fleet-management")
	ready["telemetry-credentials"] = deletionPreparedPushSecret("teamdemo01-telemetry-credentials")
	ready["stack-token"] = deletionPreparedToken("StackServiceAccountRotatingToken", "teamdemo01-admin", "serviceAccountId", "67890")
	ready["fleet-management-token"] = deletionPreparedToken("AccessPolicyRotatingToken", "teamdemo01-fleet-management", "accessPolicyId", "fleet-policy-12345")
	ready["telemetry-token"] = deletionPreparedToken("AccessPolicyRotatingToken", "teamdemo01-telemetry-publisher", "accessPolicyId", "policy-12345")
	rsp := runExpiryStack(t, claim, ready, input, time.Date(2030, time.January, 2, 0, 0, 0, 0, time.UTC))
	if fatal := fatalResult(rsp); fatal != "" {
		t.Fatalf("ready expiry fatal result = %q", fatal)
	}
	if _, exists := rsp.GetDesired().GetResources()["stack"]; !exists {
		t.Fatal("expiry skipped reviewed access-claim removal and deleted the stack")
	}
	if got := expiryStatus(t, rsp)["deletionReady"]; got != true {
		t.Fatalf("expired armed request deletion readiness = %v, want true", got)
	}
}

func TestExpiryRenderIsReproducibleWithInjectedClock(t *testing.T) {
	claim := expiryStackDocument(map[string]any{
		"usage":  "sandbox",
		"expiry": map[string]any{"expiresAt": "2030-01-02T00:00:00Z"},
	})
	now := time.Date(2030, time.January, 1, 0, 0, 0, 0, time.UTC)
	first := runExpiryStack(t, claim, foundationReadyObserved(), expiryInput(nil), now)
	second := runExpiryStack(t, claim, foundationReadyObserved(), expiryInput(nil), now)
	if diff := reflect.DeepEqual(first.GetDesired(), second.GetDesired()); !diff {
		t.Fatal("same injected reconcile time produced different desired state")
	}
}

func TestExpiryDeletionStateUsesEffectiveExpiryBoundary(t *testing.T) {
	deadline := time.Date(2030, time.January, 2, 0, 0, 0, 0, time.UTC)
	xr := resource.MustStructJSON(expiryStackDocument(map[string]any{
		"usage":     "sandbox",
		"expiry":    map[string]any{"expiresAt": deadline.Format(time.RFC3339)},
		"lifecycle": map[string]any{"externalResources": "Delete"},
	})).AsMap()
	config := resource.MustStructJSON(expiryInput(map[string]any{
		"deletionAuthorizations": []any{expiryDeletionAuthorization()},
	})).AsMap()

	for _, test := range []struct {
		name      string
		now       time.Time
		wantArmed bool
	}{
		{name: "before", now: deadline.Add(-time.Second), wantArmed: false},
		{name: "due", now: deadline, wantArmed: true},
		{name: "after", now: deadline.Add(time.Second), wantArmed: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			config[reconcileTimeConfigKey] = test.now
			armed, ready, err := expiryDeletionState(xr, nil, config)
			if err != nil {
				t.Fatalf("expiryDeletionState returned error: %v", err)
			}
			if armed != test.wantArmed {
				t.Errorf("armed = %v, want %v", armed, test.wantArmed)
			}
			if ready {
				t.Error("deletion became ready without observed deletion preparation")
			}
		})
	}
}

func TestExpiryArmsActualDeletionFieldsAtEffectiveExpiry(t *testing.T) {
	deadline := time.Date(2030, time.January, 2, 0, 0, 0, 0, time.UTC)
	claim := expiryStackDocument(map[string]any{
		"usage":     "sandbox",
		"expiry":    map[string]any{"expiresAt": deadline.Format(time.RFC3339)},
		"lifecycle": map[string]any{"externalResources": "Delete"},
	})
	input := expiryInput(map[string]any{
		"deletionAuthorizations": []any{expiryDeletionAuthorization()},
	})

	for _, test := range []struct {
		name      string
		now       time.Time
		wantArmed bool
	}{
		{name: "before", now: deadline.Add(-time.Second), wantArmed: false},
		{name: "due", now: deadline, wantArmed: true},
		{name: "after", now: deadline.Add(time.Second), wantArmed: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			rsp := runExpiryStack(t, claim, expiryDeletionPreparedObserved(), input, test.now)
			if fatal := fatalResult(rsp); fatal != "" {
				t.Fatalf("expiry render fatal result = %q", fatal)
			}
			assertExpiryDeletionFields(t, rsp, test.wantArmed)

			status := nestedMap(t, rsp.GetDesired().GetComposite().GetResource().AsMap(), "status")
			for _, field := range []string{"deletionArmed", "deletionReady"} {
				if got := status[field]; got != test.wantArmed {
					t.Errorf("top-level %s = %v, want %v", field, got, test.wantArmed)
				}
				if got := expiryStatus(t, rsp)[field]; got != test.wantArmed {
					t.Errorf("nested expiry %s = %v, want %v", field, got, test.wantArmed)
				}
			}
		})
	}
}

func TestExpiryDeleteAuthorizationIsRequiredBeforeAndAfterDeadline(t *testing.T) {
	deadline := time.Date(2030, time.January, 2, 0, 0, 0, 0, time.UTC)
	claim := expiryStackDocument(map[string]any{
		"usage":     "sandbox",
		"expiry":    map[string]any{"expiresAt": deadline.Format(time.RFC3339)},
		"lifecycle": map[string]any{"externalResources": "Delete"},
	})

	for _, timing := range []struct {
		name string
		now  time.Time
	}{
		{name: "before", now: deadline.Add(-time.Second)},
		{name: "after", now: deadline.Add(time.Second)},
	} {
		for _, authorization := range []struct {
			name  string
			input string
		}{
			{name: "missing", input: expiryInput(nil)},
			{name: "mismatched", input: expiryInput(map[string]any{
				"deletionAuthorizations": []any{map[string]any{
					"namespace": "grafana-vending", "name": "teamdemo01", "uid": "different-request-uid", "profile": "standard",
				}},
			})},
		} {
			t.Run(timing.name+"/"+authorization.name, func(t *testing.T) {
				rsp := runExpiryStack(t, claim, foundationReadyObserved(), authorization.input, timing.now)
				if fatal := fatalResult(rsp); !strings.Contains(fatal, "is not authorized to delete external resources") {
					t.Fatalf("authorization fatal result = %q", fatal)
				}
			})
		}
	}
}

func expiryStackDocument(additions map[string]any) string {
	merged := map[string]any{
		"incidentIntegration": map[string]any{
			"enabled": true, "profile": "incident-relay", "templateMode": "enforced",
		},
	}
	for key, value := range additions {
		merged[key] = value
	}
	return stackDocument(merged)
}

func expiryDeletionAuthorization() map[string]any {
	return map[string]any{
		"namespace": "grafana-vending", "name": "teamdemo01", "uid": "example-request-uid", "profile": "standard",
	}
}

func expiryDeletionPreparedObserved() map[string]*fnv1.Resource {
	observed := foundationReadyObserved()
	observed["stack"] = observedResource(`{
		"apiVersion":"cloud.grafana.m.crossplane.io/v1alpha1",
		"kind":"Stack",
		"metadata":{"name":"teamdemo01","namespace":"grafana-vending"},
		"status":{
			"atProvider":{"id":"12345","slug":"teamdemo01","url":"https://teamdemo01.grafana.net","deleteProtection":false},
			"conditions":[{"type":"Ready","status":"True","reason":"Available"}]
		}
	}`)
	observed["fleet-management-access-policy"] = observedResource(`{"status":{"atProvider":{"policyId":"fleet-policy-12345"}}}`)
	observed["telemetry-access-policy"] = observedResource(`{"status":{"atProvider":{"policyId":"policy-12345"}}}`)
	observed["credentials"] = deletionPreparedPushSecret("teamdemo01-credentials")
	observed["fleet-management-credentials"] = deletionPreparedPushSecret("teamdemo01-fleet-management")
	observed["telemetry-credentials"] = deletionPreparedPushSecret("teamdemo01-telemetry-credentials")
	observed["stack-token"] = deletionPreparedToken("StackServiceAccountRotatingToken", "teamdemo01-admin", "serviceAccountId", "67890")
	observed["fleet-management-token"] = deletionPreparedToken("AccessPolicyRotatingToken", "teamdemo01-fleet-management", "accessPolicyId", "fleet-policy-12345")
	observed["telemetry-token"] = deletionPreparedToken("AccessPolicyRotatingToken", "teamdemo01-telemetry-publisher", "accessPolicyId", "policy-12345")
	return observed
}

func assertExpiryDeletionFields(t *testing.T, rsp *fnv1.RunFunctionResponse, armed bool) {
	t.Helper()
	wantPolicies := managementPolicies
	wantDeleteProtection := true
	wantDeleteOnDestroy := false
	wantDeletionPolicy := "None"
	if armed {
		wantPolicies = []any{"*"}
		wantDeleteProtection = false
		wantDeleteOnDestroy = true
		wantDeletionPolicy = "Delete"
	}

	for _, name := range []string{"stack", "stack-service-account", "stack-token", "fleet-management-access-policy", "fleet-management-token", "telemetry-access-policy", "telemetry-token"} {
		spec := nestedMap(t, desiredResource(t, rsp, name), "spec")
		if !reflect.DeepEqual(spec["managementPolicies"], wantPolicies) {
			t.Errorf("%s managementPolicies = %#v, want %#v", name, spec["managementPolicies"], wantPolicies)
		}
	}
	if got := nestedMap(t, desiredResource(t, rsp, "stack"), "spec", "forProvider")["deleteProtection"]; got != wantDeleteProtection {
		t.Errorf("stack deleteProtection = %v, want %v", got, wantDeleteProtection)
	}
	for _, name := range []string{"stack-token", "fleet-management-token", "telemetry-token"} {
		if got := nestedMap(t, desiredResource(t, rsp, name), "spec", "forProvider")["deleteOnDestroy"]; got != wantDeleteOnDestroy {
			t.Errorf("%s deleteOnDestroy = %v, want %v", name, got, wantDeleteOnDestroy)
		}
	}
	for _, name := range []string{"credentials", "fleet-management-credentials", "telemetry-credentials"} {
		if got := nestedMap(t, desiredResource(t, rsp, name), "spec")["deletionPolicy"]; got != wantDeletionPolicy {
			t.Errorf("%s deletionPolicy = %v, want %s", name, got, wantDeletionPolicy)
		}
	}
}

func expiryInput(additions map[string]any) string {
	spec := map[string]any{
		"allowedUsages": []any{"sandbox", "production"},
		"expiryPolicies": []any{map[string]any{
			"usage": "sandbox", "warningBefore": "24h", "warningFolderUID": "vending-home-folder",
		}},
		"incidentProfiles": []any{map[string]any{
			"name": "incident-relay", "url": "https://incident-relay.example.invalid/events",
			"authorizationSecretRef": map[string]any{"name": "incident-relay", "key": "authorization"},
		}},
	}
	for key, value := range additions {
		spec[key] = value
	}
	return mustJSON(map[string]any{"spec": spec})
}

func runExpiryStack(t *testing.T, composite string, observed map[string]*fnv1.Resource, input string, now time.Time) *fnv1.RunFunctionResponse {
	t.Helper()
	input = stackInputWithDefaultOrganization(t, composite, input)
	req := &fnv1.RunFunctionRequest{
		Observed: &fnv1.State{
			Composite: &fnv1.Resource{Resource: resource.MustStructJSON(composite)},
			Resources: observed,
		},
		Input: resource.MustStructJSON(input),
	}
	rsp, err := (&Function{log: logging.NewNopLogger(), now: func() time.Time { return now }}).RunFunction(context.Background(), req)
	if err != nil {
		t.Fatalf("RunFunction returned an error: %v", err)
	}
	return rsp
}

func expiryStatus(t *testing.T, rsp *fnv1.RunFunctionResponse) map[string]any {
	t.Helper()
	status := nestedMap(t, rsp.GetDesired().GetComposite().GetResource().AsMap(), "status")
	expiry, ok := status["expiry"].(map[string]any)
	if !ok {
		t.Fatalf("expiry status = %T, want object", status["expiry"])
	}
	return expiry
}
