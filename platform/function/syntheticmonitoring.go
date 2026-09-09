package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
	"github.com/crossplane/function-sdk-go/resource/composite"
)

const syntheticMonitoringRendererImplemented = true

const (
	syntheticMonitoringInstallationName = "synthetic-monitoring-installation"
	syntheticMonitoringVerifierPrefix   = "synthetic-monitoring-verifier-"
	privateProbeUnsupportedMessage      = "privateProbes are not supported because the pinned provider cannot bound probe token lifetime"
)

type syntheticMonitoringBudget struct {
	maxAPI                    int
	maxBrowser                int
	maxProbeLocations         int
	minIntervalSeconds        int
	browserExecutionWeight    int
	maxWeightedExecutionsHour int
	verificationProbeName     string
}

type syntheticMonitoringCheck struct {
	name             string
	kind             string
	target           string
	frequencySeconds int
	probeNames       []any
	settings         map[string]any
	alerts           []any
}

func renderSyntheticMonitoring(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	metadata, _ := xr["metadata"].(map[string]any)
	spec, _ := xr["spec"].(map[string]any)
	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	stackRef, _ := spec["stackRef"].(map[string]any)
	stackName, _ := stackRef["name"].(string)
	if name == "" || namespace == "" || stackName == "" {
		return nil, errors.New("Synthetic Monitoring request must set metadata name and namespace plus stackRef.name")
	}
	if name != stackName {
		return nil, errors.New("metadata.name must match spec.stackRef.name so one composite owns the stack check set")
	}
	if _, requested := spec["privateProbes"]; requested {
		return nil, errors.New(privateProbeUnsupportedMessage)
	}

	stack, available, err := syntheticMonitoringStackContext(config, stackName, namespace)
	if err != nil {
		return nil, err
	}
	if !available {
		return map[resource.Name]*resource.DesiredComposed{}, nil
	}
	stackID, _ := stack["stackID"].(string)
	usage, _ := stack["usage"].(string)
	region, _ := stack["region"].(string)
	organization, _ := stack["organization"].(string)
	outputSecretPath, _ := stack["outputSecretPath"].(string)
	bootstrapSecretRef, _ := stack["bootstrapSecretRef"].(map[string]any)
	if stackID == "" || usage == "" || region == "" || organization == "" || outputSecretPath == "" {
		return nil, errors.New("trusted referenced stack context is incomplete")
	}
	stackIDInteger, err := strconv.ParseInt(stackID, 10, 64)
	if err != nil {
		return nil, errors.Errorf("trusted referenced stack ID %q is not numeric", stackID)
	}
	if bootstrapSecretRef["ready"] != true || bootstrapSecretRef["name"] == "" || bootstrapSecretRef["key"] == "" {
		return map[resource.Name]*resource.DesiredComposed{}, nil
	}

	settings := configuredPlatformSettings(config)
	organizationSettings, err := settings.resolveOrganization(organization, region, usage)
	if err != nil {
		return nil, err
	}
	budget, err := configuredSyntheticMonitoringBudget(config, usage)
	if err != nil {
		return nil, err
	}
	checks, _, err := syntheticMonitoringChecks(spec, budget)
	if err != nil {
		return nil, err
	}

	if installedStackID := observedString(observed, syntheticMonitoringInstallationName, "status.atProvider.stackId"); installedStackID != "" && installedStackID != stackID {
		return nil, errors.Errorf("Synthetic Monitoring installation stack %s does not match referenced stack %s", installedStackID, stackID)
	}
	derivedCredentialSecret := stackName + "-synthetic-monitoring-credentials"
	derivedCredentialPath := outputSecretPath + "/synthetic-monitoring"
	desired := map[resource.Name]*resource.DesiredComposed{}
	desired[syntheticMonitoringInstallationName] = newDesired(
		"sm.grafana.m.crossplane.io/v1alpha1", "Installation", namespace, stackName+"-synthetic-monitoring",
		map[string]any{"crossplane.io/external-name": stackID},
		map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider": map[string]any{
				"stackId": stackID,
				"metricsPublisherKeySecretRef": map[string]any{
					"name": bootstrapSecretRef["name"], "key": bootstrapSecretRef["key"],
				},
			},
			"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": organizationSettings.providerConfigName},
		},
	)

	derivedToken := observedString(observed, syntheticMonitoringInstallationName, "status.atProvider.smAccessToken")
	installationURL := observedString(observed, syntheticMonitoringInstallationName, "status.atProvider.stackSmApiUrl")
	if derivedToken == "" || installationURL == "" {
		return desired, nil
	}
	verifierResourceName := syntheticMonitoringVerifierResourceName(derivedToken)
	credentials, err := json.Marshal(map[string]string{"sm_access_token": derivedToken})
	if err != nil {
		return nil, errors.Wrap(err, "cannot encode derived Synthetic Monitoring credential")
	}
	persistedDocument, err := json.Marshal(map[string]string{"synthetic_monitoring_token": derivedToken})
	if err != nil {
		return nil, errors.Wrap(err, "cannot encode persisted Synthetic Monitoring credential")
	}
	derivedValues := map[string][]byte{
		"credentials":               credentials,
		"synthetic-monitoring.json": persistedDocument,
	}
	desired["synthetic-monitoring-derived-credential"] = syntheticMonitoringSecret(namespace, derivedCredentialSecret, derivedValues)
	if syntheticMonitoringObservedSecretMatches(observed, "synthetic-monitoring-derived-credential", namespace, derivedCredentialSecret, derivedValues) {
		desired["synthetic-monitoring-derived-credential"].Ready = resource.ReadyTrue
	}
	desired["synthetic-monitoring-credential-publish"] = syntheticMonitoringPushSecret(
		namespace, stackName+"-synthetic-monitoring-credential", derivedCredentialSecret,
		"synthetic-monitoring.json", `{{ index . "synthetic-monitoring.json" | toString }}`,
		derivedCredentialPath, settings,
	)
	desired["synthetic-monitoring-provider-config"] = newDesired(
		"grafana.m.crossplane.io/v1beta1", "ProviderConfig", namespace, stackName+"-synthetic-monitoring", nil,
		map[string]any{
			"credentials": map[string]any{
				"source":    "Secret",
				"secretRef": map[string]any{"name": derivedCredentialSecret, "namespace": namespace, "key": "credentials"},
			},
			"smUrl":   installationURL,
			"stackId": stackIDInteger,
		},
	)
	desired[verifierResourceName] = newDesired(
		"sm.grafana.m.crossplane.io/v1alpha1", "Check", namespace, stackName+"-"+string(verifierResourceName), nil,
		map[string]any{
			"managementPolicies": []any{"*"},
			"forProvider": map[string]any{
				"enabled": false, "frequency": 3600000,
				"job": "vending-installation-verifier", "target": "https://synthetic-monitoring-verification.invalid",
				"probeNames": []any{budget.verificationProbeName},
				"settings":   []any{map[string]any{"http": []any{map[string]any{"method": "GET"}}}},
			},
			"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": stackName + "-synthetic-monitoring"},
		},
	)
	if !syntheticMonitoringInstallationVerified(observed, stackID) {
		return desired, nil
	}

	for _, check := range checks {
		parameters := map[string]any{
			"enabled": true, "frequency": check.frequencySeconds * 1000,
			"job": check.name, "target": check.target, "probeNames": check.probeNames,
			"settings": []any{check.settings},
		}
		checkResourceName := resource.Name("check-" + check.name)
		desired[checkResourceName] = newDesired(
			"sm.grafana.m.crossplane.io/v1alpha1", "Check", namespace, stackName+"-"+check.name, nil,
			map[string]any{
				"managementPolicies": []any{"*"},
				"forProvider":        parameters,
				"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": stackName + "-synthetic-monitoring"},
			},
		)
		checkID, present := syntheticMonitoringObservedCheckID(observed, checkResourceName)
		if !present || len(check.alerts) == 0 {
			continue
		}
		desired[resource.Name("check-alerts-"+check.name)] = newDesired(
			"sm.grafana.m.crossplane.io/v1alpha1", "CheckAlerts", namespace, stackName+"-"+check.name+"-alerts",
			map[string]any{"crossplane.io/external-name": strconv.FormatFloat(checkID, 'f', -1, 64)},
			map[string]any{
				"managementPolicies": []any{"*"},
				"forProvider": map[string]any{
					"checkId": checkID,
					"alerts":  check.alerts,
				},
				"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": stackName + "-synthetic-monitoring"},
			},
		)
	}
	return desired, nil
}

func syntheticMonitoringStackContext(config map[string]any, stackName, namespace string) (map[string]any, bool, error) {
	stack, ok := config["referencedStack"].(map[string]any)
	if !ok {
		return nil, false, nil
	}
	if stack["name"] != stackName || stack["namespace"] != namespace {
		return nil, false, errors.New("trusted referenced stack context does not match the request")
	}
	return stack, true, nil
}

func configuredSyntheticMonitoringBudget(config map[string]any, usage string) (syntheticMonitoringBudget, error) {
	spec, _ := config["spec"].(map[string]any)
	profiles, _ := spec["syntheticMonitoringBudgets"].([]any)
	for _, value := range profiles {
		profile, _ := value.(map[string]any)
		if profile["usage"] != usage {
			continue
		}
		budget := syntheticMonitoringBudget{
			maxAPI:                    syntheticMonitoringInteger(profile["maxApiChecks"]),
			maxBrowser:                syntheticMonitoringInteger(profile["maxBrowserChecks"]),
			maxProbeLocations:         syntheticMonitoringInteger(profile["maxProbeLocationsPerCheck"]),
			minIntervalSeconds:        syntheticMonitoringInteger(profile["minCheckIntervalSeconds"]),
			browserExecutionWeight:    syntheticMonitoringInteger(profile["browserExecutionWeight"]),
			maxWeightedExecutionsHour: syntheticMonitoringInteger(profile["maxWeightedExecutionsPerHour"]),
			verificationProbeName:     stringValue(profile, "verificationProbeName", ""),
		}
		if budget.maxAPI < 0 || budget.maxBrowser < 0 || budget.maxProbeLocations < 1 || budget.minIntervalSeconds < 1 || budget.browserExecutionWeight < 1 || budget.maxWeightedExecutionsHour < 1 || budget.verificationProbeName == "" {
			return syntheticMonitoringBudget{}, errors.Errorf("Synthetic Monitoring budget for usage %q is incomplete", usage)
		}
		return budget, nil
	}
	return syntheticMonitoringBudget{}, errors.Errorf("no Synthetic Monitoring budget is configured for stack usage %q", usage)
}

func syntheticMonitoringChecks(spec map[string]any, budget syntheticMonitoringBudget) ([]syntheticMonitoringCheck, int, error) {
	items, _ := spec["checks"].([]any)
	checks := make([]syntheticMonitoringCheck, 0, len(items))
	seen := map[string]struct{}{}
	apiCount, browserCount, weighted := 0, 0, 0
	for index, item := range items {
		value, _ := item.(map[string]any)
		name, _ := value["name"].(string)
		kind, _ := value["type"].(string)
		target, _ := value["target"].(string)
		frequency := syntheticMonitoringInteger(value["frequencySeconds"])
		probeNames, _ := value["probeNames"].([]any)
		if name == "" || target == "" || frequency < 1 || len(probeNames) == 0 {
			return nil, 0, errors.Errorf("checks[%d] must set name, target, frequencySeconds, and at least one probe name", index)
		}
		if _, exists := seen[name]; exists {
			return nil, 0, errors.Errorf("checks[%d] repeats check name %q", index, name)
		}
		seen[name] = struct{}{}
		if len(probeNames) > budget.maxProbeLocations {
			return nil, 0, errors.Errorf("check %q uses %d probe locations; platform maximum is %d", name, len(probeNames), budget.maxProbeLocations)
		}
		if frequency < budget.minIntervalSeconds {
			return nil, 0, errors.Errorf("check %q runs every %d seconds; platform minimum interval is %d", name, frequency, budget.minIntervalSeconds)
		}
		for probeIndex, probe := range probeNames {
			if probeName, ok := probe.(string); !ok || probeName == "" {
				return nil, 0, errors.Errorf("check %q probeNames[%d] must be a non-empty string", name, probeIndex)
			}
		}

		alerts, err := syntheticMonitoringAlerts(value, name)
		if err != nil {
			return nil, 0, err
		}

		settings := map[string]any{}
		weight := 1
		switch kind {
		case "http":
			http, _ := value["http"].(map[string]any)
			method, _ := http["method"].(string)
			if method == "" || value["browser"] != nil {
				return nil, 0, errors.Errorf("HTTP check %q must set only http.method", name)
			}
			settings["http"] = []any{map[string]any{"method": method}}
			apiCount++
		case "browser":
			browser, _ := value["browser"].(map[string]any)
			script, _ := browser["script"].(string)
			if script == "" || value["http"] != nil {
				return nil, 0, errors.Errorf("browser check %q must set only browser.script", name)
			}
			settings["browser"] = []any{map[string]any{"script": script}}
			browserCount++
			weight = budget.browserExecutionWeight
		default:
			return nil, 0, errors.Errorf("unsupported check type %q", kind)
		}
		executionsPerHour := (3600 + frequency - 1) / frequency
		weighted += executionsPerHour * len(probeNames) * weight
		checks = append(checks, syntheticMonitoringCheck{
			name: name, kind: kind, target: target, frequencySeconds: frequency,
			probeNames: probeNames, settings: settings, alerts: alerts,
		})
	}
	if apiCount > budget.maxAPI {
		return nil, 0, errors.Errorf("API check count %d exceeds platform maximum %d", apiCount, budget.maxAPI)
	}
	if browserCount > budget.maxBrowser {
		return nil, 0, errors.Errorf("browser check count %d exceeds platform maximum %d", browserCount, budget.maxBrowser)
	}
	if weighted > budget.maxWeightedExecutionsHour {
		return nil, 0, errors.Errorf("weighted executions per hour %d exceeds platform maximum %d", weighted, budget.maxWeightedExecutionsHour)
	}
	return checks, weighted, nil
}

func syntheticMonitoringAlerts(check map[string]any, checkName string) ([]any, error) {
	if _, supplied := check["alerts"]; !supplied {
		return nil, nil
	}
	items, ok := check["alerts"].([]any)
	if !ok || len(items) == 0 {
		return nil, errors.Errorf("check %q must set at least one alert", checkName)
	}
	alerts := make([]any, 0, len(items))
	seen := map[string]struct{}{}
	for index, item := range items {
		alert, ok := item.(map[string]any)
		if !ok {
			return nil, errors.Errorf("check %q alerts[%d] must be an object", checkName, index)
		}
		name, _ := alert["name"].(string)
		period, _ := alert["period"].(string)
		runbookURL, _ := alert["runbookURL"].(string)
		threshold, ok := syntheticMonitoringNumber(alert["threshold"])
		if name == "" || !ok || threshold < 0 {
			return nil, errors.Errorf("check %q alerts[%d] must set name and a non-negative threshold", checkName, index)
		}
		if _, exists := seen[name]; exists {
			return nil, errors.Errorf("check %q repeats alert name %q", checkName, name)
		}
		seen[name] = struct{}{}
		alerts = append(alerts, map[string]any{
			"name": name, "period": period, "runbookUrl": runbookURL, "threshold": threshold,
		})
	}
	return alerts, nil
}

func syntheticMonitoringObservedCheckID(observed map[resource.Name]resource.ObservedComposed, resourceName resource.Name) (float64, bool) {
	id := observedString(observed, resourceName, "status.atProvider.id")
	if id == "" {
		return 0, false
	}
	parsed, err := strconv.ParseUint(id, 10, 53)
	if err != nil || parsed == 0 {
		return 0, false
	}
	return float64(parsed), true
}

func syntheticMonitoringInstallationVerified(observed map[resource.Name]resource.ObservedComposed, stackID string) bool {
	derivedToken := observedString(observed, syntheticMonitoringInstallationName, "status.atProvider.smAccessToken")
	verifierName := syntheticMonitoringVerifierResourceName(derivedToken)
	return observedString(observed, syntheticMonitoringInstallationName, "status.atProvider.stackId") == stackID &&
		derivedToken != "" &&
		observedString(observed, syntheticMonitoringInstallationName, "status.atProvider.stackSmApiUrl") != "" &&
		observedReady(observed, verifierName) &&
		observedIntegerString(observed, verifierName, "status.atProvider.tenantId") != ""
}

func syntheticMonitoringVerifierResourceName(derivedToken string) resource.Name {
	digest := sha256.Sum256([]byte(derivedToken))
	return resource.Name(syntheticMonitoringVerifierPrefix + fmt.Sprintf("%x", digest[:6]))
}

func desiredSyntheticMonitoringStatus(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) *resource.Composite {
	status := map[string]any{"installationVerified": false}
	stack, ok := config["referencedStack"].(map[string]any)
	if ok {
		stackID, _ := stack["stackID"].(string)
		usage, _ := stack["usage"].(string)
		budget, budgetErr := configuredSyntheticMonitoringBudget(config, usage)
		spec, _ := xr["spec"].(map[string]any)
		checks, weighted, checksErr := syntheticMonitoringChecks(spec, budget)
		if budgetErr == nil && checksErr == nil {
			apiCount, browserCount := 0, 0
			for _, check := range checks {
				if check.kind == "browser" {
					browserCount++
				} else {
					apiCount++
				}
			}
			status["budget"] = map[string]any{
				"apiChecks": apiCount, "browserChecks": browserCount,
				"weightedExecutionsPerHour":    weighted,
				"maxWeightedExecutionsPerHour": budget.maxWeightedExecutionsHour,
			}
		}
		status["installationVerified"] = syntheticMonitoringInstallationVerified(observed, stackID)
	}
	r := composite.New()
	r.SetUnstructuredContent(map[string]any{"status": status})
	return &resource.Composite{Resource: r}
}

func syntheticMonitoringPushSecret(namespace, name, sourceSecret, documentKey, document, remotePath string, settings platformSettings) *resource.DesiredComposed {
	return newDesired(
		"external-secrets.io/v1alpha1", "PushSecret", namespace, name, nil,
		map[string]any{
			"refreshInterval": "1h", "updatePolicy": "Replace", "deletionPolicy": "None",
			"secretStoreRefs": []any{settings.secretStoreReference()},
			"selector":        map[string]any{"secret": map[string]any{"name": sourceSecret}},
			"template": map[string]any{
				"engineVersion": "v2", "mergePolicy": "Replace",
				"data": map[string]any{documentKey: document},
			},
			"data": []any{map[string]any{
				"match": map[string]any{"secretKey": documentKey, "remoteRef": map[string]any{"remoteKey": remotePath}},
				"metadata": map[string]any{
					"apiVersion": "kubernetes.external-secrets.io/v1alpha1", "kind": "PushSecretMetadata",
					"spec": map[string]any{"secretPushFormat": "string", "tags": map[string]any{"grafana-cloud-vending-machine": "managed"}},
				},
			}},
		},
	)
}

func syntheticMonitoringSecret(namespace, name string, values map[string][]byte) *resource.DesiredComposed {
	data := make(map[string]any, len(values))
	for key, value := range values {
		data[key] = base64.StdEncoding.EncodeToString(value)
	}
	r := composed.New()
	r.SetUnstructuredContent(map[string]any{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata":   map[string]any{"name": name, "namespace": namespace},
		"type":       "Opaque",
		"data":       data,
	})
	return &resource.DesiredComposed{Resource: r}
}

func syntheticMonitoringObservedSecretMatches(observed map[resource.Name]resource.ObservedComposed, resourceName resource.Name, namespace, name string, values map[string][]byte) bool {
	r, ok := observed[resourceName]
	if !ok || r.Resource == nil {
		return false
	}
	object := r.Resource.UnstructuredContent()
	metadata, _ := object["metadata"].(map[string]any)
	data, _ := object["data"].(map[string]any)
	if object["apiVersion"] != "v1" || object["kind"] != "Secret" || metadata["namespace"] != namespace || metadata["name"] != name || len(data) != len(values) {
		return false
	}
	for key, value := range values {
		if data[key] != base64.StdEncoding.EncodeToString(value) {
			return false
		}
	}
	return true
}

func syntheticMonitoringInteger(value any) int {
	switch value := value.(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

func syntheticMonitoringNumber(value any) (float64, bool) {
	switch value := value.(type) {
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	case float64:
		return value, true
	default:
		return 0, false
	}
}
