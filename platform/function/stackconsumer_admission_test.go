package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/crossplane/function-sdk-go/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func consumerPolicyDocuments(t *testing.T) map[string]*unstructured.Unstructured {
	t.Helper()
	raw, err := os.ReadFile("../apis/stack-consumer-v1beta1.yaml")
	if err != nil {
		t.Fatal(err)
	}
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(raw), 4096)
	docs := map[string]*unstructured.Unstructured{}
	for {
		obj := &unstructured.Unstructured{}
		if err := decoder.Decode(&obj.Object); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		docs[obj.GetKind()] = obj
	}
	return docs
}

func TestStackConsumerAdmissionAuthorization(t *testing.T) {
	e := &admissionEnv{paths: []string{"../apis/stack-consumer-v1beta1.yaml"}}
	if err := e.Start(t); err != nil {
		t.Fatalf("start consumer admission environment: %v", err)
	}
	t.Cleanup(func() {
		if err := e.Stop(); err != nil {
			t.Error(err)
		}
	})
	ctx := context.Background()
	for _, name := range []string{"telemetry-consumers", "unauthorized-consumer"} {
		if err := e.Apply(ctx, &unstructured.Unstructured{Object: map[string]any{"apiVersion": "v1", "kind": "Namespace", "metadata": map[string]any{"name": name}}}); err != nil {
			t.Fatal(err)
		}
	}
	docs := consumerPolicyDocuments(t)
	composition := docs["Composition"]
	pipeline, _, _ := unstructured.NestedSlice(composition.Object, "spec", "pipeline")
	input := pipeline[0].(map[string]any)["input"].(map[string]any)["spec"].(map[string]any)
	baseline := input["stackConsumerProfiles"].([]any)[0].(map[string]any)
	clone := func(v map[string]any) map[string]any {
		return (&unstructured.Unstructured{Object: v}).DeepCopy().Object
	}
	profiles := []any{baseline}
	for _, name := range []string{"wrong-target", "wrong-namespace", "incomplete", "duplicate-owner", "duplicate-output"} {
		p := clone(baseline)
		p["name"] = name
		p["consumer"].(map[string]any)["name"] = name
		p["consumer"].(map[string]any)["outputSecretPath"] = "/platform/grafana-cloud/consumers/" + name
		if name == "incomplete" {
			delete(p, "scopes")
		}
		profiles = append(profiles, p)
		if name == "duplicate-owner" || name == "duplicate-output" {
			duplicate := clone(p)
			duplicate["name"] = name + "-peer"
			if name == "duplicate-owner" {
				duplicate["consumer"].(map[string]any)["outputSecretPath"] = "/platform/grafana-cloud/consumers/owner-peer"
			} else {
				duplicate["consumer"].(map[string]any)["name"] = "output-peer"
			}
			profiles = append(profiles, duplicate)
		}
	}
	input["stackConsumerProfiles"] = profiles
	if err := unstructured.SetNestedSlice(composition.Object, pipeline, "spec", "pipeline"); err != nil {
		t.Fatal(err)
	}
	if err := e.Apply(ctx, composition); err != nil {
		t.Fatal(err)
	}
	request := func(profile, namespace, slug string) *unstructured.Unstructured {
		return &unstructured.Unstructured{Object: map[string]any{"apiVersion": "platform.example.org/v1beta1", "kind": "GrafanaStackConsumer", "metadata": map[string]any{"name": profile, "namespace": namespace}, "spec": map[string]any{"profile": profile, "stack": map[string]any{"slug": slug, "region": "prod-gb-south-1"}}}}
	}
	applyUntil := func(t *testing.T, obj *unstructured.Unstructured, message string) error {
		t.Helper()
		deadline := time.Now().Add(20 * time.Second)
		for {
			err := e.Apply(ctx, obj)
			if message == "" && err == nil {
				return nil
			}
			if message != "" && err != nil && strings.Contains(err.Error(), message) {
				return err
			}
			if time.Now().After(deadline) {
				t.Fatalf("consumer admission expected %q: %v", message, err)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	valid := request("robk-telemetry-writer", "telemetry-consumers", "robk")
	if err := applyUntil(t, valid, ""); err != nil {
		t.Fatal(err)
	}
	t.Log("API-SERVER consumer without local stack claim: admitted")
	policy := docs["ValidatingAdmissionPolicy"]
	validations, _, _ := unstructured.NestedSlice(policy.Object, "spec", "validations")
	cases := []struct {
		name, namespace, slug, message string
		rule                           int
	}{
		{"duplicate-owner", "telemetry-consumers", "robk", "selected stack consumer profile is not a complete platform-owned credential configuration", 0},
		{"duplicate-output", "telemetry-consumers", "robk", "selected stack consumer profile is not a complete platform-owned credential configuration", 0},
		{"incomplete", "telemetry-consumers", "robk", "selected stack consumer profile is not a complete platform-owned credential configuration", 0},
		{"wrong-namespace", "unauthorized-consumer", "robk", "selected stack consumer profile does not authorize this namespace", 1},
		{"wrong-target", "telemetry-consumers", "differentstack", "selected stack consumer profile does not authorize this stack slug and region", 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			obj := request(tc.name, tc.namespace, tc.slug)
			t.Logf("API-SERVER create denied: %v", applyUntil(t, obj, tc.message))
			weakened := policy.DeepCopy()
			rules, _, _ := unstructured.NestedSlice(weakened.Object, "spec", "validations")
			rules[tc.rule].(map[string]any)["expression"] = "true"
			if err := unstructured.SetNestedSlice(weakened.Object, rules, "spec", "validations"); err != nil {
				t.Fatal(err)
			}
			if err := e.Apply(ctx, weakened); err != nil {
				t.Fatal(err)
			}
			if err := applyUntil(t, obj, ""); err != nil {
				t.Fatal(err)
			}
			t.Log("API-SERVER weakened individual guard: admitted")
			if err := unstructured.SetNestedSlice(policy.Object, validations, "spec", "validations"); err != nil {
				t.Fatal(err)
			}
			if err := e.Apply(ctx, policy); err != nil {
				t.Fatal(err)
			}
			t.Logf("API-SERVER restored guard update denied: %v", applyUntil(t, obj, tc.message))
		})
	}
	for _, field := range []string{"stack", "profile"} {
		changed := valid.DeepCopy()
		if field == "stack" {
			changed.Object["spec"].(map[string]any)["stack"].(map[string]any)["slug"] = "differentstack"
		} else {
			changed.Object["spec"].(map[string]any)["profile"] = "wrong-target"
		}
		err := e.Apply(ctx, changed)
		if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("%s is immutable", field)) {
			t.Fatalf("%s mutation: %v", field, err)
		}
		t.Logf("API-SERVER immutable identity denied: %v", err)
	}
}

func TestShippedStackConsumerProfileAdmissionAndCredentialPublication(t *testing.T) {
	e := &admissionEnv{paths: []string{"../apis/stack-consumer-v1beta1.yaml"}}
	if err := e.Start(t); err != nil {
		t.Fatalf("start consumer admission environment: %v", err)
	}
	t.Cleanup(func() {
		if err := e.Stop(); err != nil {
			t.Error(err)
		}
	})
	ctx := context.Background()
	if err := e.Apply(ctx, &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "Namespace", "metadata": map[string]any{"name": "telemetry-consumers"},
	}}); err != nil {
		t.Fatal(err)
	}
	docs := consumerPolicyDocuments(t)
	composition := docs["Composition"]
	if err := e.Apply(ctx, composition); err != nil {
		t.Fatal(err)
	}
	request := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "platform.example.org/v1beta1",
		"kind":       "GrafanaStackConsumer",
		"metadata":   map[string]any{"name": "robk-telemetry-writer", "namespace": "telemetry-consumers"},
		"spec": map[string]any{
			"profile": "robk-telemetry-writer",
			"stack":   map[string]any{"slug": "robk", "region": "prod-gb-south-1"},
		},
	}}
	deadline := time.Now().Add(20 * time.Second)
	probe := request.DeepCopy()
	probe.SetName("profile-admission-probe")
	probe.Object["spec"].(map[string]any)["profile"] = "profile-admission-probe"
	for {
		err := e.Apply(ctx, probe)
		if err != nil && strings.Contains(err.Error(), stackConsumerIncompleteProfileMessage) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("API-SERVER consumer profile policy did not become ready: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	for {
		err := e.Apply(ctx, request)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("API-SERVER shipped profile admission: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Log("API-SERVER shipped robk telemetry consumer: admitted")

	pipeline, _, _ := unstructured.NestedSlice(composition.Object, "spec", "pipeline")
	config := pipeline[0].(map[string]any)["input"].(map[string]any)
	bootstrap, err := renderStackConsumer(request.Object, nil, config)
	if err != nil {
		t.Fatal(err)
	}
	stackID := strconv.Itoa(len("robk-telemetry-writer"))
	observed := map[resource.Name]resource.ObservedComposed{
		stackConsumerObserverName: stackConsumerObserved(stackConsumerDesired(t, bootstrap, stackConsumerObserverName), map[string]any{
			"status": map[string]any{"atProvider": map[string]any{"id": stackID, "slug": "robk", "regionSlug": "prod-gb-south-1"}},
		}),
	}
	policyStage, err := renderStackConsumer(request.Object, observed, config)
	if err != nil {
		t.Fatal(err)
	}
	policy := stackConsumerDesired(t, policyStage, stackConsumerPolicyName)
	if got := nestedMap(t, policy, "spec", "forProvider")["scopes"]; mustJSON(got) != mustJSON([]any{"metrics:write", "logs:write", "traces:write"}) {
		t.Fatalf("consumer policy scopes = %s, want telemetry write scopes only", mustJSON(got))
	}
	policyID := strconv.Itoa(len("telemetry-consumers"))
	observed[stackConsumerPolicyName] = stackConsumerObserved(policy, map[string]any{
		"status": map[string]any{"atProvider": map[string]any{"policyId": policyID}},
	})
	ready, err := renderStackConsumer(request.Object, observed, config)
	if err != nil {
		t.Fatal(err)
	}
	publication := stackConsumerDesired(t, ready, stackConsumerCredentialsName)
	remoteKey := nestedMap(t, publication, "spec")["data"].([]any)[0].(map[string]any)["match"].(map[string]any)["remoteRef"].(map[string]any)["remoteKey"]
	if got, want := remoteKey, "/platform/grafana-cloud/consumers/robk-telemetry"; got != want {
		t.Fatalf("credential output path = %v, want %q", got, want)
	}
	document := nestedMap(t, publication, "spec", "template", "data")["telemetry.json"]
	wantDocument := `{{ $token := index . "attribute.token" | toString }}{"stack_slug":"robk","stack_region":"prod-gb-south-1","access_policy_name":"robk-telemetry-access-policy","access_policy_token":{{ $token | toJson }}}`
	if got := document; got != wantDocument {
		t.Fatalf("telemetry.json template = %q, want %q", got, wantDocument)
	}
	t.Log("RENDERER telemetry.json and fixed output path: verified")
}
