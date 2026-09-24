package main

import (
	"context"
	"strings"
	"testing"

	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
)

func projectContentComplete(t *testing.T) (map[string]any, map[string]any, map[resource.Name]resource.ObservedComposed, map[resource.Name]*resource.DesiredComposed) {
	t.Helper()
	xr, config := projectContentFixture()
	config[provisioningConnectionCredentialConfigKey] = "ZXhhbXBsZQ=="
	observed := map[resource.Name]resource.ObservedComposed{}
	var desired map[resource.Name]*resource.DesiredComposed
	for stage := 0; stage < 6; stage++ {
		var err error
		desired, err = renderProjectContent(xr, observed, config)
		if err != nil {
			t.Fatalf("stage %d: %v", stage, err)
		}
		for key, child := range desired {
			at := map[string]any{}
			switch key {
			case projectContentObserver:
				at = map[string]any{"id": "54321", "slug": "exampleproject", "regionSlug": "prod-us-central-0"}
			case stackConsumerObserverName:
				at = map[string]any{"id": "12345", "slug": "exampleobservedstack", "regionSlug": "prod-us-central-0", "prometheusUrl": "https://metrics.example.invalid", "prometheusUserId": float64(12345), "logsUrl": "https://logs.example.invalid", "logsUserId": float64(67890)}
			case stackConsumerPolicyName:
				at = map[string]any{"policyId": "67890"}
			}
			observed[key] = stackConsumerObserved(child.Resource.UnstructuredContent(), map[string]any{"status": map[string]any{"atProvider": at, "conditions": []any{map[string]any{"type": "Ready", "status": "True"}, map[string]any{"type": "Synced", "status": "True"}}}})
		}
	}
	return xr, config, observed, desired
}

func TestProjectContentCompleteCredentialAndGitContract(t *testing.T) {
	_, config, _, desired := projectContentComplete(t)
	if len(desired) != 13 {
		t.Fatalf("resource count = %d, want bounded 13", len(desired))
	}
	realm := nestedMap(t, stackConsumerDesired(t, desired, stackConsumerPolicyName), "spec", "forProvider")["realm"].([]any)[0].(map[string]any)
	if realm["identifier"] != "12345" || mustJSON(realm["labelPolicy"]) != `[{"selector":"{project=\"example\"}"}]` {
		t.Fatalf("central credential restriction = %s", mustJSON(realm))
	}
	for _, key := range []resource.Name{"metrics-datasource", "logs-datasource", "folder", "repository", "git-connection"} {
		object := stackConsumerDesired(t, desired, key)
		if nestedMap(t, object, "spec", "providerConfigRef")["name"] != "project-provider" {
			t.Fatalf("%s routed outside project provider", key)
		}
		if nestedMap(t, object, "metadata", "annotations")["crossplane.io/external-name"] == nil {
			t.Fatalf("%s omitted deterministic import identity", key)
		}
	}
	secure := nestedMap(t, stackConsumerDesired(t, desired, "git-connection"), "spec", "forProvider", "secure", "privateKey")
	if secure["create"] != "ZXhhbXBsZQ==" || secure["name"] != nil {
		t.Fatalf("Git Sync must use accepted create form: %v", secure)
	}
	_, ready := desiredProjectContentStatus(config)
	if !ready {
		t.Fatal("complete observed content did not become ready")
	}
}

func TestProjectContentRefusalIsAttributedAndNoPartialSuccess(t *testing.T) {
	for _, tc := range []struct {
		child resource.Name
		side  string
	}{{stackConsumerPolicyName, "centralStack"}, {"metrics-datasource", "projectStack"}} {
		t.Run(tc.side, func(t *testing.T) {
			xr, config, observed, _ := projectContentComplete(t)
			object := observed[tc.child].Resource.UnstructuredContent()
			object["status"] = map[string]any{"conditions": []any{map[string]any{"type": "Synced", "status": "False", "message": "sensitive-provider-error"}}}
			desired, err := renderProjectContent(xr, observed, config)
			if err != nil || len(desired) != 13 {
				t.Fatalf("unsafe refusal: %v, %d resources", err, len(desired))
			}
			status, ready := desiredProjectContentStatus(config)
			if ready || nestedMap(t, status.Resource.UnstructuredContent(), "status", "projectContent")[tc.side] != "Refused" {
				t.Fatal("refusal did not reach side-specific unready status")
			}
		})
	}
}

func TestRunFunctionProjectContentRefusalPreservesDesiredChildren(t *testing.T) {
	for _, tc := range []struct {
		name          string
		refused       resource.Name
		missing       resource.Name
		refusedSide   string
		policyDrift   bool
		policyMissing bool
		centralDrift  bool
		absentProject bool
	}{
		{name: "central refusal", refused: stackConsumerPolicyName, refusedSide: "centralStack"},
		{name: "project refusal", refused: "metrics-datasource", refusedSide: "projectStack"},
		{name: "dependency disappearance", missing: stackConsumerObserverName, refusedSide: "centralStack"},
		{name: "central observer identity drift", centralDrift: true, refusedSide: "centralStack"},
		{name: "project identity disappearance", missing: projectContentObserver, refusedSide: "projectStack"},
		{name: "project observer absent", absentProject: true, refusedSide: "projectStack"},
		{name: "project observer refusal", refused: projectContentObserver, refusedSide: "projectStack"},
		{name: "policy external identity mismatch", policyDrift: true, refusedSide: "centralStack"},
		{name: "opposite side refusal with policy identity recovery", refused: projectContentObserver, policyMissing: true, refusedSide: "centralStack"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			xr, config, observed, desired := projectContentComplete(t)
			if tc.refused != "" {
				object := observed[tc.refused].Resource.UnstructuredContent()
				object["status"] = map[string]any{"conditions": []any{map[string]any{"type": "Synced", "status": "False", "message": "sensitive-provider-error"}}}
			}
			if tc.missing != "" {
				delete(nestedMap(t, observed[tc.missing].Resource.UnstructuredContent(), "status", "atProvider"), "id")
			}
			if tc.centralDrift {
				nestedMap(t, observed[stackConsumerObserverName].Resource.UnstructuredContent(), "metadata", "annotations")["crossplane.io/external-name"] = "drifted-stack"
			}
			if tc.policyDrift {
				nestedMap(t, observed[stackConsumerPolicyName].Resource.UnstructuredContent(), "metadata")["annotations"] = map[string]any{"crossplane.io/external-name": "prod-us-central-0:99999"}
			}
			if tc.policyMissing {
				delete(nestedMap(t, observed[stackConsumerPolicyName].Resource.UnstructuredContent(), "status", "atProvider"), "policyId")
			}
			if len(observed) != 13 {
				t.Fatalf("observed fixture = %d, want 13", len(observed))
			}
			if tc.absentProject {
				delete(observed, projectContentObserver)
			}
			// Refusal must re-author the configuration, not endorse observed drift.
			nestedMap(t, observed["logs-datasource"].Resource.UnstructuredContent(), "spec", "providerConfigRef")["name"] = "drifted-provider"
			before := make(map[string]string, len(desired))
			for name, child := range desired {
				before[string(name)] = mustJSON(child.Resource.UnstructuredContent())
			}
			req := projectContentRequest(t, xr, config, observed)
			if len(req.Desired.Resources) != 0 {
				t.Fatal("incoming desired must be empty")
			}
			incomingDesired := len(req.Desired.Resources)
			rsp, err := (&Function{log: logging.NewNopLogger()}).RunFunction(context.Background(), req)
			t.Logf("observed=%d incomingDesired=%d outputDesired=%d fatal=%q", len(observed), incomingDesired, len(rsp.GetDesired().GetResources()), fatalResult(rsp))
			if err != nil {
				t.Fatal(err)
			}
			if fatal := fatalResult(rsp); fatal != "" {
				t.Fatalf("refusal returned fatal result: %s", fatal)
			}
			if got, want := len(rsp.GetDesired().GetResources()), len(desired); got != want {
				t.Fatalf("preserved resources = %d, want %d", got, want)
			}
			for name, beforeChild := range before {
				after := rsp.GetDesired().GetResources()[name]
				if after == nil || mustJSON(after.GetResource().AsMap()) != beforeChild {
					t.Fatalf("desired child %q changed during refusal: before=%s after=%s", name, beforeChild, mustJSON(after.GetResource().AsMap()))
				}
			}
			composite := rsp.GetDesired().GetComposite()
			if composite.GetReady() != fnv1.Ready_READY_FALSE {
				t.Fatalf("composite readiness = %s, want false", composite.GetReady())
			}
			status := nestedMap(t, composite.GetResource().AsMap(), "status", "projectContent")
			if status[tc.refusedSide] != "Refused" {
				t.Fatalf("%s status = %v, want Refused", tc.refusedSide, status[tc.refusedSide])
			}
			if len(rsp.GetResults()) != 1 || rsp.GetResults()[0].GetSeverity() != fnv1.Severity_SEVERITY_WARNING || strings.Contains(mustJSON(rsp), "sensitive-provider-error") {
				t.Fatalf("missing or unsafe refusal warning: %v", rsp.GetResults())
			}
		})
	}
}

func projectContentRequest(t *testing.T, xr, config map[string]any, observed map[resource.Name]resource.ObservedComposed) *fnv1.RunFunctionRequest {
	t.Helper()
	return &fnv1.RunFunctionRequest{
		Observed: &fnv1.State{
			Composite: &fnv1.Resource{Resource: resource.MustStructJSON(mustJSON(xr))},
			Resources: projectContentProtoResources(t, observed),
		},
		Desired: &fnv1.State{}, Input: resource.MustStructJSON(mustJSON(config)),
		RequiredResources: map[string]*fnv1.Resources{
			provisioningConnectionCredentialRequirement: {Items: []*fnv1.Resource{{Resource: resource.MustStructJSON(mustJSON(map[string]any{
				"apiVersion": "v1", "kind": "Secret",
				"metadata": map[string]any{"name": "example-reader-git-credential", "namespace": "consumer-platform"},
				"data":     map[string]any{provisioningConnectionCredentialKey: "ZXhhbXBsZQ=="},
			}))}}},
		},
	}
}

func TestRunFunctionProjectContentUnrenderable(t *testing.T) {
	for _, dependency := range []string{"assigned-identity", "git-credential", "backend-endpoint", "conflicting-policy-identity"} {
		t.Run(dependency, func(t *testing.T) {
			xr, config, observed, _ := projectContentComplete(t)
			if dependency == "conflicting-policy-identity" {
				delete(nestedMap(t, observed[stackConsumerPolicyName].Resource.UnstructuredContent(), "status", "atProvider"), "policyId")
				nestedMap(t, observed[stackConsumerTokenName].Resource.UnstructuredContent(), "spec", "forProvider")["accessPolicyId"] = "99999"
			}
			if dependency == "assigned-identity" {
				delete(nestedMap(t, observed[stackConsumerObserverName].Resource.UnstructuredContent(), "status", "atProvider"), "id")
				realm := nestedMap(t, observed[stackConsumerPolicyName].Resource.UnstructuredContent(), "spec", "forProvider")["realm"].([]any)[0].(map[string]any)
				delete(realm, "identifier")
			}
			if dependency == "backend-endpoint" {
				delete(nestedMap(t, observed[stackConsumerObserverName].Resource.UnstructuredContent(), "status", "atProvider"), "prometheusUrl")
				// The old child still carries a URL, but it is not an authorized
				// recovery source for configuration. Do not endorse its drift.
				nestedMap(t, observed["metrics-datasource"].Resource.UnstructuredContent(), "spec", "forProvider")["url"] = "https://drifted.example.invalid"
			}
			req := projectContentRequest(t, xr, config, observed)
			if dependency == "git-credential" {
				req.RequiredResources = nil
			}
			rsp, err := (&Function{log: logging.NewNopLogger()}).RunFunction(context.Background(), req)
			if err != nil || fatalResult(rsp) == "" || len(rsp.GetResults()) != 1 {
				t.Fatalf("unrenderable request did not stop composition: fatal=%q resources=%d results=%d error=%v", fatalResult(rsp), len(rsp.GetDesired().GetResources()), len(rsp.GetResults()), err)
			}
			if rsp.GetDesired().GetComposite().GetReady() != fnv1.Ready_READY_FALSE {
				t.Fatal("unrenderable response was marked ready")
			}
		})
	}
}

func TestRunFunctionProjectContentInitialRefusal(t *testing.T) {
	xr, config := projectContentFixture()
	delete(config["spec"].(map[string]any), "projectContentProfiles")
	req := &fnv1.RunFunctionRequest{
		Observed: &fnv1.State{Composite: &fnv1.Resource{Resource: resource.MustStructJSON(mustJSON(xr))}},
		Desired:  &fnv1.State{}, Input: resource.MustStructJSON(mustJSON(config)),
	}
	rsp, err := (&Function{log: logging.NewNopLogger()}).RunFunction(context.Background(), req)
	if err != nil || fatalResult(rsp) != "" || len(rsp.GetDesired().GetResources()) != 0 {
		t.Fatalf("initial refusal: response=%v error=%v", rsp, err)
	}
	composite := rsp.GetDesired().GetComposite()
	if composite.GetReady() != fnv1.Ready_READY_FALSE || nestedMap(t, composite.GetResource().AsMap(), "status", "projectContent")["projectStack"] != "Refused" {
		t.Fatalf("initial refusal status=%v", composite)
	}
	if len(rsp.GetResults()) != 1 || rsp.GetResults()[0].GetSeverity() != fnv1.Severity_SEVERITY_WARNING {
		t.Fatalf("initial refusal warning=%v", rsp.GetResults())
	}
}

func projectContentProtoResources(t *testing.T, observed map[resource.Name]resource.ObservedComposed) map[string]*fnv1.Resource {
	t.Helper()
	result := make(map[string]*fnv1.Resource, len(observed))
	for name, child := range observed {
		value, err := resource.AsStruct(child.Resource)
		if err != nil {
			t.Fatal(err)
		}
		result[string(name)] = &fnv1.Resource{Resource: value}
	}
	return result
}

func TestProjectContentSecurityBoundaries(t *testing.T) {
	for _, mutation := range []string{"namespace", "write-scope", "empty-label", "same-stack", "shared-consumer", "invalid-git"} {
		t.Run(mutation, func(t *testing.T) {
			xr, config := projectContentFixture()
			cs := config["spec"].(map[string]any)
			p := cs["projectContentProfiles"].([]any)[0].(map[string]any)
			switch mutation {
			case "namespace":
				xr["metadata"].(map[string]any)["namespace"] = "other"
			case "write-scope":
				cs["stackConsumerProfiles"].([]any)[0].(map[string]any)["scopes"] = []any{"metrics:write", "logs:read"}
			case "empty-label":
				p["projectLabel"].(map[string]any)["value"] = ""
			case "same-stack":
				p["projectStack"].(map[string]any)["slug"] = "exampleobservedstack"
			case "shared-consumer":
				cs["projectContentProfiles"] = append(cs["projectContentProfiles"].([]any), map[string]any{"name": "other", "consumerProfile": "example-reader"})
			case "invalid-git":
				p["git"].(map[string]any)["url"] = "http://example.invalid"
			}
			if desired, err := renderProjectContent(xr, nil, config); err == nil || len(desired) != 0 {
				t.Fatalf("%s did not fail closed", mutation)
			}
		})
	}
	xr, config, observed, _ := projectContentComplete(t)
	delete(observed, stackConsumerObserverName)
	if desired, err := renderProjectContent(xr, observed, config); err == nil || desired != nil {
		t.Fatal("missing authoritative backend endpoints did not stop composition")
	}
}

func TestProjectContentProviderAdmission(t *testing.T) {
	e := &admissionEnv{paths: []string{"../apis/project-content-v1beta1.yaml"}}
	if err := e.Start(t); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := e.Stop(); err != nil {
			t.Error(err)
		}
	})
	installProviderAdmissionCRDs(t, e)
	_, _, _, desired := projectContentComplete(t)
	supported := map[resource.Name]*resource.DesiredComposed{}
	for key, child := range desired {
		switch child.Resource.GetKind() {
		case "Folder", "ExternalSecret", "PushSecret":
			t.Logf("NO SCHEMA COVERAGE: %s %s is absent from the provider fixture", key, child.Resource.GetKind())
		default:
			supported[key] = child
		}
	}
	if len(supported) != 9 {
		t.Fatalf("provider fixture coverage = %d, want 9 children", len(supported))
	}
	admitProviderChildren(t, e, supported)
}

func TestProjectContentCredentialConfigPreservesAlias(t *testing.T) {
	xr, config := projectContentFixture()
	config[expiryStatusConfigKey] = map[string]any{"effectiveExpiresAt": "2030-01-01T00:00:00Z"}
	rsp := &fnv1.RunFunctionResponse{}
	result := projectContentCredentialConfig(&fnv1.RunFunctionRequest{}, rsp, xr, config)
	result["alias-proof"] = true
	if config["alias-proof"] != true || result[expiryStatusConfigKey] == nil {
		t.Fatal("credential config lost map identity or expiry status")
	}
	selector := rsp.GetRequirements().GetResources()[provisioningConnectionCredentialRequirement]
	if selector.GetMatchName() != "example-reader-git-credential" || selector.GetNamespace() != "consumer-platform" {
		t.Fatalf("incorrect credential selector: %v", selector)
	}
}

func projectContentFixture() (map[string]any, map[string]any) {
	xr := map[string]any{"kind": "GrafanaProjectContent", "metadata": map[string]any{"name": "example-reader", "namespace": "consumer-platform"}, "spec": map[string]any{"profile": "example-reader"}}
	config := stackConsumerConfig()
	config["spec"].(map[string]any)["projectContentProfiles"] = []any{map[string]any{
		"name": "example-reader", "namespace": "consumer-platform", "consumerProfile": "example-reader",
		"projectStack": map[string]any{"slug": "exampleproject", "region": "prod-us-central-0", "providerConfigName": "project-provider"},
		"centralStack": map[string]any{"slug": "exampleobservedstack", "region": "prod-us-central-0"},
		"projectLabel": map[string]any{"name": "project", "value": "example"},
		"git":          map[string]any{"title": "Example", "type": "github", "description": "Example Git Sync", "url": "https://github.com", "github": map[string]any{"appId": "example-app", "installationId": "example-installation"}, "credential": map[string]any{"remoteRef": map[string]any{"key": "example/git", "property": "privateKey"}}, "decrypters": []any{"grafana"}, "secureVersion": float64(1)},
		"repository":   map[string]any{"uid": "example-repository", "title": "Example", "type": "github", "github": map[string]any{"url": "https://github.com/example/project", "branch": "main", "path": "dashboards"}, "sync": map[string]any{"enabled": true, "target": "folder", "intervalSeconds": float64(60)}, "workflows": []any{}},
	}}
	config["spec"].(map[string]any)["stackConsumerProfiles"].([]any)[0].(map[string]any)["scopes"] = []any{"metrics:read", "logs:read"}
	return xr, config
}

func TestProjectContentWaitsForBothIdentities(t *testing.T) {
	xr, config := projectContentFixture()
	desired, err := renderProjectContent(xr, nil, config)
	if err != nil {
		t.Fatal(err)
	}
	if len(desired) != 2 {
		t.Fatalf("initial resources = %d, want only two observe-only stacks", len(desired))
	}
	for name, child := range desired {
		if child.Resource.GetKind() != "Stack" || mustJSON(nestedMap(t, child.Resource.UnstructuredContent(), "spec")["managementPolicies"]) != `["Observe"]` {
			t.Fatalf("%s is not an observe-only stack", name)
		}
	}
}

func TestProjectContentRejectsMissingProfile(t *testing.T) {
	xr, config := projectContentFixture()
	delete(config["spec"].(map[string]any), "projectContentProfiles")
	if desired, err := renderProjectContent(xr, map[resource.Name]resource.ObservedComposed{}, config); err == nil || len(desired) != 0 {
		t.Fatal("unapproved project produced content or did not fail")
	}
}
