package main

import (
	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

func renderRoleBinding(xr map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	metadata, _ := xr["metadata"].(map[string]any)
	spec, _ := xr["spec"].(map[string]any)
	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	stackRef, _ := spec["stackRef"].(map[string]any)
	stackName, _ := stackRef["name"].(string)
	team, _ := spec["team"].(map[string]any)
	teamName, _ := team["name"].(string)
	groups, _ := team["groups"].([]any)
	role, _ := spec["role"].(map[string]any)
	roleName, _ := role["name"].(string)
	roleUID, _ := role["uid"].(string)
	permissions, _ := role["permissions"].([]any)
	if name == "" || namespace == "" || stackName == "" || teamName == "" || len(groups) == 0 || roleName == "" || roleUID == "" || len(permissions) == 0 {
		return nil, errors.New("role binding must set metadata name and namespace, stackRef.name, team name/groups, and role name/uid/permissions")
	}

	teamResourceName := name + "-team"
	roleResourceName := name + "-role"
	roleParams := map[string]any{
		"name": roleName, "uid": roleUID, "global": false, "permissions": permissions,
	}
	if description, ok := role["description"].(string); ok && description != "" {
		roleParams["description"] = description
	}
	if displayName, ok := role["displayName"].(string); ok && displayName != "" {
		roleParams["displayName"] = displayName
	}

	desired := map[resource.Name]*resource.DesiredComposed{}
	desired["team"] = newDesired("oss.grafana.m.crossplane.io/v1alpha1", "Team", namespace, teamResourceName, nil,
		map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider":        map[string]any{"name": teamName, "teamSync": []any{map[string]any{"groups": groups}}},
			"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": stackName},
		})
	desired["role"] = newDesired("enterprise.grafana.m.crossplane.io/v1alpha1", "Role", namespace, roleResourceName, nil,
		map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider":        roleParams,
			"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": stackName},
		})
	desired["role-assignment"] = newDesired("enterprise.grafana.m.crossplane.io/v1alpha1", "RoleAssignment", namespace, name+"-assignment", nil,
		map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider": map[string]any{
				"roleRef":  map[string]any{"name": roleResourceName},
				"teamRefs": []any{map[string]any{"name": teamResourceName}},
			},
			"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": stackName},
		})
	return desired, nil
}
