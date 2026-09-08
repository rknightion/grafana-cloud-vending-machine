package main

import (
	"encoding/json"
	"strings"
	"testing"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/google/go-cmp/cmp"
)

func TestDatasourceAccessAggregatesAllTeamRulesIntoOneLBACResource(t *testing.T) {
	desired, err := renderDatasourceAccess(datasourceAccessClaim(), nil, nil)
	if err != nil {
		t.Fatalf("renderDatasourceAccess returned an error: %v", err)
	}

	var lbacCount int
	for _, child := range desired {
		if child.Resource.GetKind() == "DataSourceConfigLbacRules" {
			lbacCount++
		}
	}
	if lbacCount != 1 {
		t.Fatalf("rendered %d LBAC resources, want exactly one", lbacCount)
	}

	lbac := datasourceAccessObject(t, desired, "lbac-rules")
	parameters := datasourceAccessNestedMap(t, lbac, "spec", "forProvider")
	var got map[string][]string
	if err := json.Unmarshal([]byte(parameters["rules"].(string)), &got); err != nil {
		t.Fatalf("LBAC rules are not JSON: %v", err)
	}
	want := map[string][]string{
		"team-blue":  {`{ namespace = "blue" }`},
		"team-green": {`{ namespace = "green" }`, `{ environment = "production" }`},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("aggregated LBAC rules differ (-want +got):\n%s", diff)
	}
	if got := datasourceAccessExternalName(t, lbac); got != "shared-metrics" {
		t.Fatalf("LBAC external name = %q, want shared-metrics", got)
	}
}

func TestDatasourceAccessOwnerIdentityMustMatchDatasourceUID(t *testing.T) {
	claim := datasourceAccessClaim()
	claim["metadata"].(map[string]any)["name"] = "second-owner"

	_, err := renderDatasourceAccess(claim, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "metadata.name must match spec.datasource.uid") {
		t.Fatalf("second datasource owner error = %v, want identity rejection", err)
	}
}

func TestDatasourceAccessPermissionSetRemovesBroadQueryGrants(t *testing.T) {
	desired, err := renderDatasourceAccess(datasourceAccessClaim(), nil, nil)
	if err != nil {
		t.Fatalf("renderDatasourceAccess returned an error: %v", err)
	}

	var permissionSetCount int
	for _, child := range desired {
		switch child.Resource.GetKind() {
		case "DataSourcePermission":
			permissionSetCount++
		case "DataSourcePermissionItem":
			t.Fatal("rendered a permission item; only the authoritative permission set can remove omitted broad grants")
		}
	}
	if permissionSetCount != 1 {
		t.Fatalf("rendered %d datasource permission sets, want exactly one", permissionSetCount)
	}

	permissionSet := datasourceAccessObject(t, desired, "permissions")
	if got := datasourceAccessExternalName(t, permissionSet); got != "shared-metrics" {
		t.Fatalf("permission-set external name = %q, want datasource UID shared-metrics", got)
	}
	parameters := datasourceAccessNestedMap(t, permissionSet, "spec", "forProvider")
	permissions, ok := parameters["permissions"].([]any)
	if !ok || len(permissions) != 2 {
		t.Fatalf("permissions = %T %v, want two explicit team grants", parameters["permissions"], parameters["permissions"])
	}
	for index, item := range permissions {
		permission, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("permissions[%d] = %T, want object", index, item)
		}
		if got := permission["permission"]; got != "Query" {
			t.Errorf("permissions[%d].permission = %v, want Query", index, got)
		}
		if _, exists := permission["builtInRole"]; exists {
			t.Errorf("permissions[%d] retains a broad built-in-role grant", index)
		}
		if _, exists := permission["userId"]; exists {
			t.Errorf("permissions[%d] contains an undeclared user grant", index)
		}
		if _, exists := permission["teamId"]; !exists {
			t.Errorf("permissions[%d] has no teamId", index)
		}
	}
}

func TestDatasourceAccessUsesRequestDerivedExternalNamesOnTheFirstPass(t *testing.T) {
	desired, err := renderDatasourceAccess(datasourceAccessClaim(), nil, nil)
	if err != nil {
		t.Fatalf("renderDatasourceAccess returned an error: %v", err)
	}
	for _, name := range []resource.Name{"datasource", "permissions", "lbac-rules"} {
		object := datasourceAccessObject(t, desired, name)
		if got := datasourceAccessExternalName(t, object); got != "shared-metrics" {
			t.Errorf("%s external name = %q, want datasource UID shared-metrics", name, got)
		}
	}
}

func TestDatasourceAccessRefusesUnsupportedAuthentication(t *testing.T) {
	claim := datasourceAccessClaim()
	claim["spec"].(map[string]any)["datasource"].(map[string]any)["authMode"] = "oauth"

	_, err := renderDatasourceAccess(claim, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "LBAC requires authMode basicAuth") {
		t.Fatalf("unsupported authentication error = %v, want fail-closed refusal", err)
	}
}

func TestDatasourceAccessGatesChildrenOnTheReferencedStack(t *testing.T) {
	claim := mustJSON(datasourceAccessClaim())
	unresolved := callFunctionWithRequiredResources(t, claim, nil, nil, requiredResourceCapabilities())
	if fatal := fatalResult(unresolved); fatal != "" {
		t.Fatalf("unresolved stack gate returned a fatal result: %s", fatal)
	}
	if got := len(unresolved.GetDesired().GetResources()); got != 0 {
		t.Fatalf("unresolved stack gate admitted %d children, want none", got)
	}
	selector := unresolved.GetRequirements().GetResources()[referencedStackRequirement]
	if selector == nil || selector.GetApiVersion() != "platform.example.org/v1beta1" || selector.GetKind() != "GrafanaCloudStackRequest" || selector.GetMatchName() != "teamdemo01" || selector.GetNamespace() != "grafana-vending" {
		t.Fatalf("referenced stack selector = %#v, want the claim's namespaced stack", selector)
	}

	ready := callFunctionWithRequiredResources(
		t,
		claim,
		nil,
		requiredStackResource("teamdemo01", "grafana-vending", "True"),
		[]fnv1.Capability{fnv1.Capability_CAPABILITY_CAPABILITIES, fnv1.Capability_CAPABILITY_REQUIRED_RESOURCES},
	)
	if fatal := fatalResult(ready); fatal != "" {
		t.Fatalf("ready stack gate returned a fatal result: %s", fatal)
	}
	for _, name := range []string{"datasource", "lbac-rules"} {
		if _, exists := ready.GetDesired().GetResources()[name]; !exists {
			t.Errorf("ready stack gate did not admit %s", name)
		}
	}
}

func datasourceAccessClaim() map[string]any {
	return map[string]any{
		"apiVersion": "platform.example.org/v1beta1",
		"kind":       "GrafanaDatasourceAccess",
		"metadata": map[string]any{
			"name":      "shared-metrics",
			"namespace": "grafana-vending",
		},
		"spec": map[string]any{
			"stackRef": map[string]any{"name": "teamdemo01"},
			"datasource": map[string]any{
				"uid":      "shared-metrics",
				"name":     "Shared metrics",
				"type":     "prometheus",
				"authMode": "basicAuth",
			},
			"teams": []any{
				map[string]any{"uid": "team-blue", "id": "101", "rules": []any{`{ namespace = "blue" }`}},
				map[string]any{"uid": "team-green", "id": "202", "rules": []any{`{ namespace = "green" }`, `{ environment = "production" }`}},
			},
		},
	}
}

func datasourceAccessObject(t *testing.T, desired map[resource.Name]*resource.DesiredComposed, name resource.Name) map[string]any {
	t.Helper()
	child, ok := desired[name]
	if !ok {
		t.Fatalf("desired resource %q was not rendered", name)
	}
	return child.Resource.UnstructuredContent()
}

func datasourceAccessNestedMap(t *testing.T, object map[string]any, fields ...string) map[string]any {
	t.Helper()
	current := object
	for _, field := range fields {
		next, ok := current[field].(map[string]any)
		if !ok {
			t.Fatalf("field %q in path %v is %T, want object", field, fields, current[field])
		}
		current = next
	}
	return current
}

func datasourceAccessExternalName(t *testing.T, object map[string]any) string {
	t.Helper()
	annotations := datasourceAccessNestedMap(t, object, "metadata", "annotations")
	value, _ := annotations["crossplane.io/external-name"].(string)
	return value
}
