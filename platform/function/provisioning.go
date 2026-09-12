package main

import (
	"net/url"

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
	if hasRepository {
		repository = provisioningLegacyLadderRepository(repository)
	}

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
	repositoryType, _ := repository["type"].(string)
	connectionRef, _ := repository["connectionRef"].(map[string]any)
	connectionName, _ := connectionRef["name"].(string)
	if repositoryUID == "" || title == "" || repositoryType == "" || connectionName == "" {
		return nil, errors.New("provisioning repository must set repository uid, title, type, and connectionRef.name")
	}
	providerBlock, err := provisioningProviderBlock(repository, repositoryType)
	if err != nil {
		return nil, err
	}
	sync, err := provisioningSync(repository)
	if err != nil {
		return nil, err
	}
	workflows, err := provisioningWorkflows(repository)
	if err != nil {
		return nil, err
	}

	repositorySpec := map[string]any{
		"title": title, "type": repositoryType,
		repositoryType: providerBlock,
		"connection":   map[string]any{"name": connectionName},
		"sync":         sync,
		"workflows":    workflows,
	}
	if description, ok := repository["description"].(string); ok {
		repositorySpec["description"] = description
	}
	for _, key := range []string{"branch", "pullRequest", "commit", "webhook"} {
		if value, ok := repository[key].(map[string]any); ok {
			if key == "webhook" {
				baseURL, _ := value["baseUrl"].(string)
				if !provisioningHTTPSURL(baseURL) {
					return nil, errors.New("provisioning repository webhook.baseUrl must use https without URL userinfo")
				}
			}
			repositorySpec[key] = value
		}
	}
	forProvider := map[string]any{
		"metadata": map[string]any{"uid": repositoryUID},
		"spec":     repositorySpec,
	}
	if secure, ok := repository["secure"].(map[string]any); ok {
		renderedSecure, err := provisioningSecureValues(secure)
		if err != nil {
			return nil, err
		}
		forProvider["secure"] = renderedSecure
	}
	if secureVersion, ok := repository["secureVersion"]; ok {
		if _, ok := secureVersion.(float64); !ok {
			return nil, errors.New("provisioning repository secureVersion must be a number")
		}
		forProvider["secureVersion"] = secureVersion
	}

	providerConfig := stackName
	desired := map[resource.Name]*resource.DesiredComposed{}
	desired["repository"] = newDesired("oss.grafana.m.crossplane.io/v1alpha1", "RepositoryV0Alpha1", namespace, name,
		map[string]any{"crossplane.io/external-name": repositoryUID}, map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider":        forProvider,
			"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": providerConfig},
		})
	return desired, nil
}

// provisioningLegacyLadderRepository is an internal compatibility adapter for
// the stack ladder renderer. Public claims are structurally required to use
// the typed repository blocks; this preserves the ladder's fixed GitHub,
// folder-sync, read-only compilation until its own API is widened.
func provisioningLegacyLadderRepository(repository map[string]any) map[string]any {
	if _, typed := repository["type"]; typed {
		return repository
	}
	url, _ := repository["url"].(string)
	branch, _ := repository["branch"].(string)
	path, _ := repository["path"].(string)
	if url == "" || branch == "" || path == "" {
		return repository
	}
	adapted := map[string]any{}
	for key, value := range repository {
		adapted[key] = value
	}
	adapted["type"] = "github"
	adapted["github"] = map[string]any{"url": url, "branch": branch, "path": path}
	adapted["sync"] = map[string]any{"enabled": true, "target": "folder", "intervalSeconds": float64(60)}
	adapted["workflows"] = []any{}
	return adapted
}

func provisioningProviderBlock(repository map[string]any, repositoryType string) (map[string]any, error) {
	allowed := map[string]bool{"local": true, "github": true, "githubEnterprise": true, "git": true, "bitbucket": true, "gitlab": true}
	if !allowed[repositoryType] {
		return nil, errors.New("provisioning repository type is unsupported")
	}
	block, ok := repository[repositoryType].(map[string]any)
	if !ok {
		return nil, errors.New("provisioning repository type must set its matching provider block")
	}
	for providerType := range allowed {
		if providerType == repositoryType {
			continue
		}
		if _, found := repository[providerType]; found {
			return nil, errors.New("provisioning repository type must set exactly one matching provider block")
		}
	}
	if repositoryType == "local" {
		if path, _ := block["path"].(string); path == "" {
			return nil, errors.New("local provisioning repository must set path")
		}
		return block, nil
	}
	url, _ := block["url"].(string)
	branch, _ := block["branch"].(string)
	path, _ := block["path"].(string)
	if url == "" || branch == "" || path == "" {
		return nil, errors.New("remote provisioning repository must set url, branch, and path")
	}
	if !provisioningHTTPSURL(url) {
		return nil, errors.New("provisioning repository url must use https without URL userinfo")
	}
	if repositoryType == "githubEnterprise" {
		serverURL, _ := block["serverUrl"].(string)
		if !provisioningHTTPSURL(serverURL) {
			return nil, errors.New("GitHub Enterprise provisioning repository serverUrl must use https without URL userinfo")
		}
	}
	if repositoryType == "git" || repositoryType == "bitbucket" {
		if tokenUser, _ := block["tokenUser"].(string); tokenUser == "" {
			return nil, errors.New("basic-auth provisioning repository must set tokenUser")
		}
	}
	return block, nil
}

func provisioningHTTPSURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme == "https" && parsed.Host != "" && parsed.User == nil
}

func provisioningSync(repository map[string]any) (map[string]any, error) {
	sync, ok := repository["sync"].(map[string]any)
	if !ok {
		return nil, errors.New("provisioning repository must set sync")
	}
	if _, ok := sync["enabled"].(bool); !ok {
		return nil, errors.New("provisioning repository sync.enabled must be boolean")
	}
	target, _ := sync["target"].(string)
	if target == "instance" {
		return nil, errors.New("sync.target instance is refused because one declarative owner is allowed per folder subtree")
	}
	if target != "folder" && target != "folderless" {
		return nil, errors.New("provisioning repository sync.target must be folder or folderless")
	}
	interval, ok := sync["intervalSeconds"].(float64)
	if !ok || interval < 1 {
		return nil, errors.New("provisioning repository sync.intervalSeconds must be a positive number")
	}
	return sync, nil
}

func provisioningWorkflows(repository map[string]any) ([]any, error) {
	workflows, ok := repository["workflows"].([]any)
	if !ok {
		return nil, errors.New("provisioning repository must set workflows, including an explicit empty list for read-only")
	}
	for _, workflow := range workflows {
		value, ok := workflow.(string)
		if !ok || (value != "write" && value != "branch") {
			return nil, errors.New("provisioning repository workflows must contain only write or branch")
		}
	}
	return workflows, nil
}

func provisioningSecureValues(secure map[string]any) (map[string]any, error) {
	rendered := map[string]any{}
	for _, key := range []string{"token", "webhookSecret", "commitSigningKey"} {
		value, found := secure[key]
		if !found {
			continue
		}
		ref, ok := value.(map[string]any)
		if !ok {
			return nil, errors.New("provisioning repository secure values must be name references")
		}
		name, _ := ref["name"].(string)
		if name == "" || len(ref) != 1 {
			return nil, errors.New("provisioning repository secure values must contain only a name reference")
		}
		rendered[key] = map[string]any{"name": name}
	}
	if len(rendered) != len(secure) {
		return nil, errors.New("provisioning repository secure values must be token, webhookSecret, or commitSigningKey name references")
	}
	return rendered, nil
}
