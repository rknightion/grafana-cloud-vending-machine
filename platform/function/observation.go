package main

import (
	"encoding/json"
	"fmt"
	"github.com/crossplane/function-sdk-go/resource"
)

// observedDesiredCurrent requires the provider to have reconciled this generation
// and all fields the function owns. API defaults and provider-added fields may remain.
func observedDesiredCurrent(observed resource.ObservedComposed, desired *resource.DesiredComposed) bool {
	if observed.Resource == nil || desired == nil || desired.Resource == nil {
		return false
	}
	object := observed.Resource.UnstructuredContent()
	want := desired.Resource.UnstructuredContent()
	meta, _ := object["metadata"].(map[string]any)
	wm, _ := want["metadata"].(map[string]any)
	if object["apiVersion"] != want["apiVersion"] || object["kind"] != want["kind"] || meta["name"] != wm["name"] || meta["namespace"] != wm["namespace"] || meta["deletionTimestamp"] != nil {
		return false
	}
	status, _ := object["status"].(map[string]any)
	generation := fmt.Sprint(meta["generation"])
	if generation == "<nil>" || generation == "0" || !requiredStackReady(object) {
		return false
	}
	// crossplane-runtime v2.1.0 propagates generations to the Synced
	// condition; the provider does not populate top-level observedGeneration.
	conditions, _ := status["conditions"].([]any)
	synced := false
	for _, raw := range conditions {
		condition, _ := raw.(map[string]any)
		switch condition["type"] {
		case "Synced":
			synced = condition["status"] == "True" && fmt.Sprint(condition["observedGeneration"]) == generation
		case "AsyncOperation", "LastAsyncOperation":
			if condition["status"] != "True" {
				return false
			}
		}
	}
	return synced && containsDesiredFields(object["spec"], want["spec"])

}

func containsDesiredFields(got, want any) bool {
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			return false
		}
		for key, value := range w {
			if !containsDesiredFields(g[key], value) {
				return false
			}
		}
		return true
	case []any:
		g, ok := got.([]any)
		if !ok || len(g) != len(w) {
			return false
		}
		for i := range w {
			if !containsDesiredFields(g[i], w[i]) {
				return false
			}
		}
		return true
	default:
		a, _ := json.Marshal(got)
		b, _ := json.Marshal(want)
		return string(a) == string(b)
	}
}

// observedDesiredReady is the keyed form used by staged product renderers.
func observedDesiredReady(observed map[resource.Name]resource.ObservedComposed, name resource.Name, desired *resource.DesiredComposed) bool {
	return observedDesiredCurrent(observed[name], desired)
}
