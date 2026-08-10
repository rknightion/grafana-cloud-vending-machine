---
title: Architecture
description: The three-controller split between Argo CD, Crossplane, and External Secrets Operator, reconciliation ownership, and where state lives
---

# Architecture

```mermaid
flowchart LR
    Git[Git request] --> Argo[Argo CD]
    Argo --> XR[GrafanaCloudStackRequest]
    XR --> XP[Crossplane composition]
    XP --> Cloud[Grafana Cloud resources]
    AWSIn[AWS Secrets Manager\norganization and profile credentials] --> ESO[External Secrets Operator]
    ESO --> K8sIn[Kubernetes Secrets]
    K8sIn --> XP
    Cloud --> Generated[Kubernetes connection Secrets]
    Generated --> ESO
    ESO --> AWSOut[AWS Secrets Manager\nper-stack outputs]
```

There are four separate ownership layers:

1. **Argo CD** owns controller installation, platform definitions, environment configuration, and
   request objects stored in Git.
2. **Crossplane** owns the composed Kubernetes managed resources and continuously reconciles
   their external Grafana objects.
3. **External Secrets Operator (ESO)** owns materialization of secret inputs and publication of
   generated secret outputs.
4. **Grafana Cloud** remains the external system of record, observed and corrected by the Grafana
   provider.

**Argo CD must not also declare the managed resources emitted by the Composition.** That would
give two controllers ownership of the same Kubernetes objects. This is why
`application.resourceTrackingMethod: annotation` and the Crossplane-aware health customizations
in `deploy/argocd/argocd-values.yaml` matter — see [Installation](installation.md).

## The vending API

The primary API is `GrafanaCloudStackRequest`. It is namespaced so teams or environments can be
separated with Kubernetes namespaces and RBAC. Three access APIs — `GrafanaCustomRoleBinding`,
`GrafanaTeamAccess`, and `GrafanaContentAccessPolicy` — sit beside it; see
[Configuration](configuration.md) for every field.

The XRD uses `defaultCompositionUpdatePolicy: Automatic` and an `enforcedCompositionRef`.
Existing requests therefore move to the latest Composition revision automatically after a
platform update. Treat an XRD or function change like a production API release: render it,
inspect the desired-resource diff, and roll it through a non-production request first.

## How a request becomes managed resources

Each of the four kinds is backed by a single-step Crossplane `Composition` in `Pipeline` mode
that calls the `function-grafana-vending` composition function (a Go program built from
`platform/function/`). The function reads the request spec plus the platform-owned
`GrafanaVendingConfig` Composition input and renders the complete set of desired Kubernetes
managed resources — there is no templating language involved, the rendering logic is Go code
covered by `platform/function/fn_test.go`.

## Reconciliation and out-of-band changes

Crossplane providers poll the external APIs and compare observed state with desired state. The
exact delay is the provider poll interval plus API and controller latency — it is not an
immediate webhook response.

An administrator's out-of-band change (through the Grafana UI, say) is repaired only when the
affected field's reconciliation mode says so:

| Resource or field | Mode | Effect of an out-of-band edit |
| --- | --- | --- |
| Stack name, region, labels, readiness flags | managed `forProvider` fields | Crossplane attempts to restore the value in the request |
| Stack deletion | `deleteProtection` plus no `Delete` policy | Normal controller-driven deletion is refused |
| Baseline dashboard JSON | `createOnly` | Initial JSON is supplied through `initProvider`; later UI edits are preserved |
| Baseline dashboard JSON | `enforced` | UI edits are detected and restored from Git-rendered desired state |
| Folder title and dashboard folder relationship | managed `forProvider` fields | Drift is restored even when dashboard JSON is create-only |
| Home dashboard UID | `createOnly` | Initial choice is set; a later administrator change is preserved |
| Home dashboard UID | `enforced` | A later administrator change is restored |
| SSO settings | `enforced` | Crossplane owns OAuth/SAML settings and repairs UI drift |
| SSO settings | `createOnly` | Crossplane initializes settings but does not own later changes |
| SSO settings | `observeOnly` | Crossplane observes the named provider and never creates or updates it |
| SSO settings | `disabled` | No SSO managed resource is desired |
| Plugins | listed | Crossplane reconciles listed installations; omitted plugins are not adopted |
| Custom roles, assignments, team sync | present in a binding | Crossplane repairs drift in the managed fields |
| Team direct members and preferences | present in `GrafanaTeamAccess` | Crossplane restores the declared direct members and preferences; external sync members are ignored when configured |
| Fixed/custom `RoleAssignmentItem` | present in `GrafanaTeamAccess` | Crossplane restores the role-to-team assignment without owning other actors assigned to that role |
| Folder or dashboard ACL | present in `GrafanaContentAccessPolicy` | Crossplane restores the complete declared ACL; omitted entries are removed by Grafana's whole-set permission API |
| OnCall outgoing-webhook data | `createOnly` | Initial generic payload is set; later UI template edits are preserved |
| OnCall outgoing-webhook data | `enforced` | Later UI template edits are restored from the Composition |
| Alerting contact-point payload | enabled | Crossplane always restores the relay payload and Secret-backed authorization contract |

Changing SSO from `enforced` to `createOnly` moves `oauth2Settings`/`samlSettings` from
`forProvider` to `initProvider` while retaining a stable external name for the provider — the
supported handoff from platform ownership to administrator ownership. `observeOnly` removes write
authority. `disabled` removes the managed-resource object from the Composition; because `Delete`
is not permitted, the external SSO configuration remains but is no longer observed. Changing the
identity type itself (e.g. `generic_oauth` to `saml`) is an identity-provider migration, not a
routine mode toggle — plan a tested login and rollback path.

Argo CD should ignore Crossplane-generated resource churn rather than carrying broad ignore rules
for the request itself. If the request in Git changes, Argo CD applies the request; the
Composition then computes the resulting managed-resource change.

## Baseline and optional resources

The baseline creates three portable starting points occupying the same resource slots as a
traditional billing/usage, endpoints, and home-dashboard bundle. They are intentionally simple
and generic; no source-specific dashboard JSON is copied or implied. Keep service-owned
dashboards outside the stack identity API so an ordinary content release cannot disturb stack
identity or credentials.

Everything else is a separately owned domain — alerting rule groups and notification policy,
data sources, cloud integrations, SLOs, Synthetic Monitoring, OnCall schedules and escalation
chains, Fleet Management, k6, ML, Asserts, and Assistant are all out of scope for automatic
creation. See the provider-family table in the project
[README](https://github.com/rknightion/grafana-cloud-vending-machine#complete-provider-surface-and-ownership-boundaries)
for the complete ownership map across all 111 managed-resource kinds the provider exposes.

## Where state lives

- **Desired state** lives in Git, as `GrafanaCloudStackRequest` and access-API objects under
  `enabled/`.
- **Composed managed-resource state** lives in Kubernetes, generated by the composition function
  and never hand-edited — a hand-applied patch to a generated managed resource is overwritten on
  the next reconciliation and can turn an intended adoption into an attempted create.
- **External state** lives in Grafana Cloud, observed and corrected by the Grafana provider on
  its poll interval.
- **Credential material** never lives in Git or in a Composition input. It flows through ESO — see
  [Secrets](secrets.md).

## Next steps

- [Configuration](configuration.md) — every request field.
- [Secrets](secrets.md) — the credential rotation model in detail.
- [Security](security.md) — supply-chain verification and the non-destructive lifecycle.
