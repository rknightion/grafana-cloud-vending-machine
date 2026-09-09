package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

const assertsRendererImplemented = true

const (
	assertsAPIVersion = "asserts.grafana.m.crossplane.io/v1alpha1"
	assertsCapMessage = "selected asserts profile exceeds a platform-owned resource or list cap"
)

type assertsNamedKind struct {
	field, logicalPrefix, kind, limit string
	required                          []string
	allowed                           []string
}

var assertsNamedKinds = []assertsNamedKind{
	{field: "customModelRules", logicalPrefix: "custom-model-rules", kind: "CustomModelRules", limit: "maxCustomModelRules", required: []string{"name", "rules"}, allowed: []string{"name", "rules"}},
	{field: "logConfigs", logicalPrefix: "log-config", kind: "LogConfig", limit: "maxLogConfigs", required: []string{"dataSourceUid", "defaultConfig", "name", "priority"}, allowed: []string{"dataSourceUid", "defaultConfig", "entityPropertyToLogLabelMapping", "errorLabel", "filterBySpanId", "filterByTraceId", "match", "name", "priority"}},
	{field: "notificationAlertsConfigs", logicalPrefix: "notification-alerts-config", kind: "NotificationAlertsConfig", limit: "maxNotificationAlertsConfigs", required: []string{"name"}, allowed: []string{"alertLabels", "duration", "matchLabels", "name", "silenced"}},
	{field: "profileConfigs", logicalPrefix: "profile-config", kind: "ProfileConfig", limit: "maxProfileConfigs", required: []string{"dataSourceUid", "defaultConfig", "name", "priority"}, allowed: []string{"dataSourceUid", "defaultConfig", "entityPropertyToProfileLabelMapping", "match", "name", "priority"}},
	{field: "promRuleFiles", logicalPrefix: "prom-rule-file", kind: "PromRuleFile", limit: "maxPromRuleFiles", required: []string{"group", "name"}, allowed: []string{"active", "group", "name"}},
	{field: "suppressedAssertionsConfigs", logicalPrefix: "suppressed-assertions-config", kind: "SuppressedAssertionsConfig", limit: "maxSuppressedAssertionsConfigs", required: []string{"name"}, allowed: []string{"matchLabels", "name"}},
	{field: "traceConfigs", logicalPrefix: "trace-config", kind: "TraceConfig", limit: "maxTraceConfigs", required: []string{"dataSourceUid", "defaultConfig", "name", "priority"}, allowed: []string{"dataSourceUid", "defaultConfig", "entityPropertyToTraceLabelMapping", "match", "name", "priority"}},
}

var assertsNestedCaps = []struct {
	field string
	path  []string
	limit string
}{
	{"customModelRules", []string{"rules"}, "maxRulesPerCustomModelRules"},
	{"customModelRules", []string{"rules", "entity"}, "maxEntitiesPerRule"},
	{"customModelRules", []string{"rules", "entity", "definedBy"}, "maxDefinedByPerEntity"},
	{"customModelRules", []string{"rules", "entity", "enrichedBy"}, "maxEnrichedByPerEntity"},
	{"logConfigs", []string{"match"}, "maxMatchesPerConfig"},
	{"logConfigs", []string{"match", "values"}, "maxMatchValuesPerMatch"},
	{"profileConfigs", []string{"match"}, "maxMatchesPerConfig"},
	{"profileConfigs", []string{"match", "values"}, "maxMatchValuesPerMatch"},
	{"traceConfigs", []string{"match"}, "maxMatchesPerConfig"},
	{"traceConfigs", []string{"match", "values"}, "maxMatchValuesPerMatch"},
	{"promRuleFiles", []string{"group"}, "maxPromGroupsPerFile"},
	{"promRuleFiles", []string{"group", "rule"}, "maxPromRulesPerGroup"},
	{"promRuleFiles", []string{"group", "rule", "disableInGroups"}, "maxDisableInGroupsPerRule"},
	{"stack", []string{"dataset"}, "maxDatasets"},
	{"stack", []string{"dataset", "disabledVendors"}, "maxDisabledVendorsPerDataset"},
	{"stack", []string{"dataset", "filterGroup"}, "maxFilterGroupsPerDataset"},
	{"stack", []string{"dataset", "filterGroup", "envLabelValues"}, "maxEnvLabelValuesPerFilterGroup"},
	{"stack", []string{"dataset", "filterGroup", "filter"}, "maxFiltersPerFilterGroup"},
	{"stack", []string{"dataset", "filterGroup", "filter", "values"}, "maxFilterValuesPerFilter"},
	{"stack", []string{"dataset", "filterGroup", "siteLabelValues"}, "maxSiteLabelValuesPerFilterGroup"},
	{"thresholds", []string{"healthThresholds"}, "maxHealthThresholds"},
	{"thresholds", []string{"requestThresholds"}, "maxRequestThresholds"},
	{"thresholds", []string{"resourceThresholds"}, "maxResourceThresholds"},
}

// renderAsserts renders the whole namespaced Asserts surface from exactly one
// platform profile. Profile values never originate in the public request.
func renderAsserts(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	metadata, _ := xr["metadata"].(map[string]any)
	spec, _ := xr["spec"].(map[string]any)
	name := stringValue(metadata, "name", "")
	namespace := stringValue(metadata, "namespace", "")
	stackRef, _ := spec["stackRef"].(map[string]any)
	stackName := stringValue(stackRef, "name", "")
	profileName := stringValue(spec, "profile", "")
	if name == "" || namespace == "" || stackName == "" || profileName == "" {
		return nil, errors.New("asserts must set metadata name and namespace, stackRef.name, and profile")
	}
	if name != stackName {
		return nil, errors.New("metadata.name must match spec.stackRef.name so one composite owns this stack surface")
	}
	stack, ready := referencedProductStack(config)
	if !ready {
		if len(observed) > 0 {
			return nil, errors.New("waiting for observed stack identity; preserving existing composed resources")
		}
		return map[resource.Name]*resource.DesiredComposed{}, nil
	}
	stackID, err := referencedProductStackID(stack)
	if err != nil {
		return nil, err
	}
	providerConfigName := stringValue(stack, "providerConfigName", "")
	if providerConfigName == "" || providerConfigName != stackName {
		return nil, errors.New("trusted referenced stack context is incomplete")
	}
	profile, err := configuredAssertsProfile(config, profileName)
	if err != nil {
		return nil, err
	}
	limits, err := assertsProfileLimits(profile)
	if err != nil {
		return nil, err
	}
	if err := validateAssertsProfile(profile, limits); err != nil {
		return nil, err
	}

	desired := map[resource.Name]*resource.DesiredComposed{}
	providerConfigRef := map[string]any{"kind": "ProviderConfig", "name": providerConfigName}
	for _, descriptor := range assertsNamedKinds {
		items, _ := profile[descriptor.field].([]any)
		for _, raw := range items {
			item := raw.(map[string]any)
			itemName := stringValue(item, "name", "")
			logicalName := assertsNamedResourceName(descriptor.logicalPrefix, itemName)
			if _, duplicate := desired[logicalName]; duplicate {
				return nil, errors.Errorf("asserts profile contains duplicate %s name %q", descriptor.kind, itemName)
			}
			child, err := newAssertsNamedDesired(descriptor, namespace, name+"-"+string(logicalName), itemName, item, providerConfigRef)
			if err != nil {
				return nil, err
			}
			desired[logicalName] = child
		}
	}
	if raw, exists := profile["stack"]; exists {
		item := raw.(map[string]any)
		desired["stack"] = newDesired(assertsAPIVersion, "Stack", namespace, name+"-stack", map[string]any{"crossplane.io/external-name": fmt.Sprintf("%d", stackID)}, map[string]any{
			"managementPolicies": managementPolicies, "forProvider": selectAssertsFields(item, []string{"cloudAccessPolicyTokenSecretRef", "dataset", "grafanaTokenSecretRef"}), "providerConfigRef": providerConfigRef,
		})
	}
	if raw, exists := profile["thresholds"]; exists {
		item := raw.(map[string]any)
		desired["thresholds"] = newDesired(assertsAPIVersion, "Thresholds", namespace, name+"-thresholds", map[string]any{"crossplane.io/external-name": "custom_thresholds"}, map[string]any{
			"managementPolicies": managementPolicies, "forProvider": selectAssertsFields(item, []string{"healthThresholds", "requestThresholds", "resourceThresholds"}), "providerConfigRef": providerConfigRef,
		})
	}
	for child := range observed {
		if desired[child] == nil {
			return nil, errors.Errorf("asserts profile or prerequisite transition would withdraw existing child %q; retain it until an explicit decommission", child)
		}
	}
	return desired, nil
}

// assertsNamedResourceName separates Kubernetes identity from the provider's
// exact configuration name, which may contain non-DNS characters.
func assertsNamedResourceName(prefix, providerName string) resource.Name {
	digest := sha256.Sum256([]byte(providerName))
	return resource.Name(prefix + "-" + hex.EncodeToString(digest[:]))
}

func configuredAssertsProfile(config map[string]any, name string) (map[string]any, error) {
	spec, _ := config["spec"].(map[string]any)
	profiles, _ := spec["assertsProfiles"].([]any)
	var selected map[string]any
	for _, raw := range profiles {
		profile, ok := raw.(map[string]any)
		if !ok || stringValue(profile, "name", "") != name {
			continue
		}
		if selected != nil {
			return nil, errors.Errorf("asserts profile %q is configured more than once", name)
		}
		selected = profile
	}
	if selected == nil {
		return nil, errors.Errorf("asserts profile %q is not configured by the platform", name)
	}
	return selected, nil
}

// newAssertsNamedDesired keeps every provider GVK literal in source. The
// provider-family census deliberately rejects an unbounded dynamic kind loop.
func newAssertsNamedDesired(descriptor assertsNamedKind, namespace, name, externalName string, item, providerConfigRef map[string]any) (*resource.DesiredComposed, error) {
	spec := map[string]any{"managementPolicies": managementPolicies, "forProvider": selectAssertsFields(item, descriptor.allowed), "providerConfigRef": providerConfigRef}
	annotations := map[string]any{"crossplane.io/external-name": externalName}
	switch descriptor.kind {
	case "CustomModelRules":
		return newDesired(assertsAPIVersion, "CustomModelRules", namespace, name, annotations, spec), nil
	case "LogConfig":
		return newDesired(assertsAPIVersion, "LogConfig", namespace, name, annotations, spec), nil
	case "NotificationAlertsConfig":
		return newDesired(assertsAPIVersion, "NotificationAlertsConfig", namespace, name, annotations, spec), nil
	case "ProfileConfig":
		return newDesired(assertsAPIVersion, "ProfileConfig", namespace, name, annotations, spec), nil
	case "PromRuleFile":
		return newDesired(assertsAPIVersion, "PromRuleFile", namespace, name, annotations, spec), nil
	case "SuppressedAssertionsConfig":
		return newDesired(assertsAPIVersion, "SuppressedAssertionsConfig", namespace, name, annotations, spec), nil
	case "TraceConfig":
		return newDesired(assertsAPIVersion, "TraceConfig", namespace, name, annotations, spec), nil
	default:
		return nil, errors.Errorf("unsupported asserts provider kind %q", descriptor.kind)
	}
}

func assertsProfileLimits(profile map[string]any) (map[string]int, error) {
	raw, ok := profile["limits"].(map[string]any)
	if !ok {
		return nil, errors.New("asserts profile must set limits")
	}
	keys := []string{"maxManagedResources", "maxCustomModelRules", "maxLogConfigs", "maxNotificationAlertsConfigs", "maxProfileConfigs", "maxPromRuleFiles", "maxSuppressedAssertionsConfigs", "maxTraceConfigs", "maxRulesPerCustomModelRules", "maxEntitiesPerRule", "maxDefinedByPerEntity", "maxEnrichedByPerEntity", "maxMatchesPerConfig", "maxMatchValuesPerMatch", "maxPromGroupsPerFile", "maxPromRulesPerGroup", "maxDisableInGroupsPerRule", "maxDatasets", "maxDisabledVendorsPerDataset", "maxFilterGroupsPerDataset", "maxEnvLabelValuesPerFilterGroup", "maxFiltersPerFilterGroup", "maxFilterValuesPerFilter", "maxSiteLabelValuesPerFilterGroup", "maxHealthThresholds", "maxRequestThresholds", "maxResourceThresholds"}
	result := make(map[string]int, len(keys))
	for _, key := range keys {
		value, ok := raw[key].(float64)
		if !ok || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value != math.Trunc(value) || value > float64(math.MaxInt) {
			return nil, errors.Errorf("asserts profile limits.%s must be a finite non-negative integer", key)
		}
		result[key] = int(value)
	}
	if result["maxRulesPerCustomModelRules"] > 1 {
		return nil, errors.New("asserts profile limits.maxRulesPerCustomModelRules must be at most 1")
	}
	return result, nil
}

func validateAssertsProfile(profile map[string]any, limits map[string]int) error {
	total := 0
	for _, descriptor := range assertsNamedKinds {
		raw, exists := profile[descriptor.field]
		if !exists {
			return errors.Errorf("asserts profile %s must be an array", descriptor.field)
		}
		items, ok := raw.([]any)
		if !ok {
			return errors.Errorf("asserts profile %s must be an array", descriptor.field)
		}
		if len(items) > limits[descriptor.limit] {
			return errors.New(assertsCapMessage)
		}
		total += len(items)
		seen := map[string]struct{}{}
		for _, rawItem := range items {
			item, ok := rawItem.(map[string]any)
			if !ok {
				return errors.Errorf("asserts profile %s entries must be objects", descriptor.field)
			}
			itemName := stringValue(item, "name", "")
			if itemName == "" {
				return errors.Errorf("%s must set name", descriptor.kind)
			}
			if _, duplicate := seen[itemName]; duplicate {
				return errors.Errorf("asserts profile contains duplicate %s name %q", descriptor.kind, itemName)
			}
			seen[itemName] = struct{}{}
			if err := validateAssertsRequired(descriptor.kind, itemName, item, descriptor.required); err != nil {
				return err
			}
			if descriptor.kind == "CustomModelRules" && len(item["rules"].([]any)) != 1 {
				return errors.Errorf("CustomModelRules %q must set exactly one rules block", itemName)
			}
		}
	}
	if raw, exists := profile["stack"]; exists {
		item, ok := raw.(map[string]any)
		if !ok {
			return errors.New("asserts profile stack must be an object")
		}
		if datasets, ok := item["dataset"].([]any); !ok || len(datasets) == 0 {
			return errors.New("Stack must set a non-empty dataset list; automatic discovery is not bounded")
		}
		if err := validatesAssertsSecretRef(item, "cloudAccessPolicyTokenSecretRef"); err != nil {
			return err
		}
		if _, present := item["grafanaTokenSecretRef"]; present {
			if err := validatesAssertsSecretRef(item, "grafanaTokenSecretRef"); err != nil {
				return err
			}
		}
		total++
	}
	if raw, exists := profile["thresholds"]; exists {
		if _, ok := raw.(map[string]any); !ok {
			return errors.New("asserts profile thresholds must be an object")
		}
		total++
	}
	if total > limits["maxManagedResources"] {
		return errors.New(assertsCapMessage)
	}
	for _, cap := range assertsNestedCaps {
		raw, exists := profile[cap.field]
		if !exists {
			continue
		}
		if err := validateAssertsNestedCap(raw, cap.path, limits[cap.limit]); err != nil {
			return err
		}
	}
	return nil
}

func validateAssertsRequired(kind, name string, item map[string]any, required []string) error {
	for _, field := range required {
		value, found := item[field]
		if !found || value == nil {
			return errors.Errorf("%s %q must set %s", kind, name, field)
		}
		switch field {
		case "name", "dataSourceUid":
			if _, ok := value.(string); !ok || value == "" {
				return errors.Errorf("%s %q must set %s", kind, name, field)
			}
		case "defaultConfig":
			if _, ok := value.(bool); !ok {
				return errors.Errorf("%s %q must set %s", kind, name, field)
			}
		case "priority":
			if _, ok := value.(float64); !ok {
				return errors.Errorf("%s %q must set %s", kind, name, field)
			}
		case "rules", "group":
			values, ok := value.([]any)
			if !ok || len(values) == 0 {
				return errors.Errorf("%s %q must set %s", kind, name, field)
			}
		}
	}
	return nil
}

func validatesAssertsSecretRef(item map[string]any, field string) error {
	ref, _ := item[field].(map[string]any)
	if stringValue(ref, "name", "") == "" || stringValue(ref, "key", "") == "" {
		return errors.Errorf("Stack must set %s.name and key", field)
	}
	return nil
}

func validateAssertsNestedCap(raw any, path []string, cap int) error {
	items, ok := raw.([]any)
	if !ok {
		if item, object := raw.(map[string]any); object {
			items = []any{item}
		} else {
			return errors.New("asserts profile nested values must be arrays")
		}
	}
	return validateAssertsNestedItems(items, path, cap)
}

func validateAssertsNestedItems(items []any, path []string, cap int) error {
	if len(path) == 0 {
		return nil
	}
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			return errors.New("asserts profile nested entries must be objects")
		}
		value, exists := item[path[0]]
		if !exists {
			continue
		}
		children, ok := value.([]any)
		if !ok {
			return errors.New("asserts profile nested values must be arrays")
		}
		if len(path) == 1 && len(children) > cap {
			return errors.New(assertsCapMessage)
		}
		if err := validateAssertsNestedItems(children, path[1:], cap); err != nil {
			return err
		}
	}
	return nil
}

func selectAssertsFields(input map[string]any, allowed []string) map[string]any {
	result := map[string]any{}
	for _, field := range allowed {
		if value, found := input[field]; found {
			result[field] = value
		}
	}
	return result
}
