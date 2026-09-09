package main

import (
	"fmt"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"net/url"
)

// observedSurfaceStackConfig supplies new product renderers with identity-bound
// stack facts. Composition input cannot override the provider-assigned stack ID.
func observedSurfaceStackConfig(req *fnv1.RunFunctionRequest, rsp *fnv1.RunFunctionResponse, xr, config map[string]any) (map[string]any, bool, error) {
	result := make(map[string]any, len(config)+1)
	for key, value := range config {
		if key != "referencedStack" {
			result[key] = value
		}
	}
	ready, err := accessResourcesAdmittedWithRequest(req, rsp, xr)
	if err != nil || !ready {
		return result, false, err
	}
	resources, resolved, err := request.GetRequiredResource(req, referencedStackRequirement)
	if err != nil || !resolved || len(resources) != 1 {
		return result, false, err
	}
	stack := resources[0].Resource.UnstructuredContent()
	meta, _ := stack["metadata"].(map[string]any)
	spec, _ := stack["spec"].(map[string]any)
	status, _ := stack["status"].(map[string]any)
	identity, _ := status["stack"].(map[string]any)
	id := stringValue(identity, "id", "")
	if id == "" {
		return result, false, nil
	}
	name := accessStackReference(xr)
	organizationProvider := ""
	var organizationSecret map[string]any
	cloudProviderURL := ""
	if xr["kind"] == "GrafanaCloudIntegrations" || xr["kind"] == "GrafanaPDC" || xr["kind"] == "GrafanaFrontendObservability" {
		organization, err := configuredPlatformSettings(config).resolveOrganization(stringValue(spec, "organization", ""), stringValue(spec, "region", ""), stringValue(spec, "usage", ""))
		if err != nil {
			return result, false, err
		}
		organizationProvider = organization.providerConfigName
	}
	if xr["kind"] == "GrafanaCloudIntegrations" {
		namespace := stringValue(meta, "namespace", "")
		if rsp.Requirements == nil {
			rsp.Requirements = &fnv1.Requirements{}
		}
		if rsp.Requirements.Resources == nil {
			rsp.Requirements.Resources = map[string]*fnv1.ResourceSelector{}
		}
		const key = "surface-organization-provider"
		rsp.Requirements.Resources[key] = &fnv1.ResourceSelector{ApiVersion: "grafana.m.crossplane.io/v1beta1", Kind: "ProviderConfig", Namespace: &namespace, Match: &fnv1.ResourceSelector_MatchName{MatchName: organizationProvider}}
		providers, available, err := request.GetRequiredResource(req, key)
		if err != nil || !available || len(providers) != 1 || providers[0].Resource == nil {
			return result, false, err
		}
		provider := providers[0].Resource.UnstructuredContent()
		pm, _ := provider["metadata"].(map[string]any)
		ps, _ := provider["spec"].(map[string]any)
		creds, _ := ps["credentials"].(map[string]any)
		ref, _ := creds["secretRef"].(map[string]any)
		if provider["apiVersion"] != "grafana.m.crossplane.io/v1beta1" || provider["kind"] != "ProviderConfig" || pm["name"] != organizationProvider || pm["namespace"] != namespace || pm["deletionTimestamp"] != nil || creds["source"] != "Secret" || ref["namespace"] != namespace || stringValue(ref, "name", "") == "" || stringValue(ref, "key", "") == "" {
			return result, false, fmt.Errorf("organization ProviderConfig identity or credential reference mismatch")
		}
		organizationSecret = ref
		const stackKey = "surface-managed-stack"
		rsp.Requirements.Resources[stackKey] = &fnv1.ResourceSelector{ApiVersion: "cloud.grafana.m.crossplane.io/v1alpha1", Kind: "Stack", Namespace: &namespace, Match: &fnv1.ResourceSelector_MatchName{MatchName: name}}
		managed, available, err := request.GetRequiredResource(req, stackKey)
		if err != nil || !available || len(managed) != 1 || managed[0].Resource == nil {
			return result, false, err
		}
		object := managed[0].Resource.UnstructuredContent()
		mm, _ := object["metadata"].(map[string]any)
		ms, _ := object["status"].(map[string]any)
		at, _ := ms["atProvider"].(map[string]any)
		annotations, _ := mm["annotations"].(map[string]any)
		if object["apiVersion"] != "cloud.grafana.m.crossplane.io/v1alpha1" || object["kind"] != "Stack" || mm["name"] != name || mm["namespace"] != namespace || mm["deletionTimestamp"] != nil || annotations["crossplane.io/external-name"] != name || stringValue(at, "id", "") != id || !requiredStackReady(object) {
			return result, false, fmt.Errorf("managed Stack identity or readiness mismatch")
		}
		cloudProviderURL = stringValue(at, "cloudProviderUrl", "")
		if cloudProviderURL == "" {
			return result, false, nil
		}
		endpoint, err := url.Parse(cloudProviderURL)
		if err != nil || endpoint.Scheme != "https" || endpoint.Hostname() == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
			return result, false, fmt.Errorf("observed Cloud Provider endpoint must be an absolute HTTPS URL without credentials, query or fragment")
		}

	}
	result["referencedStack"] = map[string]any{
		"name": name, "namespace": meta["namespace"], "uid": meta["uid"],
		"stackID": id, "providerConfigName": name, "usage": spec["usage"],
		"organization": spec["organization"], "region": spec["region"],
		"outputSecretPath":                status["outputSecretPath"],
		"organizationProviderConfigName":  organizationProvider,
		"organizationCredentialSecretRef": organizationSecret,
		"cloudProviderURL":                cloudProviderURL,
	}
	return result, true, nil
}
