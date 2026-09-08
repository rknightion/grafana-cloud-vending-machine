package main

import (
	"strings"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

const retentionRendererImplemented = true

type retentionFanoutProfile struct {
	configType string
	contents   string
	matchers   []any
}

func addRetentionFanout(desired map[resource.Name]*resource.DesiredComposed, xr map[string]any, _ map[resource.Name]resource.ObservedComposed, config map[string]any) error {
	spec, _ := xr["spec"].(map[string]any)
	rawRetention, requested := spec["retention"]
	if !requested || rawRetention == nil {
		return nil
	}

	retention, ok := rawRetention.(map[string]any)
	if !ok {
		return errors.New("spec.retention must be an object")
	}
	for field := range retention {
		if field != "class" {
			return errors.Errorf("retention supports only class; retention periods are not reconciled")
		}
	}
	class, ok := retention["class"].(string)
	if !ok || strings.TrimSpace(class) == "" {
		return errors.New("spec.retention.class must be a non-empty string")
	}
	class = strings.TrimSpace(class)

	profile, configured := configuredRetentionProfile(config, class)
	if !configured {
		return errors.Errorf("retention class %q is not configured by the platform", class)
	}
	profileConfig, err := retentionFanoutProfileFromConfig(profile, class)
	if err != nil {
		return err
	}

	metadata, _ := xr["metadata"].(map[string]any)
	namespace, _ := metadata["namespace"].(string)
	slug, _ := spec["slug"].(string)
	if namespace == "" || slug == "" {
		return errors.New("retention fan-out requires metadata.namespace and spec.slug")
	}

	pipelineName := slug + "-retention-fanout"
	desired["retention-fanout"] = newDesired(
		"fleetmanagement.grafana.m.crossplane.io/v1alpha1",
		"Pipeline",
		namespace,
		pipelineName,
		map[string]any{"crossplane.io/external-name": pipelineName},
		map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider": map[string]any{
				"configType":               profileConfig.configType,
				"contents":                 profileConfig.contents,
				"enabled":                  true,
				"matchers":                 profileConfig.matchers,
				"name":                     pipelineName,
				"terraformSourceNamespace": "default",
			},
			"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": slug},
		},
	)
	return nil
}

func configuredRetentionProfile(config map[string]any, class string) (map[string]any, bool) {
	spec, _ := config["spec"].(map[string]any)
	rawProfiles, ok := spec["retentionProfiles"]
	if !ok {
		return nil, false
	}

	switch profiles := rawProfiles.(type) {
	case []any:
		for _, rawProfile := range profiles {
			profile, ok := rawProfile.(map[string]any)
			if ok && stringValue(profile, "name", "") == class {
				return profile, true
			}
		}
	case map[string]any:
		profile, ok := profiles[class].(map[string]any)
		if ok {
			return profile, true
		}
	}
	return nil, false
}

func retentionFanoutProfileFromConfig(profile map[string]any, class string) (retentionFanoutProfile, error) {
	contents, ok := profile["contents"].(string)
	if !ok || strings.TrimSpace(contents) == "" {
		return retentionFanoutProfile{}, errors.Errorf("retention profile %q must set contents", class)
	}

	configType := "ALLOY"
	if rawConfigType, exists := profile["configType"]; exists {
		var ok bool
		configType, ok = rawConfigType.(string)
		if !ok {
			return retentionFanoutProfile{}, errors.Errorf("retention profile %q configType must be ALLOY or OTEL", class)
		}
	}
	if !oneOf(configType, "ALLOY", "OTEL") {
		return retentionFanoutProfile{}, errors.Errorf("retention profile %q configType must be ALLOY or OTEL", class)
	}

	rawMatchers, ok := profile["matchers"].([]any)
	if !ok || len(rawMatchers) == 0 {
		return retentionFanoutProfile{}, errors.Errorf("retention profile %q must set at least one matcher", class)
	}
	matchers := make([]any, len(rawMatchers))
	for index, rawMatcher := range rawMatchers {
		matcher, ok := rawMatcher.(string)
		if !ok || strings.TrimSpace(matcher) == "" {
			return retentionFanoutProfile{}, errors.Errorf("retention profile %q matcher %d must be a non-empty string", class, index)
		}
		matchers[index] = matcher
	}

	return retentionFanoutProfile{configType: configType, contents: contents, matchers: matchers}, nil
}
