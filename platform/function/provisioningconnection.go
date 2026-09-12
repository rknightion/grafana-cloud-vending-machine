package main

import (
	"github.com/crossplane/function-sdk-go/errors"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"
)

const provisioningConnectionRendererImplemented = true

const (
	provisioningConnectionCredentialKey  = "privateKey"
	provisioningConnectionExternalSecret = "credential"
	provisioningConnectionSecureValue    = "secure-value"
	provisioningConnectionConnection     = "connection"
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

	credentialSecret := name + "-credential"
	secureValueUID := name + "-secure-value"
	connectionUID := name
	settings := configuredPlatformSettings(config)
	annotations := func(externalName string) map[string]any {
		return map[string]any{"crossplane.io/external-name": externalName}
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
		provisioningConnectionSecureValue: newDesired("enterprise.grafana.m.crossplane.io/v1alpha1", "SecurevalueV1Beta1", namespace, secureValueUID, annotations(secureValueUID), map[string]any{
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
		}),
	}

	// The provider only accepts the connection after the secure value exists.
	// Its UID is claim-derived, so no provider-assigned identifier is guessed.
	if !observedExists(observed, provisioningConnectionSecureValue) {
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
	desired[provisioningConnectionConnection] = newDesired("oss.grafana.m.crossplane.io/v1alpha1", "ConnectionV0Alpha1", namespace, connectionUID, annotations(connectionUID), map[string]any{
		"managementPolicies": managementPolicies,
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
	return desired, nil
}
