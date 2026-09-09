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

const assertsCapDenial = "selected asserts profile exceeds a platform-owned resource or list cap"

func assertsAdmissionDocuments(t *testing.T) map[string]*unstructured.Unstructured {
	t.Helper()
	raw, err := os.ReadFile("../apis/asserts-v1beta1.yaml")
	if err != nil {
		t.Fatal(err)
	}
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(raw), 4096)
	result := map[string]*unstructured.Unstructured{}
	for {
		obj := &unstructured.Unstructured{}
		if err := decoder.Decode(&obj.Object); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		result[obj.GetKind()] = obj
	}
	return result
}

func assertsAdmissionRequest(name, profile string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "platform.example.org/v1beta1", "kind": "GrafanaAsserts",
		"metadata": map[string]any{"name": name, "namespace": "default"},
		"spec":     map[string]any{"stackRef": map[string]any{"name": name}, "profile": profile},
	}}
}

func TestAssertsAdmissionPlatformCaps(t *testing.T) {
	e := &admissionEnv{paths: []string{"../apis/asserts-v1beta1.yaml"}}
	if err := e.Start(t); err != nil {
		t.Fatalf("start asserts admission environment: %v", err)
	}
	t.Cleanup(func() {
		if err := e.Stop(); err != nil {
			t.Error(err)
		}
	})
	ctx := context.Background()
	docs := assertsAdmissionDocuments(t)
	composition := docs["Composition"]
	pipeline, _, _ := unstructured.NestedSlice(composition.Object, "spec", "pipeline")
	input := pipeline[0].(map[string]any)["input"].(map[string]any)["spec"].(map[string]any)
	baseline := input["assertsProfiles"].([]any)[0].(map[string]any)
	// Use one valid input and two distinct over-budget inputs: child count and
	// nested rule count. Only platform profile data changes, never request limits.
	clone := func(v map[string]any) map[string]any {
		return (&unstructured.Unstructured{Object: v}).DeepCopy().Object
	}
	aggregate := clone(baseline)
	aggregate["name"] = "overaggregate"
	aggregate["limits"].(map[string]any)["maxManagedResources"] = int64(1)
	nested := clone(baseline)
	nested["name"] = "overnested"
	nested["limits"].(map[string]any)["maxPromRulesPerGroup"] = int64(1)
	group := nested["promRuleFiles"].([]any)[0].(map[string]any)["group"].([]any)[0].(map[string]any)
	rule := clone(group["rule"].([]any)[0].(map[string]any))
	rule["record"] = "platform:second:count"
	group["rule"] = append(group["rule"].([]any), rule)
	sparse := clone(baseline)
	sparse["name"] = "sparse"
	for _, field := range []string{"logConfigs", "profileConfigs", "traceConfigs"} {
		delete(sparse[field].([]any)[0].(map[string]any), "match")
	}
	delete(sparse["thresholds"].(map[string]any), "requestThresholds")
	delete(sparse["thresholds"].(map[string]any), "resourceThresholds")
	entity := sparse["customModelRules"].([]any)[0].(map[string]any)["rules"].([]any)[0].(map[string]any)["entity"].([]any)[0].(map[string]any)
	delete(entity, "enrichedBy")
	promRule := sparse["promRuleFiles"].([]any)[0].(map[string]any)["group"].([]any)[0].(map[string]any)["rule"].([]any)[0].(map[string]any)
	delete(promRule, "disableInGroups")
	emptyDatasets := clone(baseline)
	emptyDatasets["name"] = "emptydatasets"
	emptyDatasets["stack"].(map[string]any)["dataset"] = []any{}
	missingDatasets := clone(baseline)
	missingDatasets["name"] = "missingdatasets"
	delete(missingDatasets["stack"].(map[string]any), "dataset")
	input["assertsProfiles"] = []any{baseline, aggregate, nested, sparse, emptyDatasets, missingDatasets}
	if err := unstructured.SetNestedSlice(composition.Object, pipeline, "spec", "pipeline"); err != nil {
		t.Fatal(err)
	}
	if err := e.Apply(ctx, composition); err != nil {
		t.Fatal(err)
	}
	counter := 0
	applyUntil := func(prefix, profile string, wantDenied bool) error {
		t.Helper()
		deadline := time.Now().Add(20 * time.Second)
		for {
			counter++
			err := e.Apply(ctx, assertsAdmissionRequest(fmt.Sprintf("%s%d", prefix, counter), profile))
			if wantDenied && err != nil && strings.Contains(err.Error(), assertsCapDenial) {
				return err
			}
			if !wantDenied && err == nil {
				return nil
			}
			if err != nil && !strings.Contains(err.Error(), assertsCapDenial) {
				t.Fatalf("unexpected asserts admission result: %v", err)
			}
			if time.Now().After(deadline) {
				t.Fatalf("asserts admission did not reach denied=%v: %v", wantDenied, err)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	if err := applyUntil("valid", "baseline", false); err != nil {
		t.Fatal(err)
	}
	applyUntil("sparse", "sparse", false)
	t.Log("API-SERVER omitted optional lists: admitted")
	for _, profile := range []string{"overaggregate", "overnested", "missing", "emptydatasets", "missingdatasets"} {
		t.Logf("API-SERVER create %s: %v", profile, applyUntil("denied", profile, true))
	}
	update := assertsAdmissionRequest("profileupdate", "baseline")
	if err := e.Apply(ctx, update); err != nil {
		t.Fatal(err)
	}
	update.Object["spec"].(map[string]any)["profile"] = "overaggregate"
	if err := e.Apply(ctx, update); err == nil || !strings.Contains(err.Error(), assertsCapDenial) {
		t.Fatalf("over-budget profile update = %v, want own cap denial", err)
	} else {
		t.Logf("API-SERVER update: %v", err)
	}
	policy := docs["ValidatingAdmissionPolicy"]
	weakened := policy.DeepCopy()
	if err := unstructured.SetNestedSlice(weakened.Object, []any{map[string]any{"expression": "true", "message": assertsCapDenial}}, "spec", "validations"); err != nil {
		t.Fatal(err)
	}
	if err := e.Apply(ctx, weakened); err != nil {
		t.Fatal(err)
	}
	for _, profile := range []string{"overaggregate", "overnested", "emptydatasets", "missingdatasets"} {
		applyUntil("weakened", profile, false)
		t.Logf("API-SERVER weakened cap %s: admitted", profile)
	}
	if err := e.Apply(ctx, policy); err != nil {
		t.Fatal(err)
	}
	for _, profile := range []string{"overaggregate", "overnested", "emptydatasets", "missingdatasets"} {
		t.Logf("API-SERVER restored cap %s: %v", profile, applyUntil("restored", profile, true))
	}
	wrong := assertsAdmissionRequest("wrongowner", "baseline")
	wrong.Object["spec"].(map[string]any)["stackRef"].(map[string]any)["name"] = "anotherstack"
	if err := e.Apply(ctx, wrong); err == nil || !strings.Contains(err.Error(), "metadata.name must match spec.stackRef.name") {
		t.Fatalf("duplicate surface owner = %v, want own ownership denial", err)
	} else {
		t.Logf("API-SERVER ownership: %v", err)
	}
}
