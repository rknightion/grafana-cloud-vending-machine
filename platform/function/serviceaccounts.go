package main

import (
	"fmt"
	"time"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

const serviceAccountsRendererImplemented = true

type serviceAccountProfile struct {
	name          string
	role          string
	tokenLifetime time.Duration
	permissions   []any
}

// renderServiceAccounts vends named in-stack accounts from a platform-owned
// profile. A request selects the profile and account names only: its role,
// token lifetime, and complete service-account permission set never come from
// the request.
func renderServiceAccounts(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	metadata, _ := xr["metadata"].(map[string]any)
	spec, _ := xr["spec"].(map[string]any)
	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	stackRef, _ := spec["stackRef"].(map[string]any)
	stackName, _ := stackRef["name"].(string)
	profileName, _ := spec["profile"].(string)
	accounts, _ := spec["accounts"].([]any)
	if name == "" || namespace == "" || stackName == "" || profileName == "" || len(accounts) == 0 {
		return nil, errors.New("service account request must set metadata name and namespace, stackRef.name, profile, and at least one account")
	}

	if name != stackName {
		return nil, errors.New("metadata.name must match spec.stackRef.name so one composite owns this stack surface")
	}
	profile, err := configuredServiceAccountProfile(config, profileName)
	if err != nil {
		return nil, err
	}
	settings := configuredPlatformSettings(config)
	tokenLifetime, err := boundedTokenLifetime(settings.maximumTokenLifetime, profile.tokenLifetime)
	if err != nil {
		return nil, err
	}
	// A profile is platform configuration. Silently clamping it would hide a
	// policy error, so use the frozen shared helper and reject any clamp.
	if tokenLifetime != profile.tokenLifetime {
		return nil, errors.Errorf("service account profile %q token lifetime %s exceeds platform maximumTokenLifetime %s", profile.name, tokenDurationString(profile.tokenLifetime), settings.maximumTokenLifetime)
	}
	rotationWindow, err := boundedTokenEarlyRotationWindow(tokenLifetime)
	if err != nil {
		return nil, err
	}

	stack, available, err := serviceAccountStackContext(config, stackName, namespace)
	if err != nil {
		return nil, err
	}
	if !available {
		return map[resource.Name]*resource.DesiredComposed{}, nil
	}
	outputSecretPath, _ := stack["outputSecretPath"].(string)
	providerConfigName, _ := stack["providerConfigName"].(string)
	if outputSecretPath == "" || providerConfigName == "" {
		return nil, errors.New("trusted referenced stack context is incomplete")
	}

	desired := map[resource.Name]*resource.DesiredComposed{}
	seen := map[string]struct{}{}
	for index, raw := range accounts {
		account, _ := raw.(map[string]any)
		accountName, _ := account["name"].(string)
		if accountName == "" {
			return nil, errors.Errorf("accounts[%d].name must be a non-empty string", index)
		}
		if _, duplicate := seen[accountName]; duplicate {
			return nil, errors.Errorf("accounts contains duplicate name %q", accountName)
		}
		seen[accountName] = struct{}{}

		logicalPrefix := resource.Name("service-account-" + accountName)
		resourcePrefix := name + "-" + accountName
		desired[logicalPrefix] = newDesired(
			"oss.grafana.m.crossplane.io/v1alpha1", "ServiceAccount", namespace, resourcePrefix,
			nil,
			map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider": map[string]any{
					"name": accountName, "role": profile.role, "isDisabled": false,
				},
				"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": providerConfigName},
			},
		)

		serviceAccountID := observedString(observed, logicalPrefix, "status.atProvider.id")
		if serviceAccountID == "" {
			continue
		}

		tokenSecret := resourcePrefix + "-token"
		desired[resource.Name(string(logicalPrefix)+"-token")] = newDesired(
			"oss.grafana.m.crossplane.io/v1alpha1", "ServiceAccountRotatingToken", namespace, resourcePrefix+"-token",
			nil,
			map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider": map[string]any{
					"serviceAccountId":           serviceAccountID,
					"namePrefix":                 resourcePrefix + "-",
					"secondsToLive":              int64(tokenLifetime / time.Second),
					"earlyRotationWindowSeconds": int64(rotationWindow / time.Second),
					"deleteOnDestroy":            false,
				},
				"providerConfigRef":          map[string]any{"kind": "ProviderConfig", "name": providerConfigName},
				"writeConnectionSecretToRef": map[string]any{"name": tokenSecret},
			},
		)

		// ServiceAccountPermission is the provider's authoritative whole-set
		// resource. ServiceAccountPermissionItem intentionally never appears:
		// an item writer would race this owner and preserve unmanaged grants.
		desired[resource.Name(string(logicalPrefix)+"-permissions")] = newDesired(
			"oss.grafana.m.crossplane.io/v1alpha1", "ServiceAccountPermission", namespace, resourcePrefix+"-permissions",
			map[string]any{"crossplane.io/external-name": serviceAccountID},
			map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider": map[string]any{
					"serviceAccountId": serviceAccountID, "permissions": profile.permissions,
				},
				"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": providerConfigName},
			},
		)

		desired[resource.Name(string(logicalPrefix)+"-credentials")] = newDesired(
			"external-secrets.io/v1alpha1", "PushSecret", namespace, resourcePrefix+"-credentials",
			nil,
			map[string]any{
				"refreshInterval": "1h", "updatePolicy": "Replace", "deletionPolicy": "None",
				"secretStoreRefs": []any{settings.secretStoreReference()},
				"selector":        map[string]any{"secret": map[string]any{"name": tokenSecret}},
				"template": map[string]any{
					"engineVersion": "v2", "mergePolicy": "Replace",
					"data": map[string]any{"service-account.json": `{{ $token := index . "attribute.key" | toString }}{"service_account_token":{{ $token | toJson }}}`},
				},
				"data": []any{map[string]any{
					"match":    map[string]any{"secretKey": "service-account.json", "remoteRef": map[string]any{"remoteKey": outputSecretPath + "/service-accounts/" + accountName}},
					"metadata": map[string]any{"apiVersion": "kubernetes.external-secrets.io/v1alpha1", "kind": "PushSecretMetadata", "spec": map[string]any{"secretPushFormat": "string", "tags": map[string]any{"grafana-cloud-vending-machine": "managed"}}},
				}},
			},
		)
	}
	return desired, nil
}

func serviceAccountStackContext(config map[string]any, stackName, namespace string) (map[string]any, bool, error) {
	stack, ok := config["referencedStack"].(map[string]any)
	if !ok {
		return nil, false, nil
	}
	if stack["name"] != stackName || stack["namespace"] != namespace {
		return nil, false, errors.New("trusted referenced stack context does not match the request")
	}
	return stack, true, nil
}

func configuredServiceAccountProfile(config map[string]any, name string) (serviceAccountProfile, error) {
	spec, _ := config["spec"].(map[string]any)
	profiles, _ := spec["serviceAccountProfiles"].([]any)
	for index, raw := range profiles {
		profile, _ := raw.(map[string]any)
		if profile["name"] != name {
			continue
		}
		role, _ := profile["role"].(string)
		lifetimeRaw, _ := profile["tokenLifetime"].(string)
		lifetime, err := time.ParseDuration(lifetimeRaw)
		if role == "" || lifetimeRaw == "" || err != nil || lifetime < 2*time.Second || lifetime%time.Second != 0 {
			return serviceAccountProfile{}, errors.Errorf("serviceAccountProfiles[%d] %q must set role and a whole-second tokenLifetime of at least 2s", index, name)
		}
		permissions, ok := profile["permissions"].([]any)
		if !ok {
			return serviceAccountProfile{}, errors.Errorf("serviceAccountProfiles[%d] %q must set permissions as an array", index, name)
		}
		return serviceAccountProfile{name: name, role: role, tokenLifetime: lifetime, permissions: permissions}, nil
	}
	return serviceAccountProfile{}, fmt.Errorf("service account profile %q is not configured", name)
}
