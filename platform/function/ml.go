package main

import (
	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

const mlRendererImplemented = true

const mlAPIVersion = "ml.grafana.m.crossplane.io/v1alpha1"

// renderML vends a platform-owned ML profile. Jobs and outlier detectors are
// continuously running resources; the profile's maxRunningResources
// bounds them before any resource is rendered. Holiday IDs are provider assigned,
// so jobs that reference a holiday wait for its observed ID instead of deriving one.
func renderML(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	metadata, _ := xr["metadata"].(map[string]any)
	spec, _ := xr["spec"].(map[string]any)
	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	stackRef, _ := spec["stackRef"].(map[string]any)
	stackName, _ := stackRef["name"].(string)
	profileName, _ := spec["profile"].(string)
	if name == "" || namespace == "" || stackName == "" || profileName == "" {
		return nil, errors.New("ML must set metadata name and namespace, stackRef.name, and profile")
	}
	if name != stackName {
		return nil, errors.New("metadata.name must match spec.stackRef.name so one composite owns this stack surface")
	}
	if _, ready := referencedProductStack(config); !ready {
		return map[resource.Name]*resource.DesiredComposed{}, nil
	}

	profile, ok := configuredProfile(config, "mlProfiles", profileName)
	if !ok {
		return nil, errors.Errorf("ML profile %q is not configured by the platform", profileName)
	}
	jobs, _ := profile["jobs"].([]any)
	detectors, _ := profile["outlierDetectors"].([]any)
	maximum, ok := profile["maxRunningResources"].(float64)
	if !ok || maximum < 1 || maximum != float64(int(maximum)) {
		return nil, errors.Errorf("ML profile %q must set a positive integer maxRunningResources", profileName)
	}
	if len(jobs)+len(detectors) > int(maximum) {
		return nil, errors.Errorf("ML profile %q exceeds maxRunningResources: %d jobs and outlier detectors exceed cap %d", profileName, len(jobs)+len(detectors), int(maximum))
	}

	desired := map[resource.Name]*resource.DesiredComposed{}
	if err := renderMLHolidays(desired, profile, namespace, name, stackName); err != nil {
		return nil, err
	}
	if err := renderMLJobs(desired, observed, jobs, namespace, name, stackName); err != nil {
		return nil, err
	}
	if err := renderMLOutlierDetectors(desired, detectors, namespace, name, stackName); err != nil {
		return nil, err
	}
	for key := range observed {
		if _, retained := desired[key]; !retained {
			return nil, errors.Errorf("ML prerequisite or profile change would withdraw existing child %q; retain it until an explicit decommission", key)
		}
	}
	return desired, nil
}

func renderMLHolidays(desired map[resource.Name]*resource.DesiredComposed, profile map[string]any, namespace, claimName, stackName string) error {
	holidays, _ := profile["holidays"].([]any)
	for index, raw := range holidays {
		holiday, ok := raw.(map[string]any)
		if !ok {
			return errors.Errorf("ML holiday %d must be an object", index)
		}
		holidayName := stringValue(holiday, "name", "")
		periods, _ := holiday["customPeriods"].([]any)
		if holidayName == "" || len(periods) == 0 {
			return errors.Errorf("ML holiday %d must set name and at least one customPeriods entry", index)
		}
		resourceName := resource.Name("holiday-" + holidayName)
		if _, duplicate := desired[resourceName]; duplicate {
			return errors.Errorf("ML profile contains duplicate holiday name %q", holidayName)
		}
		desired[resourceName] = newDesired(mlAPIVersion, "Holiday", namespace, claimName+"-"+string(resourceName), nil, map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider": map[string]any{
				"name": holidayName, "description": stringValue(holiday, "description", ""), "customPeriods": periods,
			},
			"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": stackName},
		})
	}
	return nil
}

func renderMLJobs(desired map[resource.Name]*resource.DesiredComposed, observed map[resource.Name]resource.ObservedComposed, jobs []any, namespace, claimName, stackName string) error {
	declaredNames := map[string]struct{}{}
	for index, raw := range jobs {
		job, ok := raw.(map[string]any)
		if !ok {
			return errors.Errorf("ML job %d must be an object", index)
		}
		if err := validateMLQueryResource("job", index, job); err != nil {
			return err
		}
		jobName := stringValue(job, "name", "")
		if _, duplicate := declaredNames[jobName]; duplicate {
			return errors.Errorf("ML profile contains duplicate job name %q", jobName)
		}
		declaredNames[jobName] = struct{}{}
		holidays, ready, err := observedMLHolidayIDs(job, observed)
		if err != nil {
			return err
		}
		if !ready {
			continue
		}
		resourceName := resource.Name("job-" + jobName)
		if _, duplicate := desired[resourceName]; duplicate {
			return errors.Errorf("ML profile contains duplicate job name %q", jobName)
		}
		forProvider := mlQueryResourceParameters(job)
		forProvider["holidays"] = holidays
		desired[resourceName] = newDesired(mlAPIVersion, "Job", namespace, claimName+"-"+string(resourceName), nil, map[string]any{
			"managementPolicies": managementPolicies, "forProvider": forProvider,
			"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": stackName},
		})
	}
	return nil
}

func renderMLOutlierDetectors(desired map[resource.Name]*resource.DesiredComposed, detectors []any, namespace, claimName, stackName string) error {
	for index, raw := range detectors {
		detector, ok := raw.(map[string]any)
		if !ok {
			return errors.Errorf("ML outlier detector %d must be an object", index)
		}
		if err := validateMLQueryResource("outlier detector", index, detector); err != nil {
			return err
		}
		algorithm, _ := detector["algorithm"].(map[string]any)
		if stringValue(algorithm, "name", "") == "" {
			return errors.Errorf("ML outlier detector %d must set algorithm.name", index)
		}
		detectorName := stringValue(detector, "name", "")
		resourceName := resource.Name("outlier-detector-" + detectorName)
		if _, duplicate := desired[resourceName]; duplicate {
			return errors.Errorf("ML profile contains duplicate outlier detector name %q", detectorName)
		}
		forProvider := mlQueryResourceParameters(detector)
		forProvider["algorithm"] = []any{algorithm}
		desired[resourceName] = newDesired(mlAPIVersion, "OutlierDetector", namespace, claimName+"-"+string(resourceName), nil, map[string]any{
			"managementPolicies": managementPolicies, "forProvider": forProvider,
			"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": stackName},
		})
	}
	return nil
}

func validateMLQueryResource(kind string, index int, value map[string]any) error {
	for _, field := range []string{"name", "datasourceType", "metric"} {
		if stringValue(value, field, "") == "" {
			return errors.Errorf("ML %s %d must set %s", kind, index, field)
		}
	}
	queryParams, _ := value["queryParams"].(map[string]any)
	if len(queryParams) == 0 {
		return errors.Errorf("ML %s %d must set queryParams", kind, index)
	}
	return nil
}

func mlQueryResourceParameters(value map[string]any) map[string]any {
	parameters := map[string]any{}
	for _, field := range []string{"name", "datasourceType", "datasourceUid", "description", "metric", "queryParams", "interval", "trainingWindow", "customLabels", "hyperParams"} {
		if raw, ok := value[field]; ok {
			parameters[field] = raw
		}
	}
	return parameters
}

func observedMLHolidayIDs(job map[string]any, observed map[resource.Name]resource.ObservedComposed) ([]any, bool, error) {
	names, _ := job["holidayRefs"].([]any)
	ids := make([]any, 0, len(names))
	seen := map[string]struct{}{}
	for index, raw := range names {
		name, ok := raw.(string)
		if !ok || name == "" {
			return nil, false, errors.Errorf("ML job holidayRefs[%d] must be a non-empty string", index)
		}
		if _, duplicate := seen[name]; duplicate {
			return nil, false, errors.Errorf("ML job contains duplicate holiday reference %q", name)
		}
		seen[name] = struct{}{}
		id := observedString(observed, resource.Name("holiday-"+name), "status.atProvider.id")
		if id == "" {
			return nil, false, nil
		}
		ids = append(ids, id)
	}
	return ids, true, nil
}
