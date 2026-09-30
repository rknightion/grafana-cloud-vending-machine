package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
)

const onCallContextKey = "_onCallTrustedContext"
const onCallPlanKey = "_onCallPlan"
const onCallManagedVersion = "oncall.grafana.m.crossplane.io/v1alpha1"
const onCallObserveVersion = "oncall.grafana.o.crossplane.io/v1alpha1"

// No link, provider status dump, or URL-derived fingerprint is stored here.
type onCallSelectionConfig struct {
	Type        string `json:"type"`
	ChannelMode string `json:"channelMode"`
	ChannelName string `json:"channelName,omitempty"`
	ChannelID   string `json:"channelID,omitempty"`
}
type onCallBinding struct {
	ID  string `json:"id,omitempty"`
	UID string `json:"uid,omitempty"`
}
type onCallConfiguration struct {
	Selection  onCallSelectionConfig    `json:"selection"`
	Responders []string                 `json:"responders"`
	ShiftStart string                   `json:"shiftStart"`
	Bindings   map[string]onCallBinding `json:"bindings"`
}
type onCallInventory struct {
	State              string        `json:"state"`
	Integration        onCallBinding `json:"integration"`
	Route              onCallBinding `json:"route"`
	RetiredTransaction string        `json:"retiredTransaction,omitempty"`
}
type onCallTransition struct {
	FromType                 string              `json:"fromType"`
	ToType                   string              `json:"toType"`
	Phase                    string              `json:"phase"`
	Disposition              string              `json:"disposition"`
	Generation               string              `json:"generation"`
	Fingerprint              string              `json:"fingerprint"`
	Source                   onCallConfiguration `json:"source"`
	Target                   onCallConfiguration `json:"target"`
	SourceIncomplete         bool                `json:"sourceIncomplete,omitempty"`
	SourceIntegrationPlanned bool                `json:"sourceIntegrationPlanned,omitempty"`
	SourceRoutePlanned       bool                `json:"sourceRoutePlanned,omitempty"`
	PlannedShared            []string            `json:"plannedShared,omitempty"`
	PlannedIntegration       bool                `json:"plannedIntegration"`
	PlannedRoute             bool                `json:"plannedRoute"`
	KeepType                 string              `json:"keepType,omitempty"`
	RetireType               string              `json:"retireType,omitempty"`
}
type onCallStableEdit struct {
	Configuration onCallConfiguration `json:"configuration"`
	Fingerprint   string              `json:"fingerprint"`
	PlannedUsers  []string            `json:"plannedUsers"`
}
type onCallJournal struct {
	Version             string                     `json:"version"`
	OwnerUID            string                     `json:"ownerUID"`
	Name                string                     `json:"name"`
	Namespace           string                     `json:"namespace"`
	Stack               string                     `json:"stack"`
	ActiveType          string                     `json:"activeType"`
	ActiveConfiguration onCallConfiguration        `json:"activeConfiguration"`
	Inventory           map[string]onCallInventory `json:"inventory"`
	Transition          *onCallTransition          `json:"transition,omitempty"`
	StableEdit          *onCallStableEdit          `json:"stableEdit,omitempty"`
}
type onCallTrustedContext struct {
	Admitted bool
	Request  *fnv1.RunFunctionRequest
	Response *fnv1.RunFunctionResponse
}
type onCallPlan struct {
	Journal            *onCallJournal
	Prune              []resource.Name
	CurrentRequest     bool
	PendingDestination bool
}

func onCallValidSlackID(value string) bool {
	matched, err := regexp.MatchString(`^(?:C[A-Z0-9]{2,}|[GD][A-Z0-9]{8,})$`, value)
	return err == nil && matched
}

func onCallSelection(xr map[string]any) (onCallSelectionConfig, error) {
	spec, _ := xr["spec"].(map[string]any)
	s := onCallSelectionConfig{Type: "inbound_email", ChannelMode: "none"}
	if raw, exists := spec["integrationType"]; exists {
		value, ok := raw.(string)
		if !ok || (value != "inbound_email" && value != "grafana_alerting") {
			return s, fmt.Errorf("unsupported OnCall integration type")
		}
		s.Type = value
	}
	route, _ := spec["route"].(map[string]any)
	id, hasID := route["channelId"]
	ref, hasRef := route["channelRef"]
	if hasID && hasRef {
		return s, fmt.Errorf("OnCall route must select at most one Slack destination")
	}
	if hasID {
		value, ok := id.(string)
		if !ok || !onCallValidSlackID(value) {
			return s, fmt.Errorf("OnCall Slack destination must be an opaque channel ID")
		}
		s.ChannelMode = "direct"
		s.ChannelID = value
	}
	if hasRef {
		r, ok := ref.(map[string]any)
		value := stringValue(r, "name", "")
		if !ok || strings.TrimSpace(value) == "" || strings.TrimSpace(value) != value {
			return s, fmt.Errorf("OnCall Slack reference requires a nonempty name")
		}
		s.ChannelMode = "reference"
		s.ChannelName = value
	}
	return s, nil
}
func onCallJournalGoverned(xr map[string]any) bool {
	status, _ := xr["status"].(map[string]any)
	if _, ok := status["onCallIdentity"]; ok {
		return true
	}
	spec, _ := xr["spec"].(map[string]any)
	if _, ok := spec["integrationType"]; ok {
		return true
	}
	route, _ := spec["route"].(map[string]any)
	_, id := route["channelId"]
	_, ref := route["channelRef"]
	return id || ref
}
func onCallKeys(t string) (resource.Name, resource.Name) {
	if t == "grafana_alerting" {
		return "integration-grafana-alerting", "catch-all-route-grafana-alerting"
	}
	return "integration", "catch-all-route"
}
func onCallSlackKey(t string) resource.Name {
	if t == "grafana_alerting" {
		return "slack-channel-grafana-alerting"
	}
	return "slack-channel"
}
func onCallNames(name, t string) (string, string, string) {
	if t == "grafana_alerting" {
		return name + "-grafana-alerting", name + "-catch-all-route-grafana-alerting", name + " grafana alerting"
	}
	return name + "-inbound-email", name + "-catch-all-route", name + " inbound email"
}
func onCallDecodeJournal(xr map[string]any) (*onCallJournal, error) {
	status, _ := xr["status"].(map[string]any)
	raw, present := status["onCallIdentity"]
	if !present {
		return nil, nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("OnCall journal is invalid")
	}
	j := &onCallJournal{}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if decoder.Decode(j) != nil {
		return nil, fmt.Errorf("OnCall journal is invalid")
	}
	meta, _ := xr["metadata"].(map[string]any)
	if j.Version != "v1" || j.OwnerUID == "" || j.OwnerUID != stringValue(meta, "uid", "") || j.Name != stringValue(meta, "name", "") || j.Namespace != stringValue(meta, "namespace", "") || j.Stack != accessStackReference(xr) || j.Inventory == nil {
		return nil, fmt.Errorf("OnCall journal ownership or binding mismatch")
	}
	if j.ActiveType != "" && j.ActiveType != "inbound_email" && j.ActiveType != "grafana_alerting" {
		return nil, fmt.Errorf("OnCall journal active type is invalid")
	}
	for typ, entry := range j.Inventory {
		if (typ != "inbound_email" && typ != "grafana_alerting") || (entry.State != "Active" && entry.State != "Retired") {
			return nil, fmt.Errorf("OnCall journal inventory is invalid")
		}
	}
	if edit := j.StableEdit; edit != nil {
		if j.Transition != nil || edit.Configuration.Selection.Type != j.ActiveType || edit.Fingerprint != onCallFingerprint(j.ActiveConfiguration, edit.Configuration) {
			return nil, fmt.Errorf("OnCall stable edit is invalid")
		}
	}
	if tr := j.Transition; tr != nil {
		switch tr.Phase {
		case "Building", "Cancelling", "Prepared", "RetiringRoute", "RetiringIntegration":
		default:
			return nil, fmt.Errorf("OnCall journal phase is invalid")
		}
		if (tr.ToType != "inbound_email" && tr.ToType != "grafana_alerting") || (tr.FromType != "" && tr.FromType != "inbound_email" && tr.FromType != "grafana_alerting") || tr.ToType == tr.FromType || (tr.Disposition != "promotion" && tr.Disposition != "cancellation") || tr.Fingerprint != onCallFingerprint(tr.Source, tr.Target) {
			return nil, fmt.Errorf("OnCall journal transaction is invalid")
		}
		if tr.Phase == "Prepared" || tr.Phase == "RetiringRoute" || tr.Phase == "RetiringIntegration" {
			keep, lose := tr.ToType, tr.FromType
			if tr.Disposition == "cancellation" {
				keep, lose = tr.FromType, tr.ToType
			}
			if tr.KeepType != keep || tr.RetireType != lose || keep == "" {
				return nil, fmt.Errorf("OnCall retirement authority is invalid")
			}
		}
	}
	return j, nil
}
func onCallCopyConfiguration(c onCallConfiguration) onCallConfiguration {
	result := c
	result.Responders = append([]string(nil), c.Responders...)
	result.Bindings = map[string]onCallBinding{}
	for key, b := range c.Bindings {
		result.Bindings[key] = b
	}
	return result
}

func onCallFingerprint(source, target onCallConfiguration) string {
	data, _ := json.Marshal([]onCallConfiguration{source, target})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
func onCallConfigurationFor(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, s onCallSelectionConfig) (onCallConfiguration, error) {
	spec, _ := xr["spec"].(map[string]any)
	c := onCallConfiguration{Selection: s, ShiftStart: stringValue(spec, "shiftStart", ""), Bindings: map[string]onCallBinding{}}
	responders, _ := spec["responders"].([]any)
	for _, raw := range responders {
		r, _ := raw.(map[string]any)
		c.Responders = append(c.Responders, stringValue(r, "username", ""))
	}
	for key, observedChild := range observed {
		var obj resource.ObservedComposed = observedChild
		if obj.Resource == nil {
			return c, fmt.Errorf("OnCall observed child is invalid")
		}
		if key == onCallSlackKey(s.Type) || strings.HasPrefix(string(key), "responder-") || key == "rotating-shift" || key == "schedule" || key == "escalation-chain" || key == "schedule-escalation" {
			c.Bindings[string(key)] = onCallBinding{ID: observedExternalName(observed, key), UID: string(obj.Resource.GetUID())}
		}
	}
	return c, nil
}
func onCallSyntheticObservations(c onCallConfiguration, inventory map[string]onCallInventory, typ string) map[resource.Name]resource.ObservedComposed {
	result := map[resource.Name]resource.ObservedComposed{}
	add := func(key resource.Name, b onCallBinding) {
		if b.ID == "" {
			return
		}
		r := composed.New()
		r.SetUnstructuredContent(map[string]any{"metadata": map[string]any{"annotations": map[string]any{"crossplane.io/external-name": b.ID}}})
		result[key] = resource.ObservedComposed{Resource: r}
	}
	for key, b := range c.Bindings {
		add(resource.Name(key), b)
	}
	// Legacy reconstruction uses these keys internally. No child identity moves.
	entry := inventory[typ]
	add("integration", entry.Integration)
	add("catch-all-route", entry.Route)
	return result
}
func onCallRenderConfiguration(j *onCallJournal, c onCallConfiguration) (map[resource.Name]*resource.DesiredComposed, error) {
	responders := make([]any, 0, len(c.Responders))
	for _, username := range c.Responders {
		responders = append(responders, map[string]any{"username": username})
	}
	xr := map[string]any{"metadata": map[string]any{"name": j.Name, "namespace": j.Namespace}, "spec": map[string]any{"stackRef": map[string]any{"name": j.Stack}, "shiftStart": c.ShiftStart, "responders": responders}}
	desired, err := renderOnCallLegacy(xr, onCallSyntheticObservations(c, j.Inventory, c.Selection.Type), nil)
	if err != nil {
		return nil, fmt.Errorf("OnCall frozen configuration cannot be reconstructed")
	}
	ik, rk := onCallKeys(c.Selection.Type)
	iname, rname, display := onCallNames(j.Name, c.Selection.Type)
	if child := desired["integration"]; child != nil {
		delete(desired, "integration")
		obj := child.Resource.UnstructuredContent()
		obj["metadata"].(map[string]any)["name"] = iname
		p := obj["spec"].(map[string]any)["forProvider"].(map[string]any)
		p["name"] = display
		p["type"] = c.Selection.Type
		desired[ik] = child
	}
	if child := desired["catch-all-route"]; child != nil {
		delete(desired, "catch-all-route")
		child.Resource.UnstructuredContent()["metadata"].(map[string]any)["name"] = rname
		p := child.Resource.UnstructuredContent()["spec"].(map[string]any)["forProvider"].(map[string]any)
		p["slack"] = []any{}
		if c.Selection.ChannelMode == "reference" && c.Selection.ChannelID == "" {
			delete(desired, rk)
			child = nil
		}
		if c.Selection.ChannelID != "" {
			slack := map[string]any{"channelId": c.Selection.ChannelID, "enabled": true}
			if c.Selection.ChannelMode == "reference" {
				slack["slackChannelRef"] = map[string]any{"name": j.Name + "-" + string(onCallSlackKey(c.Selection.Type)), "policy": map[string]any{"resolution": "Required", "resolve": "Always"}}
			}
			p["slack"] = []any{slack}
		}
		if child != nil {
			desired[rk] = child
		}
	}
	if c.Selection.ChannelMode == "reference" {
		desired[onCallSlackKey(c.Selection.Type)] = newDesired(onCallObserveVersion, "SlackChannel", j.Namespace, j.Name+"-"+string(onCallSlackKey(c.Selection.Type)), nil, map[string]any{"managementPolicies": []any{"Observe"}, "forProvider": map[string]any{"name": c.Selection.ChannelName}, "providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": j.Stack}})
	}
	return desired, nil
}
func onCallOwned(j *onCallJournal, key resource.Name, obj map[string]any, want *resource.DesiredComposed) bool {
	if want == nil || want.Resource == nil {
		return false
	}
	m, _ := obj["metadata"].(map[string]any)
	wm, _ := want.Resource.UnstructuredContent()["metadata"].(map[string]any)
	if obj["apiVersion"] != want.Resource.GetAPIVersion() || obj["kind"] != want.Resource.GetKind() || m["name"] != wm["name"] || m["namespace"] != j.Namespace || stringValue(m, "uid", "") == "" {
		return false
	}
	spec, _ := obj["spec"].(map[string]any)
	provider, _ := spec["providerConfigRef"].(map[string]any)
	if provider["name"] != j.Stack || provider["kind"] != "ProviderConfig" {
		return false
	}
	wa, _ := wm["annotations"].(map[string]any)
	annotations, _ := m["annotations"].(map[string]any)
	if id := stringValue(wa, "crossplane.io/external-name", ""); id != "" && annotations["crossplane.io/external-name"] != id {
		return false
	}
	refs, _ := m["ownerReferences"].([]any)
	controllers := 0
	for _, raw := range refs {
		r, _ := raw.(map[string]any)
		if r["controller"] == true {
			controllers++
			if r["uid"] != j.OwnerUID || r["name"] != j.Name || r["kind"] != "GrafanaOnCall" || r["apiVersion"] != "platform.example.org/v1beta1" {
				return false
			}
		}
	}
	return controllers == 1
}
func onCallCapturePair(j *onCallJournal, typ string, observed map[resource.Name]resource.ObservedComposed, desired map[resource.Name]*resource.DesiredComposed) error {
	ik, rk := onCallKeys(typ)
	entry := j.Inventory[typ]
	if entry.State == "" {
		entry.State = "Active"
	}
	for _, key := range []resource.Name{ik, rk} {
		obj, exists := observed[key]
		if !exists {
			continue
		}
		if obj.Resource == nil || !onCallOwned(j, key, obj.Resource.UnstructuredContent(), desired[key]) {
			return fmt.Errorf("OnCall pair observation identity mismatch")
		}
		b := onCallBinding{ID: observedExternalName(observed, key), UID: string(obj.Resource.GetUID())}
		old := entry.Integration
		if key == rk {
			old = entry.Route
		}
		if (old.ID != "" && old.ID != b.ID) || (old.UID != "" && old.UID != b.UID) {
			return fmt.Errorf("OnCall pair incarnation or import identity mismatch")
		}
		if key == ik {
			entry.Integration = b
		} else {
			entry.Route = b
		}
	}
	if entry.Integration.ID != "" && entry.Route.ID != "" && entry.Integration.ID == entry.Route.ID {
		return fmt.Errorf("OnCall pair external identities conflict")
	}
	for other, e := range j.Inventory {
		if other != typ && ((entry.Integration.ID != "" && entry.Integration.ID == e.Integration.ID) || (entry.Route.ID != "" && entry.Route.ID == e.Route.ID)) {
			return fmt.Errorf("OnCall type-specific external identities conflict")
		}
	}
	j.Inventory[typ] = entry
	return nil
}

// Readback equality is deliberately negative as well as positive. Omitted
// desired Slack fields do not mean that the provider removed a destination.
func onCallSlackMatches(observed resource.ObservedComposed, want string) bool {
	if observed.Resource == nil {
		return false
	}
	status, _ := observed.Resource.UnstructuredContent()["status"].(map[string]any)
	at, _ := status["atProvider"].(map[string]any)
	raw, exists := at["slack"]
	if !exists || raw == nil {
		return want == ""
	}
	items, ok := raw.([]any)
	if !ok {
		return false
	}
	enabled := 0
	for _, raw := range items {
		s, ok := raw.(map[string]any)
		if !ok {
			return false
		}
		flag, ok := s["enabled"].(bool)
		if !ok {
			return false
		}
		if flag {
			enabled++
			if s["channelId"] != want {
				return false
			}
		}
	}
	if want == "" {
		return enabled == 0
	}
	return enabled == 1
}
func onCallBarrier(j *onCallJournal, c onCallConfiguration, observed map[resource.Name]resource.ObservedComposed) bool {
	desired, err := onCallRenderConfiguration(j, c)
	if err != nil {
		return false
	}
	ik, rk := onCallKeys(c.Selection.Type)
	if desired[rk] == nil {
		return false
	}
	for key, want := range desired {
		o, ok := observed[key]
		if !ok || o.Resource == nil || !onCallOwned(j, key, o.Resource.UnstructuredContent(), want) || !observedDesiredCurrent(o, want) {
			return false
		}
		if b, exists := c.Bindings[string(key)]; exists && (b.ID != observedExternalName(observed, key) || b.UID != string(o.Resource.GetUID())) {
			return false
		}
	}
	entry := j.Inventory[c.Selection.Type]
	if entry.Integration.ID == "" || entry.Route.ID == "" || entry.Integration.UID != string(observed[ik].Resource.GetUID()) || entry.Route.UID != string(observed[rk].Resource.GetUID()) {
		return false
	}
	status, _ := observed[ik].Resource.UnstructuredContent()["status"].(map[string]any)
	at, _ := status["atProvider"].(map[string]any)
	if at["type"] != c.Selection.Type {
		return false
	}
	routeStatus, _ := observed[rk].Resource.UnstructuredContent()["status"].(map[string]any)
	routeAt, _ := routeStatus["atProvider"].(map[string]any)
	if routeAt["integrationId"] != entry.Integration.ID || routeAt["escalationChainId"] != c.Bindings["escalation-chain"].ID || !onCallSlackMatches(observed[rk], c.Selection.ChannelID) {
		return false
	}
	if c.Selection.Type == "grafana_alerting" && !validOnCallURL(stringValue(at, "link", "")) {
		return false
	}
	return true
}
func onCallStableUsers(j *onCallJournal) (map[resource.Name]*resource.DesiredComposed, error) {
	edit := j.StableEdit
	responders := make([]any, 0, len(edit.Configuration.Responders))
	for _, username := range edit.Configuration.Responders {
		responders = append(responders, map[string]any{"username": username})
	}
	xr := map[string]any{"metadata": map[string]any{"name": j.Name, "namespace": j.Namespace}, "spec": map[string]any{"stackRef": map[string]any{"name": j.Stack}, "shiftStart": edit.Configuration.ShiftStart, "responders": responders}}
	users, err := renderOnCallLegacy(xr, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("OnCall stable configuration is invalid")
	}
	if len(users) != len(edit.PlannedUsers) {
		return nil, fmt.Errorf("OnCall stable lookup plan is invalid")
	}
	seen := map[string]bool{}
	for _, key := range edit.PlannedUsers {
		if users[resource.Name(key)] == nil || seen[key] {
			return nil, fmt.Errorf("OnCall stable lookup plan is invalid")
		}
		seen[key] = true
	}
	return users, nil
}

func onCallApplyStableEdit(j *onCallJournal, observed map[resource.Name]resource.ObservedComposed, config map[string]any, plan onCallPlan) (map[resource.Name]*resource.DesiredComposed, error) {
	desired, err := onCallRenderConfiguration(j, j.ActiveConfiguration)
	if err != nil {
		return nil, err
	}
	users, err := onCallStableUsers(j)
	if err != nil {
		return nil, err
	}
	complete := true
	for key, want := range users {
		desired[key] = want
		o, exists := observed[key]
		if !exists || o.Resource == nil || !onCallOwned(j, key, o.Resource.UnstructuredContent(), want) || !observedDesiredCurrent(o, want) || observedExternalName(observed, key) == "" {
			complete = false
			continue
		}
		binding := onCallBinding{ID: observedExternalName(observed, key), UID: string(o.Resource.GetUID())}
		if old := j.StableEdit.Configuration.Bindings[string(key)]; (old.ID != "" && old.ID != binding.ID) || (old.UID != "" && old.UID != binding.UID) {
			return nil, fmt.Errorf("OnCall stable responder binding changed")
		}
		j.StableEdit.Configuration.Bindings[string(key)] = binding
	}
	j.StableEdit.Fingerprint = onCallFingerprint(j.ActiveConfiguration, j.StableEdit.Configuration)
	if complete {
		j.ActiveConfiguration = j.StableEdit.Configuration
		j.StableEdit = nil
		updated, e := onCallRenderConfiguration(j, j.ActiveConfiguration)
		if e != nil {
			return nil, e
		}
		_, rk := onCallKeys(j.ActiveType)
		if updated[rk] == nil {
			updated[rk] = desired[rk]
		}
		desired = updated
	}
	config[onCallPlanKey] = plan
	return desired, nil
}

func onCallPreserve(j *onCallJournal, observed map[resource.Name]resource.ObservedComposed, req *fnv1.RunFunctionRequest) (map[resource.Name]*resource.DesiredComposed, error) {
	allowed, err := onCallRenderConfiguration(j, j.ActiveConfiguration)
	if err != nil {
		return nil, err
	}
	if tr := j.Transition; tr != nil {
		for _, c := range []onCallConfiguration{tr.Source, tr.Target} {
			if c.Selection.Type == "" {
				continue
			}
			d, e := onCallRenderConfiguration(j, c)
			if e != nil {
				return nil, e
			}
			for k, v := range d {
				allowed[k] = v
			}
		}
	}
	if j.StableEdit != nil {
		users, e := onCallStableUsers(j)
		if e != nil {
			return nil, e
		}
		for key, child := range users {
			allowed[key] = child
		}
	}
	desired := map[resource.Name]*resource.DesiredComposed{}
	for key, observedChild := range observed {
		var o resource.ObservedComposed = observedChild
		if o.Resource == nil || !onCallOwned(j, key, o.Resource.UnstructuredContent(), allowed[key]) {
			return nil, fmt.Errorf("OnCall trusted preservation refused")
		}
		obj := o.Resource.UnstructuredContent()
		data, err := json.Marshal(obj["spec"])
		if err != nil {
			return nil, fmt.Errorf("OnCall trusted preservation refused")
		}
		var spec map[string]any
		if json.Unmarshal(data, &spec) != nil {
			return nil, fmt.Errorf("OnCall trusted preservation refused")
		}
		// The allowed constructor has a statically registered GVK. Replace only
		// its spec with the trusted applied spec; never copy provider status.
		child := allowed[key]
		child.Resource.UnstructuredContent()["spec"] = spec
		if id := observedExternalName(observed, key); id != "" {
			child.Resource.SetAnnotations(map[string]string{"crossplane.io/external-name": id})
		}
		desired[key] = child
	}
	for key := range req.GetDesired().GetResources() {
		if desired[resource.Name(key)] == nil {
			return nil, fmt.Errorf("OnCall incomplete preservation refused")
		}
	}
	// Every issued pair incarnation must be observed before an empty map can be
	// interpreted as complete preservation. Retired ownership is not resurrected.
	for typ, entry := range j.Inventory {
		if entry.State == "Retired" {
			continue
		}
		ik, rk := onCallKeys(typ)
		if (entry.Integration.UID != "" && desired[ik] == nil) || (entry.Route.UID != "" && desired[rk] == nil) {
			return nil, fmt.Errorf("OnCall incomplete preservation refused")
		}
	}
	return desired, nil
}
func onCallNamedAbsent(ctx onCallTrustedContext, j *onCallJournal, typ string, route bool, observed map[resource.Name]resource.ObservedComposed) (bool, error) {
	ik, rk := onCallKeys(typ)
	key := ik
	kind := "Integration"
	iname, rname, _ := onCallNames(j.Name, typ)
	name := iname
	if route {
		key = rk
		kind = "Route"
		name = rname
	}
	requirement := "oncall-absence-" + string(key)
	if ctx.Response.Requirements == nil {
		ctx.Response.Requirements = &fnv1.Requirements{}
	}
	if ctx.Response.Requirements.Resources == nil {
		ctx.Response.Requirements.Resources = map[string]*fnv1.ResourceSelector{}
	}
	ctx.Response.Requirements.Resources[requirement] = &fnv1.ResourceSelector{ApiVersion: onCallManagedVersion, Kind: kind, Namespace: &j.Namespace, Match: &fnv1.ResourceSelector_MatchName{MatchName: name}}
	items, resolved, err := request.GetRequiredResource(ctx.Request, requirement)
	if err != nil {
		return false, fmt.Errorf("OnCall absence witness unavailable")
	}
	if _, present := observed[key]; present {
		return false, nil
	}
	if !resolved {
		return false, nil
	}
	if len(items) == 0 {
		return true, nil
	}
	if len(items) != 1 || items[0].Resource == nil {
		return false, fmt.Errorf("OnCall absence witness identity mismatch")
	}
	obj := items[0].Resource.UnstructuredContent()
	m, _ := obj["metadata"].(map[string]any)
	entry := j.Inventory[typ]
	expected := entry.Integration
	if route {
		expected = entry.Route
	}
	if obj["apiVersion"] != onCallManagedVersion || obj["kind"] != kind || m["name"] != name || m["namespace"] != j.Namespace || (expected.UID != "" && m["uid"] != expected.UID) {
		return false, fmt.Errorf("OnCall absence witness identity mismatch")
	}
	return false, nil
}
func onCallCurrentSelection(xr map[string]any, c onCallConfiguration) bool {
	s, err := onCallSelection(xr)
	if err != nil {
		return false
	}
	if s.ChannelMode == "reference" {
		s.ChannelID = c.Selection.ChannelID
	}
	spec, _ := xr["spec"].(map[string]any)
	data, _ := json.Marshal(spec["responders"])
	responders := make([]any, 0, len(c.Responders))
	for _, u := range c.Responders {
		responders = append(responders, map[string]any{"username": u})
	}
	expected, _ := json.Marshal(responders)
	return s == c.Selection && stringValue(spec, "shiftStart", "") == c.ShiftStart && string(data) == string(expected)
}

func renderOnCallIdentity(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	ctx, ok := config[onCallContextKey].(onCallTrustedContext)
	if !ok {
		return nil, fmt.Errorf("OnCall migration requires trusted admission context")
	}
	j, err := onCallDecodeJournal(xr)
	if err != nil {
		return nil, err
	}
	selection, selectionErr := onCallSelection(xr)
	if j == nil {
		if selectionErr != nil {
			return nil, selectionErr
		}
		meta, _ := xr["metadata"].(map[string]any)
		j = &onCallJournal{Version: "v1", OwnerUID: stringValue(meta, "uid", ""), Name: stringValue(meta, "name", ""), Namespace: stringValue(meta, "namespace", ""), Stack: accessStackReference(xr), Inventory: map[string]onCallInventory{}}
		if j.OwnerUID == "" {
			return nil, fmt.Errorf("OnCall migration requires composite ownership identity")
		}
		// Bootstrap from a fully applied predecessor, never infer an installation
		// from an incomplete provider-assigned pair.
		sourceSelection := onCallSelectionConfig{Type: "inbound_email", ChannelMode: "none"}
		c, e := onCallConfigurationFor(xr, observed, sourceSelection)
		if e != nil {
			return nil, e
		}
		j.ActiveConfiguration = c
		if !ctx.Admitted {
			return onCallPreserve(j, observed, ctx.Request)
		}
		if len(observed) > 0 {
			legacy, e := renderOnCallLegacy(xr, observed, nil)
			if e != nil {
				return nil, e
			}
			if legacy["catch-all-route"] == nil {
				return nil, fmt.Errorf("OnCall bootstrap inventory is incomplete")
			}
			if e = onCallCapturePair(j, "inbound_email", observed, legacy); e != nil {
				return nil, e
			}
			for key, want := range legacy {
				o, exists := observed[key]
				if !exists || o.Resource == nil || !onCallOwned(j, key, o.Resource.UnstructuredContent(), want) || !observedDesiredCurrent(o, want) {
					return nil, fmt.Errorf("OnCall bootstrap predecessor is not current")
				}
			}
			j.ActiveType = "inbound_email"
		} else {
			// Fresh staging needs its own journal before issuing even shared lookups.
			j.ActiveType = ""
		}
		target := onCallCopyConfiguration(c)
		target.Selection = selection
		j.Transition = &onCallTransition{FromType: j.ActiveType, ToType: selection.Type, Phase: "Building", Disposition: "promotion", Generation: fmt.Sprint(meta["generation"]), Source: c, Target: target, PlannedIntegration: true}
		if j.ActiveType == selection.Type {
			// No identity replacement is needed for a same-type Slack opt-in.
			// Persist configuration first, retaining the applied predecessor this turn.
			j.ActiveConfiguration = target
			j.Transition = nil
			config[onCallPlanKey] = onCallPlan{Journal: j}
			return onCallRenderConfiguration(j, c)
		}
		if j.ActiveType == "" {
			d, e := onCallRenderConfiguration(j, target)
			if e != nil {
				return nil, e
			}
			for key := range d {
				if key != "integration" && key != "integration-grafana-alerting" && key != "catch-all-route" && key != "catch-all-route-grafana-alerting" {
					j.Transition.PlannedShared = append(j.Transition.PlannedShared, string(key))
				}
			}
		}
		j.Transition.Fingerprint = onCallFingerprint(c, target)
		config[onCallPlanKey] = onCallPlan{Journal: j}
		if j.ActiveType == "" {
			return map[resource.Name]*resource.DesiredComposed{}, nil
		}
		return onCallRenderConfiguration(j, c)
	}
	if !ctx.Admitted {
		return onCallPreserve(j, observed, ctx.Request)
	}
	plan := onCallPlan{Journal: j}
	if j.Transition == nil {
		if selectionErr != nil {
			return nil, selectionErr
		}
		if j.StableEdit != nil {
			return onCallApplyStableEdit(j, observed, config, plan)
		}
		if selection.Type == j.ActiveType {
			if !onCallCurrentSelection(xr, j.ActiveConfiguration) {
				old := j.ActiveConfiguration
				updated, e := onCallConfigurationFor(xr, observed, selection)
				if e != nil {
					return nil, e
				}
				// Persist the lookup plan separately from active authority. Issuing
				// missing User children must not shrink the existing shared graph.
				updated.Bindings = onCallCopyConfiguration(old).Bindings
				if updated.Selection != old.Selection {
					delete(updated.Bindings, string(onCallSlackKey(j.ActiveType)))
				}
				edit := &onCallStableEdit{Configuration: updated}
				for _, username := range updated.Responders {
					edit.PlannedUsers = append(edit.PlannedUsers, "responder-"+stableResourceSuffix(username))
				}
				edit.Fingerprint = onCallFingerprint(old, updated)
				j.StableEdit = edit
				config[onCallPlanKey] = plan
				return onCallRenderConfiguration(j, old)
			}
			if j.ActiveConfiguration.Selection.ChannelMode == "reference" && j.ActiveConfiguration.Selection.ChannelID == "" {
				lookup, e := onCallRenderConfiguration(j, j.ActiveConfiguration)
				if e != nil {
					return nil, e
				}
				key := onCallSlackKey(j.ActiveType)
				if o, exists := observed[key]; exists && o.Resource != nil && onCallOwned(j, key, o.Resource.UnstructuredContent(), lookup[key]) && observedDesiredCurrent(o, lookup[key]) {
					s, _ := o.Resource.UnstructuredContent()["status"].(map[string]any)
					at, _ := s["atProvider"].(map[string]any)
					id := stringValue(at, "slackID", "")
					if !onCallValidSlackID(id) {
						return nil, fmt.Errorf("OnCall Slack reference did not resolve a valid destination")
					}
					j.ActiveConfiguration.Selection.ChannelID = id
					j.ActiveConfiguration.Bindings[string(key)] = onCallBinding{ID: observedExternalName(observed, key), UID: string(o.Resource.GetUID())}
				}
			}
			desired, e := onCallRenderConfiguration(j, j.ActiveConfiguration)
			if e != nil {
				return nil, e
			}
			for typ, entry := range j.Inventory {
				if entry.State == "Retired" {
					ik, rk := onCallKeys(typ)
					plan.Prune = append(plan.Prune, ik, rk, onCallSlackKey(typ))
				}
			}
			if j.ActiveConfiguration.Selection.ChannelMode == "reference" && j.ActiveConfiguration.Selection.ChannelID == "" {
				_, rk := onCallKeys(j.ActiveType)
				o, present := observed[rk]
				if !present || o.Resource == nil {
					return nil, fmt.Errorf("OnCall Slack lookup pending; preserving existing graph")
				}
				base := onCallCopyConfiguration(j.ActiveConfiguration)
				base.Selection.ChannelMode = "none"
				base.Selection.ChannelName = ""
				base.Selection.ChannelID = ""
				old, e := onCallRenderConfiguration(j, base)
				if e != nil || !onCallOwned(j, rk, o.Resource.UnstructuredContent(), old[rk]) {
					return nil, fmt.Errorf("OnCall Slack pending preservation refused")
				}
				obj := o.Resource.UnstructuredContent()
				spec, _ := obj["spec"].(map[string]any)
				desired[rk] = newDesired(onCallManagedVersion, "Route", j.Namespace, o.Resource.GetName(), onCallExternalAnnotations(observed, rk), spec)
			}
			_, rk := onCallKeys(j.ActiveType)
			plan.PendingDestination = !onCallSlackMatches(observed[rk], j.ActiveConfiguration.Selection.ChannelID)
			plan.CurrentRequest = true
			config[onCallPlanKey] = plan
			return desired, nil
		}
		c := j.ActiveConfiguration
		target := onCallCopyConfiguration(c)
		target.Selection = selection
		if entry, exists := j.Inventory[selection.Type]; exists {
			entry.State = "Active"
			entry.Integration.UID = ""
			entry.Route.UID = ""
			j.Inventory[selection.Type] = entry
		}
		meta, _ := xr["metadata"].(map[string]any)
		j.Transition = &onCallTransition{FromType: j.ActiveType, ToType: selection.Type, Phase: "Building", Disposition: "promotion", Generation: fmt.Sprint(meta["generation"]), Source: c, Target: target, PlannedIntegration: true}
		j.Transition.Fingerprint = onCallFingerprint(c, target)
		config[onCallPlanKey] = plan
		return onCallRenderConfiguration(j, c)
	}
	tr := j.Transition
	observedPhase := tr.Phase
	committed := observedPhase == "RetiringRoute" || observedPhase == "RetiringIntegration"
	if !committed && selectionErr != nil {
		return nil, selectionErr
	}
	if !committed && tr.Disposition == "promotion" && selection.Type != tr.ToType {
		if tr.FromType == "" {
			// Establish inbound-email before retiring any potentially issued fresh
			// Grafana Alerting child. Persist this new plan without issuing it yet.
			if selection.Type == tr.ToType {
				return nil, fmt.Errorf("OnCall fresh configuration edit is pending")
			}
			old := onCallCopyConfiguration(tr.Target)
			replacement := onCallCopyConfiguration(old)
			replacement.Selection = selection
			oldDesired, e := onCallRenderConfiguration(j, old)
			if e != nil {
				return nil, e
			}
			oldIK, oldRK := onCallKeys(tr.ToType)
			if !tr.PlannedIntegration {
				delete(oldDesired, oldIK)
			}
			if !tr.PlannedRoute {
				delete(oldDesired, oldRK)
			}
			j.Transition = &onCallTransition{FromType: tr.ToType, ToType: selection.Type, Phase: "Building", Disposition: "promotion", Generation: tr.Generation, Source: old, Target: replacement, SourceIncomplete: true, SourceIntegrationPlanned: tr.PlannedIntegration, SourceRoutePlanned: tr.PlannedRoute, PlannedIntegration: true, PlannedShared: tr.PlannedShared}
			j.Transition.Fingerprint = onCallFingerprint(old, replacement)
			config[onCallPlanKey] = onCallPlan{Journal: j}
			return oldDesired, nil
		}
		tr.Disposition = "cancellation"
		tr.Phase = "Cancelling"
	}
	// Reconstruct source and target with their own inventoried identities.
	source := map[resource.Name]*resource.DesiredComposed{}
	if tr.FromType != "" {
		source, err = onCallRenderConfiguration(j, tr.Source)
		if err != nil {
			return nil, err
		}
		if err = onCallCapturePair(j, tr.FromType, observed, source); err != nil {
			return nil, err
		}
		if tr.SourceIncomplete {
			si, sr := onCallKeys(tr.FromType)
			if !tr.SourceIntegrationPlanned {
				delete(source, si)
			}
			if !tr.SourceRoutePlanned {
				delete(source, sr)
			}
		}
	}
	target, err := onCallRenderConfiguration(j, tr.Target)
	if err != nil {
		return nil, err
	}
	issuedShared := map[string]bool{}
	for _, key := range tr.PlannedShared {
		issuedShared[key] = true
	}
	if tr.FromType == "" {
		for key, o := range observed {
			if !issuedShared[string(key)] {
				continue
			}
			if o.Resource == nil || !onCallOwned(j, key, o.Resource.UnstructuredContent(), target[key]) {
				return nil, fmt.Errorf("OnCall fresh shared observation mismatch")
			}
			old := tr.Target.Bindings[string(key)]
			b := onCallBinding{ID: observedExternalName(observed, key), UID: string(o.Resource.GetUID())}
			if (old.ID != "" && old.ID != b.ID) || (old.UID != "" && old.UID != b.UID) {
				return nil, fmt.Errorf("OnCall shared binding changed")
			}
			tr.Target.Bindings[string(key)] = b
		}
		target, err = onCallRenderConfiguration(j, tr.Target)
		if err != nil {
			return nil, err
		}
		for key := range target {
			if key == "integration" || key == "integration-grafana-alerting" || key == "catch-all-route" || key == "catch-all-route-grafana-alerting" {
				continue
			}
			if !issuedShared[string(key)] {
				tr.PlannedShared = append(tr.PlannedShared, string(key))
			}
		}
	}
	if err = onCallCapturePair(j, tr.ToType, observed, target); err != nil {
		return nil, err
	}
	target, err = onCallRenderConfiguration(j, tr.Target)
	if err != nil {
		return nil, err
	}
	ik, rk := onCallKeys(tr.ToType)
	if tr.Target.Selection.ChannelMode == "reference" && tr.Target.Selection.ChannelID == "" {
		key := onCallSlackKey(tr.ToType)
		lookup := target[key]
		o, exists := observed[key]
		if exists && o.Resource != nil && onCallOwned(j, key, o.Resource.UnstructuredContent(), lookup) && observedDesiredCurrent(o, lookup) {
			status, _ := o.Resource.UnstructuredContent()["status"].(map[string]any)
			at, _ := status["atProvider"].(map[string]any)
			id := stringValue(at, "slackID", "")
			if !onCallValidSlackID(id) {
				return nil, fmt.Errorf("OnCall Slack reference did not resolve a valid destination")
			}
			tr.Target.Selection.ChannelID = id
			tr.Target.Bindings[string(key)] = onCallBinding{ID: observedExternalName(observed, key), UID: string(o.Resource.GetUID())}
			tr.Fingerprint = onCallFingerprint(tr.Source, tr.Target)
		}
	}
	desired := source
	for key, child := range target {
		if key == ik || key == rk {
			continue
		}
		if desired[key] == nil && (tr.FromType != "" || issuedShared[string(key)]) {
			desired[key] = child
		}
	}
	if tr.PlannedIntegration {
		desired[ik] = target[ik]
	}
	if tr.PlannedRoute {
		if target[rk] == nil {
			return nil, fmt.Errorf("OnCall planned Route has lost its Integration identity")
		}
		desired[rk] = target[rk]
	}
	if observedPhase == "Building" && tr.Phase == "Building" {
		if j.Inventory[tr.ToType].Integration.ID != "" && !tr.PlannedRoute && (tr.Target.Selection.ChannelMode != "reference" || tr.Target.Selection.ChannelID != "") {
			tr.PlannedRoute = true
		} else if tr.PlannedRoute && onCallBarrier(j, tr.Target, observed) {
			if tr.FromType == "" {
				j.ActiveType = tr.ToType
				j.ActiveConfiguration = tr.Target
				j.Transition = nil
				plan.CurrentRequest = onCallCurrentSelection(xr, j.ActiveConfiguration)
			} else {
				tr.Phase = "Prepared"
				tr.KeepType = tr.ToType
				tr.RetireType = tr.FromType
			}
		}
	}
	if observedPhase == "Cancelling" && tr.Disposition == "cancellation" && onCallBarrier(j, tr.Source, observed) {
		tr.Phase = "Prepared"
		tr.KeepType = tr.FromType
		tr.RetireType = tr.ToType
	}
	keepConfig := tr.Target
	if tr.Disposition == "cancellation" {
		keepConfig = tr.Source
	}
	if observedPhase == "Prepared" {
		if !onCallBarrier(j, keepConfig, observed) {
			return nil, fmt.Errorf("OnCall keep pair is not current; preserving migration")
		}
		tr.Phase = "RetiringRoute"
	}
	if committed {
		if !onCallBarrier(j, keepConfig, observed) {
			return nil, fmt.Errorf("OnCall committed keep pair is not current; preserving migration")
		}
		li, lr := onCallKeys(tr.RetireType)
		delete(desired, lr)
		plan.Prune = append(plan.Prune, lr)
		routeAbsent, e := onCallNamedAbsent(ctx, j, tr.RetireType, true, observed)
		if e != nil {
			return nil, e
		}
		if observedPhase == "RetiringRoute" && routeAbsent {
			tr.Phase = "RetiringIntegration"
		}
		if observedPhase == "RetiringIntegration" {
			delete(desired, li)
			plan.Prune = append(plan.Prune, li)
			integrationAbsent, e := onCallNamedAbsent(ctx, j, tr.RetireType, false, observed)
			if e != nil {
				return nil, e
			}
			if routeAbsent && integrationAbsent {
				entry := j.Inventory[tr.RetireType]
				entry.State = "Retired"
				entry.RetiredTransaction = tr.Fingerprint
				j.Inventory[tr.RetireType] = entry
				delete(desired, onCallSlackKey(tr.RetireType))
				plan.Prune = append(plan.Prune, onCallSlackKey(tr.RetireType))
				j.ActiveType = tr.KeepType
				j.ActiveConfiguration = keepConfig
				j.Transition = nil
			}
		}
	}
	if j.Transition != nil {
		j.Transition.Fingerprint = onCallFingerprint(j.Transition.Source, j.Transition.Target)
	}
	plan.CurrentRequest = j.Transition == nil && onCallCurrentSelection(xr, j.ActiveConfiguration)
	if keepConfig.Selection.ChannelMode != "none" {
		_, rk := onCallKeys(keepConfig.Selection.Type)
		plan.PendingDestination = !onCallSlackMatches(observed[rk], keepConfig.Selection.ChannelID)
	}
	config[onCallPlanKey] = plan
	for key, child := range desired {
		if child == nil {
			delete(desired, key)
		}
	}
	return desired, nil
}

func pruneOnCallDesired(rsp *fnv1.RunFunctionResponse, plan onCallPlan) {
	if rsp.GetDesired() == nil {
		return
	}
	for _, key := range plan.Prune {
		delete(rsp.Desired.Resources, string(key))
	}
}
func onCallPlanStatus(status *resource.Composite, plan onCallPlan) error {
	if plan.Journal == nil {
		return nil
	}
	data, err := json.Marshal(plan.Journal)
	if err != nil {
		return fmt.Errorf("cannot serialize OnCall journal")
	}
	var journal map[string]any
	if json.Unmarshal(data, &journal) != nil {
		return fmt.Errorf("cannot serialize OnCall journal")
	}
	content := status.Resource.UnstructuredContent()
	s, _ := content["status"].(map[string]any)
	s["onCallIdentity"] = journal
	return nil
}
