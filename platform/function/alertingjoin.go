package main

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"strconv"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
	"github.com/crossplane/function-sdk-go/resource/composite"
)

// The consumer retains only a reference. Secret-bearing data is copied into
// internal renderer input and necessary managed-resource transport only.
func resolveAlertingReceivers(req *fnv1.RunFunctionRequest, rsp *fnv1.RunFunctionResponse, xr map[string]any) (map[string]any, bool, error) {
	raw, err := json.Marshal(xr)
	if err != nil {
		return nil, false, fmt.Errorf("cannot serialize alerting receiver input")
	}
	var resolved map[string]any
	if json.Unmarshal(raw, &resolved) != nil {
		return nil, false, fmt.Errorf("cannot serialize alerting receiver input")
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
		if rsp.Requirements == nil {
			rsp.Requirements = &fnv1.Requirements{}
		}
		if rsp.Requirements.Resources == nil {
			rsp.Requirements.Resources = map[string]*fnv1.ResourceSelector{}
		}
		key := "oncall-receiver-" + name
		rsp.Requirements.Resources[key] = &fnv1.ResourceSelector{ApiVersion: "platform.example.org/v1beta1", Kind: "GrafanaOnCall", Namespace: &namespace, Match: &fnv1.ResourceSelector_MatchName{MatchName: name}}
		items, available, err := request.GetRequiredResource(req, key)
		if err != nil {
			return nil, false, fmt.Errorf("cannot resolve OnCall receiver")
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
		receiverStack := stringValue(receiver, "stackName", "")
		if len(receiver) == 0 || receiverStack == "" {
			ready = false
			continue
		}
		if receiverStack != accessStackReference(xr) {
			return nil, false, fmt.Errorf("OnCall receiver status stack binding mismatch")
		}
		if !requiredStackReady(object) || stringValue(receiver, "integrationID", "") == "" || fmt.Sprint(receiver["observedGeneration"]) != fmt.Sprint(om["generation"]) {
			ready = false
			continue
		}
		selected, e := onCallSelection(object)
		if e != nil {
			return nil, false, fmt.Errorf("OnCall receiver type is invalid")
		}
		if selected.Type == "inbound_email" {
			address := stringValue(receiver, "address", "")
			parsed, err := mail.ParseAddress(address)
			if err != nil || parsed.Address != address {
				return nil, false, fmt.Errorf("observed OnCall inbound email address is invalid")
			}
			point["email"] = map[string]any{"addresses": []any{address}}
			delete(point, "onCallRef")
			continue
		}
		if receiver["integrationType"] != selected.Type {
			ready = false
			continue
		}
		iname, _, _ := onCallNames(name, selected.Type)
		if receiver["integrationName"] != iname {
			return nil, false, fmt.Errorf("OnCall receiver integration identity mismatch")
		}
		ikey := "oncall-integration-" + name
		rsp.Requirements.Resources[ikey] = &fnv1.ResourceSelector{ApiVersion: onCallManagedVersion, Kind: "Integration", Namespace: &namespace, Match: &fnv1.ResourceSelector_MatchName{MatchName: iname}}
		integrations, available, e := request.GetRequiredResource(req, ikey)
		if e != nil {
			return nil, false, fmt.Errorf("cannot resolve OnCall integration")
		}
		if !available || len(integrations) == 0 {
			ready = false
			continue
		}
		if len(integrations) != 1 || integrations[0].Resource == nil {
			return nil, false, fmt.Errorf("OnCall integration requirement is invalid")
		}
		j, e := onCallDecodeJournal(object)
		if e != nil || j == nil || j.Transition != nil || j.ActiveType != selected.Type {
			return nil, false, fmt.Errorf("OnCall integration journal is invalid")
		}
		desired, e := onCallRenderConfiguration(j, j.ActiveConfiguration)
		if e != nil {
			return nil, false, fmt.Errorf("OnCall integration configuration is invalid")
		}
		ik, _ := onCallKeys(selected.Type)
		integration := integrations[0].Resource.UnstructuredContent()
		im, _ := integration["metadata"].(map[string]any)
		annotations, _ := im["annotations"].(map[string]any)
		if !onCallOwned(j, ik, integration, desired[ik]) || annotations["crossplane.io/external-name"] != receiver["integrationID"] || stringValue(im, "uid", "") != j.Inventory[selected.Type].Integration.UID {
			return nil, false, fmt.Errorf("OnCall integration identity or ownership mismatch")
		}
		c := composed.New()
		c.SetUnstructuredContent(integration)
		if !observedDesiredCurrent(resource.ObservedComposed{Resource: c}, desired[ik]) {
			ready = false
			continue
		}
		is, _ := integration["status"].(map[string]any)
		at, _ := is["atProvider"].(map[string]any)
		if at["type"] != selected.Type {
			return nil, false, fmt.Errorf("OnCall integration type mismatch")
		}
		link := stringValue(at, "link", "")
		if !validOnCallURL(link) {
			return nil, false, fmt.Errorf("OnCall integration link is invalid")
		}
		point["oncall"] = map[string]any{"url": link, "oncallIntegrationRef": map[string]any{"name": iname, "policy": map[string]any{"resolution": "Required", "resolve": "Always"}}}
		delete(point, "onCallRef")
	}
	return resolved, ready, nil
}

func onCallReceiverStatus(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, desired map[resource.Name]*resource.DesiredComposed) (*resource.Composite, bool) {
	result := composite.New()
	result.SetUnstructuredContent(map[string]any{"status": map[string]any{"alertReceiver": nil}})
	status := &resource.Composite{Resource: result, Ready: resource.ReadyUnspecified}
	meta, _ := xr["metadata"].(map[string]any)
	generation, err := strconv.ParseInt(fmt.Sprint(meta["generation"]), 10, 64)
	if err != nil || generation < 1 {
		return status, false
	}
	if onCallJournalGoverned(xr) {
		j, err := onCallDecodeJournal(xr)
		if err != nil || j == nil || j.Transition != nil || !onCallCurrentSelection(xr, j.ActiveConfiguration) || !onCallBarrier(j, j.ActiveConfiguration, observed) {
			return status, false
		}
		ik, _ := onCallKeys(j.ActiveType)
		receiver := map[string]any{"integrationID": observedExternalName(observed, ik), "stackName": j.Stack, "observedGeneration": generation, "integrationType": j.ActiveType}
		if j.ActiveType == "grafana_alerting" {
			iname, _, _ := onCallNames(j.Name, j.ActiveType)
			receiver["integrationName"] = iname
		} else {
			cs, _ := observed[ik].Resource.UnstructuredContent()["status"].(map[string]any)
			at, _ := cs["atProvider"].(map[string]any)
			address := stringValue(at, "inboundEmail", "")
			parsed, err := mail.ParseAddress(address)
			if err != nil || parsed.Address != address {
				return status, false
			}
			receiver["address"] = address
		}
		result.SetUnstructuredContent(map[string]any{"status": map[string]any{"alertReceiver": receiver}})
		return status, true
	}
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
	result.SetUnstructuredContent(map[string]any{"status": map[string]any{"alertReceiver": map[string]any{"address": address, "integrationID": id, "stackName": accessStackReference(xr), "observedGeneration": generation}}})
	return status, true
}
