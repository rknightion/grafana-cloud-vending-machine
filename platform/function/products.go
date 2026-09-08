package main

import "github.com/crossplane/function-sdk-go/resource"

func addObservabilityProducts(desired map[resource.Name]*resource.DesiredComposed, namespace, providerConfig string, spec map[string]any) error {
	products, _ := spec["products"].(map[string]any)
	for _, product := range []struct {
		field, resourceName, kind string
	}{
		{field: "applicationObservability", resourceName: "application-observability", kind: "Appo11YconfigV1Alpha1"},
		{field: "kubernetesObservability", resourceName: "kubernetes-observability", kind: "K8So11YconfigV1Alpha1"},
		{field: "databaseObservability", resourceName: "database-observability", kind: "Dbo11YconfigV1Alpha1"},
	} {
		if products[product.field] != true {
			continue
		}
		desired[resource.Name(product.resourceName)] = newDesired(
			"cloud.grafana.m.crossplane.io/v1alpha1",
			product.kind,
			namespace,
			providerConfig+"-"+product.resourceName,
			map[string]any{"crossplane.io/external-name": "global"},
			map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider": map[string]any{
					"metadata": map[string]any{"uid": "global"},
					"spec":     map[string]any{"enabled": true},
				},
				"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": providerConfig},
			},
		)
	}
	return nil
}
