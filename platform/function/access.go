package main

import (
	"fmt"
	"net"
	"time"

	"github.com/crossplane/function-sdk-go/resource"
)

const (
	requestedTokenLifetime            = 720 * time.Hour
	requestedTokenEarlyRotationWindow = 168 * time.Hour
)

func addTelemetryAccess(
	desired map[resource.Name]*resource.DesiredComposed,
	observed map[resource.Name]resource.ObservedComposed,
	namespace, slug, region, outputPath string,
	spec map[string]any,
	settings platformSettings,
	organizationProviderConfigName string,
	deletingExternalResources bool,
) error {
	if !telemetryAccessEnabled(spec) {
		return nil
	}

	tokenLifetime, err := boundedTokenLifetime(settings.maximumTokenLifetime, requestedTokenLifetime)
	if err != nil {
		return err
	}
	rotationWindow, err := boundedTokenEarlyRotationWindow(tokenLifetime)
	if err != nil {
		return err
	}
	profile := stringValue(spec, "profile", "standard")
	allowedSubnets, err := selectedTokenUseAllowedSubnets(settings.tokenUseNetworkProfiles, profile)
	if err != nil {
		return err
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
	policyForProvider := map[string]any{
		"displayName": "Telemetry publisher for " + slug,
		"name":        policyName,
		"realm": []any{map[string]any{
			"stackRef": map[string]any{"name": slug},
			"type":     "stack",
		}},
		"region": region,
		"scopes": []any{"stacks:read", "metrics:write", "logs:write", "traces:write"},
	}
	if len(allowedSubnets) > 0 {
		policyForProvider["conditions"] = []any{map[string]any{"allowedSubnets": allowedSubnets}}
	}
	desired["telemetry-access-policy"] = newDesired(
		"cloud.grafana.m.crossplane.io/v1alpha1",
		"AccessPolicy",
		namespace,
		policyName,
		nil,
		map[string]any{
			"managementPolicies": externalResourcePolicies,
			"forProvider":        policyForProvider,
			"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": organizationProviderConfigName},
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
					"earlyRotationWindow": tokenDurationString(rotationWindow),
					"expireAfter":         tokenDurationString(tokenLifetime),
					"namePrefix":          slug + "-telemetry-",
					"region":              region,
				},
				"providerConfigRef":          map[string]any{"kind": "ProviderConfig", "name": organizationProviderConfigName},
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

// boundedTokenLifetime applies the platform-owned maximum to a renderer's
// requested lifetime. An absent or malformed maximum is an unsafe platform
// configuration, so callers fail closed instead of rendering a token.
func boundedTokenLifetime(maximum string, requested time.Duration) (time.Duration, error) {
	if maximum == "" {
		return 0, fmt.Errorf("platform configuration must set maximumTokenLifetime")
	}
	limit, err := time.ParseDuration(maximum)
	if err != nil {
		return 0, fmt.Errorf("platform maximumTokenLifetime %q is invalid: %w", maximum, err)
	}
	if limit < 2*time.Second || limit%time.Second != 0 {
		return 0, fmt.Errorf("platform maximumTokenLifetime must be a whole number of seconds and at least 2s")
	}
	if requested < 2*time.Second || requested%time.Second != 0 {
		return 0, fmt.Errorf("requested token lifetime must be a whole number of seconds and at least 2s")
	}
	if requested > limit {
		return limit, nil
	}
	return requested, nil
}

// boundedTokenEarlyRotationWindow keeps the provider-required rotation window
// below the resulting lifetime when a platform chooses a short ceiling.
func boundedTokenEarlyRotationWindow(lifetime time.Duration) (time.Duration, error) {
	if lifetime < 2*time.Second || lifetime%time.Second != 0 {
		return 0, fmt.Errorf("bounded token lifetime must be a whole number of seconds and at least 2s")
	}
	window := requestedTokenEarlyRotationWindow
	if window >= lifetime {
		window = (lifetime / 2).Truncate(time.Second)
	}
	if window < time.Second {
		return 0, fmt.Errorf("bounded token lifetime %s leaves no valid early rotation window", lifetime)
	}
	return window, nil
}

func tokenDurationString(value time.Duration) string {
	seconds := int64(value / time.Second)
	if seconds%3600 == 0 {
		return fmt.Sprintf("%dh", seconds/3600)
	}
	if seconds%60 == 0 {
		return fmt.Sprintf("%dm", seconds/60)
	}
	return fmt.Sprintf("%ds", seconds)
}

// selectedTokenUseAllowedSubnets resolves a network allow-list from a
// platform-owned profile. The stack request selects only the profile name; it
// never supplies CIDRs that are copied into an AccessPolicy.
func selectedTokenUseAllowedSubnets(profiles []any, selectedProfile string) ([]any, error) {
	var selected map[string]any
	for _, value := range profiles {
		profile, ok := value.(map[string]any)
		if !ok || stringValue(profile, "name", "") != selectedProfile {
			continue
		}
		if selected != nil {
			return nil, fmt.Errorf("token-use network profile %q is configured more than once", selectedProfile)
		}
		selected = profile
	}
	if selected == nil {
		return nil, nil
	}
	value, defined := selected["allowedSubnets"]
	if !defined {
		return nil, nil
	}
	subnets, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("token-use network profile %q allowedSubnets must be an array", selectedProfile)
	}
	if len(subnets) == 0 {
		return nil, fmt.Errorf("token-use network profile %q allowedSubnets must not be empty; omit the field for unrestricted token use", selectedProfile)
	}
	result := make([]any, 0, len(subnets))
	for index, value := range subnets {
		subnet, ok := value.(string)
		if !ok || subnet == "" {
			return nil, fmt.Errorf("token-use network profile %q allowedSubnets[%d] must be a non-empty CIDR string", selectedProfile, index)
		}
		if _, _, err := net.ParseCIDR(subnet); err != nil {
			return nil, fmt.Errorf("token-use network profile %q allowedSubnets[%d] %q is not a valid CIDR: %w", selectedProfile, index, subnet, err)
		}
		result = append(result, subnet)
	}
	return result, nil
}

// publishTokenExpiryStatus copies provider-observed expiry timestamps to the
// composite without trying to predict a rotating token's actual replacement
// time. The two provider token kinds expose different status field names.
func publishTokenExpiryStatus(status map[string]any, observed map[resource.Name]resource.ObservedComposed) {
	expiries := map[string]any{}
	for _, token := range []struct {
		resourceName resource.Name
		fieldPath    string
		statusName   string
	}{
		{resourceName: "stack-token", fieldPath: "status.atProvider.expiration", statusName: "administrator"},
		{resourceName: "fleet-management-token", fieldPath: "status.atProvider.expiresAt", statusName: "fleetManagement"},
		{resourceName: "telemetry-token", fieldPath: "status.atProvider.expiresAt", statusName: "telemetryPublisher"},
	} {
		if expiry := observedString(observed, token.resourceName, token.fieldPath); expiry != "" {
			expiries[token.statusName] = expiry
		}
	}
	if len(expiries) == 0 {
		delete(status, "tokenExpiries")
		return
	}
	status["tokenExpiries"] = expiries
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
