package main

import (
	"strconv"
	"strings"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

const frontendObservabilityRendererImplemented = true

const frontendObservabilityAPIVersion = "frontendobservability.grafana.m.crossplane.io/v1alpha1"

// renderFrontendObservability creates only apps selected from a platform-owned
// profile. The provider exposes collectorEndpoint after creation. That URL
// contains the browser ingestion key by design, so it is intentionally neither
// accepted from the request nor copied into a Secret.
func renderFrontendObservability(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	metadata, _ := xr["metadata"].(map[string]any)
	spec, _ := xr["spec"].(map[string]any)
	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	stackRef, _ := spec["stackRef"].(map[string]any)
	stackName, _ := stackRef["name"].(string)
	profileName, _ := spec["profile"].(string)
	if name == "" || namespace == "" || stackName == "" || profileName == "" {
		return nil, errors.New("frontend observability must set metadata name and namespace, stackRef.name, and profile")
	}

	if name != stackName {
		return nil, errors.New("metadata.name must match spec.stackRef.name so one composite owns this stack surface")
	}
	stack, ready := referencedProductStack(config)
	if !ready {
		return map[resource.Name]*resource.DesiredComposed{}, nil
	}
	stackID, err := referencedProductStackID(stack)
	if err != nil {
		return nil, err
	}
	organizationProviderConfigName := stringValue(stack, "organizationProviderConfigName", "")
	if organizationProviderConfigName == "" {
		return nil, errors.New("trusted referenced stack context is incomplete")
	}
	profile, ok := configuredProfile(config, "frontendObservabilityProfiles", profileName)
	if !ok {
		return nil, errors.Errorf("frontend observability profile %q is not configured by the platform", profileName)
	}
	apps, _ := profile["apps"].([]any)
	if len(apps) == 0 {
		return nil, errors.Errorf("frontend observability profile %q must set at least one app", profileName)
	}

	desired := map[resource.Name]*resource.DesiredComposed{}
	for index, raw := range apps {
		app, ok := raw.(map[string]any)
		if !ok {
			return nil, errors.Errorf("frontend observability profile %q app %d must be an object", profileName, index)
		}
		appName := stringValue(app, "name", "")
		if appName == "" {
			return nil, errors.Errorf("frontend observability profile %q app %d must set name", profileName, index)
		}
		allowedOrigins, _ := app["allowedOrigins"].([]any)
		extraLogAttributes, _ := app["extraLogAttributes"].(map[string]any)
		settings, _ := app["settings"].(map[string]any)
		if len(allowedOrigins) == 0 || extraLogAttributes == nil || settings == nil {
			return nil, errors.Errorf("frontend observability profile %q app %q must set allowedOrigins, extraLogAttributes, and settings", profileName, appName)
		}
		resourceName := resource.Name("app-" + appName)
		if _, duplicate := desired[resourceName]; duplicate {
			return nil, errors.Errorf("frontend observability profile %q contains duplicate app name %q", profileName, appName)
		}
		desired[resourceName] = newDesired(
			frontendObservabilityAPIVersion,
			"App",
			namespace,
			name+"-"+appName,
			map[string]any{"crossplane.io/external-name": strconv.FormatInt(stackID, 10) + ":" + appName},
			map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider": map[string]any{
					"allowedOrigins":     allowedOrigins,
					"extraLogAttributes": extraLogAttributes,
					"name":               appName,
					"settings":           settings,
					"stackId":            float64(stackID),
				},
				"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": organizationProviderConfigName},
			},
		)
	}
	for name := range observed {
		if strings.HasPrefix(string(name), "app-") && desired[name] == nil {
			return nil, errors.Errorf("frontend observability profile would withdraw observed app %q; retain it until an explicit decommission", name)
		}
	}
	return desired, nil
}

func referencedProductStack(config map[string]any) (map[string]any, bool) {
	stack, ok := config["referencedStack"].(map[string]any)
	return stack, ok
}

func referencedProductStackID(stack map[string]any) (int64, error) {
	value, _ := stack["stackID"].(string)
	if value == "" {
		return 0, errors.New("trusted referenced stack context is incomplete")
	}
	stackID, err := strconv.ParseInt(value, 10, 64)
	if err != nil || stackID <= 0 {
		return 0, errors.Errorf("trusted referenced stack ID %q is not a positive integer", value)
	}
	return stackID, nil
}
