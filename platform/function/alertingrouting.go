package main

import (
	"fmt"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

const alertingRoutingRendererImplemented = true

// renderAlertingRouting owns one complete NotificationPolicy tree per stack. The
// provider replaces every policy node it manages, so omitted nodes are removed.
// Existing direct rule-level routes are a separate Grafana mechanism and are
// not claimed here. The ordinary rule groups emitted below omit that bypass and
// therefore use this policy tree.
func renderAlertingRouting(xr map[string]any, _ map[resource.Name]resource.ObservedComposed, _ map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	metadata, _ := xr["metadata"].(map[string]any)
	spec, _ := xr["spec"].(map[string]any)
	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	stackRef, _ := spec["stackRef"].(map[string]any)
	stackName, _ := stackRef["name"].(string)
	contactPoints, _ := spec["contactPoints"].([]any)
	defaultContactPoint, _ := spec["defaultContactPoint"].(string)
	if name == "" || namespace == "" || stackName == "" || len(contactPoints) == 0 || defaultContactPoint == "" {
		return nil, errors.New("alerting routing must set metadata name and namespace, stackRef.name, contactPoints, and defaultContactPoint")
	}

	contactNames := map[string]string{}
	for index, item := range contactPoints {
		contactPoint, ok := item.(map[string]any)
		if !ok {
			return nil, errors.Errorf("contactPoints[%d] must be an object", index)
		}
		shortName, _ := contactPoint["name"].(string)
		email, _ := contactPoint["email"].(map[string]any)
		addresses, _ := email["addresses"].([]any)
		if shortName == "" || len(addresses) == 0 {
			return nil, errors.Errorf("contactPoints[%d] must set name and email.addresses", index)
		}
		if _, exists := contactNames[shortName]; exists {
			return nil, errors.Errorf("contactPoints contains duplicate name %q", shortName)
		}
		contactNames[shortName] = name + "-routing-" + shortName
	}
	if _, exists := contactNames[defaultContactPoint]; !exists {
		return nil, errors.Errorf("defaultContactPoint %q is not declared in contactPoints", defaultContactPoint)
	}

	desired := map[resource.Name]*resource.DesiredComposed{}
	for _, item := range contactPoints {
		contactPoint := item.(map[string]any)
		shortName := contactPoint["name"].(string)
		objectName := contactNames[shortName]
		desired[resource.Name("contact-point-"+shortName)] = newDesired("alerting.grafana.m.crossplane.io/v1alpha1", "ContactPoint", namespace, name+"-contact-point-"+shortName,
			map[string]any{"crossplane.io/external-name": objectName}, map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider": map[string]any{
					"name":  objectName,
					"email": []any{contactPoint["email"]},
				},
				"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": stackName},
			})
	}

	routes, _ := spec["routes"].([]any)
	policies := make([]any, 0, len(routes))
	seenRoutes := map[string]struct{}{}
	for index, item := range routes {
		route, ok := item.(map[string]any)
		if !ok {
			return nil, errors.Errorf("routes[%d] must be an object", index)
		}
		routeName, _ := route["name"].(string)
		contactPoint, _ := route["contactPoint"].(string)
		matchers, _ := route["matchers"].([]any)
		if routeName == "" || contactPoint == "" || len(matchers) == 0 {
			return nil, errors.Errorf("routes[%d] must set name, contactPoint, and matchers", index)
		}
		if _, exists := seenRoutes[routeName]; exists {
			return nil, errors.Errorf("routes contains duplicate name %q", routeName)
		}
		seenRoutes[routeName] = struct{}{}
		contactName, exists := contactNames[contactPoint]
		if !exists {
			return nil, errors.Errorf("routes[%d] references undeclared contact point %q", index, contactPoint)
		}
		policy := map[string]any{"contactPoint": contactName, "matcher": matchers}
		copyOptionalFields(policy, route, "continue", "groupBy", "groupWait", "groupInterval", "repeatInterval")
		policies = append(policies, policy)
	}

	ruleGroups, _ := spec["ruleGroups"].([]any)
	seenRuleGroups := map[string]struct{}{}
	for index, item := range ruleGroups {
		group, ok := item.(map[string]any)
		if !ok {
			return nil, errors.Errorf("ruleGroups[%d] must be an object", index)
		}
		shortName, _ := group["name"].(string)
		folderUID, _ := group["folderUid"].(string)
		intervalSeconds, hasIntervalSeconds := group["intervalSeconds"]
		rules, _ := group["rules"].([]any)
		if shortName == "" || folderUID == "" || !hasIntervalSeconds || len(rules) == 0 {
			return nil, errors.Errorf("ruleGroups[%d] must set name, folderUid, intervalSeconds, and at least one rule", index)
		}
		if _, exists := seenRuleGroups[shortName]; exists {
			return nil, errors.Errorf("ruleGroups contains duplicate name %q", shortName)
		}
		seenRuleGroups[shortName] = struct{}{}
		renderedRules, err := renderRoutingRules(rules)
		if err != nil {
			return nil, errors.Wrapf(err, "ruleGroups[%d]", index)
		}
		objectName := name + "-routing-" + shortName
		parameters := map[string]any{
			"name":              objectName,
			"folderUid":         folderUID,
			"intervalSeconds":   intervalSeconds,
			"rule":              renderedRules,
			"disableProvenance": false,
		}
		desired[resource.Name("rule-group-"+shortName)] = newDesired("alerting.grafana.m.crossplane.io/v1alpha1", "RuleGroup", namespace, name+"-rule-group-"+shortName,
			map[string]any{"crossplane.io/external-name": fmt.Sprintf("%s:%s", folderUID, objectName)}, map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider":        parameters,
				"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": stackName},
			})
	}

	// NotificationPolicy has a fixed singleton identity in each ProviderConfig.
	// Its policy list is authoritative: any provider-managed node absent from
	// spec.routes is removed on reconciliation.
	desired["notification-policy"] = newDesired("alerting.grafana.m.crossplane.io/v1alpha1", "NotificationPolicy", namespace, name+"-notification-policy",
		map[string]any{"crossplane.io/external-name": "notification-policy"}, map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider": map[string]any{
				"contactPoint": contactNames[defaultContactPoint],
				"groupBy":      []any{"alertname", "grafana_folder"},
				"policy":       policies,
			},
			"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": stackName},
		})
	return desired, nil
}

func renderRoutingRules(rules []any) ([]any, error) {
	rendered := make([]any, 0, len(rules))
	for index, item := range rules {
		rule, ok := item.(map[string]any)
		if !ok {
			return nil, errors.Errorf("rules[%d] must be an object", index)
		}
		if _, bypass := rule["notificationSettings"]; bypass {
			return nil, errors.Errorf("rules[%d] must not set notificationSettings; GrafanaAlertingRouting uses its NotificationPolicy tree", index)
		}
		name, _ := rule["name"].(string)
		condition, _ := rule["condition"].(string)
		data, _ := rule["data"].([]any)
		if name == "" || condition == "" || len(data) == 0 {
			return nil, errors.Errorf("rules[%d] must set name, condition, and data", index)
		}
		copy := map[string]any{}
		for key, value := range rule {
			copy[key] = value
		}
		rendered = append(rendered, copy)
	}
	return rendered, nil
}
