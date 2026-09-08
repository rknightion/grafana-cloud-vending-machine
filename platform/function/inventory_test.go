package main

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
)

func TestRenderStackInventoryUsesOnlyObserveOnlyProviderSets(t *testing.T) {
	xr := inventoryTestXR(map[string]any{
		"stackRef": map[string]any{"name": "example-stack"},
	})

	desired, err := renderStackInventory(xr, nil, nil)
	if err != nil {
		t.Fatalf("renderStackInventory returned an error: %v", err)
	}

	want := map[string]string{
		"folders":        "FolderSet",
		"dashboards":     "DashboardSet",
		"teams":          "TeamSet",
		"users":          "UserSet",
		"library-panels": "LibraryPanelSet",
		"probes":         "ProbeSet",
		"collectors":     "CollectorSet",
	}
	got := map[string]string{}
	for name, child := range desired {
		content := child.Resource.UnstructuredContent()
		got[string(name)] = content["kind"].(string)
		spec := content["spec"].(map[string]any)
		if diff := reflect.DeepEqual(spec["managementPolicies"], []any{"Observe"}); !diff {
			t.Errorf("%s managementPolicies = %#v, want [Observe]", name, spec["managementPolicies"])
		}
		if _, ok := spec["initProvider"]; ok {
			t.Errorf("%s unexpectedly contains initProvider", name)
		}
		providerRef := spec["providerConfigRef"].(map[string]any)
		if providerRef["name"] != "example-stack" || providerRef["kind"] != "ProviderConfig" {
			t.Errorf("%s providerConfigRef = %#v, want namespaced example-stack ProviderConfig", name, providerRef)
		}
		if metadata := content["metadata"].(map[string]any); metadata["namespace"] != "example-namespace" {
			t.Errorf("%s namespace = %v, want example-namespace", name, metadata["namespace"])
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("desired provider set resources = %#v, want %#v", got, want)
	}
}

func TestRenderStackInventoryRendersDeclaredOrganizationUserQueriesReadOnly(t *testing.T) {
	xr := inventoryTestXR(map[string]any{
		"stackRef": map[string]any{"name": "example-stack"},
		"declared": []any{map[string]any{
			"apiVersion": "oss.grafana.o.crossplane.io/v1alpha1",
			"kind":       "OrganizationUser",
			"email":      "operator@example.invalid",
		}},
	})

	desired, err := renderStackInventory(xr, nil, nil)
	if err != nil {
		t.Fatalf("renderStackInventory returned an error: %v", err)
	}

	var found map[string]any
	for name, child := range desired {
		content := child.Resource.UnstructuredContent()
		if content["kind"] == "OrganizationUser" {
			found = content
			if name == "" {
				t.Fatal("organization user child has an empty logical name")
			}
		}
	}
	if found == nil {
		t.Fatal("declared OrganizationUser query was not rendered")
	}
	spec := found["spec"].(map[string]any)
	if diff := reflect.DeepEqual(spec["managementPolicies"], []any{"Observe"}); !diff {
		t.Fatalf("OrganizationUser managementPolicies = %#v, want [Observe]", spec["managementPolicies"])
	}
	if got := spec["forProvider"].(map[string]any)["email"]; got != "operator@example.invalid" {
		t.Fatalf("OrganizationUser query email = %v, want operator@example.invalid", got)
	}
}

func TestRenderStackInventoryRejectsOrganizationUserWithoutQuery(t *testing.T) {
	xr := inventoryTestXR(map[string]any{
		"stackRef": map[string]any{"name": "example-stack"},
		"declared": []any{map[string]any{
			"apiVersion": "oss.grafana.o.crossplane.io/v1alpha1",
			"kind":       "OrganizationUser",
		}},
	})

	if _, err := renderStackInventory(xr, nil, nil); err == nil {
		t.Fatal("renderStackInventory accepted an OrganizationUser without email or login")
	}
}

func TestRunFunctionGatesInventoryAndPublishesStatus(t *testing.T) {
	claim := mustInventoryJSON(inventoryTestXR(map[string]any{
		"stackRef": map[string]any{"name": "teamdemo01"},
		"declared": []any{map[string]any{
			"apiVersion":   "oss.grafana.m.crossplane.io/v1alpha1",
			"kind":         "Folder",
			"externalName": "declared-folder",
		}},
	}))

	unresolved := callFunctionWithRequiredResources(t, claim, nil, nil, nil)
	selector, ok := unresolved.GetRequirements().GetResources()[referencedStackRequirement]
	if !ok {
		t.Fatal("inventory did not request the referenced stack")
	}
	if got, want := selector.GetMatchName(), "teamdemo01"; got != want {
		t.Fatalf("referenced stack name = %q, want %q", got, want)
	}
	if got := len(unresolved.GetDesired().GetResources()); got != 0 {
		t.Fatalf("unresolved referenced stack rendered %d inventory children", got)
	}
	composite := unresolved.GetDesired().GetComposite()
	if composite.GetReady() != fnv1.Ready_READY_FALSE {
		t.Fatalf("unresolved inventory readiness = %s, want READY_FALSE", composite.GetReady())
	}
	status := nestedMap(t, composite.GetResource().AsMap(), "status")
	if got := len(status["declared"].([]any)); got != 1 {
		t.Fatalf("declared status count = %d, want 1", got)
	}

	ready := callFunctionWithRequiredResources(t, claim, nil,
		requiredStackResource("teamdemo01", "example-namespace", "True"), requiredResourceCapabilities())
	if fatal := fatalResult(ready); fatal != "" {
		t.Fatalf("ready inventory returned a fatal result: %s", fatal)
	}
	if got, want := len(ready.GetDesired().GetResources()), len(stackInventoryProviderSets); got != want {
		t.Fatalf("ready inventory rendered %d children, want %d", got, want)
	}
}

func TestStackInventoryStatusDistinguishesDeclaredManagedAndUnmanaged(t *testing.T) {
	xr := inventoryTestXR(map[string]any{
		"stackRef": map[string]any{"name": "example-stack"},
		"declared": []any{
			map[string]any{
				"apiVersion":   "oss.grafana.m.crossplane.io/v1alpha1",
				"kind":         "Folder",
				"externalName": "managed-folder",
			},
			map[string]any{
				"apiVersion":   "oss.grafana.m.crossplane.io/v1alpha1",
				"kind":         "Dashboard",
				"externalName": "managed-dashboard",
			},
		},
	})
	observed := map[resource.Name]resource.ObservedComposed{
		"folders": inventoryObserved(`{
			"apiVersion":"oss.grafana.o.crossplane.io/v1alpha1",
			"kind":"FolderSet",
			"status":{"atProvider":{"folders":[
				{"uid":"managed-folder","title":"Managed folder"},
				{"uid":"unmanaged-folder","title":"Unmanaged folder"}
			]}}
		}`),
		"dashboards": inventoryObserved(`{
			"apiVersion":"oss.grafana.o.crossplane.io/v1alpha1",
			"kind":"DashboardSet",
			"status":{"atProvider":{"dashboards":[
				{"uid":"managed-dashboard","title":"Managed dashboard"},
				{"uid":"unmanaged-dashboard","title":"Unmanaged dashboard"}
			]}}
		}`),
	}

	status := desiredStackInventoryStatus(xr, observed).Resource.UnstructuredContent()["status"].(map[string]any)
	if got, want := len(status["declared"].([]any)), 2; got != want {
		t.Fatalf("declared count = %d, want %d", got, want)
	}
	if got, want := inventoryStatusIdentities(status["observedAndManaged"]), []string{"Dashboard:managed-dashboard", "Folder:managed-folder"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("observedAndManaged = %#v, want %#v", got, want)
	}
	if got, want := inventoryStatusIdentities(status["observedButUnmanaged"]), []string{"Dashboard:unmanaged-dashboard", "Folder:unmanaged-folder"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("observedButUnmanaged = %#v, want %#v", got, want)
	}
}

func TestStackInventoryStatusSupportsMapAndScalarSetOutputs(t *testing.T) {
	xr := inventoryTestXR(map[string]any{
		"stackRef": map[string]any{"name": "example-stack"},
		"declared": []any{map[string]any{
			"apiVersion":   "sm.grafana.o.crossplane.io/v1alpha1",
			"kind":         "Probe",
			"externalName": "42",
		}},
	})
	observed := map[resource.Name]resource.ObservedComposed{
		"probes": inventoryObserved(`{
			"apiVersion":"sm.grafana.o.crossplane.io/v1alpha1",
			"kind":"ProbeSet",
			"status":{"atProvider":{"probes":{"synthetic-check":"42","other-check":"43"}}}
		}`),
		"collectors": inventoryObserved(`{
			"apiVersion":"fleetmanagement.grafana.o.crossplane.io/v1alpha1",
			"kind":"CollectorSet",
			"status":{"atProvider":{"collectors":["collector-a"]}}
		}`),
	}

	status := desiredStackInventoryStatus(xr, observed).Resource.UnstructuredContent()["status"].(map[string]any)
	if got, want := inventoryStatusIdentities(status["observedAndManaged"]), []string{"Probe:42"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("map output managed identities = %#v, want %#v", got, want)
	}
	if got, want := inventoryStatusIdentities(status["observedButUnmanaged"]), []string{"Collector:collector-a", "Probe:43"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("scalar output unmanaged identities = %#v, want %#v", got, want)
	}
}

func inventoryTestXR(spec map[string]any) map[string]any {
	return map[string]any{
		"apiVersion": "platform.example.org/v1beta1",
		"kind":       "GrafanaStackInventory",
		"metadata": map[string]any{
			"name":      "example-inventory",
			"namespace": "example-namespace",
		},
		"spec": spec,
	}
}

func inventoryObserved(document string) resource.ObservedComposed {
	return resource.ObservedComposed{Resource: func() *composed.Unstructured {
		r := composed.New()
		r.SetUnstructuredContent(resource.MustStructJSON(document).AsMap())
		return r
	}()}
}

func inventoryStatusIdentities(value any) []string {
	items, _ := value.([]any)
	identities := make([]string, 0, len(items))
	for _, item := range items {
		object := item.(map[string]any)
		identities = append(identities, object["kind"].(string)+":"+object["externalName"].(string))
	}
	sort.Strings(identities)
	return identities
}

func mustInventoryJSON(value any) string {
	data, _ := json.Marshal(value)
	return string(data)
}
