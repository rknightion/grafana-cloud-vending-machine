package main

import (
	"fmt"
	hashfnv "hash/fnv"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/resource"
)

func addSSO(desired map[resource.Name]*resource.DesiredComposed, namespace, providerConfig string, spec, config map[string]any) error {
	sso, _ := spec["sso"].(map[string]any)
	mode := stringValue(sso, "mode", "disabled")
	if !oneOf(mode, "enforced", "createOnly", "observeOnly", "disabled") {
		return errors.Errorf("unsupported SSO reconciliation mode %q", mode)
	}
	if mode == "disabled" {
		return nil
	}

	profileName, _ := sso["profile"].(string)
	profile, ok := configuredProfile(config, "ssoProfiles", profileName)
	if !ok {
		return errors.Errorf("SSO profile %q is not configured by the platform", profileName)
	}
	providerName, _ := profile["providerName"].(string)
	if providerName == "" {
		return errors.Errorf("SSO profile %q has no providerName", profileName)
	}

	forProvider := map[string]any{"providerName": providerName}
	resourceSpec := map[string]any{
		"managementPolicies": managementPolicies,
		"forProvider":        forProvider,
		"providerConfigRef":  map[string]any{"kind": "ProviderConfig", "name": providerConfig},
	}
	if mode == "observeOnly" {
		resourceSpec["managementPolicies"] = []any{"Observe"}
	} else {
		settings, err := safeOAuthSettings(profile)
		if err != nil {
			return errors.Wrapf(err, "invalid SSO profile %q", profileName)
		}
		if mode == "enforced" {
			forProvider["oauth2Settings"] = []any{settings}
		} else {
			resourceSpec["initProvider"] = map[string]any{"oauth2Settings": []any{settings}}
		}
	}

	desired["sso"] = newDesired("oss.grafana.m.crossplane.io/v1alpha1", "SsoSettings", namespace, providerConfig+"-sso",
		map[string]any{"crossplane.io/external-name": providerName}, resourceSpec)
	return nil
}

func safeOAuthSettings(profile map[string]any) (map[string]any, error) {
	raw, _ := profile["oauth2Settings"].(map[string]any)
	allowed := []string{
		"allowAssignGrafanaAdmin", "allowSignUp", "allowedDomains", "allowedGroups", "allowedOrganizations",
		"apiUrl", "authStyle", "authUrl", "autoLogin", "clientId", "custom", "defineAllowedGroups",
		"defineAllowedTeamsIds", "emailAttributeName", "emailAttributePath", "emptyScopes", "enabled",
		"groupsAttributePath", "idTokenAttributeName", "loginAttributePath", "loginPrompt", "name",
		"nameAttributePath", "orgAttributePath", "orgMapping", "roleAttributePath", "roleAttributeStrict",
		"scopes", "signoutRedirectUrl", "skipOrgRoleSync", "teamIds", "teamIdsAttributePath", "teamsUrl",
		"tlsClientCa", "tlsClientCert", "tlsClientKey", "tlsSkipVerifyInsecure", "tokenUrl", "usePkce", "useRefreshToken",
	}
	out := map[string]any{}
	for _, key := range allowed {
		if value, ok := raw[key]; ok {
			out[key] = value
		}
	}
	secretRef, _ := raw["clientSecretSecretRef"].(map[string]any)
	name, _ := secretRef["name"].(string)
	key, _ := secretRef["key"].(string)
	if name == "" || key == "" {
		return nil, errors.New("oauth2Settings.clientSecretSecretRef must set name and key")
	}
	out["clientSecretSecretRef"] = map[string]any{"name": name, "key": key}
	return out, nil
}

func addMonthlyReport(desired map[resource.Name]*resource.DesiredComposed, namespace, providerConfig, slug string, spec map[string]any, baselineEnabled bool) error {
	report, _ := spec["monthlyReport"].(map[string]any)
	enabled, _ := report["enabled"].(bool)
	if !enabled {
		return nil
	}
	if !baselineEnabled {
		return errors.New("monthlyReport requires baselineDashboards.enabled")
	}
	recipients, _ := report["recipients"].([]any)
	replyTo, _ := report["replyTo"].(string)
	if len(recipients) == 0 || replyTo == "" {
		return errors.New("monthlyReport requires at least one recipient and replyTo")
	}

	desired["monthly-report"] = newDesired("enterprise.grafana.m.crossplane.io/v1alpha1", "Report", namespace, slug+"-monthly-report", nil,
		map[string]any{
			"managementPolicies": managementPolicies,
			"forProvider": map[string]any{
				"name": "Monthly usage report for " + slug,
				"dashboards": []any{map[string]any{
					"uid": "vending-billing", "timeRange": []any{map[string]any{"from": "now-1M/M", "to": "now-1M/M"}},
				}},
				"message": "Please find the previous month's usage report attached.", "recipients": recipients,
				"schedule": []any{map[string]any{"frequency": "monthly", "startTime": deterministicReportStart(slug), "timezone": "UTC"}},
				"replyTo":  replyTo, "formats": []any{"pdf", "csv"}, "includeTableCsv": true,
			},
			"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": providerConfig},
		})
	return nil
}

func deterministicReportStart(slug string) string {
	h := hashfnv.New32a()
	_, _ = h.Write([]byte(slug))
	slot := h.Sum32() % (24 * 6)
	return fmt.Sprintf("2020-01-01T%02d:%02d:00", slot/6, (slot%6)*10)
}

func addIncidentIntegration(desired map[resource.Name]*resource.DesiredComposed, namespace, providerConfig, slug string, spec, config map[string]any) error {
	incident, _ := spec["incidentIntegration"].(map[string]any)
	enabled, _ := incident["enabled"].(bool)
	if !enabled {
		return nil
	}
	profileName, _ := incident["profile"].(string)
	profile, ok := configuredProfile(config, "incidentProfiles", profileName)
	if !ok {
		return errors.Errorf("incident integration profile %q is not configured by the platform", profileName)
	}
	url, _ := profile["url"].(string)
	secretRef, _ := profile["authorizationSecretRef"].(map[string]any)
	secretName, _ := secretRef["name"].(string)
	secretKey, _ := secretRef["key"].(string)
	if url == "" || secretName == "" || secretKey == "" {
		return errors.Errorf("incident integration profile %q must set url and authorizationSecretRef", profileName)
	}
	authRef := map[string]any{"name": secretName, "key": secretKey}

	for _, environment := range []string{"test", "production"} {
		for _, trigger := range []struct{ suffix, value string }{{"firing", "escalation"}, {"resolved", "resolve"}} {
			name := "incident-oncall-" + environment + "-" + trigger.suffix
			desired[resource.Name(name)] = newDesired("oncall.grafana.m.crossplane.io/v1alpha1", "OutgoingWebhook", namespace, slug+"-"+name, nil,
				map[string]any{
					"managementPolicies": managementPolicies,
					"forProvider": map[string]any{
						"name": fmt.Sprintf("Incident relay %s %s", environment, trigger.suffix), "url": url, "httpMethod": "POST",
						"authorizationHeaderSecretRef": authRef, "triggerType": trigger.value,
						"data": `{"environment":"` + environment + `","event":{{ payload | toJson }}}`,
					},
					"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": providerConfig},
				})
		}

		name := "incident-alerting-" + environment
		desired[resource.Name(name)] = newDesired("alerting.grafana.m.crossplane.io/v1alpha1", "ContactPoint", namespace, slug+"-"+name, nil,
			map[string]any{
				"managementPolicies": managementPolicies,
				"forProvider": map[string]any{
					"name": "Incident relay " + environment,
					"webhook": []any{map[string]any{
						"url": url, "httpMethod": "POST", "authorizationScheme": "Bearer",
						"authorizationCredentialsSecretRef": authRef,
						"payload":                           []any{map[string]any{"template": `{"environment":"` + environment + `","alerts":{{ .Alerts | toJson }}}`}},
					}},
				},
				"providerConfigRef": map[string]any{"kind": "ProviderConfig", "name": providerConfig},
			})
	}
	return nil
}

func configuredProfile(config map[string]any, field, name string) (map[string]any, bool) {
	spec, _ := config["spec"].(map[string]any)
	profiles, _ := spec[field].([]any)
	for _, item := range profiles {
		profile, _ := item.(map[string]any)
		if profile["name"] == name && name != "" {
			return profile, true
		}
	}
	return nil, false
}

func ownedParameters(spec map[string]any, mode string) map[string]any {
	field := "forProvider"
	if mode == "createOnly" {
		field = "initProvider"
	}
	parameters, _ := spec[field].(map[string]any)
	if parameters == nil {
		parameters = map[string]any{}
		spec[field] = parameters
	}
	return parameters
}

func stringValue(object map[string]any, key, fallback string) string {
	if value, ok := object[key].(string); ok && value != "" {
		return value
	}
	return fallback
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
