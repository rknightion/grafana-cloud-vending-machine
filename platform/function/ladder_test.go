package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
)

func TestRenderStackLadderRendersEachRungAndRepository(t *testing.T) {
	desired, err := renderStackLadder(ladderTestXR(nil), nil, ladderTestConfig(3))
	if err != nil {
		t.Fatalf("renderStackLadder returned an error: %v", err)
	}

	for _, name := range []resource.Name{"rung-development-stack", "rung-production-stack"} {
		child, found := desired[name]
		if !found {
			t.Fatalf("%s was not rendered", name)
		}
		if got := child.Resource.UnstructuredContent()["kind"]; got != "Stack" {
			t.Errorf("%s kind = %v, want Stack", name, got)
		}
	}
	for name, want := range map[resource.Name]map[string]string{
		"rung-development-repository": {"branch": "development", "connection": "development-connection"},
		"rung-production-repository":  {"branch": "main", "connection": "production-connection"},
	} {
		child, found := desired[name]
		if !found {
			t.Fatalf("%s was not rendered", name)
		}
		content := child.Resource.UnstructuredContent()
		if got := content["kind"]; got != "RepositoryV0Alpha1" {
			t.Errorf("%s kind = %v, want RepositoryV0Alpha1", name, got)
		}
		forProvider := ladderNestedMap(t, content, "spec", "forProvider")
		github := ladderNestedMap(t, forProvider, "spec", "github")
		if got := github["branch"]; got != want["branch"] {
			t.Errorf("%s branch = %v, want %s", name, got, want["branch"])
		}
		if got := ladderNestedMap(t, forProvider, "spec", "connection")["name"]; got != want["connection"] {
			t.Errorf("%s connection = %v, want %s", name, got, want["connection"])
		}
	}
}

func TestRenderStackLadderRejectsPlatformCap(t *testing.T) {
	if _, err := renderStackLadder(ladderTestXR(nil), nil, ladderTestConfig(1)); err == nil || !strings.Contains(err.Error(), "platform maximum") {
		t.Fatalf("ladder above the platform cap returned %v, want cap error", err)
	}
}

func TestStackLadderStatusReportsDirectionReadinessAndDrift(t *testing.T) {
	xr := ladderTestXR(map[string]any{"promotionDirection": "reverse"})
	observed := map[resource.Name]resource.ObservedComposed{
		"rung-development-stack":      ladderObserved(`{"status":{"conditions":[{"type":"Ready","status":"True"}]}}`),
		"rung-production-stack":       ladderObserved(`{"status":{"conditions":[{"type":"Ready","status":"True"}]}}`),
		"rung-development-repository": ladderObserved(`{"status":{"atProvider":{"metadata":{"version":"revision-a"}}}}`),
		"rung-production-repository":  ladderObserved(`{"status":{"atProvider":{"metadata":{"version":"revision-b"}}}}`),
	}

	status := desiredStackLadderStatus(xr, observed).Resource.UnstructuredContent()["status"].(map[string]any)
	if got := status["promotionDirection"]; got != "reverse" {
		t.Fatalf("promotion direction = %v, want reverse", got)
	}
	rungs := status["rungs"].([]any)
	if got, want := rungs[1].(map[string]any)["drift"], "Source"; got != want {
		t.Errorf("reverse source drift = %v, want %s", got, want)
	}
	development := rungs[0].(map[string]any)
	if got, want := development["sourceRung"], "production"; got != want {
		t.Errorf("development source rung = %v, want %s", got, want)
	}
	if got, want := development["drift"], "Drifted"; got != want {
		t.Errorf("development drift = %v, want %s", got, want)
	}
	if got, want := development["ready"], true; got != want {
		t.Errorf("development ready = %v, want %v", got, want)
	}
}

func TestStackLadderStatusMarksUnknownWithoutObservedRevision(t *testing.T) {
	status := desiredStackLadderStatus(ladderTestXR(nil), nil).Resource.UnstructuredContent()["status"].(map[string]any)
	rungs := status["rungs"].([]any)
	if got, want := rungs[0].(map[string]any)["drift"], "Source"; got != want {
		t.Errorf("source drift = %v, want %s", got, want)
	}
	if got, want := rungs[1].(map[string]any)["drift"], "Unknown"; got != want {
		t.Errorf("unobserved drift = %v, want %s", got, want)
	}
}

func ladderTestXR(additions map[string]any) map[string]any {
	spec := map[string]any{
		"organization": "example-primary", "region": "prod-us-central-0", "promotionDirection": "forward",
		"repository": map[string]any{"url": "https://github.com/example/grafana-dashboards", "path": "platform"},
		"rungs": []any{
			map[string]any{"name": "development", "slug": "ladderdev01", "displayName": "Example development", "usage": "development", "branch": "development", "connectionRef": map[string]any{"name": "development-connection"}},
			map[string]any{"name": "production", "slug": "ladderprod01", "displayName": "Example production", "usage": "production", "branch": "main", "connectionRef": map[string]any{"name": "production-connection"}},
		},
	}
	for key, value := range additions {
		spec[key] = value
	}
	return map[string]any{
		"apiVersion": "platform.example.org/v1beta1", "kind": "GrafanaStackLadder",
		"metadata": map[string]any{"name": "example-ladder", "namespace": "grafana-vending"},
		"spec":     spec,
	}
}

func ladderTestConfig(maxStacks int) map[string]any {
	return map[string]any{"spec": map[string]any{
		"maximumTokenLifetime": "720h",
		"ladderPolicy":         map[string]any{"maxStacks": int64(maxStacks)},
		"allowedUsages":        []any{"development", "production"},
		"organizations": []any{map[string]any{
			"name": "example-primary", "providerConfigName": "organization-provider",
			"allowedRegions": []any{"prod-us-central-0"}, "allowedUsages": []any{"development", "production"},
		}},
	}}
}

func ladderObserved(document string) resource.ObservedComposed {
	return resource.ObservedComposed{Resource: func() *composed.Unstructured {
		value := composed.New()
		value.SetUnstructuredContent(resource.MustStructJSON(document).AsMap())
		return value
	}()}
}

func ladderNestedMap(t *testing.T, object map[string]any, fields ...string) map[string]any {
	t.Helper()
	current := object
	for _, field := range fields {
		next, ok := current[field].(map[string]any)
		if !ok {
			t.Fatalf("%s is not an object in %#v", field, current)
		}
		current = next
	}
	return current
}

func TestLadderStatusUsesDeclaredRungOrder(t *testing.T) {
	status := desiredStackLadderStatus(ladderTestXR(nil), nil).Resource.UnstructuredContent()["status"].(map[string]any)
	rungs := status["rungs"].([]any)
	got := []string{rungs[0].(map[string]any)["name"].(string), rungs[1].(map[string]any)["name"].(string)}
	if want := []string{"development", "production"}; !reflect.DeepEqual(got, want) {
		t.Errorf("status rung order = %v, want %v", got, want)
	}
}
