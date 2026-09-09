package main

import (
	"bytes"
	"os"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

func TestEveryImplementedSurfaceHasAnEnforcedComposition(t *testing.T) {
	for _, path := range xrdPaths() {
		crds, err := crdsFromXRD(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, crd := range crds {
			kind := crd.Spec.Names.Kind
			if !compositeRenderers[kind].implemented {
				continue
			}
			t.Run(kind, func(t *testing.T) {
				documents, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				var xrd, composition map[string]any
				for _, doc := range bytes.Split(documents, []byte("\n---\n")) {
					var obj map[string]any
					if err := yaml.Unmarshal(doc, &obj); err != nil {
						t.Fatal(err)
					}
					if obj["kind"] == "CompositeResourceDefinition" {
						k, _, _ := unstructured.NestedString(obj, "spec", "names", "kind")
						if k == kind {
							xrd = obj
						}
					}
					if obj["kind"] == "Composition" {
						k, _, _ := unstructured.NestedString(obj, "spec", "compositeTypeRef", "kind")
						if k == kind {
							composition = obj
						}
					}
				}
				enforced, _, _ := unstructured.NestedString(xrd, "spec", "enforcedCompositionRef", "name")
				name, _, _ := unstructured.NestedString(composition, "metadata", "name")
				if enforced == "" || name != enforced {
					t.Fatalf("%s: implemented %s lacks its enforced Composition %q", path, kind, enforced)
				}
				pipeline, _, _ := unstructured.NestedSlice(composition, "spec", "pipeline")
				found := false
				for _, step := range pipeline {
					ref, _, _ := unstructured.NestedString(step.(map[string]any), "functionRef", "name")
					found = found || ref == "function-grafana-vending"
				}
				if !found {
					t.Fatalf("%s: Composition %s does not call the vending function", path, name)
				}
			})
		}
	}
}
