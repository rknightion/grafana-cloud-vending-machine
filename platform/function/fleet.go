package main

import (
	"fmt"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

const fleetRendererImplemented = true

type fleetPipelineProfile struct {
	name     string
	matchers []any
	labels   map[string]string
}

// renderFleetPipelines renders the fixed attribution pipeline selected by a
// platform-owned profile. The request deliberately supplies only a profile
// name: matcher expressions and label values belong in the Composition input.
func renderFleetPipelines(xr map[string]any, _ map[resource.Name]resource.ObservedComposed, config map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	metadata, _ := xr["metadata"].(map[string]any)
	spec, _ := xr["spec"].(map[string]any)
	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	stackRef, _ := spec["stackRef"].(map[string]any)
	stackName, _ := stackRef["name"].(string)
	profileName, _ := spec["profile"].(string)
	if name == "" || namespace == "" || stackName == "" || profileName == "" {
		return nil, errors.New("fleet pipelines must set metadata name and namespace, stackRef.name, and profile")
	}

	profile, err := configuredFleetPipelineProfile(config, profileName)
	if err != nil {
		return nil, err
	}

	pipelineName := name + "-attribution"
	contents := fmt.Sprintf(`otelcol.processor.attributes "enforce_attribution" {
  action {
    key = "team"
    action = "upsert"
    value = %q
  }
  action {
    key = "cost-centre"
    action = "upsert"
    value = %q
  }
  action {
    key = "environment"
    action = "upsert"
    value = %q
  }
}`,
		profile.labels["team"], profile.labels["cost-centre"], profile.labels["environment"])

	return map[resource.Name]*resource.DesiredComposed{
		"attribution-pipeline": newDesired(
			"fleetmanagement.grafana.m.crossplane.io/v1alpha1",
			"Pipeline",
			namespace,
			pipelineName,
			map[string]any{"crossplane.io/external-name": pipelineName},
			map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider": map[string]any{
					"configType":               "ALLOY",
					"contents":                 contents,
					"enabled":                  true,
					"matchers":                 profile.matchers,
					"name":                     pipelineName,
					"terraformSourceNamespace": "default",
				},
				"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": stackName},
			},
		),
	}, nil
}

func configuredFleetPipelineProfile(config map[string]any, name string) (fleetPipelineProfile, error) {
	spec, _ := config["spec"].(map[string]any)
	profiles, _ := spec["fleetPipelineProfiles"].([]any)
	for _, value := range profiles {
		profile, _ := value.(map[string]any)
		if stringValue(profile, "name", "") != name {
			continue
		}
		labelsValue, _ := profile["labels"].(map[string]any)
		labels := map[string]string{}
		for _, key := range []string{"team", "cost-centre", "environment"} {
			labels[key] = stringValue(labelsValue, key, "")
			if labels[key] == "" {
				return fleetPipelineProfile{}, errors.Errorf("fleet pipeline profile %q must set labels.%s", name, key)
			}
		}
		matchers, _ := profile["matchers"].([]any)
		if len(matchers) == 0 {
			return fleetPipelineProfile{}, errors.Errorf("fleet pipeline profile %q must set at least one matcher", name)
		}
		for index, matcher := range matchers {
			if value, ok := matcher.(string); !ok || value == "" {
				return fleetPipelineProfile{}, errors.Errorf("fleet pipeline profile %q matcher %d must be a non-empty string", name, index)
			}
		}
		return fleetPipelineProfile{name: name, matchers: matchers, labels: labels}, nil
	}
	return fleetPipelineProfile{}, errors.Errorf("fleet pipeline profile %q is not configured by the platform", name)
}

// addFleetAccess is called by the stack renderer. Fleet Management has a
// distinct basic-auth credential, so its access policy and rotating token are
// deliberately separate from the stack service-account token.
func addFleetAccess(
	desired map[resource.Name]*resource.DesiredComposed,
	observed map[resource.Name]resource.ObservedComposed,
	namespace, slug, region, outputPath, profile string,
	settings platformSettings,
	organizationProviderConfigName string,
	deletingExternalResources bool,
) error {
	tokenLifetime, err := boundedTokenLifetime(settings.maximumTokenLifetime, requestedTokenLifetime)
	if err != nil {
		return err
	}
	rotationWindow, err := boundedTokenEarlyRotationWindow(tokenLifetime)
	if err != nil {
		return err
	}
	allowedSubnets, err := selectedTokenUseAllowedSubnets(settings.tokenUseNetworkProfiles, profile)
	if err != nil {
		return err
	}
	policyName := slug + "-fleet-management"
	tokenSecret := slug + "-fleet-management-token"
	externalResourcePolicies := managementPolicies
	deleteOnDestroy := false
	pushSecretDeletionPolicy := "None"
	if deletingExternalResources {
		externalResourcePolicies = []any{"*"}
		deleteOnDestroy = true
		pushSecretDeletionPolicy = "Delete"
	}

	policyForProvider := map[string]any{
		"displayName": "Fleet Management for " + slug,
		"name":        policyName,
		"realm":       []any{map[string]any{"stackRef": map[string]any{"name": slug}, "type": "stack"}},
		"region":      region,
		"scopes":      []any{"fleet-management:read", "fleet-management:write"},
	}
	if len(allowedSubnets) > 0 {
		policyForProvider["conditions"] = []any{map[string]any{"allowedSubnets": allowedSubnets}}
	}
	desired["fleet-management-access-policy"] = newDesired(
		"cloud.grafana.m.crossplane.io/v1alpha1", "AccessPolicy", namespace, policyName, nil,
		map[string]any{
			"managementPolicies": externalResourcePolicies,
			"forProvider":        policyForProvider,
			"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": organizationProviderConfigName},
		},
	)

	policyID := observedString(observed, "fleet-management-access-policy", "status.atProvider.policyId")
	if policyID == "" {
		policyID = observedString(observed, "fleet-management-token", "spec.forProvider.accessPolicyId")
	}
	if policyID == "" {
		return nil
	}
	desired["fleet-management-token"] = newDesired(
		"cloud.grafana.m.crossplane.io/v1alpha1", "AccessPolicyRotatingToken", namespace, policyName, nil,
		map[string]any{
			"managementPolicies": externalResourcePolicies,
			"forProvider": map[string]any{
				"accessPolicyId": policyID, "deleteOnDestroy": deleteOnDestroy,
				"displayName":         "Fleet Management token for " + slug,
				"earlyRotationWindow": tokenDurationString(rotationWindow), "expireAfter": tokenDurationString(tokenLifetime),
				"namePrefix": slug + "-fleet-management-", "region": region,
			},
			"providerConfigRef":          map[string]any{"kind": "ProviderConfig", "name": organizationProviderConfigName},
			"writeConnectionSecretToRef": map[string]any{"name": tokenSecret},
		},
	)

	stackID := observedString(observed, "stack", "status.atProvider.id")
	if stackID == "" {
		return nil
	}
	outputDocument := fmt.Sprintf(
		`{{ $token := index . "attribute.token" | toString }}{"fleet_management_auth":{{ printf "%%s:%%s" %q $token | toJson }}}`, stackID,
	)
	desired["fleet-management-credentials"] = newDesired(
		"external-secrets.io/v1alpha1", "PushSecret", namespace, policyName, nil,
		map[string]any{
			"refreshInterval": "1h", "updatePolicy": "Replace", "deletionPolicy": pushSecretDeletionPolicy,
			"secretStoreRefs": []any{settings.secretStoreReference()},
			"selector":        map[string]any{"secret": map[string]any{"name": tokenSecret}},
			"template": map[string]any{
				"engineVersion": "v2", "mergePolicy": "Replace",
				"data": map[string]any{"fleet-management.json": outputDocument},
			},
			"data": []any{map[string]any{
				"match": map[string]any{"secretKey": "fleet-management.json", "remoteRef": map[string]any{"remoteKey": outputPath + "/fleet-management"}},
				"metadata": map[string]any{
					"apiVersion": "kubernetes.external-secrets.io/v1alpha1", "kind": "PushSecretMetadata",
					"spec": map[string]any{"secretPushFormat": "string", "tags": map[string]any{"grafana-cloud-vending-machine": "managed"}},
				},
			}},
		},
	)
	return nil
}
