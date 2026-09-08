---
title: Request Schema Reference
description: The CompositeResourceDefinitions this repository ships, their kinds, short names, and CRD identity
---

# Request Schema Reference

This is the CRD-identity reference. Field-level defaults and ownership details are in
[Configuration](../configuration.md); worked manifests are in the
[catalog](catalog.md). All types are namespaced, use
`platform.example.org/v1beta1` as a placeholder group, set
`defaultCompositionUpdatePolicy: Automatic`, and enforce one Pipeline Composition.

## CompositeResourceDefinitions

| Kind | Plural | Short name | Enforced Composition | Required `spec` fields |
| --- | --- | --- | --- | --- |
| `GrafanaCloudStackRequest` | `grafanacloudstackrequests` | `gcstackrequest` | `grafana-cloud-stack-request-v1beta1` | `displayName`, `slug`, `region`, `usage`, `organization` |
| `GrafanaCustomRoleBinding` | `grafanacustomrolebindings` | `gcrole` | `grafana-custom-role-binding-v1beta1` | `stackRef`, `team`, `role` |
| `GrafanaTeamAccess` | `grafanateamaccesses` | `gcteamaccess` | `grafana-team-access-v1beta1` | `stackRef`, `team` |
| `GrafanaContentAccessPolicy` | `grafanacontentaccesspolicies` | `gccontentaccess` | `grafana-content-access-policy-v1beta1` | `stackRef`, `target`, `permissions` |
| `GrafanaStackInventory` | `grafanastackinventories` | `gcinventory` | `grafana-stack-inventory-v1beta1` | `stackRef` |
| `GrafanaFleetPipelines` | `grafanafleetpipelines` | `gcfleet` | `grafana-fleet-pipelines-v1beta1` | `stackRef`, `profile` |
| `GrafanaAlertingBundle` | `grafanaalertingbundles` | `gcalerts` | `grafana-alerting-bundle-v1beta1` | `stackRef`, `provenance` |
| `GrafanaAgentObservability` | `grafanaagentobservabilities` | `gcagento11y` | `grafana-agent-observability-v1beta1` | `stackRef` |
| `GrafanaAssistantGovernance` | `grafanaassistantgovernances` | `gcassistant` | `grafana-assistant-governance-v1beta1` | `stackRef`, `termsAcceptance` |
| `GrafanaDatasourceAccess` | `grafanadatasourceaccesses` | `gcdatasourceaccess` | `grafana-datasource-access-v1beta1` | `stackRef`, `datasource`, `teams` |
| `GrafanaProvisioningRepository` | `grafanaprovisioningrepositories` | `gcprovisioningrepo` | `grafana-provisioning-repository-v1beta1` | `stackRef`, `repository` |

| `GrafanaK6Project` | `grafanak6projects` | `gck6` | `grafana-k6-project-v1beta1` | `stackRef`, `grafanaUser`, `allowedLoadZones` |
| `GrafanaSyntheticMonitoring` | `grafanasyntheticmonitorings` | `gcsm` | `grafana-synthetic-monitoring-v1beta1` | `stackRef` |
| `GrafanaStackLadder` | `grafanastackladders` | None | `grafana-stack-ladder-v1beta1` | `organization`, `region`, `promotionDirection`, `rungs`, `repository` |

The XRDs and Compositions are split by API under `platform/apis/`. Every Composition has one
Pipeline step that calls `function-grafana-vending`; the function, rather than a separate
templating language, renders the managed resources.

## Key admission and ownership guards

- A stack request requires immutable `spec.organization`, `spec.slug`, `spec.usage`, and
  `spec.profile`. Its organization key must resolve in the platform-owned registry; the selected
  organization must allow the requested region and usage.
- `GrafanaDatasourceAccess.metadata.name` must equal `spec.datasource.uid`; its datasource UID and
  stack reference are immutable. This gives one namespace-local owner for the datasource access
  set.
- `GrafanaProvisioningRepository` rejects a `dashboard` field. A Git-provisioned subtree and a
  classic Crossplane Dashboard must never claim the same content route.
- `GrafanaAlertingBundle.provenance` is required: `enforced` retains provisioning provenance and
  locks UI changes, while `createOnly` seeds values and preserves later UI edits.
- `GrafanaAssistantGovernance.termsAcceptance.accepted` is the safety gate. Rules and MCP servers
  are withheld until acceptance is observed, and a false value withdraws them.

## Quick lookups by short name

```bash
kubectl get gcstackrequest,gcrole,gcteamaccess,gccontentaccess -A
kubectl get gcinventory,gcfleet,gcalerts,gcagento11y,gcassistant -A
kubectl get gcdatasourceaccess,gcprovisioningrepo -A
```

## Status conditions

Every kind reports standard Crossplane `Synced` and `Ready` conditions. The stack request also
publishes `status.outputSecretPath`, `status.telemetrySecretPath` when enabled,
`status.deletionArmed`, `status.deletionReady`, `status.stack.id`, and `status.stack.url`.
`GrafanaStackInventory` additionally publishes its declared, managed, and unmanaged observation
classification. These status values are observations, not credentials or a substitute for an
inventory/adoption review.

## Next steps

- [Configuration](../configuration.md) - API fields, activation choices, and limitations.
- [Catalog Reference](catalog.md) - inert examples for every public API.

## Governance additions

`spec.retention.class` selects an immutable creation-time durable fan-out profile, not a retention period. `spec.expiry` declares an initial RFC3339 timestamp and append-only extension records (`extendedTo`, `reason`, `requestedBy`, `recordedAt`); declared requester/time fields require Kubernetes audit-log correlation for authenticated provenance. SCIM input is rejected.

`GrafanaK6Project`, `GrafanaSyntheticMonitoring` and `GrafanaStackLadder` have separate XRDs. Their platform policy and evidence boundaries are in [Governance](../governance.md).
