package main

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const admissionHarnessImplemented = false

// crdFromXRD derives the composite CRD without changing its validation rules.
func crdFromXRD(path string) (*apiextensionsv1.CustomResourceDefinition, error) {
	return nil, errors.New("admission harness not implemented")
}

// admissionEnv owns an ephemeral local control plane. Start will install every XRD;
// Apply will create or update an object and return the server error unchanged.
type admissionEnv struct{}

func (e *admissionEnv) Start(t testing.TB) error {
	return errors.New("admission harness not implemented")
}

func (e *admissionEnv) Stop() error {
	return errors.New("admission harness not implemented")
}

func (e *admissionEnv) Apply(ctx context.Context, obj client.Object) error {
	return errors.New("admission harness not implemented")
}

// xrdPaths is evaluated from the Go package directory, as with all package tests.
func xrdPaths() []string {
	paths, err := filepath.Glob("../apis/*.yaml")
	if err != nil {
		panic(err)
	}
	return paths
}
