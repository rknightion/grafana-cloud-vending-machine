package main

import (
	"fmt"
	"hash/fnv"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

const sloRendererImplemented = true

const (
	goldenSLOLogicalName           resource.Name = "golden-slo"
	goldenSLODatasourceLogicalName resource.Name = "golden-slo-datasource"
)

func addGoldenSLO(desired map[resource.Name]*resource.DesiredComposed, xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) error {
	metadata, _ := xr["metadata"].(map[string]any)
	spec, _ := xr["spec"].(map[string]any)
	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	slug, _ := spec["slug"].(string)
	usage, _ := spec["usage"].(string)
	if name == "" || namespace == "" || slug == "" || usage == "" {
		return errors.New("golden SLO requires stack metadata name and namespace plus slug and usage")
	}

	platformSpec, _ := config["spec"].(map[string]any)
	profilesValue, configured := platformSpec["goldenSLOProfiles"]
	if !configured {
		return nil
	}
	profiles, ok := profilesValue.(map[string]any)
	if !ok || len(profiles) == 0 {
		return errors.New("goldenSLOProfiles must be a non-empty object keyed by usage")
	}
	profileValue, selected := profiles[usage]
	if !selected {
		return nil
	}
	profile, ok := profileValue.(map[string]any)
	if !ok {
		return errors.Errorf("goldenSLOProfiles.%s must be an object", usage)
	}
	parameters, datasourceUID, err := goldenSLOParameters(profile, usage)
	if err != nil {
		return err
	}

	desired[goldenSLODatasourceLogicalName] = newDesired(
		"oss.grafana.m.crossplane.io/v1alpha1",
		"DataSource",
		namespace,
		slug+"-golden-slo-datasource",
		map[string]any{"crossplane.io/external-name": datasourceUID},
		map[string]any{
			"managementPolicies": []any{"Observe"},
			"forProvider":        map[string]any{},
			"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": slug},
		},
	)

	observedDatasourceUID := observedString(observed, goldenSLODatasourceLogicalName, "status.atProvider.uid")
	if observedDatasourceUID == "" {
		return nil
	}
	parameters["destinationDatasource"] = []any{map[string]any{"uid": observedDatasourceUID}}
	parameters["uuid"] = goldenSLOUUID(namespace, name, usage)
	desired[goldenSLOLogicalName] = newDesired(
		"slo.grafana.m.crossplane.io/v1alpha1",
		"SLO",
		namespace,
		slug+"-golden-slo",
		map[string]any{"crossplane.io/external-name": parameters["uuid"]},
		map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider":        map[string]any{},
			"initProvider":       parameters,
			"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": slug},
		},
	)
	return nil
}

func goldenSLOParameters(profile map[string]any, usage string) (map[string]any, string, error) {
	name, _ := profile["name"].(string)
	description, _ := profile["description"].(string)
	datasourceUID, _ := profile["datasourceUID"].(string)
	objective, ok := profile["objective"].(map[string]any)
	if name == "" || datasourceUID == "" || !ok {
		return nil, "", errors.Errorf("goldenSLOProfiles.%s must set name, datasourceUID, and objective", usage)
	}
	objectiveValue, ok := objective["value"].(float64)
	window, _ := objective["window"].(string)
	if !ok || objectiveValue <= 0 || objectiveValue >= 1 || window == "" {
		return nil, "", errors.Errorf("goldenSLOProfiles.%s.objective must set value strictly between 0 and 1 and a window", usage)
	}
	ratio, ok := profile["ratio"].(map[string]any)
	if !ok {
		return nil, "", errors.Errorf("goldenSLOProfiles.%s.ratio must be an object", usage)
	}
	successMetric, _ := ratio["successMetric"].(string)
	totalMetric, _ := ratio["totalMetric"].(string)
	if successMetric == "" || totalMetric == "" {
		return nil, "", errors.Errorf("goldenSLOProfiles.%s.ratio must set successMetric and totalMetric", usage)
	}

	parameters := map[string]any{
		"name":       name,
		"objectives": []any{map[string]any{"value": objectiveValue, "window": window}},
		"query":      []any{map[string]any{"type": "ratio", "ratio": []any{map[string]any{"successMetric": successMetric, "totalMetric": totalMetric}}}},
		"alerting":   []any{map[string]any{"fastburn": []any{map[string]any{}}, "slowburn": []any{map[string]any{}}}},
	}
	if description != "" {
		parameters["description"] = description
	}
	return parameters, datasourceUID, nil
}

func goldenSLOUUID(namespace, name, usage string) string {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(namespace + "/" + name + "/" + usage))
	value := hash.Sum64()
	return fmt.Sprintf("00000000-0000-5000-8000-%012x", value&0xffffffffffff)
}
