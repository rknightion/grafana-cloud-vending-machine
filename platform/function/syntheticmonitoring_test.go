package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestSyntheticMonitoringWaitsForTrustedBootstrapContext(t *testing.T) {
	desired, err := renderSyntheticMonitoring(syntheticMonitoringClaim(), nil, syntheticMonitoringPlatformConfig(false))
	if err != nil {
		t.Fatalf("renderSyntheticMonitoring returned an error: %v", err)
	}
	if len(desired) != 0 {
		t.Fatalf("rendered %d resources without trusted bootstrap context, want none", len(desired))
	}
}

func TestSyntheticMonitoringPersistsOnlyDerivedCredential(t *testing.T) {
	observed := map[resource.Name]resource.ObservedComposed{
		"synthetic-monitoring-installation": syntheticMonitoringObserved(`{"status":{"conditions":[{"type":"Ready","status":"True"}],"atProvider":{"stackId":"12345","stackSmApiUrl":"https://synthetic-monitoring-api.example.com","smAccessToken":"example-derived-token"}}}`),
	}
	desired, err := renderSyntheticMonitoring(syntheticMonitoringClaim(), observed, syntheticMonitoringPlatformConfig(true))
	if err != nil {
		t.Fatalf("renderSyntheticMonitoring returned an error: %v", err)
	}

	installation := syntheticMonitoringObject(t, desired, "synthetic-monitoring-installation")
	parameters := syntheticMonitoringNestedMap(t, installation, "spec", "forProvider")
	bootstrap := parameters["metricsPublisherKeySecretRef"].(map[string]any)
	if got, want := bootstrap["name"], "teamdemo01-telemetry-token"; got != want {
		t.Fatalf("installation bootstrap Secret = %v, want %v", got, want)
	}
	if got, want := bootstrap["key"], "attribute.token"; got != want {
		t.Fatalf("installation bootstrap key = %v, want %v", got, want)
	}

	derivedSecret := syntheticMonitoringObject(t, desired, "synthetic-monitoring-derived-credential")
	if got := derivedSecret["kind"]; got != "Secret" {
		t.Fatalf("derived credential materializer kind = %v, want Secret", got)
	}
	if _, exists := derivedSecret["spec"]; exists {
		t.Fatal("core Secret fields were incorrectly nested under spec")
	}
	if data, ok := derivedSecret["data"].(map[string]any); !ok || len(data) != 2 {
		t.Fatalf("derived credential Secret data = %T %v, want two base64 values", derivedSecret["data"], derivedSecret["data"])
	}
	derived := syntheticMonitoringObject(t, desired, "synthetic-monitoring-credential-publish")
	selector := syntheticMonitoringNestedMap(t, derived, "spec", "selector", "secret")
	if got, want := selector["name"], "teamdemo01-synthetic-monitoring-credentials"; got != want {
		t.Fatalf("persisted Secret source = %v, want derived credential Secret %v", got, want)
	}
	for name, child := range desired {
		if child.Resource.GetKind() != "PushSecret" {
			continue
		}
		secret := syntheticMonitoringNestedMap(t, child.Resource.UnstructuredContent(), "spec", "selector", "secret")
		if secret["name"] == "teamdemo01-telemetry-token" {
			t.Fatalf("PushSecret %q persists the bootstrap credential", name)
		}
	}
}

func TestSyntheticMonitoringMarksDerivedSecretReadyOnlyAfterExactObservation(t *testing.T) {
	observed := map[resource.Name]resource.ObservedComposed{
		"synthetic-monitoring-installation": syntheticMonitoringObserved(`{"status":{"conditions":[{"type":"Ready","status":"True"}],"atProvider":{"stackId":"12345","stackSmApiUrl":"https://synthetic-monitoring-api.example.com","smAccessToken":"example-derived-token"}}}`),
	}
	desired, err := renderSyntheticMonitoring(syntheticMonitoringClaim(), observed, syntheticMonitoringPlatformConfig(true))
	if err != nil {
		t.Fatalf("renderSyntheticMonitoring returned an error: %v", err)
	}
	if got := desired["synthetic-monitoring-derived-credential"].Ready; got == resource.ReadyTrue {
		t.Fatal("derived Secret was marked ready before it was observed")
	}

	secret := desired["synthetic-monitoring-derived-credential"].Resource.UnstructuredContent()
	observed["synthetic-monitoring-derived-credential"] = syntheticMonitoringObservedMap(secret)
	desired, err = renderSyntheticMonitoring(syntheticMonitoringClaim(), observed, syntheticMonitoringPlatformConfig(true))
	if err != nil {
		t.Fatalf("renderSyntheticMonitoring with observed Secret returned an error: %v", err)
	}
	if got := desired["synthetic-monitoring-derived-credential"].Ready; got != resource.ReadyTrue {
		t.Fatalf("exactly observed derived Secret readiness = %v, want ReadyTrue", got)
	}
}

func TestSyntheticMonitoringDoesNotTrustInstallationConditionOrPublicProbeInventory(t *testing.T) {
	observed := map[resource.Name]resource.ObservedComposed{
		"synthetic-monitoring-installation": syntheticMonitoringObserved(`{"status":{"conditions":[{"type":"Ready","status":"True"}],"atProvider":{"stackId":"12345","stackSmApiUrl":"https://synthetic-monitoring-api.example.com","smAccessToken":"example-derived-token"}}}`),
		"synthetic-monitoring-probe-set":    syntheticMonitoringObserved(`{"status":{"conditions":[{"type":"Ready","status":"True"}],"atProvider":{"probes":{"public-one":"1"}}}}`),
	}
	if syntheticMonitoringInstallationVerified(observed, "12345") {
		t.Fatal("Installation Ready or public ProbeSet inventory was accepted as installation evidence")
	}
}

func TestSyntheticMonitoringVerificationIsBoundToReferencedStackIdentity(t *testing.T) {
	observed := syntheticMonitoringVerifiedObserved("67890")
	if !syntheticMonitoringInstallationVerified(observed, "12345") {
		t.Fatal("tenant-scoped disabled Check observation from the stack-bound derived credential was not accepted")
	}
	if syntheticMonitoringInstallationVerified(observed, "54321") {
		t.Fatal("verification was accepted for a different installation stack identity")
	}
	rotated := syntheticMonitoringVerifiedObserved("67890")
	rotated["synthetic-monitoring-installation"] = syntheticMonitoringObserved(`{"status":{"conditions":[{"type":"Ready","status":"True"}],"atProvider":{"stackId":"12345","stackSmApiUrl":"https://synthetic-monitoring-api.example.com","smAccessToken":"rotated-derived-token"}}}`)
	if syntheticMonitoringInstallationVerified(rotated, "12345") {
		t.Fatal("verification observed through an older derived credential was accepted after credential rotation")
	}

	_, err := renderSyntheticMonitoring(syntheticMonitoringClaim(), observed, syntheticMonitoringPlatformConfig(true))
	if err != nil {
		t.Fatalf("matching verification returned an error: %v", err)
	}

	mismatched := syntheticMonitoringVerifiedObserved("67890")
	mismatched["synthetic-monitoring-installation"] = syntheticMonitoringObserved(`{"status":{"conditions":[{"type":"Ready","status":"True"}],"atProvider":{"stackId":"54321","stackSmApiUrl":"https://synthetic-monitoring-api.example.com","smAccessToken":"example-derived-token"}}}`)
	_, err = renderSyntheticMonitoring(syntheticMonitoringClaim(), mismatched, syntheticMonitoringPlatformConfig(true))
	if err == nil || !strings.Contains(err.Error(), "installation stack 54321 does not match referenced stack 12345") {
		t.Fatalf("mismatched installation error = %v, want fail-closed identity rejection", err)
	}
}

func TestSyntheticMonitoringBudgetRejectsExpensiveCheckSets(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{
			name: "API check count",
			mutate: func(claim map[string]any) {
				checks := claim["spec"].(map[string]any)["checks"].([]any)
				claim["spec"].(map[string]any)["checks"] = append(checks, syntheticMonitoringHTTPCheck("second-api", 60, "public-one"))
			},
			want: "API check count 2 exceeds platform maximum 1",
		},
		{
			name: "browser check count",
			mutate: func(claim map[string]any) {
				checks := claim["spec"].(map[string]any)["checks"].([]any)
				claim["spec"].(map[string]any)["checks"] = append(checks, syntheticMonitoringBrowserCheck("second-browser", 600, "public-one"))
			},
			want: "browser check count 2 exceeds platform maximum 1",
		},
		{
			name: "probe locations",
			mutate: func(claim map[string]any) {
				check := claim["spec"].(map[string]any)["checks"].([]any)[0].(map[string]any)
				check["probeNames"] = []any{"public-one", "public-two", "public-three"}
			},
			want: "uses 3 probe locations; platform maximum is 2",
		},
		{
			name: "frequency",
			mutate: func(claim map[string]any) {
				claim["spec"].(map[string]any)["checks"].([]any)[0].(map[string]any)["frequencySeconds"] = 10
			},
			want: "runs every 10 seconds; platform minimum interval is 60",
		},
		{
			name: "weighted executions",
			mutate: func(claim map[string]any) {
				claim["spec"].(map[string]any)["checks"].([]any)[1].(map[string]any)["probeNames"] = []any{"public-one", "public-two"}
				claim["spec"].(map[string]any)["checks"].([]any)[1].(map[string]any)["frequencySeconds"] = 60
			},
			want: "weighted executions per hour 1206 exceeds platform maximum 700",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			claim := syntheticMonitoringClaim()
			tc.mutate(claim)
			_, err := renderSyntheticMonitoring(claim, nil, syntheticMonitoringPlatformConfig(true))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("budget error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestSyntheticMonitoringBudgetRemainsBoundedWithCheckAlerts(t *testing.T) {
	claim := syntheticMonitoringClaim()
	checks := claim["spec"].(map[string]any)["checks"].([]any)
	claim["spec"].(map[string]any)["checks"] = append(checks, syntheticMonitoringHTTPCheck("second-api", 60, "public-one"))

	config := syntheticMonitoringPlatformConfig(true)
	_, err := renderSyntheticMonitoring(claim, nil, config)
	if err == nil || !strings.Contains(err.Error(), "API check count 2 exceeds platform maximum 1") {
		t.Fatalf("baseline renderer budget error = %v, want API check cap refusal", err)
	}
	t.Logf("renderer-only baseline budget control: %v", err)

	profile := config["spec"].(map[string]any)["syntheticMonitoringBudgets"].([]any)[0].(map[string]any)
	profile["maxApiChecks"] = 2
	if _, err := renderSyntheticMonitoring(claim, nil, config); err != nil {
		t.Fatalf("weakened renderer budget error = %v, want admission", err)
	}
	t.Log("renderer-only weakened budget control: admitted")

	profile["maxApiChecks"] = 1
	_, err = renderSyntheticMonitoring(claim, nil, config)
	if err == nil || !strings.Contains(err.Error(), "API check count 2 exceeds platform maximum 1") {
		t.Fatalf("restored renderer budget error = %v, want API check cap refusal", err)
	}
	t.Logf("renderer-only restored budget control: %v", err)
}

func TestSyntheticMonitoringRejectsDuplicateChecksAndUnsupportedSettings(t *testing.T) {
	duplicate := syntheticMonitoringClaim()
	checks := duplicate["spec"].(map[string]any)["checks"].([]any)
	duplicate["spec"].(map[string]any)["checks"] = append(checks, syntheticMonitoringHTTPCheck("api-home", 600, "public-one"))
	_, err := renderSyntheticMonitoring(duplicate, nil, syntheticMonitoringPlatformConfig(true))
	if err == nil || !strings.Contains(err.Error(), "repeats check name") {
		t.Fatalf("duplicate check error = %v, want one-owner rejection", err)
	}

	unsupported := syntheticMonitoringClaim()
	unsupported["spec"].(map[string]any)["checks"].([]any)[0].(map[string]any)["type"] = "scripted"
	_, err = renderSyntheticMonitoring(unsupported, nil, syntheticMonitoringPlatformConfig(true))
	if err == nil || !strings.Contains(err.Error(), "unsupported check type") {
		t.Fatalf("unsupported check error = %v, want fail-closed refusal", err)
	}
}

func TestSyntheticMonitoringWithholdsTeamChecksUntilIdentityVerification(t *testing.T) {
	observed := map[resource.Name]resource.ObservedComposed{
		"synthetic-monitoring-installation": syntheticMonitoringObserved(`{"status":{"conditions":[{"type":"Ready","status":"True"}],"atProvider":{"stackId":"12345","stackSmApiUrl":"https://synthetic-monitoring-api.example.com","smAccessToken":"example-derived-token"}}}`),
	}
	desired, err := renderSyntheticMonitoring(syntheticMonitoringClaim(), observed, syntheticMonitoringPlatformConfig(true))
	if err != nil {
		t.Fatalf("renderSyntheticMonitoring returned an error: %v", err)
	}
	if _, exists := desired["check-api-home"]; exists {
		t.Fatal("team check was admitted before independent installation verification")
	}
	verifierName := syntheticMonitoringVerifierResourceName("example-derived-token")
	if _, exists := desired[verifierName]; !exists {
		t.Fatal("identity-bound verification Check was not rendered once derived credentials were ready")
	}
	verifier := syntheticMonitoringObject(t, desired, verifierName)
	if got := verifier["kind"]; got != "Check" {
		t.Fatalf("verifier kind = %v, want Check", got)
	}
	verifierParameters := syntheticMonitoringNestedMap(t, verifier, "spec", "forProvider")
	if verifierParameters["enabled"] != false || verifierParameters["target"] != "https://synthetic-monitoring-verification.invalid" {
		t.Fatalf("verifier enabled/target = %v/%v, want disabled reserved target", verifierParameters["enabled"], verifierParameters["target"])
	}

	verified, err := renderSyntheticMonitoring(syntheticMonitoringClaim(), syntheticMonitoringVerifiedObserved("67890"), syntheticMonitoringPlatformConfig(true))
	if err != nil {
		t.Fatalf("verified render returned an error: %v", err)
	}
	for _, name := range []resource.Name{"check-api-home", "check-browser-home"} {
		if _, exists := verified[name]; !exists {
			t.Errorf("verified render did not admit %q", name)
		}
	}
	for _, name := range []resource.Name{"check-alerts-api-home", "check-alerts-browser-home"} {
		if _, exists := verified[name]; exists {
			t.Errorf("rendered %q before the provider reported a check ID", name)
		}
	}
	for name, child := range verified {
		if child.Resource.GetKind() == "Probe" {
			t.Fatalf("rendered private Probe %q; private probes and their tokens are outside this API", name)
		}
	}
}

func TestSyntheticMonitoringRendersCheckAlertsOnlyFromObservedCheckIDs(t *testing.T) {
	observed := syntheticMonitoringVerifiedObserved("67890")
	observed["check-api-home"] = syntheticMonitoringObserved(`{"status":{"atProvider":{"id":"2468"}}}`)
	observed["check-browser-home"] = syntheticMonitoringObserved(`{"status":{"atProvider":{"id":"not-a-number"}}}`)
	desired, err := renderSyntheticMonitoring(syntheticMonitoringClaim(), observed, syntheticMonitoringPlatformConfig(true))
	if err != nil {
		t.Fatalf("renderSyntheticMonitoring returned an error: %v", err)
	}
	alerts := syntheticMonitoringObject(t, desired, "check-alerts-api-home")
	if got, want := alerts["apiVersion"], "sm.grafana.m.crossplane.io/v1alpha1"; got != want {
		t.Fatalf("CheckAlerts apiVersion = %v, want %s", got, want)
	}
	if got, want := alerts["kind"], "CheckAlerts"; got != want {
		t.Fatalf("rendered kind = %v, want %s", got, want)
	}
	metadata := syntheticMonitoringNestedMap(t, alerts, "metadata")
	annotations := syntheticMonitoringNestedMap(t, metadata, "annotations")
	if got, want := annotations["crossplane.io/external-name"], "2468"; got != want {
		t.Fatalf("CheckAlerts external name = %v, want observed ID %s", got, want)
	}
	parameters := syntheticMonitoringNestedMap(t, alerts, "spec", "forProvider")
	if got, want := parameters["checkId"], float64(2468); got != want {
		t.Fatalf("CheckAlerts checkId = %v, want %v", got, want)
	}
	wantAlerts := []any{map[string]any{"name": "ProbeFailedExecutionsTooHigh", "period": "15m", "runbookUrl": "https://runbooks.example.invalid/synthetic-monitoring", "threshold": float64(1)}}
	if got, want := parameters["alerts"], wantAlerts; !syntheticMonitoringEqual(got, want) {
		t.Fatalf("CheckAlerts alerts = %#v, want %#v", got, want)
	}
	if _, exists := desired["check-alerts-browser-home"]; exists {
		t.Fatal("CheckAlerts rendered from an invalid provider-assigned check ID")
	}
}

func TestSyntheticMonitoringRejectsPrivateProbesWithoutRenderingTokens(t *testing.T) {
	claim := syntheticMonitoringClaim()
	claim["spec"].(map[string]any)["privateProbes"] = []any{map[string]any{"name": "private-one"}}
	desired, err := renderSyntheticMonitoring(claim, syntheticMonitoringVerifiedObserved("67890"), syntheticMonitoringPlatformConfig(true))
	if err == nil || !strings.Contains(err.Error(), privateProbeUnsupportedMessage) {
		t.Fatalf("private probe error = %v, want %q", err, privateProbeUnsupportedMessage)
	}
	if desired != nil {
		t.Fatalf("private probe request rendered %d resources", len(desired))
	}
}

func TestSyntheticMonitoringAdmissionRejectsPrivateProbesOnCreateAndUpdate(t *testing.T) {
	env := &admissionEnv{paths: []string{"../apis/synthetic-monitoring-v1beta1.yaml"}}
	if err := env.Start(t); err != nil {
		t.Fatalf("start synthetic monitoring admission environment: %v", err)
	}
	t.Cleanup(func() {
		if err := env.Stop(); err != nil {
			t.Errorf("stop synthetic monitoring admission environment: %v", err)
		}
	})
	ctx := context.Background()
	allowed := syntheticMonitoringAdmissionRequest("smadmission")
	if err := env.Apply(ctx, allowed); err != nil {
		t.Fatalf("allowed private-probe control create was refused: %v", err)
	}

	for _, tc := range []struct {
		name string
		obj  *unstructured.Unstructured
	}{
		{name: "create", obj: syntheticMonitoringAdmissionRequest("smprivatecreate")},
		{name: "update", obj: allowed.DeepCopy()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := unstructured.SetNestedSlice(tc.obj.Object, []any{map[string]any{"name": "private-one"}}, "spec", "privateProbes"); err != nil {
				t.Fatal(err)
			}
			err := env.Apply(ctx, tc.obj)
			if err == nil || !strings.Contains(err.Error(), privateProbeUnsupportedMessage) {
				t.Fatalf("private-probe %s error = %v, want %q", tc.name, err, privateProbeUnsupportedMessage)
			}
			t.Logf("private-probe %s refused: %v", tc.name, err)
		})
	}
}

func TestSyntheticMonitoringNeverPublishesProbeTokens(t *testing.T) {
	const probeToken = "probe-token-value-that-must-not-publish"
	privateProbeRequest := syntheticMonitoringClaim()
	privateProbeRequest["spec"].(map[string]any)["privateProbes"] = []any{map[string]any{"name": "private-one", "token": probeToken}}
	observed := syntheticMonitoringVerifiedObserved("67890")
	observed["check-api-home"] = syntheticMonitoringObserved(`{"status":{"atProvider":{"id":"2468"}}}`)
	desired, err := renderSyntheticMonitoring(privateProbeRequest, observed, syntheticMonitoringPlatformConfig(true))
	if err == nil || !strings.Contains(err.Error(), privateProbeUnsupportedMessage) {
		t.Fatalf("private probe token request error = %v, want %q", err, privateProbeUnsupportedMessage)
	}
	if desired != nil {
		t.Fatalf("private probe token request rendered %d resources", len(desired))
	}

	desired, err = renderSyntheticMonitoring(syntheticMonitoringClaim(), observed, syntheticMonitoringPlatformConfig(true))
	if err != nil {
		t.Fatal(err)
	}
	for name, child := range desired {
		encoded, err := json.Marshal(child.Resource.UnstructuredContent())
		if err != nil {
			t.Fatalf("encode %s: %v", name, err)
		}
		if strings.Contains(string(encoded), probeToken) {
			t.Fatalf("rendered %s publishes a probe token: %s", name, encoded)
		}
	}
	status := desiredSyntheticMonitoringStatus(privateProbeRequest, observed, syntheticMonitoringPlatformConfig(true)).Resource.UnstructuredContent()
	encoded, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), probeToken) {
		t.Fatalf("status publishes a probe token: %s", encoded)
	}
	example, err := os.ReadFile("../../examples/catalog/synthetic-monitoring/synthetic-monitoring.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(example), probeToken) || strings.Contains(strings.ToLower(string(example)), "privateprobes") {
		t.Fatalf("catalog example publishes a private probe token request: %s", example)
	}
}

func TestSyntheticMonitoringStatusPublishesBudgetAndNeverTokens(t *testing.T) {
	status := desiredSyntheticMonitoringStatus(syntheticMonitoringClaim(), syntheticMonitoringVerifiedObserved("67890"), syntheticMonitoringPlatformConfig(true))
	content := status.Resource.UnstructuredContent()
	statusFields := syntheticMonitoringNestedMap(t, content, "status")
	if got := statusFields["installationVerified"]; got != true {
		t.Fatalf("installationVerified = %v, want true", got)
	}
	budget := syntheticMonitoringNestedMap(t, content, "status", "budget")
	if got, want := budget["weightedExecutionsPerHour"], 66; got != want {
		t.Fatalf("weighted executions = %v, want %v", got, want)
	}
	encoded, err := json.Marshal(statusFields)
	if err != nil {
		t.Fatalf("cannot encode status: %v", err)
	}
	if strings.Contains(strings.ToLower(string(encoded)), "token") {
		t.Fatalf("composite status contains a token field: %s", encoded)
	}
}

func syntheticMonitoringClaim() map[string]any {
	return map[string]any{
		"apiVersion": "platform.example.org/v1beta1",
		"kind":       "GrafanaSyntheticMonitoring",
		"metadata": map[string]any{
			"name":      "teamdemo01",
			"namespace": "grafana-vending",
		},
		"spec": map[string]any{
			"stackRef": map[string]any{"name": "teamdemo01"},
			"checks": []any{
				syntheticMonitoringHTTPCheck("api-home", 600, "public-one"),
				syntheticMonitoringBrowserCheck("browser-home", 600, "public-one"),
			},
		},
	}
}

func syntheticMonitoringAdmissionRequest(name string) *unstructured.Unstructured {
	return requestObject("GrafanaSyntheticMonitoring", name, map[string]any{
		"stackRef": map[string]any{"name": name},
		"checks": []any{map[string]any{
			"name": "api-home", "type": "http", "target": "https://service.example.invalid/health",
			"frequencySeconds": float64(300), "probeNames": []any{"public-one"}, "alerts": syntheticMonitoringAlertsClaim(),
			"http": map[string]any{"method": "GET"},
		}},
	})
}

func syntheticMonitoringHTTPCheck(name string, frequency int, probes ...string) map[string]any {
	return map[string]any{
		"name": name, "type": "http", "target": "https://service.example.com/health",
		"frequencySeconds": frequency, "probeNames": syntheticMonitoringStrings(probes),
		"alerts": syntheticMonitoringAlertsClaim(),
		"http":   map[string]any{"method": "GET"},
	}
}

func syntheticMonitoringBrowserCheck(name string, frequency int, probes ...string) map[string]any {
	return map[string]any{
		"name": name, "type": "browser", "target": "https://service.example.com",
		"frequencySeconds": frequency, "probeNames": syntheticMonitoringStrings(probes),
		"alerts":  syntheticMonitoringAlertsClaim(),
		"browser": map[string]any{"script": "export default function () {}"},
	}
}

func syntheticMonitoringAlertsClaim() []any {
	return []any{map[string]any{
		"name": "ProbeFailedExecutionsTooHigh", "period": "15m", "threshold": float64(1),
		"runbookURL": "https://runbooks.example.invalid/synthetic-monitoring",
	}}
}

func syntheticMonitoringEqual(got, want any) bool {
	gotJSON, gotErr := json.Marshal(got)
	wantJSON, wantErr := json.Marshal(want)
	return gotErr == nil && wantErr == nil && string(gotJSON) == string(wantJSON)
}

func syntheticMonitoringStrings(values []string) []any {
	result := make([]any, len(values))
	for i, value := range values {
		result[i] = value
	}
	return result
}

func syntheticMonitoringPlatformConfig(withContext bool) map[string]any {
	config := map[string]any{
		"spec": map[string]any{
			"outputSecretPrefix": "/example/platform/grafana/stacks",
			"secretStoreRef":     map[string]any{"name": "grafana-vending-secrets", "kind": "SecretStore"},
			"organizations": []any{map[string]any{
				"name": "example-primary", "providerConfigName": "grafana-cloud-org-example-primary",
				"allowedRegions": []any{"prod-us-central-0"}, "allowedUsages": []any{"development"},
			}},
			"syntheticMonitoringBudgets": []any{map[string]any{
				"usage": "development", "maxApiChecks": 1, "maxBrowserChecks": 1,
				"maxProbeLocationsPerCheck": 2, "minCheckIntervalSeconds": 60,
				"browserExecutionWeight": 10, "maxWeightedExecutionsPerHour": 700,
				"verificationProbeName": "public-one",
			}},
		},
	}
	if withContext {
		config["referencedStack"] = map[string]any{
			"name": "teamdemo01", "namespace": "grafana-vending", "uid": "example-stack-uid",
			"stackID": "12345", "region": "prod-us-central-0", "organization": "example-primary",
			"usage": "development", "outputSecretPath": "/example/platform/grafana/stacks/example-primary/development/teamdemo01",
			"bootstrapSecretRef": map[string]any{"name": "teamdemo01-telemetry-token", "key": "attribute.token", "ready": true},
		}
	}
	return config
}

func syntheticMonitoringVerifiedObserved(tenantID string) map[resource.Name]resource.ObservedComposed {
	return map[resource.Name]resource.ObservedComposed{
		"synthetic-monitoring-installation":                              syntheticMonitoringObserved(`{"status":{"conditions":[{"type":"Ready","status":"True"}],"atProvider":{"stackId":"12345","stackSmApiUrl":"https://synthetic-monitoring-api.example.com","smAccessToken":"example-derived-token"}}}`),
		syntheticMonitoringVerifierResourceName("example-derived-token"): syntheticMonitoringObserved(`{"status":{"conditions":[{"type":"Ready","status":"True"}],"atProvider":{"tenantId":` + tenantID + `}}}`),
	}
}

func syntheticMonitoringObserved(raw string) resource.ObservedComposed {
	r := composed.New()
	r.SetUnstructuredContent(resource.MustStructJSON(raw).AsMap())
	return resource.ObservedComposed{Resource: r}
}

func syntheticMonitoringObservedMap(object map[string]any) resource.ObservedComposed {
	r := composed.New()
	r.SetUnstructuredContent(object)
	return resource.ObservedComposed{Resource: r}
}

func syntheticMonitoringObject(t *testing.T, desired map[resource.Name]*resource.DesiredComposed, name resource.Name) map[string]any {
	t.Helper()
	child, ok := desired[name]
	if !ok {
		t.Fatalf("desired resource %q was not rendered", name)
	}
	return child.Resource.UnstructuredContent()
}

func syntheticMonitoringNestedMap(t *testing.T, object map[string]any, fields ...string) map[string]any {
	t.Helper()
	current := object
	for _, field := range fields {
		next, ok := current[field].(map[string]any)
		if !ok {
			t.Fatalf("field %q in path %v is %T, want object", field, fields, current[field])
		}
		current = next
	}
	return current
}

func TestSyntheticMonitoringExistingCheckCanOmitAlerts(t *testing.T) {
	alerts, err := syntheticMonitoringAlerts(map[string]any{"name": "legacy"}, "legacy")
	if err != nil || alerts != nil {
		t.Fatalf("legacy check changed: alerts=%v error=%v", alerts, err)
	}
	if _, err := syntheticMonitoringAlerts(map[string]any{"alerts": []any{}}, "invalid"); err == nil {
		t.Fatal("explicit empty alert set admitted")
	}
	minimal, err := syntheticMonitoringAlerts(map[string]any{"alerts": []any{map[string]any{"name": "ProbeFailedExecutionsTooHigh", "threshold": float64(1)}}}, "minimal")
	if err != nil {
		t.Fatal(err)
	}
	entry := minimal[0].(map[string]any)
	for _, optional := range []string{"period", "runbookUrl"} {
		if _, found := entry[optional]; found {
			t.Fatalf("minimal alert emitted omitted optional field %q", optional)
		}
	}
}
