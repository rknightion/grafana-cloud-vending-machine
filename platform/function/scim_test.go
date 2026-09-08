package main

import (
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
	"strings"
	"testing"
)

func TestSCIMStackRequestsRejected(t *testing.T) {
	for _, value := range []any{map[string]any{"enableGroupSync": true}, map[string]any{}, nil, false} {
		rsp := callFunction(t, stackDocument(map[string]any{"scim": value}), nil, "")
		if len(rsp.GetResults()) == 0 || !strings.Contains(rsp.GetResults()[0].GetMessage(), "SCIM is out of scope") {
			t.Fatalf("unexpected result: %v", rsp.GetResults())
		}
		if len(rsp.GetDesired().GetResources()) != 0 {
			t.Fatal("SCIM request rendered children")
		}
	}
}

func TestSCIMExternalGroupMutualExclusion(t *testing.T) {
	for _, kind := range []string{"GrafanaTeamAccess", "GrafanaCustomRoleBinding"} {
		groupKey := "externalGroups"
		if kind == "GrafanaCustomRoleBinding" {
			groupKey = "groups"
		}
		xr := map[string]any{"apiVersion": "platform.example.org/v1beta1", "kind": kind, "metadata": map[string]any{"name": "access", "namespace": "grafana-vending"}, "spec": map[string]any{"stackRef": map[string]any{"name": "teamdemo01"}, "team": map[string]any{groupKey: []any{"example-group"}}}}
		stack := requiredStackResource("teamdemo01", "grafana-vending", "True")
		obj := stack.Items[0].Resource.AsMap()
		obj["spec"] = map[string]any{"scim": map[string]any{}}
		stack.Items[0].Resource = resource.MustStructJSON(mustJSON(obj))
		req := &fnv1.RunFunctionRequest{RequiredResources: map[string]*fnv1.Resources{referencedStackRequirement: stack}}
		_, err := accessResourcesAdmittedWithRequest(req, &fnv1.RunFunctionResponse{}, xr)
		if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
			t.Fatalf("%s error=%v", kind, err)
		}
	}
}
