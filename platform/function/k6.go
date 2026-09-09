package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

const k6RendererImplemented = true

const resolvedK6StackConfigKey = "_resolvedK6Stack"

const k6APIVersion = "k6.grafana.m.crossplane.io/v1alpha1"

// k6DynamicManagementPolicies deliberately include Delete. Unlike the
// project and its policy resources, a load test or schedule is owned only by
// this composite. It must be removed from the provider when the request is
// removed, otherwise a schedule could continue to run against an orphaned
// project.
var k6DynamicManagementPolicies = []any{"Create", "Observe", "Update", "Delete", "LateInitialize"}

// renderK6Project creates the bounded k6 project surface. It deliberately
// waits for trusted stack context because the bootstrap token belongs to the
// referenced stack and must never be accepted or stored on this request.
func renderK6Project(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	stack, ready := resolvedK6Stack(config)
	if !ready {
		return map[resource.Name]*resource.DesiredComposed{}, nil
	}

	metadata, _ := xr["metadata"].(map[string]any)
	spec, _ := xr["spec"].(map[string]any)
	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	stackRef, _ := spec["stackRef"].(map[string]any)
	stackName, _ := stackRef["name"].(string)
	grafanaUser, _ := spec["grafanaUser"].(string)
	if name == "" || namespace == "" || stackName == "" || grafanaUser == "" {
		return nil, errors.New("k6 project must set metadata name and namespace plus stackRef.name and grafanaUser")
	}

	usage, _ := stack["usage"].(string)
	stackID, _ := stack["stackId"].(string)
	outputSecretPath, _ := stack["outputSecretPath"].(string)
	organizationProviderConfigName, _ := stack["organizationProviderConfigName"].(string)
	bootstrapSecret, _ := stack["serviceAccountTokenSecret"].(map[string]any)
	bootstrapName, _ := bootstrapSecret["name"].(string)
	bootstrapKey, _ := bootstrapSecret["key"].(string)
	bootstrapReady, _ := bootstrapSecret["ready"].(bool)
	if usage == "" || stackID == "" || outputSecretPath == "" || organizationProviderConfigName == "" || bootstrapName == "" || bootstrapKey == "" || !bootstrapReady {
		return map[resource.Name]*resource.DesiredComposed{}, nil
	}

	limits, err := configuredK6LimitProfile(config, usage)
	if err != nil {
		return nil, err
	}
	allowedLoadZones, err := requestedK6LoadZones(spec, limits.allowedLoadZones)
	if err != nil {
		return nil, err
	}
	loadTests, err := configuredK6LoadTests(spec, limits, allowedLoadZones)
	if err != nil {
		return nil, err
	}
	schedules, err := configuredK6Schedules(spec, loadTests)
	if err != nil {
		return nil, err
	}
	if len(loadTests) > 0 || len(schedules) > 0 {
		declaredUsage, _ := spec["usage"].(string)
		if declaredUsage == "" {
			return nil, errors.New("spec.usage is required when loadTests or schedules are requested")
		}
		if declaredUsage != usage {
			return nil, errors.Errorf("spec.usage %q does not match trusted referenced stack usage %q", declaredUsage, usage)
		}
	}

	desired := map[resource.Name]*resource.DesiredComposed{}
	desired["installation"] = newDesired(
		k6APIVersion,
		"Installation",
		namespace,
		name+"-installation",
		nil,
		map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider": map[string]any{
				"grafanaSaTokenSecretRef": map[string]any{"name": bootstrapName, "key": bootstrapKey},
				"grafanaUser":             grafanaUser,
				"stackId":                 stackID,
			},
			"providerConfigRef":          k6ProviderConfigReference(organizationProviderConfigName),
			"writeConnectionSecretToRef": map[string]any{"name": name + "-k6-token"},
		},
	)

	// The installation creates the derived k6 token. Keep the project tree out
	// of desired state until the provider has observed that bootstrap complete.
	if !observedReady(observed, "installation") {
		return desired, nil
	}

	derivedSecret := name + "-k6-token"
	providerCredentialsSecret := name + "-k6-provider-credentials"
	providerConfigName := name + "-k6"
	outputDocument := `{{ $token := index . "attribute.k6_access_token" | toString }}{"k6_access_token":{{ $token | toJson }}}`
	settings := configuredPlatformSettings(config)
	desired["credentials"] = newDesired(
		"external-secrets.io/v1alpha1",
		"PushSecret",
		namespace,
		name+"-k6-credentials",
		nil,
		map[string]any{
			"refreshInterval": "1h", "updatePolicy": "Replace", "deletionPolicy": "None",
			"secretStoreRefs": []any{settings.secretStoreReference()},
			"selector":        map[string]any{"secret": map[string]any{"name": derivedSecret}},
			"template": map[string]any{
				"engineVersion": "v2", "mergePolicy": "Replace", "data": map[string]any{"k6.json": outputDocument},
			},
			"data": []any{map[string]any{
				"match": map[string]any{"secretKey": "k6.json", "remoteRef": map[string]any{"remoteKey": outputSecretPath + "/k6"}},
				"metadata": map[string]any{
					"apiVersion": "kubernetes.external-secrets.io/v1alpha1", "kind": "PushSecretMetadata",
					"spec": map[string]any{"secretPushFormat": "string", "tags": map[string]any{"grafana-cloud-vending-machine": "managed"}},
				},
			}},
		},
	)
	desired["provider-credentials"] = newDesired(
		"external-secrets.io/v1",
		"ExternalSecret",
		namespace,
		providerCredentialsSecret,
		nil,
		map[string]any{
			"refreshInterval": "1h",
			"secretStoreRef":  settings.secretStoreReference(),
			"target": map[string]any{
				"name": providerCredentialsSecret, "creationPolicy": "Owner", "deletionPolicy": "Retain",
				"template": map[string]any{
					"engineVersion": "v2",
					"data":          map[string]any{"credentials": `{"k6_access_token":{{ .k6AccessToken | toJson }}}`},
				},
			},
			"data": []any{map[string]any{
				"secretKey": "k6AccessToken",
				"remoteRef": map[string]any{"key": outputSecretPath + "/k6", "property": "k6_access_token"},
			}},
		},
	)
	desired["provider-config"] = newDesired(
		"grafana.m.crossplane.io/v1beta1",
		"ProviderConfig",
		namespace,
		providerConfigName,
		nil,
		map[string]any{
			"credentials": map[string]any{
				"source": "Secret",
				"secretRef": map[string]any{
					"name": providerCredentialsSecret, "namespace": namespace, "key": "credentials",
				},
			},
		},
	)

	if !observedReady(observed, "provider-credentials") {
		return desired, nil
	}

	desired["project"] = newDesired(
		k6APIVersion,
		"Project",
		namespace,
		name,
		nil,
		map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider":        map[string]any{"name": name},
			"providerConfigRef":  k6ProviderConfigReference(providerConfigName),
		},
	)
	projectID := observedString(observed, "project", "status.atProvider.id")
	if projectID == "" {
		return desired, nil
	}

	desired["limits"] = newDesired(
		k6APIVersion,
		"ProjectLimits",
		namespace,
		name+"-limits",
		map[string]any{"crossplane.io/external-name": projectID},
		map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider": map[string]any{
				"projectId":           projectID,
				"vuhMaxPerMonth":      limits.vuhMaxPerMonth,
				"vuMaxPerTest":        limits.vuMaxPerTest,
				"vuBrowserMaxPerTest": limits.vuBrowserMaxPerTest,
				"durationMaxPerTest":  limits.durationMaxPerTest,
			},
			"providerConfigRef": k6ProviderConfigReference(providerConfigName),
		},
	)

	desired["allowed-load-zones"] = newDesired(
		k6APIVersion,
		"ProjectAllowedLoadZones",
		namespace,
		name+"-allowed-load-zones",
		map[string]any{"crossplane.io/external-name": projectID},
		map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider": map[string]any{
				"projectId": projectID, "allowedLoadZones": allowedLoadZones,
			},
			"providerConfigRef": k6ProviderConfigReference(providerConfigName),
		},
	)

	// ProjectLimits and ProjectAllowedLoadZones are the enforcement boundary for
	// the dynamic children. Do not let either child be created in the same
	// reconciliation that first creates the cap: the provider might otherwise
	// accept a test before the cap has reached Grafana Cloud.
	if len(loadTests) > 0 || len(schedules) > 0 {
		if !observedDesiredCurrent(observed["limits"], desired["limits"]) || !observedDesiredCurrent(observed["allowed-load-zones"], desired["allowed-load-zones"]) {
			return desired, nil
		}
	}

	loadTestIDs := map[string]string{}
	for _, test := range loadTests {
		logicalName := resource.Name("load-test-" + test.name)
		parameters := map[string]any{
			"projectId": projectID,
			"name":      test.name,
			"script":    test.script,
		}
		if test.k6Version != "" {
			parameters["k6Version"] = test.k6Version
		}
		desired[logicalName] = newDesired(
			k6APIVersion,
			"LoadTest",
			namespace,
			name+"-load-test-"+test.name,
			k6ObservedExternalName(observed, logicalName),
			map[string]any{
				"managementPolicies": k6DynamicManagementPolicies,
				"forProvider":        parameters,
				"providerConfigRef":  k6ProviderConfigReference(providerConfigName),
			},
		)
		if id := observedString(observed, logicalName, "status.atProvider.id"); id != "" {
			loadTestIDs[test.name] = id
		}
	}

	for _, schedule := range schedules {
		logicalName := resource.Name("schedule-" + schedule.name)
		loadTestID := loadTestIDs[schedule.loadTest]
		if loadTestID == "" {
			// LoadTest IDs are assigned by Grafana Cloud. A schedule without the
			// observed ID would either fail provider validation or target an
			// unrelated test, so hold it until the ID is real.
			continue
		}
		parameters := map[string]any{
			"loadTestId": loadTestID,
			"starts":     schedule.starts,
		}
		if schedule.cron != nil {
			parameters["cron"] = schedule.cron
		}
		if schedule.recurrenceRule != nil {
			parameters["recurrenceRule"] = schedule.recurrenceRule
		}
		desired[logicalName] = newDesired(
			k6APIVersion,
			"Schedule",
			namespace,
			name+"-schedule-"+schedule.name,
			k6ObservedExternalName(observed, logicalName),
			map[string]any{
				"managementPolicies": k6DynamicManagementPolicies,
				"forProvider":        parameters,
				"providerConfigRef":  k6ProviderConfigReference(providerConfigName),
			},
		)
	}

	return desired, nil
}

type k6LoadTest struct {
	name            string
	script          string
	workloadURL     string
	k6Version       string
	vus             float64
	browserVUs      float64
	durationSeconds float64
	loadZones       []string
}

type k6Schedule struct {
	name           string
	loadTest       string
	starts         string
	cron           map[string]any
	recurrenceRule map[string]any
}

func configuredK6WorkloadURL(item map[string]any, index int) (string, error) {
	rawWorkload, ok := item["workload"]
	if !ok || rawWorkload == nil {
		return "", errors.Errorf("loadTests[%d].workload must be supplied", index)
	}
	workload, ok := rawWorkload.(map[string]any)
	if !ok {
		return "", errors.Errorf("loadTests[%d].workload must be an object", index)
	}
	rawHTTPGet, ok := workload["httpGet"]
	if !ok || rawHTTPGet == nil {
		return "", errors.Errorf("loadTests[%d].workload.httpGet must be supplied", index)
	}
	httpGet, ok := rawHTTPGet.(map[string]any)
	if !ok {
		return "", errors.Errorf("loadTests[%d].workload.httpGet must be an object", index)
	}
	rawURL, ok := httpGet["url"]
	if !ok {
		return "", errors.Errorf("loadTests[%d].workload.httpGet.url must be supplied", index)
	}
	workloadURL, ok := rawURL.(string)
	if !ok || workloadURL == "" || workloadURL != strings.TrimSpace(workloadURL) {
		return "", errors.Errorf("loadTests[%d].workload.httpGet.url must be a non-empty HTTPS URL without credentials", index)
	}
	parsed, err := url.Parse(workloadURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.Hostname() == "" || parsed.Opaque != "" || parsed.User != nil || strings.ContainsAny(workloadURL, "\r\n\t") {
		return "", errors.Errorf("loadTests[%d].workload.httpGet.url must be a non-empty HTTPS URL without credentials", index)
	}
	return workloadURL, nil
}

func generatedK6Script(test k6LoadTest) string {
	var script strings.Builder
	script.WriteString("import http from \"k6/http\";\n\n")
	script.WriteString("export const options = {\n")
	script.WriteString("  scenarios: {\n    default: {\n      executor: \"constant-vus\",\n")
	fmt.Fprintf(&script, "      vus: %s,\n", strconv.FormatInt(int64(test.vus), 10))
	fmt.Fprintf(&script, "      duration: %s,\n", k6JSONString(strconv.FormatInt(int64(test.durationSeconds), 10)+"s"))
	script.WriteString("      gracefulStop: \"0s\",\n    },\n  },\n")
	if len(test.loadZones) > 0 {
		script.WriteString("  cloud: {\n    distribution: {\n")
		for index, zone := range test.loadZones {
			percent := k6DistributionPercent(len(test.loadZones), index)
			quotedZone := k6JSONString(zone)
			fmt.Fprintf(&script, "      %s: { loadZone: %s, percent: %d },\n", quotedZone, quotedZone, percent)
		}
		script.WriteString("    },\n  },\n")
	}
	script.WriteString("};\n\n")
	script.WriteString("export default function () {\n")
	fmt.Fprintf(&script, "  http.get(%s);\n", k6JSONString(test.workloadURL))
	script.WriteString("}\n")
	return script.String()
}

func k6JSONString(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func k6DistributionPercent(zoneCount, index int) int {
	base := 100 / zoneCount
	if index < 100%zoneCount {
		return base + 1
	}
	return base
}

func k6WholeNumber(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && math.Trunc(value) == value
}

func configuredK6LoadTests(spec map[string]any, limits k6LimitProfile, allowedLoadZones []any) ([]k6LoadTest, error) {
	rawTests, exists := spec["loadTests"]
	if !exists {
		return nil, nil
	}
	rawItems, ok := rawTests.([]any)
	if !ok {
		return nil, errors.New("loadTests must be an array")
	}
	allowed := make(map[string]struct{}, len(allowedLoadZones))
	for _, value := range allowedLoadZones {
		zone, _ := value.(string)
		allowed[zone] = struct{}{}
	}
	seen := map[string]struct{}{}
	result := make([]k6LoadTest, 0, len(rawItems))
	for index, raw := range rawItems {
		item, ok := raw.(map[string]any)
		if !ok {
			return nil, errors.Errorf("loadTests[%d] must be an object", index)
		}
		name, _ := item["name"].(string)
		if !validK6ChildName(name) {
			return nil, errors.Errorf("loadTests[%d].name must be a DNS-compatible non-empty name", index)
		}
		if _, exists := seen[name]; exists {
			return nil, errors.Errorf("loadTests contains duplicate name %q", name)
		}
		seen[name] = struct{}{}
		if rawScript, exists := item["script"]; exists && rawScript != nil {
			return nil, errors.Errorf("loadTests[%d].script is unsupported; use workload.httpGet.url", index)
		}
		workloadURL, err := configuredK6WorkloadURL(item, index)
		if err != nil {
			return nil, err
		}
		vus, err := requiredK6PositiveNumber(item, "vus", index)
		if err != nil {
			return nil, err
		}
		if !k6WholeNumber(vus) {
			return nil, errors.Errorf("loadTests[%d].vus must be an integer", index)
		}
		browserVUs, err := optionalK6NonNegativeNumber(item, "browserVus", index)
		if err != nil {
			return nil, err
		}
		if !k6WholeNumber(browserVUs) {
			return nil, errors.Errorf("loadTests[%d].browserVus must be an integer", index)
		}
		if browserVUs > 0 {
			return nil, errors.Errorf("loadTests[%d].browserVus is unsupported; generated browser workloads are not available", index)
		}
		duration, err := requiredK6PositiveNumber(item, "durationSeconds", index)
		if err != nil {
			return nil, err
		}
		if !k6WholeNumber(duration) {
			return nil, errors.Errorf("loadTests[%d].durationSeconds must be an integer", index)
		}
		if vus > limits.vuMaxPerTest {
			return nil, errors.Errorf("load test %q requests %.0f VUs; platform maximum is %.0f", name, vus, limits.vuMaxPerTest)
		}
		if browserVUs > limits.vuBrowserMaxPerTest {
			return nil, errors.Errorf("load test %q requests %.0f browser VUs; platform maximum is %.0f", name, browserVUs, limits.vuBrowserMaxPerTest)
		}
		if duration > limits.durationMaxPerTest {
			return nil, errors.Errorf("load test %q requests %.0f seconds; platform maximum is %.0f", name, duration, limits.durationMaxPerTest)
		}
		rawZones, ok := item["loadZones"]
		if !ok || rawZones == nil {
			return nil, errors.Errorf("loadTests[%d].loadZones must be explicitly supplied as an array", index)
		}
		zones, ok := rawZones.([]any)
		if !ok {
			return nil, errors.Errorf("loadTests[%d].loadZones must be an array", index)
		}
		seenZones := map[string]struct{}{}
		cleanZones := make([]string, 0, len(zones))
		for zoneIndex, rawZone := range zones {
			zone, ok := rawZone.(string)
			if !ok || strings.TrimSpace(zone) == "" {
				return nil, errors.Errorf("loadTests[%d].loadZones[%d] must be a non-empty string", index, zoneIndex)
			}
			if _, exists := seenZones[zone]; exists {
				return nil, errors.Errorf("load test %q repeats load zone %q", name, zone)
			}
			if _, exists := allowed[zone]; !exists {
				return nil, errors.Errorf("load test %q uses load zone %q outside the project's allowed load zones", name, zone)
			}
			seenZones[zone] = struct{}{}
			cleanZones = append(cleanZones, zone)
		}
		sort.Strings(cleanZones)
		k6Version, _ := item["k6Version"].(string)
		if k6Version != "" && strings.TrimSpace(k6Version) == "" {
			return nil, errors.Errorf("loadTests[%d].k6Version must be a non-empty string when supplied", index)
		}
		if k6Version != strings.TrimSpace(k6Version) {
			return nil, errors.Errorf("loadTests[%d].k6Version must not contain leading or trailing whitespace", index)
		}
		test := k6LoadTest{
			name:            name,
			k6Version:       k6Version,
			vus:             vus,
			browserVUs:      browserVUs,
			durationSeconds: duration,
			workloadURL:     workloadURL,
			loadZones:       cleanZones,
		}
		test.script = generatedK6Script(test)
		result = append(result, test)
	}
	return result, nil
}

func configuredK6Schedules(spec map[string]any, tests []k6LoadTest) ([]k6Schedule, error) {
	rawSchedules, exists := spec["schedules"]
	if !exists {
		return nil, nil
	}
	rawItems, ok := rawSchedules.([]any)
	if !ok {
		return nil, errors.New("schedules must be an array")
	}
	testNames := make(map[string]struct{}, len(tests))
	for _, test := range tests {
		testNames[test.name] = struct{}{}
	}
	seenNames := map[string]struct{}{}
	seenTests := map[string]struct{}{}
	result := make([]k6Schedule, 0, len(rawItems))
	for index, raw := range rawItems {
		item, ok := raw.(map[string]any)
		if !ok {
			return nil, errors.Errorf("schedules[%d] must be an object", index)
		}
		name, _ := item["name"].(string)
		if !validK6ChildName(name) {
			return nil, errors.Errorf("schedules[%d].name must be a DNS-compatible non-empty name", index)
		}
		if _, exists := seenNames[name]; exists {
			return nil, errors.Errorf("schedules contains duplicate name %q", name)
		}
		seenNames[name] = struct{}{}
		loadTest, _ := item["loadTest"].(string)
		if _, exists := testNames[loadTest]; !exists {
			return nil, errors.Errorf("schedule %q references undeclared load test %q", name, loadTest)
		}
		if _, exists := seenTests[loadTest]; exists {
			return nil, errors.Errorf("schedules target load test %q more than once", loadTest)
		}
		seenTests[loadTest] = struct{}{}
		starts, _ := item["starts"].(string)
		if _, err := time.Parse(time.RFC3339, starts); err != nil {
			return nil, errors.Errorf("schedule %q starts must be RFC3339: %v", name, err)
		}
		schedule := k6Schedule{name: name, loadTest: loadTest, starts: starts}
		if rawCron, present := item["cron"]; present {
			if rawCron == nil {
				return nil, errors.Errorf("schedule %q cron cannot be null", name)
			}
			cron, err := k6ScheduleObject(rawCron, "cron", name)
			if err != nil {
				return nil, err
			}
			cronSchedule, _ := cron["schedule"].(string)
			timezone, _ := cron["timezone"].(string)
			if strings.TrimSpace(cronSchedule) == "" || strings.TrimSpace(timezone) == "" {
				return nil, errors.Errorf("schedule %q cron must set schedule and timezone", name)
			}
			schedule.cron = cron
		}
		if rawRule, present := item["recurrenceRule"]; present {
			if schedule.cron != nil {
				return nil, errors.Errorf("schedule %q cannot set both cron and recurrenceRule", name)
			}
			if rawRule == nil {
				return nil, errors.Errorf("schedule %q recurrenceRule cannot be null", name)
			}
			rule, err := k6ScheduleObject(rawRule, "recurrenceRule", name)
			if err != nil {
				return nil, err
			}
			frequency, _ := rule["frequency"].(string)
			if !oneOf(frequency, "HOURLY", "DAILY", "WEEKLY", "MONTHLY", "YEARLY") {
				return nil, errors.Errorf("schedule %q recurrenceRule.frequency must be HOURLY, DAILY, WEEKLY, MONTHLY, or YEARLY", name)
			}
			if interval, present := rule["interval"]; present {
				value, valid := k6Number(interval)
				if !valid || value <= 0 {
					return nil, errors.Errorf("schedule %q recurrenceRule.interval must be positive", name)
				}
			}
			if count, present := rule["count"]; present {
				value, valid := k6Number(count)
				if !valid || value <= 0 {
					return nil, errors.Errorf("schedule %q recurrenceRule.count must be positive", name)
				}
			}
			if until, present := rule["until"]; present {
				untilTime, valid := until.(string)
				if !valid {
					return nil, errors.Errorf("schedule %q recurrenceRule.until must be RFC3339", name)
				}
				if _, err := time.Parse(time.RFC3339, untilTime); err != nil {
					return nil, errors.Errorf("schedule %q recurrenceRule.until must be RFC3339: %v", name, err)
				}
			}
			schedule.recurrenceRule = rule
		}
		result = append(result, schedule)
	}
	return result, nil
}

func k6ScheduleObject(value any, field, name string) (map[string]any, error) {
	object, ok := value.(map[string]any)
	if !ok {
		return nil, errors.Errorf("schedule %q %s must be an object", name, field)
	}
	copy := make(map[string]any, len(object))
	for key, item := range object {
		copy[key] = item
	}
	return copy, nil
}

func requiredK6PositiveNumber(values map[string]any, key string, index int) (float64, error) {
	value, present := values[key]
	if !present {
		return 0, errors.Errorf("loadTests[%d].%s must be supplied", index, key)
	}
	result, ok := k6Number(value)
	if !ok || result <= 0 {
		return 0, errors.Errorf("loadTests[%d].%s must be a positive number", index, key)
	}
	return result, nil
}

func optionalK6NonNegativeNumber(values map[string]any, key string, index int) (float64, error) {
	value, present := values[key]
	if !present {
		return 0, nil
	}
	result, ok := k6Number(value)
	if !ok || result < 0 {
		return 0, errors.Errorf("loadTests[%d].%s must be a non-negative number", index, key)
	}
	return result, nil
}

func k6Number(value any) (float64, bool) {
	switch number := value.(type) {
	case float64:
		return number, k6FiniteNumber(number)
	case float32:
		converted := float64(number)
		return converted, k6FiniteNumber(converted)
	case int:
		return float64(number), true
	case int32:
		return float64(number), true
	case int64:
		return float64(number), true
	case uint:
		return float64(number), true
	case uint32:
		return float64(number), true
	case uint64:
		return float64(number), true
	default:
		return 0, false
	}
}

func k6FiniteNumber(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func validK6ChildName(value string) bool {
	if len(value) == 0 || len(value) > 63 {
		return false
	}
	for index, character := range []byte(value) {
		if (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') {
			continue
		}
		if character == '-' && index > 0 && index < len(value)-1 {
			continue
		}
		return false
	}
	return true
}

func k6ObservedExternalName(observed map[resource.Name]resource.ObservedComposed, name resource.Name) map[string]any {
	// The provider assigns numeric IDs for both resources. Carry an annotation
	// forward only after it has been observed; a friendly request name is not a
	// valid external ID and must never be guessed here.
	id := observedString(observed, name, "status.atProvider.id")
	if id == "" {
		return nil
	}
	return map[string]any{"crossplane.io/external-name": id}
}

type k6LimitProfile struct {
	usage               string
	vuhMaxPerMonth      float64
	vuMaxPerTest        float64
	vuBrowserMaxPerTest float64
	durationMaxPerTest  float64
	allowedLoadZones    []string
}

func resolvedK6Stack(config map[string]any) (map[string]any, bool) {
	stack, ok := config[resolvedK6StackConfigKey].(map[string]any)
	return stack, ok
}

func configuredK6LimitProfile(config map[string]any, usage string) (k6LimitProfile, error) {
	spec, _ := config["spec"].(map[string]any)
	profiles, _ := spec["k6LimitProfiles"].([]any)
	var matched *k6LimitProfile
	for index, raw := range profiles {
		profile, ok := raw.(map[string]any)
		if !ok {
			return k6LimitProfile{}, errors.Errorf("k6LimitProfiles[%d] must be an object", index)
		}
		profileUsage, _ := profile["usage"].(string)
		if profileUsage != usage {
			continue
		}
		if matched != nil {
			return k6LimitProfile{}, errors.Errorf("k6LimitProfiles contains multiple profiles for usage %q", usage)
		}
		candidate := k6LimitProfile{usage: usage}
		var valid bool
		if candidate.vuhMaxPerMonth, valid = k6PositiveNumber(profile, "vuhMaxPerMonth"); !valid {
			return k6LimitProfile{}, errors.Errorf("k6LimitProfiles[%d].vuhMaxPerMonth must be a positive number", index)
		}
		if candidate.vuMaxPerTest, valid = k6PositiveNumber(profile, "vuMaxPerTest"); !valid {
			return k6LimitProfile{}, errors.Errorf("k6LimitProfiles[%d].vuMaxPerTest must be a positive number", index)
		}
		if candidate.vuBrowserMaxPerTest, valid = k6PositiveNumber(profile, "vuBrowserMaxPerTest"); !valid {
			return k6LimitProfile{}, errors.Errorf("k6LimitProfiles[%d].vuBrowserMaxPerTest must be a positive number", index)
		}
		if candidate.durationMaxPerTest, valid = k6PositiveNumber(profile, "durationMaxPerTest"); !valid {
			return k6LimitProfile{}, errors.Errorf("k6LimitProfiles[%d].durationMaxPerTest must be a positive number", index)
		}
		candidate.allowedLoadZones = stringListValue(profile, "allowedLoadZones", nil)
		matched = &candidate
	}
	if matched == nil {
		return k6LimitProfile{}, errors.Errorf("no k6 limit profile is configured for referenced stack usage %q", usage)
	}
	return *matched, nil
}

func k6PositiveNumber(values map[string]any, key string) (float64, bool) {
	value, ok := values[key].(float64)
	return value, ok && value > 0
}

func k6ProviderConfigReference(name string) map[string]any {
	return map[string]any{"kind": "ProviderConfig", "name": name}
}

func requestedK6LoadZones(spec map[string]any, permitted []string) ([]any, error) {
	requested, ok := spec["allowedLoadZones"].([]any)
	if !ok {
		return nil, errors.New("allowedLoadZones must be explicitly supplied as an array; use an empty array to allow no private zones")
	}
	permittedSet := make(map[string]struct{}, len(permitted))
	for _, zone := range permitted {
		permittedSet[zone] = struct{}{}
	}
	result := make([]any, 0, len(requested))
	seen := map[string]struct{}{}
	for index, value := range requested {
		zone, ok := value.(string)
		if !ok || zone == "" {
			return nil, errors.Errorf("allowedLoadZones[%d] must be a non-empty string", index)
		}
		if _, duplicate := seen[zone]; duplicate {
			return nil, errors.Errorf("allowedLoadZones contains duplicate zone %q", zone)
		}
		if _, allowed := permittedSet[zone]; !allowed {
			return nil, errors.Errorf("allowedLoadZones[%d] %q is not permitted by the platform profile", index, zone)
		}
		seen[zone] = struct{}{}
		result = append(result, zone)
	}
	return result, nil
}
