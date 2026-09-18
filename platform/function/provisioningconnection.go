package main

import (
	"github.com/crossplane/function-sdk-go/errors"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composite"
)

const provisioningConnectionRendererImplemented = true

const (
	provisioningConnectionCredentialKey   = "privateKey"
	provisioningConnectionExternalSecret  = "credential"
	provisioningConnectionSecureValue     = "secure-value"
	provisioningConnectionConnection      = "connection"
	provisioningConnectionDeletionProfile = "provisioning-connection"
)

// The credential requirement and the config key it resolves into. This is the
// one place in this package where a credential VALUE, not a reference, crosses
// into a renderer, and section "Git Sync credential exposure" of the catalog
// README records why.
const (
	provisioningConnectionCredentialRequirement = "provisioning-connection-credential"
	provisioningConnectionCredentialConfigKey   = "_provisioningConnectionCredential"
)

// provisioningConnectionCredentialConfig passes the ExternalSecret-materialised
// private key to the renderer, base64 as Kubernetes already stores it.
//
// Grafana Cloud refuses a Connection whose secure.privateKey is the reference
// form {name: <securevalue>} with "403 identity type access-policy not allowed",
// an error about neither the caller nor an access policy: it was reproduced
// identically from a stack service-account token and from an interactive user,
// against secure values created by each. Only the create form is accepted, and
// only base64-encoded. The pinned provider offers no secretRef-shaped field on
// this kind, and initProvider is not an escape because LateInitialize in the
// common management policies promotes it into forProvider on the first reconcile.
//
// So the value has to reach forProvider. It is never decoded here: a Kubernetes
// Secret's data entry is already base64, which is exactly the encoding Grafana
// requires, so the plaintext PEM never exists inside this function.
//
// It adds its key IN PLACE and returns the same map. Do not copy the config
// here: expiry.go:53 writes the expiry status into whatever map the renderer
// was handed, and fn.go reads it back out of its own `config` reference to
// build the composite status. Those are the same map today, and a copy at this
// seam silently drops the expiry status instead of failing - four expiry tests
// are the only thing that catches it.
func provisioningConnectionCredentialConfig(req *fnv1.RunFunctionRequest, rsp *fnv1.RunFunctionResponse, xr, config map[string]any) map[string]any {
	kind, _ := xr["kind"].(string)
	if kind != "GrafanaProvisioningConnection" || config == nil {
		return config
	}
	delete(config, provisioningConnectionCredentialConfigKey)
	metadata, _ := xr["metadata"].(map[string]any)
	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	if name == "" || namespace == "" {
		return config
	}
	secretName := name + "-credential"
	if rsp.Requirements == nil {
		rsp.Requirements = &fnv1.Requirements{}
	}
	if rsp.Requirements.Resources == nil {
		rsp.Requirements.Resources = map[string]*fnv1.ResourceSelector{}
	}
	rsp.Requirements.Resources[provisioningConnectionCredentialRequirement] = &fnv1.ResourceSelector{
		ApiVersion: "v1", Kind: "Secret", Namespace: &namespace,
		Match: &fnv1.ResourceSelector_MatchName{MatchName: secretName},
	}
	secrets, resolved, err := request.GetRequiredResource(req, provisioningConnectionCredentialRequirement)
	if err != nil || !resolved || len(secrets) != 1 || secrets[0].Resource == nil {
		return config
	}
	secret := secrets[0].Resource.UnstructuredContent()
	secretMeta, _ := secret["metadata"].(map[string]any)
	data, _ := secret["data"].(map[string]any)
	encoded, _ := data[provisioningConnectionCredentialKey].(string)
	// Identity is checked before the value is trusted: a Secret of the right
	// name in the wrong namespace, or one being deleted, is not this claim's.
	if secret["apiVersion"] != "v1" || secret["kind"] != "Secret" || secretMeta["name"] != secretName || secretMeta["namespace"] != namespace || secretMeta["deletionTimestamp"] != nil || encoded == "" {
		return config
	}
	config[provisioningConnectionCredentialConfigKey] = encoded
	return config
}

// renderProvisioningConnection bridges a secret-store value into a Grafana
// GitHub App Connection. The CLAIM never carries a credential literal or the
// provider's secure-map create form, and those guards are unchanged; the
// composed Connection does, because Grafana Cloud refuses the reference form.
// provisioningConnectionCredentialConfig carries the evidence and the reason.
//
// The Securevalue is still rendered. It is no longer what the Connection reads,
// so it duplicates the credential inside Grafana, and it is kept deliberately:
// it is the half of the chain Grafana accepts, so restoring the reference form
// once the 403 is fixed upstream is a one-line change here rather than a
// re-design. GCV-0074 tracks that, and the catalog README states the cost.
func renderProvisioningConnection(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	metadata, _ := xr["metadata"].(map[string]any)
	spec, _ := xr["spec"].(map[string]any)
	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	stackRef, _ := spec["stackRef"].(map[string]any)
	stackName, _ := stackRef["name"].(string)
	title, _ := spec["title"].(string)
	connectionType, _ := spec["type"].(string)
	description, _ := spec["description"].(string)
	url, _ := spec["url"].(string)
	github, _ := spec["github"].(map[string]any)
	appID, _ := github["appId"].(string)
	installationID, _ := github["installationId"].(string)
	credential, _ := spec["credential"].(map[string]any)
	remoteRef, _ := credential["remoteRef"].(map[string]any)
	remoteKey, _ := remoteRef["key"].(string)
	remoteProperty, _ := remoteRef["property"].(string)
	decrypters, _ := spec["decrypters"].([]any)
	secureVersion, hasSecureVersion := spec["secureVersion"]

	if name == "" || namespace == "" || stackName == "" || title == "" || connectionType != "github" || description == "" || !provisioningHTTPSURL(url) || appID == "" || installationID == "" || remoteKey == "" || remoteProperty == "" || len(decrypters) != 1 || !hasSecureVersion {
		return nil, errors.New("provisioning connection must set metadata name and namespace, stackRef.name, title, type github, description, url, github appId and installationId, credential.remoteRef key and property, decrypters, and secureVersion")
	}
	if _, found := spec["secure"]; found {
		return nil, errors.New("provisioning connection forbids provider secure maps; use credential.remoteRef")
	}
	if _, found := credential["inline"]; found {
		return nil, errors.New("provisioning connection forbids inline credentials; use credential.remoteRef")
	}
	if _, found := credential["create"]; found {
		return nil, errors.New("provisioning connection forbids secure-map create values; use credential.remoteRef")
	}
	decommission, err := provisioningConnectionDecommissionProgressFor(xr, observed, config)
	if err != nil {
		return nil, err
	}
	lifecycle := decommission.lifecycle

	credentialSecret := name + "-credential"
	secureValueUID := name + "-secure-value"
	connectionUID := name
	settings := configuredPlatformSettings(config)
	annotations := func(externalName string) map[string]any {
		return map[string]any{"crossplane.io/external-name": externalName}
	}
	secureValueSpec := map[string]any{
		"managementPolicies": managementPolicies,
		"forProvider": map[string]any{
			"metadata": map[string]any{"uid": secureValueUID},
			"spec": map[string]any{
				"description":    "Git Sync credential for " + title,
				"decrypters":     decrypters,
				"valueSecretRef": map[string]any{"name": credentialSecret, "key": provisioningConnectionCredentialKey},
			},
		},
		"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": stackName},
	}
	desired := map[resource.Name]*resource.DesiredComposed{
		provisioningConnectionExternalSecret: newDesired("external-secrets.io/v1", "ExternalSecret", namespace, credentialSecret, annotations(credentialSecret), map[string]any{
			"refreshInterval": "1h",
			"secretStoreRef":  settings.secretStoreReference(),
			// Retain preserves the last synced value if the remote source disappears;
			// deleting this owner still removes its Secret. External Grafana credential
			// deletion requires platform authorization and Connection-before-Securevalue
			// ordering, tracked separately from this retain-default renderer.
			"target": map[string]any{
				"name": credentialSecret, "creationPolicy": "Owner", "deletionPolicy": "Retain",
			},
			"data": []any{map[string]any{"secretKey": provisioningConnectionCredentialKey, "remoteRef": map[string]any{"key": remoteKey, "property": remoteProperty}}},
		}),
		provisioningConnectionSecureValue: newDesired("enterprise.grafana.m.crossplane.io/v1alpha1", "SecurevalueV1Beta1", namespace, secureValueUID, annotations(secureValueUID), secureValueSpec),
	}
	if decommission.terminating && !observedExists(observed, provisioningConnectionConnection) {
		// Each withdrawal is preceded by a persisted parent-status witness and
		// an observed Crossplane deletion policy/finalizer on that exact child.
		// This avoids treating a missing composed object as deletion evidence.
		if decommission.withdrawSecureValue {
			return map[resource.Name]*resource.DesiredComposed{}, nil
		}
		if decommission.withdrawConnection {
			delete(desired, provisioningConnectionConnection)
			return desired, nil
		}
		if decommission.prepareSecureValue || decommission.phase == "SecurevalueDeleting" {
			secureValueSpec["managementPolicies"] = []any{"*"}
		}
		return desired, nil
	}

	// The provider only accepts the connection after the secure value exists.
	// Its UID is claim-derived, so no provider-assigned identifier is guessed.
	if !observedExists(observed, provisioningConnectionSecureValue) {
		if decommission.terminating {
			return nil, errors.New("provisioning credential is not materialised; refusing to withdraw the existing connection")
		}
		return desired, nil
	}
	// Emit nothing rather than a Connection with no credential. The
	// ExternalSecret above materialises the Secret this reads, so on a first
	// reconcile it is legitimately absent for a moment.
	//
	// Once a Connection exists, though, the same absence means the Secret went
	// away underneath it - an ESO sync failure, or someone deleting it - and
	// returning the shorter desired set would WITHDRAW a live Connection and
	// its external Grafana resource. Fail instead, the way the on-call renderer
	// refuses to withdraw its own dependents.
	materialised, _ := config[provisioningConnectionCredentialConfigKey].(string)
	if materialised == "" {
		if observedExists(observed, provisioningConnectionConnection) {
			return nil, errors.New("provisioning credential is not materialised; refusing to withdraw the existing connection")
		}
		return desired, nil
	}
	connectionPolicies := managementPolicies
	if lifecycle == "Delete" {
		connectionPolicies = []any{"*"}
	}
	desired[provisioningConnectionConnection] = newDesired("oss.grafana.m.crossplane.io/v1alpha1", "ConnectionV0Alpha1", namespace, connectionUID, annotations(connectionUID), map[string]any{
		"managementPolicies": connectionPolicies,
		"forProvider": map[string]any{
			"metadata": map[string]any{"uid": connectionUID},
			// The create form, base64 as the Secret already stores it. The
			// reference form {name: secureValueUID} is what Grafana Cloud 403s;
			// see provisioningConnectionCredentialConfig for the evidence.
			"secure":        map[string]any{"privateKey": map[string]any{"create": materialised}},
			"secureVersion": secureVersion,
			"spec": map[string]any{
				"title": title, "type": connectionType, "description": description, "url": url,
				"github": map[string]any{"appId": appID, "installationId": installationID},
			},
		},
		"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": stackName},
	})
	if decommission.terminating {
		if decommission.withdrawConnection {
			delete(desired, provisioningConnectionConnection)
		}
		return desired, nil
	}
	return desired, nil
}

// provisioningConnectionExternalResourcesLifecycle reuses the stack's
// platform-owned authorization list. The profile is constant for this API, so
// a consumer cannot select a profile that happens to be authorized.
func provisioningConnectionExternalResourcesLifecycle(xr, config map[string]any) (string, error) {
	spec, _ := xr["spec"].(map[string]any)
	lifecycle, err := externalResourcesLifecycle(spec)
	if err != nil || lifecycle != "Delete" {
		return lifecycle, err
	}
	metadata, _ := xr["metadata"].(map[string]any)
	namespace, _ := metadata["namespace"].(string)
	name, _ := metadata["name"].(string)
	uid, _ := metadata["uid"].(string)
	if !configuredPlatformSettings(config).deletionAuthorized(namespace, name, uid, provisioningConnectionDeletionProfile) {
		return "", errors.Errorf("provisioning connection %s/%s is not authorized to delete external resources", namespace, name)
	}
	return lifecycle, nil
}

func provisioningConnectionTerminating(xr map[string]any) bool {
	metadata, _ := xr["metadata"].(map[string]any)
	return metadata["deletionTimestamp"] != nil
}

func provisioningConnectionDeletePrepared(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, name resource.Name) bool {
	r, ok := observed[name]
	if !ok || r.Resource == nil {
		return false
	}
	object := r.Resource.UnstructuredContent()
	metadata, _ := object["metadata"].(map[string]any)
	xrMetadata, _ := xr["metadata"].(map[string]any)
	xrName, _ := xrMetadata["name"].(string)
	xrNamespace, _ := xrMetadata["namespace"].(string)
	childName := xrName
	childKind := "ConnectionV0Alpha1"
	childAPI := "oss.grafana.m.crossplane.io/v1alpha1"
	if name == provisioningConnectionSecureValue {
		childName += "-secure-value"
		childKind = "SecurevalueV1Beta1"
		childAPI = "enterprise.grafana.m.crossplane.io/v1alpha1"
	}
	annotations, _ := metadata["annotations"].(map[string]any)
	finalizers, _ := metadata["finalizers"].([]any)
	if object["apiVersion"] != childAPI || object["kind"] != childKind || metadata["name"] != childName || metadata["namespace"] != xrNamespace || annotations["crossplane.io/external-name"] != childName || !oneOf("finalizer.managedresource.crossplane.io", stringValues(finalizers)...) {
		return false
	}
	spec, _ := object["spec"].(map[string]any)
	policies, _ := spec["managementPolicies"].([]any)
	return len(policies) == 1 && policies[0] == "*"
}

type provisioningConnectionDecommissionProgress struct {
	lifecycle           string
	phase               string
	complete            bool
	terminating         bool
	withdrawConnection  bool
	prepareSecureValue  bool
	withdrawSecureValue bool
}

func provisioningConnectionDecommissionProgressFor(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) (provisioningConnectionDecommissionProgress, error) {
	lifecycle, err := provisioningConnectionExternalResourcesLifecycle(xr, config)
	if err != nil {
		return provisioningConnectionDecommissionProgress{}, err
	}
	state := provisioningConnectionDecommissionProgress{lifecycle: lifecycle, phase: "Retained", terminating: provisioningConnectionTerminating(xr)}
	if lifecycle != "Delete" {
		return state, nil
	}
	connectionPrepared := provisioningConnectionDeletePrepared(xr, observed, provisioningConnectionConnection)
	secureValuePrepared := provisioningConnectionDeletePrepared(xr, observed, provisioningConnectionSecureValue)
	if !state.terminating {
		if connectionPrepared {
			state.phase = "Armed"
		} else {
			state.phase = "PreparingConnection"
		}
		return state, nil
	}
	priorPhase := provisioningConnectionObservedDecommissionPhase(xr)
	if observedExists(observed, provisioningConnectionConnection) {
		if !connectionPrepared {
			state.phase = "PreparingConnection"
			return state, nil
		}
		state.phase = "ConnectionDeleting"
		state.withdrawConnection = priorPhase == "ConnectionDeleting"
		return state, nil
	}
	connectionWitness := oneOf(priorPhase, "ConnectionDeleting", "SecurevalueArming", "SecurevalueDeleting")
	if !connectionWitness {
		state.phase = "ConnectionDeletionUnproven"
		return state, nil
	}
	if !observedExists(observed, provisioningConnectionSecureValue) {
		if priorPhase == "SecurevalueDeleting" {
			state.phase = "Complete"
			state.complete = true
		} else {
			state.phase = "SecurevalueDeletionUnproven"
		}
		return state, nil
	}
	switch priorPhase {
	case "ConnectionDeleting":
		state.phase = "SecurevalueArming"
		state.prepareSecureValue = true
	case "SecurevalueArming":
		if secureValuePrepared {
			state.phase = "SecurevalueDeleting"
		} else {
			state.phase = "SecurevalueArming"
			state.prepareSecureValue = true
		}
	case "SecurevalueDeleting":
		if secureValuePrepared {
			state.phase = "SecurevalueDeleting"
			state.withdrawSecureValue = true
		} else {
			state.phase = "SecurevalueArming"
			state.prepareSecureValue = true
		}
	}
	return state, nil
}

func provisioningConnectionObservedDecommissionPhase(xr map[string]any) string {
	status, _ := xr["status"].(map[string]any)
	decommission, _ := status["decommission"].(map[string]any)
	phase, _ := decommission["phase"].(string)
	return phase
}

// desiredProvisioningConnectionStatus records the decommission phase on the
// claim itself. It distinguishes an observed ordered completion from the
// absence of a child in a rendered desired set.
func desiredProvisioningConnectionStatus(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) *resource.Composite {
	state, _ := provisioningConnectionDecommissionProgressFor(xr, observed, config)
	decommission := map[string]any{"mode": state.lifecycle, "authorized": state.lifecycle == "Delete", "phase": state.phase, "complete": state.complete}
	desired := &resource.Composite{Resource: composite.New()}
	desired.Resource.SetUnstructuredContent(map[string]any{"status": map[string]any{"decommission": decommission}})
	return desired
}
