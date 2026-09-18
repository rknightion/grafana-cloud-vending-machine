package main

import (
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
)

const credentialHealthConfigKey = "_credentialHealth"

type rotatingTokenKind struct {
	apiVersion string
	kind       string
}

type rotatingTokenMember struct {
	rotatingTokenKind
	name      string
	namespace string
}

var (
	stackServiceAccountRotatingToken = rotatingTokenKind{apiVersion: "cloud.grafana.m.crossplane.io/v1alpha1", kind: "StackServiceAccountRotatingToken"}
	accessPolicyRotatingToken        = rotatingTokenKind{apiVersion: "cloud.grafana.m.crossplane.io/v1alpha1", kind: "AccessPolicyRotatingToken"}
	serviceAccountRotatingToken      = rotatingTokenKind{apiVersion: "oss.grafana.m.crossplane.io/v1alpha1", kind: "ServiceAccountRotatingToken"}
)

// addCredentialHealth records the condition-only health of every configured
// rotating-token family. It deliberately does not inspect rendered children:
// a configured family that has not rendered a token is unhealthy. The caller
// owns applying the resulting composite readiness after its normal gates.
func addCredentialHealth(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) {
	expected, configured := configuredRotatingTokenMembers(xr, config)
	if !configured {
		delete(config, credentialHealthConfigKey)
		return
	}

	matched := map[rotatingTokenMember]bool{}
	healthy := true
	for _, child := range observed {
		member, ok := observedRotatingTokenMember(child)
		if !ok {
			continue
		}
		if _, expected := expected[member]; !expected {
			continue
		}
		matched[member] = true
		if !observedRotatingTokenSynced(child) {
			healthy = false
		}
	}
	// The expected count comes from the current request and platform input, not
	// from renderer output or the observed family count. Matching by current
	// identity prevents an old child from substituting for a newly configured one.
	if len(matched) != len(expected) {
		healthy = false
	}
	config[credentialHealthConfigKey] = healthy
}

// credentialHealthReady reports whether the current composite has configured
// rotating-token families and, if so, whether every observed token has
// Synced=True. A healthy result must leave Crossplane to aggregate readiness;
// only an unhealthy result overrides the composite to Ready=False.
func credentialHealthReady(config map[string]any) (healthy, configured bool) {
	healthy, configured = config[credentialHealthConfigKey].(bool)
	return healthy, configured
}

// applyCredentialHealthReadiness preserves normal Crossplane readiness
// aggregation for healthy children. A missing or unsynced configured token is
// the one case that must make the owning composite explicitly Ready=False.
func applyCredentialHealthReadiness(rsp *fnv1.RunFunctionResponse, config map[string]any) error {
	healthy, configured := credentialHealthReady(config)
	if !configured || healthy {
		return nil
	}
	return setAccessCompositeReadiness(rsp, false)
}

func configuredRotatingTokenMembers(xr, config map[string]any) (map[rotatingTokenMember]struct{}, bool) {
	kind, _ := xr["kind"].(string)
	spec, _ := xr["spec"].(map[string]any)
	metadata, _ := xr["metadata"].(map[string]any)
	name := stringValue(metadata, "name", "")
	namespace := stringValue(metadata, "namespace", "")
	switch kind {
	case "GrafanaCloudStackRequest":
		slug := stringValue(spec, "slug", "")
		if name == "" || namespace == "" || slug == "" {
			return nil, false
		}
		expected := map[rotatingTokenMember]struct{}{
			{rotatingTokenKind: stackServiceAccountRotatingToken, name: slug + "-admin", namespace: namespace}:     {},
			{rotatingTokenKind: accessPolicyRotatingToken, name: slug + "-fleet-management", namespace: namespace}: {},
		}
		if telemetryAccessEnabled(spec) {
			expected[rotatingTokenMember{rotatingTokenKind: accessPolicyRotatingToken, name: slug + "-telemetry-publisher", namespace: namespace}] = struct{}{}
		}
		return expected, true
	case "GrafanaStackConsumer":
		profileName := stringValue(spec, "profile", "")
		if namespace == "" || profileName == "" {
			return nil, false
		}
		input, _ := config["spec"].(map[string]any)
		profiles, _ := input["stackConsumerProfiles"].([]any)
		consumerName := ""
		for _, raw := range profiles {
			profile, _ := raw.(map[string]any)
			if stringValue(profile, "name", "") != profileName {
				continue
			}
			consumer, _ := profile["consumer"].(map[string]any)
			candidate := stringValue(consumer, "name", "")
			if candidate == "" || consumerName != "" {
				return nil, false
			}
			consumerName = candidate
		}
		if consumerName == "" {
			return nil, false
		}
		return map[rotatingTokenMember]struct{}{{rotatingTokenKind: accessPolicyRotatingToken, name: consumerName + "-token", namespace: namespace}: {}}, true
	case "GrafanaServiceAccounts":
		accounts, _ := spec["accounts"].([]any)
		if name == "" || namespace == "" || len(accounts) == 0 {
			return nil, false
		}
		expected := make(map[rotatingTokenMember]struct{}, len(accounts))
		for _, raw := range accounts {
			account, _ := raw.(map[string]any)
			accountName := stringValue(account, "name", "")
			if accountName == "" {
				return nil, false
			}
			expected[rotatingTokenMember{rotatingTokenKind: serviceAccountRotatingToken, name: name + "-" + accountName + "-token", namespace: namespace}] = struct{}{}
		}
		if len(expected) != len(accounts) {
			return nil, false
		}
		return expected, true
	default:
		return nil, false
	}
}

func observedRotatingTokenMember(child resource.ObservedComposed) (rotatingTokenMember, bool) {
	if child.Resource == nil {
		return rotatingTokenMember{}, false
	}
	object := child.Resource.UnstructuredContent()
	actual := rotatingTokenKind{apiVersion: stringValue(object, "apiVersion", ""), kind: stringValue(object, "kind", "")}
	for _, expected := range []rotatingTokenKind{stackServiceAccountRotatingToken, accessPolicyRotatingToken, serviceAccountRotatingToken} {
		if actual == expected {
			metadata, _ := object["metadata"].(map[string]any)
			name := stringValue(metadata, "name", "")
			namespace := stringValue(metadata, "namespace", "")
			if name == "" || namespace == "" {
				return rotatingTokenMember{}, false
			}
			return rotatingTokenMember{rotatingTokenKind: actual, name: name, namespace: namespace}, true
		}
	}
	return rotatingTokenMember{}, false
}

func observedRotatingTokenSynced(child resource.ObservedComposed) bool {
	object := child.Resource.UnstructuredContent()
	status, _ := object["status"].(map[string]any)
	conditions, _ := status["conditions"].([]any)
	for _, raw := range conditions {
		condition, _ := raw.(map[string]any)
		if condition["type"] == "Synced" {
			return condition["status"] == "True"
		}
	}
	return false
}
