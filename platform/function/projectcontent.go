package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composite"
)

const projectContentObserver resource.Name = "project-observed-stack"
const projectContentStatusKey = "_projectContentStatus"

type projectContentProfile struct {
	name, namespace, consumerProfile         string
	project, central, label, git, repository map[string]any
}

func configuredProjectContent(xr, config map[string]any) (projectContentProfile, error) {
	meta, _ := xr["metadata"].(map[string]any)
	spec, _ := xr["spec"].(map[string]any)
	p := projectContentProfile{name: stringValue(spec, "profile", ""), namespace: stringValue(meta, "namespace", "")}
	if p.name == "" || p.namespace == "" || meta["name"] != p.name {
		return p, fmt.Errorf("project-content requires metadata.name equal to spec.profile and a namespace")
	}
	cs, _ := config["spec"].(map[string]any)
	profiles, _ := cs["projectContentProfiles"].([]any)
	var selected map[string]any
	for _, raw := range profiles {
		candidate, _ := raw.(map[string]any)
		if candidate["name"] == p.name {
			if selected != nil {
				return p, fmt.Errorf("duplicate project-content profile")
			}
			selected = candidate
		}
	}
	if selected == nil || selected["namespace"] != p.namespace {
		return p, fmt.Errorf("project-stack profile does not authorize this project namespace")
	}
	p.consumerProfile = stringValue(selected, "consumerProfile", "")
	p.project, _ = selected["projectStack"].(map[string]any)
	p.central, _ = selected["centralStack"].(map[string]any)
	p.label, _ = selected["projectLabel"].(map[string]any)
	p.git, _ = selected["git"].(map[string]any)
	p.repository, _ = selected["repository"].(map[string]any)
	for _, stack := range []map[string]any{p.project, p.central} {
		if stringValue(stack, "slug", "") == "" || stringValue(stack, "region", "") == "" {
			return p, fmt.Errorf("project-content profile must authorize both stack slug and region pairs")
		}
	}
	if p.project["slug"] == p.central["slug"] {
		return p, fmt.Errorf("project-content requires two distinct stacks")
	}
	if stringValue(p.project, "providerConfigName", "") == "" || p.git == nil || p.repository == nil {
		return p, fmt.Errorf("project-stack content profile is incomplete")
	}
	labelName, labelValue := stringValue(p.label, "name", ""), stringValue(p.label, "value", "")
	if !regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(labelName) || labelValue == "" {
		return p, fmt.Errorf("central-stack requires a nonempty platform-owned project label equality")
	}
	consumer, err := configuredStackConsumerProfile(config, p.consumerProfile, p.namespace, stringValue(p.central, "slug", ""), stringValue(p.central, "region", ""))
	if err != nil {
		return p, fmt.Errorf("central-stack: %w", err)
	}
	if len(consumer.scopes) != 2 || !((consumer.scopes[0] == "metrics:read" && consumer.scopes[1] == "logs:read") || (consumer.scopes[1] == "metrics:read" && consumer.scopes[0] == "logs:read")) {
		return p, fmt.Errorf("central-stack consumer scopes must be exactly metrics:read and logs:read")
	}
	for _, raw := range profiles {
		other, _ := raw.(map[string]any)
		if other["name"] != p.name && other["consumerProfile"] == p.consumerProfile {
			return p, fmt.Errorf("central-stack credential profile cannot be shared between projects")
		}
	}
	return p, nil
}

func projectContentGitClaim(p projectContentProfile) map[string]any {
	spec := map[string]any{}
	for key, value := range p.git {
		spec[key] = value
	}
	spec["stackRef"] = map[string]any{"name": p.project["providerConfigName"]}
	return map[string]any{"kind": "GrafanaProvisioningConnection", "metadata": map[string]any{"name": p.name + "-git", "namespace": p.namespace}, "spec": spec}
}

// Resolve the same namespace-bound Secret used by the existing Git Sync path.
// The shared config map remains aliased for expiry and other status producers.
func projectContentCredentialConfig(req *fnv1.RunFunctionRequest, rsp *fnv1.RunFunctionResponse, xr, config map[string]any) map[string]any {
	if xr["kind"] != "GrafanaProjectContent" {
		return config
	}
	p, err := configuredProjectContent(xr, config)
	if err != nil {
		return config
	}
	return provisioningConnectionCredentialConfig(req, rsp, projectContentGitClaim(p), config)
}

func renderProjectContent(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	state := map[string]any{"projectStack": "Waiting", "centralStack": "Waiting", "ready": false}
	config[projectContentStatusKey] = state
	p, err := configuredProjectContent(xr, config)
	if err != nil {
		side := "projectStack"
		if strings.Contains(fmt.Sprint(err), "central-stack") {
			side = "centralStack"
		}
		state[side] = "Refused"
		return nil, err
	}
	for name, child := range observed {
		if child.Resource == nil {
			continue
		}
		status, _ := child.Resource.UnstructuredContent()["status"].(map[string]any)
		conditions, _ := status["conditions"].([]any)
		for _, raw := range conditions {
			condition, _ := raw.(map[string]any)
			if condition["type"] == "Synced" && condition["status"] == "False" {
				side := "projectStack"
				if name == stackConsumerObserverName || name == stackConsumerPolicyName || name == stackConsumerTokenName || name == stackConsumerCredentialsName {
					side = "centralStack"
				}
				state[side] = "Refused"
				// Do not echo provider error text: it may contain credentials.
			}
		}
	}
	// Validate all static content before producing either stack's credential.
	gitObserved := map[resource.Name]resource.ObservedComposed{}
	for _, key := range []resource.Name{"credential", "secure-value", "connection"} {
		if child, ok := observed[resource.Name("git-"+string(key))]; ok {
			gitObserved[key] = child
		}
	}
	git, err := renderProvisioningConnection(projectContentGitClaim(p), gitObserved, config)
	if err != nil {
		state["projectStack"] = "Refused"
		return nil, fmt.Errorf("project-stack Git Sync: %w", err)
	}
	repository := map[string]any{}
	for key, value := range p.repository {
		repository[key] = value
	}
	repository["uid"] = p.name + "-repository"
	repository["connectionRef"] = map[string]any{"name": p.name + "-git"}
	if sync, _ := repository["sync"].(map[string]any); sync["target"] != "folder" {
		state["projectStack"] = "Refused"
		return nil, fmt.Errorf("project-stack repository must use folder sync")
	}
	repoXR := map[string]any{"metadata": map[string]any{"name": p.name + "-repository", "namespace": p.namespace}, "spec": map[string]any{"stackRef": map[string]any{"name": p.project["providerConfigName"]}, "repository": repository}}
	repo, err := renderProvisioningRepository(repoXR, nil, config)
	if err != nil {
		state["projectStack"] = "Refused"
		return nil, fmt.Errorf("project-stack repository: %w", err)
	}
	consumer, _ := configuredStackConsumerProfile(config, p.consumerProfile, p.namespace, p.central["slug"].(string), p.central["region"].(string))
	projectObserverProfile := stackConsumerProfile{consumerName: p.name + "-project", providerConfigName: consumer.providerConfigName}
	projectObserver := stackConsumerObserver(p.namespace, p.project["slug"].(string), p.project["region"].(string), projectObserverProfile)
	centralObserver := stackConsumerObserver(p.namespace, p.central["slug"].(string), p.central["region"].(string), consumer)
	desired := map[resource.Name]*resource.DesiredComposed{projectContentObserver: projectObserver, stackConsumerObserverName: centralObserver}
	projectObserved := map[resource.Name]resource.ObservedComposed{}
	if child, found := observed[projectContentObserver]; found {
		projectObserved[stackConsumerObserverName] = child
	}
	_, projectReady, err := observedStackConsumerIdentity(projectObserved, projectObserver, p.namespace, p.project["slug"].(string), p.project["region"].(string), projectObserverProfile)
	if err != nil || (!projectReady && observedExists(observed, stackConsumerPolicyName)) {
		// The project ID authorizes readiness, but is not an input to any
		// child spec. Keep authoring existing content from the approved profile.
		state["projectStack"] = "Refused"
	}
	stackID, centralReady, err := observedStackConsumerIdentity(observed, centralObserver, p.namespace, p.central["slug"].(string), p.central["region"].(string), consumer)
	if err != nil {
		state["centralStack"] = "Refused"
		// The existing policy can still supply the provider-assigned ID;
		// refuse readiness and re-author the observer from the profile.
	}
	// A policy already binds the provider-assigned stack identity. Recover only
	// that identity when the observer loses it; all authored fields are rendered
	// again from the profile, never copied wholesale from an observed child.
	if !centralReady && observedExists(observed, stackConsumerPolicyName) {
		state["centralStack"] = "Refused"
		stackID = observedString(observed, stackConsumerPolicyName, "spec.forProvider.realm[0].identifier")
		parsed, parseErr := strconv.ParseInt(stackID, 10, 64)
		if parseErr != nil || parsed <= 0 {
			return nil, fmt.Errorf("central-stack provider identity is unavailable")
		}
		centralReady = true
	}
	if projectReady && state["projectStack"] != "Refused" {
		state["projectStack"] = "Provisioning"
	}
	if centralReady && state["centralStack"] != "Refused" {
		state["centralStack"] = "Provisioning"
	}
	if (!projectReady && !observedExists(observed, stackConsumerPolicyName)) || !centralReady {
		return projectContentWait(desired, observed, state)
	}
	selector := "{" + p.label["name"].(string) + "=" + strconv.Quote(p.label["value"].(string)) + "}"
	settings := configuredPlatformSettings(config)
	lifetime, err := boundedTokenLifetime(settings.maximumTokenLifetime, requestedTokenLifetime)
	if err != nil {
		state["centralStack"] = "Refused"
		return nil, err
	}
	window, err := boundedTokenEarlyRotationWindow(lifetime)
	if err != nil {
		state["centralStack"] = "Refused"
		return nil, err
	}
	subnets, err := selectedTokenUseAllowedSubnets(settings.tokenUseNetworkProfiles, p.consumerProfile)
	if err != nil {
		state["centralStack"] = "Refused"
		return nil, err
	}
	policy := stackConsumerPolicy(p.namespace, p.central["slug"].(string), p.central["region"].(string), stackID, subnets, consumer)
	policy.Resource.UnstructuredContent()["spec"].(map[string]any)["forProvider"].(map[string]any)["realm"].([]any)[0].(map[string]any)["labelPolicy"] = []any{map[string]any{"selector": selector}}
	desired[stackConsumerPolicyName] = policy
	policyID := observedString(observed, stackConsumerPolicyName, "status.atProvider.policyId")
	policyIDFromStatus := policyID != ""
	if policyID == "" && observedExists(observed, stackConsumerTokenName) {
		state["centralStack"] = "Refused"
		policyID = observedString(observed, stackConsumerTokenName, "spec.forProvider.accessPolicyId")
	}
	if policyID == "" {
		return projectContentWait(desired, observed, state)
	}
	// Validate both fresh and recovered identities against the same external
	// identity contract as StackConsumer. Never preserve a mismatched import
	// annotation: the provider ID and approved region author the desired one.
	policyExternalName := p.central["region"].(string) + ":" + policyID
	if actual := observed[stackConsumerPolicyName]; actual.Resource != nil {
		if externalName := stackConsumerObservedExternalName(actual.Resource.UnstructuredContent()); externalName != "" && externalName != policyExternalName {
			state["centralStack"] = "Refused"
			if !policyIDFromStatus {
				// Conflicting recovery sources cannot authorize either policy
				// import identity or the token's target. Preserve through Fatal.
				return nil, fmt.Errorf("central-stack recovered policy identity conflicts with its import identity")
			}
		}
	}
	policy.Resource.UnstructuredContent()["metadata"].(map[string]any)["annotations"] = map[string]any{"crossplane.io/external-name": policyExternalName}
	token := stackConsumerToken(p.namespace, p.central["slug"].(string), p.central["region"].(string), policyID, lifetime, window, consumer)
	stackConsumerPreserveObservedExternalName(token, observed, stackConsumerTokenName)
	desired[stackConsumerTokenName] = token
	desired[stackConsumerCredentialsName] = stackConsumerCredentials(p.namespace, p.central["slug"].(string), p.central["region"].(string), consumer, settings)
	for _, key := range []resource.Name{stackConsumerPolicyName, stackConsumerTokenName, stackConsumerCredentialsName} {
		label := "observed credential child"
		if key == stackConsumerPolicyName {
			label = "observed access policy"
		}
		if !projectContentObservedMatches(observed, desired, key, label) {
			state["centralStack"] = "Refused"
		}
	}
	// Endpoint and tenant identity come only from the observed central stack.
	endpoints := map[string][3]string{}
	for kind, fields := range map[string][3]string{"metrics": {"prometheusUrl", "prometheusUserId", "prometheus"}, "logs": {"logsUrl", "logsUserId", "loki"}} {
		url := observedString(observed, stackConsumerObserverName, "status.atProvider."+fields[0])
		user := observedIntegerString(observed, stackConsumerObserverName, "status.atProvider."+fields[1])
		if url == "" || user == "" {
			return projectContentWait(desired, observed, state)
		}
		if !provisioningHTTPSURL(url) {
			state["centralStack"] = "Refused"
			return nil, fmt.Errorf("central-stack backend URL is invalid")
		}
		endpoints[kind] = [3]string{url, user, fields[2]}
	}
	secretName := p.name + "-datasource-credentials"
	desired["datasource-credentials"] = newDesired("external-secrets.io/v1", "ExternalSecret", p.namespace, secretName, nil, map[string]any{
		"refreshInterval": "1h", "secretStoreRef": settings.secretStoreReference(),
		"target": map[string]any{"name": secretName, "creationPolicy": "Owner", "deletionPolicy": "Retain", "template": map[string]any{"engineVersion": "v2", "data": map[string]any{"secureJsonData": `{"basicAuthPassword":{{ .token | toJson }}}`}}},
		"data":   []any{map[string]any{"secretKey": "token", "remoteRef": map[string]any{"key": consumer.outputSecretPath, "property": "access_policy_token"}}},
	})
	for key, child := range git {
		desired[resource.Name("git-"+string(key))] = child
	}
	// Readiness is status only. Once identities and Git credentials exist, emit
	// the complete authored set even when a provider reports refusal.
	if desired["git-connection"] == nil {
		return projectContentWait(desired, observed, state)
	}
	for kind, connection := range endpoints {
		uid := p.name + "-" + kind
		desired[resource.Name(kind+"-datasource")] = newDesired("oss.grafana.m.crossplane.io/v1alpha1", "DataSource", p.namespace, uid, map[string]any{"crossplane.io/external-name": uid}, map[string]any{"managementPolicies": managementPolicies, "providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": p.project["providerConfigName"]}, "forProvider": map[string]any{"uid": uid, "name": uid, "type": connection[2], "url": connection[0], "accessMode": "proxy", "basicAuthEnabled": true, "basicAuthUsername": connection[1], "secureJsonDataEncodedSecretRef": map[string]any{"name": secretName, "key": "secureJsonData"}}})
	}
	folder := p.name + "-folder"
	desired["folder"] = newDesired("oss.grafana.m.crossplane.io/v1alpha1", "Folder", p.namespace, folder, map[string]any{"crossplane.io/external-name": folder}, map[string]any{"managementPolicies": managementPolicies, "providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": p.project["providerConfigName"]}, "forProvider": map[string]any{"uid": folder, "title": p.name}})
	for key, child := range repo {
		desired[key] = child
	}
	ready := state["projectStack"] != "Refused" && state["centralStack"] != "Refused"
	for key := range desired {
		ready = ready && projectContentReady(observed, key)
	}
	state["ready"] = ready
	if ready {
		state["centralStack"] = "Ready"
		state["projectStack"] = "Ready"
	}
	return desired, nil
}

func projectContentObservedMatches(observed map[resource.Name]resource.ObservedComposed, desired map[resource.Name]*resource.DesiredComposed, key resource.Name, label string) bool {
	actual, exists := observed[key]
	if !exists || actual.Resource == nil {
		return true
	}
	expected := desired[key]
	return stackConsumerObservedChildMatchesWithLabelPolicy(actual.Resource.UnstructuredContent(), expected.Resource.UnstructuredContent(), label, true) == nil
}

func projectContentWait(desired map[resource.Name]*resource.DesiredComposed, observed map[resource.Name]resource.ObservedComposed, state map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	hasDownstream := false
	for key := range observed {
		hasDownstream = hasDownstream || (key != projectContentObserver && key != stackConsumerObserverName)
	}
	if hasDownstream && !observedExists(observed, projectContentObserver) {
		state["projectStack"] = "Refused"
		return nil, fmt.Errorf("project-content project-stack identity disappeared; refusing to withdraw existing children")
	}
	if hasDownstream && !observedExists(observed, stackConsumerObserverName) {
		state["centralStack"] = "Refused"
		return nil, fmt.Errorf("project-content central-stack identity disappeared; refusing to withdraw existing children")
	}
	for key := range observed {
		if desired[key] == nil {
			side := "projectStack"
			if key == stackConsumerObserverName || key == stackConsumerPolicyName || key == stackConsumerTokenName || key == stackConsumerCredentialsName {
				side = "centralStack"
			}
			state[side] = "Refused"
			return nil, fmt.Errorf("project-content dependency unavailable; refusing to withdraw existing child %s", key)
		}
	}
	return desired, nil
}

func projectContentReady(observed map[resource.Name]resource.ObservedComposed, name resource.Name) bool {
	child, ok := observed[name]
	return ok && child.Resource != nil && requiredStackReady(child.Resource.UnstructuredContent())
}

func desiredProjectContentStatus(config map[string]any) (*resource.Composite, bool) {
	state, _ := config[projectContentStatusKey].(map[string]any)
	if state == nil {
		state = map[string]any{"projectStack": "Refused", "centralStack": "Waiting", "ready": false}
	}
	result := &resource.Composite{Resource: composite.New()}
	result.Resource.SetUnstructuredContent(map[string]any{"status": map[string]any{"projectContent": state}})
	ready, _ := state["ready"].(bool)
	return result, ready
}
