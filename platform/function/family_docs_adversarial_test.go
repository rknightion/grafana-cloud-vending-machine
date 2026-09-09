package main

import (
	"strings"
	"testing"
)

// Unsupported local dispatch must be rejected, never mistaken for an imported
// terminal call. These shapes may either resolve the constructor or fail by path.
func TestProviderFamilyDocumentationLocalDispatchCannotDisappear(t *testing.T) {
	cases := map[string]string{
		"imported switch call arguments": `func root(x any){switch v:=x.(type){case *bytes.Buffer: v.Write(payload())}}; func payload() []byte {leaf();return nil}`,
		"shadowed switch variable":       `type receiver struct{}; func(receiver) Run(){leaf()}; func root(x any){switch v:=x.(type){case *bytes.Buffer: {v:=receiver{}; v.Run()}; v.String()}}`,
		"mixed result positions":         `type receiver struct{}; func(receiver) Run(){leaf()}; func makeReceiver()(interface{Run()},time.Time){return receiver{},time.Time{}}; func root(){r,_:=makeReceiver();r.Run()}`,
		"shadowed import":                `type receiver struct{}; func(receiver) Run(){leaf()}; func makeReceiver() interface{Run()}{return receiver{}}; func root(){bytes:=makeReceiver();bytes.Run()}`,
		"type switch default":            `type receiver struct{}; func(receiver) String() string {leaf();return ""}; func root(x interface{String() string}) {switch v:=x.(type){case *bytes.Buffer: v.String(); default: v.String()}}`,
		"local factory return":           `type receiver struct{}; type factory struct{}; func(factory) Make() receiver{return receiver{}}; func(receiver) Run(){leaf()}; func root(){r:=factory{}.Make(); r.Run()}`,
		"mixed type switch":              `type receiver struct{}; func(receiver) Run(){leaf()}; func root(){var value any; switch v := value.(type) {case receiver: v.Run(); case time.Time: v.UTC()}}`,
		"global function value":          `var alias = leaf; func root(){ alias() }`,
		"pointer method":                 `type receiver struct{}; func (*receiver) Run(){ leaf() }; func root(){ r := &receiver{}; r.Run() }`,
		"nested receiver":                `type receiver struct{}; type outer struct{ inner receiver }; func (receiver) Run(){ leaf() }; func root(){ r := outer{}; r.inner.Run() }`,
		"indexed receiver":               `type receiver struct{}; func (receiver) Run(){ leaf() }; func root(){ r := []receiver{{}}; r[0].Run() }`,
		"reassigned function value":      `func noop(){}; func root(){ f := leaf; f(); f = noop; f() }`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			repo := coverageRepositoryRoot(t)
			sources := rendererFixtureSources(t, repo, `package main
  import ("time"; "bytes")
  type compositeRenderer struct{ render func() }
  var compositeRenderers = map[string]compositeRenderer{"GrafanaHidden":{render:root}}
  func leaf(){ newDesired("cloud.grafana.m.crossplane.io/v1alpha1", "Stack") }
  `+body)
			roots := rendererRoots(t, sources["fn.go"].file)
			families, err := reachableProviderFamilies(t, repo, sources, roots)
			if err != nil {
				if !strings.Contains(err.Error(), "platform/function/") {
					t.Fatalf("unresolved local dispatch lacks source path: %v", err)
				}
				t.Logf("unsupported local dispatch refused: %v", err)
				return
			}
			if !families["cloud"]["GrafanaHidden"] {
				t.Fatal("local renderer silently disappeared from cloud family ownership")
			}
		})
	}
}
