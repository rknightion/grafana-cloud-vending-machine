package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/crossplane/function-sdk-go/resource"
)

func TestRefusedVendorShapeAssertionCoversEveryEmittedKind(t *testing.T) {
	repo := coverageRepositoryRoot(t)
	sources := parseFunctionSources(t, filepath.Join(repo, "platform", "function"))
	emitted := newDesiredKinds(t, sources)

	children := map[resource.Name]*resource.DesiredComposed{}
	for key, gvk := range emitted {
		children[resource.Name(key)] = refusedShapeDesired(gvk, map[string]any{})
	}
	if err := refusedVendorShapeError(children); err != nil {
		t.Fatalf("clean fixture for every emitted child kind: %v", err)
	}

	for _, shape := range refusedVendorShapes {
		if _, found := emitted[shape.gvk().key()]; !found {
			t.Fatalf("refused-shape record targets %s, which is not an emitted child kind", shape.gvk().key())
		}
	}

	refused := refusedShapeFixture(refusedVendorShapes[0])
	err := refusedVendorShapeError(refused)
	if err == nil {
		t.Fatal("controlled fixture with a known refused shape passed")
	}
	if os.Getenv("REFUSED_SHAPES_PROVE_FAILURE") == "1" {
		t.Fatalf("controlled refused-shape fixture failed as expected: %v", err)
	}
	t.Logf("controlled refused-shape fixture was rejected: %v", err)
}

func refusedShapeDesired(gvk managedGVK, spec map[string]any) *resource.DesiredComposed {
	return newDesired(gvk.APIVersion, gvk.Kind, "test", "test", nil, spec)
}

func refusedShapeFixture(shape refusedVendorShape) map[resource.Name]*resource.DesiredComposed {
	value := map[string]any{}
	for index := len(shape.FieldPath) - 1; index >= 1; index-- {
		value = map[string]any{shape.FieldPath[index]: value}
	}
	return map[resource.Name]*resource.DesiredComposed{
		"controlled-refused-shape": refusedShapeDesired(shape.gvk(), value),
	}
}
