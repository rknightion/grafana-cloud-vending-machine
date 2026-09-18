package main

import (
	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

// addPluginInstallations preserves an observed concrete version for newest-version
// requests. The provider makes version ForceNew, so an existing resource stuck with
// a latest-to-concrete diff cannot repair itself; delete and recreate it.
func addPluginInstallations(desired map[resource.Name]*resource.DesiredComposed, namespace, slug string, spec map[string]any, organizationProviderConfigName string, observed map[resource.Name]resource.ObservedComposed) error {
	plugins, _ := spec["plugins"].([]any)
	for _, item := range plugins {
		plugin, _ := item.(map[string]any)
		pluginSlug, _ := plugin["slug"].(string)
		if pluginSlug == "" {
			return errors.New("each plugin must set slug")
		}
		name := "plugin-" + pluginSlug
		version := stringValue(plugin, "version", "latest")
		if version == "latest" {
			if installed := observedString(observed, resource.Name(name), "status.atProvider.version"); installed != "" && installed != "latest" {
				version = installed
			}
		}
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
