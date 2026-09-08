package main

import (
	"encoding/json"
	"sort"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

const datasourceAccessRendererImplemented = true

type datasourceAccessTeam struct {
	uid   string
	id    string
	rules []string
}

func renderDatasourceAccess(xr map[string]any, _ map[resource.Name]resource.ObservedComposed, _ map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	metadata, _ := xr["metadata"].(map[string]any)
	spec, _ := xr["spec"].(map[string]any)
	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	stackRef, _ := spec["stackRef"].(map[string]any)
	stackName, _ := stackRef["name"].(string)
	datasource, _ := spec["datasource"].(map[string]any)
	datasourceUID, _ := datasource["uid"].(string)
	datasourceName, _ := datasource["name"].(string)
	datasourceType, _ := datasource["type"].(string)
	authMode, _ := datasource["authMode"].(string)
	if name == "" || namespace == "" || stackName == "" || datasourceUID == "" || datasourceName == "" || datasourceType == "" {
		return nil, errors.New("datasource access must set metadata name and namespace, stackRef.name, and datasource uid, name, and type")
	}
	if name != datasourceUID {
		return nil, errors.New("metadata.name must match spec.datasource.uid so Kubernetes admission enforces one owner per datasource")
	}
	if authMode != "basicAuth" {
		return nil, errors.Errorf("LBAC requires authMode basicAuth; unsupported datasource auth mode %q", authMode)
	}

	teams, err := datasourceAccessTeams(spec)
	if err != nil {
		return nil, err
	}

	datasourceParameters := map[string]any{
		"uid":              datasourceUID,
		"name":             datasourceName,
		"type":             datasourceType,
		"basicAuthEnabled": true,
	}
	if connection, ok := datasource["connection"].(map[string]any); ok {
		copyDatasourceAccessOptionalFields(datasourceParameters, connection,
			"accessMode",
			"basicAuthUsername",
			"databaseName",
			"httpHeadersSecretRef",
			"isDefault",
			"jsonDataEncoded",
			"secureJsonDataEncodedSecretRef",
			"url",
			"username",
		)
	}

	desired := map[resource.Name]*resource.DesiredComposed{
		"datasource": newDesired(
			"oss.grafana.m.crossplane.io/v1alpha1",
			"DataSource",
			namespace,
			name,
			map[string]any{"crossplane.io/external-name": datasourceUID},
			map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider":        datasourceParameters,
				"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": stackName},
			},
		),
	}

	rules := make(map[string][]string, len(teams))
	permissions := make([]any, 0, len(teams))
	for _, team := range teams {
		rules[team.uid] = team.rules
		permissions = append(permissions, map[string]any{
			"teamId":     team.id,
			"permission": "Query",
		})
	}
	rulesJSON, err := json.Marshal(rules)
	if err != nil {
		return nil, errors.Wrap(err, "cannot encode aggregated LBAC rules")
	}
	desired["lbac-rules"] = newDesired(
		"enterprise.grafana.m.crossplane.io/v1alpha1",
		"DataSourceConfigLbacRules",
		namespace,
		name+"-lbac",
		map[string]any{"crossplane.io/external-name": datasourceUID},
		map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider": map[string]any{
				"datasourceUid": datasourceUID,
				"rules":         string(rulesJSON),
			},
			"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": stackName},
		},
	)
	desired["permissions"] = newDesired(
		"enterprise.grafana.m.crossplane.io/v1alpha1",
		"DataSourcePermission",
		namespace,
		name+"-permissions",
		map[string]any{"crossplane.io/external-name": datasourceUID},
		map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider": map[string]any{
				"datasourceUid":  datasourceUID,
				"datasourceType": datasourceType,
				"permissions":    permissions,
			},
			"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": stackName},
		},
	)

	return desired, nil
}

func datasourceAccessTeams(spec map[string]any) ([]datasourceAccessTeam, error) {
	items, _ := spec["teams"].([]any)
	if len(items) == 0 {
		return nil, errors.New("datasource access must declare at least one team with LBAC rules")
	}

	teams := make([]datasourceAccessTeam, 0, len(items))
	seenUIDs := map[string]struct{}{}
	seenIDs := map[string]struct{}{}
	for index, item := range items {
		value, _ := item.(map[string]any)
		uid, _ := value["uid"].(string)
		id, _ := value["id"].(string)
		ruleItems, _ := value["rules"].([]any)
		if uid == "" || id == "" || len(ruleItems) == 0 {
			return nil, errors.Errorf("teams[%d] must set uid, id, and at least one LBAC rule", index)
		}
		if _, exists := seenUIDs[uid]; exists {
			return nil, errors.Errorf("teams[%d] repeats team UID %q", index, uid)
		}
		if _, exists := seenIDs[id]; exists {
			return nil, errors.Errorf("teams[%d] repeats team ID %q", index, id)
		}
		seenUIDs[uid] = struct{}{}
		seenIDs[id] = struct{}{}

		rules := make([]string, 0, len(ruleItems))
		for ruleIndex, ruleItem := range ruleItems {
			rule, ok := ruleItem.(string)
			if !ok || rule == "" {
				return nil, errors.Errorf("teams[%d].rules[%d] must be a non-empty string", index, ruleIndex)
			}
			rules = append(rules, rule)
		}
		teams = append(teams, datasourceAccessTeam{uid: uid, id: id, rules: rules})
	}

	sort.Slice(teams, func(i, j int) bool { return teams[i].uid < teams[j].uid })
	return teams, nil
}

func copyDatasourceAccessOptionalFields(destination, source map[string]any, fields ...string) {
	for _, field := range fields {
		if value, ok := source[field]; ok {
			destination[field] = value
		}
	}
}
