package main

import (
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

const (
	stackConsumerObserverName    resource.Name = "stack-consumer-observed-stack"
	stackConsumerPolicyName      resource.Name = "stack-consumer-access-policy"
	stackConsumerTokenName       resource.Name = "stack-consumer-token"
	stackConsumerCredentialsName resource.Name = "stack-consumer-credentials"

	stackConsumerIncompleteProfileMessage = "selected stack consumer profile is not a complete platform-owned credential configuration"
	stackConsumerNamespaceMessage         = "selected stack consumer profile does not authorize this namespace"
	stackConsumerTargetMessage            = "selected stack consumer profile does not authorize this stack slug and region"
)

type stackConsumerProfile struct {
	name               string
	namespace          string
	providerConfigName string
	consumerName       string
	outputSecretPath   string
	scopes             []any
}

// renderStackConsumer observes a platform-authorized stack before minting any
// credential against it. The request supplies a slug and region, never the
// provider-assigned stack ID, provider configuration, scopes, or output path.
func renderStackConsumer(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	metadata, _ := xr["metadata"].(map[string]any)
	spec, _ := xr["spec"].(map[string]any)
	name := stringValue(metadata, "name", "")
	namespace := stringValue(metadata, "namespace", "")
	profileName := stringValue(spec, "profile", "")
	stack, _ := spec["stack"].(map[string]any)
	slug := stringValue(stack, "slug", "")
	region := stringValue(stack, "region", "")
	if name == "" || namespace == "" || profileName == "" || slug == "" || region == "" {
		return nil, errors.New("stack consumer must set metadata name and namespace, profile, and stack slug and region")
	}
	if name != profileName {
		return nil, errors.New("stack consumer metadata.name must match spec.profile")
	}

	profile, err := configuredStackConsumerProfile(config, profileName, namespace, slug, region)
	if err != nil {
		return nil, err
	}
	settings := configuredPlatformSettings(config)
	tokenLifetime, err := boundedTokenLifetime(settings.maximumTokenLifetime, requestedTokenLifetime)
	if err != nil {
		return nil, err
	}
	allowedSubnets, err := selectedTokenUseAllowedSubnets(settings.tokenUseNetworkProfiles, profileName)
	if err != nil {
		return nil, err
	}
	rotationWindow, err := boundedTokenEarlyRotationWindow(tokenLifetime)
	if err != nil {
		return nil, err
	}

	observer := stackConsumerObserver(namespace, slug, region, profile)
	desired := map[resource.Name]*resource.DesiredComposed{stackConsumerObserverName: observer}
	stackID, found, err := observedStackConsumerIdentity(observed, observer, namespace, slug, region, profile)
	if err != nil {
		return nil, err
	}
	if !found {
		if stackConsumerCredentialChildObserved(observed) {
			return nil, errors.New("observed stack consumer credential child would be withdrawn while stack identity is unavailable")
		}
		return desired, nil
	}

	policy := stackConsumerPolicy(namespace, slug, region, stackID, allowedSubnets, profile)
	desired[stackConsumerPolicyName] = policy
	policyID, found, err := observedStackConsumerPolicyID(observed, policy, namespace, region, profile, stackID)
	if err != nil {
		return nil, err
	}
	stackConsumerPreserveObservedExternalName(policy, observed, stackConsumerPolicyName)
	if !found {
		if observedExists(observed, stackConsumerTokenName) || observedExists(observed, stackConsumerCredentialsName) {
			return nil, errors.New("observed stack consumer credential child would be withdrawn while access policy identity is unavailable")
		}
		return desired, nil
	}

	token := stackConsumerToken(namespace, slug, region, policyID, tokenLifetime, rotationWindow, profile)
	if err := validateObservedStackConsumerToken(observed, token, namespace, profile, policyID); err != nil {
		return nil, err
	}
	stackConsumerPreserveObservedExternalName(token, observed, stackConsumerTokenName)
	desired[stackConsumerTokenName] = token
	credentials := stackConsumerCredentials(namespace, slug, region, profile, settings)
	if err := validateObservedStackConsumerCredentials(observed, credentials, namespace, profile); err != nil {
		return nil, err
	}
	stackConsumerPreserveObservedExternalName(credentials, observed, stackConsumerCredentialsName)
	desired[stackConsumerCredentialsName] = credentials
	return desired, nil
}

func configuredStackConsumerProfile(config map[string]any, name, namespace, slug, region string) (stackConsumerProfile, error) {
	spec, _ := config["spec"].(map[string]any)
	profiles, _ := spec["stackConsumerProfiles"].([]any)
	var selected map[string]any
	for _, raw := range profiles {
		candidate, ok := raw.(map[string]any)
		if !ok || stringValue(candidate, "name", "") != name {
			continue
		}
		if selected != nil {
			return stackConsumerProfile{}, errors.New(stackConsumerIncompleteProfileMessage)
		}
		selected = candidate
	}
	if selected == nil {
		return stackConsumerProfile{}, errors.New(stackConsumerIncompleteProfileMessage)
	}
	profile := stackConsumerProfile{
		name:               name,
		namespace:          stringValue(selected, "namespace", ""),
		providerConfigName: stringValue(selected, "providerConfigName", ""),
		scopes:             nil,
	}
	consumer, _ := selected["consumer"].(map[string]any)
	profile.consumerName = stringValue(consumer, "name", "")
	profile.outputSecretPath = stringValue(consumer, "outputSecretPath", "")
	profile.scopes, _ = selected["scopes"].([]any)
	if profile.namespace == "" || profile.providerConfigName == "" || profile.consumerName == "" || profile.outputSecretPath == "" || len(profile.scopes) == 0 {
		return stackConsumerProfile{}, errors.New(stackConsumerIncompleteProfileMessage)
	}
	for _, scope := range profile.scopes {
		if value, ok := scope.(string); !ok || value == "" {
			return stackConsumerProfile{}, errors.New(stackConsumerIncompleteProfileMessage)
		}
	}
	consumerNames, outputPaths := 0, 0
	for _, raw := range profiles {
		candidate, _ := raw.(map[string]any)
		consumer, _ := candidate["consumer"].(map[string]any)
		if stringValue(consumer, "name", "") == profile.consumerName {
			consumerNames++
		}
		if stringValue(consumer, "outputSecretPath", "") == profile.outputSecretPath {
			outputPaths++
		}
	}
	if consumerNames != 1 || outputPaths != 1 {
		return stackConsumerProfile{}, errors.New(stackConsumerIncompleteProfileMessage)
	}
	if profile.namespace != namespace {
		return stackConsumerProfile{}, errors.New(stackConsumerNamespaceMessage)
	}
	targets, _ := selected["targets"].([]any)
	matched := false
	for _, raw := range targets {
		target, _ := raw.(map[string]any)
		targetSlug := stringValue(target, "slug", "")
		targetRegion := stringValue(target, "region", "")
		if targetSlug == "" || targetRegion == "" {
			return stackConsumerProfile{}, errors.New(stackConsumerIncompleteProfileMessage)
		}
		if targetSlug == slug && targetRegion == region {
			matched = true
		}
	}
	if matched {
		return profile, nil
	}
	return stackConsumerProfile{}, errors.New(stackConsumerTargetMessage)
}

func stackConsumerObserver(namespace, slug, region string, profile stackConsumerProfile) *resource.DesiredComposed {
	return newDesired("cloud.grafana.m.crossplane.io/v1alpha1", "Stack", namespace, profile.consumerName+"-observed-stack",
		map[string]any{"crossplane.io/external-name": slug}, map[string]any{
			"managementPolicies": []any{"Observe"},
			"forProvider":        map[string]any{"slug": slug, "regionSlug": region},
			"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": profile.providerConfigName},
		})
}

func stackConsumerPolicy(namespace, slug, region, stackID string, allowedSubnets []any, profile stackConsumerProfile) *resource.DesiredComposed {
	policyName := profile.consumerName + "-access-policy"
	forProvider := map[string]any{
		"displayName": "Stack consumer access for " + slug,
		"name":        policyName,
		"realm":       []any{map[string]any{"identifier": stackID, "type": "stack"}},
		"region":      region,
		"scopes":      profile.scopes,
	}
	if len(allowedSubnets) > 0 {
		forProvider["conditions"] = []any{map[string]any{"allowedSubnets": allowedSubnets}}
	}
	return newDesired("cloud.grafana.m.crossplane.io/v1alpha1", "AccessPolicy", namespace, policyName, nil, map[string]any{
		"managementPolicies": managementPolicies,
		"forProvider":        forProvider,
		"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": profile.providerConfigName},
	})
}

func stackConsumerToken(namespace, slug, region, policyID string, lifetime, rotationWindow time.Duration, profile stackConsumerProfile) *resource.DesiredComposed {
	tokenName := profile.consumerName + "-token"
	return newDesired("cloud.grafana.m.crossplane.io/v1alpha1", "AccessPolicyRotatingToken", namespace, tokenName, nil, map[string]any{
		"managementPolicies": managementPolicies,
		"forProvider": map[string]any{
			"accessPolicyId":      policyID,
			"deleteOnDestroy":     false,
			"displayName":         "Stack consumer token for " + slug,
			"earlyRotationWindow": tokenDurationString(rotationWindow),
			"expireAfter":         tokenDurationString(lifetime),
			"namePrefix":          profile.consumerName + "-",
			"region":              region,
		},
		"providerConfigRef":          map[string]any{"kind": "ProviderConfig", "name": profile.providerConfigName},
		"writeConnectionSecretToRef": map[string]any{"name": profile.consumerName + "-credentials"},
	})
}

func stackConsumerCredentials(namespace, slug, region string, profile stackConsumerProfile, settings platformSettings) *resource.DesiredComposed {
	policyName := profile.consumerName + "-access-policy"
	outputDocument := fmt.Sprintf(
		`{{ $token := index . "attribute.token" | toString }}{"stack_slug":%q,"stack_region":%q,"access_policy_name":%q,"access_policy_token":{{ $token | toJson }}}`,
		slug, region, policyName,
	)
	return newDesired("external-secrets.io/v1alpha1", "PushSecret", namespace, profile.consumerName+"-credentials", nil, map[string]any{
		"refreshInterval": "1h",
		"updatePolicy":    "Replace",
		"deletionPolicy":  "None",
		"secretStoreRefs": []any{settings.secretStoreReference()},
		"selector":        map[string]any{"secret": map[string]any{"name": profile.consumerName + "-credentials"}},
		"template": map[string]any{
			"engineVersion": "v2",
			"mergePolicy":   "Replace",
			"data":          map[string]any{"telemetry.json": outputDocument},
		},
		"data": []any{map[string]any{
			"match": map[string]any{
				"secretKey": "telemetry.json",
				"remoteRef": map[string]any{"remoteKey": profile.outputSecretPath},
			},
			"metadata": map[string]any{
				"apiVersion": "kubernetes.external-secrets.io/v1alpha1",
				"kind":       "PushSecretMetadata",
				"spec":       map[string]any{"secretPushFormat": "string", "tags": map[string]any{"grafana-cloud-vending-machine": "managed"}},
			},
		}},
	})
}

func observedStackConsumerIdentity(observed map[resource.Name]resource.ObservedComposed, expected *resource.DesiredComposed, namespace, slug, region string, profile stackConsumerProfile) (string, bool, error) {
	actual, exists := observed[stackConsumerObserverName]
	if !exists || actual.Resource == nil {
		return "", false, nil
	}
	if err := stackConsumerObservedChildMatches(actual.Resource.UnstructuredContent(), expected.Resource.UnstructuredContent(), "observed stack"); err != nil {
		return "", false, err
	}
	if got := stackConsumerObservedExternalName(actual.Resource.UnstructuredContent()); got != slug {
		return "", false, errors.Errorf("observed stack external name %q does not match authorized slug %q", got, slug)
	}
	id := observedString(observed, stackConsumerObserverName, "status.atProvider.id")
	if id == "" {
		return "", false, nil
	}
	if got := observedString(observed, stackConsumerObserverName, "status.atProvider.slug"); got != slug {
		return "", false, errors.Errorf("observed stack slug %q does not match authorized slug %q", got, slug)
	}
	if got := observedString(observed, stackConsumerObserverName, "status.atProvider.regionSlug"); got != region {
		return "", false, errors.Errorf("observed stack region %q does not match authorized region %q", got, region)
	}
	parsed, err := strconv.ParseInt(id, 10, 64)
	if err != nil || parsed <= 0 {
		return "", false, errors.Errorf("observed stack ID %q must be a positive numeric provider identity", id)
	}
	return id, true, nil
}

func observedStackConsumerPolicyID(observed map[resource.Name]resource.ObservedComposed, expected *resource.DesiredComposed, namespace, region string, profile stackConsumerProfile, stackID string) (string, bool, error) {
	actual, exists := observed[stackConsumerPolicyName]
	if !exists || actual.Resource == nil {
		return "", false, nil
	}
	if err := stackConsumerObservedChildMatches(actual.Resource.UnstructuredContent(), expected.Resource.UnstructuredContent(), "observed access policy"); err != nil {
		return "", false, err
	}
	id := observedString(observed, stackConsumerPolicyName, "status.atProvider.policyId")
	if id == "" {
		return "", false, nil
	}
	if externalName := stackConsumerObservedExternalName(actual.Resource.UnstructuredContent()); externalName != "" && externalName != region+":"+id {
		return "", false, errors.Errorf("observed access policy external name %q does not match its region and provider ID", externalName)
	}
	return id, true, nil
}

func validateObservedStackConsumerToken(observed map[resource.Name]resource.ObservedComposed, expected *resource.DesiredComposed, namespace string, profile stackConsumerProfile, policyID string) error {
	actual, exists := observed[stackConsumerTokenName]
	if !exists || actual.Resource == nil {
		return nil
	}
	return stackConsumerObservedChildMatches(actual.Resource.UnstructuredContent(), expected.Resource.UnstructuredContent(), "observed rotating token")
}

func validateObservedStackConsumerCredentials(observed map[resource.Name]resource.ObservedComposed, expected *resource.DesiredComposed, namespace string, profile stackConsumerProfile) error {
	actual, exists := observed[stackConsumerCredentialsName]
	if !exists || actual.Resource == nil {
		return nil
	}
	return stackConsumerObservedChildMatches(actual.Resource.UnstructuredContent(), expected.Resource.UnstructuredContent(), "observed credential publication")
}

func stackConsumerCredentialChildObserved(observed map[resource.Name]resource.ObservedComposed) bool {
	return observedExists(observed, stackConsumerPolicyName) || observedExists(observed, stackConsumerTokenName) || observedExists(observed, stackConsumerCredentialsName)
}

func stackConsumerPreserveObservedExternalName(desired *resource.DesiredComposed, observed map[resource.Name]resource.ObservedComposed, name resource.Name) {
	actual, exists := observed[name]
	if !exists || actual.Resource == nil {
		return
	}
	externalName := stackConsumerObservedExternalName(actual.Resource.UnstructuredContent())
	if externalName == "" {
		return
	}
	metadata, _ := desired.Resource.UnstructuredContent()["metadata"].(map[string]any)
	metadata["annotations"] = map[string]any{"crossplane.io/external-name": externalName}
}

func stackConsumerObservedExternalName(object map[string]any) string {
	annotations, _ := nestedStackConsumerValue(object, "metadata", "annotations")
	values, _ := annotations.(map[string]any)
	return stringValue(values, "crossplane.io/external-name", "")
}

// stackConsumerObservedChildMatches checks the routing and identity fields that
// make an observed child safe to keep. It intentionally excludes status because
// status is provider-owned; every request-controlled or profile-owned binding is
// compared before a provider ID can unlock the next credential-bearing stage.
func stackConsumerObservedChildMatches(actual, expected map[string]any, label string) error {
	for _, path := range [][]string{{"apiVersion"}, {"kind"}, {"metadata", "namespace"}, {"metadata", "name"}, {"spec"}} {
		actualValue, actualFound := nestedStackConsumerValue(actual, path...)
		expectedValue, expectedFound := nestedStackConsumerValue(expected, path...)
		if actualFound != expectedFound {
			return errors.Errorf("%s does not match its authorized identity and profile configuration at %s", label, stackConsumerPath(path))
		}
		if err := stackConsumerExpectedSubset(actualValue, expectedValue, stackConsumerPath(path)); err != nil {
			return errors.Wrap(err, label+" does not match its authorized identity and profile configuration")
		}
	}
	if label == "observed access policy" {
		realm, _ := nestedStackConsumerValue(actual, "spec", "forProvider", "realm")
		for _, raw := range realm.([]any) {
			value, _ := raw.(map[string]any)
			for _, forbidden := range []string{"stackRef", "stackSelector", "labelPolicy"} {
				if _, found := value[forbidden]; found {
					return errors.Errorf("%s has an unexpected identity-bearing realm field %q", label, forbidden)
				}
			}
		}
	}
	return nil
}

// stackConsumerExpectedSubset permits provider defaults and LateInitialize
// fields while retaining every authored identity, provider, output, and secret
// binding. Exact whole-spec comparisons would deadlock after a legitimate readback.
func stackConsumerExpectedSubset(actual, expected any, path string) error {
	switch want := expected.(type) {
	case map[string]any:
		got, ok := actual.(map[string]any)
		if !ok {
			return errors.Errorf("%s is not an object", path)
		}
		for key, expectedValue := range want {
			actualValue, found := got[key]
			if !found {
				return errors.Errorf("%s.%s is missing", path, key)
			}
			if err := stackConsumerExpectedSubset(actualValue, expectedValue, path+"."+key); err != nil {
				return err
			}
		}
	case []any:
		got, ok := actual.([]any)
		if !ok || len(got) != len(want) {
			return errors.Errorf("%s does not retain the authored list", path)
		}
		for index, expectedValue := range want {
			if err := stackConsumerExpectedSubset(got[index], expectedValue, fmt.Sprintf("%s[%d]", path, index)); err != nil {
				return err
			}
		}
	default:
		if !reflect.DeepEqual(actual, expected) {
			return errors.Errorf("%s changed", path)
		}
	}
	return nil
}

func nestedStackConsumerValue(object map[string]any, path ...string) (any, bool) {
	var value any = object
	for _, key := range path {
		mapValue, ok := value.(map[string]any)
		if !ok {
			return nil, false
		}
		value, ok = mapValue[key]
		if !ok {
			return nil, false
		}
	}
	return value, true
}

func stackConsumerPath(path []string) string {
	result := ""
	for index, item := range path {
		if index > 0 {
			result += "."
		}
		result += item
	}
	return result
}
