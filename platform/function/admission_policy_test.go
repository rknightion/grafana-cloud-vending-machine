package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

// installCompositionAdmissionPolicies uses the same Composition input as the
// function, so admission does not carry a second copy of a platform ceiling.
// testdata/compositions-crd.json is the unmodified JSON encoding of Crossplane
// v2.4.0 cluster/crds/apiextensions.crossplane.io_compositions.yaml.
func installCompositionAdmissionPolicies(t testing.TB, e *admissionEnv, paths []string) error {
	t.Helper()
	var compositions, policies, bindings []*unstructured.Unstructured
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(raw), 4096)
		for {
			obj := &unstructured.Unstructured{}
			if err := decoder.Decode(&obj.Object); err == io.EOF {
				break
			} else if err != nil {
				return err
			}
			switch obj.GetAPIVersion() + "/" + obj.GetKind() {
			case "apiextensions.crossplane.io/v1/Composition":
				compositions = append(compositions, obj)
			case "admissionregistration.k8s.io/v1/ValidatingAdmissionPolicy":
				policies = append(policies, obj)
			case "admissionregistration.k8s.io/v1/ValidatingAdmissionPolicyBinding":
				bindings = append(bindings, obj)
			}
		}
	}
	if len(policies) == 0 && len(bindings) == 0 {
		return nil
	}
	if len(policies) == 0 || len(bindings) == 0 {
		return fmt.Errorf("admission policies and bindings must both be present")
	}
	raw, err := os.ReadFile("testdata/compositions-crd.json")
	if err != nil {
		return err
	}
	crd := &apiextensionsv1.CustomResourceDefinition{}
	if err := json.Unmarshal(raw, crd); err != nil {
		return err
	}
	if _, err := envtest.InstallCRDs(e.environment.Config, envtest.CRDInstallOptions{CRDs: []*apiextensionsv1.CustomResourceDefinition{crd}}); err != nil {
		return err
	}
	for _, group := range [][]*unstructured.Unstructured{compositions, policies, bindings} {
		for _, obj := range group {
			if err := e.Apply(context.Background(), obj); err != nil {
				return fmt.Errorf("install %s/%s: %w", obj.GetKind(), obj.GetName(), err)
			}
		}
	}
	return nil
}
