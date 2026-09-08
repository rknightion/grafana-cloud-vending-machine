package main

import (
	"fmt"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

const agentObservabilityRendererImplemented = true

const agentObservabilityAPIVersion = "agento11y.grafana.m.crossplane.io/v1alpha1"

// renderAgentObservability renders only the resources explicitly declared by a
// GrafanaAgentObservability request. The guards and workload sections are kept
// separate in the request API because they have different owners: guards are
// platform policy, while collections, evaluators, and evaluation rules belong
// to the workload being observed.
func renderAgentObservability(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, _ map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	metadata, _ := xr["metadata"].(map[string]any)
	spec, _ := xr["spec"].(map[string]any)
	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	stackRef, _ := spec["stackRef"].(map[string]any)
	stackName, _ := stackRef["name"].(string)
	if name == "" || namespace == "" || stackName == "" {
		return nil, errors.New("agent observability claim must set metadata name and namespace plus stackRef.name")
	}

	guards, err := agentObservabilityObject(spec, "guards")
	if err != nil {
		return nil, err
	}
	workload, err := agentObservabilityObject(spec, "workload")
	if err != nil {
		return nil, err
	}

	desired := map[resource.Name]*resource.DesiredComposed{}
	collectionResources, err := renderAgentObservabilityCollections(desired, workload, namespace, stackName, name)
	if err != nil {
		return nil, err
	}
	if err := renderAgentObservabilityEvaluators(desired, workload, namespace, stackName, name); err != nil {
		return nil, err
	}
	if err := renderAgentObservabilityEvaluationRules(desired, workload, namespace, stackName, name); err != nil {
		return nil, err
	}
	if err := renderAgentObservabilityHookRules(desired, guards, namespace, stackName, name); err != nil {
		return nil, err
	}
	if err := renderAgentObservabilityRuleActions(desired, observed, guards, collectionResources, namespace, stackName, name); err != nil {
		return nil, err
	}
	return desired, nil
}

func agentObservabilityObject(spec map[string]any, field string) (map[string]any, error) {
	value, ok := spec[field]
	if !ok {
		return map[string]any{}, nil
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, errors.Errorf("spec.%s must be an object", field)
	}
	return object, nil
}

func agentObservabilityList(object map[string]any, field, path string) ([]any, error) {
	value, ok := object[field]
	if !ok {
		return nil, nil
	}
	items, ok := value.([]any)
	if !ok {
		return nil, errors.Errorf("%s must be an array", path)
	}
	return items, nil
}

func agentObservabilityItem(value any, path string) (map[string]any, error) {
	item, ok := value.(map[string]any)
	if !ok {
		return nil, errors.Errorf("%s must be an object", path)
	}
	return item, nil
}

func agentObservabilityRequiredString(object map[string]any, field, path string) (string, error) {
	value, ok := object[field]
	if !ok {
		return "", errors.Errorf("%s must set %s", path, field)
	}
	result, ok := value.(string)
	if !ok || result == "" {
		return "", errors.Errorf("%s.%s must be a non-empty string", path, field)
	}
	return result, nil
}

func agentObservabilityStringList(value any, path string, required bool) ([]any, error) {
	items, ok := value.([]any)
	if !ok {
		return nil, errors.Errorf("%s must be an array", path)
	}
	if required && len(items) == 0 {
		return nil, errors.Errorf("%s must not be empty", path)
	}
	for index, item := range items {
		value, ok := item.(string)
		if !ok || value == "" {
			return nil, errors.Errorf("%s[%d] must be a non-empty string", path, index)
		}
	}
	return items, nil
}

func renderAgentObservabilityCollections(desired map[resource.Name]*resource.DesiredComposed, workload map[string]any, namespace, stackName, claimName string) (map[string]resource.Name, error) {
	items, err := agentObservabilityList(workload, "collections", "spec.workload.collections")
	if err != nil {
		return nil, err
	}
	resources := map[string]resource.Name{}
	for index, value := range items {
		path := fmt.Sprintf("spec.workload.collections[%d]", index)
		item, err := agentObservabilityItem(value, path)
		if err != nil {
			return nil, err
		}
		collectionName, err := agentObservabilityRequiredString(item, "name", path)
		if err != nil {
			return nil, err
		}
		if _, exists := resources[collectionName]; exists {
			return nil, errors.Errorf("%s.name %q is duplicated", path, collectionName)
		}
		suffix := stableResourceSuffix(collectionName)
		logicalName := resource.Name("workload-collection-" + suffix)
		resourceName := claimName + "-collection-" + suffix
		parameters := map[string]any{"name": collectionName}
		copyOptionalFields(parameters, item, "description")
		desired[logicalName] = newDesired(agentObservabilityAPIVersion, "Collection", namespace, resourceName, nil, map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider":        parameters,
			"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": stackName},
		})
		resources[collectionName] = logicalName
	}
	return resources, nil
}

func renderAgentObservabilityEvaluators(desired map[resource.Name]*resource.DesiredComposed, workload map[string]any, namespace, stackName, claimName string) error {
	items, err := agentObservabilityList(workload, "evaluators", "spec.workload.evaluators")
	if err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for index, value := range items {
		path := fmt.Sprintf("spec.workload.evaluators[%d]", index)
		item, err := agentObservabilityItem(value, path)
		if err != nil {
			return err
		}
		evaluatorID, err := agentObservabilityRequiredString(item, "evaluatorId", path)
		if err != nil {
			return err
		}
		if _, exists := seen[evaluatorID]; exists {
			return errors.Errorf("%s.evaluatorId %q is duplicated", path, evaluatorID)
		}
		seen[evaluatorID] = struct{}{}
		kind, err := agentObservabilityRequiredString(item, "kind", path)
		if err != nil {
			return err
		}
		config, err := agentObservabilityRequiredString(item, "config", path)
		if err != nil {
			return err
		}
		outputKeys, err := agentObservabilityRequiredString(item, "outputKeys", path)
		if err != nil {
			return err
		}
		version, err := agentObservabilityRequiredString(item, "version", path)
		if err != nil {
			return err
		}
		parameters := map[string]any{
			"config": config, "evaluatorId": evaluatorID, "kind": kind, "outputKeys": outputKeys, "version": version,
		}
		copyOptionalFields(parameters, item, "description")
		suffix := stableResourceSuffix(evaluatorID)
		desired[resource.Name("workload-evaluator-"+suffix)] = newDesired(agentObservabilityAPIVersion, "Evaluator", namespace, claimName+"-evaluator-"+suffix,
			map[string]any{"crossplane.io/external-name": evaluatorID},
			map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider":        parameters,
				"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": stackName},
			})
	}
	return nil
}

func renderAgentObservabilityEvaluationRules(desired map[resource.Name]*resource.DesiredComposed, workload map[string]any, namespace, stackName, claimName string) error {
	items, err := agentObservabilityList(workload, "evaluationRules", "spec.workload.evaluationRules")
	if err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for index, value := range items {
		path := fmt.Sprintf("spec.workload.evaluationRules[%d]", index)
		item, err := agentObservabilityItem(value, path)
		if err != nil {
			return err
		}
		ruleID, err := agentObservabilityRequiredString(item, "ruleId", path)
		if err != nil {
			return err
		}
		if _, exists := seen[ruleID]; exists {
			return errors.Errorf("%s.ruleId %q is duplicated", path, ruleID)
		}
		seen[ruleID] = struct{}{}
		evaluatorIDs, ok := item["evaluatorIds"]
		if !ok {
			return errors.Errorf("%s must set evaluatorIds", path)
		}
		evaluatorIDList, err := agentObservabilityStringList(evaluatorIDs, path+".evaluatorIds", true)
		if err != nil {
			return err
		}
		parameters := map[string]any{"evaluatorIds": evaluatorIDList, "ruleId": ruleID}
		copyOptionalFields(parameters, item, "alertRuleUids", "enabled", "executionMode", "filterableTagKeys", "match", "minIdleSeconds", "sampleRate", "selector")
		suffix := stableResourceSuffix(ruleID)
		desired[resource.Name("workload-evaluation-rule-"+suffix)] = newDesired(agentObservabilityAPIVersion, "EvaluationRule", namespace, claimName+"-evaluation-rule-"+suffix,
			map[string]any{"crossplane.io/external-name": ruleID},
			map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider":        parameters,
				"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": stackName},
			})
	}
	return nil
}

func renderAgentObservabilityHookRules(desired map[resource.Name]*resource.DesiredComposed, guards map[string]any, namespace, stackName, claimName string) error {
	items, err := agentObservabilityList(guards, "hookRules", "spec.guards.hookRules")
	if err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for index, value := range items {
		path := fmt.Sprintf("spec.guards.hookRules[%d]", index)
		item, err := agentObservabilityItem(value, path)
		if err != nil {
			return err
		}
		ruleID, err := agentObservabilityRequiredString(item, "ruleId", path)
		if err != nil {
			return err
		}
		if _, exists := seen[ruleID]; exists {
			return errors.Errorf("%s.ruleId %q is duplicated", path, ruleID)
		}
		seen[ruleID] = struct{}{}
		parameters := map[string]any{"ruleId": ruleID}
		copyOptionalFields(parameters, item, "actionOnFail", "enabled", "evaluatorIds", "match", "phase", "priority", "selector", "shortCircuit")
		if value, ok := item["evaluatorIds"]; ok {
			if _, err := agentObservabilityStringList(value, path+".evaluatorIds", false); err != nil {
				return err
			}
		}
		if value, ok := item["blockedTools"]; ok {
			blockedTools, err := agentObservabilityStringList(value, path+".blockedTools", false)
			if err != nil {
				return err
			}
			parameters["blockedTools"] = blockedTools
		}
		redact, hasRedact := item["redact"]
		if hasRedact {
			patterns, err := agentObservabilityRedactionPatterns(redact, path+".redact")
			if err != nil {
				return err
			}
			parameters["redact"] = patterns
		}
		if !hasNonEmptyAgentObservabilityList(item["evaluatorIds"]) && !hasNonEmptyAgentObservabilityList(item["blockedTools"]) && !hasNonEmptyAgentObservabilityList(redact) {
			return errors.Errorf("%s must set evaluatorIds, blockedTools, or redact", path)
		}
		suffix := stableResourceSuffix(ruleID)
		desired[resource.Name("guard-hook-"+suffix)] = newDesired(agentObservabilityAPIVersion, "HookRule", namespace, claimName+"-hook-"+suffix,
			map[string]any{"crossplane.io/external-name": ruleID},
			map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider":        parameters,
				"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": stackName},
			})
	}
	return nil
}

func hasNonEmptyAgentObservabilityList(value any) bool {
	items, ok := value.([]any)
	return ok && len(items) > 0
}

func agentObservabilityRedactionPatterns(value any, path string) ([]any, error) {
	items, ok := value.([]any)
	if !ok {
		return nil, errors.Errorf("%s must be an array", path)
	}
	patterns := make([]any, 0, len(items))
	for index, value := range items {
		itemPath := fmt.Sprintf("%s[%d]", path, index)
		item, err := agentObservabilityItem(value, itemPath)
		if err != nil {
			return nil, err
		}
		regex, err := agentObservabilityRequiredString(item, "regex", itemPath)
		if err != nil {
			return nil, err
		}
		pattern := map[string]any{"regex": regex}
		copyOptionalFields(pattern, item, "id")
		patterns = append(patterns, pattern)
	}
	return patterns, nil
}

func renderAgentObservabilityRuleActions(desired map[resource.Name]*resource.DesiredComposed, observed map[resource.Name]resource.ObservedComposed, guards map[string]any, collectionResources map[string]resource.Name, namespace, stackName, claimName string) error {
	items, err := agentObservabilityList(guards, "ruleActions", "spec.guards.ruleActions")
	if err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for index, value := range items {
		path := fmt.Sprintf("spec.guards.ruleActions[%d]", index)
		item, err := agentObservabilityItem(value, path)
		if err != nil {
			return err
		}
		localName, err := agentObservabilityRequiredString(item, "name", path)
		if err != nil {
			return err
		}
		if _, exists := seen[localName]; exists {
			return errors.Errorf("%s.name %q is duplicated", path, localName)
		}
		seen[localName] = struct{}{}
		ruleID, err := agentObservabilityRequiredString(item, "ruleId", path)
		if err != nil {
			return err
		}
		condition, err := agentObservabilityRequiredString(item, "condition", path)
		if err != nil {
			return err
		}
		if !oneOf(condition, "all_evaluators_pass", "all_evaluators_fail") {
			return errors.Errorf("%s.condition %q is unsupported", path, condition)
		}
		refs, ok := item["collectionRefs"]
		if !ok {
			return errors.Errorf("%s must set collectionRefs", path)
		}
		refItems, ok := refs.([]any)
		if !ok || len(refItems) == 0 {
			return errors.Errorf("%s.collectionRefs must be a non-empty array", path)
		}
		collectionIDs := make([]any, 0, len(refItems))
		for refIndex, value := range refItems {
			refPath := fmt.Sprintf("%s.collectionRefs[%d]", path, refIndex)
			ref, err := agentObservabilityItem(value, refPath)
			if err != nil {
				return err
			}
			collectionName, err := agentObservabilityRequiredString(ref, "name", refPath)
			if err != nil {
				return err
			}
			collectionResource, exists := collectionResources[collectionName]
			if !exists {
				return errors.Errorf("%s references undeclared collection %q", refPath, collectionName)
			}
			collectionID := observedString(observed, collectionResource, "status.atProvider.id")
			if collectionID == "" {
				// Collection IDs are provider-assigned. Do not invent one or
				// emit a RuleAction that the provider cannot reconcile yet.
				collectionIDs = nil
				break
			}
			collectionIDs = append(collectionIDs, collectionID)
		}
		if len(collectionIDs) != len(refItems) {
			continue
		}
		parameters := map[string]any{"collectionIds": collectionIDs, "condition": condition, "ruleId": ruleID}
		copyOptionalFields(parameters, item, "enabled")
		logicalName := resource.Name("guard-action-" + localName)
		desired[logicalName] = newDesired(agentObservabilityAPIVersion, "RuleAction", namespace, claimName+"-action-"+localName, nil,
			map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider":        parameters,
				"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": stackName},
			})
	}
	return nil
}
