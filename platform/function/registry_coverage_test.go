package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestManagedKindActivationCoverage keeps the platform's emitted managed kinds
// aligned with the provider CRDs and the activation policy. It derives the
// emitted set from Go syntax. Literal and package-constant arguments are
// resolved by the restricted resolver below; every other form must be one of
// the explicit dynamic renderers audited in dynamicNewDesiredKinds.
func TestManagedKindActivationCoverage(t *testing.T) {
	repo := coverageRepositoryRoot(t)
	sources := parseFunctionSources(t, filepath.Join(repo, "platform", "function"))
	emitted := newDesiredKinds(t, sources)
	exclusions := loadNonManagedExclusions(t, repo)
	assertSyntheticMonitoringSecretExcluded(t, sources, exclusions)
	assertExclusionsMatchEmittedKinds(t, emitted, exclusions)

	mapFile := filepath.Join(repo, "platform", "provider", "managed-kind-map.json")
	mapping := loadManagedKindMap(t, mapFile)
	providerPackage, activated := providerActivationPolicy(t, filepath.Join(repo, "platform", "provider", "provider-grafana.yaml"))
	if mapping.ProviderPackageDigest != providerPackage {
		t.Fatalf("managed kind map providerPackageDigest = %q, want current provider pin %q", mapping.ProviderPackageDigest, providerPackage)
	}

	managed := map[string]managedGVK{}
	for key, value := range emitted {
		if exclusion, ok := exclusions[key]; ok {
			if strings.TrimSpace(exclusion.Reason) == "" {
				t.Fatalf("non-managed exclusion %s has no reason", key)
			}
			continue
		}
		managed[key] = value
	}

	entries := map[string]managedKindMapEntry{}
	for _, entry := range mapping.ManagedKinds {
		key := managedGVK{APIVersion: entry.APIVersion, Kind: entry.Kind}.key()
		if entry.APIVersion == "" || entry.Kind == "" || entry.Plural == "" {
			t.Fatalf("managed kind map has incomplete entry %#v", entry)
		}
		if _, exists := entries[key]; exists {
			t.Fatalf("managed kind map duplicates %s", key)
		}
		entries[key] = entry
	}
	assertSameGVKKeys(t, "emitted managed kinds", mapKeys(managed), "platform/provider/managed-kind-map.json", mapKeys(entries))

	for key, emittedGVK := range managed {
		entry := entries[key]
		if entry.APIVersion != emittedGVK.APIVersion || entry.Kind != emittedGVK.Kind {
			t.Fatalf("managed kind map entry %s does not match emitted GVK", key)
		}
		if !activated[entry.Plural+"."+groupOfAPIVersion(entry.APIVersion)] {
			t.Fatalf("platform/provider/provider-grafana.yaml: ManagedResourceActivationPolicy is missing %s for emitted %s", entry.Plural+"."+groupOfAPIVersion(entry.APIVersion), key)
		}
	}
}

type managedGVK struct {
	APIVersion string
	Kind       string
}

func (g managedGVK) key() string { return g.APIVersion + "/" + g.Kind }

type managedKindMap struct {
	ProviderPackageDigest string                `json:"providerPackageDigest"`
	ManagedKinds          []managedKindMapEntry `json:"managedKinds"`
	NonManagedExclusions  []nonManagedExclusion `json:"nonManagedExclusions"`
}

type managedKindMapEntry struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Plural     string `json:"plural"`
}

type nonManagedExclusion struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Reason     string `json:"reason"`
}

type sourceFile struct {
	path string
	file *ast.File
	fset *token.FileSet
}

func coverageRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate registry coverage test")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func parseFunctionSources(t *testing.T, directory string) map[string]sourceFile {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("read function source directory: %v", err)
	}
	result := map[string]sourceFile{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(directory, entry.Name())
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		result[entry.Name()] = sourceFile{path: path, file: parsed, fset: fset}
	}
	if len(result) == 0 {
		t.Fatal("no non-test function Go sources found")
	}
	return result
}

func newDesiredKinds(t *testing.T, sources map[string]sourceFile) map[string]managedGVK {
	t.Helper()
	constants := packageStringConstants(t, sources)
	result := map[string]managedGVK{}
	for name, source := range sources {
		assertNewDesiredIsNeverAliased(t, name, source)
		assertNoPackageConstructors(t, name, source)
		for _, declaration := range source.file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok || !isNewDesiredCall(call) {
					return true
				}
				position := source.fset.Position(call.Pos())
				if len(call.Args) < 2 {
					t.Fatalf("newDesired at %s has fewer than two arguments", coverageSite(name, position.Line))
				}
				apiVersion, apiErr := resolveStringConstant(call.Args[0], constants)
				kind, kindErr := resolveStringConstant(call.Args[1], constants)
				if apiErr == nil && kindErr == nil {
					addManagedGVK(t, result, managedGVK{APIVersion: apiVersion, Kind: kind}, coverageSite(name, position.Line))
					return true
				}
				for _, gvk := range dynamicNewDesiredKinds(t, function, coverageSite(name, position.Line), call, constants) {
					addManagedGVK(t, result, gvk, coverageSite(name, position.Line))
				}
				return true
			})
			assertDesiredComposedConstructionBoundary(t, name, source.fset, function)
		}
	}
	return result
}

func isNewDesiredCall(call *ast.CallExpr) bool {
	ident, ok := call.Fun.(*ast.Ident)
	return ok && ident.Name == "newDesired"
}

func coverageSite(name string, line int) string { return fmt.Sprintf("%s:%d", name, line) }

func assertNewDesiredIsNeverAliased(t *testing.T, filename string, source sourceFile) {
	t.Helper()
	walkWithParent(source.file, func(node, parent ast.Node) bool {
		ident, ok := node.(*ast.Ident)
		if !ok || ident.Name != "newDesired" {
			return true
		}
		if call, ok := parent.(*ast.CallExpr); ok && call.Fun == ident {
			return true
		}
		if function, ok := parent.(*ast.FuncDecl); ok && function.Name == ident {
			return true
		}
		position := source.fset.Position(ident.Pos())
		t.Fatalf("unsupported non-call newDesired reference at %s; aliases and indirect calls escape managed-kind coverage", coverageSite(filename, position.Line))
		return false
	})
}

func walkWithParent(root ast.Node, visit func(node, parent ast.Node) bool) {
	stack := []ast.Node{}
	ast.Inspect(root, func(node ast.Node) bool {
		if node == nil {
			stack = stack[:len(stack)-1]
			return false
		}
		var parent ast.Node
		if len(stack) > 0 {
			parent = stack[len(stack)-1]
		}
		if !visit(node, parent) {
			return false
		}
		stack = append(stack, node)
		return true
	})
}

func assertDesiredComposedConstructionBoundary(t *testing.T, filename string, fset *token.FileSet, function *ast.FuncDecl) {
	t.Helper()
	if function.Name.Name == "newDesired" || function.Name.Name == "syntheticMonitoringSecret" {
		return
	}
	assertNoDesiredComposedLiteral(t, filename, fset, function.Body, function.Name.Name)
}

func assertNoPackageConstructors(t *testing.T, filename string, source sourceFile) {
	t.Helper()
	for _, declaration := range source.file.Decls {
		if _, ok := declaration.(*ast.FuncDecl); ok {
			continue
		}
		assertNoDesiredComposedLiteral(t, filename, source.fset, declaration, "package scope")
		ast.Inspect(declaration, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if ok && isNewDesiredCall(call) {
				position := source.fset.Position(call.Pos())
				t.Fatalf("unsupported package-scope newDesired constructor at %s; managed resources must be constructed inside audited functions", coverageSite(filename, position.Line))
			}
			return true
		})
	}
}

func assertNoDesiredComposedLiteral(t *testing.T, filename string, fset *token.FileSet, node ast.Node, owner string) {
	t.Helper()
	ast.Inspect(node, func(node ast.Node) bool {
		literal, ok := node.(*ast.CompositeLit)
		if !ok || !isResourceDesiredComposed(literal) {
			return true
		}
		position := fset.Position(literal.Pos())
		t.Fatalf("unsupported direct resource.DesiredComposed construction at %s in %s; route it through newDesired or explicitly audit it", coverageSite(filename, position.Line), owner)
		return false
	})
}

func isResourceDesiredComposed(literal *ast.CompositeLit) bool {
	selector, ok := literal.Type.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "DesiredComposed" {
		return false
	}
	pkg, ok := selector.X.(*ast.Ident)
	return ok && pkg.Name == "resource"
}

func packageStringConstants(t *testing.T, sources map[string]sourceFile) map[string]string {
	t.Helper()
	expressions := map[string]ast.Expr{}
	for _, source := range sources {
		for _, declaration := range source.file.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.CONST {
				continue
			}
			for _, declaration := range general.Specs {
				valueSpec := declaration.(*ast.ValueSpec)
				for index, name := range valueSpec.Names {
					if index >= len(valueSpec.Values) {
						// An implicit or iota constant is irrelevant unless a constructor
						// uses it. In that case resolveStringConstant rejects it at the
						// constructor rather than failing on unrelated package constants.
						continue
					}
					expressions[name.Name] = valueSpec.Values[index]
				}
			}
		}
	}
	result := map[string]string{}
	for name, expression := range expressions {
		value, err := resolveStringExpression(expression, result, expressions, map[string]bool{name: true})
		if err != nil {
			continue
		}
		result[name] = value
	}
	return result
}

func resolveStringConstant(expression ast.Expr, constants map[string]string) (string, error) {
	return resolveStringExpression(expression, constants, nil, nil)
}

func resolveStringExpression(expression ast.Expr, constants map[string]string, unresolved map[string]ast.Expr, visiting map[string]bool) (string, error) {
	switch value := expression.(type) {
	case *ast.BasicLit:
		if value.Kind != token.STRING {
			return "", fmt.Errorf("not a string literal")
		}
		return strconv.Unquote(value.Value)
	case *ast.ParenExpr:
		return resolveStringExpression(value.X, constants, unresolved, visiting)
	case *ast.BinaryExpr:
		if value.Op != token.ADD {
			return "", fmt.Errorf("unsupported string operator %s", value.Op)
		}
		left, err := resolveStringExpression(value.X, constants, unresolved, visiting)
		if err != nil {
			return "", err
		}
		right, err := resolveStringExpression(value.Y, constants, unresolved, visiting)
		if err != nil {
			return "", err
		}
		return left + right, nil
	case *ast.Ident:
		if result, ok := constants[value.Name]; ok {
			return result, nil
		}
		if unresolved == nil {
			return "", fmt.Errorf("unresolved identifier %q", value.Name)
		}
		if visiting[value.Name] {
			return "", fmt.Errorf("constant cycle at %q", value.Name)
		}
		next, ok := unresolved[value.Name]
		if !ok {
			return "", fmt.Errorf("unresolved identifier %q", value.Name)
		}
		visiting[value.Name] = true
		result, err := resolveStringExpression(next, constants, unresolved, visiting)
		delete(visiting, value.Name)
		if err != nil {
			return "", err
		}
		constants[value.Name] = result
		return result, nil
	default:
		return "", fmt.Errorf("unsupported expression %T", expression)
	}
}

// dynamicNewDesiredKinds is deliberately closed. A new unresolved constructor
// argument cannot silently escape coverage: it must be added here with a
// source-backed resolver and a comment naming its complete input set.
func dynamicNewDesiredKinds(t *testing.T, function *ast.FuncDecl, site string, call *ast.CallExpr, constants map[string]string) []managedGVK {
	t.Helper()
	switch function.Name.Name {
	case "renderStackInventory":
		if selectorArgumentsAre(call, "set", "apiVersion", "kind") {
			result := make([]managedGVK, 0, len(stackInventoryProviderSets))
			for _, set := range stackInventoryProviderSets {
				result = append(result, managedGVK{APIVersion: set.apiVersion, Kind: set.kind})
			}
			return result
		}
		if selectorArgumentsAre(call, "inventoryOrganizationUser", "apiVersion", "itemKind") {
			return []managedGVK{{APIVersion: inventoryOrganizationUser.apiVersion, Kind: inventoryOrganizationUser.itemKind}}
		}
		t.Fatalf("uncovered dynamic newDesired constructor at %s in renderStackInventory", site)
	case "addObservabilityProducts":
		apiVersion, err := resolveStringConstant(call.Args[0], constants)
		if err != nil {
			t.Fatalf("dynamic constructor at %s has unresolved product API version: %v", site, err)
		}
		assertSelectorArgument(t, call.Args[1], "product", "kind", site)
		return inlineProductKinds(t, function.Body, apiVersion)
	case "renderContentAccessPolicy":
		apiVersion, err := resolveStringConstant(call.Args[0], constants)
		if err != nil {
			t.Fatalf("dynamic constructor at %s has unresolved content-access API version: %v", site, err)
		}
		assertIdentifierArgument(t, call.Args[1], "resourceKind", site)
		return contentAccessPolicyKinds(t, function.Body, apiVersion, site)
	default:
		t.Fatalf("uncovered dynamic newDesired constructor at %s in %s", site, function.Name.Name)
		return nil
	}
	return nil
}

func assertSelectorArgument(t *testing.T, expression ast.Expr, variable, field, site string) {
	t.Helper()
	selector, ok := expression.(*ast.SelectorExpr)
	if !ok {
		t.Fatalf("dynamic constructor at %s must pass %s.%s, got %T", site, variable, field, expression)
	}
	ident, identOK := selector.X.(*ast.Ident)
	if !identOK || ident.Name != variable || selector.Sel.Name != field {
		t.Fatalf("dynamic constructor at %s must pass %s.%s, got %T", site, variable, field, expression)
	}
}

func assertIdentifierArgument(t *testing.T, expression ast.Expr, want, site string) {
	t.Helper()
	ident, ok := expression.(*ast.Ident)
	if !ok || ident.Name != want {
		t.Fatalf("dynamic constructor at %s must pass %s, got %T", site, want, expression)
	}
}

func selectorArgumentsAre(call *ast.CallExpr, variable, apiField, kindField string) bool {
	if len(call.Args) < 2 {
		return false
	}
	for index, want := range []string{apiField, kindField} {
		selector, ok := call.Args[index].(*ast.SelectorExpr)
		if !ok {
			return false
		}
		ident, ok := selector.X.(*ast.Ident)
		if !ok || ident.Name != variable || selector.Sel.Name != want {
			return false
		}
	}
	return true
}

func inlineProductKinds(t *testing.T, body *ast.BlockStmt, apiVersion string) []managedGVK {
	t.Helper()
	var products *ast.CompositeLit
	ast.Inspect(body, func(node ast.Node) bool {
		rangeStatement, ok := node.(*ast.RangeStmt)
		if !ok {
			return true
		}
		ident, ok := rangeStatement.Value.(*ast.Ident)
		if !ok || ident.Name != "product" {
			return true
		}
		literal, ok := rangeStatement.X.(*ast.CompositeLit)
		if ok {
			products = literal
		}
		return false
	})
	if products == nil {
		t.Fatal("products renderer no longer ranges over an inline product table")
	}
	result := []managedGVK{}
	for _, element := range products.Elts {
		literal, ok := element.(*ast.CompositeLit)
		if !ok {
			t.Fatalf("product table element %T is not a struct literal", element)
		}
		kind := structStringField(t, literal, "kind")
		result = append(result, managedGVK{APIVersion: apiVersion, Kind: kind})
	}
	if len(result) == 0 {
		t.Fatal("product table is empty")
	}
	return result
}

func structStringField(t *testing.T, literal *ast.CompositeLit, field string) string {
	t.Helper()
	for _, element := range literal.Elts {
		pair, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		name, ok := pair.Key.(*ast.Ident)
		if !ok || name.Name != field {
			continue
		}
		value, err := resolveStringConstant(pair.Value, nil)
		if err != nil {
			t.Fatalf("product table %s: %v", field, err)
		}
		return value
	}
	t.Fatalf("product table element has no %q field", field)
	return ""
}

func contentAccessPolicyKinds(t *testing.T, body *ast.BlockStmt, apiVersion, site string) []managedGVK {
	t.Helper()
	allowed := map[string]bool{}
	resourceKindIsBounded := false
	ast.Inspect(body, func(node ast.Node) bool {
		assignment, ok := node.(*ast.AssignStmt)
		if ok && len(assignment.Lhs) == 1 && len(assignment.Rhs) == 1 {
			name, nameOK := assignment.Lhs[0].(*ast.Ident)
			if value, valueOK := assignment.Rhs[0].(*ast.BinaryExpr); nameOK && valueOK {
				input, inputOK := value.X.(*ast.Ident)
				suffix, suffixErr := resolveStringConstant(value.Y, nil)
				if inputOK && suffixErr == nil && name.Name == "resourceKind" && input.Name == "targetKind" && value.Op == token.ADD && suffix == "Permission" {
					resourceKindIsBounded = true
				}
			}
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		ident, ok := call.Fun.(*ast.Ident)
		if !ok || ident.Name != "oneOf" || len(call.Args) < 2 {
			return true
		}
		variable, ok := call.Args[0].(*ast.Ident)
		if !ok || variable.Name != "targetKind" {
			return true
		}
		for _, value := range call.Args[1:] {
			text, err := resolveStringConstant(value, nil)
			if err != nil {
				t.Fatalf("content access target kind: %v", err)
			}
			allowed[text] = true
		}
		return true
	})
	if len(allowed) == 0 {
		t.Fatal("content access renderer no longer declares targetKind alternatives")
	}
	if !resourceKindIsBounded {
		t.Fatalf("content access dynamic constructor at %s no longer derives resourceKind as targetKind plus Permission", site)
	}
	result := make([]managedGVK, 0, len(allowed))
	for kind := range allowed {
		result = append(result, managedGVK{APIVersion: apiVersion, Kind: kind + "Permission"})
	}
	return result
}

func addManagedGVK(t *testing.T, result map[string]managedGVK, gvk managedGVK, source string) {
	t.Helper()
	if gvk.APIVersion == "" || gvk.Kind == "" {
		t.Fatalf("%s emits incomplete GVK %#v", source, gvk)
	}
	result[gvk.key()] = gvk
}

func loadManagedKindMap(t *testing.T, path string) managedKindMap {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read managed kind map %s: %v", path, err)
	}
	var result managedKindMap
	if err := json.Unmarshal(contents, &result); err != nil {
		t.Fatalf("decode managed kind map: %v", err)
	}
	return result
}

func loadNonManagedExclusions(t *testing.T, repo string) map[string]nonManagedExclusion {
	t.Helper()
	mapping := loadManagedKindMap(t, filepath.Join(repo, "platform", "provider", "managed-kind-map.json"))
	result := map[string]nonManagedExclusion{}
	for _, exclusion := range mapping.NonManagedExclusions {
		key := managedGVK{APIVersion: exclusion.APIVersion, Kind: exclusion.Kind}.key()
		if exclusion.APIVersion == "" || exclusion.Kind == "" || exclusion.Reason == "" {
			t.Fatalf("incomplete non-managed exclusion %#v", exclusion)
		}
		if _, exists := result[key]; exists {
			t.Fatalf("duplicate non-managed exclusion %s", key)
		}
		result[key] = exclusion
	}
	return result
}

func assertSyntheticMonitoringSecretExcluded(t *testing.T, sources map[string]sourceFile, exclusions map[string]nonManagedExclusion) {
	t.Helper()
	secret := managedGVK{APIVersion: "v1", Kind: "Secret"}
	if exclusion, ok := exclusions[secret.key()]; !ok || strings.TrimSpace(exclusion.Reason) == "" {
		t.Fatalf("syntheticMonitoringSecret must be explicitly excluded from MRA coverage as %s", secret.key())
	}
	source, ok := sources["syntheticmonitoring.go"]
	if !ok || source.file == nil {
		t.Fatal("syntheticmonitoring.go not found; update the synthetic monitoring coverage assertion")
	}
	file := source.file
	var found bool
	ast.Inspect(file, func(node ast.Node) bool {
		function, ok := node.(*ast.FuncDecl)
		if !ok || function.Name.Name != "syntheticMonitoringSecret" {
			return true
		}
		found = compositeLiteralHasGVK(function.Body, secret)
		return false
	})
	if !found {
		t.Fatalf("syntheticMonitoringSecret no longer constructs the excluded core %s", secret.key())
	}
}

func assertExclusionsMatchEmittedKinds(t *testing.T, emitted map[string]managedGVK, exclusions map[string]nonManagedExclusion) {
	t.Helper()
	directSecret := managedGVK{APIVersion: "v1", Kind: "Secret"}.key()
	for key := range exclusions {
		if key == directSecret {
			continue
		}
		if _, ok := emitted[key]; !ok {
			t.Fatalf("non-managed exclusion %s does not match an emitted newDesired GVK", key)
		}
	}
}

func compositeLiteralHasGVK(node ast.Node, want managedGVK) bool {
	foundAPI, foundKind := false, false
	ast.Inspect(node, func(node ast.Node) bool {
		literal, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		for _, element := range literal.Elts {
			pair, ok := element.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := pair.Key.(*ast.BasicLit)
			if !ok || key.Kind != token.STRING {
				continue
			}
			name, err := strconv.Unquote(key.Value)
			if err != nil {
				continue
			}
			value, err := resolveStringConstant(pair.Value, nil)
			if err != nil {
				continue
			}
			if name == "apiVersion" && value == want.APIVersion {
				foundAPI = true
			}
			if name == "kind" && value == want.Kind {
				foundKind = true
			}
		}
		return true
	})
	return foundAPI && foundKind
}

func providerActivationPolicy(t *testing.T, path string) (string, map[string]bool) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open provider policy: %v", err)
	}
	defer file.Close()
	decoder := yaml.NewDecoder(file)
	var packageDigest string
	activated := map[string]bool{}
	for {
		var document struct {
			Kind string `yaml:"kind"`
			Spec struct {
				Package  string   `yaml:"package"`
				Activate []string `yaml:"activate"`
			} `yaml:"spec"`
		}
		err := decoder.Decode(&document)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("decode provider policy: %v", err)
		}
		switch document.Kind {
		case "Provider":
			at := strings.LastIndex(document.Spec.Package, "@")
			if at < 0 || document.Spec.Package[at+1:] == "" {
				t.Fatalf("provider package %q has no digest", document.Spec.Package)
			}
			packageDigest = document.Spec.Package[at+1:]
		case "ManagedResourceActivationPolicy":
			for _, plural := range document.Spec.Activate {
				if activated[plural] {
					t.Fatalf("ManagedResourceActivationPolicy duplicates %s", plural)
				}
				activated[plural] = true
			}
		}
	}
	if packageDigest == "" {
		t.Fatal("provider package digest not found")
	}
	if len(activated) == 0 {
		t.Fatal("ManagedResourceActivationPolicy activate list not found")
	}
	return packageDigest, activated
}

func groupOfAPIVersion(apiVersion string) string {
	group, _, ok := strings.Cut(apiVersion, "/")
	if !ok || group == "" {
		return ""
	}
	return group
}

func mapKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func assertSameGVKKeys(t *testing.T, leftName string, left []string, rightName string, right []string) {
	t.Helper()
	if strings.Join(left, "\n") != strings.Join(right, "\n") {
		t.Fatalf("%s and %s differ\n%s only: %s\n%s only: %s", leftName, rightName, leftName, strings.Join(difference(left, right), ", "), rightName, strings.Join(difference(right, left), ", "))
	}
}

func difference(left, right []string) []string {
	contained := map[string]bool{}
	for _, value := range right {
		contained[value] = true
	}
	result := []string{}
	for _, value := range left {
		if !contained[value] {
			result = append(result, value)
		}
	}
	return result
}
