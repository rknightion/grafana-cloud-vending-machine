package main

import (
	"go/ast"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// The ownership table is checked against the constructors reachable from each
// registered renderer. There is no second hand-maintained family-to-API map.
func TestProviderFamilyDocumentation(t *testing.T) {
	repo := coverageRepositoryRoot(t)
	sources := parseFunctionSources(t, filepath.Join(repo, "platform", "function"))
	constants := packageStringConstants(t, sources)
	functions := map[string]*ast.FuncDecl{}
	for _, source := range sources {
		for _, declaration := range source.file.Decls {
			if f, ok := declaration.(*ast.FuncDecl); ok && f.Recv == nil {
				functions[f.Name.Name] = f
			}
		}
	}
	roots := map[string]ast.Node{}
	ast.Inspect(sources["fn.go"].file, func(node ast.Node) bool {
		decl, ok := node.(*ast.ValueSpec)
		if !ok || len(decl.Names) != 1 || decl.Names[0].Name != "compositeRenderers" {
			return true
		}
		registry := decl.Values[0].(*ast.CompositeLit)
		for _, entry := range registry.Elts {
			pair := entry.(*ast.KeyValueExpr)
			name, err := strconv.Unquote(pair.Key.(*ast.BasicLit).Value)
			if err != nil {
				t.Fatal(err)
			}
			for _, field := range pair.Value.(*ast.CompositeLit).Elts {
				field := field.(*ast.KeyValueExpr)
				if field.Key.(*ast.Ident).Name == "render" {
					roots[name] = field.Value
				}
			}
		}
		return false
	})
	if len(roots) != len(compositeRenderers) {
		t.Fatal("provider-family documentation: could not resolve every registered renderer")
	}
	expected := map[string]map[string]bool{}
	for composite, root := range roots {
		seen := map[string]bool{}
		var visit func(ast.Node, *ast.FuncDecl)
		visit = func(node ast.Node, owner *ast.FuncDecl) {
			if identifier, ok := node.(*ast.Ident); ok {
				if seen[identifier.Name] {
					return
				}
				seen[identifier.Name] = true
				if f := functions[identifier.Name]; f != nil {
					visit(f.Body, f)
				}
				return
			}
			ast.Inspect(node, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				if isNewDesiredCall(call) {
					api, ae := resolveStringConstant(call.Args[0], constants)
					kind, ke := resolveStringConstant(call.Args[1], constants)
					kinds := []managedGVK{{APIVersion: api, Kind: kind}}
					if ae != nil || ke != nil {
						if owner == nil {
							t.Fatal("unresolved inline renderer constructor")
						}
						kinds = dynamicNewDesiredKinds(t, owner, "provider-family documentation", call, constants)
					}
					for _, gvk := range kinds {
						group := groupOfAPIVersion(gvk.APIVersion)
						if !strings.HasSuffix(group, ".crossplane.io") || !strings.Contains(group, "grafana.") {
							continue
						}
						family := strings.Split(group, ".")[0]
						if composite == "GrafanaStackInventory" {
							family = "observe-only inventory"
						}
						if expected[family] == nil {
							expected[family] = map[string]bool{}
						}
						expected[family][composite] = true
					}
					return false
				}
				if identifier, ok := call.Fun.(*ast.Ident); ok {
					visit(identifier, owner)
				}
				return true
			})
		}
		visit(root, nil)
	}
	path := filepath.Join(repo, "docs", "architecture.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	header := "| Provider family | Vending status | Composite API | Treatment in this reference |"
	lines := strings.Split(string(raw), "\n")
	active, found := false, false
	rows := map[string]bool{}
	composites := regexp.MustCompile("`(Grafana[^`]+)`")
	for _, line := range lines {
		if line == header {
			active, found = true, true
			continue
		}
		if !active {
			continue
		}
		if !strings.HasPrefix(line, "|") {
			break
		}
		if strings.HasPrefix(line, "| ---") {
			continue
		}
		cells := strings.Split(line, "|")
		if len(cells) != 6 {
			t.Fatalf("docs/architecture.md: malformed provider-family row: %s", line)
		}
		family, status := strings.TrimSpace(cells[1]), strings.TrimSpace(cells[2])
		if rows[family] {
			t.Fatalf("docs/architecture.md: duplicate provider family %s", family)
		}
		rows[family] = true
		actual := expected[family]
		if len(actual) == 0 {
			t.Fatalf("docs/architecture.md: provider family %s has no reachable provider constructors", family)
		}
		wantStatus := "vended"
		if family == "observe-only inventory" {
			wantStatus = "observe-only"
		}
		if status != wantStatus {
			t.Errorf("docs/architecture.md: provider family %s status %q, want %q from registered renderer constructors", family, status, wantStatus)
		}
		documented := map[string]bool{}
		for _, match := range composites.FindAllStringSubmatch(cells[3], -1) {
			documented[match[1]] = true
		}
		sorted := func(values map[string]bool) string {
			names := make([]string, 0, len(values))
			for name := range values {
				names = append(names, name)
			}
			sort.Strings(names)
			return strings.Join(names, ", ")
		}
		if sorted(actual) != sorted(documented) {
			t.Errorf("docs/architecture.md: provider family %s composite APIs [%s], want [%s] from reachable constructors", family, sorted(documented), sorted(actual))
		}
	}
	if !found {
		t.Fatal("docs/architecture.md: missing checked provider-family table")
	}
	for family := range expected {
		if !rows[family] {
			t.Errorf("docs/architecture.md: missing vended provider family %s", family)
		}
	}
}
