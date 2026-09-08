package main

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composite"
)

const inventoryRendererImplemented = true

const (
	observedInventoryAPIVersion = "v1alpha1"
	observedInventoryOSSGroup   = "oss.grafana.o.crossplane.io"
	observedInventorySMGroup    = "sm.grafana.o.crossplane.io"
	observedInventoryFleetGroup = "fleetmanagement.grafana.o.crossplane.io"
)

// inventoryProviderSet is deliberately a data-only description of a provider
// observe-only set. Keeping the exact group, kind, and child status field in
// one table makes it difficult for a new renderer path to accidentally emit a
// mutating .m. resource or to silently use a guessed plural.
type inventoryProviderSet struct {
	apiVersion  string
	kind        string
	plural      string
	itemKind    string
	statusField string
	childName   string
	forProvider map[string]any
}

var stackInventoryProviderSets = []inventoryProviderSet{
	{
		apiVersion:  observedInventoryOSSGroup + "/" + observedInventoryAPIVersion,
		kind:        "FolderSet",
		plural:      "foldersets.oss.grafana.o.crossplane.io",
		itemKind:    "Folder",
		statusField: "folders",
		childName:   "folders",
	},
	{
		apiVersion:  observedInventoryOSSGroup + "/" + observedInventoryAPIVersion,
		kind:        "DashboardSet",
		plural:      "dashboardsets.oss.grafana.o.crossplane.io",
		itemKind:    "Dashboard",
		statusField: "dashboards",
		childName:   "dashboards",
	},
	{
		apiVersion:  observedInventoryOSSGroup + "/" + observedInventoryAPIVersion,
		kind:        "TeamSet",
		plural:      "teamsets.oss.grafana.o.crossplane.io",
		itemKind:    "Team",
		statusField: "teams",
		childName:   "teams",
	},
	{
		apiVersion:  observedInventoryOSSGroup + "/" + observedInventoryAPIVersion,
		kind:        "UserSet",
		plural:      "usersets.oss.grafana.o.crossplane.io",
		itemKind:    "User",
		statusField: "users",
		childName:   "users",
	},
	{
		apiVersion:  observedInventoryOSSGroup + "/" + observedInventoryAPIVersion,
		kind:        "LibraryPanelSet",
		plural:      "librarypanelsets.oss.grafana.o.crossplane.io",
		itemKind:    "LibraryPanel",
		statusField: "panels",
		childName:   "library-panels",
	},
	{
		apiVersion:  observedInventorySMGroup + "/" + observedInventoryAPIVersion,
		kind:        "ProbeSet",
		plural:      "probesets.sm.grafana.o.crossplane.io",
		itemKind:    "Probe",
		statusField: "probes",
		childName:   "probes",
		forProvider: map[string]any{"filterDeprecated": false},
	},
	{
		apiVersion:  observedInventoryFleetGroup + "/" + observedInventoryAPIVersion,
		kind:        "CollectorSet",
		plural:      "collectorsets.fleetmanagement.grafana.o.crossplane.io",
		itemKind:    "Collector",
		statusField: "collectors",
		childName:   "collectors",
	},
}

var inventoryOrganizationUser = inventoryProviderSet{
	apiVersion: observedInventoryOSSGroup + "/" + observedInventoryAPIVersion,
	plural:     "organizationusers.oss.grafana.o.crossplane.io",
	itemKind:   "OrganizationUser",
}

type inventoryDeclaration struct {
	apiVersion   string
	kind         string
	name         string
	externalName string
	email        string
	login        string
}

type inventoryObject struct {
	apiVersion   string
	kind         string
	name         string
	externalName string
	email        string
	login        string
}

func renderStackInventory(xr map[string]any, _ map[resource.Name]resource.ObservedComposed, _ map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	metadata, _ := xr["metadata"].(map[string]any)
	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	spec, _ := xr["spec"].(map[string]any)
	stackRef, _ := spec["stackRef"].(map[string]any)
	stackName, _ := stackRef["name"].(string)
	if name == "" || namespace == "" || stackName == "" {
		return nil, errors.New("stack inventory must set metadata.name, metadata.namespace, and spec.stackRef.name")
	}

	declarations, err := inventoryDeclarations(spec)
	if err != nil {
		return nil, err
	}

	desired := make(map[resource.Name]*resource.DesiredComposed, len(stackInventoryProviderSets)+len(declarations))
	for _, set := range stackInventoryProviderSets {
		forProvider := copyInventoryMap(set.forProvider)
		desired[resource.Name(set.childName)] = newDesired(
			set.apiVersion,
			set.kind,
			namespace,
			name+"-"+set.childName,
			nil,
			map[string]any{
				"managementPolicies": []any{"Observe"},
				"forProvider":        forProvider,
				"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": stackName},
			},
		)
	}

	seenOrganizationUsers := map[string]struct{}{}
	for index, declaration := range declarations {
		if declaration.kind != inventoryOrganizationUser.itemKind {
			continue
		}
		query := ""
		if declaration.email != "" {
			query = "email:" + declaration.email
		} else if declaration.login != "" {
			query = "login:" + declaration.login
		}
		if query == "" {
			return nil, errors.Errorf("declared[%d] OrganizationUser must set email or login", index)
		}
		if _, exists := seenOrganizationUsers[query]; exists {
			return nil, errors.Errorf("declared[%d] duplicates OrganizationUser query %q", index, query)
		}
		seenOrganizationUsers[query] = struct{}{}

		forProvider := map[string]any{}
		if declaration.email != "" {
			forProvider["email"] = declaration.email
		} else {
			forProvider["login"] = declaration.login
		}
		logicalName := "organization-user-" + stableResourceSuffix(query)
		desired[resource.Name(logicalName)] = newDesired(
			inventoryOrganizationUser.apiVersion,
			inventoryOrganizationUser.itemKind,
			namespace,
			name+"-"+logicalName,
			nil,
			map[string]any{
				"managementPolicies": []any{"Observe"},
				"forProvider":        forProvider,
				"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": stackName},
			},
		)
	}

	return desired, nil
}

// desiredStackInventoryStatus is kept separate from the renderer because the
// composition function's shared RunFunction wiring owns response mutation. It
// gives that wiring one deterministic, side-effect-free status builder while
// keeping the inventory feature in its own file.
func desiredStackInventoryStatus(xr map[string]any, observed map[resource.Name]resource.ObservedComposed) *resource.Composite {
	spec, _ := xr["spec"].(map[string]any)
	declarations, _ := inventoryDeclarations(spec)
	declared := make([]any, 0, len(declarations))
	for _, declaration := range declarations {
		declared = append(declared, declaration.statusMap())
	}
	sort.SliceStable(declared, func(i, j int) bool {
		return inventoryStatusSortKey(declared[i].(map[string]any)) < inventoryStatusSortKey(declared[j].(map[string]any))
	})

	managed := make([]any, 0)
	unmanaged := make([]any, 0)
	for _, object := range observedInventoryObjects(observed) {
		value := object.statusMap()
		if inventoryObjectDeclared(object, declarations) {
			managed = append(managed, value)
		} else {
			unmanaged = append(unmanaged, value)
		}
	}
	sortInventoryStatus(managed)
	sortInventoryStatus(unmanaged)

	status := map[string]any{
		"declared":             declared,
		"observedAndManaged":   managed,
		"observedButUnmanaged": unmanaged,
	}
	result := composite.New()
	result.SetUnstructuredContent(map[string]any{"status": status})
	return &resource.Composite{Resource: result, Ready: resource.ReadyUnspecified}
}

func inventoryDeclarations(spec map[string]any) ([]inventoryDeclaration, error) {
	raw, exists := spec["declared"]
	if !exists || raw == nil {
		return nil, nil
	}
	values, ok := raw.([]any)
	if !ok {
		return nil, errors.New("spec.declared must be an array")
	}
	result := make([]inventoryDeclaration, 0, len(values))
	for index, value := range values {
		item, ok := value.(map[string]any)
		if !ok {
			return nil, errors.Errorf("spec.declared[%d] must be an object", index)
		}
		declaration := inventoryDeclaration{
			apiVersion:   inventoryString(item["apiVersion"]),
			kind:         inventoryString(item["kind"]),
			name:         inventoryString(item["name"]),
			externalName: inventoryString(item["externalName"]),
			email:        inventoryString(item["email"]),
			login:        inventoryString(item["login"]),
		}
		if declaration.apiVersion == "" || declaration.kind == "" {
			return nil, errors.Errorf("spec.declared[%d] must set apiVersion and kind", index)
		}
		if declaration.name == "" && declaration.externalName == "" && declaration.email == "" && declaration.login == "" {
			return nil, errors.Errorf("spec.declared[%d] must set name, externalName, email, or login", index)
		}
		result = append(result, declaration)
	}
	return result, nil
}

func observedInventoryObjects(observed map[resource.Name]resource.ObservedComposed) []inventoryObject {
	result := make([]inventoryObject, 0)
	for _, set := range stackInventoryProviderSets {
		observedResource, ok := observed[resource.Name(set.childName)]
		if !ok || observedResource.Resource == nil {
			continue
		}
		content := observedResource.Resource.UnstructuredContent()
		status, _ := content["status"].(map[string]any)
		atProvider, _ := status["atProvider"].(map[string]any)
		result = append(result, inventoryObjectsFromValue(set, atProvider[set.statusField])...)
	}

	for name, observedResource := range observed {
		if !strings.HasPrefix(string(name), "organization-user-") || observedResource.Resource == nil {
			continue
		}
		content := observedResource.Resource.UnstructuredContent()
		status, _ := content["status"].(map[string]any)
		atProvider, _ := status["atProvider"].(map[string]any)
		if object := inventoryObjectFromMap(inventoryOrganizationUser, atProvider, ""); object.externalName != "" || object.name != "" || object.email != "" || object.login != "" {
			result = append(result, object)
		}
	}
	return result
}

func inventoryObjectsFromValue(set inventoryProviderSet, value any) []inventoryObject {
	result := make([]inventoryObject, 0)
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if object := inventoryObjectFromValue(set, item, ""); object.externalName != "" || object.name != "" || object.email != "" || object.login != "" {
				result = append(result, object)
			}
		}
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if object := inventoryObjectFromValue(set, typed[key], key); object.externalName != "" || object.name != "" || object.email != "" || object.login != "" {
				result = append(result, object)
			}
		}
	case string, float64, float32, int, int32, int64, json.Number:
		if object := inventoryObjectFromValue(set, typed, ""); object.externalName != "" || object.name != "" {
			result = append(result, object)
		}
	}
	return result
}

func inventoryObjectFromValue(set inventoryProviderSet, value any, mapKey string) inventoryObject {
	object := inventoryObject{apiVersion: set.apiVersion, kind: set.itemKind}
	if mapKey != "" {
		object.name = mapKey
	}
	switch typed := value.(type) {
	case map[string]any:
		object = inventoryObjectFromMap(set, typed, mapKey)
	case string, float64, float32, int, int32, int64, json.Number:
		identity := inventoryString(typed)
		if mapKey != "" {
			object.externalName = identity
		} else {
			object.name = identity
			object.externalName = identity
		}
	}
	return object
}

func inventoryObjectFromMap(set inventoryProviderSet, value map[string]any, mapKey string) inventoryObject {
	object := inventoryObject{apiVersion: set.apiVersion, kind: set.itemKind}
	object.email = inventoryString(value["email"])
	object.login = inventoryString(value["login"])
	if mapKey != "" {
		object.name = mapKey
	}
	for _, key := range []string{"uid", "id", "userID", "teamID", "teamId", "externalName"} {
		if identity := inventoryString(value[key]); identity != "" {
			object.externalName = identity
			break
		}
	}
	for _, key := range []string{"name", "title", "login", "email", "uid", "id"} {
		if displayName := inventoryString(value[key]); displayName != "" {
			object.name = displayName
			break
		}
	}
	return object
}

func inventoryObjectDeclared(object inventoryObject, declarations []inventoryDeclaration) bool {
	for _, declaration := range declarations {
		if declaration.kind != object.kind || normalizeInventoryAPIVersion(declaration.apiVersion) != normalizeInventoryAPIVersion(object.apiVersion) {
			continue
		}
		if declaration.externalName != "" {
			if declaration.externalName == object.externalName {
				return true
			}
			continue
		}
		if declaration.name != "" && declaration.name == object.name {
			return true
		}
		if declaration.email != "" && declaration.email == object.email {
			return true
		}
		if declaration.login != "" && declaration.login == object.login {
			return true
		}
	}
	return false
}

func normalizeInventoryAPIVersion(apiVersion string) string {
	return strings.Replace(apiVersion, ".m.", ".o.", 1)
}

func (d inventoryDeclaration) statusMap() map[string]any {
	result := map[string]any{"apiVersion": d.apiVersion, "kind": d.kind}
	if d.name != "" {
		result["name"] = d.name
	}
	if d.externalName != "" {
		result["externalName"] = d.externalName
	}
	if d.email != "" {
		result["email"] = d.email
	}
	if d.login != "" {
		result["login"] = d.login
	}
	return result
}

func (o inventoryObject) statusMap() map[string]any {
	result := map[string]any{"apiVersion": o.apiVersion, "kind": o.kind}
	if o.name != "" {
		result["name"] = o.name
	}
	if o.externalName != "" {
		result["externalName"] = o.externalName
	}
	if o.email != "" {
		result["email"] = o.email
	}
	if o.login != "" {
		result["login"] = o.login
	}
	return result
}

func sortInventoryStatus(items []any) {
	sort.SliceStable(items, func(i, j int) bool {
		return inventoryStatusSortKey(items[i].(map[string]any)) < inventoryStatusSortKey(items[j].(map[string]any))
	})
}

func inventoryStatusSortKey(value map[string]any) string {
	return fmt.Sprintf("%s/%s/%s/%s", inventoryString(value["apiVersion"]), inventoryString(value["kind"]), inventoryString(value["externalName"]), inventoryString(value["name"]))
}

func inventoryString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	case float64:
		if typed == math.Trunc(typed) {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 32)
	case int:
		return strconv.Itoa(typed)
	case int32:
		return strconv.FormatInt(int64(typed), 10)
	case int64:
		return strconv.FormatInt(typed, 10)
	default:
		return ""
	}
}

func copyInventoryMap(value map[string]any) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	result := make(map[string]any, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}
