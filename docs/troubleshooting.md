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

**Cause.** This is intentional in the default mode, not a bug. Unless the request was armed with
an authorized `spec.lifecycle.externalResources: Delete`, pruning a `GrafanaCloudStackRequest`
uses `Retain`: the Kubernetes composite and composed objects disappear, while the Stack, credential
resources, and generated documents are orphaned. Stack-local content is also retain/orphan because
deleting the Stack destroys it. `Delete` is rejected unless the request namespace, name, UID, and
immutable profile exactly match a platform-owned `deletionAuthorizations` entry; the list is empty by default.

**Diagnosis.** Inspect the request status before assuming the intent was accepted:

```bash
kubectl get grafanacloudstackrequest -n grafana-vending <slug> -o yaml
```

`status.deletionArmed=true` confirms accepted intent. Do not remove the request until
`status.deletionReady=true`; that condition means observed provider state reports
`deleteProtection=false` and ESO has finalized and successfully synced every enabled credential
PushSecret at its current generation. Arming itself never deletes anything.

**Fix.** If destruction is approved, follow the [decommission runbook in the project
README](https://github.com/rknightion/grafana-cloud-vending-machine#decommission-runbook). It uses
three reviewed stages: first arm Delete and wait for readiness; then remove dependent access claims,
merge or sync, and wait until their Kubernetes objects and finalizers are gone while the Stack still
exists; finally remove the request. Armed Delete affects only the Stack, administrator service
account/token, telemetry access policy/token, and administrator/telemetry credential documents. AWS
Secrets Manager uses a 30-day recovery window by default for deleted `PushSecret` documents, and
another backend must be checked for Delete support.

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
