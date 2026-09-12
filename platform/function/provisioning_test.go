package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestProvisioningRepositoryRejectsBothDashboardOwners(t *testing.T) {
	claim := provisioningRepositoryDocument(map[string]any{
		"dashboard": map[string]any{"uid": "application-overview"},
	})

	rsp := callProvisioning(t, claim)
	if fatal := fatalResult(rsp); !strings.Contains(fatal, "both repository and Crossplane dashboard ownership") {
		t.Fatalf("mixed dashboard ownership returned fatal result %q", fatal)
	}
}

func TestProvisioningRepositoryRejectsCrossplaneDashboardOwnership(t *testing.T) {
	claim := provisioningRepositoryDocument(map[string]any{
		"repository": nil,
		"dashboard":  map[string]any{"uid": "application-overview"},
	})

	rsp := callProvisioning(t, claim)
	if fatal := fatalResult(rsp); !strings.Contains(fatal, "does not render Crossplane Dashboards") {
		t.Fatalf("Crossplane dashboard ownership returned fatal result %q", fatal)
	}
}

func TestProvisioningRepositoryRendersCredentialReferenceWithoutDashboard(t *testing.T) {
	rsp := runProvisioning(t, provisioningRepositoryDocument(nil))
	resources := rsp.GetDesired().GetResources()
	if len(resources) != 1 {
		t.Fatalf("desired resource count = %d, want 1", len(resources))
	}
	repository := desiredResource(t, rsp, "repository")
	if got := repository["kind"]; got != "RepositoryV0Alpha1" {
		t.Fatalf("repository kind = %v, want RepositoryV0Alpha1", got)
	}
	if got := desiredExternalName(t, repository); got != "application-overview" {
		t.Fatalf("repository external name = %q, want application-overview", got)
	}
	forProvider := nestedMap(t, repository, "spec", "forProvider")
	if got := nestedMap(t, forProvider, "spec", "connection")["name"]; got != "shared-github-app" {
		t.Fatalf("repository connection reference = %v, want shared-github-app", got)
	}
	if _, found := forProvider["secure"]; found {
		t.Fatal("repository without secure value names must not render forProvider.secure")
	}
	if got := nestedMap(t, repository, "spec", "providerConfigRef")["name"]; got != "teamdemo01" {
		t.Fatalf("repository ProviderConfig = %v, want teamdemo01", got)
	}
	for name, desired := range resources {
		if kind := desired.GetResource().GetFields()["kind"].GetStringValue(); kind == "Dashboard" {
			t.Fatalf("repository-provisioned subtree rendered Crossplane Dashboard %q", name)
		}
	}
}

func TestProvisioningRepositoryRendersFullProviderSurfaceAndKeepsEmptyWorkflows(t *testing.T) {
	claim := provisioningRepositoryClaim("all-options", provisioningRepositorySpec("github"))
	repository := claim["spec"].(map[string]any)["repository"].(map[string]any)
	repository["sync"] = map[string]any{"enabled": false, "target": "folderless", "intervalSeconds": float64(120)}
	repository["github"].(map[string]any)["generateDashboardPreviews"] = true
	repository["branch"] = map[string]any{"nameTemplate": "feature/{{ .Name }}", "enforceTemplate": true}
	repository["pullRequest"] = map[string]any{"titleTemplate": "Update {{ .Name }}", "enforceTemplate": true}
	repository["commit"] = map[string]any{
		"signerName": "Grafana automation", "signerEmail": "automation@example.invalid", "signingMethod": "smime",
		"smimeCertificate":              "-----BEGIN CERTIFICATE----- public -----END CERTIFICATE-----",
		"singleResourceMessageTemplate": "Update {{ .Name }}", "enforceTemplate": true,
	}
	repository["webhook"] = map[string]any{"baseUrl": "https://hooks.example.invalid/grafana"}
	repository["secure"] = map[string]any{
		"token": map[string]any{"name": "repository-token"}, "webhookSecret": map[string]any{"name": "repository-webhook"},
		"commitSigningKey": map[string]any{"name": "repository-signing-key"},
	}
	repository["secureVersion"] = float64(2)

	desired, err := renderProvisioningRepository(claim, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	forProvider := nestedMap(t, desired["repository"].Resource.UnstructuredContent(), "spec", "forProvider")
	providerSpec := nestedMap(t, forProvider, "spec")
	if _, found := providerSpec["description"]; found {
		t.Fatal("omitted repository description was rendered as an explicit value")
	}
	if got := providerSpec["workflows"]; !provisioningEqual(got, []any{}) {
		t.Fatalf("workflows = %#v, want explicit empty list", got)
	}
	if got := providerSpec["sync"]; !provisioningEqual(got, map[string]any{"enabled": false, "target": "folderless", "intervalSeconds": float64(120)}) {
		t.Fatalf("sync = %#v, want explicit folderless configuration", got)
	}
	if got := nestedMap(t, providerSpec, "github")["generateDashboardPreviews"]; got != true {
		t.Fatalf("github.generateDashboardPreviews = %v, want true", got)
	}
	if got := nestedMap(t, providerSpec, "commit")["smimeCertificate"]; got != "-----BEGIN CERTIFICATE----- public -----END CERTIFICATE-----" {
		t.Fatalf("commit.smimeCertificate = %v", got)
	}
	if got := nestedMap(t, forProvider, "secure"); !provisioningEqual(got, map[string]any{
		"token": map[string]any{"name": "repository-token"}, "webhookSecret": map[string]any{"name": "repository-webhook"},
		"commitSigningKey": map[string]any{"name": "repository-signing-key"},
	}) {
		t.Fatalf("secure = %#v, want stable name references", got)
	}
	if got := forProvider["secureVersion"]; got != float64(2) {
		t.Fatalf("secureVersion = %v, want 2", got)
	}
}

func TestProvisioningRepositoryClaimsShareAStackWithoutSharingRepositoryOwnership(t *testing.T) {
	first, err := renderProvisioningRepository(provisioningRepositoryClaim("first-subtree", provisioningRepositorySpec("github")), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := renderProvisioningRepository(provisioningRepositoryClaim("second-subtree", provisioningRepositorySpec("gitlab")), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	firstRepository := first["repository"].Resource.UnstructuredContent()
	secondRepository := second["repository"].Resource.UnstructuredContent()
	if got := nestedMap(t, firstRepository, "spec", "providerConfigRef")["name"]; got != "teamdemo01" {
		t.Fatalf("first repository ProviderConfig = %v, want teamdemo01", got)
	}
	if got := nestedMap(t, secondRepository, "spec", "providerConfigRef")["name"]; got != "teamdemo01" {
		t.Fatalf("second repository ProviderConfig = %v, want teamdemo01", got)
	}
	if firstRepository["metadata"].(map[string]any)["name"] == secondRepository["metadata"].(map[string]any)["name"] {
		t.Fatal("two repository claims for one stack rendered the same managed resource name")
	}
	if desiredExternalName(t, firstRepository) == desiredExternalName(t, secondRepository) {
		t.Fatal("two repository claims for one stack rendered the same repository UID")
	}
}

func TestProvisioningRepositoryRejectsCredentialBearingURLsAtReconcile(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
	}{
		{name: "provider URL", mutate: func(repository map[string]any) {
			repository["github"].(map[string]any)["url"] = "https://user:credential@example.invalid/dashboards"
		}},
		{name: "webhook URL", mutate: func(repository map[string]any) {
			repository["webhook"] = map[string]any{"baseUrl": "https://user:credential@example.invalid/hooks"}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			claim := provisioningRepositoryClaim("userinfo", provisioningRepositorySpec("github"))
			tc.mutate(claim["spec"].(map[string]any)["repository"].(map[string]any))
			if desired, err := renderProvisioningRepository(claim, nil, nil); err == nil || !strings.Contains(err.Error(), "without URL userinfo") || len(desired) != 0 {
				t.Fatalf("credential-bearing URL = desired %v, error %v", desired, err)
			}
		})
	}
}

func TestProvisioningRepositoryAllProviderTypesRoundTripPinnedProviderCRD(t *testing.T) {
	env := &admissionEnv{paths: []string{"../apis/provisioning-v1beta1.yaml"}}
	if err := env.Start(t); err != nil {
		t.Fatalf("start provisioning admission environment: %v", err)
	}
	t.Cleanup(func() {
		if err := env.Stop(); err != nil {
			t.Errorf("stop provisioning admission environment: %v", err)
		}
	})
	installProviderAdmissionCRDs(t, env)

	for _, providerType := range []string{"local", "github", "githubEnterprise", "git", "bitbucket", "gitlab"} {
		t.Run(providerType, func(t *testing.T) {
			claim := provisioningRepositoryClaim("provider-"+strings.ToLower(providerType), provisioningRepositorySpec(providerType))
			desired, err := renderProvisioningRepository(claim, nil, nil)
			if err != nil {
				t.Fatalf("render %s repository: %v", providerType, err)
			}
			providerSpec := nestedMap(t, desired["repository"].Resource.UnstructuredContent(), "spec", "forProvider", "spec")
			if got := providerSpec["type"]; got != providerType {
				t.Fatalf("rendered provider type = %v, want %s", got, providerType)
			}
			if _, found := providerSpec[providerType]; !found {
				t.Fatalf("rendered provider block %q is absent", providerType)
			}
			admitProviderChildren(t, env, desired)
		})
	}
}

func TestProvisioningRepositoryAdmissionRejectsMismatchesInstanceAndInsecureShapesOnCreateAndUpdate(t *testing.T) {
	env := &admissionEnv{paths: []string{"../apis/provisioning-v1beta1.yaml"}}
	if err := env.Start(t); err != nil {
		t.Fatalf("start provisioning admission environment: %v", err)
	}
	t.Cleanup(func() {
		if err := env.Stop(); err != nil {
			t.Errorf("stop provisioning admission environment: %v", err)
		}
	})
	ctx := context.Background()
	allowed := provisioningRepositoryAdmissionRequest("provisioning-allowed")
	if err := env.Apply(ctx, allowed); err != nil {
		t.Fatalf("allowed provisioning repository was refused: %v", err)
	}
	retargeted := allowed.DeepCopy()
	retargeted.Object["spec"].(map[string]any)["stackRef"].(map[string]any)["name"] = "teamdemo02"
	if err := env.Apply(ctx, retargeted); err == nil || !strings.Contains(err.Error(), "stackRef.name is immutable") {
		t.Fatalf("stackRef update error = %v, want immutability denial", err)
	}
	retargetedUID := allowed.DeepCopy()
	retargetedUID.Object["spec"].(map[string]any)["repository"].(map[string]any)["uid"] = "other-repository"
	if err := env.Apply(ctx, retargetedUID); err == nil || !strings.Contains(err.Error(), "repository.uid is immutable") {
		t.Fatalf("repository UID update error = %v, want immutability denial", err)
	}

	for _, tc := range []struct {
		name    string
		mutate  func(*unstructured.Unstructured)
		message string
	}{
		{
			name: "type block mismatch", message: "repository type requires exactly its matching provider block",
			mutate: func(obj *unstructured.Unstructured) {
				obj.Object["spec"].(map[string]any)["repository"].(map[string]any)["type"] = "gitlab"
			},
		},
		{
			name: "instance sync target", message: "sync.target instance is refused because one declarative owner is allowed per folder subtree",
			mutate: func(obj *unstructured.Unstructured) {
				obj.Object["spec"].(map[string]any)["repository"].(map[string]any)["sync"].(map[string]any)["target"] = "instance"
			},
		},
		{
			name: "secure value literal", message: "Invalid value",
			mutate: func(obj *unstructured.Unstructured) {
				obj.Object["spec"].(map[string]any)["repository"].(map[string]any)["secure"] = map[string]any{"token": "not-a-reference"}
			},
		},
		{
			name: "secure value creation shape", message: "secure values must reference an existing secure value by name; create is forbidden",
			mutate: func(obj *unstructured.Unstructured) {
				obj.Object["spec"].(map[string]any)["repository"].(map[string]any)["secure"] = map[string]any{"token": map[string]any{"name": "existing-secure-value", "create": map[string]any{"value": "not-a-reference"}}}
			},
		},
	} {
		for _, operation := range []string{"create", "update"} {
			t.Run(tc.name+" "+operation, func(t *testing.T) {
				var candidate *unstructured.Unstructured
				if operation == "create" {
					candidate = provisioningRepositoryAdmissionRequest("provisioning-" + strings.ReplaceAll(tc.name, " ", "-"))
				} else {
					candidate = allowed.DeepCopy()
				}
				tc.mutate(candidate)
				err := env.Apply(ctx, candidate)
				if err == nil || !strings.Contains(err.Error(), tc.message) {
					t.Fatalf("%s error = %v, want %q", operation, err, tc.message)
				}
				t.Logf("%s refused: %v", operation, err)
			})
		}
	}

	const (
		xrdPath      = "../apis/provisioning-v1beta1.yaml"
		originalRule = "- rule: '!has(self.create)'"
		weakenedRule = "- rule: 'true'"
		denial       = "secure values must reference an existing secure value by name; create is forbidden"
	)
	original, err := os.ReadFile(xrdPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := bytes.Count(original, []byte(originalRule)); got != 1 {
		t.Fatalf("secure create guard count = %d, want 1", got)
	}
	weakened := bytes.Replace(original, []byte(originalRule), []byte(weakenedRule), 1)
	scratch := filepath.Join(t.TempDir(), "provisioning-v1beta1.yaml")
	if err := os.WriteFile(scratch, weakened, 0o600); err != nil {
		t.Fatal(err)
	}
	weakenedCRD, err := crdFromXRD(scratch)
	if err != nil {
		t.Fatal(err)
	}
	if err := env.Apply(ctx, weakenedCRD); err != nil {
		t.Fatalf("apply weakened secure create guard: %v", err)
	}
	weakenedRequest := provisioningRepositoryAdmissionRequest("provisioning-secure-weakened")
	weakenedRequest.Object["spec"].(map[string]any)["repository"].(map[string]any)["secure"] = map[string]any{"token": map[string]any{"name": "existing-secure-value", "create": map[string]any{"value": "not-a-reference"}}}
	if err := env.Apply(ctx, weakenedRequest); err != nil {
		t.Fatalf("weakened secure create guard did not admit: %v", err)
	}
	t.Log("API-SERVER weakened repository secure create guard: admitted")
	originalCRD, err := crdFromXRD(xrdPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := env.Apply(ctx, originalCRD); err != nil {
		t.Fatalf("restore secure create guard: %v", err)
	}
	restored := provisioningRepositoryAdmissionRequest("provisioning-secure-restored")
	restored.Object["spec"].(map[string]any)["repository"].(map[string]any)["secure"] = map[string]any{"token": map[string]any{"name": "existing-secure-value", "create": map[string]any{"value": "not-a-reference"}}}
	if err := env.Apply(ctx, restored); err == nil || !strings.Contains(err.Error(), denial) {
		t.Fatalf("restored secure create guard = %v, want own denial", err)
	}
}

func TestProvisioningRepositoryAdmissionRejectsCredentialBearingURL(t *testing.T) {
	const (
		xrdPath      = "../apis/provisioning-v1beta1.yaml"
		originalRule = `- rule: "!self.matches('^https://[^/]*@')"`
		weakenedRule = `- rule: "true"`
		denial       = "URL userinfo is forbidden because credentials must use secure value references"
	)
	env := &admissionEnv{paths: []string{xrdPath}}
	if err := env.Start(t); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = env.Stop() })
	ctx := context.Background()
	valid := provisioningRepositoryAdmissionRequest("provisioning-url-valid")
	if err := env.Apply(ctx, valid); err != nil {
		t.Fatalf("valid provisioning repository: %v", err)
	}
	for _, operation := range []string{"create", "update"} {
		var candidate *unstructured.Unstructured
		if operation == "create" {
			candidate = provisioningRepositoryAdmissionRequest("provisioning-url-" + operation)
		} else {
			candidate = valid.DeepCopy()
		}
		repository := candidate.Object["spec"].(map[string]any)["repository"].(map[string]any)
		repository["github"].(map[string]any)["url"] = "https://user:credential@example.invalid/repository"
		if err := env.Apply(ctx, candidate); err == nil || !strings.Contains(err.Error(), denial) {
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
	scratch := filepath.Join(t.TempDir(), "provisioning-v1beta1.yaml")
	if err := os.WriteFile(scratch, weakened, 0o600); err != nil {
		t.Fatal(err)
	}
	weakenedCRD, err := crdFromXRD(scratch)
	if err != nil {
		t.Fatal(err)
	}
	if err := env.Apply(ctx, weakenedCRD); err != nil {
		t.Fatalf("apply weakened URL guard: %v", err)
	}
	weakenedRequest := provisioningRepositoryAdmissionRequest("provisioning-url-weakened")
	weakenedRequest.Object["spec"].(map[string]any)["repository"].(map[string]any)["github"].(map[string]any)["url"] = "https://user:credential@example.invalid/repository"
	if err := env.Apply(ctx, weakenedRequest); err != nil {
		t.Fatalf("weakened URL guard did not admit: %v", err)
	}
	t.Log("API-SERVER weakened repository URL userinfo guard: admitted")
	originalCRD, err := crdFromXRD(xrdPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := env.Apply(ctx, originalCRD); err != nil {
		t.Fatalf("restore URL guard: %v", err)
	}
	restored := provisioningRepositoryAdmissionRequest("provisioning-url-restored")
	restored.Object["spec"].(map[string]any)["repository"].(map[string]any)["github"].(map[string]any)["url"] = "https://user:credential@example.invalid/repository"
	if err := env.Apply(ctx, restored); err == nil || !strings.Contains(err.Error(), denial) {
		t.Fatalf("restored URL guard = %v, want own denial", err)
	}
}

func provisioningRepositoryDocument(additions map[string]any) string {
	spec := map[string]any{
		"stackRef": map[string]any{"name": "teamdemo01"},
		"repository": map[string]any{
			"uid": "application-overview", "title": "Application overview", "description": "Git-backed dashboards for the application overview subtree.",
			"type": "github", "github": map[string]any{"url": "https://github.com/example/grafana-dashboards", "branch": "main", "path": "application-overview"},
			"connectionRef": map[string]any{"name": "shared-github-app"},
			"sync":          map[string]any{"enabled": true, "target": "folder", "intervalSeconds": float64(60)}, "workflows": []any{},
		},
	}
	for key, value := range additions {
		spec[key] = value
	}
	return mustJSON(map[string]any{
		"apiVersion": "platform.example.org/v1beta1", "kind": "GrafanaProvisioningRepository",
		"metadata": map[string]any{"name": "application-overview", "namespace": "grafana-vending"},
		"spec":     spec,
	})
}

func provisioningRepositoryClaim(name string, repository map[string]any) map[string]any {
	return map[string]any{
		"apiVersion": "platform.example.org/v1beta1", "kind": "GrafanaProvisioningRepository",
		"metadata": map[string]any{"name": name, "namespace": "grafana-vending"},
		"spec":     map[string]any{"stackRef": map[string]any{"name": "teamdemo01"}, "repository": repository},
	}
}

func provisioningRepositorySpec(providerType string) map[string]any {
	repository := map[string]any{
		"uid": "repository-" + strings.ToLower(providerType), "title": "Provisioning repository " + providerType,
		"type": providerType, "connectionRef": map[string]any{"name": "shared-git-connection"},
		"sync": map[string]any{"enabled": true, "target": "folder", "intervalSeconds": float64(60)}, "workflows": []any{},
	}
	switch providerType {
	case "local":
		repository["local"] = map[string]any{"path": "local-dashboards"}
	case "github":
		repository["github"] = map[string]any{"url": "https://github.example.invalid/dashboards", "branch": "main", "path": "dashboards"}
	case "githubEnterprise":
		repository["githubEnterprise"] = map[string]any{"url": "https://github.example.invalid/dashboards", "serverUrl": "https://github.example.invalid", "branch": "main", "path": "dashboards"}
	case "git":
		repository["git"] = map[string]any{"url": "https://git.example.invalid/dashboards", "branch": "main", "path": "dashboards", "tokenUser": "automation"}
	case "bitbucket":
		repository["bitbucket"] = map[string]any{"url": "https://bitbucket.example.invalid/dashboards", "branch": "main", "path": "dashboards", "tokenUser": "x-token-auth"}
	case "gitlab":
		repository["gitlab"] = map[string]any{"url": "https://gitlab.example.invalid/dashboards", "branch": "main", "path": "dashboards"}
	}
	return repository
}

func provisioningRepositoryAdmissionRequest(name string) *unstructured.Unstructured {
	request := provisioningRepositoryClaim(name, provisioningRepositorySpec("github"))
	request["metadata"].(map[string]any)["namespace"] = "default"
	return &unstructured.Unstructured{Object: request}
}

func provisioningEqual(got, want any) bool {
	return mustJSON(got) == mustJSON(want)
}

func runProvisioning(t *testing.T, composite string) *fnv1.RunFunctionResponse {
	t.Helper()
	rsp := callProvisioning(t, composite)
	if fatal := fatalResult(rsp); fatal != "" {
		t.Fatalf("RunFunction returned a fatal result: %s", fatal)
	}
	return rsp
}

func callProvisioning(t *testing.T, composite string) *fnv1.RunFunctionResponse {
	return callFunctionWithRequiredResources(t, composite, nil, requiredStackResource("teamdemo01", "grafana-vending", "True"), requiredResourceCapabilities())
}
