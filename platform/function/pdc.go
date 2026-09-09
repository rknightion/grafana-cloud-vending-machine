package main

import (
	"strconv"
	"strings"
	"time"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

const pdcRendererImplemented = true

// renderPDC owns the PDC networks, their bounded bootstrap tokens, and the
// datasource objects that use those networks. Keeping the datasource in this
// composite prevents a second claim from attaching an arbitrary network.
func renderPDC(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	metadata, _ := xr["metadata"].(map[string]any)
	spec, _ := xr["spec"].(map[string]any)
	name := stringValue(metadata, "name", "")
	namespace := stringValue(metadata, "namespace", "")
	stackRef, _ := spec["stackRef"].(map[string]any)
	stackName := stringValue(stackRef, "name", "")
	profileName := stringValue(spec, "profile", "")
	if name == "" || namespace == "" || stackName == "" || profileName == "" {
		return nil, errors.New("PDC must set metadata name and namespace, stackRef.name, and profile")
	}
	profile, err := configuredPDCProfile(config, profileName)
	if err != nil {
		return nil, err
	}
	stack, available, err := pdcStackContext(config, stackName, namespace)
	if err != nil {
		return nil, err
	}
	if !available {
		return map[resource.Name]*resource.DesiredComposed{}, nil
	}
	organizationProviderConfigName := stringValue(stack, "organizationProviderConfigName", "")
	if organizationProviderConfigName == "" {
		return nil, errors.New("trusted referenced stack context is incomplete")
	}

	token, _ := spec["token"].(map[string]any)
	expiresAfter := stringValue(token, "expiresAfter", "")
	if expiresAfter == "" {
		return nil, errors.New("PDC token must set expiresAfter")
	}
	requestedLifetime, err := time.ParseDuration(expiresAfter)
	if err != nil {
		return nil, errors.Wrap(err, "PDC token expiresAfter is invalid")
	}
	settings := configuredPlatformSettings(config)
	boundedLifetime, err := boundedTokenLifetime(settings.maximumTokenLifetime, requestedLifetime)
	if err != nil {
		return nil, err
	}
	if boundedLifetime != requestedLifetime {
		return nil, errors.Errorf("PDC token expiresAfter %q exceeds platform maximumTokenLifetime %q", expiresAfter, settings.maximumTokenLifetime)
	}
	issuance, err := pdcTokenIssuance(metadata, config, boundedLifetime)
	if err != nil {
		return nil, err
	}

	networks, err := pdcNetworks(spec)
	if err != nil {
		return nil, err
	}
	datasources, err := pdcDatasources(spec)
	if err != nil {
		return nil, err
	}

	desired := map[resource.Name]*resource.DesiredComposed{}
	networkIDs := map[string]string{}
	for _, network := range networks {
		logicalName := pdcNetworkResourceName(network.name)
		resourceName := name + "-pdc-" + stableResourceSuffix(network.name)
		desired[logicalName] = newDesired(
			"cloud.grafana.m.crossplane.io/v1alpha1",
			"PrivateDataSourceConnectNetwork",
			namespace,
			resourceName,
			pdcObservedExternalName(observed, logicalName),
			map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider": map[string]any{
					"cloudStackRef": map[string]any{"name": stackName},
					"displayName":   "PDC " + name + " " + network.name,
					"name":          name + "-" + network.name,
					"region":        profile.region,
				},
				"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": organizationProviderConfigName},
			},
		)
		tokenLogicalName := pdcTokenResourceName(network.name, issuance.window)
		id := pdcObservedNetworkID(observed, logicalName, tokenLogicalName)
		if id == "" {
			if pdcObservedTokenExists(observed, network.name) {
				return nil, errors.Errorf("waiting for observed PDC network ID for token network %q; preserving observed dependency", network.name)
			}
			continue
		}
		networkIDs[network.name] = id
		tokenResourceName := resourceName + "-token-" + strconv.FormatInt(issuance.window, 10)
		desired[tokenLogicalName] = newDesired(
			"cloud.grafana.m.crossplane.io/v1alpha1",
			"PrivateDataSourceConnectNetworkToken",
			namespace,
			tokenResourceName,
			pdcObservedExternalName(observed, tokenLogicalName),
			map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider": map[string]any{
					"displayName":  "PDC token " + name + " " + network.name,
					"expiresAt":    issuance.expiresAt.Format(time.RFC3339),
					"name":         name + "-" + network.name + "-token-" + strconv.FormatInt(issuance.window, 10),
					"pdcNetworkId": id,
					"region":       profile.region,
				},
				"providerConfigRef":          map[string]any{"kind": "ProviderConfig", "name": organizationProviderConfigName},
				"writeConnectionSecretToRef": map[string]any{"name": tokenResourceName},
			},
		)
	}

	for _, datasource := range datasources {
		networkID := networkIDs[datasource.network]
		suffix := stableResourceSuffix(name + ":" + datasource.name)
		logicalName := resource.Name("datasource-" + suffix)
		if networkID == "" {
			if _, exists := observed[logicalName]; exists {
				return nil, errors.Errorf("waiting for observed PDC network ID for datasource %q; preserving observed dependency", datasource.name)
			}
			continue
		}
		datasourceUID := "pdc-" + suffix
		parameters := map[string]any{
			"name":                              datasource.name,
			"privateDataSourceConnectNetworkId": networkID,
			"type":                              datasource.kind,
			"uid":                               datasourceUID,
		}
		if datasource.url != "" {
			parameters["url"] = datasource.url
		}
		desired[logicalName] = newDesired(
			"oss.grafana.m.crossplane.io/v1alpha1",
			"DataSource",
			namespace,
			name+"-datasource-"+suffix,
			map[string]any{"crossplane.io/external-name": datasourceUID},
			map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider":        parameters,
				"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": stackName},
			},
		)
	}
	return desired, nil
}

func pdcStackContext(config map[string]any, stackName, namespace string) (map[string]any, bool, error) {
	stack, ok := config["referencedStack"].(map[string]any)
	if !ok {
		return nil, false, nil
	}
	if stack["name"] != stackName || stack["namespace"] != namespace {
		return nil, false, errors.New("trusted referenced stack context does not match the request")
	}
	return stack, true, nil
}

type pdcNetwork struct{ name string }

func pdcNetworks(spec map[string]any) ([]pdcNetwork, error) {
	items, _ := spec["networks"].([]any)
	if len(items) == 0 {
		return nil, errors.New("PDC must declare at least one network")
	}
	result := make([]pdcNetwork, 0, len(items))
	seen := map[string]struct{}{}
	for index, value := range items {
		item, _ := value.(map[string]any)
		name := stringValue(item, "name", "")
		if name == "" {
			return nil, errors.Errorf("networks[%d] must set name", index)
		}
		if _, exists := seen[name]; exists {
			return nil, errors.Errorf("networks[%d] repeats name %q", index, name)
		}
		seen[name] = struct{}{}
		result = append(result, pdcNetwork{name: name})
	}
	return result, nil
}

type pdcDatasource struct {
	name    string
	kind    string
	network string
	url     string
}

func pdcDatasources(spec map[string]any) ([]pdcDatasource, error) {
	items, _ := spec["datasources"].([]any)
	result := make([]pdcDatasource, 0, len(items))
	seen := map[string]struct{}{}
	for index, value := range items {
		item, _ := value.(map[string]any)
		datasource := pdcDatasource{name: stringValue(item, "name", ""), kind: stringValue(item, "type", ""), network: stringValue(item, "network", ""), url: stringValue(item, "url", "")}
		if datasource.name == "" || datasource.kind == "" || datasource.network == "" {
			return nil, errors.Errorf("datasources[%d] must set name, type, and network", index)
		}
		if _, exists := seen[datasource.name]; exists {
			return nil, errors.Errorf("datasources[%d] repeats name %q", index, datasource.name)
		}
		seen[datasource.name] = struct{}{}
		result = append(result, datasource)
	}
	return result, nil
}

type pdcProfile struct {
	region string
}

func configuredPDCProfile(config map[string]any, requested string) (pdcProfile, error) {
	spec, _ := config["spec"].(map[string]any)
	profiles, _ := spec["pdcProfiles"].([]any)
	for _, value := range profiles {
		profile, _ := value.(map[string]any)
		if stringValue(profile, "name", "") == requested {
			region := stringValue(profile, "region", "")
			if region == "" {
				return pdcProfile{}, errors.Errorf("PDC profile %q must set region", requested)
			}
			return pdcProfile{region: region}, nil
		}
	}
	return pdcProfile{}, errors.Errorf("PDC profile %q is not configured by the platform", requested)
}

type pdcTokenWindow struct {
	window    int64
	expiresAt time.Time
}

// pdcTokenIssuance creates at most one token per half-lifetime window. The
// provider marks expiresAt ForceNew, so each window gets a new child identity
// while the preceding token remains valid for one more half-lifetime and then
// expires naturally under the existing Delete-free lifecycle.
func pdcTokenIssuance(metadata, config map[string]any, lifetime time.Duration) (pdcTokenWindow, error) {
	if lifetime < 2*time.Second {
		return pdcTokenWindow{}, errors.New("PDC token lifetime leaves no renewal window")
	}
	windowDuration := lifetime / 2
	if value := stringValue(metadata, "creationTimestamp", ""); value != "" {
		issuedAt, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return pdcTokenWindow{}, errors.Wrap(err, "PDC metadata.creationTimestamp is invalid")
		}
		now, ok := config[reconcileTimeConfigKey].(time.Time)
		if !ok {
			now = issuedAt
		}
		elapsed := now.UTC().Sub(issuedAt.UTC())
		window := int64(0)
		if elapsed > 0 {
			window = int64(elapsed / windowDuration)
		}
		startsAt := issuedAt.UTC().Add(time.Duration(window) * windowDuration)
		return pdcTokenWindow{window: window, expiresAt: startsAt.Add(lifetime)}, nil
	}
	return pdcTokenWindow{}, errors.New("PDC requires metadata.creationTimestamp to anchor the token renewal window")
}

func pdcNetworkResourceName(name string) resource.Name {
	return resource.Name("network-" + stableResourceSuffix(name))
}

func pdcTokenResourceName(name string, window int64) resource.Name {
	return resource.Name("network-token-" + stableResourceSuffix(name) + "-" + strconv.FormatInt(window, 10))
}

// pdcObservedNetworkID retains a provider-observed network ID when the
// network's status briefly omits it. The token itself reports the same ID, so
// retaining an observed token cannot fabricate a remote identity.
func pdcObservedNetworkID(observed map[resource.Name]resource.ObservedComposed, network, token resource.Name) string {
	if id := observedString(observed, network, "status.atProvider.pdcNetworkId"); id != "" {
		return id
	}
	if id := observedString(observed, token, "status.atProvider.pdcNetworkId"); id != "" {
		return id
	}
	if id := observedString(observed, token, "spec.forProvider.pdcNetworkId"); id != "" {
		return id
	}
	// A token created in the preceding renewal window is equally authoritative
	// for this network. It prevents a transient status gap from withdrawing the
	// current datasource while its replacement is being created.
	prefix := strings.TrimSuffix(string(token), strconv.FormatInt(tokenWindow(token), 10))
	for name := range observed {
		if !strings.HasPrefix(string(name), prefix) {
			continue
		}
		if id := observedString(observed, name, "status.atProvider.pdcNetworkId"); id != "" {
			return id
		}
		if id := observedString(observed, name, "spec.forProvider.pdcNetworkId"); id != "" {
			return id
		}
	}
	return ""
}

func pdcObservedTokenExists(observed map[resource.Name]resource.ObservedComposed, network string) bool {
	prefix := "network-token-" + stableResourceSuffix(network) + "-"
	for name := range observed {
		if strings.HasPrefix(string(name), prefix) {
			return true
		}
	}
	return false
}

func tokenWindow(name resource.Name) int64 {
	index := strings.LastIndexByte(string(name), '-')
	if index == -1 {
		return 0
	}
	value, err := strconv.ParseInt(string(name)[index+1:], 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func pdcObservedExternalName(observed map[resource.Name]resource.ObservedComposed, name resource.Name) map[string]any {
	item, exists := observed[name]
	if !exists || item.Resource == nil {
		return nil
	}
	metadata, _ := item.Resource.UnstructuredContent()["metadata"].(map[string]any)
	annotations, _ := metadata["annotations"].(map[string]any)
	externalName := stringValue(annotations, "crossplane.io/external-name", "")
	if externalName == "" {
		return nil
	}
	return map[string]any{"crossplane.io/external-name": externalName}
}
