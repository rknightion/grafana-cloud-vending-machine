package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/crossplane/function-sdk-go/resource"
	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestServiceAccountsRenderObservedIDsAndDerivedCredentials(t *testing.T) {
	observed := map[resource.Name]resource.ObservedComposed{
		"service-account-ci-runner": observedComposed(`{"status":{"atProvider":{"id":"101"}}}`),
		"service-account-terraform": observedComposed(`{"status":{"atProvider":{"id":"102"}}}`),
	}
	desired, err := renderServiceAccounts(serviceAccountsClaim(), observed, serviceAccountsConfig("336h", "336h"))
	if err != nil {
		t.Fatalf("renderServiceAccounts returned error: %v", err)
	}
	if got, want := len(desired), 8; got != want {
		t.Fatalf("rendered resources = %d, want %d", got, want)
	}

	for logicalName, wantKind := range map[resource.Name]string{
		"service-account-ci-runner":             "ServiceAccount",
		"service-account-ci-runner-token":       "ServiceAccountRotatingToken",
		"service-account-ci-runner-permissions": "ServiceAccountPermission",
		"service-account-ci-runner-credentials": "PushSecret",
		"service-account-terraform":             "ServiceAccount",
		"service-account-terraform-token":       "ServiceAccountRotatingToken",
		"service-account-terraform-permissions": "ServiceAccountPermission",
		"service-account-terraform-credentials": "PushSecret",
	} {
		if got := serviceAccountsDesired(t, desired, logicalName)["kind"]; got != wantKind {
			t.Errorf("%s kind = %v, want %s", logicalName, got, wantKind)
		}
	}

	serviceAccount := serviceAccountsDesired(t, desired, "service-account-ci-runner")
	if got, want := nestedMap(t, serviceAccount, "spec", "forProvider"), map[string]any{"name": "ci-runner", "role": "Editor", "isDisabled": false}; !cmp.Equal(got, want) {
		t.Fatalf("service account fields differ (-want +got):\n%s", cmp.Diff(want, got))
	}
	if annotations, exists := nestedMap(t, serviceAccount, "metadata")["annotations"]; exists || annotations != nil {
		t.Fatalf("service account pre-derived provider ID as external name: %v", annotations)
	}

	token := serviceAccountsDesired(t, desired, "service-account-ci-runner-token")
	tokenParameters := nestedMap(t, token, "spec", "forProvider")
	if got, want := tokenParameters["serviceAccountId"], "101"; got != want {
		t.Fatalf("rotating token serviceAccountId = %v, want %s", got, want)
	}
	if got, want := tokenParameters["secondsToLive"], int64(336*time.Hour/time.Second); got != want {
		t.Fatalf("rotating token secondsToLive = %v, want %v", got, want)
	}
	if got, want := tokenParameters["earlyRotationWindowSeconds"], int64(168*time.Hour/time.Second); got != want {
		t.Fatalf("rotating token earlyRotationWindowSeconds = %v, want %v", got, want)
	}

	permissions := serviceAccountsDesired(t, desired, "service-account-ci-runner-permissions")
	if got, want := nestedMap(t, permissions, "metadata")["annotations"], map[string]any{"crossplane.io/external-name": "101"}; !cmp.Equal(got, want) {
		t.Fatalf("whole-set permission external name differs (-want +got):\n%s", cmp.Diff(want, got))
	}
	if got, want := nestedMap(t, permissions, "spec", "forProvider")["permissions"], []any{map[string]any{"permission": "Admin", "teamId": "42"}}; !cmp.Equal(got, want) {
		t.Fatalf("whole-set permissions differ (-want +got):\n%s", cmp.Diff(want, got))
	}
	for name, child := range desired {
		if child.Resource.GetKind() == "ServiceAccountPermissionItem" || child.Resource.GetKind() == "ServiceAccountToken" {
			t.Fatalf("%s rendered forbidden competing or static credential kind %s", name, child.Resource.GetKind())
		}
	}

	credentials := serviceAccountsDesired(t, desired, "service-account-ci-runner-credentials")
	template := nestedMap(t, credentials, "spec", "template", "data")["service-account.json"]
	if got, want := template, `{{ $token := index . "attribute.key" | toString }}{"service_account_token":{{ $token | toJson }}}`; got != want {
		t.Fatalf("published credential template = %v, want %s", got, want)
	}
	if _, hasStatus := credentials["status"]; strings.Contains(mustJSON(credentials), "example-token") || hasStatus {
		t.Fatal("rendered credential contained a token value or status")
	}
}

func TestServiceAccountsWaitForObservedProviderID(t *testing.T) {
	desired, err := renderServiceAccounts(serviceAccountsClaim(), nil, serviceAccountsConfig("336h", "336h"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(desired), 2; got != want {
		t.Fatalf("rendered resources before IDs = %d, want service accounts only", got)
	}
	for name, child := range desired {
		if child.Resource.GetKind() != "ServiceAccount" {
			t.Fatalf("%s kind = %s before observed ID, want ServiceAccount", name, child.Resource.GetKind())
		}
	}
}

func TestServiceAccountsUseTheSharedTokenCeilingWithoutAParallelLimit(t *testing.T) {
	config := serviceAccountsConfig("336h", "336h")
	desired, err := renderServiceAccounts(serviceAccountsClaim(), map[resource.Name]resource.ObservedComposed{
		"service-account-ci-runner": observedComposed(`{"status":{"atProvider":{"id":"101"}}}`),
	}, config)
	if err != nil {
		t.Fatal(err)
	}
	got := nestedMap(t, serviceAccountsDesired(t, desired, "service-account-ci-runner-token"), "spec", "forProvider")["secondsToLive"]
	shared, err := boundedTokenLifetime("336h", 336*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if want := int64(shared / time.Second); got != want {
		t.Fatalf("in-stack token lifetime = %v, want shared boundedTokenLifetime result %v", got, want)
	}

	_, err = renderServiceAccounts(serviceAccountsClaim(), nil, serviceAccountsConfig("336h", "720h"))
	if err == nil || !strings.Contains(err.Error(), "exceeds platform maximumTokenLifetime") {
		t.Fatalf("profile over the existing ceiling error = %v", err)
	}
}

func TestServiceAccountsAdmissionRejectsProfileChangesAndAllowsAccountUpdates(t *testing.T) {
	env := &admissionEnv{paths: []string{"../apis/service-accounts-v1beta1.yaml"}}
	if err := env.Start(t); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = env.Stop() })

	claim := serviceAccountsAdmissionObject("admission-service-accounts", "ci")
	if err := env.Apply(context.Background(), claim); err != nil {
		t.Fatalf("Apply(valid service account request) error = %v", err)
	}
	claim.Object["spec"].(map[string]any)["profile"] = "other"
	if err := env.Apply(context.Background(), claim); err == nil || !strings.Contains(err.Error(), "profile is immutable after creation") {
		t.Fatalf("profile update error = %v, want profile immutability refusal", err)
	}
	claim.Object["spec"].(map[string]any)["profile"] = "ci"
	claim.Object["spec"].(map[string]any)["accounts"] = []any{map[string]any{"name": "ci-runner"}, map[string]any{"name": "terraform"}}
	if err := env.Apply(context.Background(), claim); err != nil {
		t.Fatalf("Apply(account update with unchanged profile) error = %v", err)
	}
}

func TestServiceAccountsAdmissionTokenCeilingNegativeControl(t *testing.T) {
	original, err := os.ReadFile("../apis/service-accounts-v1beta1.yaml")
	if err != nil {
		t.Fatal(err)
	}
	overCeiling := bytes.Replace(original, []byte("tokenLifetime: 168h"), []byte("tokenLifetime: 721h"), 1)
	if bytes.Equal(overCeiling, original) {
		t.Fatal("profile token lifetime control value was not found")
	}
	weakRule := bytes.Replace(overCeiling, []byte("profile.name == object.spec.profile && duration(profile.tokenLifetime) <= duration(step.input.spec.maximumTokenLifetime)"), []byte("profile.name == object.spec.profile"), 1)
	if bytes.Equal(weakRule, overCeiling) {
		t.Fatal("token ceiling validation expression was not found")
	}
	scratch := filepath.Join(t.TempDir(), "service-accounts-v1beta1.yaml")
	assertServiceAccountsAdmissionOutcome(t, scratch, overCeiling, true, "service account profile token lifetime exceeds the platform maximumTokenLifetime")
	assertServiceAccountsAdmissionOutcome(t, scratch, weakRule, false, "")
	assertServiceAccountsAdmissionOutcome(t, scratch, overCeiling, true, "service account profile token lifetime exceeds the platform maximumTokenLifetime")
	if err := os.WriteFile(scratch, original, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Log("negative control restored the original service-account Composition and policy source")
}

func assertServiceAccountsAdmissionOutcome(t *testing.T, path string, source []byte, refused bool, message string) {
	t.Helper()
	if err := os.WriteFile(path, source, 0o600); err != nil {
		t.Fatal(err)
	}
	env := &admissionEnv{paths: []string{path}}
	if err := env.Start(t); err != nil {
		t.Fatalf("Start(%s) error = %v", filepath.Base(path), err)
	}
	var err error
	deadline := time.Now().Add(5 * time.Second)
	for attempt := 0; ; attempt++ {
		err = env.Apply(context.Background(), serviceAccountsAdmissionObject("token-ceiling-control-"+strconv.Itoa(attempt), "ci"))
		if !refused || (err != nil && strings.Contains(err.Error(), message)) || time.Now().After(deadline) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if stopErr := env.Stop(); stopErr != nil {
		t.Fatalf("Stop(%s) error = %v", filepath.Base(path), stopErr)
	}
	if refused {
		if err == nil || !strings.Contains(err.Error(), message) {
			t.Fatalf("admission error = %v, want refusal containing %q", err, message)
		}
		t.Logf("token ceiling control refused: %v", err)
		return
	}
	if err != nil {
		t.Fatalf("weakened token ceiling admitted error = %v", err)
	}
	t.Log("weakened token ceiling admitted the otherwise-forbidden request")
}

func serviceAccountsClaim() map[string]any {
	return map[string]any{
		"metadata": map[string]any{"name": "teamdemo01", "namespace": "grafana-vending"},
		"spec": map[string]any{
			"stackRef": map[string]any{"name": "teamdemo01"}, "profile": "ci",
			"accounts": []any{map[string]any{"name": "ci-runner"}, map[string]any{"name": "terraform"}},
		},
	}
}

func serviceAccountsConfig(maximum, requested string) map[string]any {
	return map[string]any{
		"referencedStack": map[string]any{
			"name": "teamdemo01", "namespace": "grafana-vending", "outputSecretPath": "/platform/example/service-accounts", "providerConfigName": "teamdemo01",
		},
		"spec": map[string]any{
			"maximumTokenLifetime": maximum,
			"secretStoreRef":       map[string]any{"name": "grafana-vending-secrets", "kind": "SecretStore"},
			"serviceAccountProfiles": []any{map[string]any{
				"name": "ci", "role": "Editor", "tokenLifetime": requested,
				"permissions": []any{map[string]any{"permission": "Admin", "teamId": "42"}},
			}},
		},
	}
}

func serviceAccountsDesired(t *testing.T, desired map[resource.Name]*resource.DesiredComposed, name resource.Name) map[string]any {
	t.Helper()
	child, ok := desired[name]
	if !ok {
		t.Fatalf("missing desired %s", name)
	}
	return child.Resource.UnstructuredContent()
}

func serviceAccountsAdmissionObject(name, profile string) *unstructured.Unstructured {
	name = strings.ReplaceAll(name, "-", "")
	return requestObject("GrafanaServiceAccounts", name, map[string]any{
		"stackRef": map[string]any{"name": name}, "profile": profile,
		"accounts": []any{map[string]any{"name": "ci-runner"}},
	})
}
