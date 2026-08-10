---
title: Troubleshooting
description: Common problems vending or reconciling a Grafana Cloud stack, and how to diagnose them
---

# Troubleshooting

## Request never becomes Ready

**Cause.** `GrafanaCloudStackRequest` becomes `Ready` only when every currently desired composed
resource reports `Ready`. Rotating-token resources (`StackServiceAccountRotatingToken`,
`AccessPolicyRotatingToken`) are intentionally rendered one reconciliation *after* Grafana
reports the parent service-account or access-policy ID — a brand-new request is expected to take
at least two reconciliation passes.

**Diagnosis.**

```bash
kubectl describe grafanacloudstackrequest -n grafana-vending <slug>
kubectl get managed -n grafana-vending
```

Look for any composed resource stuck `Synced=False` — its condition message names the upstream
API error.

## Two controllers fighting over the same resource

**Cause.** Argo CD's default label-based resource tracking claims Crossplane-generated children
as its own, because it inherits ownership by label rather than annotation. This produces a loop
where Argo CD and Crossplane each try to reconcile the same object toward different desired
state.

**Fix.** Set `application.resourceTrackingMethod: annotation` in the Argo CD `argocd-cm`
ConfigMap, as shipped in `deploy/argocd/argocd-values.yaml`. See
[Installation → Argo CD installation](installation.md#argo-cd-installation).

## Request stuck `Progressing` in the Argo CD UI

**Cause.** The default Argo CD health check has no opinion on Crossplane `Composition`,
`CompositeResourceDefinition`, `ProviderConfig`, or the `platform.example.org/*` custom
resources, so it may report a state that does not match actual readiness.

**Fix.** Apply the `resource.customizations` health Lua scripts from
`deploy/argocd/argocd-values.yaml`. They teach Argo CD to read Crossplane's own `Synced`/`Ready`
conditions and treat a request as `Progressing` until its composed resources are ready, `Healthy`
once they are, and `Degraded` on a `Synced=False` condition.

## Rendered request has the wrong API group

**Cause.** Every XRD, Composition, and example in this repository ships with the placeholder API
group `platform.example.org`. If a fork has not replaced it consistently, some manifests apply
against the placeholder group while others reference the real one, and the composite resource
never matches its Composition.

**Fix.** Replace `platform.example.org` in every XRD `spec.group`, every Composition
`spec.compositeTypeRef.apiVersion`, the stack Composition input `apiVersion`, the examples, and
your own request manifests. See [Configuration](configuration.md) for the complete list of
places it appears, and [Getting started](getting-started.md) for the warning at the top of that
flow.

## `PushSecret` fails to create an AWS Secrets Manager target

**Cause.** ESO 2.7.0 and 2.8.0 carry an open regression
([external-secrets/external-secrets#6593](https://github.com/external-secrets/external-secrets/issues/6593))
that sends an empty replica-region request when creating a new AWS Secrets Manager `PushSecret`
target, which AWS rejects.

**Fix.** Stay on ESO Helm chart 2.6.0 until a fixed release exists, and prove creation of a
brand-new remote secret against the fixed release before removing the pin. Do not add a replica
region merely to work around the symptom — see the pinned-versions note in
[Getting started](getting-started.md#pinned-versions).

## SSO profile Secret never syncs

**Cause.** `deploy/aws/optional-profile-secrets.yaml` is deliberately excluded from
`deploy/aws/kustomization.yaml`. Applying it before the corresponding remote secret and profile
exist leaves the `ExternalSecret` unable to resolve its `remoteRef`, and it never syncs.

**Fix.** Confirm the remote secret exists at the profile's path
(`/platform/grafana-cloud/profiles/<profile-name>`) before applying
`optional-profile-secrets.yaml`. See [Secrets → SSO and incident profile secrets](secrets.md#sso-and-incident-profile-secrets).

## An administrator's SSO change keeps getting reverted

**Cause.** `spec.sso.mode: enforced` makes Crossplane the sole owner of OAuth/SAML settings —
any UI edit is treated as drift and repaired on the next reconciliation.

**Fix.** This is expected behaviour for `enforced` mode. If stack administrators should own later
SSO edits, move the request to `createOnly` — see
[Architecture → reconciliation and out-of-band changes](architecture.md#reconciliation-and-out-of-band-changes)
for the complete mode table, and [SSO](sso.md) for the handoff semantics.

## A deleted request left the stack and tokens behind

**Cause.** This is intentional, not a bug. Pruning a `GrafanaCloudStackRequest` deletes the
Kubernetes composite and composed objects, but the `Stack` has delete protection enabled, no
composed resource carries the `Delete` management policy, rotating tokens set
`deleteOnDestroy: false`, and `PushSecret` uses `deletionPolicy: None`. External Grafana and
secret-manager objects are orphaned, not destroyed.

**Fix.** There is no fix — this is the designed non-destructive lifecycle. To actually destroy a
stack, follow the decommission runbook in the project
[README](https://github.com/rknightion/grafana-cloud-vending-machine#decommission-runbook), which
is a separately reviewed operation, not something Git deletion alone triggers. See
[Security → non-destructive lifecycle](security.md#non-destructive-lifecycle).

## Running the validation gate

Run the complete local gate before opening a change:

```bash
./scripts/validate.sh
```

It performs public-release scanning, Go formatting and module consistency checks, race-enabled
unit tests with coverage, `go vet`, YAML syntax parsing, and Kustomize rendering for the
platform, AWS examples, and comprehensive catalog base. See
[Security → public-release scanning](security.md#public-release-scanning) for what the scan
itself covers.

## Next steps

- [Architecture](architecture.md) — the full reconciliation-mode table.
- [Secrets](secrets.md) — the credential flow this page's secret-related problems reference.
- [FAQ](faq.md) — shorter, less diagnostic questions.
