package main

import (
	"sort"
	"time"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

const onCallRendererImplemented = true

func renderOnCall(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	metadata, _ := xr["metadata"].(map[string]any)
	spec, _ := xr["spec"].(map[string]any)
	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	stackRef, _ := spec["stackRef"].(map[string]any)
	stackName, _ := stackRef["name"].(string)
	shiftStart, _ := spec["shiftStart"].(string)
	responders, _ := spec["responders"].([]any)
	if name == "" || namespace == "" || stackName == "" || shiftStart == "" || len(responders) == 0 {
		return nil, errors.New("on-call must set metadata name and namespace, stackRef.name, shiftStart, and at least one responder")
	}
	if _, err := time.Parse("2006-01-02T15:04:05", shiftStart); err != nil {
		return nil, errors.Wrap(err, "shiftStart must be a valid UTC date and time")
	}
	if name != stackName {
		return nil, errors.New("metadata.name must match spec.stackRef.name so one composite owns the stack on-call route")
	}

	type responder struct {
		key      resource.Name
		username string
	}
	resolved := make([]responder, 0, len(responders))
	seen := map[string]struct{}{}
	seenKeys := map[resource.Name]bool{}
	for index, item := range responders {
		entry, _ := item.(map[string]any)
		username, _ := entry["username"].(string)
		if username == "" {
			return nil, errors.Errorf("responders[%d].username must not be empty", index)
		}
		if _, exists := seen[username]; exists {
			return nil, errors.Errorf("responders contains duplicate username %q", username)
		}
		seen[username] = struct{}{}
		key := resource.Name("responder-" + stableResourceSuffix(username))
		if seenKeys[key] {
			return nil, errors.New("responders have colliding resource identities")
		}
		seenKeys[key] = true
		resolved = append(resolved, responder{key: key, username: username})
	}
	sort.Slice(resolved, func(i, j int) bool { return resolved[i].key < resolved[j].key })

	desired := map[resource.Name]*resource.DesiredComposed{}
	for _, responder := range resolved {
		desired[responder.key] = newDesired("oncall.grafana.o.crossplane.io/v1alpha1", "User", namespace, name+"-oncall-user-"+stableResourceSuffix(responder.username), nil,
			map[string]any{
				"managementPolicies": []any{"Observe"},
				"forProvider":        map[string]any{"username": responder.username},
				"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": stackName},
			})
	}

	rollingUsers := make([]any, 0, len(resolved))
	for _, responder := range resolved {
		id := observedExternalName(observed, responder.key)
		if id == "" {
			if onCallObservedDependent(observed, "rotating-shift", "schedule", "escalation-chain", "schedule-escalation", "integration", "catch-all-route") {
				return nil, errors.New("on-call responder identity was lost; refusing to withdraw dependent resources")
			}
			return desired, nil
		}
		// A rolling_users shift is a sequence of groups. Each group must contain
		// one responder: placing every identity in one group pages them all.
		rollingUsers = append(rollingUsers, []any{id})
	}

	shiftKey := resource.Name("rotating-shift")
	shiftName := name + "-rotating-shift"
	desired[shiftKey] = newDesired("oncall.grafana.m.crossplane.io/v1alpha1", "OnCallShift", namespace, shiftName, onCallExternalAnnotations(observed, shiftKey),
		map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider": map[string]any{
				"name": name + " rotating responders", "type": "rolling_users", "start": shiftStart,
				"duration": 604800, "frequency": "weekly", "interval": 1, "rollingUsers": rollingUsers,
			},
			"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": stackName},
		})
	shiftID := observedExternalName(observed, shiftKey)
	if shiftID == "" {
		if onCallObservedDependent(observed, "schedule", "escalation-chain", "schedule-escalation", "integration", "catch-all-route") {
			return nil, errors.New("on-call shift identity was lost; refusing to withdraw dependent resources")
		}
		return desired, nil
	}

	scheduleKey := resource.Name("schedule")
	scheduleName := name + "-schedule"
	desired[scheduleKey] = newDesired("oncall.grafana.m.crossplane.io/v1alpha1", "Schedule", namespace, scheduleName, onCallExternalAnnotations(observed, scheduleKey),
		map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider":        map[string]any{"name": name + " schedule", "type": "calendar", "timeZone": "UTC", "shifts": []any{shiftID}},
			"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": stackName},
		})
	scheduleID := observedExternalName(observed, scheduleKey)
	if scheduleID == "" {
		if onCallObservedDependent(observed, "escalation-chain", "schedule-escalation", "integration", "catch-all-route") {
			return nil, errors.New("on-call schedule identity was lost; refusing to withdraw dependent resources")
		}
		return desired, nil
	}

	chainKey := resource.Name("escalation-chain")
	chainName := name + "-escalation-chain"
	desired[chainKey] = newDesired("oncall.grafana.m.crossplane.io/v1alpha1", "EscalationChain", namespace, chainName, onCallExternalAnnotations(observed, chainKey),
		map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider":        map[string]any{"name": name + " responders"},
			"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": stackName},
		})
	chainID := observedExternalName(observed, chainKey)
	if chainID == "" {
		if onCallObservedDependent(observed, "schedule-escalation", "integration", "catch-all-route") {
			return nil, errors.New("on-call escalation chain identity was lost; refusing to withdraw dependent resources")
		}
		return desired, nil
	}

	desired["schedule-escalation"] = newDesired("oncall.grafana.m.crossplane.io/v1alpha1", "Escalation", namespace, name+"-schedule-escalation", onCallExternalAnnotations(observed, "schedule-escalation"),
		map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider": map[string]any{
				"escalationChainId": chainID, "notifyOnCallFromSchedule": scheduleID, "position": 0, "type": "notify_on_call_from_schedule",
			},
			"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": stackName},
		})
	desired["integration"] = newDesired("oncall.grafana.m.crossplane.io/v1alpha1", "Integration", namespace, name+"-inbound-email", onCallExternalAnnotations(observed, "integration"),
		map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider": map[string]any{
				"name": name + " inbound email", "type": "inbound_email", "defaultRoute": []any{map[string]any{"escalationChainId": chainID}},
			},
			"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": stackName},
		})
	integrationID := observedExternalName(observed, "integration")
	if integrationID == "" {
		if onCallObservedDependent(observed, "catch-all-route") {
			return nil, errors.New("on-call integration identity was lost; refusing to withdraw dependent resources")
		}
		return desired, nil
	}
	desired["catch-all-route"] = newDesired("oncall.grafana.m.crossplane.io/v1alpha1", "Route", namespace, name+"-catch-all-route", onCallExternalAnnotations(observed, "catch-all-route"),
		map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider": map[string]any{
				"integrationId": integrationID, "escalationChainId": chainID, "routingType": "regex", "routingRegex": ".*", "position": -1,
			},
			"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": stackName},
		})
	return desired, nil
}

// observedExternalName is the provider-assigned import identity. It must be
// observed after create, never made up from a composite request.
func observedExternalName(observed map[resource.Name]resource.ObservedComposed, name resource.Name) string {
	r, ok := observed[name]
	if !ok || r.Resource == nil {
		return ""
	}
	metadata, _ := r.Resource.UnstructuredContent()["metadata"].(map[string]any)
	annotations, _ := metadata["annotations"].(map[string]any)
	externalName, _ := annotations["crossplane.io/external-name"].(string)
	return externalName
}

func onCallExternalAnnotations(observed map[resource.Name]resource.ObservedComposed, name resource.Name) map[string]any {
	if value := observedExternalName(observed, name); value != "" {
		return map[string]any{"crossplane.io/external-name": value}
	}
	return nil
}

func onCallObservedDependent(observed map[resource.Name]resource.ObservedComposed, names ...resource.Name) bool {
	for _, name := range names {
		if _, found := observed[name]; found {
			return true
		}
	}
	return false
}
