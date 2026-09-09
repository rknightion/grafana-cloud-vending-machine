package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestStackConsumerWaitsForTrustedObservedStackIdentity(t *testing.T) {
	desired, err := renderStackConsumer(stackConsumerClaim(), nil, stackConsumerConfig())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(desired), 1; got != want {
		t.Fatalf("children without an observed stack ID = %d, want only the non-creating observer", got)
	}
	stack := stackConsumerDesired(t, desired, stackConsumerObserverName)
	if got := stack["kind"]; got != "Stack" {
		t.Fatalf("bootstrap child kind = %v, want Stack", got)
	}
	if got := nestedMap(t, stack, "metadata", "annotations")["crossplane.io/external-name"]; got != "exampleobservedstack" {
		t.Fatalf("observer external name = %v, want platform-authorized slug", got)
	}
	spec := nestedMap(t, stack, "spec")
	if got := spec["managementPolicies"]; mustJSON(got) != mustJSON([]any{"Observe"}) {
		t.Fatalf("observer management policies = %s, want Observe only", mustJSON(got))
	}
	if got := nestedMap(t, stack, "spec", "forProvider")["regionSlug"]; got != "prod-us-central-0" {
		t.Fatalf("observer region = %v, want request region", got)
	}
	for _, name := range []resource.Name{stackConsumerPolicyName, stackConsumerTokenName, stackConsumerCredentialsName} {
		if _, ok := desired[name]; ok {
			t.Fatalf("credential-bearing child %q rendered before trusted observed stack identity", name)
		}
	}
	observed := map[resource.Name]resource.ObservedComposed{stackConsumerObserverName: stackConsumerObserved(stack, nil)}
	desired, err = renderStackConsumer(stackConsumerClaim(), observed, stackConsumerConfig())
	if err != nil {
		t.Fatalf("unobserved stack object returned an error instead of waiting: %v", err)
	}
	if got, want := len(desired), 1; got != want {
		t.Fatalf("children with an observer awaiting status = %d, want %d", got, want)
	}
}

func TestStackConsumerBindsProfileOwnedCredentialConfiguration(t *testing.T) {
	bootstrap, err := renderStackConsumer(stackConsumerClaim(), nil, stackConsumerConfig())
	if err != nil {
		t.Fatal(err)
	}
	observed := map[resource.Name]resource.ObservedComposed{
		stackConsumerObserverName: stackConsumerObserved(stackConsumerDesired(t, bootstrap, stackConsumerObserverName), map[string]any{"status": map[string]any{"atProvider": map[string]any{"id": "12345", "slug": "exampleobservedstack", "regionSlug": "prod-us-central-0"}}}),
	}
	policyStage, err := renderStackConsumer(stackConsumerClaim(), observed, stackConsumerConfig())
	if err != nil {
		t.Fatal(err)
	}
	policy := stackConsumerDesired(t, policyStage, stackConsumerPolicyName)
	parameters := nestedMap(t, policy, "spec", "forProvider")
	realm := parameters["realm"].([]any)[0].(map[string]any)
	if got, want := realm["identifier"], "12345"; got != want {
		t.Fatalf("policy realm identifier = %v, want observed stack ID %q", got, want)
	}
	if got, want := realm["type"], "stack"; got != want {
		t.Fatalf("policy realm type = %v, want %q", got, want)
	}
	if _, hasRef := realm["stackRef"]; hasRef {
		t.Fatal("consumer policy retained a local Stack reference")
	}
	if got := nestedMap(t, policy, "spec", "providerConfigRef")["name"]; got != "grafana-cloud-org-example-primary" {
		t.Fatalf("policy provider configuration = %v, want profile-owned provider", got)
	}
	if got := parameters["scopes"]; mustJSON(got) != mustJSON([]any{"metrics:read"}) {
		t.Fatalf("policy scopes = %s, want profile-owned scopes", mustJSON(got))
	}

	policy["metadata"].(map[string]any)["annotations"] = map[string]any{"crossplane.io/external-name": "prod-us-central-0:67890"}
	observed[stackConsumerPolicyName] = stackConsumerObserved(policy, map[string]any{"status": map[string]any{"atProvider": map[string]any{"policyId": "67890"}}})
	ready, err := renderStackConsumer(stackConsumerClaim(), observed, stackConsumerConfig())
	if err != nil {
		t.Fatal(err)
	}
	token := stackConsumerDesired(t, ready, stackConsumerTokenName)
	if got := nestedMap(t, token, "spec", "forProvider")["accessPolicyId"]; got != "67890" {
		t.Fatalf("rotating token access policy ID = %v, want observed policy ID", got)
	}
	if got := nestedMap(t, token, "spec", "forProvider")["deleteOnDestroy"]; got != false {
		t.Fatalf("rotating token deleteOnDestroy = %v, want false", got)
	}
	publication := stackConsumerDesired(t, ready, stackConsumerCredentialsName)
	if got := nestedMap(t, publication, "spec")["data"].([]any)[0].(map[string]any)["match"].(map[string]any)["remoteRef"].(map[string]any)["remoteKey"]; got != "/platform/grafana-cloud/consumers/example-consumer-a" {
		t.Fatalf("credential output = %v, want profile-owned output path", got)
	}
	if got := nestedMap(t, publication, "spec")["deletionPolicy"]; got != "None" {
		t.Fatalf("credential deletion policy = %v, want None", got)
	}
	if _, found := nestedMap(t, publication, "spec", "template", "data")["telemetry.json"]; !found {
		t.Fatal("credential publication does not use the established telemetry.json output document contract")
	}
	if got := nestedMap(t, stackConsumerDesired(t, ready, stackConsumerPolicyName), "metadata", "annotations")["crossplane.io/external-name"]; got != "prod-us-central-0:67890" {
		t.Fatalf("observed policy external name = %v, want preserved provider identity", got)
	}
}

func TestStackConsumerInheritsNetworkProfileAndRejectsMalformedSubnet(t *testing.T) {
	config := stackConsumerConfig()
	config["spec"].(map[string]any)["tokenUseNetworkProfiles"] = []any{map[string]any{"name": "example-reader", "allowedSubnets": []any{"192.0.2.0/24"}}}
	bootstrap, err := renderStackConsumer(stackConsumerClaim(), nil, config)
	if err != nil {
		t.Fatal(err)
	}
	observed := map[resource.Name]resource.ObservedComposed{
		stackConsumerObserverName: stackConsumerObserved(stackConsumerDesired(t, bootstrap, stackConsumerObserverName), map[string]any{"status": map[string]any{"atProvider": map[string]any{"id": "12345", "slug": "exampleobservedstack", "regionSlug": "prod-us-central-0"}}}),
	}
	desired, err := renderStackConsumer(stackConsumerClaim(), observed, config)
	if err != nil {
		t.Fatal(err)
	}
	conditionsValue, found := nestedMap(t, stackConsumerDesired(t, desired, stackConsumerPolicyName), "spec", "forProvider")["conditions"]
	if !found {
		t.Fatal("profile-owned allowedSubnets were not rendered")
	}
	conditions, ok := conditionsValue.([]any)
	if !ok {
		t.Fatalf("policy conditions = %T, want list", conditionsValue)
	}
	if len(conditions) != 1 {
		t.Fatalf("policy conditions = %d entries, want one profile-owned condition", len(conditions))
	}
	if got := conditions[0].(map[string]any)["allowedSubnets"]; mustJSON(got) != mustJSON([]any{"192.0.2.0/24"}) {
		t.Fatalf("policy allowedSubnets = %s, want profile-owned CIDR", mustJSON(got))
	}
	config["spec"].(map[string]any)["tokenUseNetworkProfiles"] = []any{map[string]any{"name": "example-reader", "allowedSubnets": []any{"not-a-cidr"}}}
	if desired, err := renderStackConsumer(stackConsumerClaim(), nil, config); err == nil || !strings.Contains(err.Error(), "not a valid CIDR") {
		t.Fatalf("malformed subnet error = %v, want CIDR refusal", err)
	} else if desired != nil {
		t.Fatalf("malformed subnet rendered %d children", len(desired))
	}
}

func TestStackConsumerRefusesUntrustedIdentityAndObservedWithdrawal(t *testing.T) {
	bootstrap, err := renderStackConsumer(stackConsumerClaim(), nil, stackConsumerConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name     string
		observed map[resource.Name]resource.ObservedComposed
		message  string
	}{
		{
			name: "non-numeric stack ID",
			observed: map[resource.Name]resource.ObservedComposed{
				stackConsumerObserverName: stackConsumerObserved(stackConsumerDesired(t, bootstrap, stackConsumerObserverName), map[string]any{"status": map[string]any{"atProvider": map[string]any{"id": "not-a-number", "slug": "exampleobservedstack", "regionSlug": "prod-us-central-0"}}}),
			},
			message: "positive numeric",
		},
		{
			name: "zero stack ID",
			observed: map[resource.Name]resource.ObservedComposed{
				stackConsumerObserverName: stackConsumerObserved(stackConsumerDesired(t, bootstrap, stackConsumerObserverName), map[string]any{"status": map[string]any{"atProvider": map[string]any{"id": "0", "slug": "exampleobservedstack", "regionSlug": "prod-us-central-0"}}}),
			},
			message: "positive numeric",
		},
		{
			name: "retargeted observer",
			observed: map[resource.Name]resource.ObservedComposed{
				stackConsumerObserverName: stackConsumerObserved(stackConsumerDesired(t, bootstrap, stackConsumerObserverName), map[string]any{"spec": map[string]any{"managementPolicies": []any{"Observe"}, "forProvider": map[string]any{"slug": "otherstack", "regionSlug": "prod-us-central-0"}, "providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": "grafana-cloud-org-example-primary"}}, "status": map[string]any{"atProvider": map[string]any{"id": "12345", "slug": "otherstack", "regionSlug": "prod-us-central-0"}}}),
			},
			message: "does not match",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			desired, err := renderStackConsumer(stackConsumerClaim(), tc.observed, stackConsumerConfig())
			if err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("render error = %v, want %q", err, tc.message)
			}
			if desired != nil {
				t.Fatalf("untrusted identity rendered %d children", len(desired))
			}
		})
	}

	valid := map[resource.Name]resource.ObservedComposed{
		stackConsumerObserverName: stackConsumerObserved(stackConsumerDesired(t, bootstrap, stackConsumerObserverName), map[string]any{"status": map[string]any{"atProvider": map[string]any{"id": "12345", "slug": "exampleobservedstack", "regionSlug": "prod-us-central-0"}}}),
	}
	policyStage, err := renderStackConsumer(stackConsumerClaim(), valid, stackConsumerConfig())
	if err != nil {
		t.Fatal(err)
	}
	valid[stackConsumerPolicyName] = stackConsumerObserved(stackConsumerDesired(t, policyStage, stackConsumerPolicyName), map[string]any{"status": map[string]any{"atProvider": map[string]any{"policyId": "67890"}}})
	ready, err := renderStackConsumer(stackConsumerClaim(), valid, stackConsumerConfig())
	if err != nil {
		t.Fatal(err)
	}
	retargetedToken := stackConsumerDesired(t, ready, stackConsumerTokenName)
	nestedMap(t, retargetedToken, "spec", "providerConfigRef")["name"] = "different-provider"
	valid[stackConsumerTokenName] = stackConsumerObserved(retargetedToken, nil)
	if desired, err := renderStackConsumer(stackConsumerClaim(), valid, stackConsumerConfig()); err == nil || !strings.Contains(err.Error(), "observed rotating token") {
		t.Fatalf("withdrawal-protection error = %v, want observed token mismatch", err)
	} else if desired != nil {
		t.Fatalf("retargeted observed token rendered %d replacement children", len(desired))
	}

	redirectedPublication := stackConsumerDesired(t, ready, stackConsumerCredentialsName)
	redirectedPublication["spec"].(map[string]any)["data"].([]any)[0].(map[string]any)["match"].(map[string]any)["remoteRef"].(map[string]any)["remoteKey"] = "/not-platform-owned"
	valid[stackConsumerTokenName] = stackConsumerObserved(stackConsumerToken("consumer-platform", "exampleobservedstack", "prod-us-central-0", "67890", 720*time.Hour, 168*time.Hour, stackConsumerProfile{name: "example-reader", namespace: "consumer-platform", providerConfigName: "grafana-cloud-org-example-primary", consumerName: "example-consumer-a", outputSecretPath: "/platform/grafana-cloud/consumers/example-consumer-a", scopes: []any{"metrics:read"}}).Resource.UnstructuredContent(), nil)
	valid[stackConsumerCredentialsName] = stackConsumerObserved(redirectedPublication, nil)
	if desired, err := renderStackConsumer(stackConsumerClaim(), valid, stackConsumerConfig()); err == nil || !strings.Contains(err.Error(), "observed credential publication") {
		t.Fatalf("publication redirect error = %v, want output binding refusal", err)
	} else if desired != nil {
		t.Fatalf("redirected publication rendered %d replacement children", len(desired))
	}

	if desired, err := renderStackConsumer(stackConsumerClaim(), map[resource.Name]resource.ObservedComposed{stackConsumerPolicyName: valid[stackConsumerPolicyName]}, stackConsumerConfig()); err == nil || !strings.Contains(err.Error(), "would be withdrawn") {
		t.Fatalf("identity withdrawal error = %v, want observed child preservation refusal", err)
	} else if desired != nil {
		t.Fatalf("identity withdrawal rendered %d children", len(desired))
	}
}

func TestStackConsumerRejectsIncompletePlatformProfile(t *testing.T) {
	config := stackConsumerConfig()
	profile := config["spec"].(map[string]any)["stackConsumerProfiles"].([]any)[0].(map[string]any)
	delete(profile, "consumer")
	desired, err := renderStackConsumer(stackConsumerClaim(), nil, config)
	if err == nil || !strings.Contains(err.Error(), stackConsumerIncompleteProfileMessage) {
		t.Fatalf("incomplete profile error = %v, want %q", err, stackConsumerIncompleteProfileMessage)
	}
	if desired != nil {
		t.Fatalf("incomplete profile rendered %d children", len(desired))
	}
	config = stackConsumerConfig()
	config["spec"].(map[string]any)["stackConsumerProfiles"].([]any)[0].(map[string]any)["targets"] = []any{map[string]any{"slug": "", "region": "prod-us-central-0"}}
	if desired, err = renderStackConsumer(stackConsumerClaim(), nil, config); err == nil || !strings.Contains(err.Error(), stackConsumerIncompleteProfileMessage) {
		t.Fatalf("incomplete target error = %v, want %q", err, stackConsumerIncompleteProfileMessage)
	} else if desired != nil {
		t.Fatalf("incomplete target rendered %d children", len(desired))
	}

	config = stackConsumerConfig()
	profiles := config["spec"].(map[string]any)["stackConsumerProfiles"].([]any)
	profiles = append(profiles, map[string]any{
		"name": "other-reader", "consumer": map[string]any{"name": "example-consumer-a", "outputSecretPath": "/platform/grafana-cloud/consumers/other"},
	})
	config["spec"].(map[string]any)["stackConsumerProfiles"] = profiles
	if desired, err = renderStackConsumer(stackConsumerClaim(), nil, config); err == nil || !strings.Contains(err.Error(), stackConsumerIncompleteProfileMessage) {
		t.Fatalf("duplicate consumer identity error = %v, want %q", err, stackConsumerIncompleteProfileMessage)
	} else if desired != nil {
		t.Fatalf("duplicate consumer identity rendered %d children", len(desired))
	}
}

func TestStackConsumerProviderAdmissionAndReadback(t *testing.T) {
	e := &admissionEnv{paths: []string{"../apis/stack-consumer-v1beta1.yaml"}}
	if err := e.Start(t); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = e.Stop() })
	installProviderAdmissionCRDs(t, e)
	bootstrap, err := renderStackConsumer(stackConsumerClaim(), nil, stackConsumerConfig())
	if err != nil {
		t.Fatal(err)
	}
	observed := map[resource.Name]resource.ObservedComposed{
		stackConsumerObserverName: stackConsumerObserved(stackConsumerDesired(t, bootstrap, stackConsumerObserverName), map[string]any{"status": map[string]any{"atProvider": map[string]any{"id": "12345", "slug": "exampleobservedstack", "regionSlug": "prod-us-central-0"}}}),
	}
	policyStage, err := renderStackConsumer(stackConsumerClaim(), observed, stackConsumerConfig())
	if err != nil {
		t.Fatal(err)
	}
	observed[stackConsumerPolicyName] = stackConsumerObserved(stackConsumerDesired(t, policyStage, stackConsumerPolicyName), map[string]any{"status": map[string]any{"atProvider": map[string]any{"policyId": "67890"}}})
	children, err := renderStackConsumer(stackConsumerClaim(), observed, stackConsumerConfig())
	if err != nil {
		t.Fatal(err)
	}
	admitProviderChildren(t, e, children)
	readback := map[resource.Name]resource.ObservedComposed{
		stackConsumerObserverName: stackConsumerReadback(t, e, children[stackConsumerObserverName], map[string]any{"status": map[string]any{"atProvider": map[string]any{"id": "12345", "slug": "exampleobservedstack", "regionSlug": "prod-us-central-0"}}}),
		stackConsumerPolicyName:   stackConsumerReadback(t, e, children[stackConsumerPolicyName], map[string]any{"status": map[string]any{"atProvider": map[string]any{"policyId": "67890"}}}),
		stackConsumerTokenName:    stackConsumerReadback(t, e, children[stackConsumerTokenName], nil),
	}
	if _, err := renderStackConsumer(stackConsumerClaim(), readback, stackConsumerConfig()); err != nil {
		t.Fatalf("provider API-server readback deadlocked the renderer: %v", err)
	}
}

// This test becomes reachable as soon as the root wiring pass registers the
// renderer. It deliberately supplies no local GrafanaCloudStackRequest.
func TestRunFunctionRoutesStackConsumer(t *testing.T) {
	composite := mustJSON(stackConsumerClaim())
	input := mustJSON(stackConsumerConfig())
	request := &fnv1.RunFunctionRequest{
		Observed: &fnv1.State{Composite: &fnv1.Resource{Resource: resource.MustStructJSON(composite)}},
		Input:    resource.MustStructJSON(input),
	}
	response, err := (&Function{log: logging.NewNopLogger()}).RunFunction(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if fatal := fatalResult(response); fatal != "" {
		t.Fatalf("RunFunction rejected stack consumer: %s", fatal)
	}
	if _, ok := response.GetDesired().GetResources()[string(stackConsumerObserverName)]; !ok {
		t.Fatal("RunFunction did not reach the stack consumer observer renderer")
	}
}

func stackConsumerClaim() map[string]any {
	return map[string]any{
		"apiVersion": "platform.example.org/v1beta1",
		"kind":       "GrafanaStackConsumer",
		"metadata":   map[string]any{"name": "example-reader", "namespace": "consumer-platform"},
		"spec": map[string]any{
			"profile": "example-reader",
			"stack":   map[string]any{"slug": "exampleobservedstack", "region": "prod-us-central-0"},
		},
	}
}

func stackConsumerConfig() map[string]any {
	return map[string]any{"spec": map[string]any{
		"maximumTokenLifetime": "720h",
		"secretStoreRef":       map[string]any{"name": "grafana-vending-secrets", "kind": "SecretStore"},
		"stackConsumerProfiles": []any{map[string]any{
			"name":               "example-reader",
			"namespace":          "consumer-platform",
			"providerConfigName": "grafana-cloud-org-example-primary",
			"targets":            []any{map[string]any{"slug": "exampleobservedstack", "region": "prod-us-central-0"}},
			"consumer":           map[string]any{"name": "example-consumer-a", "outputSecretPath": "/platform/grafana-cloud/consumers/example-consumer-a"},
			"scopes":             []any{"metrics:read"},
		}},
	}}
}

func stackConsumerDesired(t *testing.T, desired map[resource.Name]*resource.DesiredComposed, name resource.Name) map[string]any {
	t.Helper()
	child := desired[name]
	if child == nil {
		t.Fatalf("missing child %q", name)
	}
	return child.Resource.UnstructuredContent()
}

func stackConsumerObserved(document map[string]any, overlay map[string]any) resource.ObservedComposed {
	encoded, _ := json.Marshal(document)
	var copied map[string]any
	_ = json.Unmarshal(encoded, &copied)
	for key, value := range overlay {
		copied[key] = value
	}
	return observedComposed(mustJSON(copied))
}

func stackConsumerReadback(t *testing.T, e *admissionEnv, child *resource.DesiredComposed, overlay map[string]any) resource.ObservedComposed {
	t.Helper()
	want := child.Resource.UnstructuredContent()
	got := &unstructured.Unstructured{}
	got.SetAPIVersion(want["apiVersion"].(string))
	got.SetKind(want["kind"].(string))
	if err := e.client.Get(context.Background(), client.ObjectKey{Namespace: nestedMap(t, want, "metadata")["namespace"].(string), Name: nestedMap(t, want, "metadata")["name"].(string)}, got); err != nil {
		t.Fatal(err)
	}
	return stackConsumerObserved(got.Object, overlay)
}
