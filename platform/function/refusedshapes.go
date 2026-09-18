package main

import (
	"fmt"
	"strings"

	"github.com/crossplane/function-sdk-go/resource"
)

// refusedVendorShape is a vendor request form this function must never emit.
// Evidence is intentionally data rather than prose so the central renderer
// dispatch can reject a newly introduced form before it reaches a provider.
type refusedVendorShape struct {
	APIVersion string
	Kind       string
	FieldPath  []string
	Error      string
	Date       string
	Evidence   string
}

type managedGVK struct {
	APIVersion string
	Kind       string
}

func (g managedGVK) key() string { return g.APIVersion + "/" + g.Kind }

func (s refusedVendorShape) gvk() managedGVK {
	return managedGVK{APIVersion: s.APIVersion, Kind: s.Kind}
}

// refusedVendorShapes is the complete known list as of the stated evidence
// date. An "inferred" entry has not been exercised by this repository; it is
// prohibited because the sibling form was observed to receive the same error.
var refusedVendorShapes = []refusedVendorShape{
	{
		APIVersion: "oss.grafana.m.crossplane.io/v1alpha1",
		Kind:       "ConnectionV0Alpha1",
		FieldPath:  []string{"spec", "forProvider", "secure", "privateKey", "name"},
		Error:      "403 identity type access-policy not allowed",
		Date:       "2026-09-18",
		Evidence:   "observed",
	},
	{
		APIVersion: "oss.grafana.m.crossplane.io/v1alpha1",
		Kind:       "RepositoryV0Alpha1",
		FieldPath:  []string{"spec", "forProvider", "secure", "token", "name"},
		Error:      "403 identity type access-policy not allowed",
		Date:       "2026-09-18",
		Evidence:   "inferred",
	},
	{
		APIVersion: "oss.grafana.m.crossplane.io/v1alpha1",
		Kind:       "RepositoryV0Alpha1",
		FieldPath:  []string{"spec", "forProvider", "secure", "webhookSecret", "name"},
		Error:      "403 identity type access-policy not allowed",
		Date:       "2026-09-18",
		Evidence:   "inferred",
	},
	{
		APIVersion: "oss.grafana.m.crossplane.io/v1alpha1",
		Kind:       "RepositoryV0Alpha1",
		FieldPath:  []string{"spec", "forProvider", "secure", "commitSigningKey", "name"},
		Error:      "403 identity type access-policy not allowed",
		Date:       "2026-09-18",
		Evidence:   "inferred",
	},
}

// refusedVendorShapeError rejects a desired child whose GVK and populated field
// path match a known refused request form. Root wires this immediately after
// every registered renderer returns its complete desired set.
func refusedVendorShapeError(children map[resource.Name]*resource.DesiredComposed) error {
	for logicalName, child := range children {
		if child == nil || child.Resource == nil {
			continue
		}
		content := child.Resource.UnstructuredContent()
		apiVersion, _ := content["apiVersion"].(string)
		kind, _ := content["kind"].(string)
		for _, shape := range refusedVendorShapes {
			if shape.APIVersion != apiVersion || shape.Kind != kind || !refusedShapePathPresent(content, shape.FieldPath) {
				continue
			}
			return fmt.Errorf("refused vendor shape emitted by child %q (%s %s) at %s: vendor returns %q", logicalName, apiVersion, kind, strings.Join(shape.FieldPath, "."), shape.Error)
		}
	}
	return nil
}

func refusedShapePathPresent(value map[string]any, path []string) bool {
	var current any = value
	for _, segment := range path {
		object, ok := current.(map[string]any)
		if !ok {
			return false
		}
		current, ok = object[segment]
		if !ok {
			return false
		}
	}
	return current != nil
}
