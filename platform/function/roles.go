package main

import (
	"fmt"
	hashfnv "hash/fnv"

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

func renderTeamAccess(xr map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	metadata, _ := xr["metadata"].(map[string]any)
	spec, _ := xr["spec"].(map[string]any)
	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	stackRef, _ := spec["stackRef"].(map[string]any)
	stackName, _ := stackRef["name"].(string)
	team, _ := spec["team"].(map[string]any)
	teamName, _ := team["name"].(string)
	if name == "" || namespace == "" || stackName == "" || teamName == "" {
		return nil, errors.New("team access must set metadata name and namespace, stackRef.name, and team.name")
	}

	teamResourceName := name + "-team"
	teamParams := map[string]any{"name": teamName}
	copyOptionalFields(teamParams, team, "email", "members", "ignoreExternallySyncedMembers")
	if preferences, ok := team["preferences"].(map[string]any); ok && len(preferences) > 0 {
		teamParams["preferences"] = []any{preferences}
	}
	if groups, ok := team["externalGroups"].([]any); ok && len(groups) > 0 {
		teamParams["teamSync"] = []any{map[string]any{"groups": groups}}
	}

	desired := map[resource.Name]*resource.DesiredComposed{
		"team": newDesired("oss.grafana.m.crossplane.io/v1alpha1", "Team", namespace, teamResourceName, nil,
			map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider":        teamParams,
				"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": stackName},
			}),
	}

	customRoles, _ := spec["customRoles"].([]any)
	for index, item := range customRoles {
		role, _ := item.(map[string]any)
		roleName, _ := role["name"].(string)
		roleUID, _ := role["uid"].(string)
		permissions, _ := role["permissions"].([]any)
		if roleName == "" || roleUID == "" || len(permissions) == 0 {
			return nil, errors.Errorf("customRoles[%d] must set name, uid, and permissions", index)
		}

		suffix := stableResourceSuffix(roleUID)
		logicalName := "custom-role-" + suffix
		roleResourceName := name + "-role-" + suffix
		roleParams := map[string]any{
			"name": roleName, "uid": roleUID, "global": false, "permissions": permissions,
		}
		copyOptionalFields(roleParams, role, "description", "displayName", "group", "hidden")
		desired[resource.Name(logicalName)] = newDesired("enterprise.grafana.m.crossplane.io/v1alpha1", "Role", namespace, roleResourceName, nil,
			map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider":        roleParams,
				"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": stackName},
			})
		desired[resource.Name("custom-role-assignment-"+suffix)] = newDesired("enterprise.grafana.m.crossplane.io/v1alpha1", "RoleAssignmentItem", namespace, name+"-custom-"+suffix, nil,
			map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider": map[string]any{
					"roleRef": map[string]any{"name": roleResourceName},
					"teamRef": map[string]any{"name": teamResourceName},
				},
				"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": stackName},
			})
	}

	fixedRoleUIDs, _ := spec["fixedRoleUids"].([]any)
	for index, item := range fixedRoleUIDs {
		roleUID, _ := item.(string)
		if roleUID == "" {
			return nil, errors.Errorf("fixedRoleUids[%d] must not be empty", index)
		}
		suffix := stableResourceSuffix(roleUID)
		desired[resource.Name("fixed-role-assignment-"+suffix)] = newDesired("enterprise.grafana.m.crossplane.io/v1alpha1", "RoleAssignmentItem", namespace, name+"-fixed-"+suffix, nil,
			map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider": map[string]any{
					"roleUid": roleUID,
					"teamRef": map[string]any{"name": teamResourceName},
				},
				"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": stackName},
			})
	}
	return desired, nil
}

func renderContentAccessPolicy(xr map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	metadata, _ := xr["metadata"].(map[string]any)
	spec, _ := xr["spec"].(map[string]any)
	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	stackRef, _ := spec["stackRef"].(map[string]any)
	stackName, _ := stackRef["name"].(string)
	target, _ := spec["target"].(map[string]any)
	targetKind, _ := target["kind"].(string)
	ref, _ := target["ref"].(map[string]any)
	refName, _ := ref["name"].(string)
	targetUID, _ := ref["uid"].(string)
	permissions, _ := spec["permissions"].([]any)
	if name == "" || namespace == "" || stackName == "" || !oneOf(targetKind, "Folder", "Dashboard") || (refName == "") == (targetUID == "") || len(permissions) == 0 {
		return nil, errors.New("content access policy must set metadata name and namespace, stackRef.name, a Folder or Dashboard target with exactly one ref name or uid, and permissions")
	}

	renderedPermissions := make([]any, 0, len(permissions))
	for index, item := range permissions {
		permission, _ := item.(map[string]any)
		level, _ := permission["permission"].(string)
		basicRole, _ := permission["basicRole"].(string)
		teamRef, _ := permission["teamRef"].(map[string]any)
		teamName, _ := teamRef["name"].(string)
		userID, _ := permission["userId"].(string)
		actors := 0
		if basicRole != "" {
			actors++
		}
		if teamName != "" {
			actors++
		}
		if userID != "" {
			actors++
		}
		if !oneOf(level, "View", "Edit", "Admin") || actors != 1 {
			return nil, errors.Errorf("permissions[%d] must set View, Edit, or Admin and exactly one basicRole, teamRef.name, or userId", index)
		}
		entry := map[string]any{"permission": level}
		if basicRole != "" {
			entry["role"] = basicRole
		}
		if teamName != "" {
			entry["teamRef"] = map[string]any{"name": teamName}
		}
		if userID != "" {
			entry["userId"] = userID
		}
		renderedPermissions = append(renderedPermissions, entry)
	}

	resourceKind := targetKind + "Permission"
	targetKey := "folderRef"
	uidKey := "folderUid"
	if targetKind == "Dashboard" {
		targetKey = "dashboardRef"
		uidKey = "dashboardUid"
	}
	parameters := map[string]any{"permissions": renderedPermissions}
	if refName != "" {
		parameters[targetKey] = map[string]any{"name": refName}
	} else {
		parameters[uidKey] = targetUID
	}

	return map[resource.Name]*resource.DesiredComposed{
		"access-policy": newDesired("oss.grafana.m.crossplane.io/v1alpha1", resourceKind, namespace, name, nil,
			map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider":        parameters,
				"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": stackName},
			}),
	}, nil
}

func copyOptionalFields(destination, source map[string]any, fields ...string) {
	for _, field := range fields {
		if value, ok := source[field]; ok {
			destination[field] = value
		}
	}
}

func stableResourceSuffix(value string) string {
	h := hashfnv.New32a()
	_, _ = h.Write([]byte(value))
	return fmt.Sprintf("%08x", h.Sum32())
}
