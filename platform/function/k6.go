package main

import (
	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

const k6RendererImplemented = true

const resolvedK6StackConfigKey = "_resolvedK6Stack"

const k6APIVersion = "k6.grafana.m.crossplane.io/v1alpha1"

// renderK6Project creates the bounded k6 project surface. It deliberately
// waits for trusted stack context because the bootstrap token belongs to the
// referenced stack and must never be accepted or stored on this request.
func renderK6Project(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	stack, ready := resolvedK6Stack(config)
	if !ready {
		return map[resource.Name]*resource.DesiredComposed{}, nil
	}

	metadata, _ := xr["metadata"].(map[string]any)
	spec, _ := xr["spec"].(map[string]any)
	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	stackRef, _ := spec["stackRef"].(map[string]any)
	stackName, _ := stackRef["name"].(string)
	grafanaUser, _ := spec["grafanaUser"].(string)
	if name == "" || namespace == "" || stackName == "" || grafanaUser == "" {
		return nil, errors.New("k6 project must set metadata name and namespace plus stackRef.name and grafanaUser")
	}

	usage, _ := stack["usage"].(string)
	stackID, _ := stack["stackId"].(string)
	outputSecretPath, _ := stack["outputSecretPath"].(string)
	organizationProviderConfigName, _ := stack["organizationProviderConfigName"].(string)
	bootstrapSecret, _ := stack["serviceAccountTokenSecret"].(map[string]any)
	bootstrapName, _ := bootstrapSecret["name"].(string)
	bootstrapKey, _ := bootstrapSecret["key"].(string)
	bootstrapReady, _ := bootstrapSecret["ready"].(bool)
	if usage == "" || stackID == "" || outputSecretPath == "" || organizationProviderConfigName == "" || bootstrapName == "" || bootstrapKey == "" || !bootstrapReady {
		return map[resource.Name]*resource.DesiredComposed{}, nil
	}

	limits, err := configuredK6LimitProfile(config, usage)
	if err != nil {
		return nil, err
	}
	allowedLoadZones, err := requestedK6LoadZones(spec, limits.allowedLoadZones)
	if err != nil {
		return nil, err
	}

	desired := map[resource.Name]*resource.DesiredComposed{}
	desired["installation"] = newDesired(
		k6APIVersion,
		"Installation",
		namespace,
		name+"-installation",
		nil,
		map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider": map[string]any{
				"grafanaSaTokenSecretRef": map[string]any{"name": bootstrapName, "key": bootstrapKey},
				"grafanaUser":             grafanaUser,
				"stackId":                 stackID,
			},
			"providerConfigRef":          k6ProviderConfigReference(organizationProviderConfigName),
			"writeConnectionSecretToRef": map[string]any{"name": name + "-k6-token"},
		},
	)

	// The installation creates the derived k6 token. Keep the project tree out
	// of desired state until the provider has observed that bootstrap complete.
	if !observedReady(observed, "installation") {
		return desired, nil
	}

	derivedSecret := name + "-k6-token"
	providerCredentialsSecret := name + "-k6-provider-credentials"
	providerConfigName := name + "-k6"
	outputDocument := `{{ $token := index . "attribute.k6_access_token" | toString }}{"k6_access_token":{{ $token | toJson }}}`
	settings := configuredPlatformSettings(config)
	desired["credentials"] = newDesired(
		"external-secrets.io/v1alpha1",
		"PushSecret",
		namespace,
		name+"-k6-credentials",
		nil,
		map[string]any{
			"refreshInterval": "1h", "updatePolicy": "Replace", "deletionPolicy": "None",
			"secretStoreRefs": []any{settings.secretStoreReference()},
			"selector":        map[string]any{"secret": map[string]any{"name": derivedSecret}},
			"template": map[string]any{
				"engineVersion": "v2", "mergePolicy": "Replace", "data": map[string]any{"k6.json": outputDocument},
			},
			"data": []any{map[string]any{
				"match": map[string]any{"secretKey": "k6.json", "remoteRef": map[string]any{"remoteKey": outputSecretPath + "/k6"}},
				"metadata": map[string]any{
					"apiVersion": "kubernetes.external-secrets.io/v1alpha1", "kind": "PushSecretMetadata",
					"spec": map[string]any{"secretPushFormat": "string", "tags": map[string]any{"grafana-cloud-vending-machine": "managed"}},
				},
			}},
		},
	)
	desired["provider-credentials"] = newDesired(
		"external-secrets.io/v1",
		"ExternalSecret",
		namespace,
		providerCredentialsSecret,
		nil,
		map[string]any{
			"refreshInterval": "1h",
			"secretStoreRef":  settings.secretStoreReference(),
			"target": map[string]any{
				"name": providerCredentialsSecret, "creationPolicy": "Owner", "deletionPolicy": "Retain",
				"template": map[string]any{
					"engineVersion": "v2",
					"data":          map[string]any{"credentials": `{"k6_access_token":{{ .k6AccessToken | toJson }}}`},
				},
			},
			"data": []any{map[string]any{
				"secretKey": "k6AccessToken",
				"remoteRef": map[string]any{"key": outputSecretPath + "/k6", "property": "k6_access_token"},
			}},
		},
	)
	desired["provider-config"] = newDesired(
		"grafana.m.crossplane.io/v1beta1",
		"ProviderConfig",
		namespace,
		providerConfigName,
		nil,
		map[string]any{
			"credentials": map[string]any{
				"source": "Secret",
				"secretRef": map[string]any{
					"name": providerCredentialsSecret, "namespace": namespace, "key": "credentials",
				},
			},
		},
	)

	if !observedReady(observed, "provider-credentials") {
		return desired, nil
	}

	desired["project"] = newDesired(
		k6APIVersion,
		"Project",
		namespace,
		name,
		nil,
		map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider":        map[string]any{"name": name},
			"providerConfigRef":  k6ProviderConfigReference(providerConfigName),
		},
	)
	projectID := observedString(observed, "project", "status.atProvider.id")
	if projectID == "" {
		return desired, nil
	}

	desired["limits"] = newDesired(
		k6APIVersion,
		"ProjectLimits",
		namespace,
		name+"-limits",
		map[string]any{"crossplane.io/external-name": projectID},
		map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider": map[string]any{
				"projectId":           projectID,
				"vuhMaxPerMonth":      limits.vuhMaxPerMonth,
				"vuMaxPerTest":        limits.vuMaxPerTest,
				"vuBrowserMaxPerTest": limits.vuBrowserMaxPerTest,
				"durationMaxPerTest":  limits.durationMaxPerTest,
			},
			"providerConfigRef": k6ProviderConfigReference(providerConfigName),
		},
	)

	desired["allowed-load-zones"] = newDesired(
		k6APIVersion,
		"ProjectAllowedLoadZones",
		namespace,
		name+"-allowed-load-zones",
		map[string]any{"crossplane.io/external-name": projectID},
		map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider": map[string]any{
				"projectId": projectID, "allowedLoadZones": allowedLoadZones,
			},
			"providerConfigRef": k6ProviderConfigReference(providerConfigName),
		},
	)

	return desired, nil
}

type k6LimitProfile struct {
	usage               string
	vuhMaxPerMonth      float64
	vuMaxPerTest        float64
	vuBrowserMaxPerTest float64
	durationMaxPerTest  float64
	allowedLoadZones    []string
}

func resolvedK6Stack(config map[string]any) (map[string]any, bool) {
	stack, ok := config[resolvedK6StackConfigKey].(map[string]any)
	return stack, ok
}

func configuredK6LimitProfile(config map[string]any, usage string) (k6LimitProfile, error) {
	spec, _ := config["spec"].(map[string]any)
	profiles, _ := spec["k6LimitProfiles"].([]any)
	var matched *k6LimitProfile
	for index, raw := range profiles {
		profile, ok := raw.(map[string]any)
		if !ok {
			return k6LimitProfile{}, errors.Errorf("k6LimitProfiles[%d] must be an object", index)
		}
		profileUsage, _ := profile["usage"].(string)
		if profileUsage != usage {
			continue
		}
		if matched != nil {
			return k6LimitProfile{}, errors.Errorf("k6LimitProfiles contains multiple profiles for usage %q", usage)
		}
		candidate := k6LimitProfile{usage: usage}
		var valid bool
		if candidate.vuhMaxPerMonth, valid = k6PositiveNumber(profile, "vuhMaxPerMonth"); !valid {
			return k6LimitProfile{}, errors.Errorf("k6LimitProfiles[%d].vuhMaxPerMonth must be a positive number", index)
		}
		if candidate.vuMaxPerTest, valid = k6PositiveNumber(profile, "vuMaxPerTest"); !valid {
			return k6LimitProfile{}, errors.Errorf("k6LimitProfiles[%d].vuMaxPerTest must be a positive number", index)
		}
		if candidate.vuBrowserMaxPerTest, valid = k6PositiveNumber(profile, "vuBrowserMaxPerTest"); !valid {
			return k6LimitProfile{}, errors.Errorf("k6LimitProfiles[%d].vuBrowserMaxPerTest must be a positive number", index)
		}
		if candidate.durationMaxPerTest, valid = k6PositiveNumber(profile, "durationMaxPerTest"); !valid {
			return k6LimitProfile{}, errors.Errorf("k6LimitProfiles[%d].durationMaxPerTest must be a positive number", index)
		}
		candidate.allowedLoadZones = stringListValue(profile, "allowedLoadZones", nil)
		matched = &candidate
	}
	if matched == nil {
		return k6LimitProfile{}, errors.Errorf("no k6 limit profile is configured for referenced stack usage %q", usage)
	}
	return *matched, nil
}

func k6PositiveNumber(values map[string]any, key string) (float64, bool) {
	value, ok := values[key].(float64)
	return value, ok && value > 0
}

func k6ProviderConfigReference(name string) map[string]any {
	return map[string]any{"kind": "ProviderConfig", "name": name}
}

func requestedK6LoadZones(spec map[string]any, permitted []string) ([]any, error) {
	requested, ok := spec["allowedLoadZones"].([]any)
	if !ok {
		return nil, errors.New("allowedLoadZones must be explicitly supplied as an array; use an empty array to allow no private zones")
	}
	permittedSet := make(map[string]struct{}, len(permitted))
	for _, zone := range permitted {
		permittedSet[zone] = struct{}{}
	}
	result := make([]any, 0, len(requested))
	seen := map[string]struct{}{}
	for index, value := range requested {
		zone, ok := value.(string)
		if !ok || zone == "" {
			return nil, errors.Errorf("allowedLoadZones[%d] must be a non-empty string", index)
		}
		if _, duplicate := seen[zone]; duplicate {
			return nil, errors.Errorf("allowedLoadZones contains duplicate zone %q", zone)
		}
		if _, allowed := permittedSet[zone]; !allowed {
			return nil, errors.Errorf("allowedLoadZones[%d] %q is not permitted by the platform profile", index, zone)
		}
		seen[zone] = struct{}{}
		result = append(result, zone)
	}
	return result, nil
}
