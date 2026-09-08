package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type admissionRuleCase struct {
	name            string
	expectedMessage string
	skipReason      string
	run             func(context.Context, *admissionEnv, *testing.T) error
}

type admissionMutation func(*testing.T, *unstructured.Unstructured)

func TestAdmissionRules(t *testing.T) {
	if !admissionHarnessImplemented {
		t.Skip("admission harness pre-pass: implementation pending")
	}

	ctx := context.Background()
	env := &admissionEnv{paths: []string{
		"../apis/stack-v1beta1.yaml",
		"../apis/ladder-v1beta1.yaml",
		"../apis/k6-v1beta1.yaml",
	}}
	if err := env.Start(t); err != nil {
		t.Fatalf("start admission environment: %v", err)
	}
	t.Cleanup(func() {
		if err := env.Stop(); err != nil {
			t.Errorf("stop admission environment: %v", err)
		}
	})

	expiry := map[string]any{
		"expiresAt": "2030-01-02T03:04:05Z",
		"extensions": []any{
			extensionRecord("2030-02-03T04:05:06Z", "initial extension", "requester-a", "2029-12-01T00:00:00Z"),
		},
	}
	appendedExtensions := []any{
		extensionRecord("2030-02-03T04:05:06Z", "initial extension", "requester-a", "2029-12-01T00:00:00Z"),
		extensionRecord("2030-03-04T05:06:07Z", "approved follow-up", "requester-b", "2029-12-02T00:00:00Z"),
	}

	cases := []admissionRuleCase{
		{
			name:            "SCIM object is refused",
			expectedMessage: "SCIM is out of scope; use external-group mapping",
			run: createRule(
				func() *unstructured.Unstructured { return stackRequest("scimallowed", nil) },
				func() *unstructured.Unstructured {
					return stackRequest("scimobject", map[string]any{"scim": map[string]any{"enabled": true}})
				},
			),
		},

		{
			name:            "retention cannot be added on update",
			expectedMessage: "retention must be selected at creation",
			run: updateRule(
				func() *unstructured.Unstructured { return stackRequest("retentionadd", nil) },
				setStackDisplayName("Allowed retention add control"),
				setNested(map[string]any{"class": "standard"}, "spec", "retention"),
			),
		},
		{
			name:            "retention cannot be removed on update",
			expectedMessage: "retention must be selected at creation",
			run: updateRule(
				func() *unstructured.Unstructured {
					return stackRequest("retentionremove", map[string]any{"retention": map[string]any{"class": "standard"}})
				},
				setStackDisplayName("Allowed retention remove control"),
				removeNested("spec", "retention"),
			),
		},
		{
			name:            "retention class cannot change on update",
			expectedMessage: "retention class is a creation-time decision",
			run: updateRule(
				func() *unstructured.Unstructured {
					return stackRequest("retentionclass", map[string]any{"retention": map[string]any{"class": "standard"}})
				},
				setStackDisplayName("Allowed retention class control"),
				setNested("archive", "spec", "retention", "class"),
			),
		},
		{
			name:            "expiry cannot be added on update",
			expectedMessage: "expiry must be selected at creation",
			run: updateRule(
				func() *unstructured.Unstructured { return stackRequest("expiryadd", nil) },
				setStackDisplayName("Allowed expiry add control"),
				setNested(expiry, "spec", "expiry"),
			),
		},
		{
			name:            "expiry cannot be removed on update",
			expectedMessage: "expiry must be selected at creation",
			run: updateRule(
				func() *unstructured.Unstructured {
					return stackRequest("expiryremove", map[string]any{"expiry": expiry})
				},
				setStackDisplayName("Allowed expiry remove control"),
				removeNested("spec", "expiry"),
			),
		},
		{
			name:            "extension record cannot be removed",
			expectedMessage: "extension records cannot be removed or changed",
			run: updateRule(
				func() *unstructured.Unstructured {
					return stackRequest("extensionremove", map[string]any{"expiry": expiry})
				},
				setNested(appendedExtensions, "spec", "expiry", "extensions"),
				setNested(appendedExtensions[1:], "spec", "expiry", "extensions"),
			),
		},
		{
			name:            "extension record cannot be altered",
			expectedMessage: "extension records cannot be removed or changed",
			run: updateRule(
				func() *unstructured.Unstructured {
					return stackRequest("extensionalter", map[string]any{"expiry": expiry})
				},
				setNested(appendedExtensions, "spec", "expiry", "extensions"),
				setNested([]any{
					extensionRecord("2030-02-03T04:05:06Z", "altered extension", "requester-a", "2029-12-01T00:00:00Z"),
					appendedExtensions[1],
				}, "spec", "expiry", "extensions"),
			),
		},
		{
			name:            "ladder organization cannot change",
			expectedMessage: "ladder organization is immutable",
			run: updateRule(
				func() *unstructured.Unstructured { return ladderRequest("ladderorg") },
				setNested("reverse", "spec", "promotionDirection"),
				setNested("example-secondary", "spec", "organization"),
			),
		},
		{
			name:            "ladder region cannot change",
			expectedMessage: "region is immutable; a ladder cannot replace stacks",
			run: updateRule(
				func() *unstructured.Unstructured { return ladderRequest("ladderregion") },
				setNested("reverse", "spec", "promotionDirection"),
				setNested("prod-eu-west-0", "spec", "region"),
			),
		},
		{
			name:            "ladder rung name cannot change",
			expectedMessage: "rung identities and order are immutable",
			run: updateRule(
				func() *unstructured.Unstructured { return ladderRequest("ladderrungname") },
				setNested("reverse", "spec", "promotionDirection"),
				mutateLadderRungs(func(rungs []any) { rungs[0].(map[string]any)["name"] = "qa" }),
			),
		},
		{
			name:            "ladder rung order cannot change",
			expectedMessage: "rung identities and order are immutable",
			run: updateRule(
				func() *unstructured.Unstructured { return ladderRequest("ladderrungorder") },
				setNested("reverse", "spec", "promotionDirection"),
				mutateLadderRungs(func(rungs []any) { rungs[0], rungs[1] = rungs[1], rungs[0] }),
			),
		},
		{
			name:            "ladder rung slug cannot change",
			expectedMessage: "rung stack slugs are immutable",
			run: updateRule(
				func() *unstructured.Unstructured { return ladderRequest("ladderrungslug") },
				setNested("reverse", "spec", "promotionDirection"),
				mutateLadderRungs(func(rungs []any) { rungs[0].(map[string]any)["slug"] = "stage02" }),
			),
		},
		{
			name:            "omitted k6 load-zone intent is refused",
			expectedMessage: "spec.allowedLoadZones: Required value",
			run: createRule(
				func() *unstructured.Unstructured { return k6Request("k6zonesallowed", true) },
				func() *unstructured.Unstructured { return k6Request("k6zonesomitted", false) },
			),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skipReason != "" {
				t.Skip(tc.skipReason)
			}
			err := tc.run(ctx, env, t)
			if err == nil {
				t.Fatalf("forbidden request was admitted; expected %q", tc.expectedMessage)
			}
			if !strings.Contains(err.Error(), tc.expectedMessage) {
				t.Fatalf("request was refused for the wrong reason: expected %q in %q", tc.expectedMessage, err)
			}
			t.Logf("rejection output: %v", err)
		})
	}

	t.Run("explicit null SCIM persists and is refused at reconcile", func(t *testing.T) {
		omitted := stackRequest("scimnullok", nil)
		requireAdmitted(ctx, env, t, omitted, "omitted SCIM create")
		if err := env.client.Get(ctx, client.ObjectKeyFromObject(omitted), omitted); err != nil {
			t.Fatal(err)
		}
		omittedSpec := omitted.Object["spec"].(map[string]any)
		if _, present := omittedSpec["scim"]; present {
			t.Fatal("omitted SCIM acquired a persisted key")
		}
		t.Log("PERSISTED omitted: spec has 'scim' key = false")
		obj := stackRequest("scimnull", map[string]any{"scim": nil})
		requireAdmitted(ctx, env, t, obj, "explicit null SCIM create")
		if err := env.client.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
			t.Fatal(err)
		}
		spec := obj.Object["spec"].(map[string]any)
		value, present := spec["scim"]
		if !present || value != nil {
			t.Fatalf("explicit null SCIM persistence changed: present=%t value=%v", present, value)
		}
		t.Log("PERSISTED explicit-null: spec has 'scim' key = true, value = null")
		// TestSCIMStackRequestsRejected in scim_test.go covers the wider value table.
		// This assertion connects the actual persisted API object to the renderer.
		desired, err := renderStack(obj.Object, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "SCIM is out of scope; use GrafanaTeamAccess or GrafanaCustomRoleBinding external-group mapping") || len(desired) != 0 {
			t.Fatalf("persisted null must fail closed at reconcile: desired=%v error=%v", desired, err)
		}
		t.Logf("RECONCILE persisted explicit-null refused; children=%d: %v", len(desired), err)
	})

	t.Run("weakened SCIM rule negative control", func(t *testing.T) {
		runSCIMNegativeControl(ctx, env, t)
	})
}

func createRule(allowed, rejected func() *unstructured.Unstructured) func(context.Context, *admissionEnv, *testing.T) error {
	return func(ctx context.Context, env *admissionEnv, t *testing.T) error {
		requireAdmitted(ctx, env, t, allowed(), "paired allowed create")
		return env.Apply(ctx, rejected())
	}
}

func updateRule(base func() *unstructured.Unstructured, allowed, rejected admissionMutation) func(context.Context, *admissionEnv, *testing.T) error {
	return func(ctx context.Context, env *admissionEnv, t *testing.T) error {
		obj := base()
		requireAdmitted(ctx, env, t, obj, "initial create")
		allowed(t, obj)
		requireAdmitted(ctx, env, t, obj, "paired allowed update")
		rejected(t, obj)
		return env.Apply(ctx, obj)
	}
}

func requireAdmitted(ctx context.Context, env *admissionEnv, t *testing.T, obj *unstructured.Unstructured, label string) {
	t.Helper()
	if err := env.Apply(ctx, obj); err != nil {
		t.Fatalf("%s was refused: %v", label, err)
	}
}

func stackRequest(name string, extraSpec map[string]any) *unstructured.Unstructured {
	spec := map[string]any{
		"displayName":  "Admission Test Stack",
		"slug":         name,
		"region":       "prod-us-central-0",
		"usage":        "development",
		"organization": "example-primary",
	}
	for key, value := range extraSpec {
		spec[key] = value
	}
	return requestObject("GrafanaCloudStackRequest", name, spec)
}

func ladderRequest(name string) *unstructured.Unstructured {
	return requestObject("GrafanaStackLadder", name, map[string]any{
		"organization":       "example-primary",
		"region":             "prod-us-central-0",
		"promotionDirection": "forward",
		"repository": map[string]any{
			"url":  "https://git.example.org/platform/requests",
			"path": "stacks",
		},
		"rungs": []any{
			map[string]any{
				"name": "staging", "slug": "stage01", "displayName": "Staging", "usage": "development",
				"branch": "staging", "connectionRef": map[string]any{"name": "staging-connection"},
			},
			map[string]any{
				"name": "production", "slug": "prod001", "displayName": "Production", "usage": "production",
				"branch": "main", "connectionRef": map[string]any{"name": "production-connection"},
			},
		},
	})
}

func k6Request(name string, includeLoadZones bool) *unstructured.Unstructured {
	spec := map[string]any{
		"stackRef":    map[string]any{"name": "stack001"},
		"grafanaUser": "example-user",
	}
	if includeLoadZones {
		spec["allowedLoadZones"] = []any{}
	}
	return requestObject("GrafanaK6Project", name, spec)
}

func requestObject(kind, name string, spec map[string]any) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "platform.example.org/v1beta1",
		"kind":       kind,
		"metadata": map[string]any{
			"name":      name,
			"namespace": "default",
		},
		"spec": spec,
	}}
}

func extensionRecord(extendedTo, reason, requestedBy, recordedAt string) map[string]any {
	return map[string]any{
		"extendedTo":  extendedTo,
		"reason":      reason,
		"requestedBy": requestedBy,
		"recordedAt":  recordedAt,
	}
}

func setStackDisplayName(value string) admissionMutation {
	return setNested(value, "spec", "displayName")
}

func setNested(value any, fields ...string) admissionMutation {
	return func(t *testing.T, obj *unstructured.Unstructured) {
		t.Helper()
		if err := unstructured.SetNestedField(obj.Object, value, fields...); err != nil {
			t.Fatalf("set %s: %v", strings.Join(fields, "."), err)
		}
	}
}

func removeNested(fields ...string) admissionMutation {
	return func(_ *testing.T, obj *unstructured.Unstructured) {
		unstructured.RemoveNestedField(obj.Object, fields...)
	}
}

func mutateLadderRungs(mutate func([]any)) admissionMutation {
	return func(t *testing.T, obj *unstructured.Unstructured) {
		t.Helper()
		rungs, found, err := unstructured.NestedSlice(obj.Object, "spec", "rungs")
		if err != nil || !found {
			t.Fatalf("read spec.rungs: found=%t error=%v", found, err)
		}
		mutate(rungs)
		if err := unstructured.SetNestedSlice(obj.Object, rungs, "spec", "rungs"); err != nil {
			t.Fatalf("set spec.rungs: %v", err)
		}
	}
}

func runSCIMNegativeControl(ctx context.Context, env *admissionEnv, t *testing.T) {
	t.Helper()
	const (
		xrdPath         = "../apis/stack-v1beta1.yaml"
		originalRule    = `- rule: "!has(self.scim)"`
		weakenedRule    = `- rule: "true"`
		expectedMessage = "SCIM is out of scope; use external-group mapping"
	)

	baselineErr := env.Apply(ctx, stackRequest("scimbaseline", map[string]any{"scim": map[string]any{}}))
	if baselineErr == nil || !strings.Contains(baselineErr.Error(), expectedMessage) {
		t.Fatalf("negative-control baseline did not reject SCIM with its own message: %v", baselineErr)
	}
	t.Logf("negative control baseline output: %v", baselineErr)

	source, err := os.ReadFile(xrdPath)
	if err != nil {
		t.Fatalf("read negative-control source XRD: %v", err)
	}
	if strings.Count(string(source), originalRule) != 1 {
		t.Fatalf("negative-control source contains %d copies of %q", strings.Count(string(source), originalRule), originalRule)
	}
	scratchDir, err := os.MkdirTemp("", "gcv-admission-negative-control-")
	if err != nil {
		t.Fatalf("create preserved scratch directory: %v", err)
	}
	t.Logf("negative control scratch directory: %s", filepath.Base(scratchDir))
	scratchPath := filepath.Join(scratchDir, "stack-v1beta1.yaml")
	weakened := strings.Replace(string(source), originalRule, weakenedRule, 1)
	if err := os.WriteFile(scratchPath, []byte(weakened), 0o600); err != nil {
		t.Fatalf("write scratch XRD: %v", err)
	}

	weakenedCRD, err := crdFromXRD(scratchPath)
	if err != nil {
		t.Fatalf("derive weakened scratch CRD: %v", err)
	}
	if err := env.Apply(ctx, weakenedCRD); err != nil {
		t.Fatalf("apply weakened scratch CRD: %v", err)
	}
	restored := false
	t.Cleanup(func() {
		if restored {
			return
		}
		originalCRD, deriveErr := crdFromXRD(xrdPath)
		if deriveErr != nil {
			t.Errorf("derive original CRD during cleanup: %v", deriveErr)
			return
		}
		if applyErr := env.Apply(ctx, originalCRD); applyErr != nil {
			t.Errorf("restore original CRD during cleanup: %v", applyErr)
		}
	})

	weakenedErr := env.Apply(ctx, stackRequest("scimweakened", map[string]any{"scim": map[string]any{}}))
	if weakenedErr != nil {
		t.Errorf("negative control weakened output: %v", weakenedErr)
	} else {
		t.Log("negative control weakened output: admitted")
	}

	originalCRD, err := crdFromXRD(xrdPath)
	if err != nil {
		t.Fatalf("derive original CRD for restore: %v", err)
	}
	if err := env.Apply(ctx, originalCRD); err != nil {
		t.Fatalf("restore original CRD: %v", err)
	}
	restored = true

	restoredErr := env.Apply(ctx, stackRequest("scimrestored", map[string]any{"scim": map[string]any{}}))
	if restoredErr == nil || !strings.Contains(restoredErr.Error(), expectedMessage) {
		t.Fatalf("negative control restored output did not reject SCIM with its own message: %v", restoredErr)
	}
	t.Logf("negative control restored output: %v", restoredErr)
}
