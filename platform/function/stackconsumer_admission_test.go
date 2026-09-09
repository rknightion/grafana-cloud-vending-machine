package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

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
	for _, name := range []string{"consumer-platform", "unauthorized-consumer"} {
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
		return &unstructured.Unstructured{Object: map[string]any{"apiVersion": "platform.example.org/v1beta1", "kind": "GrafanaStackConsumer", "metadata": map[string]any{"name": profile, "namespace": namespace}, "spec": map[string]any{"profile": profile, "stack": map[string]any{"slug": slug, "region": "prod-us-central-0"}}}}
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
	valid := request("example-reader", "consumer-platform", "exampleobservedstack")
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
		{"duplicate-owner", "consumer-platform", "exampleobservedstack", "selected stack consumer profile is not a complete platform-owned credential configuration", 0},
		{"duplicate-output", "consumer-platform", "exampleobservedstack", "selected stack consumer profile is not a complete platform-owned credential configuration", 0},
		{"incomplete", "consumer-platform", "exampleobservedstack", "selected stack consumer profile is not a complete platform-owned credential configuration", 0},
		{"wrong-namespace", "unauthorized-consumer", "exampleobservedstack", "selected stack consumer profile does not authorize this namespace", 1},
		{"wrong-target", "consumer-platform", "differentstack", "selected stack consumer profile does not authorize this stack slug and region", 2},
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
