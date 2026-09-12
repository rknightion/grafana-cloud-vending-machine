package main

import (
	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

const provisioningConnectionRendererImplemented = true

const (
	provisioningConnectionCredentialKey  = "privateKey"
	provisioningConnectionExternalSecret = "credential"
	provisioningConnectionSecureValue    = "secure-value"
	provisioningConnectionConnection     = "connection"
)

// renderProvisioningConnection bridges a secret-store value into a Grafana
// secure value before a GitHub App Connection references it. The claim never
// carries a credential literal or the provider's secure-map create form.
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
	desired[provisioningConnectionConnection] = newDesired("oss.grafana.m.crossplane.io/v1alpha1", "ConnectionV0Alpha1", namespace, connectionUID, annotations(connectionUID), map[string]any{
		"managementPolicies": managementPolicies,
		"forProvider": map[string]any{
			"metadata":      map[string]any{"uid": connectionUID},
			"secure":        map[string]any{"privateKey": map[string]any{"name": secureValueUID}},
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
