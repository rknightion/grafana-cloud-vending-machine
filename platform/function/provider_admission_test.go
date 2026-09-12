package main

import (
	"context"
	"encoding/json"
	"os"
	"sort"
	"testing"

	"github.com/crossplane/function-sdk-go/resource"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

// The fixture contains complete, unmodified CRDs from provider-grafana v2.14.0,
// source dc795606df97a72dce81a0c953e0ec0750e0b489. These are the commissioned
// managed kinds, not a hand-written approximation of their provider schemas.
//
// A top-level status stanza is optional here and carries no information: the
// generated CRDs ship an empty one (acceptedNames blank, conditions and
// storedVersions null) and InstallCRDs ignores it, so entries added by different
// passes disagree about whether it is present. Do not normalise the file to make
// them agree, and never gate a newly imported kind on byte equality against a
// neighbour — compare it against the pinned package instead.
func installProviderAdmissionCRDs(t *testing.T, e *admissionEnv) {
	t.Helper()
	raw, err := os.ReadFile("testdata/provider-crds.json")
	if err != nil {
		t.Fatal(err)
	}
	var crds []*apiextensionsv1.CustomResourceDefinition
	if err := json.Unmarshal(raw, &crds); err != nil {
		t.Fatal(err)
	}
	if _, err := envtest.InstallCRDs(e.environment.Config, envtest.CRDInstallOptions{CRDs: crds}); err != nil {
		t.Fatal(err)
	}
}

func admitProviderChildren(t *testing.T, e *admissionEnv, children map[resource.Name]*resource.DesiredComposed) {
	t.Helper()
	names := make([]string, 0, len(children))
	for name := range children {
		names = append(names, string(name))
	}
	sort.Strings(names)
	for _, name := range names {
		child := children[resource.Name(name)].Resource
		if child.GetKind() == "PushSecret" {
			continue
		} // Existing support-resource contract.
		t.Run(name, func(t *testing.T) {
			obj := &unstructured.Unstructured{}
			if err := json.Unmarshal([]byte(mustJSON(child.UnstructuredContent())), &obj.Object); err != nil {
				t.Fatal(err)
			}
			ns := &unstructured.Unstructured{Object: map[string]any{"apiVersion": "v1", "kind": "Namespace", "metadata": map[string]any{"name": obj.GetNamespace()}}}
			if err := e.client.Create(context.Background(), ns); err != nil && !apierrors.IsAlreadyExists(err) {
				t.Fatal(err)
			}
			if err := e.Apply(context.Background(), obj); err != nil {
				t.Fatal(err)
			}
			persisted := &unstructured.Unstructured{}
			persisted.SetGroupVersionKind(obj.GroupVersionKind())
			if err := e.client.Get(context.Background(), client.ObjectKeyFromObject(obj), persisted); err != nil {
				t.Fatal(err)
			}
			assertPersistedSubset(t, "spec", obj.Object["spec"], persisted.Object["spec"])
			t.Logf("API-SERVER admitted and retained %s %s: %s", obj.GetAPIVersion(), obj.GetKind(), mustJSON(persisted.Object["spec"]))
		})
	}
}

func assertPersistedSubset(t *testing.T, path string, expected, actual any) {
	t.Helper()
	switch value := expected.(type) {
	case map[string]any:
		got, ok := actual.(map[string]any)
		if !ok {
			t.Fatalf("%s lost object", path)
		}
		for key, item := range value {
			assertPersistedSubset(t, path+"."+key, item, got[key])
		}
	case []any:
		got, ok := actual.([]any)
		if !ok || len(got) != len(value) {
			t.Fatalf("%s changed list: %v", path, actual)
		}
		for i, item := range value {
			assertPersistedSubset(t, path, item, got[i])
		}
	default:
		if mustJSON(expected) != mustJSON(actual) {
			t.Fatalf("%s changed or pruned: expected %s, got %s", path, mustJSON(expected), mustJSON(actual))
		}
	}
}
