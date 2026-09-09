package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func TestPDCRendersNetworksBeforeObservedIDsAndPreservesObservedIdentity(t *testing.T) {
	claim := pdcClaim()
	config := pdcConfig("720h")
	desired, err := renderPDC(claim, nil, config)
	if err != nil {
		t.Fatalf("renderPDC() error = %v", err)
	}
	networkName := pdcNetworkResourceName("primary")
	network, ok := desired[networkName]
	if !ok {
		t.Fatalf("network %q was not rendered", networkName)
	}
	if _, exists := network.Resource.UnstructuredContent()["metadata"].(map[string]any)["annotations"]; exists {
		t.Fatal("first-pass PDC network guessed a provider-assigned external name")
	}
	if _, exists := desired[pdcTokenResourceName("primary", 0)]; exists {
		t.Fatal("PDC token rendered before the provider observed a PDC network ID")
	}
	if _, exists := desired[resource.Name("datasource-"+stableResourceSuffix("example-pdc:orders"))]; exists {
		t.Fatal("datasource rendered before the provider observed its PDC network ID")
	}

	observed := map[resource.Name]resource.ObservedComposed{
		networkName:                        pdcObserved(`{"metadata":{"annotations":{"crossplane.io/external-name":"provider-network-id"}},"status":{"atProvider":{"pdcNetworkId":"provider-network-id"}}}`),
		pdcTokenResourceName("primary", 0): pdcObserved(`{"metadata":{"annotations":{"crossplane.io/external-name":"provider-token-id"}},"status":{"atProvider":{"pdcNetworkId":"provider-network-id"}}}`),
	}
	config[reconcileTimeConfigKey] = time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	desired, err = renderPDC(claim, observed, config)
	if err != nil {
		t.Fatalf("renderPDC() with observed network error = %v", err)
	}
	network = desired[networkName]
	if got := desiredExternalName(t, network.Resource.UnstructuredContent()); got != "provider-network-id" {
		t.Fatalf("observed PDC external name = %q, want provider-network-id", got)
	}
	token := pdcObject(t, desired, pdcTokenResourceName("primary", 0))
	if got := desiredExternalName(t, token); got != "provider-token-id" {
		t.Fatalf("observed PDC token external name = %q, want provider-token-id", got)
	}
	parameters := nestedMap(t, token, "spec", "forProvider")
	if got, want := parameters["pdcNetworkId"], "provider-network-id"; got != want {
		t.Fatalf("PDC token network ID = %v, want %v", got, want)
	}
	if got, want := parameters["region"], "prod-us-central-0"; got != want {
		t.Fatalf("PDC token region = %v, want platform-owned %v", got, want)
	}
	if got, want := parameters["expiresAt"], "2030-01-31T00:00:00Z"; got != want {
		t.Fatalf("PDC token expiry = %v, want issuance anchor plus bounded lifetime %s", got, want)
	}
	if got, want := nestedMap(t, token, "spec", "writeConnectionSecretToRef")["name"], "example-pdc-pdc-"+stableResourceSuffix("primary")+"-token-0"; got != want {
		t.Fatalf("PDC token connection Secret = %v, want distinct window name %s", got, want)
	}
	datasource := pdcObject(t, desired, resource.Name("datasource-"+stableResourceSuffix("example-pdc:orders")))
	datasourceParameters := nestedMap(t, datasource, "spec", "forProvider")
	if got, want := datasourceParameters["privateDataSourceConnectNetworkId"], "provider-network-id"; got != want {
		t.Fatalf("datasource PDC linkage = %v, want %v", got, want)
	}
	if got, want := datasourceParameters["uid"], "pdc-"+stableResourceSuffix("example-pdc:orders"); got != want {
		t.Fatalf("datasource UID = %v, want deterministic exclusive identity %s", got, want)
	}
	assertPDCProviderConfig(t, network.Resource.UnstructuredContent(), "grafana-cloud-org-example-primary")
	assertPDCProviderConfig(t, token, "grafana-cloud-org-example-primary")
	assertPDCProviderConfig(t, datasource, "teamdemo01")
}

func TestPDCRenewsWithStableWindowsAndRetainsObservedDependentIdentity(t *testing.T) {
	claim := pdcClaim()
	config := pdcConfig("720h")
	config[reconcileTimeConfigKey] = time.Date(2030, 1, 16, 0, 0, 0, 0, time.UTC)
	network := pdcNetworkResourceName("primary")
	firstToken := pdcTokenResourceName("primary", 0)
	observed := map[resource.Name]resource.ObservedComposed{
		network:    pdcObserved(`{"metadata":{"annotations":{"crossplane.io/external-name":"provider-network-id"}},"status":{"atProvider":{}}}`),
		firstToken: pdcObserved(`{"metadata":{"annotations":{"crossplane.io/external-name":"provider-token-id"}},"status":{"atProvider":{"pdcNetworkId":"provider-network-id"}}}`),
	}
	desired, err := renderPDC(claim, observed, config)
	if err != nil {
		t.Fatal(err)
	}
	secondToken := pdcTokenResourceName("primary", 1)
	token := pdcObject(t, desired, secondToken)
	if got, want := nestedMap(t, token, "spec", "forProvider")["expiresAt"], "2030-02-15T00:00:00Z"; got != want {
		t.Fatalf("renewal expiry = %v, want %v", got, want)
	}
	if _, exists := desired[firstToken]; exists {
		t.Fatal("renewal retained the prior token child instead of allowing its bounded expiry")
	}
	if got, want := nestedMap(t, token, "spec", "writeConnectionSecretToRef")["name"], "example-pdc-pdc-"+stableResourceSuffix("primary")+"-token-1"; got != want {
		t.Fatalf("renewed PDC token connection Secret = %v, want distinct window name %s", got, want)
	}
	datasource := pdcObject(t, desired, resource.Name("datasource-"+stableResourceSuffix("example-pdc:orders")))
	if got := nestedMap(t, datasource, "spec", "forProvider")["privateDataSourceConnectNetworkId"]; got != "provider-network-id" {
		t.Fatalf("datasource lost observed PDC network identity: %v", got)
	}

	observed[secondToken] = pdcObserved(`{"metadata":{"annotations":{"crossplane.io/external-name":"provider-token-id-second"}},"status":{"atProvider":{"pdcNetworkId":"provider-network-id"}}}`)
	config[reconcileTimeConfigKey] = time.Date(2030, 1, 30, 23, 59, 59, 0, time.UTC)
	stillDesired, err := renderPDC(claim, observed, config)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := stillDesired[secondToken]; !exists {
		t.Fatal("same renewal window changed the token identity")
	}
	stableToken := pdcObject(t, stillDesired, secondToken)
	if got := desiredExternalName(t, stableToken); got != "provider-token-id-second" {
		t.Fatalf("same-window token external name = %q, want provider-token-id-second", got)
	}
	if got, want := nestedMap(t, stableToken, "spec", "forProvider")["expiresAt"], "2030-02-15T00:00:00Z"; got != want {
		t.Fatalf("same-window expiry = %v, want %v", got, want)
	}
}

func TestPDCWaitsRatherThanMovingAnObservedDatasourceToAnUnreadyNetwork(t *testing.T) {
	claim := pdcClaim()
	claim["spec"].(map[string]any)["networks"] = []any{map[string]any{"name": "primary"}, map[string]any{"name": "secondary"}}
	claim["spec"].(map[string]any)["datasources"].([]any)[0].(map[string]any)["network"] = "secondary"
	logicalDatasource := resource.Name("datasource-" + stableResourceSuffix("example-pdc:orders"))
	observed := map[resource.Name]resource.ObservedComposed{
		logicalDatasource: pdcObserved(`{"spec":{"forProvider":{"privateDataSourceConnectNetworkId":"old-network-id"}}}`),
	}
	if _, err := renderPDC(claim, observed, pdcConfig("720h")); err == nil || !strings.Contains(err.Error(), "waiting for observed PDC network ID for datasource") {
		t.Fatalf("unready requested network error = %v", err)
	}
}

func assertPDCProviderConfig(t *testing.T, object map[string]any, want string) {
	t.Helper()
	if got := nestedMap(t, object, "spec", "providerConfigRef")["name"]; got != want {
		t.Fatalf("providerConfigRef.name = %v, want %s", got, want)
	}
}

func TestPDCRejectsUnconfiguredProfilesAndOverLimitTokenLifetime(t *testing.T) {
	claim := pdcClaim()
	claim["spec"].(map[string]any)["profile"] = "unapproved"
	if _, err := renderPDC(claim, nil, pdcConfig("720h")); err == nil || !strings.Contains(err.Error(), "not configured by the platform") {
		t.Fatalf("unconfigured profile error = %v", err)
	}

	claim = pdcClaim()
	claim["spec"].(map[string]any)["token"].(map[string]any)["expiresAfter"] = "721h"
	if _, err := renderPDC(claim, nil, pdcConfig("720h")); err == nil || !strings.Contains(err.Error(), "exceeds platform maximumTokenLifetime") {
		t.Fatalf("over-limit token error = %v", err)
	}
}

func TestPDCFunctionGatesChildrenAndNeverPublishesTokenValues(t *testing.T) {
	claim := mustJSON(pdcClaim())
	input := mustJSON(pdcConfig("720h"))
	unresolved := callFunctionWithRequiredResourcesAndInput(t, claim, nil, nil, requiredResourceCapabilities(), input)
	if fatal := fatalResult(unresolved); fatal != "" {
		t.Fatalf("unresolved PDC stack returned fatal result: %s", fatal)
	}
	if got := len(unresolved.GetDesired().GetResources()); got != 0 {
		t.Fatalf("unresolved stack rendered %d PDC children, want none", got)
	}

	readyStack := requiredStackResource("teamdemo01", "grafana-vending", "True")
	stack := readyStack.Items[0].Resource.AsMap()
	stack["spec"] = map[string]any{"organization": "example-primary", "region": "prod-us-central-0", "usage": "development"}
	stack["status"].(map[string]any)["stack"] = map[string]any{"id": "provider-stack-id"}
	readyStack.Items[0].Resource = resource.MustStructJSON(mustJSON(stack))
	ready := callFunctionWithRequiredResourcesAndInput(t, claim, nil, readyStack, requiredResourceCapabilities(), input)
	if fatal := fatalResult(ready); fatal != "" {
		t.Fatalf("ready PDC stack returned fatal result: %s", fatal)
	}
	if _, exists := ready.GetDesired().GetResources()[string(pdcNetworkResourceName("primary"))]; !exists {
		t.Fatal("ready PDC stack did not render the network")
	}
	for _, child := range ready.GetDesired().GetResources() {
		encoded, err := json.Marshal(child.GetResource().AsMap())
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(encoded), "never-render-me") {
			t.Fatalf("rendered PDC child carries a token value: %s", encoded)
		}
	}
	if status := ready.GetDesired().GetComposite().GetResource().AsMap()["status"]; status != nil {
		encoded, err := json.Marshal(status)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(strings.ToLower(string(encoded)), "token") {
			t.Fatalf("PDC composite status carries token content: %s", encoded)
		}
	}
}

func TestPDCAdmissionRefusesUnvendedDatasourceNetworksAndTokenLifetimes(t *testing.T) {
	ctx := context.Background()
	env := &admissionEnv{paths: []string{"../apis/pdc-v1beta1.yaml"}}
	if err := env.Start(t); err != nil {
		t.Fatalf("start PDC admission environment: %v", err)
	}
	t.Cleanup(func() {
		if err := env.Stop(); err != nil {
			t.Errorf("stop PDC admission environment: %v", err)
		}
	})

	allowed := pdcAdmissionRequest("pdcallowed", "24h", "primary")
	requireAdmitted(ctx, env, t, allowed, "allowed PDC create")
	networksOnly := pdcAdmissionRequest("pdcnodatasources", "24h", "primary")
	delete(networksOnly.Object["spec"].(map[string]any), "datasources")
	requireAdmitted(ctx, env, t, networksOnly, "PDC networks-only create")

	wrongNetwork := pdcAdmissionRequest("pdcwrongnetwork", "24h", "unvended")
	if err := env.Apply(ctx, wrongNetwork); err == nil || !strings.Contains(err.Error(), "every datasource network must name a network vended by this request") {
		t.Fatalf("unvended datasource network error = %v", err)
	} else {
		t.Logf("unvended datasource network refusal: %v", err)
	}

	update := pdcAdmissionRequest("pdcupdate", "24h", "primary")
	requireAdmitted(ctx, env, t, update, "PDC update baseline")
	datasources, _, err := unstructured.NestedSlice(update.Object, "spec", "datasources")
	if err != nil {
		t.Fatal(err)
	}
	datasources[0].(map[string]any)["network"] = "unvended"
	if err := unstructured.SetNestedSlice(update.Object, datasources, "spec", "datasources"); err != nil {
		t.Fatal(err)
	}
	if err := env.Apply(ctx, update); err == nil || !strings.Contains(err.Error(), "every datasource network must name a network vended by this request") {
		t.Fatalf("unvended datasource update error = %v", err)
	} else {
		t.Logf("unvended datasource update refusal: %v", err)
	}

	tokenUpdate := pdcAdmissionRequest("pdctokenupdate", "24h", "primary")
	requireAdmitted(ctx, env, t, tokenUpdate, "PDC token update baseline")
	tokenUpdate.Object["spec"].(map[string]any)["token"].(map[string]any)["expiresAfter"] = "48h"
	if err := env.Apply(ctx, tokenUpdate); err == nil || !strings.Contains(err.Error(), "token expiry is immutable") {
		t.Fatalf("token expiry update error = %v", err)
	} else {
		t.Logf("PDC token expiry update refusal: %v", err)
	}

	pdcAwaitLifetimeDenial(ctx, env, t, "PDC token expiresAfter exceeds the Composition maximumTokenLifetime")
	pdcRunLifetimeNegativeControl(ctx, env, t)
	// Prove the network-reference rule itself is necessary, independently of the token policy.
	const networkRule = `rule: "!has(self.spec.datasources) || self.spec.datasources.all(datasource, self.spec.networks.exists(network, network.name == datasource.network))"`
	source, err := os.ReadFile("../apis/pdc-v1beta1.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(source), networkRule) != 1 {
		t.Fatal("network rule must occur exactly once")
	}
	scratchPath := filepath.Join(t.TempDir(), "pdc-network-control.yaml")
	if err := os.WriteFile(scratchPath, []byte(strings.Replace(string(source), networkRule, `rule: "true"`, 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	weakenedCRD, err := crdFromXRD(scratchPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := env.Apply(ctx, weakenedCRD); err != nil {
		t.Fatal(err)
	}
	if err := env.Apply(ctx, pdcAdmissionRequest("pdcnetworkweakened", "24h", "unvended")); err != nil {
		t.Fatalf("weakened network control: %v", err)
	}
	t.Log("PDC network rule weakened output: admitted")
	restoredCRD, err := crdFromXRD("../apis/pdc-v1beta1.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := env.Apply(ctx, restoredCRD); err != nil {
		t.Fatal(err)
	}
	if err := env.Apply(ctx, pdcAdmissionRequest("pdcnetworkrestored", "24h", "unvended")); err == nil || !strings.Contains(err.Error(), "every datasource network must name a network vended by this request") {
		t.Fatalf("restored network control: %v", err)
	} else {
		t.Logf("PDC network rule restored output: %v", err)
	}
}

func pdcAwaitLifetimeDenial(ctx context.Context, env *admissionEnv, t *testing.T, message string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for attempt := 0; time.Now().Before(deadline); attempt++ {
		err := env.Apply(ctx, pdcAdmissionRequest("pdcoverlimit"+strconv.Itoa(attempt), "721h", "primary"))
		if err != nil && strings.Contains(err.Error(), message) {
			t.Logf("PDC token ceiling refusal: %v", err)
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("PDC token ceiling policy did not become active with message %q", message)
}

func pdcRunLifetimeNegativeControl(ctx context.Context, env *admissionEnv, t *testing.T) {
	t.Helper()
	const (
		xrdPath      = "../apis/pdc-v1beta1.yaml"
		originalRule = "duration(object.spec.token.expiresAfter) <= duration(params.spec.pipeline[0].input.spec.maximumTokenLifetime)"
		weakenedRule = `"true"`
		message      = "PDC token expiresAfter exceeds the Composition maximumTokenLifetime"
	)
	source, err := os.ReadFile(xrdPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(source), originalRule); got != 1 {
		t.Fatalf("PDC negative-control source rule count = %d, want 1", got)
	}
	scratch := strings.Replace(string(source), originalRule, weakenedRule, 1)
	scratchPath := filepath.Join(t.TempDir(), "pdc-v1beta1.yaml")
	if err := os.WriteFile(scratchPath, []byte(scratch), 0o600); err != nil {
		t.Fatal(err)
	}
	weakenedPolicy := pdcAdmissionDocument(t, scratchPath, "ValidatingAdmissionPolicy")
	if err := env.Apply(ctx, weakenedPolicy); err != nil {
		t.Fatalf("apply weakened PDC lifetime policy: %v", err)
	}
	pdcAwaitLifetimeAdmission(ctx, env, t)

	restoredPolicy := pdcAdmissionDocument(t, xrdPath, "ValidatingAdmissionPolicy")
	if err := env.Apply(ctx, restoredPolicy); err != nil {
		t.Fatalf("restore PDC lifetime policy: %v", err)
	}
	pdcAwaitLifetimeDenial(ctx, env, t, message)
}

func pdcAwaitLifetimeAdmission(ctx context.Context, env *admissionEnv, t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for attempt := 0; time.Now().Before(deadline); attempt++ {
		err := env.Apply(ctx, pdcAdmissionRequest("pdcweakened"+strconv.Itoa(attempt), "721h", "primary"))
		if err == nil {
			t.Log("PDC weakened ceiling output: admitted")
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("PDC weakened ceiling policy did not admit the over-limit control")
}

func pdcAdmissionDocument(t *testing.T, path, kind string) *unstructured.Unstructured {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(raw), 4096)
	for {
		object := &unstructured.Unstructured{}
		err := decoder.Decode(&object.Object)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if object.GetKind() == kind {
			return object
		}
	}
	t.Fatalf("%s does not contain %s", path, kind)
	return nil
}

func pdcClaim() map[string]any {
	return map[string]any{
		"apiVersion": "platform.example.org/v1beta1",
		"kind":       "GrafanaPDC",
		"metadata": map[string]any{
			"name": "example-pdc", "namespace": "grafana-vending", "creationTimestamp": "2030-01-01T00:00:00Z",
		},
		"spec": map[string]any{
			"stackRef": map[string]any{"name": "teamdemo01"}, "profile": "standard",
			"networks": []any{map[string]any{"name": "primary"}},
			"token":    map[string]any{"expiresAfter": "720h"},
			"datasources": []any{map[string]any{
				"name": "orders", "type": "prometheus", "network": "primary", "url": "https://datasource.example.invalid",
			}},
		},
	}
}

func pdcConfig(maximum string) map[string]any {
	return map[string]any{"referencedStack": map[string]any{
		"name": "teamdemo01", "namespace": "grafana-vending", "organizationProviderConfigName": "grafana-cloud-org-example-primary",
	}, "spec": map[string]any{
		"maximumTokenLifetime": maximum,
		"organizations":        []any{map[string]any{"name": "example-primary", "providerConfigName": "grafana-cloud-org-example-primary", "allowedRegions": []any{"prod-us-central-0"}, "allowedUsages": []any{"development"}}},
		"pdcProfiles":          []any{map[string]any{"name": "standard", "region": "prod-us-central-0"}},
	}}
}

func pdcObserved(document string) resource.ObservedComposed {
	value := composed.New()
	value.SetUnstructuredContent(resource.MustStructJSON(document).AsMap())
	return resource.ObservedComposed{Resource: value}
}

func pdcObject(t *testing.T, desired map[resource.Name]*resource.DesiredComposed, name resource.Name) map[string]any {
	t.Helper()
	child, ok := desired[name]
	if !ok {
		t.Fatalf("PDC desired resource %q was not rendered", name)
	}
	return child.Resource.UnstructuredContent()
}

func pdcAdmissionRequest(name, expiresAfter, network string) *unstructured.Unstructured {
	return requestObject("GrafanaPDC", name, map[string]any{
		"stackRef": map[string]any{"name": "teamdemo01"}, "profile": "standard",
		"networks": []any{map[string]any{"name": "primary"}},
		"token":    map[string]any{"expiresAfter": expiresAfter},
		"datasources": []any{map[string]any{
			"name": "orders", "type": "prometheus", "network": network,
		}},
	})
}

func TestPDCTokenRenewalRequiresCreationAnchor(t *testing.T) {
	_, err := pdcTokenIssuance(map[string]any{}, map[string]any{reconcileTimeConfigKey: time.Now()}, time.Hour)
	if err == nil || !strings.Contains(err.Error(), "creationTimestamp") {
		t.Fatalf("missing creation anchor must refuse: %v", err)
	}
}
