package main

import (
	"fmt"
	"github.com/crossplane/function-sdk-go/resource"
)

const cloudIntegrationsRendererImplemented = false

func renderCloudIntegrations(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	return nil, fmt.Errorf("GrafanaCloudIntegrations renderer is not implemented")
}
