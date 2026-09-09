package main

import (
	"strings"
	"testing"

	"github.com/crossplane/function-sdk-go/resource"
)

func TestStackConsumerTokenCeilingAtCredentialBoundary(t *testing.T) {
	for _, ceiling := range []string{"", "invalid"} {
		t.Run("refuse_"+ceiling, func(t *testing.T) {
			config := stackConsumerConfig()
			config["spec"].(map[string]any)["maximumTokenLifetime"] = ceiling
			desired, err := renderStackConsumer(stackConsumerClaim(), nil, config)
			if err == nil || !strings.Contains(err.Error(), "maximumTokenLifetime") || len(desired) != 0 {
				t.Fatalf("unsafe ceiling %q: error=%v children=%d", ceiling, err, len(desired))
			}
		})
	}
	config := stackConsumerConfig()
	config["spec"].(map[string]any)["maximumTokenLifetime"] = "2h"
	bootstrap, err := renderStackConsumer(stackConsumerClaim(), nil, config)
	if err != nil {
		t.Fatal(err)
	}
	observed := map[resource.Name]resource.ObservedComposed{
		stackConsumerObserverName: stackConsumerObserved(stackConsumerDesired(t, bootstrap, stackConsumerObserverName), map[string]any{"status": map[string]any{"atProvider": map[string]any{"id": "12345", "slug": "exampleobservedstack", "regionSlug": "prod-us-central-0"}}}),
	}
	policyStage, err := renderStackConsumer(stackConsumerClaim(), observed, config)
	if err != nil {
		t.Fatal(err)
	}
	if len(policyStage) != 2 {
		t.Fatalf("children without observed policy ID = %d, want observer and policy only", len(policyStage))
	}
	observed[stackConsumerPolicyName] = stackConsumerObserved(stackConsumerDesired(t, policyStage, stackConsumerPolicyName), map[string]any{"status": map[string]any{"atProvider": map[string]any{"policyId": "67890"}}})
	desired, err := renderStackConsumer(stackConsumerClaim(), observed, config)
	if err != nil {
		t.Fatal(err)
	}
	token := nestedMap(t, stackConsumerDesired(t, desired, stackConsumerTokenName), "spec", "forProvider")
	if token["expireAfter"] != "2h" || token["earlyRotationWindow"] != "1h" {
		t.Fatalf("token ceiling/window = %v/%v, want 2h/1h", token["expireAfter"], token["earlyRotationWindow"])
	}
}

func TestStackConsumerPreservesAssignedPolicyIdentityWhileStatusIsPending(t *testing.T) {
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
	if stackConsumerObservedExternalName(policy) != "" {
		t.Fatal("first policy external name must not be fabricated")
	}
	policy["metadata"].(map[string]any)["annotations"] = map[string]any{"crossplane.io/external-name": "prod-us-central-0:67890"}
	observed[stackConsumerPolicyName] = stackConsumerObserved(policy, nil)
	desired, err := renderStackConsumer(stackConsumerClaim(), observed, stackConsumerConfig())
	if err != nil {
		t.Fatal(err)
	}
	if len(desired) != 2 {
		t.Fatalf("pending policy status emitted %d children, want 2", len(desired))
	}
	if got := stackConsumerObservedExternalName(stackConsumerDesired(t, desired, stackConsumerPolicyName)); got != "prod-us-central-0:67890" {
		t.Fatalf("pending policy lost provider identity: %q", got)
	}
}

func TestStackConsumerProfileAuthorizationRefusals(t *testing.T) {
	for _, tc := range []struct {
		name, message string
		alter         func(map[string]any)
	}{
		{"namespace", stackConsumerNamespaceMessage, func(c map[string]any) {
			c["spec"].(map[string]any)["stackConsumerProfiles"].([]any)[0].(map[string]any)["namespace"] = "other-namespace"
		}},
		{"target", stackConsumerTargetMessage, func(c map[string]any) {
			c["spec"].(map[string]any)["stackConsumerProfiles"].([]any)[0].(map[string]any)["targets"] = []any{map[string]any{"slug": "otherstack", "region": "prod-us-central-0"}}
		}},
		{"duplicate output", stackConsumerIncompleteProfileMessage, func(c map[string]any) {
			s := c["spec"].(map[string]any)
			s["stackConsumerProfiles"] = append(s["stackConsumerProfiles"].([]any), map[string]any{"name": "another-profile", "consumer": map[string]any{"name": "another-consumer", "outputSecretPath": "/platform/grafana-cloud/consumers/example-consumer-a"}})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := stackConsumerConfig()
			tc.alter(config)
			desired, err := renderStackConsumer(stackConsumerClaim(), nil, config)
			if err == nil || !strings.Contains(err.Error(), tc.message) || len(desired) != 0 {
				t.Fatalf("authorization refusal = %v children=%d, want %s", err, len(desired), tc.message)
			}
		})
	}
}
