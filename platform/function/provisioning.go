package main

import (
	"strings"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

const provisioningRendererImplemented = true

// renderProvisioningRepository renders a Repository that references a
// separately credential-managed Grafana Connection. This keeps Git
// credentials out of the claim and out of the Repository resource while giving
// a folder subtree exactly one declarative dashboard owner.
func renderProvisioningRepository(xr map[string]any, _ map[resource.Name]resource.ObservedComposed, _ map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	spec, _ := xr["spec"].(map[string]any)
	metadata, _ := xr["metadata"].(map[string]any)
	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	stackRef, _ := spec["stackRef"].(map[string]any)
	stackName, _ := stackRef["name"].(string)
	repository, hasRepository := spec["repository"].(map[string]any)
	_, hasDashboard := spec["dashboard"]

	if hasRepository && hasDashboard {
		return nil, errors.New("provisioning repository cannot declare both repository and Crossplane dashboard ownership for one folder subtree")
	}
	if hasDashboard {
		return nil, errors.New("GrafanaProvisioningRepository does not render Crossplane Dashboards; use GrafanaCloudStackRequest for classic dashboard ownership")
	}
	if name == "" || namespace == "" || stackName == "" || !hasRepository {
		return nil, errors.New("provisioning repository must set metadata.name and namespace, spec.stackRef.name, and spec.repository")
	}

	repositoryUID, _ := repository["uid"].(string)
	title, _ := repository["title"].(string)
	description, _ := repository["description"].(string)
	url, _ := repository["url"].(string)
	branch, _ := repository["branch"].(string)
	path, _ := repository["path"].(string)
	connectionRef, _ := repository["connectionRef"].(map[string]any)
	connectionName, _ := connectionRef["name"].(string)
	if repositoryUID == "" || title == "" || url == "" || branch == "" || path == "" || connectionName == "" {
		return nil, errors.New("provisioning repository must set repository uid, title, url, branch, path, and connectionRef.name")
	}
	if !strings.HasPrefix(url, "https://") {
		return nil, errors.New("provisioning repository url must use https")
	}

	providerConfig := stackName + "-provider"
	desired := map[resource.Name]*resource.DesiredComposed{}
	desired["repository"] = newDesired("oss.grafana.m.crossplane.io/v1alpha1", "RepositoryV0Alpha1", namespace, name,
		map[string]any{"crossplane.io/external-name": repositoryUID}, map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider": map[string]any{
				"metadata": map[string]any{"uid": repositoryUID},
				"spec": map[string]any{
					"title": title, "description": description, "type": "github",
					"github":     map[string]any{"url": url, "branch": branch, "path": path},
					"connection": map[string]any{"name": connectionName},
					"sync":       map[string]any{"enabled": true, "target": "folder", "intervalSeconds": 60},
				},
			},
			"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": providerConfig},
		})
	return desired, nil
}
