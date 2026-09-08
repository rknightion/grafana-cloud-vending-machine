package main

import (
	"encoding/base64"
	"fmt"
	"github.com/crossplane/function-sdk-go/resource"
	"strings"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
)

// serviceBootstrapConfig passes only identity-bound references, never credential
// values, from the stack and its existing bootstrap Secret to product renderers.
func serviceBootstrapConfig(req *fnv1.RunFunctionRequest, rsp *fnv1.RunFunctionResponse, xr, config map[string]any) map[string]any {
	kind, _ := xr["kind"].(string)
	if kind != "GrafanaK6Project" && kind != "GrafanaSyntheticMonitoring" {
		return config
	}
	result := make(map[string]any, len(config)+1)
	for key, value := range config {
		if key != "_resolvedK6Stack" && key != "referencedStack" {
			result[key] = value
		}
	}
	stacks, resolved, err := request.GetRequiredResource(req, referencedStackRequirement)
	if err != nil || !resolved || len(stacks) != 1 || stacks[0].Resource == nil {
		return result
	}
	stack := stacks[0].Resource.UnstructuredContent()
	meta, _ := xr["metadata"].(map[string]any)
	namespace, _ := meta["namespace"].(string)
	name := accessStackReference(xr)
	apiVersion, _ := xr["apiVersion"].(string)
	if !requiredStackIdentityMatches(stack, name, namespace, apiVersion) || !requiredStackReady(stack) {
		return result
	}
	spec, _ := stack["spec"].(map[string]any)
	status, _ := stack["status"].(map[string]any)
	identity, _ := status["stack"].(map[string]any)
	stackID := stringValue(identity, "id", "")
	output := stringValue(status, "outputSecretPath", "")
	if stackID == "" || output == "" || stringValue(spec, "usage", "") == "" {
		return result
	}
	secretName, secretKey := name+"-token", "attribute.key"
	if kind == "GrafanaSyntheticMonitoring" {
		secretName, secretKey = name+"-telemetry-token", "attribute.token"
	}
	if rsp.Requirements == nil {
		rsp.Requirements = &fnv1.Requirements{}
	}
	if rsp.Requirements.Resources == nil {
		rsp.Requirements.Resources = map[string]*fnv1.ResourceSelector{}
	}
	rsp.Requirements.Resources["service-bootstrap-token"] = &fnv1.ResourceSelector{ApiVersion: "v1", Kind: "Secret", Namespace: &namespace, Match: &fnv1.ResourceSelector_MatchName{MatchName: secretName}}
	secrets, resolved, err := request.GetRequiredResource(req, "service-bootstrap-token")
	if err != nil || !resolved || len(secrets) != 1 || secrets[0].Resource == nil {
		return result
	}
	secret := secrets[0].Resource.UnstructuredContent()
	smeta, _ := secret["metadata"].(map[string]any)
	data, _ := secret["data"].(map[string]any)
	encoded, _ := data[secretKey].(string)
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if secret["apiVersion"] != "v1" || secret["kind"] != "Secret" || smeta["name"] != secretName || smeta["namespace"] != namespace || smeta["deletionTimestamp"] != nil || err != nil || len(decoded) == 0 {
		return result
	}
	ref := map[string]any{"name": secretName, "key": secretKey, "ready": true}
	if kind == "GrafanaK6Project" {
		organization, err := configuredPlatformSettings(config).resolveOrganization(stringValue(spec, "organization", ""), stringValue(spec, "region", ""), stringValue(spec, "usage", ""))
		if err != nil {
			return result
		}
		result["_resolvedK6Stack"] = map[string]any{"organizationProviderConfigName": organization.providerConfigName, "usage": spec["usage"], "stackId": stackID, "outputSecretPath": output, "serviceAccountTokenSecret": ref}
	} else {
		stackMeta, _ := stack["metadata"].(map[string]any)
		result["referencedStack"] = map[string]any{"name": name, "namespace": namespace, "uid": stackMeta["uid"], "stackID": stackID, "region": spec["region"], "organization": spec["organization"], "usage": spec["usage"], "outputSecretPath": output, "bootstrapSecretRef": ref}
	}
	return result
}

// A temporary prerequisite failure must not withdraw an existing product tree.
// Explicitly removed team-authored checks remain removable.
func productWithdrawalError(kind string, xr map[string]any, desired map[resource.Name]*resource.DesiredComposed, observed map[resource.Name]resource.ObservedComposed) error {
	if kind != "GrafanaK6Project" && kind != "GrafanaSyntheticMonitoring" {
		return nil
	}
	wantedChecks := map[string]bool{}
	spec, _ := xr["spec"].(map[string]any)
	checks, _ := spec["checks"].([]any)
	for _, raw := range checks {
		if c, ok := raw.(map[string]any); ok {
			wantedChecks["check-"+stringValue(c, "name", "")] = true
		}
	}
	for name := range observed {
		if _, exists := desired[name]; exists {
			continue
		}
		if kind == "GrafanaSyntheticMonitoring" && (strings.HasPrefix(string(name), "synthetic-monitoring-verifier-") || (strings.HasPrefix(string(name), "check-") && !wantedChecks[string(name)])) {
			continue
		}
		return fmt.Errorf("product prerequisite unavailable for existing child %q; preserving composed resources", name)
	}
	return nil
}

// SetDesiredComposedResources merges with the incoming desired state. Remove
// only stale dynamic resources owned by this product before that merge.
func pruneSMDesired(rsp *fnv1.RunFunctionResponse, desired map[resource.Name]*resource.DesiredComposed) {
	if rsp.GetDesired() == nil {
		return
	}
	for name := range rsp.Desired.Resources {
		if _, exists := desired[resource.Name(name)]; exists {
			continue
		}
		if strings.HasPrefix(name, "check-") || strings.HasPrefix(name, "synthetic-monitoring-verifier-") {
			delete(rsp.Desired.Resources, name)
		}
	}
}
