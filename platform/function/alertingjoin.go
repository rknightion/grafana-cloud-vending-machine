package main

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"strconv"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composite"
)

// resolveAlertingReceivers replaces reference destinations only in the renderer
// input. The API object retains its reference, so a changed provider address is
// reconciled without a human copying it into a contact point.
func resolveAlertingReceivers(req *fnv1.RunFunctionRequest, rsp *fnv1.RunFunctionResponse, xr map[string]any) (map[string]any, bool, error) {
	raw, err := json.Marshal(xr)
	if err != nil {
		return nil, false, err
	}
	var resolved map[string]any
	if err := json.Unmarshal(raw, &resolved); err != nil {
		return nil, false, err
	}
	spec, _ := resolved["spec"].(map[string]any)
	meta, _ := resolved["metadata"].(map[string]any)
	namespace := stringValue(meta, "namespace", "")
	points, _ := spec["contactPoints"].([]any)
	ready := true
	for _, raw := range points {
		point, _ := raw.(map[string]any)
		ref, isRef := point["onCallRef"].(map[string]any)
		_, hasEmail := point["email"]
		if isRef == hasEmail {
			return nil, false, fmt.Errorf("contact point must select exactly one of email or onCallRef")
		}
		if !isRef {
			continue
		}
		name := stringValue(ref, "name", "")
		if name == "" || namespace == "" {
			return nil, false, fmt.Errorf("onCallRef requires a name and request namespace")
		}
		key := "oncall-receiver-" + name
		if rsp.Requirements == nil {
			rsp.Requirements = &fnv1.Requirements{}
		}
		if rsp.Requirements.Resources == nil {
			rsp.Requirements.Resources = map[string]*fnv1.ResourceSelector{}
		}
		rsp.Requirements.Resources[key] = &fnv1.ResourceSelector{ApiVersion: "platform.example.org/v1beta1", Kind: "GrafanaOnCall", Namespace: &namespace, Match: &fnv1.ResourceSelector_MatchName{MatchName: name}}
		items, available, err := request.GetRequiredResource(req, key)
		if err != nil {
			return nil, false, err
		}
		if !available || len(items) == 0 {
			ready = false
			continue
		}
		if len(items) != 1 || items[0].Resource == nil {
			return nil, false, fmt.Errorf("onCallRef must resolve exactly one object")
		}
		object := items[0].Resource.UnstructuredContent()
		om, _ := object["metadata"].(map[string]any)
		if object["apiVersion"] != "platform.example.org/v1beta1" || object["kind"] != "GrafanaOnCall" || om["name"] != name || om["namespace"] != namespace || accessStackReference(object) != accessStackReference(xr) {
			return nil, false, fmt.Errorf("onCallRef identity or stack binding mismatch")
		}
		status, _ := object["status"].(map[string]any)
		receiver, _ := status["alertReceiver"].(map[string]any)
		address := stringValue(receiver, "address", "")
		if stringValue(receiver, "stackName", "") != accessStackReference(xr) {
			return nil, false, fmt.Errorf("OnCall receiver status stack binding mismatch")
		}
		if !requiredStackReady(object) || stringValue(receiver, "integrationID", "") == "" || fmt.Sprint(receiver["observedGeneration"]) != fmt.Sprint(om["generation"]) {
			ready = false
			continue
		}
		parsed, err := mail.ParseAddress(address)
		if err != nil || parsed.Address != address {
			return nil, false, fmt.Errorf("observed OnCall inbound email address is invalid")
		}
		point["email"] = map[string]any{"addresses": []any{address}}
		delete(point, "onCallRef")
	}
	return resolved, ready, nil
}

func onCallReceiverStatus(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, desired map[resource.Name]*resource.DesiredComposed) (*resource.Composite, bool) {
	result := composite.New()
	result.SetUnstructuredContent(map[string]any{"status": map[string]any{"alertReceiver": nil}})
	status := &resource.Composite{Resource: result, Ready: resource.ReadyUnspecified}
	if desired["catch-all-route"] == nil {
		return status, false
	}
	for key := range desired {
		child, ok := observed[key]
		if !ok || !observedDesiredCurrent(child, desired[key]) {
			return status, false
		}
	}
	child := observed["integration"].Resource.UnstructuredContent()
	integrationSpec, _ := child["spec"].(map[string]any)
	provider, _ := integrationSpec["providerConfigRef"].(map[string]any)
	if provider["name"] != accessStackReference(xr) {
		return status, false
	}
	cs, _ := child["status"].(map[string]any)
	at, _ := cs["atProvider"].(map[string]any)
	address := stringValue(at, "inboundEmail", "")
	parsed, err := mail.ParseAddress(address)
	id := observedExternalName(observed, "integration")
	if err != nil || parsed.Address != address || id == "" {
		return status, false
	}
	meta, _ := xr["metadata"].(map[string]any)
	generation, err := strconv.ParseInt(fmt.Sprint(meta["generation"]), 10, 64)
	if err != nil || generation < 1 {
		return status, false
	}
	result.SetUnstructuredContent(map[string]any{"status": map[string]any{"alertReceiver": map[string]any{"address": address, "integrationID": id, "stackName": accessStackReference(xr), "observedGeneration": generation}}})
	return status, true
}
