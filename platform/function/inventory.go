package main

import "github.com/crossplane/function-sdk-go/resource"

const inventoryRendererImplemented = false

func renderStackInventory(map[string]any, map[resource.Name]resource.ObservedComposed, map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	return map[resource.Name]*resource.DesiredComposed{}, nil
}
