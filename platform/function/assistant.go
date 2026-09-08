package main

import (
	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

const assistantRendererImplemented = true

const assistantAPIVersion = "assistant.grafana.m.crossplane.io/v1alpha1"

func renderAssistantGovernance(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	metadata, _ := xr["metadata"].(map[string]any)
	spec, _ := xr["spec"].(map[string]any)
	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	stackRef, _ := spec["stackRef"].(map[string]any)
	stackName, _ := stackRef["name"].(string)
	if name == "" || namespace == "" || stackName == "" {
		return nil, errors.New("assistant governance must set metadata name and namespace and spec.stackRef.name")
	}

	terms, _ := spec["termsAcceptance"].(map[string]any)
	if terms == nil {
		// Accept the short form for callers that used the provider resource name
		// before this platform API was introduced. The XRD exposes the explicit
		// termsAcceptance field so there is still one documented shape.
		terms, _ = spec["terms"].(map[string]any)
	}
	accepted, ok := terms["accepted"].(bool)
	if !ok {
		return nil, errors.New("assistant governance termsAcceptance.accepted must be a boolean")
	}

	desired := map[resource.Name]*resource.DesiredComposed{}
	desired["terms"] = newDesired(
		assistantAPIVersion,
		"TermsAcceptance",
		namespace,
		name+"-terms",
		map[string]any{"crossplane.io/external-name": "terms"},
		map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider":        map[string]any{"accepted": accepted},
			"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": stackName},
		},
	)

	// A withdrawal is an immediate safety boundary. Do not keep rendering
	// children while the provider is observing the previous accepted state.
	if !accepted {
		return desired, nil
	}
	observedAccepted, observedPresent := observedBool(observed, "terms", "status", "atProvider", "accepted")
	if !observedPresent || !observedAccepted {
		return desired, nil
	}

	rules, _ := spec["rules"].([]any)
	for index, item := range rules {
		rule, ok := item.(map[string]any)
		if !ok {
			return nil, errors.Errorf("rules[%d] must be an object", index)
		}
		ruleName, _ := rule["name"].(string)
		scope, _ := rule["scope"].(string)
		profileName, _ := rule["profile"].(string)
		if ruleName == "" || scope == "" || profileName == "" {
			return nil, errors.Errorf("rules[%d] must set name, profile, and scope", index)
		}
		profile, ok := assistantRuleProfile(config, profileName)
		if !ok {
			return nil, errors.Errorf("assistant rule profile %q is not configured by the platform", profileName)
		}
		ruleContent, _ := profile["ruleContent"].(string)
		if ruleContent == "" {
			return nil, errors.Errorf("assistant rule profile %q must set ruleContent", profileName)
		}

		parameters := map[string]any{
			"name":        ruleName,
			"ruleContent": ruleContent,
			"scope":       scope,
			"enabled":     true,
			"priority":    float64(0),
		}
		copyOptionalFields(parameters, rule, "description", "enabled", "priority", "applications")
		suffix := stableResourceSuffix(ruleName)
		desired[resource.Name("rule-"+suffix)] = newDesired(
			assistantAPIVersion,
			"Rule",
			namespace,
			name+"-rule-"+suffix,
			nil,
			map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider":        parameters,
				"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": stackName},
			},
		)
	}

	servers, _ := spec["mcpServers"].([]any)
	for index, item := range servers {
		server, ok := item.(map[string]any)
		if !ok {
			return nil, errors.Errorf("mcpServers[%d] must be an object", index)
		}
		serverName, _ := server["name"].(string)
		scope, _ := server["scope"].(string)
		if serverName == "" || scope == "" {
			return nil, errors.Errorf("mcpServers[%d] must set name and scope", index)
		}

		parameters := map[string]any{
			"name":    serverName,
			"scope":   scope,
			"enabled": true,
		}
		copyOptionalFields(parameters, server, "description", "enabled", "applications")
		if raw, exists := server["configuration"]; exists {
			configuration, ok := raw.(map[string]any)
			if !ok {
				return nil, errors.Errorf("mcpServers[%d].configuration must be an object", index)
			}
			normalized, err := assistantMCPConfiguration(configuration, serverName)
			if err != nil {
				return nil, err
			}
			if len(normalized) > 0 {
				parameters["configuration"] = normalized
			}
		}
		if raw, exists := server["customHeadersSecretRef"]; exists {
			ref, ok := raw.(map[string]any)
			if !ok {
				return nil, errors.Errorf("mcpServers[%d].customHeadersSecretRef must be an object", index)
			}
			secretName, _ := ref["name"].(string)
			if secretName == "" {
				return nil, errors.Errorf("mcpServers[%d].customHeadersSecretRef.name must be set", index)
			}
			parameters["customHeadersSecretRef"] = map[string]any{"name": secretName}
		}

		suffix := stableResourceSuffix(serverName)
		desired[resource.Name("mcp-server-"+suffix)] = newDesired(
			assistantAPIVersion,
			"McpServer",
			namespace,
			name+"-mcp-"+suffix,
			nil,
			map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider":        parameters,
				"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": stackName},
			},
		)
	}

	return desired, nil
}

func assistantRuleProfile(config map[string]any, name string) (map[string]any, bool) {
	for _, field := range []string{"ruleProfiles", "assistantRuleProfiles", "assistantRules"} {
		if profile, ok := configuredProfile(config, field, name); ok {
			return profile, true
		}
	}
	return nil, false
}

func assistantMCPConfiguration(configuration map[string]any, serverName string) (map[string]any, error) {
	result := map[string]any{}
	for _, field := range []string{"url", "builtinId"} {
		if value, ok := configuration[field]; ok {
			text, ok := value.(string)
			if !ok {
				return nil, errors.Errorf("MCP server %q configuration.%s must be a string", serverName, field)
			}
			if text != "" {
				result[field] = text
			}
		}
	}

	preferences, err := assistantStringMap(configuration["toolPreferences"], "toolPreferences", serverName)
	if err != nil {
		return nil, err
	}
	for tool, preference := range preferences {
		if preference != "enabled" && preference != "disabled" {
			return nil, errors.Errorf("MCP server %q tool preference for %q must be enabled or disabled", serverName, tool)
		}
	}
	if len(preferences) > 0 {
		result["toolPreferences"] = preferences
	}

	requestedPolicies, err := assistantStringMap(configuration["toolApprovalPolicies"], "toolApprovalPolicies", serverName)
	if err != nil {
		return nil, err
	}
	policies := map[string]any{}
	for tool, policy := range requestedPolicies {
		if policy != "" && policy != "auto_approve" && policy != "always_ask" {
			return nil, errors.Errorf("MCP server %q approval policy for %q must be auto_approve or always_ask", serverName, tool)
		}
		policies[tool] = policy
	}
	for tool := range preferences {
		if _, ok := policies[tool]; !ok {
			policies[tool] = "always_ask"
		}
	}
	for tool, policy := range policies {
		if policy == "" {
			policies[tool] = "always_ask"
		}
	}
	if len(policies) > 0 {
		result["toolApprovalPolicies"] = policies
	}

	return result, nil
}

func assistantStringMap(value any, field, serverName string) (map[string]any, error) {
	if value == nil {
		return map[string]any{}, nil
	}
	result := map[string]any{}
	switch values := value.(type) {
	case map[string]any:
		for key, raw := range values {
			text, ok := raw.(string)
			if !ok {
				return nil, errors.Errorf("MCP server %q configuration.%s[%q] must be a string", serverName, field, key)
			}
			result[key] = text
		}
	case map[string]string:
		for key, text := range values {
			result[key] = text
		}
	default:
		return nil, errors.Errorf("MCP server %q configuration.%s must be a map", serverName, field)
	}
	return result, nil
}
