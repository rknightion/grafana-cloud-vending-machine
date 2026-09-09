package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/crossplane/function-sdk-go/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestCloudIntegrationsRenderPlatformAccountsCredentialsAndJobs(t *testing.T) {
	observed := map[resource.Name]resource.ObservedComposed{
		"aws-account-example-account": observedComposed(`{"status":{"atProvider":{"resourceId":"provider-assigned-account-id"}}}`),
	}
	desired, err := renderCloudIntegrations(cloudIntegrationClaim(), observed, cloudIntegrationConfig())
	if err != nil {
		t.Fatalf("renderCloudIntegrations() error = %v", err)
	}
	for name, kind := range map[resource.Name]string{
		"cloud-integrations-provider-config":  "ProviderConfig",
		"aws-account-example-account":         "AwsAccount",
		"azure-credential-example-azure":      "AzureCredential",
		"cloud-integration-linux-node":        "CloudIntegration",
		"cloudwatch-cloudwatch-compute":       "AwsCloudwatchScrapeJob",
		"resourceMetadata-resource-inventory": "AwsResourceMetadataScrapeJob",
		"metricsEndpoint-endpoint-metrics":    "MetricsEndpointScrapeJob",
	} {
		if got := cloudIntegrationDesired(t, desired, name)["kind"]; got != kind {
			t.Errorf("%s kind = %v, want %s", name, got, kind)
		}
	}
	if got, want := len(desired), 7; got != want {
		t.Fatalf("rendered %d resources, want %d", got, want)
	}
	providerConfig := cloudIntegrationDesired(t, desired, "cloud-integrations-provider-config")
	if nestedMap(t, providerConfig, "spec")["cloudProviderUrl"] != "https://cloud-provider.example.invalid" {
		t.Fatal("observed Cloud Provider endpoint was not supplied")
	}
	if got, want := nestedMap(t, providerConfig, "spec", "credentials", "secretRef"), map[string]any{"name": "grafana-cloud-org-example-credentials", "namespace": "grafana-vending", "key": "credentials"}; !equalCloudIntegrationJSON(got, want) {
		t.Fatalf("cloud integrations credential reference = %#v, want %#v", got, want)
	}
	if got, want := nestedMap(t, providerConfig, "spec", "stackSecretRef"), map[string]any{"name": "teamdemo01-stack-details", "namespace": "grafana-vending"}; !equalCloudIntegrationJSON(got, want) {
		t.Fatalf("cloud integrations stack secret reference = %#v, want %#v", got, want)
	}
	for _, name := range []resource.Name{"aws-account-example-account", "azure-credential-example-azure", "cloudwatch-cloudwatch-compute", "resourceMetadata-resource-inventory", "metricsEndpoint-endpoint-metrics"} {
		if got, want := nestedMap(t, cloudIntegrationDesired(t, desired, name), "spec", "providerConfigRef"), map[string]any{"kind": "ProviderConfig", "name": "teamdemo01-cloud-integrations"}; !equalCloudIntegrationJSON(got, want) {
			t.Fatalf("%s provider config reference = %#v, want %#v", name, got, want)
		}
	}
	if got, want := nestedMap(t, cloudIntegrationDesired(t, desired, "cloud-integration-linux-node"), "spec", "providerConfigRef"), map[string]any{"kind": "ProviderConfig", "name": "teamdemo01"}; !equalCloudIntegrationJSON(got, want) {
		t.Fatalf("CloudIntegration provider config reference = %#v, want %#v", got, want)
	}

	account := cloudIntegrationDesired(t, desired, "aws-account-example-account")
	if annotations, ok := nestedMap(t, account, "metadata")["annotations"].(map[string]any); ok && annotations["crossplane.io/external-name"] != nil {
		t.Fatalf("AWS account guessed provider-assigned external name: %v", annotations["crossplane.io/external-name"])
	}
	if got, want := nestedMap(t, account, "spec", "forProvider")["stackId"], "12345"; got != want {
		t.Fatalf("account stack ID = %v, want %s", got, want)
	}

	cloudwatch := cloudIntegrationDesired(t, desired, "cloudwatch-cloudwatch-compute")
	if got, want := cloudIntegrationExternalNameOf(t, cloudwatch), "12345:cloudwatch-compute"; got != want {
		t.Fatalf("CloudWatch external name = %q, want %q", got, want)
	}
	cloudwatchParameters := nestedMap(t, cloudwatch, "spec", "forProvider")
	if got, want := cloudwatchParameters["awsAccountResourceId"], "provider-assigned-account-id"; got != want {
		t.Fatalf("CloudWatch account ID = %v, want observed provider ID %q", got, want)
	}
	service := cloudwatchParameters["service"].([]any)[0].(map[string]any)
	if got, want := service["scrapeIntervalSeconds"], 300; got != want {
		t.Fatalf("CloudWatch interval = %v, want %d", got, want)
	}
	resourceMetadata := cloudIntegrationDesired(t, desired, "resourceMetadata-resource-inventory")
	if got, want := cloudIntegrationExternalNameOf(t, resourceMetadata), "12345:resource-inventory"; got != want {
		t.Fatalf("resource metadata external name = %q, want %q", got, want)
	}
	if got, want := nestedMap(t, resourceMetadata, "spec", "forProvider")["awsAccountResourceId"], "provider-assigned-account-id"; got != want {
		t.Fatalf("resource metadata account ID = %v, want observed provider ID %q", got, want)
	}

	endpoint := cloudIntegrationDesired(t, desired, "metricsEndpoint-endpoint-metrics")
	if got, want := cloudIntegrationExternalNameOf(t, endpoint), "12345:endpoint-metrics"; got != want {
		t.Fatalf("endpoint external name = %q, want %q", got, want)
	}
	endpointParameters := nestedMap(t, endpoint, "spec", "forProvider")
	if got, want := endpointParameters["authenticationBearerTokenSecretRef"], map[string]any{"name": "approved-endpoint-credential", "key": "token"}; !equalCloudIntegrationJSON(got, want) {
		t.Fatalf("endpoint credential reference = %#v, want %#v", got, want)
	}
	if _, exists := endpointParameters["authenticationBearerToken"]; exists {
		t.Fatal("endpoint rendered a bearer token value rather than a Secret reference")
	}
	for name, child := range desired {
		encoded, err := json.Marshal(child.Resource.UnstructuredContent())
		if err != nil {
			t.Fatalf("marshal %s: %v", name, err)
		}
		if bytes.Contains(encoded, []byte("untrusted-credential-material")) {
			t.Fatalf("%s leaked credential material into rendered state: %s", name, encoded)
		}
	}
}

func TestCloudIntegrationsWaitForObservedAccountID(t *testing.T) {
	desired, err := renderCloudIntegrations(cloudIntegrationClaim(), nil, cloudIntegrationConfig())
	if err != nil {
		t.Fatalf("renderCloudIntegrations() error = %v", err)
	}
	if _, exists := desired["cloudwatch-cloudwatch-compute"]; exists {
		t.Fatal("CloudWatch job rendered before the provider-assigned account ID was observed")
	}
	for _, name := range []resource.Name{"cloud-integrations-provider-config", "aws-account-example-account", "azure-credential-example-azure", "cloud-integration-linux-node", "metricsEndpoint-endpoint-metrics"} {
		cloudIntegrationDesired(t, desired, name)
	}
}

func TestCloudIntegrationsPreserveExistingJobWhileAccountIDIsUnavailable(t *testing.T) {
	observed := map[resource.Name]resource.ObservedComposed{
		"cloudwatch-cloudwatch-compute": observedComposed(`{"apiVersion":"cloudprovider.grafana.m.crossplane.io/v1alpha1","kind":"AwsCloudwatchScrapeJob"}`),
	}
	_, err := renderCloudIntegrations(cloudIntegrationClaim(), observed, cloudIntegrationConfig())
	if err == nil || !strings.Contains(err.Error(), "preserving observed cloud integration job") {
		t.Fatalf("renderCloudIntegrations() error = %v, want observed-job preservation refusal", err)
	}
}

func TestCloudIntegrationsRejectBudgetAndProfileEscapes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any, map[string]any)
		want   string
	}{
		{"count", func(claim, _ map[string]any) {
			spec := claim["spec"].(map[string]any)
			spec["scrapeJobs"] = append(spec["scrapeJobs"].([]any), spec["scrapeJobs"].([]any)[0])
		}, "scrape job count 4 exceeds platform maximum 3"},
		{"interval", func(claim, _ map[string]any) {
			claim["spec"].(map[string]any)["scrapeJobs"].([]any)[0].(map[string]any)["scrapeIntervalSeconds"] = 30
		}, "platform minimum interval is 60"},
		{"usage", func(_ map[string]any, config map[string]any) {
			config["referencedStack"].(map[string]any)["usage"] = "production"
		}, "is not approved for referenced stack usage"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			claim, config := cloudIntegrationClaim(), cloudIntegrationConfig()
			if tc.name == "count" {
				config["spec"].(map[string]any)["cloudIntegrationProfiles"].([]any)[0].(map[string]any)["maxScrapeJobs"] = 3
			}
			tc.mutate(claim, config)
			_, err := renderCloudIntegrations(claim, nil, config)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("renderCloudIntegrations() error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestCloudIntegrationsRejectIncompleteOrganizationCredentialReference(t *testing.T) {
	claim, config := cloudIntegrationClaim(), cloudIntegrationConfig()
	delete(config["referencedStack"].(map[string]any), "organizationCredentialSecretRef")
	_, err := renderCloudIntegrations(claim, nil, config)
	if err == nil || !strings.Contains(err.Error(), "trusted referenced stack context is incomplete") {
		t.Fatalf("renderCloudIntegrations() error = %v, want incomplete trusted context rejection", err)
	}
}

func TestCloudIntegrationsAdmissionPolicyNegativeControl(t *testing.T) {
	original, err := os.ReadFile("../apis/cloud-integrations-v1beta1.yaml")
	if err != nil {
		t.Fatal(err)
	}
	policy := []byte("params.spec.pipeline[0].input.spec.cloudIntegrationProfiles.exists(profile, profile.name == object.spec.profile && object.spec.scrapeJobs.size() <= profile.maxScrapeJobs && object.spec.scrapeJobs.all(job, job.scrapeIntervalSeconds >= profile.minScrapeIntervalSeconds))")
	weakened := bytes.Replace(original, policy, []byte("true"), 1)
	if bytes.Equal(weakened, original) {
		t.Fatal("dynamic scrape-budget admission policy was not found")
	}
	assertCloudIntegrationBudgetRefused(t, []string{"../apis/cloud-integrations-v1beta1.yaml"}, "baseline")

	scratch := filepath.Join(t.TempDir(), "cloud-integrations-v1beta1.yaml")
	if err := os.WriteFile(scratch, weakened, 0o600); err != nil {
		t.Fatal(err)
	}
	weakenedEnv := &admissionEnv{paths: []string{scratch}}
	if err := weakenedEnv.Start(t); err != nil {
		t.Fatalf("start weakened admission environment: %v", err)
	}
	if err := cloudIntegrationWaitForAdmission(t, weakenedEnv, cloudIntegrationAdmissionRequest("weakened", 11), "", 10*time.Second); err != nil {
		t.Fatalf("weakened scrape budget did not admit over-budget request: %v", err)
	}
	t.Log("weakened dynamic scrape budget admitted 11 scrape jobs")
	if err := weakenedEnv.Stop(); err != nil {
		t.Fatalf("stop weakened admission environment: %v", err)
	}

	if err := os.WriteFile(scratch, original, 0o600); err != nil {
		t.Fatal(err)
	}
	assertCloudIntegrationBudgetRefused(t, []string{scratch}, "restored")
}

func TestCloudIntegrationsAdmissionSchemaScrapeJobCardinality(t *testing.T) {
	original, err := os.ReadFile("../apis/cloud-integrations-v1beta1.yaml")
	if err != nil {
		t.Fatal(err)
	}
	profileCap := []byte("maxScrapeJobs: 10")
	withStructuralBudgetVisible := bytes.Replace(original, profileCap, []byte("maxScrapeJobs: 20"), 1)
	if bytes.Equal(withStructuralBudgetVisible, original) {
		t.Fatal("cloud integration profile scrape-job budget was not found")
	}
	scratch := filepath.Join(t.TempDir(), "cloud-integrations-v1beta1.yaml")
	if err := os.WriteFile(scratch, withStructuralBudgetVisible, 0o600); err != nil {
		t.Fatal(err)
	}
	env := &admissionEnv{paths: []string{scratch}}
	if err := env.Start(t); err != nil {
		t.Fatalf("start structural cardinality admission environment: %v", err)
	}
	defer func() { _ = env.Stop() }()
	if err := cloudIntegrationWaitForAdmission(t, env, cloudIntegrationAdmissionRequest("cardinalitycontrol", 20), "", 10*time.Second); err != nil {
		t.Fatalf("20 scrape jobs were not admitted with the matching platform budget: %v", err)
	}
	err = cloudIntegrationWaitForAdmission(t, env, cloudIntegrationAdmissionRequest("cardinalityrefusal", 21), "must have at most 20 items", 10*time.Second)
	if err == nil || !strings.Contains(err.Error(), "must have at most 20 items") {
		t.Fatalf("21 scrape jobs were not refused by schema cardinality: %v", err)
	}
	t.Logf("schema cardinality refused 21 scrape jobs: %v", err)
}

func TestCloudIntegrationsAdmissionRulesAllowControlsAndRefuseUpdates(t *testing.T) {
	env := &admissionEnv{paths: []string{"../apis/cloud-integrations-v1beta1.yaml"}}
	if err := env.Start(t); err != nil {
		t.Fatalf("start admission environment: %v", err)
	}
	defer func() { _ = env.Stop() }()
	valid := cloudIntegrationAdmissionRequest("admission", 1)
	if err := cloudIntegrationWaitForAdmission(t, env, valid, "", 10*time.Second); err != nil {
		t.Fatalf("valid control was not admitted: %v", err)
	}
	t.Log("valid cloud integration request admitted")

	basic := cloudIntegrationAdmissionRequest("basicauth", 1)
	basicAuth := basic.Object["spec"].(map[string]any)["scrapeJobs"].([]any)[0].(map[string]any)["authentication"].(map[string]any)
	basicAuth["method"] = "basic"
	basicAuth["username"] = "metrics-reader"
	if err := cloudIntegrationWaitForAdmission(t, env, basic, "", 10*time.Second); err != nil {
		t.Fatalf("basic endpoint with username was not admitted: %v", err)
	}

	missingBasicUsername := cloudIntegrationAdmissionRequest("basicauthmissingusername", 1)
	missingBasicAuth := missingBasicUsername.Object["spec"].(map[string]any)["scrapeJobs"].([]any)[0].(map[string]any)["authentication"].(map[string]any)
	missingBasicAuth["method"] = "basic"
	err := cloudIntegrationWaitForAdmission(t, env, missingBasicUsername, "metricsEndpoint jobs using basic authentication must set authentication.username", 10*time.Second)
	if err == nil || !strings.Contains(err.Error(), "metricsEndpoint jobs using basic authentication must set authentication.username") {
		t.Fatalf("basic endpoint without username was not refused: %v", err)
	}
	t.Logf("basic endpoint without username refused with its own message: %v", err)

	basicUsernameRemoved := basic.DeepCopy()
	delete(basicUsernameRemoved.Object["spec"].(map[string]any)["scrapeJobs"].([]any)[0].(map[string]any)["authentication"].(map[string]any), "username")
	err = cloudIntegrationWaitForAdmission(t, env, basicUsernameRemoved, "metricsEndpoint jobs using basic authentication must set authentication.username", 10*time.Second)
	if err == nil || !strings.Contains(err.Error(), "metricsEndpoint jobs using basic authentication must set authentication.username") {
		t.Fatalf("removing a basic endpoint username was not refused: %v", err)
	}
	t.Logf("basic endpoint username removal refused with its own message: %v", err)

	invalidType := cloudIntegrationAdmissionRequest("invalidtype", 1)
	invalidJob := invalidType.Object["spec"].(map[string]any)["scrapeJobs"].([]any)[0].(map[string]any)
	invalidJob["account"] = "example-account"
	err = cloudIntegrationWaitForAdmission(t, env, invalidType, "cloudwatch and resourceMetadata jobs set account and services only", 10*time.Second)
	if err == nil || !strings.Contains(err.Error(), "cloudwatch and resourceMetadata jobs set account and services only") {
		t.Fatalf("invalid job discriminator was not refused: %v", err)
	}
	t.Logf("invalid job discriminator refused with its own message: %v", err)

	updated := valid.DeepCopy()
	updated.Object["spec"].(map[string]any)["stackRef"].(map[string]any)["name"] = "differentstack"
	err = cloudIntegrationWaitForAdmission(t, env, updated, "stackRef.name is immutable", 10*time.Second)
	if err == nil || !strings.Contains(err.Error(), "stackRef.name is immutable") {
		t.Fatalf("stack reference update was not refused: %v", err)
	}
	t.Logf("stack reference update refused with its own immutable message: %v", err)
}

func assertCloudIntegrationBudgetRefused(t *testing.T, paths []string, phase string) {
	t.Helper()
	env := &admissionEnv{paths: paths}
	if err := env.Start(t); err != nil {
		t.Fatalf("start %s admission environment: %v", phase, err)
	}
	defer func() { _ = env.Stop() }()
	err := cloudIntegrationWaitForAdmission(t, env, cloudIntegrationAdmissionRequest(phase, 11), "cloud integration request exceeds the selected platform scrape budget", 10*time.Second)
	if err == nil || !strings.Contains(err.Error(), "cloud integration request exceeds the selected platform scrape budget") {
		t.Fatalf("%s dynamic scrape budget was not refused: %v", phase, err)
	}
	t.Logf("%s dynamic scrape budget refused 11 scrape jobs: %v", phase, err)
}

func cloudIntegrationWaitForAdmission(t *testing.T, env *admissionEnv, request *unstructured.Unstructured, want string, timeout time.Duration) error {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last error
	for time.Now().Before(deadline) {
		last = env.Apply(context.Background(), request.DeepCopy())
		if want == "" && last == nil {
			return nil
		}
		if want != "" && last != nil && strings.Contains(last.Error(), want) {
			return last
		}
		time.Sleep(100 * time.Millisecond)
	}
	if want == "" {
		return fmt.Errorf("timed out waiting for admission; last error: %w", last)
	}
	return fmt.Errorf("timed out waiting for refusal %q; last error: %w", want, last)
}

func cloudIntegrationClaim() map[string]any {
	return map[string]any{
		"apiVersion": "platform.example.org/v1beta1", "kind": "GrafanaCloudIntegrations",
		"metadata": map[string]any{"name": "teamdemo01", "namespace": "grafana-vending"},
		"spec": map[string]any{
			"stackRef": map[string]any{"name": "teamdemo01"}, "profile": "development-standard",
			"scrapeJobs": []any{
				map[string]any{"name": "cloudwatch-compute", "type": "cloudwatch", "account": "example-account", "scrapeIntervalSeconds": 300, "services": []any{map[string]any{"name": "AWS/EC2", "metrics": []any{map[string]any{"name": "CPUUtilization", "statistics": []any{"Average"}}}}}},
				map[string]any{"name": "resource-inventory", "type": "resourceMetadata", "account": "example-account", "scrapeIntervalSeconds": 300, "services": []any{map[string]any{"name": "AWS/EC2"}}},
				map[string]any{"name": "endpoint-metrics", "type": "metricsEndpoint", "scrapeIntervalSeconds": 120, "url": "https://metrics.example.invalid/metrics", "authentication": map[string]any{"method": "bearer", "secretRef": map[string]any{"name": "approved-endpoint-credential", "key": "token"}, "untrustedCredential": "untrusted-credential-material"}},
			},
		},
	}
}

func cloudIntegrationConfig() map[string]any {
	return map[string]any{
		"referencedStack": map[string]any{"name": "teamdemo01", "namespace": "grafana-vending", "stackID": "12345", "usage": "development", "organizationProviderConfigName": "grafana-cloud-org-example", "cloudProviderURL": "https://cloud-provider.example.invalid",
			"organizationCredentialSecretRef": map[string]any{"name": "grafana-cloud-org-example-credentials", "namespace": "grafana-vending", "key": "credentials"}},
		"spec": map[string]any{"cloudIntegrationProfiles": []any{map[string]any{
			"name": "development-standard", "usage": "development", "maxScrapeJobs": 10, "minScrapeIntervalSeconds": 60,
			"awsAccounts":      []any{map[string]any{"name": "example-account", "roleArn": "replace-with-platform-role-arn", "regions": []any{"us-east-1"}}},
			"azureCredentials": []any{map[string]any{"name": "example-azure", "clientId": "platform-client-id", "tenantId": "platform-tenant-id", "clientSecretSecretRef": map[string]any{"name": "platform-azure-credential", "key": "client-secret"}, "untrustedCredential": "untrusted-credential-material"}},
			"integrations":     []any{map[string]any{"slug": "linux-node", "alertsEnabled": false}},
		}}},
	}
}

func cloudIntegrationDesired(t *testing.T, desired map[resource.Name]*resource.DesiredComposed, name resource.Name) map[string]any {
	t.Helper()
	child, exists := desired[name]
	if !exists {
		t.Fatalf("desired resource %q was not rendered", name)
	}
	return child.Resource.UnstructuredContent()
}

func cloudIntegrationExternalNameOf(t *testing.T, object map[string]any) string {
	t.Helper()
	value, _ := nestedMap(t, object, "metadata", "annotations")["crossplane.io/external-name"].(string)
	return value
}

func cloudIntegrationAdmissionRequest(name string, jobs int) *unstructured.Unstructured {
	requests := make([]any, 0, jobs)
	for i := 0; i < jobs; i++ {
		requests = append(requests, map[string]any{"name": fmt.Sprintf("job-%d", i), "type": "metricsEndpoint", "scrapeIntervalSeconds": int64(60), "url": "https://metrics.example.invalid/metrics", "authentication": map[string]any{"method": "bearer", "secretRef": map[string]any{"name": "approved-credential", "key": "token"}}})
	}
	return requestObject("GrafanaCloudIntegrations", name, map[string]any{"stackRef": map[string]any{"name": name}, "profile": "development-standard", "scrapeJobs": requests})
}

func equalCloudIntegrationJSON(got, want any) bool {
	gotJSON, gotErr := json.Marshal(got)
	wantJSON, wantErr := json.Marshal(want)
	return gotErr == nil && wantErr == nil && bytes.Equal(gotJSON, wantJSON)
}
