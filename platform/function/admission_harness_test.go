package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/xcrd"
	xapiextensionsv1 "github.com/crossplane/crossplane/apis/v2/apiextensions/v1"
	xapiextensionsv2 "github.com/crossplane/crossplane/apis/v2/apiextensions/v2"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

const admissionHarnessImplemented = true

// crdFromXRD derives the composite CRD without changing its validation rules.
func crdFromXRD(path string) (*apiextensionsv1.CustomResourceDefinition, error) {
	crds, err := crdsFromXRD(path)
	if err != nil {
		return nil, err
	}
	if len(crds) != 1 {
		return nil, fmt.Errorf("%s contains %d CompositeResourceDefinitions; expected exactly one", path, len(crds))
	}
	return crds[0], nil
}

// crdsFromXRD derives one CRD for every XRD document in a multi-document file.
func crdsFromXRD(path string) ([]*apiextensionsv1.CustomResourceDefinition, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	decoder := yaml.NewYAMLOrJSONDecoder(f, 4096)
	var crds []*apiextensionsv1.CustomResourceDefinition
	for {
		var document map[string]any
		err := decoder.Decode(&document)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode %s: %w", path, err)
		}
		if len(document) == 0 {
			continue
		}

		object := &unstructured.Unstructured{Object: document}
		if object.GetAPIVersion() != xapiextensionsv2.SchemeGroupVersion.String() || object.GetKind() != xapiextensionsv2.CompositeResourceDefinitionKind {
			continue
		}

		xrd := &xapiextensionsv1.CompositeResourceDefinition{}
		data, err := json.Marshal(document)
		if err != nil {
			return nil, fmt.Errorf("marshal %s: %w", path, err)
		}
		if err := json.Unmarshal(data, xrd); err != nil {
			return nil, fmt.Errorf("decode XRD %s: %w", path, err)
		}
		if xrd.Spec.Scope == nil {
			scope := xapiextensionsv1.CompositeResourceScopeNamespaced
			xrd.Spec.Scope = &scope
		}

		crd, err := xcrd.ForCompositeResource(xrd)
		if err != nil {
			return nil, fmt.Errorf("derive CRD from %s: %w", path, err)
		}
		crd.SetOwnerReferences(nil)
		crds = append(crds, crd)
	}
	return crds, nil
}

// admissionEnv owns an ephemeral local control plane. Start will install every XRD;
// Apply will create or update an object and return the server error unchanged.
type admissionEnv struct {
	environment *envtest.Environment
	client      client.Client
	paths       []string
	installed   int
}

func (e *admissionEnv) Start(t testing.TB) error {
	t.Helper()
	if e.environment != nil {
		return fmt.Errorf("admission environment is already running")
	}
	if got, want := os.Getenv("ENVTEST_KUBERNETES_VERSION"), "1.37.0"; got != want {
		return fmt.Errorf("ENVTEST_KUBERNETES_VERSION = %q, want %q", got, want)
	}
	assets := os.Getenv("KUBEBUILDER_ASSETS")
	if assets == "" {
		return fmt.Errorf("KUBEBUILDER_ASSETS is not set")
	}
	for _, binary := range []string{"kube-apiserver", "etcd"} {
		info, err := os.Stat(filepath.Join(assets, binary))
		if err != nil {
			return fmt.Errorf("envtest binary %s: %w", binary, err)
		}
		if info.Mode()&0o111 == 0 {
			return fmt.Errorf("envtest binary %s is not executable", binary)
		}
	}

	paths := e.paths
	if len(paths) == 0 {
		paths = xrdPaths()
	}
	if len(paths) == 0 {
		return fmt.Errorf("no XRD paths found")
	}
	type sourceCRDs struct {
		path string
		crds []*apiextensionsv1.CustomResourceDefinition
	}
	sources := make([]sourceCRDs, 0, len(paths))
	for _, path := range paths {
		crds, err := crdsFromXRD(path)
		if err != nil {
			return fmt.Errorf("derive XRD %s: %w", filepath.Base(path), err)
		}
		if len(crds) == 0 {
			return fmt.Errorf("XRD source %s contains no CompositeResourceDefinition", filepath.Base(path))
		}
		sources = append(sources, sourceCRDs{path: path, crds: crds})
	}

	useExistingCluster := false
	e.environment = &envtest.Environment{
		BinaryAssetsDirectory: assets,
		UseExistingCluster:    &useExistingCluster,
	}
	config, err := e.environment.Start()
	if err != nil {
		e.environment = nil
		return fmt.Errorf("start envtest control plane: %w", err)
	}
	installed := 0
	var installErrors []error
	for _, source := range sources {
		for _, crd := range source.crds {
			if _, err := envtest.InstallCRDs(config, envtest.CRDInstallOptions{CRDs: []*apiextensionsv1.CustomResourceDefinition{crd}}); err != nil {
				installErrors = append(installErrors, fmt.Errorf("installing XRD %s (%s): %w", filepath.Base(source.path), crd.Name, err))
				continue
			}
			installed++
		}
	}
	e.installed = installed
	if len(installErrors) > 0 {
		_ = e.Stop()
		return fmt.Errorf("installed %d/%d derived CRDs: %w", installed, installed+len(installErrors), errors.Join(installErrors...))
	}
	e.client, err = client.New(config, client.Options{})
	if err != nil {
		_ = e.Stop()
		return fmt.Errorf("create envtest client: %w", err)
	}
	return nil
}

func (e *admissionEnv) Stop() error {
	if e.environment == nil {
		return nil
	}
	err := e.environment.Stop()
	e.environment = nil
	e.client = nil
	e.installed = 0
	return err
}

func (e *admissionEnv) Apply(ctx context.Context, obj client.Object) error {
	if e.client == nil {
		return fmt.Errorf("admission environment is not running")
	}
	existing, ok := obj.DeepCopyObject().(client.Object)
	if !ok {
		return fmt.Errorf("%T does not implement client.Object after DeepCopyObject", obj)
	}
	if err := e.client.Get(ctx, client.ObjectKeyFromObject(obj), existing); err != nil {
		if apierrors.IsNotFound(err) {
			return e.client.Create(ctx, obj)
		}
		return err
	}
	obj.SetResourceVersion(existing.GetResourceVersion())
	return e.client.Update(ctx, obj)
}

// xrdPaths is evaluated from the Go package directory, as with all package tests.
func xrdPaths() []string {
	paths, err := filepath.Glob("../apis/*.yaml")
	if err != nil {
		panic(err)
	}
	return paths
}

func TestCRDFromXRDDerivesSingleDocument(t *testing.T) {
	crd, err := crdFromXRD("../apis/stack-v1beta1.yaml")
	if err != nil {
		t.Fatalf("crdFromXRD() error = %v", err)
	}
	if got, want := crd.Name, "grafanacloudstackrequests.platform.example.org"; got != want {
		t.Fatalf("CRD name = %q, want %q", got, want)
	}
	if got, want := len(crd.Spec.Versions), 1; got != want {
		t.Fatalf("version count = %d, want %d", got, want)
	}
	if got := len(crd.Spec.Versions[0].Schema.OpenAPIV3Schema.XValidations); got == 0 {
		t.Fatal("derived CRD discarded XRD validation rules")
	}
}

func TestCRDFromXRDDefaultsV2ScopeToNamespaced(t *testing.T) {
	source, err := os.ReadFile("../apis/stack-v1beta1.yaml")
	if err != nil {
		t.Fatal(err)
	}
	withoutScope := bytes.Replace(source, []byte("  scope: Namespaced\n"), nil, 1)
	if bytes.Equal(withoutScope, source) {
		t.Fatal("XRD scope was not found")
	}
	scratch := filepath.Join(t.TempDir(), "stack-v1beta1.yaml")
	if err := os.WriteFile(scratch, withoutScope, 0o600); err != nil {
		t.Fatal(err)
	}

	crd, err := crdFromXRD(scratch)
	if err != nil {
		t.Fatalf("crdFromXRD() error = %v", err)
	}
	if got, want := crd.Spec.Scope, apiextensionsv1.NamespaceScoped; got != want {
		t.Fatalf("CRD scope = %q, want v2 default %q", got, want)
	}
}

func TestCRDFromXRDRejectsMultipleDocuments(t *testing.T) {
	_, err := crdFromXRD("../apis/access-v1beta1.yaml")
	if err == nil {
		t.Fatal("crdFromXRD() accepted a source with two XRD documents")
	}
	if !strings.Contains(err.Error(), "2 CompositeResourceDefinitions") {
		t.Fatalf("crdFromXRD() error = %v, want document count", err)
	}
}

func TestAdmissionEnvInstallsAllXRDsAndAdmitsCatalogExamples(t *testing.T) {
	paths := xrdPaths()
	if got, want := len(paths), 13; got != want {
		t.Fatalf("XRD source path count = %d, want %d", got, want)
	}

	derived := 0
	kinds := map[string]struct{}{}
	for _, path := range paths {
		crds, err := crdsFromXRD(path)
		if err != nil {
			t.Fatalf("crdsFromXRD(%q) error = %v", path, err)
		}
		for _, crd := range crds {
			derived++
			kinds[crd.Spec.Names.Kind] = struct{}{}
		}
	}
	if got, want := derived, 14; got != want {
		t.Fatalf("derived CRD count = %d, want %d", got, want)
	}
	if got, want := len(kinds), derived; got != want {
		t.Fatalf("distinct derived kind count = %d, want %d", got, want)
	}

	env := &admissionEnv{}
	if err := env.Start(t); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if got, want := env.installed, derived; got != want {
		t.Fatalf("installed CRD count = %d, want %d", got, want)
	}
	t.Logf("derived CRDs from 13 XRD sources: %d; API server installed %d/%d", derived, env.installed, derived)
	t.Cleanup(func() {
		if err := env.Stop(); err != nil {
			t.Errorf("Stop() error = %v", err)
		}
	})

	examples, exclusions := catalogExamples(t, kinds)
	if got, want := len(examples), 25; got != want {
		t.Fatalf("catalog examples admitted = %d, want %d", got, want)
	}
	if got, want := len(exclusions), 1; got != want {
		t.Fatalf("catalog exclusions = %d, want %d", got, want)
	}
	t.Logf("catalog examples admitted=%d excluded=%d", len(examples), len(exclusions))
	for _, exclusion := range exclusions {
		t.Logf("catalog exclusion source=%s kind=%s name=%s reason=%s", exclusion.source, exclusion.kind, exclusion.name, exclusion.reason)
	}

	representedKinds := map[string]struct{}{}
	for _, example := range examples {
		representedKinds[example.object.GetKind()] = struct{}{}
		example.object.SetNamespace("default")
		if err := env.Apply(context.Background(), &example.object); err != nil {
			t.Errorf("Apply(valid %s/%s from %s) error = %v", example.object.GetKind(), example.object.GetName(), example.source, err)
			continue
		}
		if got := example.object.GetResourceVersion(); got == "" {
			t.Errorf("Apply(valid %s/%s from %s) did not set resourceVersion", example.object.GetKind(), example.object.GetName(), example.source)
			continue
		}
		t.Logf("catalog admitted source=%s kind=%s name=%s", example.source, example.object.GetKind(), example.object.GetName())
	}
	if got, want := len(representedKinds), len(kinds); got != want {
		t.Fatalf("catalog represented kind count = %d, want %d", got, want)
	}
}

func TestAdmissionEnvRejectsCorruptCELAndAcceptsRestoredXRD(t *testing.T) {
	original, err := os.ReadFile("../apis/stack-v1beta1.yaml")
	if err != nil {
		t.Fatal(err)
	}
	corrupt := bytes.Replace(original, []byte("self.metadata.name == self.spec.slug"), []byte("self.metadata.name =="), 1)
	if bytes.Equal(corrupt, original) {
		t.Fatal("negative-control CEL rule was not found")
	}

	scratch := filepath.Join(t.TempDir(), "stack-v1beta1.yaml")
	if err := os.WriteFile(scratch, corrupt, 0o600); err != nil {
		t.Fatal(err)
	}

	corruptEnv := &admissionEnv{paths: []string{scratch}}
	err = corruptEnv.Start(t)
	if err == nil {
		t.Fatal("Start() accepted corrupt CEL")
	}
	if !strings.Contains(err.Error(), "stack-v1beta1.yaml") || !strings.Contains(err.Error(), "compilation failed") {
		t.Fatalf("corrupt CEL error = %v, want source name and API-server compilation error", err)
	}
	t.Logf("negative control rejected corrupt CEL: %v", err)
	if err := corruptEnv.Stop(); err != nil {
		t.Fatalf("Stop() after corrupt CEL error = %v", err)
	}

	if err := os.WriteFile(scratch, original, 0o600); err != nil {
		t.Fatal(err)
	}
	restoredEnv := &admissionEnv{paths: []string{scratch}}
	if err := restoredEnv.Start(t); err != nil {
		t.Fatalf("Start() after restoring CEL error = %v", err)
	}
	t.Log("restored CEL XRD was accepted")
	if err := restoredEnv.Stop(); err != nil {
		t.Fatalf("Stop() after restored CEL error = %v", err)
	}
}

func TestAdmissionEnvRejectsNonAtomicCollectionRefsAndAcceptsRestoredAllXRDs(t *testing.T) {
	const schemaPath = "spec.guards.ruleActions[].collectionRefs[]"
	original, err := os.ReadFile("../apis/agent-observability-v1beta1.yaml")
	if err != nil {
		t.Fatal(err)
	}
	broken := bytes.Replace(original, []byte("x-kubernetes-map-type: atomic\n                              required:"), []byte("required:"), 1)
	if bytes.Equal(broken, original) {
		t.Fatal("negative-control collectionRefs atomic repair was not found")
	}

	scratch := filepath.Join(t.TempDir(), "agent-observability-v1beta1.yaml")
	if err := os.WriteFile(scratch, broken, 0o600); err != nil {
		t.Fatal(err)
	}
	paths := append([]string(nil), xrdPaths()...)
	for i, path := range paths {
		if filepath.Base(path) == filepath.Base(scratch) {
			paths[i] = scratch
			break
		}
	}

	brokenEnv := &admissionEnv{paths: paths}
	err = brokenEnv.Start(t)
	if err == nil {
		t.Fatal("Start() accepted collectionRefs without atomic map type")
	}
	if !strings.Contains(err.Error(), "installed 13/14 derived CRDs") || !strings.Contains(err.Error(), "collectionRefs].items.x-kubernetes-map-type") {
		t.Fatalf("non-atomic collectionRefs error = %v, want 13/14 failure at %s", err, schemaPath)
	}
	t.Logf("negative control rejected missing atomic map type at %s: %v", schemaPath, err)
	if err := brokenEnv.Stop(); err != nil {
		t.Fatalf("Stop() after non-atomic collectionRefs error = %v", err)
	}

	if err := os.WriteFile(scratch, original, 0o600); err != nil {
		t.Fatal(err)
	}
	restoredEnv := &admissionEnv{paths: paths}
	if err := restoredEnv.Start(t); err != nil {
		t.Fatalf("Start() after restoring collectionRefs atomic map type error = %v", err)
	}
	if got, want := restoredEnv.installed, 14; got != want {
		t.Fatalf("restored collectionRefs installed CRD count = %d, want %d", got, want)
	}
	t.Log("restored collectionRefs atomic map type installed 14/14 derived CRDs")
	if err := restoredEnv.Stop(); err != nil {
		t.Fatalf("Stop() after restored collectionRefs error = %v", err)
	}
}

type catalogExample struct {
	source string
	object unstructured.Unstructured
}

type catalogExclusion struct {
	source string
	kind   string
	name   string
	reason string
}

func catalogExamples(t *testing.T, wanted map[string]struct{}) ([]catalogExample, []catalogExclusion) {
	t.Helper()
	paths, err := filepath.Glob("../../examples/catalog/*/*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	examples := []catalogExample{}
	exclusions := []catalogExclusion{}
	for _, path := range paths {
		if filepath.Base(path) == "kustomization.yaml" {
			continue
		}
		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		decoder := yaml.NewYAMLOrJSONDecoder(file, 4096)
		for {
			var object unstructured.Unstructured
			err := decoder.Decode(&object)
			if err == io.EOF {
				break
			}
			if err != nil {
				_ = file.Close()
				t.Fatalf("decode %s: %v", path, err)
			}
			if object.GetAPIVersion() != "platform.example.org/v1beta1" {
				continue
			}
			if _, found := wanted[object.GetKind()]; !found {
				exclusions = append(exclusions, catalogExclusion{
					source: path,
					kind:   object.GetKind(),
					name:   object.GetName(),
					reason: "no derived XRD; Composition input remains out of scope",
				})
				continue
			}
			examples = append(examples, catalogExample{source: path, object: object})
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return examples, exclusions
}
