package main

import (
	"fmt"
	"net/url"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

const cloudIntegrationsRendererImplemented = true

const (
	cloudProviderAPIVersion     = "cloudprovider.grafana.m.crossplane.io/v1alpha1"
	connectionsAPIVersion       = "connections.grafana.m.crossplane.io/v1alpha1"
	cloudIntegrationsAPIVersion = "cloudintegrations.grafana.m.crossplane.io/v1alpha1"
)

type cloudIntegrationBudget struct {
	maxJobs                  int
	minScrapeIntervalSeconds int
}

type cloudIntegrationProfile struct {
	budget           cloudIntegrationBudget
	awsAccounts      map[string]map[string]any
	azureCredentials []map[string]any
	integrations     []map[string]any
}

type cloudIntegrationJob struct {
	name                  string
	kind                  string
	account               string
	scrapeIntervalSeconds int
	services              []any
	url                   string
	authentication        map[string]any
}

func renderCloudIntegrations(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	metadata, _ := xr["metadata"].(map[string]any)
	spec, _ := xr["spec"].(map[string]any)
	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	stackRef, _ := spec["stackRef"].(map[string]any)
	stackName, _ := stackRef["name"].(string)
	profileName, _ := spec["profile"].(string)
	if name == "" || namespace == "" || stackName == "" || profileName == "" {
		return nil, errors.New("cloud integrations request must set metadata name and namespace plus stackRef.name and profile")
	}
	if name != stackName {
		return nil, errors.New("metadata.name must match spec.stackRef.name so one composite owns the stack integration set")
	}

	stack, available, err := cloudIntegrationStackContext(config, stackName, namespace)
	if err != nil {
		return nil, err
	}
	if !available {
		return map[resource.Name]*resource.DesiredComposed{}, nil
	}
	stackID := stringValue(stack, "stackID", "")
	usage := stringValue(stack, "usage", "")
	providerConfigName := stringValue(stack, "organizationProviderConfigName", "")
	organizationCredentialSecretRef, _ := stack["organizationCredentialSecretRef"].(map[string]any)
	if stackID == "" || usage == "" || providerConfigName == "" || stringValue(organizationCredentialSecretRef, "name", "") == "" || stringValue(organizationCredentialSecretRef, "namespace", "") == "" || stringValue(organizationCredentialSecretRef, "key", "") == "" {
		return nil, errors.New("trusted referenced stack context is incomplete")
	}

	cloudProviderURL := stringValue(stack, "cloudProviderURL", "")
	endpoint, endpointErr := url.Parse(cloudProviderURL)
	if endpointErr != nil || endpoint.Scheme != "https" || endpoint.Hostname() == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return nil, errors.New("trusted Cloud Provider endpoint must be an absolute HTTPS URL without credentials, query or fragment")
	}

	profile, err := configuredCloudIntegrationProfile(config, profileName, usage)
	if err != nil {
		return nil, err
	}
	jobs, err := requestedCloudIntegrationJobs(spec, profile)
	if err != nil {
		return nil, err
	}

	desired := map[resource.Name]*resource.DesiredComposed{}
	stackProviderConfigRef := map[string]any{"kind": "ProviderConfig", "name": stackName}
	cloudIntegrationsProviderConfigName := name + "-cloud-integrations"
	cloudIntegrationsProviderConfigRef := map[string]any{"kind": "ProviderConfig", "name": cloudIntegrationsProviderConfigName}
	desired["cloud-integrations-provider-config"] = newDesired(
		"grafana.m.crossplane.io/v1beta1", "ProviderConfig", namespace, cloudIntegrationsProviderConfigName, nil,
		map[string]any{
			"credentials": map[string]any{
				"source":    "Secret",
				"secretRef": organizationCredentialSecretRef,
			},
			"cloudProviderUrl": cloudProviderURL,
			"stackSecretRef":   map[string]any{"name": stackName + "-stack-details", "namespace": namespace},
		},
	)
	for accountName, account := range profile.awsAccounts {
		desired[resource.Name("aws-account-"+accountName)] = newDesired(
			cloudProviderAPIVersion, "AwsAccount", namespace, name+"-aws-account-"+accountName, nil,
			map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider": map[string]any{
					"name": accountName, "regions": account["regions"], "roleArn": account["roleArn"], "stackId": stackID,
				},
				"providerConfigRef": cloudIntegrationsProviderConfigRef,
			},
		)
	}
	for _, credential := range profile.azureCredentials {
		credentialName := stringValue(credential, "name", "")
		desired[resource.Name("azure-credential-"+credentialName)] = newDesired(
			cloudProviderAPIVersion, "AzureCredential", namespace, name+"-azure-credential-"+credentialName, nil,
			map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider": map[string]any{
					"name": credentialName, "clientId": credential["clientId"], "tenantId": credential["tenantId"],
					"clientSecretSecretRef": credential["clientSecretSecretRef"], "stackId": stackID,
				},
				"providerConfigRef": cloudIntegrationsProviderConfigRef,
			},
		)
	}
	for _, integration := range profile.integrations {
		slug := stringValue(integration, "slug", "")
		desired[resource.Name("cloud-integration-"+slug)] = newDesired(
			cloudIntegrationsAPIVersion, "CloudIntegration", namespace, name+"-cloud-integration-"+slug,
			map[string]any{"crossplane.io/external-name": slug},
			map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider":        map[string]any{"slug": slug, "alertsEnabled": integration["alertsEnabled"]},
				"providerConfigRef":  stackProviderConfigRef,
			},
		)
	}

	for _, job := range jobs {
		externalName := cloudIntegrationExternalName(stackID, job.name)
		logicalName := resource.Name(job.kind + "-" + job.name)
		switch job.kind {
		case "cloudwatch", "resourceMetadata":
			accountID := observedString(observed, resource.Name("aws-account-"+job.account), "status.atProvider.resourceId")
			if accountID == "" {
				// The account resource ID is assigned by Grafana Cloud. Jobs must
				// wait for it rather than derive an identifier from configuration.
				continue
			}
			parameters := map[string]any{
				"awsAccountResourceId": accountID, "enabled": true, "name": job.name, "service": job.services, "stackId": stackID,
			}
			if job.kind == "cloudwatch" {
				desired[logicalName] = newDesired(
					cloudProviderAPIVersion, "AwsCloudwatchScrapeJob", namespace, name+"-cloudwatch-"+job.name,
					map[string]any{"crossplane.io/external-name": externalName},
					map[string]any{"managementPolicies": managementPolicies, "forProvider": parameters, "providerConfigRef": cloudIntegrationsProviderConfigRef},
				)
			} else {
				desired[logicalName] = newDesired(
					cloudProviderAPIVersion, "AwsResourceMetadataScrapeJob", namespace, name+"-resource-metadata-"+job.name,
					map[string]any{"crossplane.io/external-name": externalName},
					map[string]any{"managementPolicies": managementPolicies, "forProvider": parameters, "providerConfigRef": cloudIntegrationsProviderConfigRef},
				)
			}
		case "metricsEndpoint":
			forProvider := map[string]any{
				"authenticationMethod": job.authentication["method"], "enabled": true, "name": job.name,
				"scrapeIntervalSeconds": job.scrapeIntervalSeconds, "stackId": stackID, "url": job.url,
			}
			secretRef, _ := job.authentication["secretRef"].(map[string]any)
			if job.authentication["method"] == "basic" {
				forProvider["authenticationBasicUsername"] = job.authentication["username"]
				forProvider["authenticationBasicPasswordSecretRef"] = secretRef
			} else {
				forProvider["authenticationBearerTokenSecretRef"] = secretRef
			}
			desired[logicalName] = newDesired(
				connectionsAPIVersion, "MetricsEndpointScrapeJob", namespace, name+"-metrics-endpoint-"+job.name,
				map[string]any{"crossplane.io/external-name": externalName},
				map[string]any{"managementPolicies": managementPolicies, "forProvider": forProvider, "providerConfigRef": cloudIntegrationsProviderConfigRef},
			)
		}
	}
	return desired, nil
}

func cloudIntegrationStackContext(config map[string]any, stackName, namespace string) (map[string]any, bool, error) {
	stack, ok := config["referencedStack"].(map[string]any)
	if !ok {
		return nil, false, nil
	}
	if stack["name"] != stackName || stack["namespace"] != namespace {
		return nil, false, errors.New("trusted referenced stack context does not match the request")
	}
	return stack, true, nil
}

func configuredCloudIntegrationProfile(config map[string]any, profileName, usage string) (cloudIntegrationProfile, error) {
	spec, _ := config["spec"].(map[string]any)
	profiles, _ := spec["cloudIntegrationProfiles"].([]any)
	for index, raw := range profiles {
		profile, ok := raw.(map[string]any)
		if !ok {
			return cloudIntegrationProfile{}, errors.Errorf("cloudIntegrationProfiles[%d] must be an object", index)
		}
		if profile["name"] != profileName {
			continue
		}
		if profile["usage"] != usage {
			return cloudIntegrationProfile{}, errors.Errorf("cloud integration profile %q is not approved for referenced stack usage %q", profileName, usage)
		}
		maxJobs := cloudIntegrationInteger(profile["maxScrapeJobs"])
		minInterval := cloudIntegrationInteger(profile["minScrapeIntervalSeconds"])
		if maxJobs < 1 || minInterval < 1 {
			return cloudIntegrationProfile{}, errors.Errorf("cloud integration profile %q has an incomplete scrape budget", profileName)
		}
		accounts, err := cloudIntegrationAccounts(profile["awsAccounts"])
		if err != nil {
			return cloudIntegrationProfile{}, errors.Wrapf(err, "cloud integration profile %q", profileName)
		}
		credentials, err := cloudIntegrationAzureCredentials(profile["azureCredentials"])
		if err != nil {
			return cloudIntegrationProfile{}, errors.Wrapf(err, "cloud integration profile %q", profileName)
		}
		integrations, err := cloudIntegrationIntegrations(profile["integrations"])
		if err != nil {
			return cloudIntegrationProfile{}, errors.Wrapf(err, "cloud integration profile %q", profileName)
		}
		return cloudIntegrationProfile{budget: cloudIntegrationBudget{maxJobs: maxJobs, minScrapeIntervalSeconds: minInterval}, awsAccounts: accounts, azureCredentials: credentials, integrations: integrations}, nil
	}
	return cloudIntegrationProfile{}, errors.Errorf("no cloud integration profile %q is configured for referenced stack usage %q", profileName, usage)
}

func cloudIntegrationAccounts(raw any) (map[string]map[string]any, error) {
	items, _ := raw.([]any)
	accounts := make(map[string]map[string]any, len(items))
	for index, rawAccount := range items {
		account, ok := rawAccount.(map[string]any)
		if !ok {
			return nil, errors.Errorf("awsAccounts[%d] must be an object", index)
		}
		name := stringValue(account, "name", "")
		roleARN := stringValue(account, "roleArn", "")
		regions := stringListValue(account, "regions", nil)
		if name == "" || roleARN == "" || len(regions) == 0 {
			return nil, errors.Errorf("awsAccounts[%d] must set name, roleArn, and regions", index)
		}
		if _, exists := accounts[name]; exists {
			return nil, errors.Errorf("awsAccounts contains duplicate account %q", name)
		}
		accounts[name] = account
	}
	return accounts, nil
}

func cloudIntegrationAzureCredentials(raw any) ([]map[string]any, error) {
	items, _ := raw.([]any)
	credentials := make([]map[string]any, 0, len(items))
	seen := map[string]struct{}{}
	for index, rawCredential := range items {
		credential, ok := rawCredential.(map[string]any)
		if !ok {
			return nil, errors.Errorf("azureCredentials[%d] must be an object", index)
		}
		name := stringValue(credential, "name", "")
		secretRef, _ := credential["clientSecretSecretRef"].(map[string]any)
		if name == "" || stringValue(credential, "clientId", "") == "" || stringValue(credential, "tenantId", "") == "" || stringValue(secretRef, "name", "") == "" || stringValue(secretRef, "key", "") == "" {
			return nil, errors.Errorf("azureCredentials[%d] must set name, clientId, tenantId, and clientSecretSecretRef", index)
		}
		if _, exists := seen[name]; exists {
			return nil, errors.Errorf("azureCredentials contains duplicate credential %q", name)
		}
		seen[name] = struct{}{}
		credentials = append(credentials, credential)
	}
	return credentials, nil
}

func cloudIntegrationIntegrations(raw any) ([]map[string]any, error) {
	items, _ := raw.([]any)
	integrations := make([]map[string]any, 0, len(items))
	seen := map[string]struct{}{}
	for index, rawIntegration := range items {
		integration, ok := rawIntegration.(map[string]any)
		if !ok {
			return nil, errors.Errorf("integrations[%d] must be an object", index)
		}
		slug := stringValue(integration, "slug", "")
		if slug == "" {
			return nil, errors.Errorf("integrations[%d] must set slug", index)
		}
		if _, exists := seen[slug]; exists {
			return nil, errors.Errorf("integrations contains duplicate slug %q", slug)
		}
		seen[slug] = struct{}{}
		integrations = append(integrations, integration)
	}
	return integrations, nil
}

func requestedCloudIntegrationJobs(spec map[string]any, profile cloudIntegrationProfile) ([]cloudIntegrationJob, error) {
	items, ok := spec["scrapeJobs"].([]any)
	if !ok {
		return nil, errors.New("spec.scrapeJobs must be an array")
	}
	if len(items) > profile.budget.maxJobs {
		return nil, errors.Errorf("scrape job count %d exceeds platform maximum %d", len(items), profile.budget.maxJobs)
	}
	jobs := make([]cloudIntegrationJob, 0, len(items))
	seen := map[string]struct{}{}
	for index, rawJob := range items {
		job, ok := rawJob.(map[string]any)
		if !ok {
			return nil, errors.Errorf("scrapeJobs[%d] must be an object", index)
		}
		name := stringValue(job, "name", "")
		kind := stringValue(job, "type", "")
		interval := cloudIntegrationInteger(job["scrapeIntervalSeconds"])
		if name == "" || interval < 1 {
			return nil, errors.Errorf("scrapeJobs[%d] must set name and scrapeIntervalSeconds", index)
		}
		if _, exists := seen[name]; exists {
			return nil, errors.Errorf("scrapeJobs contains duplicate job name %q", name)
		}
		seen[name] = struct{}{}
		if interval < profile.budget.minScrapeIntervalSeconds {
			return nil, errors.Errorf("scrape job %q runs every %d seconds; platform minimum interval is %d", name, interval, profile.budget.minScrapeIntervalSeconds)
		}
		parsed := cloudIntegrationJob{name: name, kind: kind, scrapeIntervalSeconds: interval}
		switch kind {
		case "cloudwatch", "resourceMetadata":
			parsed.account = stringValue(job, "account", "")
			if _, exists := profile.awsAccounts[parsed.account]; !exists {
				return nil, errors.Errorf("scrape job %q references account %q not supplied by the platform profile", name, parsed.account)
			}
			services, err := cloudIntegrationServices(job["services"], kind, interval)
			if err != nil {
				return nil, errors.Wrapf(err, "scrape job %q", name)
			}
			parsed.services = services
		case "metricsEndpoint":
			parsed.url = stringValue(job, "url", "")
			parsed.authentication, _ = job["authentication"].(map[string]any)
			if parsed.url == "" || parsed.authentication == nil {
				return nil, errors.Errorf("metrics endpoint scrape job %q must set url and authentication", name)
			}
			method := stringValue(parsed.authentication, "method", "")
			secretRef, _ := parsed.authentication["secretRef"].(map[string]any)
			if (method != "basic" && method != "bearer") || stringValue(secretRef, "name", "") == "" || stringValue(secretRef, "key", "") == "" {
				return nil, errors.Errorf("metrics endpoint scrape job %q must reference a basic or bearer credential", name)
			}
			if method == "basic" && stringValue(parsed.authentication, "username", "") == "" {
				return nil, errors.Errorf("metrics endpoint scrape job %q must set username for basic authentication", name)
			}
		default:
			return nil, errors.Errorf("scrape job %q has unsupported type %q", name, kind)
		}
		jobs = append(jobs, parsed)
	}
	return jobs, nil
}

func cloudIntegrationServices(raw any, kind string, interval int) ([]any, error) {
	items, ok := raw.([]any)
	if !ok || len(items) == 0 {
		return nil, errors.New("must set at least one service")
	}
	services := make([]any, 0, len(items))
	for index, rawService := range items {
		service, ok := rawService.(map[string]any)
		if !ok || stringValue(service, "name", "") == "" {
			return nil, errors.Errorf("services[%d] must set name", index)
		}
		result := map[string]any{"name": service["name"], "scrapeIntervalSeconds": interval}
		if filters, exists := service["resourceDiscoveryTagFilter"]; exists {
			result["resourceDiscoveryTagFilter"] = filters
		}
		if kind == "cloudwatch" {
			metrics, ok := service["metrics"].([]any)
			if !ok || len(metrics) == 0 {
				return nil, errors.Errorf("services[%d] must set metrics for a CloudWatch scrape job", index)
			}
			result["metric"] = metrics
		}
		services = append(services, result)
	}
	return services, nil
}

func cloudIntegrationInteger(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func cloudIntegrationExternalName(stackID, name string) string {
	return fmt.Sprintf("%s:%s", stackID, name)
}
