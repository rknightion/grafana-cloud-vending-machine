package main

import (
	"fmt"
	"github.com/crossplane/function-sdk-go/resource"
)

const mlRendererImplemented = false

func renderML(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	return nil, fmt.Errorf("GrafanaML renderer is not implemented")
}
