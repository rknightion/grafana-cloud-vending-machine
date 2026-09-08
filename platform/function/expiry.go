package main

import (
	"fmt"
	"time"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

const expiryRendererImplemented = true

const expiryStatusConfigKey = "_expiryStatus"

type expiryPolicy struct {
	warningBefore time.Duration
	warningFolder string
}

// addStackExpiry records the effective expiry for an approved sandbox usage,
// creates a temporary alert through the stack's existing incident contact point,
// and reports whether the separately reviewed deletion path is ready. It never
// removes a request or changes its lifecycle intent.
func addStackExpiry(desired map[resource.Name]*resource.DesiredComposed, xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) error {
	spec, _ := xr["spec"].(map[string]any)
	rawExpiry, requested := spec["expiry"]
	if !requested {
		return nil
	}
	expiry, ok := rawExpiry.(map[string]any)
	if !ok {
		return errors.New("spec.expiry must be an object")
	}
	usage, _ := spec["usage"].(string)
	policy, err := expiryPolicyForUsage(config, usage)
	if err != nil {
		return err
	}
	effective, extensions, err := effectiveExpiry(expiry)
	if err != nil {
		return err
	}
	now, ok := config[reconcileTimeConfigKey].(time.Time)
	if !ok {
		return errors.New("expiry requires the injected reconcile time")
	}
	now = now.UTC()
	armed, deletionReady, err := expiryDeletionState(xr, observed, config)
	if err != nil {
		return err
	}
	warningActive := !now.Before(effective.Add(-policy.warningBefore)) && now.Before(effective)
	config[expiryStatusConfigKey] = map[string]any{
		"effectiveExpiresAt": effective.Format(time.RFC3339),
		"expired":            !now.Before(effective),
		"warningActive":      warningActive,
		"deletionArmed":      armed,
		"deletionReady":      deletionReady,
		"extensions":         extensions,
	}
	if warningActive {
		if err := addExpiryWarning(desired, xr, policy, effective); err != nil {
			return err
		}
	}
	return nil
}

func expiryPolicyForUsage(config map[string]any, usage string) (expiryPolicy, error) {
	spec, _ := config["spec"].(map[string]any)
	policies, ok := spec["expiryPolicies"].([]any)
	if !ok {
		return expiryPolicy{}, errors.Errorf("usage %q is not approved for expiry by platform configuration", usage)
	}
	for index, raw := range policies {
		policy, ok := raw.(map[string]any)
		if !ok {
			return expiryPolicy{}, errors.Errorf("expiryPolicies[%d] must be an object", index)
		}
		policyUsage, _ := policy["usage"].(string)
		if policyUsage != usage {
			continue
		}
		warningBefore, ok := policy["warningBefore"].(string)
		if !ok || warningBefore == "" {
			return expiryPolicy{}, errors.Errorf("expiryPolicies[%d].warningBefore must be a duration", index)
		}
		duration, err := time.ParseDuration(warningBefore)
		if err != nil || duration <= 0 {
			return expiryPolicy{}, errors.Errorf("expiryPolicies[%d].warningBefore must be a positive duration", index)
		}
		folder, _ := policy["warningFolderUID"].(string)
		if folder == "" {
			return expiryPolicy{}, errors.Errorf("expiryPolicies[%d].warningFolderUID must not be empty", index)
		}
		return expiryPolicy{warningBefore: duration, warningFolder: folder}, nil
	}
	return expiryPolicy{}, errors.Errorf("usage %q is not approved for expiry by platform configuration", usage)
}

func effectiveExpiry(expiry map[string]any) (time.Time, []any, error) {
	rawExpiresAt, _ := expiry["expiresAt"].(string)
	effective, err := time.Parse(time.RFC3339, rawExpiresAt)
	if err != nil {
		return time.Time{}, nil, errors.Wrap(err, "spec.expiry.expiresAt must be an RFC3339 timestamp")
	}
	declaredExtensions, ok := expiry["extensions"].([]any)
	if !ok && expiry["extensions"] != nil {
		return time.Time{}, nil, errors.New("spec.expiry.extensions must be an array")
	}
	copy := make([]any, 0, len(declaredExtensions))
	for index, raw := range declaredExtensions {
		extension, ok := raw.(map[string]any)
		if !ok {
			return time.Time{}, nil, errors.Errorf("spec.expiry.extensions[%d] must be an object", index)
		}
		extendedTo, _ := extension["extendedTo"].(string)
		if _, err := time.Parse(time.RFC3339, extendedTo); err != nil {
			return time.Time{}, nil, errors.Wrapf(err, "spec.expiry.extensions[%d].extendedTo must be an RFC3339 timestamp", index)
		}
		recordedAt, _ := extension["recordedAt"].(string)
		if _, err := time.Parse(time.RFC3339, recordedAt); err != nil {
			return time.Time{}, nil, errors.Wrapf(err, "spec.expiry.extensions[%d].recordedAt must be an RFC3339 timestamp", index)
		}
		reason, _ := extension["reason"].(string)
		requestedBy, _ := extension["requestedBy"].(string)
		if reason == "" || requestedBy == "" {
			return time.Time{}, nil, errors.Errorf("spec.expiry.extensions[%d] must set reason and requestedBy", index)
		}
		candidate, _ := time.Parse(time.RFC3339, extendedTo)
		if candidate.After(effective) {
			effective = candidate
		}
		copy = append(copy, map[string]any{
			"extendedTo": extendedTo, "reason": reason, "requestedBy": requestedBy, "recordedAt": recordedAt,
		})
	}
	return effective.UTC(), copy, nil
}

func expiryDeletionState(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) (bool, bool, error) {
	spec, _ := xr["spec"].(map[string]any)
	lifecycle, err := externalResourcesLifecycle(spec)
	if err != nil {
		return false, false, err
	}
	if lifecycle != "Delete" {
		return false, false, nil
	}
	metadata, _ := xr["metadata"].(map[string]any)
	namespace, _ := metadata["namespace"].(string)
	name, _ := metadata["name"].(string)
	uid, _ := metadata["uid"].(string)
	profile := stringValue(spec, "profile", "standard")
	if !configuredPlatformSettings(config).deletionAuthorized(namespace, name, uid, profile) {
		return false, false, errors.New("expiry cannot use an unauthorized deletion lifecycle")
	}
	armingPermitted, err := expiryArmingPermitted(xr, config)
	if err != nil {
		return false, false, err
	}
	if !armingPermitted {
		return false, false, nil
	}
	deleteProtection, protectionObserved := observedBool(observed, "stack", "status", "atProvider", "deleteProtection")
	credentialsPrepared := observedPushSecretDeletionPrepared(observed, "credentials")
	fleetCredentialsPrepared := observedPushSecretDeletionPrepared(observed, "fleet-management-credentials")
	telemetryPrepared := !telemetryAccessEnabled(spec) || observedPushSecretDeletionPrepared(observed, "telemetry-credentials")
	administratorTokenPrepared := observedRotatingTokenDeletionPrepared(observed, "stack-token")
	fleetTokenPrepared := observedRotatingTokenDeletionPrepared(observed, "fleet-management-token")
	telemetryTokenPrepared := !telemetryAccessEnabled(spec) || observedRotatingTokenDeletionPrepared(observed, "telemetry-token")
	ready := protectionObserved && !deleteProtection && administratorTokenPrepared && fleetTokenPrepared && telemetryTokenPrepared && credentialsPrepared && fleetCredentialsPrepared && telemetryPrepared
	return true, ready, nil
}

func addExpiryWarning(desired map[resource.Name]*resource.DesiredComposed, xr map[string]any, policy expiryPolicy, effective time.Time) error {
	contact, ok := desired["incident-alerting-production"]
	if !ok || contact.Resource == nil {
		return errors.New("expiry warning requires the already-vended production incident contact point")
	}
	contactPoint, err := contact.Resource.GetString("spec.forProvider.name")
	if err != nil || contactPoint == "" {
		return errors.New("expiry warning requires a named production incident contact point")
	}
	metadata, _ := xr["metadata"].(map[string]any)
	spec, _ := xr["spec"].(map[string]any)
	namespace, _ := metadata["namespace"].(string)
	slug, _ := spec["slug"].(string)
	if namespace == "" || slug == "" {
		return errors.New("expiry warning requires stack namespace and slug")
	}
	name := "stack-expiry-warning"
	rule := map[string]any{
		"name": "Stack expiry warning", "uid": "stack-expiry-warning", "condition": "A",
		"data": []any{map[string]any{
			"refId": "A", "queryType": "", "datasourceUid": "-100",
			"model":             `{"expression":"1","refId":"A","type":"math"}`,
			"relativeTimeRange": []any{map[string]any{"from": 0, "to": 0}},
		}},
		"noDataState": "NoData", "execErrState": "Alerting",
		"annotations":          map[string]any{"summary": fmt.Sprintf("Stack %s expires at %s", slug, effective.Format(time.RFC3339))},
		"labels":               map[string]any{"severity": "warning", "vending_expiry": "true"},
		"notificationSettings": []any{map[string]any{"contactPoint": contactPoint}},
	}
	desired["expiry-warning"] = newDesired("alerting.grafana.m.crossplane.io/v1alpha1", "RuleGroup", namespace, slug+"-expiry-warning",
		map[string]any{"crossplane.io/external-name": policy.warningFolder + ":" + name},
		map[string]any{
			"managementPolicies": []any{"*"},
			"forProvider": map[string]any{
				"name": name, "folderUid": policy.warningFolder, "intervalSeconds": 60, "rule": []any{rule},
			},
			"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": slug},
		})
	return nil
}

// mergeStackExpiryStatus is called by desiredStackStatus after the shared
// lifecycle status is computed. The helper preserves all declared extension
// fields; Kubernetes audit logs remain the source of authenticated actor and
// server timestamp evidence.
func mergeStackExpiryStatus(status, config map[string]any) {
	if expiry, ok := config[expiryStatusConfigKey].(map[string]any); ok {
		status["expiry"] = expiry
	}
}
