package main

import (
	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

func addPluginInstallations(desired map[resource.Name]*resource.DesiredComposed, namespace, slug string, spec map[string]any, organizationProviderConfigName string) error {
	plugins, _ := spec["plugins"].([]any)
	for _, item := range plugins {
		plugin, _ := item.(map[string]any)
		pluginSlug, _ := plugin["slug"].(string)
		if pluginSlug == "" {
			return errors.New("each plugin must set slug")
		}
		version := stringValue(plugin, "version", "latest")
		name := "plugin-" + pluginSlug
		desired[resource.Name(name)] = newDesired(
			"cloud.grafana.m.crossplane.io/v1alpha1",
			"PluginInstallation",
			namespace,
			slug+"-"+pluginSlug,
			nil,
			map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider": map[string]any{
					"cloudStackRef": map[string]any{"name": slug},
					"slug":          pluginSlug,
					"version":       version,
				},
				"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": organizationProviderConfigName},
			},
		)
	}
	return nil
}
