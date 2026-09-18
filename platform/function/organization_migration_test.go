package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestPreOrganizationStackRequestOrganizationMigrationAcrossSchemaUpgrade(t *testing.T) {
	const currentXRD = "../apis/stack-v1beta1.yaml"
	oldXRD := preOrganizationStackXRD(t, currentXRD)

	env := &admissionEnv{paths: []string{oldXRD}}
	if err := env.Start(t); err != nil {
		t.Fatalf("start pre-organization admission environment: %v", err)
	}
	t.Cleanup(func() {
		if err := env.Stop(); err != nil {
			t.Errorf("stop pre-organization admission environment: %v", err)
		}
	})

	ctx := context.Background()
	for _, name := range []string{"migrationonly01", "migrationsibling01"} {
		if err := env.Apply(ctx, preOrganizationStackRequest(name)); err != nil {
			t.Fatalf("persist pre-organization request %s: %v", name, err)
		}
	}

	currentCRD, err := crdFromXRD(currentXRD)
	if err != nil {
		t.Fatalf("derive current stack CRD: %v", err)
	}
	if err := env.Apply(ctx, currentCRD); err != nil {
		t.Fatalf("replace pre-organization CRD with current schema: %v", err)
	}

	organizationOnly := persistedStackRequest(t, ctx, env, "migrationonly01")
	organizationOnly.Object["spec"].(map[string]any)["organization"] = "example-primary"
	organizationOnlyErr := env.Apply(ctx, organizationOnly)
	t.Logf("API-SERVER organization-only update response: %v", organizationOnlyErr)

	organizationAndSibling := persistedStackRequest(t, ctx, env, "migrationsibling01")
	organizationAndSibling.Object["spec"].(map[string]any)["organization"] = "example-primary"
	organizationAndSibling.Object["spec"].(map[string]any)["displayName"] = "Migration sibling changed"
	organizationAndSiblingErr := env.Apply(ctx, organizationAndSibling)
	t.Logf("API-SERVER organization-and-sibling update response: %v", organizationAndSiblingErr)

	const denial = "spec.organization is immutable"
	if organizationOnlyErr == nil || !strings.Contains(organizationOnlyErr.Error(), denial) {
		t.Fatalf("organization-only update response = %v, want %q", organizationOnlyErr, denial)
	}
	if organizationAndSiblingErr == nil || !strings.Contains(organizationAndSiblingErr.Error(), denial) {
		t.Fatalf("organization-and-sibling update response = %v, want %q", organizationAndSiblingErr, denial)
	}
}

// preOrganizationStackXRD reconstructs the exact organization schema delta in
// reverse while retaining the rest of the current XRD. Count checks make the
// fixture fail closed if the released transition rule or field shape changes.
func preOrganizationStackXRD(t *testing.T, currentPath string) string {
	t.Helper()
	raw, err := os.ReadFile(currentPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, removal := range [][]byte{
		[]byte("            - rule: self.spec.organization == oldSelf.spec.organization\n              message: spec.organization is immutable\n"),
		[]byte("                - organization\n"),
		[]byte("                organization:\n                  type: string\n                  minLength: 1\n                  maxLength: 63\n                  pattern: ^[a-z0-9]([a-z0-9-]*[a-z0-9])?$\n                  description: Immutable platform-owned key resolved through the organization registry.\n"),
	} {
		if got := bytes.Count(raw, removal); got != 1 {
			t.Fatalf("pre-organization XRD reverse-delta match count = %d, want 1 for %q", got, removal)
		}
		raw = bytes.Replace(raw, removal, nil, 1)
	}
	path := filepath.Join(t.TempDir(), "stack-pre-organization-v1beta1.yaml")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func preOrganizationStackRequest(name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "platform.example.org/v1beta1",
		"kind":       "GrafanaCloudStackRequest",
		"metadata": map[string]any{
			"name":      name,
			"namespace": "default",
		},
		"spec": map[string]any{
			"displayName": "Migration evidence",
			"slug":        name,
			"region":      "prod-us-central-0",
			"usage":       "development",
		},
	}}
}

func persistedStackRequest(t *testing.T, ctx context.Context, env *admissionEnv, name string) *unstructured.Unstructured {
	t.Helper()
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("platform.example.org/v1beta1")
	obj.SetKind("GrafanaCloudStackRequest")
	obj.SetName(name)
	obj.SetNamespace("default")
	if err := env.client.Get(ctx, clientObjectKey(obj), obj); err != nil {
		t.Fatalf("read persisted stack request %s: %v", name, err)
	}
	return obj
}

func clientObjectKey(obj *unstructured.Unstructured) client.ObjectKey {
	return client.ObjectKeyFromObject(obj)
}
