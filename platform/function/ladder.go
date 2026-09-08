package main

import (
	"fmt"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composite"
)

const ladderRendererImplemented = true

type stackLadderRung struct {
	name           string
	slug           string
	displayName    string
	usage          string
	profile        string
	branch         string
	connectionName string
}

// renderStackLadder builds an independent stack and Git provisioning
// repository for every declared rung. It deliberately only configures each
// repository to read its branch: moving content between branches remains a
// reviewed Git operation outside this composition.
func renderStackLadder(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, config map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	metadata, _ := xr["metadata"].(map[string]any)
	spec, _ := xr["spec"].(map[string]any)
	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	organization, _ := spec["organization"].(string)
	region, _ := spec["region"].(string)
	direction, _ := spec["promotionDirection"].(string)
	repository, _ := spec["repository"].(map[string]any)
	repositoryURL, _ := repository["url"].(string)
	repositoryPath, _ := repository["path"].(string)
	if name == "" || namespace == "" || organization == "" || region == "" || repositoryURL == "" || repositoryPath == "" {
		return nil, errors.New("stack ladder must set metadata.name and namespace plus organization, region, repository.url, and repository.path")
	}
	if direction != "forward" && direction != "reverse" {
		return nil, errors.Errorf("unsupported promotion direction %q", direction)
	}
	rungs, err := stackLadderRungs(spec)
	if err != nil {
		return nil, err
	}
	maxStacks, err := stackLadderMaxStacks(config)
	if err != nil {
		return nil, err
	}
	if len(rungs) > maxStacks {
		return nil, errors.Errorf("ladder has %d rungs, exceeding platform maximum of %d stacks", len(rungs), maxStacks)
	}

	desired := map[resource.Name]*resource.DesiredComposed{}
	for _, rung := range rungs {
		stackDesired, err := renderStack(map[string]any{
			"apiVersion": "platform.example.org/v1beta1",
			"kind":       "GrafanaCloudStackRequest",
			"metadata":   map[string]any{"name": rung.slug, "namespace": namespace},
			"spec": map[string]any{
				"displayName":  rung.displayName,
				"slug":         rung.slug,
				"region":       region,
				"usage":        rung.usage,
				"organization": organization,
				"profile":      rung.profile,
			},
		}, stackLadderRungObserved(observed, rung.name), config)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot render rung %q stack", rung.name)
		}
		mergeLadderResources(desired, rung.name, stackDesired)

		repositoryDesired, err := renderProvisioningRepository(map[string]any{
			"apiVersion": "platform.example.org/v1beta1",
			"kind":       "GrafanaProvisioningRepository",
			"metadata":   map[string]any{"name": name + "-" + rung.name + "-repository", "namespace": namespace},
			"spec": map[string]any{
				"stackRef": map[string]any{"name": rung.slug},
				"repository": map[string]any{
					"uid":           name + "-" + rung.name,
					"title":         rung.displayName + " content",
					"description":   fmt.Sprintf("Git-backed content for the %s rung of ladder %s.", rung.name, name),
					"url":           repositoryURL,
					"branch":        rung.branch,
					"path":          repositoryPath,
					"connectionRef": map[string]any{"name": rung.connectionName},
				},
			},
		}, nil, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot render rung %q provisioning repository", rung.name)
		}
		mergeLadderResources(desired, rung.name, repositoryDesired)
	}
	return desired, nil
}

// desiredStackLadderStatus exposes the ordered content flow and the observed
// repository version at every rung. A missing version remains Unknown rather
// than inventing an in-sync result.
func desiredStackLadderStatus(xr map[string]any, observed map[resource.Name]resource.ObservedComposed) *resource.Composite {
	spec, _ := xr["spec"].(map[string]any)
	direction, _ := spec["promotionDirection"].(string)
	rungs, _ := stackLadderRungs(spec)
	statusRungs := make([]any, 0, len(rungs))
	for _, rung := range rungs {
		entry := map[string]any{
			"name":      rung.name,
			"stackName": rung.slug,
			"ready":     observedReady(observed, stackLadderResourceName(rung.name, "stack")),
			"branch":    rung.branch,
		}
		if revision := observedString(observed, stackLadderResourceName(rung.name, "repository"), "status.atProvider.metadata.version"); revision != "" {
			entry["observedRevision"] = revision
		}
		statusRungs = append(statusRungs, entry)
	}

	if direction == "reverse" {
		for index := len(statusRungs) - 1; index >= 0; index-- {
			stackLadderDrift(statusRungs, index, index+1)
		}
	} else {
		for index := range statusRungs {
			stackLadderDrift(statusRungs, index, index-1)
		}
	}
	result := composite.New()
	result.SetUnstructuredContent(map[string]any{"status": map[string]any{
		"promotionDirection": direction,
		"rungs":              statusRungs,
	}})
	return &resource.Composite{Resource: result, Ready: resource.ReadyUnspecified}
}

func stackLadderDrift(rungs []any, index, sourceIndex int) {
	entry := rungs[index].(map[string]any)
	if sourceIndex < 0 || sourceIndex >= len(rungs) {
		entry["drift"] = "Source"
		return
	}
	source := rungs[sourceIndex].(map[string]any)
	entry["sourceRung"] = source["name"]
	revision, hasRevision := entry["observedRevision"].(string)
	sourceRevision, hasSourceRevision := source["observedRevision"].(string)
	if !hasRevision || !hasSourceRevision {
		entry["drift"] = "Unknown"
		return
	}
	if revision == sourceRevision {
		entry["drift"] = "InSync"
		return
	}
	entry["drift"] = "Drifted"
}

func stackLadderRungs(spec map[string]any) ([]stackLadderRung, error) {
	values, ok := spec["rungs"].([]any)
	if !ok || len(values) < 2 {
		return nil, errors.New("stack ladder must declare at least two rungs")
	}
	result := make([]stackLadderRung, 0, len(values))
	names := map[string]bool{}
	slugs := map[string]bool{}
	for index, value := range values {
		item, ok := value.(map[string]any)
		if !ok {
			return nil, errors.Errorf("ladder rung %d must be an object", index)
		}
		connectionRef, _ := item["connectionRef"].(map[string]any)
		rung := stackLadderRung{
			name:           stringValue(item, "name", ""),
			slug:           stringValue(item, "slug", ""),
			displayName:    stringValue(item, "displayName", ""),
			usage:          stringValue(item, "usage", ""),
			profile:        stringValue(item, "profile", "standard"),
			branch:         stringValue(item, "branch", ""),
			connectionName: stringValue(connectionRef, "name", ""),
		}
		if rung.name == "" || rung.slug == "" || rung.displayName == "" || rung.usage == "" || rung.branch == "" || rung.connectionName == "" {
			return nil, errors.Errorf("ladder rung %d must set name, slug, displayName, usage, branch, and connectionRef.name", index)
		}
		if names[rung.name] || slugs[rung.slug] {
			return nil, errors.Errorf("ladder rung %q repeats a name or slug", rung.name)
		}
		names[rung.name] = true
		slugs[rung.slug] = true
		result = append(result, rung)
	}
	return result, nil
}

func stackLadderMaxStacks(config map[string]any) (int, error) {
	spec, _ := config["spec"].(map[string]any)
	policy, _ := spec["ladderPolicy"].(map[string]any)
	value, exists := policy["maxStacks"]
	if !exists {
		return 0, errors.New("platform configuration must set ladderPolicy.maxStacks")
	}
	var maximum int
	switch value := value.(type) {
	case int:
		maximum = value
	case int64:
		maximum = int(value)
	case float64:
		maximum = int(value)
		if value != float64(maximum) {
			return 0, errors.New("platform ladderPolicy.maxStacks must be a whole number")
		}
	default:
		return 0, errors.New("platform ladderPolicy.maxStacks must be a number")
	}
	if maximum < 1 {
		return 0, errors.New("platform ladderPolicy.maxStacks must be positive")
	}
	return maximum, nil
}

func stackLadderResourceName(rung, child string) resource.Name {
	return resource.Name("rung-" + rung + "-" + child)
}

func stackLadderRungObserved(observed map[resource.Name]resource.ObservedComposed, rung string) map[resource.Name]resource.ObservedComposed {
	prefix := string(stackLadderResourceName(rung, ""))
	result := map[resource.Name]resource.ObservedComposed{}
	for name, value := range observed {
		key := string(name)
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			result[resource.Name(key[len(prefix):])] = value
		}
	}
	return result
}

func mergeLadderResources(destination map[resource.Name]*resource.DesiredComposed, rung string, source map[resource.Name]*resource.DesiredComposed) {
	for name, desired := range source {
		destination[stackLadderResourceName(rung, string(name))] = desired
	}
}
