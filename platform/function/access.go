package main

import (
	"fmt"

	"github.com/crossplane/function-sdk-go/resource"
)

func addTelemetryAccess(
	desired map[resource.Name]*resource.DesiredComposed,
	observed map[resource.Name]resource.ObservedComposed,
	namespace, slug, region, outputPath string,
	spec map[string]any,
	settings platformSettings,
	deletingExternalResources bool,
) error {
	if !telemetryAccessEnabled(spec) {
		return nil
	}

	policyName := slug + "-telemetry-publisher"
	tokenSecret := slug + "-telemetry-token"
	externalResourcePolicies := managementPolicies
	deleteOnDestroy := false
	pushSecretDeletionPolicy := "None"
	if deletingExternalResources {
		externalResourcePolicies = []any{"*"}
		deleteOnDestroy = true
		pushSecretDeletionPolicy = "Delete"
	}
	desired["telemetry-access-policy"] = newDesired(
		"cloud.grafana.m.crossplane.io/v1alpha1",
		"AccessPolicy",
		namespace,
		policyName,
		nil,
		map[string]any{
			"managementPolicies": externalResourcePolicies,
			"forProvider": map[string]any{
				"displayName": "Telemetry publisher for " + slug,
				"name":        policyName,
				"realm": []any{map[string]any{
					"stackRef": map[string]any{"name": slug},
					"type":     "stack",
				}},
				"region": region,
				"scopes": []any{"stacks:read", "metrics:write", "logs:write", "traces:write"},
			},
			"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": settings.organizationProviderConfigName},
		},
	)

	policyID := observedString(observed, "telemetry-access-policy", "status.atProvider.policyId")
	if policyID == "" {
		policyID = observedString(observed, "telemetry-token", "spec.forProvider.accessPolicyId")
	}
	if policyID != "" {
		desired["telemetry-token"] = newDesired(
			"cloud.grafana.m.crossplane.io/v1alpha1",
			"AccessPolicyRotatingToken",
			namespace,
			policyName,
			nil,
			map[string]any{
				"managementPolicies": externalResourcePolicies,
				"forProvider": map[string]any{
					"accessPolicyId":      policyID,
					"deleteOnDestroy":     deleteOnDestroy,
					"displayName":         "Telemetry publisher token for " + slug,
					"earlyRotationWindow": "168h",
					"expireAfter":         "720h",
					"namePrefix":          slug + "-telemetry-",
					"region":              region,
				},
				"providerConfigRef":          map[string]any{"kind": "ProviderConfig", "name": settings.organizationProviderConfigName},
				"writeConnectionSecretToRef": map[string]any{"name": tokenSecret},
			},
		)
	}

	outputDocument := fmt.Sprintf(
		`{{ $token := index . "attribute.token" | toString }}{"stack_slug":%q,"stack_region":%q,"access_policy_name":%q,"access_policy_token":{{ $token | toJson }}}`,
		slug,
		region,
		policyName,
	)
	desired["telemetry-credentials"] = newDesired(
		"external-secrets.io/v1alpha1",
		"PushSecret",
		namespace,
		policyName,
		nil,
		map[string]any{
			"refreshInterval": "1h",
			"updatePolicy":    "Replace",
			"deletionPolicy":  pushSecretDeletionPolicy,
			"secretStoreRefs": []any{settings.secretStoreReference()},
			"selector":        map[string]any{"secret": map[string]any{"name": tokenSecret}},
			"template": map[string]any{
				"engineVersion": "v2",
				"mergePolicy":   "Replace",
				"data":          map[string]any{"telemetry.json": outputDocument},
			},
			"data": []any{map[string]any{
				"match": map[string]any{
					"secretKey": "telemetry.json",
					"remoteRef": map[string]any{"remoteKey": outputPath},
				},
				"metadata": map[string]any{
					"apiVersion": "kubernetes.external-secrets.io/v1alpha1",
					"kind":       "PushSecretMetadata",
					"spec": map[string]any{
						"secretPushFormat": "string",
						"tags":             map[string]any{"grafana-cloud-vending-machine": "managed"},
					},
				},
			}},
		},
	)

	return nil
}

func telemetryAccessEnabled(spec map[string]any) bool {
	telemetry, ok := spec["telemetryAccess"].(map[string]any)
	if !ok {
		return true
	}
	enabled, exists := telemetry["enabled"].(bool)
	if !exists {
		return true
	}
	return enabled
}
