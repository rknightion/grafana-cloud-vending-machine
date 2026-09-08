package main

import (
	"fmt"
	"github.com/crossplane/function-sdk-go/resource"
)

const pdcRendererImplemented = false

func renderPDC(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	return nil, fmt.Errorf("GrafanaPDC renderer is not implemented")
}
