---
title: Request Schema Reference
description: The four CompositeResourceDefinitions this repository ships, their kinds, short names, and CRD identity
---

# Request Schema Reference

Field-by-field defaults and descriptions live on [Configuration](../configuration.md). This page
is the CRD-identity reference: what kind maps to what plural/short name, and where each type is
defined.

## CompositeResourceDefinitions

All four are `apiextensions.crossplane.io/v2`, `scope: Namespaced`, group
`platform.example.org` (a placeholder — see [Configuration](../configuration.md)), version
`v1beta1`, `defaultCompositionUpdatePolicy: Automatic`. Defined in
`platform/apis/v1beta1.yaml` and `platform/apis/access-v1beta1.yaml`.

| Kind | Plural | Short name | Enforced Composition | Required top-level fields |
| --- | --- | --- | --- | --- |
| `GrafanaCloudStackRequest` | `grafanacloudstackrequests` | `gcstackrequest` | `grafana-cloud-stack-request-v1beta1` | `displayName`, `slug`, `region`, `usage` |
| `GrafanaCustomRoleBinding` | `grafanacustomrolebindings` | `gcrole` | `grafana-custom-role-binding-v1beta1` | `stackRef`, `team`, `role` |
| `GrafanaTeamAccess` | `grafanateamaccesses` | `gcteamaccess` | `grafana-team-access-v1beta1` | `stackRef`, `team` |
| `GrafanaContentAccessPolicy` | `grafanacontentaccesspolicies` | `gccontentaccess` | `grafana-content-access-policy-v1beta1` | `stackRef`, `target`, `permissions` |

Every Composition is `mode: Pipeline` with a single step calling the `function-grafana-vending`
composition function — there is no templating layer to inspect separately from the function's Go
source (`platform/function/fn.go`, `access.go`, `plugins.go`, `roles.go`).

## Quick lookups by shortName

```bash
kubectl get gcstackrequest -A
kubectl get gcrole -A
kubectl get gcteamaccess -A
kubectl get gccontentaccess -A
```

## Status conditions

Every kind reports the standard Crossplane composite conditions (`Synced`, `Ready`) plus, for
`GrafanaCloudStackRequest`, the additional `status` fields documented on
[Configuration](../configuration.md#status-fields): `outputSecretPath`, `telemetrySecretPath`,
`stack.id`, `stack.url`.

## Next steps

- [Configuration](../configuration.md) — every `spec` field, its default, and what it does.
- [Catalog Reference](catalog.md) — worked examples for each kind.
