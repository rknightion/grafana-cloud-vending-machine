package main

import "github.com/crossplane/function-sdk-go/resource"

const agentObservabilityRendererImplemented = false

func renderAgentObservability(map[string]any, map[resource.Name]resource.ObservedComposed, map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	return map[resource.Name]*resource.DesiredComposed{}, nil
}
