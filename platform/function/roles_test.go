package main

import (
	"testing"

	"github.com/crossplane/function-sdk-go/resource"
	"github.com/google/go-cmp/cmp"
)

func TestRoleRenderersFilterPublicDashboardWriteByDefault(t *testing.T) {
	permissions := []any{
		map[string]any{"action": "dashboards.public:write", "scope": "dashboards:uid:public-example"},
		map[string]any{"action": "dashboards:read", "scope": "folders:*"},
	}

	roleBinding, err := renderRoleBinding(map[string]any{
		"metadata": map[string]any{"name": "example-editor", "namespace": "grafana-vending"},
		"spec": map[string]any{
			"stackRef": map[string]any{"name": "teamdemo01"},
			"team":     map[string]any{"name": "Example Editors", "groups": []any{"idp-example-editors"}},
			"role":     map[string]any{"name": "example-editor", "uid": "example-editor", "permissions": permissions},
		},
	})
	if err != nil {
		t.Fatalf("render custom role binding: %v", err)
	}

	teamAccess, err := renderTeamAccess(map[string]any{
		"metadata": map[string]any{"name": "example-editors", "namespace": "grafana-vending"},
		"spec": map[string]any{
			"stackRef": map[string]any{"name": "teamdemo01"},
			"team":     map[string]any{"name": "Example Editors"},
			"customRoles": []any{map[string]any{
				"name": "example-editor", "uid": "example-editor", "permissions": permissions,
			}},
		},
	}, nil)
	if err != nil {
		t.Fatalf("render team access: %v", err)
	}

	want := []any{map[string]any{"action": "dashboards:read", "scope": "folders:*"}}
	got := rolePermissions(t, roleBinding["role"])
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("custom role binding permissions differ (-want +got):\n%s", diff)
	}
	customRoleName := "custom-role-" + stableResourceSuffix("example-editor")
	got = rolePermissions(t, teamAccess[resource.Name(customRoleName)])
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("team access custom role permissions differ (-want +got):\n%s", diff)
	}
}

func rolePermissions(t *testing.T, desired *resource.DesiredComposed) []any {
	t.Helper()
	if desired == nil || desired.Resource == nil {
		t.Fatal("custom role was not rendered")
	}
	content := desired.Resource.UnstructuredContent()
	spec, ok := content["spec"].(map[string]any)
	if !ok {
		t.Fatalf("rendered custom role has no spec: %v", content)
	}
	forProvider, ok := spec["forProvider"].(map[string]any)
	if !ok {
		t.Fatalf("rendered custom role has no forProvider: %v", spec)
	}
	permissions, ok := forProvider["permissions"].([]any)
	if !ok {
		t.Fatalf("rendered custom role has no permissions: %v", forProvider)
	}
	return permissions
}

func TestPublicDashboardWriteIsPreservedOnlyForAnAllowedPlatformProfile(t *testing.T) {
	permissions := []any{
		map[string]any{"action": "dashboards.public:write", "scope": "dashboards:uid:public-example"},
		map[string]any{"action": "dashboards:read", "scope": "folders:*"},
	}

	roleBinding, err := renderRoleBindingWithPlatformProfile(map[string]any{
		"metadata": map[string]any{"name": "example-editor", "namespace": "grafana-vending"},
		"spec": map[string]any{
			"stackRef": map[string]any{"name": "teamdemo01"},
			"team":     map[string]any{"name": "Example Editors", "groups": []any{"idp-example-editors"}},
			"role":     map[string]any{"name": "example-editor", "uid": "example-editor", "permissions": permissions},
		},
	}, "public-dashboards", []string{"public-dashboards"})
	if err != nil {
		t.Fatalf("render allowed custom role binding: %v", err)
	}

	teamAccess, err := renderTeamAccessWithPlatformProfile(map[string]any{
		"metadata": map[string]any{"name": "example-editors", "namespace": "grafana-vending"},
		"spec": map[string]any{
			"stackRef": map[string]any{"name": "teamdemo01"},
			"team":     map[string]any{"name": "Example Editors"},
			"customRoles": []any{map[string]any{
				"name": "example-editor", "uid": "example-editor", "permissions": permissions,
			}},
		},
	}, nil, "public-dashboards", []string{"public-dashboards"})
	if err != nil {
		t.Fatalf("render allowed team access: %v", err)
	}

	got := rolePermissions(t, roleBinding["role"])
	if diff := cmp.Diff(permissions, got); diff != "" {
		t.Fatalf("allowed profile did not preserve custom role binding permissions (-want +got):\n%s", diff)
	}

	customRoleName := "custom-role-" + stableResourceSuffix("example-editor")
	got = rolePermissions(t, teamAccess[resource.Name(customRoleName)])
	if diff := cmp.Diff(permissions, got); diff != "" {
		t.Fatalf("allowed profile did not preserve team access permissions (-want +got):\n%s", diff)
	}
}
