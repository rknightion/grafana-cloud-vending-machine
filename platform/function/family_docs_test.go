package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestProviderFamilyDocumentationRejectsHiddenRendererDispatchByPath(t *testing.T) {
	repo := coverageRepositoryRoot(t)
	sources := rendererFixtureSources(t, repo, `package main

type compositeRenderer struct { render func() }
type hidden struct { render func() }
var compositeRenderers = map[string]compositeRenderer{"GrafanaHidden": {render: hiddenRenderer}}
func indirect(render func()) func() { return render }
func hiddenRenderer() { value := hidden{render: indirect(renderStack)}; value.render() }
func renderStack() { newDesired("stack.grafana.crossplane.io/v1beta1", "Stack") }
`)
	roots := rendererRoots(t, sources["fn.go"].file)
	_, err := reachableProviderFamilies(t, repo, sources, roots)
	if err == nil {
		t.Fatal("hidden-dispatch negative control unexpectedly passed")
	}
	const want = "unresolved renderer dispatch at platform/function/fn.go:"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("hidden-dispatch negative control did not fail by source path: %v", err)
	}
	t.Logf("hidden-dispatch negative control output: %v", err)
}

func TestProviderFamilyDocumentationFollowsMethodsAndFunctionValues(t *testing.T) {
	repo := coverageRepositoryRoot(t)
	sources := rendererFixtureSources(t, repo, `package main

type compositeRenderer struct { render func() }
type dispatch struct { render func() }
type methodDispatch struct{}
var compositeRenderers = map[string]compositeRenderer{
	"GrafanaMethod": {render: methodRoot},
	"GrafanaFunctionValue": {render: functionValueRoot},
	"GrafanaStructField": {render: structFieldRoot},
}
func (methodDispatch) render() { newDesired("stack.grafana.crossplane.io/v1beta1", "Stack") }
func methodRoot() { methodDispatch{}.render() }
func functionValueTarget() { newDesired("stack.grafana.crossplane.io/v1beta1", "Stack") }
func functionValueRoot() { render := functionValueTarget; render() }
func structFieldRoot() { value := dispatch{render: functionValueTarget}; value.render() }
`)
	expected, err := reachableProviderFamilies(t, repo, sources, rendererRoots(t, sources["fn.go"].file))
	if err != nil {
		t.Fatal(err)
	}
	got := expected["stack"]
	for _, composite := range []string{"GrafanaMethod", "GrafanaFunctionValue", "GrafanaStructField"} {
		if !got[composite] {
			t.Fatalf("method/function-value traversal omitted %s: %#v", composite, got)
		}
	}
}

func rendererFixtureSources(t *testing.T, repo, contents string) map[string]sourceFile {
	t.Helper()
	path := filepath.Join(repo, "platform", "function", "fn.go")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, contents, 0)
	if err != nil {
		t.Fatal(err)
	}
	return map[string]sourceFile{"fn.go": {path: path, file: file, fset: fset}}
}

// The ownership table is checked against the constructors reachable from each
// registered renderer. There is no second hand-maintained family-to-API map.
func TestProviderFamilyDocumentation(t *testing.T) {
	repo := coverageRepositoryRoot(t)
	sources := parseFunctionSources(t, filepath.Join(repo, "platform", "function"))
	roots := rendererRoots(t, sources["fn.go"].file)
	if len(roots) != len(compositeRenderers) {
		t.Fatal("provider-family documentation: could not resolve every registered renderer")
	}
	expected, err := reachableProviderFamilies(t, repo, sources, roots)
	if err != nil {
		t.Fatal(err)
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

type rendererFunction struct {
	declaration *ast.FuncDecl
	source      sourceFile
}

type rendererBinding struct {
	values        []ast.Expr
	typeName      string
	external      bool
	funcParameter bool
}

type rendererReachability struct {
	constants map[string]string
	functions map[string]rendererFunction
	methods   map[string]rendererFunction
	fields    map[string]map[string]ast.Expr
	globals   map[string]ast.Expr
	imports   map[string]bool
	repo      string
}

func rendererRoots(t *testing.T, source *ast.File) map[string]ast.Expr {
	t.Helper()
	roots := map[string]ast.Expr{}
	ast.Inspect(source, func(node ast.Node) bool {
		decl, ok := node.(*ast.ValueSpec)
		if !ok || len(decl.Names) != 1 || decl.Names[0].Name != "compositeRenderers" {
			return true
		}
		if len(decl.Values) != 1 {
			t.Fatal("provider-family documentation: platform/function/fn.go: compositeRenderers has no single initializer")
		}
		registry, ok := decl.Values[0].(*ast.CompositeLit)
		if !ok {
			t.Fatal("provider-family documentation: compositeRenderers is not a composite literal")
		}
		for _, entry := range registry.Elts {
			pair, ok := entry.(*ast.KeyValueExpr)
			if !ok {
				t.Fatal("provider-family documentation: malformed compositeRenderers entry")
			}
			literal, ok := pair.Key.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				t.Fatal("provider-family documentation: platform/function/fn.go: compositeRenderers key is not a string literal")
			}
			name, err := strconv.Unquote(literal.Value)
			if err != nil {
				t.Fatal(err)
			}
			value, ok := pair.Value.(*ast.CompositeLit)
			if !ok {
				t.Fatalf("provider-family documentation: renderer %s is not a composite literal", name)
			}
			for _, entry := range value.Elts {
				field, ok := entry.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := field.Key.(*ast.Ident)
				if ok && key.Name == "render" {
					roots[name] = field.Value
				}
			}
		}
		return false
	})
	return roots
}

func reachableProviderFamilies(t *testing.T, repo string, sources map[string]sourceFile, roots map[string]ast.Expr) (map[string]map[string]bool, error) {
	t.Helper()
	walker := newRendererReachability(t, sources, repo)
	expected := map[string]map[string]bool{}
	for composite, root := range roots {
		seen := map[*ast.FuncDecl]bool{}
		if err := walker.visitValue(t, root, nil, nil, seen, func(kinds []managedGVK) {
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
		}); err != nil {
			return nil, err
		}
	}
	return expected, nil
}

func newRendererReachability(t *testing.T, sources map[string]sourceFile, repo string) rendererReachability {
	walker := rendererReachability{
		constants: packageStringConstants(t, sources),
		functions: map[string]rendererFunction{},
		methods:   map[string]rendererFunction{},
		fields:    map[string]map[string]ast.Expr{},
		globals:   map[string]ast.Expr{},
		imports:   map[string]bool{},
		repo:      repo,
	}
	for _, source := range sources {
		for _, imported := range source.file.Imports {
			if imported.Name != nil {
				walker.imports[imported.Name.Name] = true
				continue
			}
			path, err := strconv.Unquote(imported.Path.Value)
			if err == nil {
				walker.imports[filepath.Base(path)] = true
			}
		}
		for _, declaration := range source.file.Decls {
			switch declaration := declaration.(type) {
			case *ast.FuncDecl:
				function := rendererFunction{declaration: declaration, source: source}
				if declaration.Recv == nil {
					walker.functions[declaration.Name.Name] = function
					continue
				}
				if typeName := receiverTypeName(declaration.Recv); typeName != "" {
					walker.methods[typeName+"."+declaration.Name.Name] = function
				}
			case *ast.GenDecl:
				if declaration.Tok == token.VAR {
					for _, spec := range declaration.Specs {
						valueSpec, ok := spec.(*ast.ValueSpec)
						if !ok {
							continue
						}
						for index, name := range valueSpec.Names {
							if index < len(valueSpec.Values) {
								walker.globals[name.Name] = valueSpec.Values[index]
							}
						}
					}
					continue
				}
				if declaration.Tok != token.TYPE {
					continue
				}
				for _, spec := range declaration.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					structure, ok := typeSpec.Type.(*ast.StructType)
					if !ok {
						continue
					}
					fields := map[string]ast.Expr{}
					for _, field := range structure.Fields.List {
						for _, name := range field.Names {
							fields[name.Name] = nil
						}
					}
					walker.fields[typeSpec.Name.Name] = fields
				}
			}
		}
	}
	return walker
}

// Renderer reachability is closed over package-local dispatch. Calls to imported
// packages are terminal, but every local function, method, function value, and
// struct field that can select a renderer must resolve to source or fail by path.
func (w rendererReachability) visitValue(t *testing.T, value ast.Expr, owner *rendererFunction, bindings map[string]rendererBinding, seen map[*ast.FuncDecl]bool, addKinds func([]managedGVK)) error {
	switch value := value.(type) {
	case *ast.Ident:
		if bindings != nil {
			if binding, ok := bindings[value.Name]; ok {
				if binding.funcParameter {
					return w.unresolved(owner, value, "func-typed parameter %q", value.Name)
				}
				if len(binding.values) > 0 {
					return w.visitValues(t, binding.values, owner, bindings, seen, addKinds)
				}
			}
		}
		if value, ok := w.globals[value.Name]; ok {
			return w.visitValue(t, value, owner, bindings, seen, addKinds)
		}
		function, ok := w.functions[value.Name]
		if !ok {
			return w.unresolved(owner, value, "unresolved function value %q", value.Name)
		}
		return w.visitFunction(t, function, seen, addKinds)
	case *ast.FuncLit:
		return w.visitBody(t, value.Body, owner, bindings, seen, addKinds)
	case *ast.SelectorExpr:
		return w.visitSelector(t, value, owner, bindings, seen, addKinds)
	case *ast.CallExpr:
		return w.unresolved(owner, value, "call result used as a renderer function")
	default:
		return w.unresolved(owner, value, "unsupported renderer function value %T", value)
	}
}

func (w rendererReachability) visitValues(t *testing.T, values []ast.Expr, owner *rendererFunction, bindings map[string]rendererBinding, seen map[*ast.FuncDecl]bool, addKinds func([]managedGVK)) error {
	for _, value := range values {
		if err := w.visitValue(t, value, owner, bindings, seen, addKinds); err != nil {
			return err
		}
	}
	return nil
}

func (w rendererReachability) visitFunction(t *testing.T, function rendererFunction, seen map[*ast.FuncDecl]bool, addKinds func([]managedGVK)) error {
	if seen[function.declaration] {
		return nil
	}
	seen[function.declaration] = true
	return w.visitBody(t, function.declaration.Body, &function, rendererBindings(function.declaration), seen, addKinds)
}

func (w rendererReachability) visitBody(t *testing.T, body *ast.BlockStmt, owner *rendererFunction, bindings map[string]rendererBinding, seen map[*ast.FuncDecl]bool, addKinds func([]managedGVK)) error {
	if body == nil {
		return nil
	}
	bindings = mergeRendererBindings(bindings, rendererBindingsFromBody(body))
	externalTypeSwitchCalls := externalTypeSwitchCalls(body)
	var visitErr error
	ast.Inspect(body, func(node ast.Node) bool {
		if visitErr != nil {
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if externalTypeSwitchCalls[call] {
			return true
		}
		if isNewDesiredCall(call) {
			if len(call.Args) < 2 {
				visitErr = w.unresolved(owner, call, "newDesired has fewer than two arguments")
				return false
			}
			api, apiErr := resolveStringConstant(call.Args[0], w.constants)
			kind, kindErr := resolveStringConstant(call.Args[1], w.constants)
			kinds := []managedGVK{{APIVersion: api, Kind: kind}}
			if apiErr != nil || kindErr != nil {
				if owner == nil {
					visitErr = w.unresolved(owner, call, "unresolved inline renderer constructor")
					return false
				}
				// dynamicNewDesiredKinds is keyed to the reached declaration, so
				// method and function-value dispatch shares this resolver boundary.
				kinds = dynamicNewDesiredKinds(t, owner.declaration, "provider-family documentation", call, w.constants)
			}
			addKinds(kinds)
			return false
		}
		visitErr = w.visitCall(t, call, owner, bindings, seen, addKinds)
		return visitErr == nil
	})
	return visitErr
}

// A type switch narrows its variable separately in each case. Only calls in a
// case whose asserted types are all imported or non-callable primitives are
// terminal; a local or default branch remains fail-closed.
func externalTypeSwitchCalls(body *ast.BlockStmt) map[*ast.CallExpr]bool {
	result := map[*ast.CallExpr]bool{}
	ast.Inspect(body, func(node ast.Node) bool {
		switchStatement, ok := node.(*ast.TypeSwitchStmt)
		if !ok {
			return true
		}
		assignment, ok := switchStatement.Assign.(*ast.AssignStmt)
		if !ok || len(assignment.Lhs) != 1 {
			return true
		}
		name, ok := assignment.Lhs[0].(*ast.Ident)
		if !ok {
			return true
		}
		for _, statement := range switchStatement.Body.List {
			clause, ok := statement.(*ast.CaseClause)
			if !ok || len(clause.List) == 0 {
				continue
			}
			safe := true
			for _, expression := range clause.List {
				safe = safe && externalOrNonCallableType(expression)
			}
			if !safe {
				continue
			}
			ast.Inspect(clause, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				receiver, ok := selector.X.(*ast.Ident)
				if ok && name.Obj != nil && receiver.Obj == name.Obj {
					result[call] = true
				}
				return true
			})
		}
		return true
	})
	return result
}

func (w rendererReachability) visitCall(t *testing.T, call *ast.CallExpr, owner *rendererFunction, bindings map[string]rendererBinding, seen map[*ast.FuncDecl]bool, addKinds func([]managedGVK)) error {
	switch function := call.Fun.(type) {
	case *ast.Ident:
		if binding, ok := bindings[function.Name]; ok {
			if binding.funcParameter {
				return w.unresolved(owner, call, "call through func-typed parameter %q", function.Name)
			}
			if len(binding.values) > 0 {
				return w.visitValues(t, binding.values, owner, bindings, seen, addKinds)
			}
			return w.unresolved(owner, call, "unresolved local callable %q", function.Name)
		}
		if declared, ok := w.functions[function.Name]; ok {
			return w.visitFunction(t, declared, seen, addKinds)
		}
		if value, ok := w.globals[function.Name]; ok {
			return w.visitValue(t, value, owner, bindings, seen, addKinds)
		}
		if rendererBuiltin(function.Name) {
			return nil
		}
		return w.unresolved(owner, call, "unresolved package-local call %q", function.Name)
	case *ast.FuncLit:
		return w.visitBody(t, function.Body, owner, bindings, seen, addKinds)
	case *ast.SelectorExpr:
		return w.visitSelector(t, function, owner, bindings, seen, addKinds)
	default:
		if isRendererTypeExpression(function) {
			return nil
		}
		return w.unresolved(owner, call, "unsupported call expression %T", call.Fun)
	}
}

func isRendererTypeExpression(expression ast.Expr) bool {
	switch expression.(type) {
	case *ast.ArrayType, *ast.ChanType, *ast.FuncType, *ast.InterfaceType, *ast.MapType, *ast.StructType:
		return true
	default:
		return false
	}
}

func rendererBuiltin(name string) bool {
	switch name {
	case "append", "cap", "clear", "close", "complex", "copy", "delete", "imag", "len", "make", "max", "min", "new", "panic", "print", "println", "real", "recover", "string", "bool", "byte", "rune", "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "float32", "float64", "complex64", "complex128", "any":
		return true
	default:
		return false
	}
}

func (w rendererReachability) visitSelector(t *testing.T, selector *ast.SelectorExpr, owner *rendererFunction, bindings map[string]rendererBinding, seen map[*ast.FuncDecl]bool, addKinds func([]managedGVK)) error {
	if w.externalReceiver(selector.X, bindings) {
		return nil
	}
	typeName, literal, known, err := w.rendererReceiver(selector.X, bindings)
	if err != nil {
		return w.unresolved(owner, selector, "%v", err)
	}
	if !known {
		if w.externalReceiver(selector.X, bindings) {
			return nil
		}
		return w.unresolved(owner, selector, "unresolved local receiver %T", selector.X)
	}
	if function, ok := w.methods[typeName+"."+selector.Sel.Name]; ok {
		return w.visitFunction(t, function, seen, addKinds)
	}
	fields, ok := w.fields[typeName]
	if !ok {
		if w.externalReceiver(selector.X, bindings) {
			return nil
		}
		return w.unresolved(owner, selector, "unresolved method or field %s.%s", typeName, selector.Sel.Name)
	}
	if _, ok := fields[selector.Sel.Name]; !ok {
		return w.unresolved(owner, selector, "unresolved struct field %s.%s", typeName, selector.Sel.Name)
	}
	if literal != nil {
		for _, entry := range literal.Elts {
			pair, ok := entry.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := pair.Key.(*ast.Ident)
			if ok && key.Name == selector.Sel.Name {
				return w.visitValue(t, pair.Value, owner, bindings, seen, addKinds)
			}
		}
	}
	return w.unresolved(owner, selector, "func-typed struct field %s.%s has no static value", typeName, selector.Sel.Name)
}

func (w rendererReachability) externalReceiver(expression ast.Expr, bindings map[string]rendererBinding) bool {
	switch expression := expression.(type) {
	case *ast.ParenExpr:
		return w.externalReceiver(expression.X, bindings)
	case *ast.BinaryExpr:
		return w.externalReceiver(expression.X, bindings) || w.externalReceiver(expression.Y, bindings)
	case *ast.UnaryExpr:
		return w.externalReceiver(expression.X, bindings)
	case *ast.Ident:
		if binding, ok := bindings[expression.Name]; ok {
			if binding.external {
				return true
			}
			for _, value := range binding.values {
				if w.externalReceiver(value, bindings) {
					return true
				}
			}
			return false
		}
		return w.imports[expression.Name]
	case *ast.SelectorExpr:
		return w.externalReceiver(expression.X, bindings)
	case *ast.CallExpr:
		if identifier, ok := expression.Fun.(*ast.Ident); ok {
			if function, ok := w.functions[identifier.Name]; ok && function.declaration.Type.Results != nil {
				// A binding to a multi-result call represents its first result.
				// An imported sibling result cannot certify this receiver.
				results := function.declaration.Type.Results.List
				return len(results) > 0 && externalType(results[0].Type)
			}
		}
		if selector, ok := expression.Fun.(*ast.SelectorExpr); ok {
			return w.externalReceiver(selector.X, bindings)
		}
	case *ast.IndexExpr:
		return w.externalReceiver(expression.X, bindings)
	case *ast.TypeAssertExpr:
		return externalType(expression.Type) || w.externalReceiver(expression.X, bindings)
	}
	return false
}

func (w rendererReachability) rendererReceiver(expression ast.Expr, bindings map[string]rendererBinding) (string, *ast.CompositeLit, bool, error) {
	switch expression := expression.(type) {
	case *ast.ParenExpr:
		return w.rendererReceiver(expression.X, bindings)
	case *ast.UnaryExpr:
		if expression.Op == token.AND || expression.Op == token.MUL {
			return w.rendererReceiver(expression.X, bindings)
		}
		return "", nil, true, fmt.Errorf("unsupported renderer receiver %T", expression)
	case *ast.CompositeLit:
		return expressionTypeName(expression.Type), expression, true, nil
	case *ast.Ident:
		binding, ok := bindings[expression.Name]
		if !ok {
			if w.imports[expression.Name] {
				return "", nil, false, nil
			}
			return "", nil, false, nil
		}
		if len(binding.values) == 1 {
			if literal, ok := binding.values[0].(*ast.CompositeLit); ok {
				return expressionTypeName(literal.Type), literal, true, nil
			}
		}
		if binding.typeName != "" {
			return binding.typeName, nil, true, nil
		}
		if binding.external {
			return "", nil, false, nil
		}
		if len(binding.values) == 1 {
			value := binding.values[0]
			if identifier, ok := value.(*ast.Ident); !ok || identifier.Name != expression.Name {
				return w.rendererReceiver(value, bindings)
			}
			return "", nil, true, fmt.Errorf("unresolved renderer receiver %T", value)
		}
		return "", nil, true, fmt.Errorf("ambiguous renderer receiver %q", expression.Name)
	case *ast.CallExpr:
		if identifier, ok := expression.Fun.(*ast.Ident); ok {
			if function, ok := w.functions[identifier.Name]; ok && function.declaration.Type.Results != nil && len(function.declaration.Type.Results.List) > 0 {
				if externalType(function.declaration.Type.Results.List[0].Type) {
					return "", nil, false, nil
				}
				if typeName := expressionTypeName(function.declaration.Type.Results.List[0].Type); typeName != "" {
					return typeName, nil, true, nil
				}
			}
		}
		if selector, ok := expression.Fun.(*ast.SelectorExpr); ok {
			if w.externalReceiver(selector.X, bindings) {
				return "", nil, false, nil
			}
			typeName, _, known, err := w.rendererReceiver(selector.X, bindings)
			if err != nil {
				return "", nil, true, err
			}
			if !known {
				return "", nil, false, nil
			}
			method, ok := w.methods[typeName+"."+selector.Sel.Name]
			if !ok || method.declaration.Type.Results == nil || len(method.declaration.Type.Results.List) == 0 {
				return "", nil, true, fmt.Errorf("unresolved local call result %s.%s", typeName, selector.Sel.Name)
			}
			result := method.declaration.Type.Results.List[0].Type
			if externalType(result) {
				return "", nil, false, nil
			}
			if resultType := expressionTypeName(result); resultType != "" {
				return resultType, nil, true, nil
			}
			return "", nil, true, fmt.Errorf("unsupported local call result %T", result)
		}
		return "", nil, false, nil
	case *ast.SelectorExpr:
		if w.externalReceiver(expression.X, bindings) {
			return "", nil, false, nil
		}
		return "", nil, true, fmt.Errorf("unresolved nested local receiver %T", expression)
	case *ast.BinaryExpr:
		if w.externalReceiver(expression.X, bindings) || w.externalReceiver(expression.Y, bindings) {
			return "", nil, false, nil
		}
		return "", nil, true, fmt.Errorf("unsupported renderer receiver %T", expression)
	case *ast.TypeAssertExpr:
		if externalType(expression.Type) || w.externalReceiver(expression.X, bindings) {
			return "", nil, false, nil
		}
		return "", nil, true, fmt.Errorf("unsupported renderer receiver %T", expression)
	case *ast.IndexExpr, *ast.SliceExpr:
		return "", nil, true, fmt.Errorf("unsupported renderer receiver %T", expression)
	default:
		return "", nil, true, fmt.Errorf("unsupported renderer receiver %T", expression)
	}
}

func (w rendererReachability) unresolved(owner *rendererFunction, node ast.Node, format string, arguments ...any) error {
	if owner == nil {
		return fmt.Errorf("provider-family documentation: unresolved renderer dispatch at inline renderer: "+format, arguments...)
	}
	position := owner.source.fset.Position(node.Pos())
	path, err := filepath.Rel(w.repo, position.Filename)
	if err != nil {
		path = position.Filename
	}
	return fmt.Errorf("provider-family documentation: unresolved renderer dispatch at %s:%d: "+format, append([]any{filepath.ToSlash(path), position.Line}, arguments...)...)
}

func receiverTypeName(fields *ast.FieldList) string {
	if fields == nil || len(fields.List) != 1 {
		return ""
	}
	return expressionTypeName(fields.List[0].Type)
}

func expressionTypeName(expression ast.Expr) string {
	switch expression := expression.(type) {
	case *ast.Ident:
		return expression.Name
	case *ast.StarExpr:
		return expressionTypeName(expression.X)
	case *ast.ParenExpr:
		return expressionTypeName(expression.X)
	default:
		return ""
	}
}

func rendererBindings(function *ast.FuncDecl) map[string]rendererBinding {
	bindings := map[string]rendererBinding{}
	if function.Type.Params == nil {
		return bindings
	}
	for _, field := range function.Type.Params.List {
		for _, name := range field.Names {
			binding := rendererBinding{typeName: expressionTypeName(field.Type), external: externalType(field.Type)}
			_, binding.funcParameter = field.Type.(*ast.FuncType)
			bindings[name.Name] = binding
		}
	}
	return bindings
}

func rendererBindingsFromBody(body *ast.BlockStmt) map[string]rendererBinding {
	bindings := map[string]rendererBinding{}
	ast.Inspect(body, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.AssignStmt:
			for index, left := range node.Lhs {
				name, ok := left.(*ast.Ident)
				if !ok || index >= len(node.Rhs) {
					continue
				}
				addRendererBinding(bindings, name.Name, rendererBinding{values: []ast.Expr{node.Rhs[index]}, typeName: expressionTypeName(node.Rhs[index])})
			}
		case *ast.DeclStmt:
			declaration, ok := node.Decl.(*ast.GenDecl)
			if !ok || declaration.Tok != token.VAR {
				return true
			}
			for _, spec := range declaration.Specs {
				valueSpec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for index, name := range valueSpec.Names {
					binding := rendererBinding{typeName: expressionTypeName(valueSpec.Type), external: externalType(valueSpec.Type)}
					if index < len(valueSpec.Values) {
						binding.values = []ast.Expr{valueSpec.Values[index]}
						if binding.typeName == "" {
							binding.typeName = expressionTypeName(valueSpec.Values[index])
						}
					}
					addRendererBinding(bindings, name.Name, binding)
				}
			}

		}
		return true
	})

	return bindings
}

func addRendererBinding(bindings map[string]rendererBinding, name string, binding rendererBinding) {
	if existing, ok := bindings[name]; ok {
		binding.values = append(existing.values, binding.values...)
		binding.external = binding.external || existing.external
		if binding.typeName == "" {
			binding.typeName = existing.typeName
		}
	}
	bindings[name] = binding
}

func mergeRendererBindings(base, additions map[string]rendererBinding) map[string]rendererBinding {
	result := map[string]rendererBinding{}
	for name, binding := range base {
		result[name] = binding
	}
	for name, binding := range additions {
		if existing, ok := result[name]; ok && existing.typeName != "" && binding.typeName == "" {
			binding.typeName = existing.typeName
		}
		if existing, ok := result[name]; ok && len(existing.values) > 0 && len(binding.values) > 0 {
			binding.values = append(existing.values, binding.values...)
		}
		if existing, ok := result[name]; ok {
			binding.external = binding.external || existing.external
		}
		result[name] = binding
	}
	return result
}

func externalType(expression ast.Expr) bool {
	switch expression := expression.(type) {
	case *ast.SelectorExpr:
		return true
	case *ast.MapType:
		return externalType(expression.Key) || externalType(expression.Value)
	case *ast.ArrayType:
		return externalType(expression.Elt)
	case *ast.StarExpr:
		return externalType(expression.X)
	case *ast.ChanType:
		return externalType(expression.Value)
	default:
		return false
	}
}

func externalOrNonCallableType(expression ast.Expr) bool {
	if externalType(expression) {
		return true
	}
	identifier, ok := expression.(*ast.Ident)
	if !ok {
		return false
	}
	switch identifier.Name {
	case "bool", "byte", "rune", "string", "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "float32", "float64", "complex64", "complex128":
		return true
	default:
		return false
	}
}
