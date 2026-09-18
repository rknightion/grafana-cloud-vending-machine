package main

import (
	"testing"

	"github.com/crossplane/function-sdk-go/resource"
)

func TestPluginInstallationsAdoptObservedVersionForNewestRequests(t *testing.T) {
	for _, tc := range []struct {
		name     string
		plugin   map[string]any
		observed string
		want     string
	}{
		{
			name:     "omitted version",
			plugin:   map[string]any{"slug": "grafana-piechart-panel"},
			observed: "2.1.3",
			want:     "2.1.3",
		},
		{
			name:     "explicit newest version",
			plugin:   map[string]any{"slug": "grafana-piechart-panel", "version": "latest"},
			observed: "2.1.3",
			want:     "2.1.3",
		},
		{
			name:     "explicit pinned version",
			plugin:   map[string]any{"slug": "grafana-piechart-panel", "version": "2.0.0"},
			observed: "2.1.3",
			want:     "2.0.0",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			desired := map[resource.Name]*resource.DesiredComposed{}
			observed := map[resource.Name]resource.ObservedComposed{
				"plugin-grafana-piechart-panel": observedComposed(`{"status":{"atProvider":{"version":"` + tc.observed + `"}}}`),
			}
			if err := addPluginInstallations(desired, "default", "stack", map[string]any{"plugins": []any{tc.plugin}}, "provider", observed); err != nil {
				t.Fatal(err)
			}
			got := nestedMap(t, pluginInstallationDesired(t, desired), "spec", "forProvider")["version"]
			if got != tc.want {
				t.Fatalf("rendered plugin version = %v, want %q", got, tc.want)
			}
		})
	}
}

func pluginInstallationDesired(t *testing.T, desired map[resource.Name]*resource.DesiredComposed) map[string]any {
	t.Helper()
	plugin, ok := desired["plugin-grafana-piechart-panel"]
	if !ok {
		t.Fatal("plugin installation was not rendered")
	}
	return plugin.Resource.UnstructuredContent()
}
