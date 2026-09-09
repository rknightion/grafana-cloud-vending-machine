package main

import (
	"strconv"
	"strings"
	"testing"

	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
)

func TestAssertsRendersEveryNamespacedKindWithPlatformOwnedInputs(t *testing.T) {
	desired, err := renderAsserts(assertsClaim(), nil, assertsConfig())
	if err != nil {
		t.Fatalf("render asserts: %v", err)
	}
	want := map[resource.Name]struct {
		kind, externalName string
	}{
		assertsNamedResourceName("custom-model-rules", "models"):         {"CustomModelRules", "models"},
		assertsNamedResourceName("log-config", "Request Errors"):         {"LogConfig", "Request Errors"},
		assertsNamedResourceName("notification-alerts-config", "alerts"): {"NotificationAlertsConfig", "alerts"},
		assertsNamedResourceName("profile-config", "profiles"):           {"ProfileConfig", "profiles"},
		assertsNamedResourceName("prom-rule-file", "rules"):              {"PromRuleFile", "rules"},
		"stack": {"Stack", "12345"},
		assertsNamedResourceName("suppressed-assertions-config", "suppressed"): {"SuppressedAssertionsConfig", "suppressed"},
		"thresholds": {"Thresholds", "custom_thresholds"},
		assertsNamedResourceName("trace-config", "traces"): {"TraceConfig", "traces"},
	}
	if len(desired) != len(want) {
		t.Fatalf("desired resource count = %d, want %d", len(desired), len(want))
	}
	for name, expected := range want {
		child, found := desired[name]
		if !found {
			t.Errorf("did not render %q", name)
			continue
		}
		object := child.Resource.UnstructuredContent()
		if got := object["apiVersion"]; got != assertsAPIVersion {
			t.Errorf("%s apiVersion = %q, want %q", name, got, assertsAPIVersion)
		}
		if got := object["kind"]; got != expected.kind {
			t.Errorf("%s kind = %q, want %q", name, got, expected.kind)
		}
		if got := desiredExternalName(t, object); got != expected.externalName {
			t.Errorf("%s external name = %q, want %q", name, got, expected.externalName)
		}
		if expected.externalName == "Request Errors" {
			if got := nestedMap(t, object, "metadata")["name"]; got == "teamdemo01-log-config-Request Errors" {
				t.Fatalf("provider name leaked into Kubernetes metadata.name: %q", got)
			}
			if got := nestedMap(t, object, "spec", "forProvider")["name"]; got != "Request Errors" {
				t.Fatalf("LogConfig provider name = %q, want exact platform name", got)
			}
		}
		provider := nestedMap(t, object, "spec", "providerConfigRef")
		if got, want := provider["name"], "teamdemo01"; got != want {
			t.Errorf("%s provider config = %q, want referenced stack config %q", name, got, want)
		}
	}
	stack := desired["stack"].Resource.UnstructuredContent()
	forProvider := nestedMap(t, stack, "spec", "forProvider")
	if _, found := forProvider["cloudAccessPolicyTokenSecretRef"]; !found {
		t.Fatal("Stack omitted its platform-owned cloud access token reference")
	}
	if _, found := forProvider["grafanaTokenSecretRef"]; found {
		t.Fatal("Stack included an omitted optional Grafana token reference")
	}
	if encoded := mustJSON(stack); strings.Contains(encoded, "token-value") {
		t.Fatal("Stack rendered a secret value")
	}
	env := &admissionEnv{paths: []string{"../apis/asserts-v1beta1.yaml"}}
	if err := env.Start(t); err != nil {
		t.Fatalf("start asserts provider admission environment: %v", err)
	}
	t.Cleanup(func() { _ = env.Stop() })
	installProviderAdmissionCRDs(t, env)
	admitProviderChildren(t, env, desired)
}

func TestAssertsFailsClosedForTrustedStackIdentityAndProfileTransitions(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{
			name:   "malformed stack ID",
			mutate: func(config map[string]any) { config["referencedStack"].(map[string]any)["stackID"] = "not-an-id" },
			want:   "trusted referenced stack ID",
		},
		{
			name: "wrong provider shape",
			mutate: func(config map[string]any) {
				config["referencedStack"].(map[string]any)["providerConfigName"] = "other-stack"
			},
			want: "trusted referenced stack context is incomplete",
		},
		{
			name: "duplicate configuration identity",
			mutate: func(config map[string]any) {
				profile := assertsProfile(config)
				profile["logConfigs"] = append(profile["logConfigs"].([]any), profile["logConfigs"].([]any)[0])
			},
			want: `duplicate LogConfig name "Request Errors"`,
		},
		{
			name: "missing required provider field",
			mutate: func(config map[string]any) {
				delete(assertsProfile(config)["traceConfigs"].([]any)[0].(map[string]any), "dataSourceUid")
			},
			want: "TraceConfig \"traces\" must set dataSourceUid",
		},
		{
			name: "missing required profile array",
			mutate: func(config map[string]any) {
				delete(assertsProfile(config), "traceConfigs")
			},
			want: "asserts profile traceConfigs must be an array",
		},
		{
			name: "provider structural rules limit",
			mutate: func(config map[string]any) {
				assertsProfile(config)["customModelRules"].([]any)[0].(map[string]any)["rules"] = []any{
					map[string]any{"entity": []any{map[string]any{"name": "first", "definedBy": []any{map[string]any{"query": "up"}}}}},
					map[string]any{"entity": []any{map[string]any{"name": "second", "definedBy": []any{map[string]any{"query": "up"}}}}},
				}
			},
			want: "must set exactly one rules block",
		},
		{
			name: "raised provider structural cap",
			mutate: func(config map[string]any) {
				assertsLimits(assertsProfile(config))["maxRulesPerCustomModelRules"] = float64(2)
			},
			want: "must be at most 1",
		},
		{
			name: "duplicate selected profile",
			mutate: func(config map[string]any) {
				profiles := config["spec"].(map[string]any)["assertsProfiles"].([]any)
				config["spec"].(map[string]any)["assertsProfiles"] = append(profiles, profiles[0])
			},
			want: `profile "standard" is configured more than once`,
		},
		{
			name: "malformed secret reference",
			mutate: func(config map[string]any) {
				assertsProfile(config)["stack"].(map[string]any)["cloudAccessPolicyTokenSecretRef"] = map[string]any{"name": "only-a-name"}
			},
			want: "Stack must set cloudAccessPolicyTokenSecretRef.name and key",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := assertsConfig()
			tc.mutate(config)
			_, err := renderAsserts(assertsClaim(), nil, config)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("renderAsserts() error = %v, want %q", err, tc.want)
			}
			t.Logf("%s refusal: %v", tc.name, err)
		})
	}

	config := assertsConfig()
	profile := assertsProfile(config)
	profile["notificationAlertsConfigs"] = []any{map[string]any{"name": "replacement"}}
	observed := map[resource.Name]resource.ObservedComposed{
		assertsNamedResourceName("notification-alerts-config", "alerts"): assertsObserved(`{"apiVersion":"asserts.grafana.m.crossplane.io/v1alpha1","kind":"NotificationAlertsConfig"}`),
	}
	_, err := renderAsserts(assertsClaim(), observed, config)
	if err == nil || !strings.Contains(err.Error(), "would withdraw existing child") {
		t.Fatalf("profile transition error = %v, want observed-child preservation refusal", err)
	}
	t.Logf("profile transition refusal: %v", err)
}

func TestAssertsRejectsOverCapNestedLists(t *testing.T) {
	config := assertsConfig()
	profile := assertsProfile(config)
	profile["customModelRules"].([]any)[0].(map[string]any)["rules"] = []any{
		map[string]any{"entity": []any{
			map[string]any{"name": "first", "definedBy": []any{map[string]any{"query": "up"}}},
			map[string]any{"name": "second", "definedBy": []any{map[string]any{"query": "up"}}},
		}},
	}
	assertsLimits(profile)["maxEntitiesPerRule"] = float64(1)
	_, err := renderAsserts(assertsClaim(), nil, config)
	if err == nil || !strings.Contains(err.Error(), assertsCapMessage) {
		t.Fatalf("nested cap error = %v, want %q", err, assertsCapMessage)
	}
	t.Logf("custom-model nested cap refusal: %v", err)
}

func TestAssertsRejectsOverCapTraceMatchValues(t *testing.T) {
	config := assertsConfig()
	profile := assertsProfile(config)
	profile["traceConfigs"].([]any)[0].(map[string]any)["match"] = []any{map[string]any{"property": "service", "op": "=", "values": []any{"first", "second"}}}
	assertsLimits(profile)["maxMatchValuesPerMatch"] = float64(1)
	_, err := renderAsserts(assertsClaim(), nil, config)
	if err == nil || !strings.Contains(err.Error(), assertsCapMessage) {
		t.Fatalf("trace nested cap error = %v, want %q", err, assertsCapMessage)
	}
	t.Logf("trace nested cap refusal: %v", err)
}

func TestAssertsAllowsDifferentParentAndLeafCaps(t *testing.T) {
	config := assertsConfig()
	entities := make([]any, 0, 30)
	for index := 0; index < 30; index++ {
		entities = append(entities, map[string]any{
			"name": "service-" + strconv.Itoa(index), "type": "Service",
			"definedBy":  []any{map[string]any{"query": "up"}},
			"enrichedBy": []any{},
		})
	}
	assertsProfile(config)["customModelRules"].([]any)[0].(map[string]any)["rules"] = []any{map[string]any{"entity": entities}}
	if _, err := renderAsserts(assertsClaim(), nil, config); err != nil {
		t.Fatalf("different parent and leaf caps were refused: %v", err)
	}
}

func TestAssertsWaitsForObservedStackContext(t *testing.T) {
	desired, err := renderAsserts(assertsClaim(), nil, map[string]any{"spec": map[string]any{}})
	if err != nil {
		t.Fatalf("missing observed stack context error = %v, want wait", err)
	}
	if len(desired) != 0 {
		t.Fatalf("missing observed stack context rendered %d resources", len(desired))
	}
	_, err = renderAsserts(assertsClaim(), map[resource.Name]resource.ObservedComposed{"stack": assertsObserved(`{"kind":"Stack"}`)}, map[string]any{"spec": map[string]any{}})
	if err == nil || !strings.Contains(err.Error(), "preserving existing composed resources") {
		t.Fatalf("missing context with observed child error = %v, want preservation refusal", err)
	}
	t.Logf("observed-context preservation refusal: %v", err)
}

func assertsClaim() map[string]any {
	return map[string]any{"metadata": map[string]any{"name": "teamdemo01", "namespace": "grafana-vending"}, "spec": map[string]any{"stackRef": map[string]any{"name": "teamdemo01"}, "profile": "standard"}}
}

func assertsConfig() map[string]any {
	return map[string]any{
		"referencedStack": map[string]any{"stackID": "12345", "providerConfigName": "teamdemo01"},
		"spec": map[string]any{"assertsProfiles": []any{map[string]any{
			"name": "standard", "limits": map[string]any{
				"maxManagedResources": 100.0, "maxCustomModelRules": 5.0, "maxLogConfigs": 20.0, "maxNotificationAlertsConfigs": 20.0, "maxProfileConfigs": 20.0, "maxPromRuleFiles": 10.0, "maxSuppressedAssertionsConfigs": 20.0, "maxTraceConfigs": 20.0,
				"maxRulesPerCustomModelRules": 1.0, "maxEntitiesPerRule": 50.0, "maxDefinedByPerEntity": 20.0, "maxEnrichedByPerEntity": 20.0, "maxMatchesPerConfig": 50.0, "maxMatchValuesPerMatch": 50.0, "maxPromGroupsPerFile": 20.0, "maxPromRulesPerGroup": 50.0, "maxDisableInGroupsPerRule": 20.0,
				"maxDatasets": 20.0, "maxDisabledVendorsPerDataset": 50.0, "maxFilterGroupsPerDataset": 50.0, "maxEnvLabelValuesPerFilterGroup": 20.0, "maxFiltersPerFilterGroup": 50.0, "maxFilterValuesPerFilter": 50.0, "maxSiteLabelValuesPerFilterGroup": 20.0, "maxHealthThresholds": 100.0, "maxRequestThresholds": 100.0, "maxResourceThresholds": 100.0,
			},
			"customModelRules": []any{map[string]any{
				"name": "models", "rules": []any{map[string]any{
					"entity": []any{map[string]any{"name": "service", "type": "Service", "definedBy": []any{map[string]any{"query": "up"}}}},
				}},
			}},
			"logConfigs":                  []any{map[string]any{"name": "Request Errors", "dataSourceUid": "logs", "defaultConfig": false, "priority": 1.0}},
			"notificationAlertsConfigs":   []any{map[string]any{"name": "alerts"}},
			"profileConfigs":              []any{map[string]any{"name": "profiles", "dataSourceUid": "profiles", "defaultConfig": false, "priority": 1.0}},
			"promRuleFiles":               []any{map[string]any{"name": "rules", "group": []any{map[string]any{"name": "availability", "rule": []any{map[string]any{"alert": "Unavailable", "expr": "up == 0"}}}}}},
			"stack":                       map[string]any{"cloudAccessPolicyTokenSecretRef": map[string]any{"name": "asserts-access", "key": "token"}, "dataset": []any{map[string]any{"type": "prometheus"}}},
			"suppressedAssertionsConfigs": []any{map[string]any{"name": "suppressed"}},
			"thresholds":                  map[string]any{"healthThresholds": []any{}, "requestThresholds": []any{}, "resourceThresholds": []any{}},
			"traceConfigs":                []any{map[string]any{"name": "traces", "dataSourceUid": "traces", "defaultConfig": false, "priority": 1.0}},
		}}},
	}
}

func assertsProfile(config map[string]any) map[string]any {
	return config["spec"].(map[string]any)["assertsProfiles"].([]any)[0].(map[string]any)
}
func assertsLimits(profile map[string]any) map[string]any { return profile["limits"].(map[string]any) }

func assertsObserved(document string) resource.ObservedComposed {
	value := composed.New()
	value.SetUnstructuredContent(observedResource(document).GetResource().AsMap())
	return resource.ObservedComposed{Resource: value}
}

func TestAssertsRejectsAutomaticDatasetDiscovery(t *testing.T) {
	for _, empty := range []bool{false, true} {
		config := assertsConfig()
		stack := assertsProfile(config)["stack"].(map[string]any)
		delete(stack, "dataset")
		if empty {
			stack["dataset"] = []any{}
		}
		_, err := renderAsserts(assertsClaim(), nil, config)
		if err == nil || !strings.Contains(err.Error(), "Stack must set a non-empty dataset list; automatic discovery is not bounded") {
			t.Fatalf("empty=%v: automatic dataset discovery error = %v, want explicit dataset refusal", empty, err)
		}
	}
}
