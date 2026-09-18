package main

import (
	"testing"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composite"
	"github.com/crossplane/function-sdk-go/response"
)

func TestCredentialHealthDerivesRotatingTokenHealthFromSyncedConditions(t *testing.T) {
	stack := map[string]any{
		"metadata": map[string]any{"name": "teamdemo", "namespace": "grafana-vending"},
		"kind":     "GrafanaCloudStackRequest",
		"spec":     map[string]any{"slug": "teamdemo"},
	}
	serviceAccounts := map[string]any{
		"metadata": map[string]any{"name": "teamdemo", "namespace": "grafana-vending"},
		"kind":     "GrafanaServiceAccounts",
		"spec":     map[string]any{"accounts": []any{map[string]any{"name": "ci"}, map[string]any{"name": "automation"}}},
	}
	consumer := map[string]any{
		"metadata": map[string]any{"name": "consumer", "namespace": "grafana-vending"},
		"kind":     "GrafanaStackConsumer",
		"spec":     map[string]any{"profile": "consumer", "stack": map[string]any{"slug": "teamdemo", "region": "prod-example-1"}},
	}
	consumerConfig := map[string]any{"spec": map[string]any{"stackConsumerProfiles": []any{map[string]any{"name": "consumer", "consumer": map[string]any{"name": "example-consumer"}}}}}

	for _, test := range []struct {
		name     string
		xr       map[string]any
		config   map[string]any
		observed map[resource.Name]resource.ObservedComposed
		want     bool
	}{
		{
			name: "healthy stack tokens include AccessPolicy without expiry fields",
			xr:   stack,
			observed: map[resource.Name]resource.ObservedComposed{
				"administrator": credentialHealthObserved("teamdemo-admin", "StackServiceAccountRotatingToken", "True", "True"),
				"fleet":         credentialHealthObserved("teamdemo-fleet-management", "AccessPolicyRotatingToken", "True", "True"),
				"telemetry":     credentialHealthObserved("teamdemo-telemetry-publisher", "AccessPolicyRotatingToken", "True", "True"),
			},
			want: true,
		},
		{
			name: "stuck token is unhealthy even while Ready is true",
			xr:   stack,
			observed: map[resource.Name]resource.ObservedComposed{
				"administrator": credentialHealthObserved("teamdemo-admin", "StackServiceAccountRotatingToken", "True", "False"),
				"fleet":         credentialHealthObserved("teamdemo-fleet-management", "AccessPolicyRotatingToken", "True", "True"),
				"telemetry":     credentialHealthObserved("teamdemo-telemetry-publisher", "AccessPolicyRotatingToken", "True", "True"),
			},
			want: false,
		},
		{
			name:     "configured stack token family absent is unhealthy",
			xr:       stack,
			observed: map[resource.Name]resource.ObservedComposed{},
			want:     false,
		},
		{
			name: "healthy service account tokens are healthy",
			xr:   serviceAccounts,
			observed: map[resource.Name]resource.ObservedComposed{
				"ci":         credentialHealthObserved("teamdemo-ci-token", "ServiceAccountRotatingToken", "True", "True"),
				"automation": credentialHealthObserved("teamdemo-automation-token", "ServiceAccountRotatingToken", "True", "True"),
			},
			want: true,
		},
		{
			name:     "configured service account token family absent is unhealthy",
			xr:       serviceAccounts,
			observed: map[resource.Name]resource.ObservedComposed{},
			want:     false,
		},
		{
			name:   "healthy stack consumer AccessPolicy token is healthy without expiry fields",
			xr:     consumer,
			config: consumerConfig,
			observed: map[resource.Name]resource.ObservedComposed{
				"consumer": credentialHealthObserved("example-consumer-token", "AccessPolicyRotatingToken", "True", "True"),
			},
			want: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := test.config
			if config == nil {
				config = map[string]any{}
			}
			addCredentialHealth(test.xr, test.observed, config)

			healthy, configured := credentialHealthReady(config)
			if !configured {
				t.Fatal("rotating-token health was not configured")
			}
			if healthy != test.want {
				t.Fatalf("credential health = %v, want %v", healthy, test.want)
			}
		})
	}
}

func TestCredentialHealthPreservesExistingExpiryStatusConfig(t *testing.T) {
	expiry := map[string]any{"effectiveExpiresAt": "2030-01-02T03:04:05Z"}
	config := map[string]any{expiryStatusConfigKey: expiry}
	observed := map[resource.Name]resource.ObservedComposed{
		"administrator": credentialHealthObserved("teamdemo-admin", "StackServiceAccountRotatingToken", "True", "True"),
		"fleet":         credentialHealthObserved("teamdemo-fleet-management", "AccessPolicyRotatingToken", "True", "True"),
		"telemetry":     credentialHealthObserved("teamdemo-telemetry-publisher", "AccessPolicyRotatingToken", "True", "True"),
	}

	addCredentialHealth(map[string]any{"metadata": map[string]any{"name": "teamdemo", "namespace": "grafana-vending"}, "kind": "GrafanaCloudStackRequest", "spec": map[string]any{"slug": "teamdemo"}}, observed, config)

	if got, ok := config[expiryStatusConfigKey].(map[string]any); !ok || got["effectiveExpiresAt"] != expiry["effectiveExpiresAt"] {
		t.Fatalf("expiry status config = %#v, want %#v", config[expiryStatusConfigKey], expiry)
	}
	status := map[string]any{}
	mergeStackExpiryStatus(status, config)
	if got := status["expiry"]; got == nil {
		t.Fatal("credential health dropped existing expiry status")
	}
}

func TestCredentialHealthMatchesObservedTokensToCurrentConfiguredMembers(t *testing.T) {
	serviceAccounts := func(accounts ...string) map[string]any {
		items := make([]any, 0, len(accounts))
		for _, account := range accounts {
			items = append(items, map[string]any{"name": account})
		}
		return map[string]any{
			"metadata": map[string]any{"name": "teamdemo", "namespace": "grafana-vending"},
			"kind":     "GrafanaServiceAccounts",
			"spec":     map[string]any{"accounts": items},
		}
	}
	for _, test := range []struct {
		name     string
		xr       map[string]any
		observed map[resource.Name]resource.ObservedComposed
		want     bool
	}{
		{
			name: "old token cannot substitute for a newly configured account",
			xr:   serviceAccounts("new"),
			observed: map[resource.Name]resource.ObservedComposed{
				"old": credentialHealthObserved("teamdemo-old-token", "ServiceAccountRotatingToken", "True", "True"),
			},
			want: false,
		},
		{
			name: "obsolete unhealthy token does not poison a current healthy account",
			xr:   serviceAccounts("keep"),
			observed: map[resource.Name]resource.ObservedComposed{
				"keep":    credentialHealthObserved("teamdemo-keep-token", "ServiceAccountRotatingToken", "True", "True"),
				"retired": credentialHealthObserved("teamdemo-retire-token", "ServiceAccountRotatingToken", "True", "False"),
			},
			want: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := map[string]any{}
			addCredentialHealth(test.xr, test.observed, config)
			healthy, configured := credentialHealthReady(config)
			if !configured || healthy != test.want {
				t.Fatalf("credential health = %v, configured = %v; want healthy = %v", healthy, configured, test.want)
			}
		})
	}
}

func TestCredentialHealthReadinessOnlyOverridesUnhealthyConfiguredTokens(t *testing.T) {
	for _, test := range []struct {
		name   string
		config map[string]any
		want   fnv1.Ready
	}{
		{name: "healthy", config: map[string]any{credentialHealthConfigKey: true}, want: fnv1.Ready_READY_UNSPECIFIED},
		{name: "unhealthy", config: map[string]any{credentialHealthConfigKey: false}, want: fnv1.Ready_READY_FALSE},
	} {
		t.Run(test.name, func(t *testing.T) {
			rsp := response.To(&fnv1.RunFunctionRequest{}, response.DefaultTTL)
			if err := response.SetDesiredCompositeResource(rsp, &resource.Composite{Resource: composite.New(), Ready: resource.ReadyUnspecified}); err != nil {
				t.Fatalf("set desired composite: %v", err)
			}
			if err := applyCredentialHealthReadiness(rsp, test.config); err != nil {
				t.Fatalf("apply credential health readiness: %v", err)
			}
			if got := rsp.GetDesired().GetComposite().GetReady(); got != test.want {
				t.Fatalf("composite Ready = %v, want %v", got, test.want)
			}
		})
	}
}

func credentialHealthObserved(name, kind, ready, synced string) resource.ObservedComposed {
	apiVersion := "cloud.grafana.m.crossplane.io/v1alpha1"
	if kind == "ServiceAccountRotatingToken" {
		apiVersion = "oss.grafana.m.crossplane.io/v1alpha1"
	}
	return observedComposed(`{"apiVersion":"` + apiVersion + `","kind":"` + kind + `","metadata":{"name":"` + name + `","namespace":"grafana-vending"},"status":{"conditions":[{"type":"Ready","status":"` + ready + `"},{"type":"Synced","status":"` + synced + `"}]}}`)
}
