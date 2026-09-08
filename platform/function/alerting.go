package main

import (
	"fmt"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

const alertingRendererImplemented = true

// renderAlertingBundle owns discrete, stack-scoped alerting objects. It
// deliberately does not render NotificationPolicy or RoutingtreeV1Beta1:
// either would replace the organisation-wide routing tree. Rules instead use
// Grafana's simplified per-rule notification settings.
func renderAlertingBundle(xr map[string]any, _ map[resource.Name]resource.ObservedComposed, _ map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	metadata, _ := xr["metadata"].(map[string]any)
	spec, _ := xr["spec"].(map[string]any)
	bundleName, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	stackRef, _ := spec["stackRef"].(map[string]any)
	stackName, _ := stackRef["name"].(string)
	provenance, _ := spec["provenance"].(string)
	if bundleName == "" || namespace == "" || stackName == "" {
		return nil, errors.New("alerting bundle must set metadata name and namespace plus stackRef.name")
	}
	if !oneOf(provenance, "enforced", "createOnly") {
		return nil, errors.Errorf("alerting bundle provenance must be enforced or createOnly, got %q", provenance)
	}

	desired := map[resource.Name]*resource.DesiredComposed{}
	contactPoints, _ := spec["contactPoints"].([]any)
	contactNames := map[string]string{}
	for index, item := range contactPoints {
		contact, ok := item.(map[string]any)
		if !ok {
			return nil, errors.Errorf("contactPoints[%d] must be an object", index)
		}
		shortName, _ := contact["name"].(string)
		if shortName == "" {
			return nil, errors.Errorf("contactPoints[%d].name must not be empty", index)
		}
		if _, exists := contactNames[shortName]; exists {
			return nil, errors.Errorf("contactPoints contains duplicate name %q", shortName)
		}
		contactNames[shortName] = alertingObjectName(bundleName, shortName)
	}

	for index, item := range contactPoints {
		contact := item.(map[string]any)
		shortName, _ := contact["name"].(string)
		objectName := contactNames[shortName]
		email, _ := contact["email"].(map[string]any)
		addresses, _ := email["addresses"].([]any)
		if len(addresses) == 0 {
			return nil, errors.Errorf("contactPoints[%d].email.addresses must contain at least one address", index)
		}
		parameters := map[string]any{"name": objectName, "disableProvenance": provenance == "createOnly", "email": []any{email}}
		desired[resource.Name("contact-point-"+shortName)] = newDesired("alerting.grafana.m.crossplane.io/v1alpha1", "ContactPoint", namespace, bundleName+"-contact-point-"+shortName,
			map[string]any{"crossplane.io/external-name": objectName}, alertingResourceSpec(parameters, provenance, stackName))
	}

	muteTimings, _ := spec["muteTimings"].([]any)
	muteNames := map[string]string{}
	for index, item := range muteTimings {
		mute, ok := item.(map[string]any)
		if !ok {
			return nil, errors.Errorf("muteTimings[%d] must be an object", index)
		}
		shortName, _ := mute["name"].(string)
		if shortName == "" {
			return nil, errors.Errorf("muteTimings[%d].name must not be empty", index)
		}
		if _, exists := muteNames[shortName]; exists {
			return nil, errors.Errorf("muteTimings contains duplicate name %q", shortName)
		}
		muteNames[shortName] = alertingObjectName(bundleName, shortName)
		parameters := map[string]any{"name": muteNames[shortName], "disableProvenance": provenance == "createOnly"}
		if intervals, exists := mute["intervals"]; exists {
			parameters["intervals"] = intervals
		}
		desired[resource.Name("mute-timing-"+shortName)] = newDesired("alerting.grafana.m.crossplane.io/v1alpha1", "MuteTiming", namespace, bundleName+"-mute-timing-"+shortName,
			map[string]any{"crossplane.io/external-name": muteNames[shortName]}, alertingResourceSpec(parameters, provenance, stackName))
	}

	messageTemplates, _ := spec["messageTemplates"].([]any)
	messageTemplateNames := map[string]struct{}{}
	for index, item := range messageTemplates {
		template, ok := item.(map[string]any)
		if !ok {
			return nil, errors.Errorf("messageTemplates[%d] must be an object", index)
		}
		shortName, _ := template["name"].(string)
		body, _ := template["template"].(string)
		if shortName == "" || body == "" {
			return nil, errors.Errorf("messageTemplates[%d] must set name and template", index)
		}
		if _, exists := messageTemplateNames[shortName]; exists {
			return nil, errors.Errorf("messageTemplates contains duplicate name %q", shortName)
		}
		messageTemplateNames[shortName] = struct{}{}
		objectName := alertingObjectName(bundleName, shortName)
		parameters := map[string]any{"name": objectName, "template": body, "disableProvenance": provenance == "createOnly"}
		desired[resource.Name("message-template-"+shortName)] = newDesired("alerting.grafana.m.crossplane.io/v1alpha1", "MessageTemplate", namespace, bundleName+"-message-template-"+shortName,
			map[string]any{"crossplane.io/external-name": objectName}, alertingResourceSpec(parameters, provenance, stackName))
	}

	inhibitionRules, _ := spec["inhibitionRules"].([]any)
	inhibitionRuleUIDs := map[string]struct{}{}
	for index, item := range inhibitionRules {
		rule, ok := item.(map[string]any)
		if !ok {
			return nil, errors.Errorf("inhibitionRules[%d] must be an object", index)
		}
		shortUID, _ := rule["uid"].(string)
		if shortUID == "" {
			return nil, errors.Errorf("inhibitionRules[%d].uid must not be empty", index)
		}
		if _, exists := inhibitionRuleUIDs[shortUID]; exists {
			return nil, errors.Errorf("inhibitionRules contains duplicate uid %q", shortUID)
		}
		inhibitionRuleUIDs[shortUID] = struct{}{}
		uid := alertingObjectName(bundleName, shortUID)
		parameters := map[string]any{
			"metadata": map[string]any{"uid": uid},
			"options":  map[string]any{"managerIdentity": "crossplane-" + bundleName},
			"spec": map[string]any{
				"sourceMatchers": rule["sourceMatchers"], "targetMatchers": rule["targetMatchers"], "equal": rule["equal"],
			},
		}
		desired[resource.Name("inhibition-rule-"+shortUID)] = newDesired("alerting.grafana.m.crossplane.io/v1alpha1", "InhibitionruleV1Beta1", namespace, bundleName+"-inhibition-rule-"+shortUID,
			map[string]any{"crossplane.io/external-name": uid}, alertingResourceSpec(parameters, provenance, stackName))
	}

	ruleGroups, _ := spec["ruleGroups"].([]any)
	ruleGroupNames := map[string]struct{}{}
	for index, item := range ruleGroups {
		group, ok := item.(map[string]any)
		if !ok {
			return nil, errors.Errorf("ruleGroups[%d] must be an object", index)
		}
		shortName, _ := group["name"].(string)
		folderUID, _ := group["folderUid"].(string)
		rules, _ := group["rules"].([]any)
		if shortName == "" || folderUID == "" || len(rules) == 0 {
			return nil, errors.Errorf("ruleGroups[%d] must set name, folderUid, and at least one rule", index)
		}
		if _, exists := ruleGroupNames[shortName]; exists {
			return nil, errors.Errorf("ruleGroups contains duplicate name %q", shortName)
		}
		ruleGroupNames[shortName] = struct{}{}
		objectName := alertingObjectName(bundleName, shortName)
		renderedRules, err := renderAlertingRules(rules, contactNames, muteNames)
		if err != nil {
			return nil, errors.Wrapf(err, "ruleGroups[%d]", index)
		}
		parameters := map[string]any{"name": objectName, "folderUid": folderUID, "rule": renderedRules, "disableProvenance": provenance == "createOnly"}
		if interval, exists := group["intervalSeconds"]; exists {
			parameters["intervalSeconds"] = interval
		}
		desired[resource.Name("rule-group-"+shortName)] = newDesired("alerting.grafana.m.crossplane.io/v1alpha1", "RuleGroup", namespace, bundleName+"-rule-group-"+shortName,
			map[string]any{"crossplane.io/external-name": fmt.Sprintf("%s:%s", folderUID, objectName)}, alertingResourceSpec(parameters, provenance, stackName))
	}

	return desired, nil
}

func alertingObjectName(bundleName, shortName string) string {
	return bundleName + "-" + shortName
}

func alertingResourceSpec(parameters map[string]any, provenance, stackName string) map[string]any {
	spec := map[string]any{"managementPolicies": managementPolicies, "providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": stackName}}
	if provenance == "createOnly" {
		spec["forProvider"] = map[string]any{}
		spec["initProvider"] = parameters
	} else {
		spec["forProvider"] = parameters
	}
	return spec
}

func renderAlertingRules(rules []any, contactNames, muteNames map[string]string) ([]any, error) {
	rendered := make([]any, 0, len(rules))
	for index, item := range rules {
		rule, ok := item.(map[string]any)
		if !ok {
			return nil, errors.Errorf("rules[%d] must be an object", index)
		}
		copy := map[string]any{}
		for key, value := range rule {
			copy[key] = value
		}
		settings, exists := rule["notificationSettings"].(map[string]any)
		if !exists {
			return nil, errors.Errorf("rules[%d].notificationSettings must be an object", index)
		}
		contactPoint, _ := settings["contactPoint"].(string)
		contactName, found := contactNames[contactPoint]
		if !found {
			return nil, errors.Errorf("rules[%d] references undeclared contact point %q", index, contactPoint)
		}
		routing := map[string]any{}
		for key, value := range settings {
			routing[key] = value
		}
		routing["contactPoint"] = contactName
		if timings, ok := settings["muteTimings"].([]any); ok {
			renderedTimings := make([]any, 0, len(timings))
			for _, item := range timings {
				shortName, _ := item.(string)
				name, found := muteNames[shortName]
				if !found {
					return nil, errors.Errorf("rules[%d] references undeclared mute timing %q", index, shortName)
				}
				renderedTimings = append(renderedTimings, name)
			}
			routing["muteTimings"] = renderedTimings
		}
		copy["notificationSettings"] = []any{routing}
		rendered = append(rendered, copy)
	}
	return rendered, nil
}
