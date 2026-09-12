package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestProvisioningConnectionStagesCredentialBridgeAndConnection(t *testing.T) {
	claim := provisioningConnectionClaim("dashboards-github")
	first, err := renderProvisioningConnection(claim, nil, provisioningConnectionConfig())
	if err != nil {
		t.Fatalf("render credential bridge: %v", err)
	}
	if got, want := len(first), 2; got != want {
		t.Fatalf("first render resource count = %d, want %d", got, want)
	}
	if _, found := first[provisioningConnectionConnection]; found {
		t.Fatal("Connection rendered before the secure value was observed")
	}

	credential := first[provisioningConnectionExternalSecret].Resource.UnstructuredContent()
	if got := desiredExternalName(t, credential); got != "dashboards-github-credential" {
		t.Fatalf("ExternalSecret external name = %q", got)
	}
	credentialSpec := nestedMap(t, credential, "spec")
	if got := credentialSpec["refreshInterval"]; got != "1h" {
		t.Fatalf("ExternalSecret refresh interval = %v, want 1h", got)
	}
	if got := nestedMap(t, credentialSpec, "secretStoreRef"); got["name"] != "platform-secrets" || got["kind"] != "ClusterSecretStore" {
		t.Fatalf("ExternalSecret store ref = %#v", got)
	}
	target := nestedMap(t, credentialSpec, "target")
	if got := target["creationPolicy"]; got != "Owner" {
		t.Fatalf("ExternalSecret creation policy = %v, want Owner", got)
	}
	if got := target["deletionPolicy"]; got != "Retain" {
		t.Fatalf("ExternalSecret deletion policy = %v, want Retain", got)
	}
	remote := nestedMap(t, credentialSpec["data"].([]any)[0].(map[string]any), "remoteRef")
	if got := remote["key"]; got != "kv/git-sync/dashboard-source" {
		t.Fatalf("ExternalSecret remote key = %v", got)
	}
	if got := remote["property"]; got != "private_key" {
		t.Fatalf("ExternalSecret remote property = %v", got)
	}

	secureValue := first[provisioningConnectionSecureValue].Resource.UnstructuredContent()
	if got := desiredExternalName(t, secureValue); got != "dashboards-github-secure-value" {
		t.Fatalf("Securevalue external name = %q", got)
	}
	secureSpec := nestedMap(t, secureValue, "spec")
	if got := nestedMap(t, secureSpec, "providerConfigRef")["name"]; got != "teamdemo01" {
		t.Fatalf("Securevalue ProviderConfig = %v, want teamdemo01", got)
	}
	if got := secureSpec["managementPolicies"]; mustJSON(got) != mustJSON(managementPolicies) {
		t.Fatalf("Securevalue management policies = %s, want %s", mustJSON(got), mustJSON(managementPolicies))
	}
	secureProvider := nestedMap(t, secureSpec, "forProvider")
	if got := nestedMap(t, secureProvider, "metadata")["uid"]; got != "dashboards-github-secure-value" {
		t.Fatalf("Securevalue UID = %v", got)
	}
	if got := nestedMap(t, secureProvider, "spec", "valueSecretRef"); got["name"] != "dashboards-github-credential" || got["key"] != provisioningConnectionCredentialKey {
		t.Fatalf("Securevalue Secret reference = %#v", got)
	}
	if got := nestedMap(t, secureProvider, "spec")["decrypters"]; mustJSON(got) != mustJSON([]any{"github-app"}) {
		t.Fatalf("Securevalue decrypters = %s", mustJSON(got))
	}

	second, err := renderProvisioningConnection(claim, map[resource.Name]resource.ObservedComposed{
		provisioningConnectionSecureValue: observedComposed(`{"metadata":{"name":"dashboards-github-secure-value"}}`),
	}, provisioningConnectionConfig())
	if err != nil {
		t.Fatalf("render Connection after secure value: %v", err)
	}
	connection := second[provisioningConnectionConnection].Resource.UnstructuredContent()
	if got := desiredExternalName(t, connection); got != "dashboards-github" {
		t.Fatalf("Connection external name = %q", got)
	}
	connectionProvider := nestedMap(t, connection, "spec", "forProvider")
	if got := nestedMap(t, connection, "spec", "providerConfigRef")["name"]; got != "teamdemo01" {
		t.Fatalf("Connection ProviderConfig = %v, want teamdemo01", got)
	}
	// Grafana Cloud 403s the reference form {name: <securevalue>}; only the
	// base64 create form is accepted. GCV-0074 carries the live evidence.
	if got := nestedMap(t, connectionProvider, "secure", "privateKey"); mustJSON(got) != `{"create":"`+provisioningConnectionMaterialisedKey+`"}` {
		t.Fatalf("Connection private key = %s", mustJSON(got))
	}
	if _, found := nestedMap(t, connectionProvider, "secure", "privateKey")["name"]; found {
		t.Fatal("Connection rendered the reference form Grafana Cloud refuses")
	}
	if _, found := nestedMap(t, connectionProvider, "secure")["token"]; found {
		t.Fatal("GitHub App Connection rendered a token secure map")
	}
	if got := connectionProvider["secureVersion"]; got != float64(2) {
		t.Fatalf("Connection secure version = %v, want 2", got)
	}
	github := nestedMap(t, connectionProvider, "spec", "github")
	if github["appId"] != "12345" || github["installationId"] != "67890" {
		t.Fatalf("Connection GitHub identity = %#v", github)
	}
	for name, child := range second {
		if got := desiredExternalName(t, child.Resource.UnstructuredContent()); got == "" {
			t.Errorf("%s has no deterministic external name", name)
		}
	}
	if encoded := mustJSON(second); strings.Contains(encoded, "credential-literal") {
		t.Fatal("rendered bridge contains a credential literal")
	}
}

func TestProvisioningConnectionClaimsHaveDistinctStableExternalNames(t *testing.T) {
	seen := map[string]string{}
	for _, name := range []string{"dashboards-github", "alerts-github"} {
		desired, err := renderProvisioningConnection(provisioningConnectionClaim(name), map[resource.Name]resource.ObservedComposed{
			provisioningConnectionSecureValue: observedComposed(`{"metadata":{"name":"observed-secure-value"}}`),
		}, provisioningConnectionConfig())
		if err != nil {
			t.Fatal(err)
		}
		for logicalName, child := range desired {
			externalName := desiredExternalName(t, child.Resource.UnstructuredContent())
			if previous, found := seen[externalName]; found {
				t.Fatalf("claims %q and %q share external name %q", previous, name, externalName)
			}
			seen[externalName] = name + "/" + string(logicalName)
		}
	}
}

func TestProvisioningConnectionRejectsCredentialLiteralShapesAtReconcile(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(map[string]any)
		message string
	}{
		{
			name: "provider secure map",
			mutate: func(spec map[string]any) {
				spec["secure"] = map[string]any{"privateKey": map[string]any{"create": "credential-literal"}}
			},
			message: "forbids provider secure maps",
		},
		{
			name: "inline credential",
			mutate: func(spec map[string]any) {
				spec["credential"].(map[string]any)["inline"] = "credential-literal"
			},
			message: "forbids inline credentials",
		},
		{
			name: "secure-map create value",
			mutate: func(spec map[string]any) {
				spec["credential"].(map[string]any)["create"] = "credential-literal"
			},
			message: "forbids secure-map create values",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			claim := provisioningConnectionClaim("dashboards-github")
			tc.mutate(claim["spec"].(map[string]any))
			if desired, err := renderProvisioningConnection(claim, nil, provisioningConnectionConfig()); err == nil || !strings.Contains(err.Error(), tc.message) || len(desired) != 0 {
				t.Fatalf("%s = desired %v, error %v", tc.name, desired, err)
			}
		})
	}
}

func TestProvisioningConnectionRejectsCredentialBearingURLAndMultipleDecryptersAtReconcile(t *testing.T) {
	claim := provisioningConnectionClaim("dashboards-github")
	claim["spec"].(map[string]any)["url"] = "https://user:credential@example.invalid/repository"
	if desired, err := renderProvisioningConnection(claim, nil, provisioningConnectionConfig()); err == nil || len(desired) != 0 {
		t.Fatalf("credential-bearing URL = desired %v, error %v", desired, err)
	}

	claim = provisioningConnectionClaim("dashboards-github")
	claim["spec"].(map[string]any)["decrypters"] = []any{"github-app", "unrelated-reader"}
	if desired, err := renderProvisioningConnection(claim, nil, provisioningConnectionConfig()); err == nil || len(desired) != 0 {
		t.Fatalf("multiple decrypters = desired %v, error %v", desired, err)
	}
}

func TestProvisioningConnectionProviderChildrenRoundTripThroughPinnedCRDs(t *testing.T) {
	e := &admissionEnv{paths: []string{"../apis/provisioning-connection-v1beta1.yaml"}}
	if err := e.Start(t); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = e.Stop() })
	installProviderAdmissionCRDs(t, e)
	children, err := renderProvisioningConnection(provisioningConnectionClaim("dashboards-github"), map[resource.Name]resource.ObservedComposed{
		provisioningConnectionSecureValue: observedComposed(`{"metadata":{"name":"dashboards-github-secure-value"}}`),
	}, provisioningConnectionConfig())
	if err != nil {
		t.Fatal(err)
	}
	admitProviderChildren(t, e, map[resource.Name]*resource.DesiredComposed{
		provisioningConnectionSecureValue: children[provisioningConnectionSecureValue],
		provisioningConnectionConnection:  children[provisioningConnectionConnection],
	})
}

func TestProvisioningConnectionAdmissionRejectsInlineCredentialOnCreateAndUpdate(t *testing.T) {
	const (
		xrdPath      = "../apis/provisioning-connection-v1beta1.yaml"
		originalRule = `- rule: "!has(self.secure) && !has(self.credential.inline) && !has(self.credential.create)"`
		weakenedRule = `- rule: "true"`
		denial       = "Credentials must be supplied only through credential.remoteRef"
	)
	e := &admissionEnv{paths: []string{xrdPath}}
	if err := e.Start(t); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = e.Stop() })
	ctx := context.Background()
	if err := e.Apply(ctx, &unstructured.Unstructured{Object: map[string]any{"apiVersion": "v1", "kind": "Namespace", "metadata": map[string]any{"name": "grafana-vending"}}}); err != nil {
		t.Fatalf("create request namespace: %v", err)
	}
	valid := provisioningConnectionAdmissionRequest("connectionvalid")
	if err := e.Apply(ctx, valid); err != nil {
		t.Fatalf("valid connection request: %v", err)
	}
	retargeted := valid.DeepCopy()
	retargeted.Object["spec"].(map[string]any)["stackRef"].(map[string]any)["name"] = "teamdemo02"
	if err := e.Apply(ctx, retargeted); err == nil || !strings.Contains(err.Error(), "stackRef.name is immutable") {
		t.Fatalf("stackRef update error = %v, want immutability denial", err)
	}
	for _, operation := range []string{"create", "update"} {
		var candidate *unstructured.Unstructured
		if operation == "create" {
			candidate = provisioningConnectionAdmissionRequest("connectiondecrypters" + operation)
		} else {
			candidate = valid.DeepCopy()
		}
		candidate.Object["spec"].(map[string]any)["decrypters"] = []any{"github-app", "unrelated-reader"}
		if err := e.Apply(ctx, candidate); err == nil || !strings.Contains(err.Error(), "must have at most 1 item") {
			t.Fatalf("multiple decrypters %s = %v, want cardinality denial", operation, err)
		}
		candidate.Object["spec"].(map[string]any)["decrypters"] = []any{}
		if err := e.Apply(ctx, candidate); err == nil || !strings.Contains(err.Error(), "at least 1 item") {
			t.Fatalf("empty decrypters %s = %v, want cardinality denial", operation, err)
		}
	}
	update := valid.DeepCopy()
	update.Object["spec"].(map[string]any)["secure"] = map[string]any{"privateKey": map[string]any{"create": "credential-literal"}}
	if err := e.Apply(ctx, update); err == nil || !strings.Contains(err.Error(), denial) {
		t.Fatalf("inline credential update = %v, want own denial", err)
	}

	inline := provisioningConnectionAdmissionRequest("connectionbaseline")
	inline.Object["spec"].(map[string]any)["credential"].(map[string]any)["inline"] = "credential-literal"
	if err := e.Apply(ctx, inline); err == nil || !strings.Contains(err.Error(), denial) {
		t.Fatalf("inline credential create = %v, want own denial", err)
	}

	original, err := os.ReadFile(xrdPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := bytes.Count(original, []byte(originalRule)); got != 1 {
		t.Fatalf("credential guard count = %d, want 1", got)
	}
	weakened := bytes.Replace(original, []byte(originalRule), []byte(weakenedRule), 1)
	scratch := filepath.Join(t.TempDir(), "provisioning-connection-v1beta1.yaml")
	if err := os.WriteFile(scratch, weakened, 0o600); err != nil {
		t.Fatal(err)
	}
	weakenedCRD, err := crdFromXRD(scratch)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Apply(ctx, weakenedCRD); err != nil {
		t.Fatalf("apply weakened credential guard: %v", err)
	}
	weakenedRequest := provisioningConnectionAdmissionRequest("connectionweakened")
	weakenedRequest.Object["spec"].(map[string]any)["secure"] = map[string]any{"privateKey": map[string]any{"create": "credential-literal"}}
	if err := e.Apply(ctx, weakenedRequest); err != nil {
		t.Fatalf("weakened credential guard did not admit: %v", err)
	}
	t.Log("API-SERVER weakened inline credential guard: admitted")

	originalCRD, err := crdFromXRD(xrdPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Apply(ctx, originalCRD); err != nil {
		t.Fatalf("restore credential guard: %v", err)
	}
	restored := provisioningConnectionAdmissionRequest("connectionrestored")
	restored.Object["spec"].(map[string]any)["credential"].(map[string]any)["create"] = "credential-literal"
	if err := e.Apply(ctx, restored); err == nil || !strings.Contains(err.Error(), denial) {
		t.Fatalf("restored credential guard = %v, want own denial", err)
	} else {
		t.Logf("API-SERVER restored inline credential guard: %v", err)
	}
}

func TestProvisioningConnectionAdmissionRejectsCredentialBearingURL(t *testing.T) {
	const (
		xrdPath      = "../apis/provisioning-connection-v1beta1.yaml"
		originalRule = `- rule: "!self.matches('^https://[^/]*@')"`
		weakenedRule = `- rule: "true"`
		denial       = "URL userinfo is forbidden because credentials must use credential.remoteRef"
	)
	e := &admissionEnv{paths: []string{xrdPath}}
	if err := e.Start(t); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = e.Stop() })
	ctx := context.Background()
	if err := e.Apply(ctx, &unstructured.Unstructured{Object: map[string]any{"apiVersion": "v1", "kind": "Namespace", "metadata": map[string]any{"name": "grafana-vending"}}}); err != nil {
		t.Fatalf("create request namespace: %v", err)
	}
	valid := provisioningConnectionAdmissionRequest("connectionurlvalid")
	if err := e.Apply(ctx, valid); err != nil {
		t.Fatalf("valid connection request: %v", err)
	}
	for _, operation := range []string{"create", "update"} {
		var candidate *unstructured.Unstructured
		if operation == "create" {
			candidate = provisioningConnectionAdmissionRequest("connectionurl" + operation)
		} else {
			candidate = valid.DeepCopy()
		}
		candidate.Object["spec"].(map[string]any)["url"] = "https://user:credential@example.invalid/repository"
		if err := e.Apply(ctx, candidate); err == nil || !strings.Contains(err.Error(), denial) {
			t.Fatalf("credential-bearing URL %s = %v, want own denial", operation, err)
		}
	}
	original, err := os.ReadFile(xrdPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := bytes.Count(original, []byte(originalRule)); got != 1 {
		t.Fatalf("URL userinfo guard count = %d, want 1", got)
	}
	weakened := bytes.Replace(original, []byte(originalRule), []byte(weakenedRule), 1)
	scratch := filepath.Join(t.TempDir(), "provisioning-connection-v1beta1.yaml")
	if err := os.WriteFile(scratch, weakened, 0o600); err != nil {
		t.Fatal(err)
	}
	weakenedCRD, err := crdFromXRD(scratch)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Apply(ctx, weakenedCRD); err != nil {
		t.Fatalf("apply weakened URL guard: %v", err)
	}
	weakenedRequest := provisioningConnectionAdmissionRequest("connectionurlweakened")
	weakenedRequest.Object["spec"].(map[string]any)["url"] = "https://user:credential@example.invalid/repository"
	if err := e.Apply(ctx, weakenedRequest); err != nil {
		t.Fatalf("weakened URL guard did not admit: %v", err)
	}
	t.Log("API-SERVER weakened connection URL userinfo guard: admitted")
	originalCRD, err := crdFromXRD(xrdPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Apply(ctx, originalCRD); err != nil {
		t.Fatalf("restore URL guard: %v", err)
	}
	restored := provisioningConnectionAdmissionRequest("connectionurlrestored")
	restored.Object["spec"].(map[string]any)["url"] = "https://user:credential@example.invalid/repository"
	if err := e.Apply(ctx, restored); err == nil || !strings.Contains(err.Error(), denial) {
		t.Fatalf("restored URL guard = %v, want own denial", err)
	}
}

func provisioningConnectionClaim(name string) map[string]any {
	return map[string]any{
		"apiVersion": "platform.example.org/v1beta1", "kind": "GrafanaProvisioningConnection",
		"metadata": map[string]any{"name": name, "namespace": "grafana-vending"},
		"spec": map[string]any{
			"stackRef": map[string]any{"name": "teamdemo01"},
			"title":    "Dashboards GitHub App", "type": "github", "description": "Git Sync access for dashboards.", "url": "https://github.com/example/grafana-dashboards",
			"github":     map[string]any{"appId": "12345", "installationId": "67890"},
			"credential": map[string]any{"remoteRef": map[string]any{"key": "kv/git-sync/dashboard-source", "property": "private_key"}},
			"decrypters": []any{"github-app"}, "secureVersion": float64(2),
		},
	}
}

// The base64 of a PEM, as a Kubernetes Secret data entry already stores it.
// Grafana requires exactly this encoding, so nothing decodes it on the way.
const provisioningConnectionMaterialisedKey = "LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCnRlc3QKLS0tLS1FTkQgUFJJVkFURSBLRVktLS0tLQo="

func provisioningConnectionConfig() map[string]any {
	config := provisioningConnectionConfigWithoutCredential()
	config[provisioningConnectionCredentialConfigKey] = provisioningConnectionMaterialisedKey
	return config
}

func provisioningConnectionConfigWithoutCredential() map[string]any {
	return map[string]any{"spec": map[string]any{"secretStoreRef": map[string]any{"name": "platform-secrets", "kind": "ClusterSecretStore"}}}
}

func provisioningConnectionAdmissionRequest(name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: provisioningConnectionClaim(name)}
}

// The Connection carries a credential value, so the Secret it comes from is
// identity-checked before it is trusted, and a Connection is never rendered
// without one. GCV-0074 recorded the live 403 that forces the create form.
func TestProvisioningConnectionCredentialConfigTrustsOnlyItsOwnSecret(t *testing.T) {
	claim := provisioningConnectionClaim("dashboards-github")
	metadata := claim["metadata"].(map[string]any)
	namespace := metadata["namespace"].(string)

	secret := func(mutate func(object map[string]any)) *fnv1.Resources {
		object := map[string]any{
			"apiVersion": "v1", "kind": "Secret",
			"metadata": map[string]any{"name": "dashboards-github-credential", "namespace": namespace},
			"data":     map[string]any{provisioningConnectionCredentialKey: provisioningConnectionMaterialisedKey},
		}
		if mutate != nil {
			mutate(object)
		}
		return &fnv1.Resources{Items: []*fnv1.Resource{{Resource: resource.MustStructJSON(mustJSON(object))}}}
	}

	for _, testCase := range []struct {
		name    string
		secret  *fnv1.Resources
		wantKey bool
	}{
		{name: "own secret", secret: secret(nil), wantKey: true},
		{name: "unresolved", secret: nil},
		{name: "wrong namespace", secret: secret(func(object map[string]any) {
			object["metadata"].(map[string]any)["namespace"] = "other"
		})},
		{name: "wrong name", secret: secret(func(object map[string]any) {
			object["metadata"].(map[string]any)["name"] = "someone-elses-credential"
		})},
		{name: "being deleted", secret: secret(func(object map[string]any) {
			object["metadata"].(map[string]any)["deletionTimestamp"] = "2026-09-12T00:00:00Z"
		})},
		{name: "wrong kind", secret: secret(func(object map[string]any) { object["kind"] = "ConfigMap" })},
		{name: "empty key", secret: secret(func(object map[string]any) {
			object["data"] = map[string]any{provisioningConnectionCredentialKey: ""}
		})},
		{name: "absent key", secret: secret(func(object map[string]any) { object["data"] = map[string]any{} })},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			req := &fnv1.RunFunctionRequest{RequiredResources: map[string]*fnv1.Resources{}}
			if testCase.secret != nil {
				req.RequiredResources[provisioningConnectionCredentialRequirement] = testCase.secret
			}
			rsp := &fnv1.RunFunctionResponse{}
			resolved := provisioningConnectionCredentialConfig(req, rsp, claim, provisioningConnectionConfigWithoutCredential())

			// The requirement is always registered, or a Secret that appears
			// later is never fetched and the Connection never renders.
			selector := rsp.GetRequirements().GetResources()[provisioningConnectionCredentialRequirement]
			if selector == nil {
				t.Fatal("credential requirement was not registered")
			}
			if selector.GetApiVersion() != "v1" || selector.GetKind() != "Secret" || selector.GetNamespace() != namespace || selector.GetMatchName() != "dashboards-github-credential" {
				t.Fatalf("credential selector = %#v", selector)
			}

			value, found := resolved[provisioningConnectionCredentialConfigKey].(string)
			if found != testCase.wantKey {
				t.Fatalf("credential present = %t, want %t", found, testCase.wantKey)
			}
			if !testCase.wantKey {
				return
			}
			// Passed through verbatim: a Secret data entry is already base64,
			// which is the encoding Grafana requires, so decoding it and
			// re-encoding it would be the only way to get it wrong.
			if value != provisioningConnectionMaterialisedKey {
				t.Fatalf("credential = %q, want the Secret entry verbatim", value)
			}
		})
	}
}

// A Connection with no private key is refused by Grafana with a 422 before it
// reaches the 403, so rendering one is strictly worse than rendering nothing.
func TestProvisioningConnectionWithheldUntilCredentialMaterialises(t *testing.T) {
	claim := provisioningConnectionClaim("dashboards-github")
	observed := map[resource.Name]resource.ObservedComposed{
		provisioningConnectionSecureValue: observedComposed(`{"metadata":{"name":"dashboards-github-secure-value"}}`),
	}
	withheld, err := renderProvisioningConnection(claim, observed, provisioningConnectionConfigWithoutCredential())
	if err != nil {
		t.Fatalf("render without a materialised credential: %v", err)
	}
	if _, found := withheld[provisioningConnectionConnection]; found {
		t.Fatal("Connection rendered before its credential was materialised")
	}
	if got, want := len(withheld), 2; got != want {
		t.Fatalf("withheld render resource count = %d, want %d", got, want)
	}

	rendered, err := renderProvisioningConnection(claim, observed, provisioningConnectionConfig())
	if err != nil {
		t.Fatalf("render with a materialised credential: %v", err)
	}
	if _, found := rendered[provisioningConnectionConnection]; !found {
		t.Fatal("Connection withheld despite a materialised credential")
	}
}

// A credential that stops materialising must never take a live Connection with
// it. The ExternalSecret can fail to sync or its Secret can be deleted, and the
// shorter desired set would withdraw the external Grafana Connection.
func TestProvisioningConnectionRefusesToWithdrawConnectionWhenCredentialVanishes(t *testing.T) {
	claim := provisioningConnectionClaim("dashboards-github")
	observed := map[resource.Name]resource.ObservedComposed{
		provisioningConnectionSecureValue: observedComposed(`{"metadata":{"name":"dashboards-github-secure-value"}}`),
		provisioningConnectionConnection:  observedComposed(`{"metadata":{"name":"dashboards-github"}}`),
	}
	if _, err := renderProvisioningConnection(claim, observed, provisioningConnectionConfigWithoutCredential()); err == nil {
		t.Fatal("render withdrew an observed Connection when the credential was not materialised")
	} else if !strings.Contains(err.Error(), "refusing to withdraw the existing connection") {
		t.Fatalf("unexpected error = %v", err)
	}

	// The same absence before the Connection exists is a normal first reconcile.
	firstReconcile := map[resource.Name]resource.ObservedComposed{
		provisioningConnectionSecureValue: observedComposed(`{"metadata":{"name":"dashboards-github-secure-value"}}`),
	}
	staged, err := renderProvisioningConnection(claim, firstReconcile, provisioningConnectionConfigWithoutCredential())
	if err != nil {
		t.Fatalf("render before the credential materialises: %v", err)
	}
	if _, found := staged[provisioningConnectionConnection]; found {
		t.Fatal("Connection rendered before its credential was materialised")
	}

	// And an observed Connection with the credential present still renders.
	rendered, err := renderProvisioningConnection(claim, observed, provisioningConnectionConfig())
	if err != nil {
		t.Fatalf("render with an observed Connection and a credential: %v", err)
	}
	if _, found := rendered[provisioningConnectionConnection]; !found {
		t.Fatal("Connection withheld despite a materialised credential")
	}
}
