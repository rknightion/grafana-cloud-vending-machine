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
your own request manifests. See [Configuration](configuration.md) for platform policy and
[Request Schema Reference](reference/request-schema.md) for the per-API fields, and [Getting started](getting-started.md)
for the warning at the top of that flow.

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

**Fix.** If destruction is approved, follow the [decommission runbook](governance.md#decommission-runbook). It uses
three reviewed stages: first arm Delete and wait for readiness; then remove dependent access claims,
merge or sync, and wait until their Kubernetes objects and finalizers are gone while the Stack still
exists; finally remove the request. Armed Delete affects only the Stack, administrator service
account/token, telemetry access policy/token, and administrator/telemetry credential documents. AWS
Secrets Manager uses a 30-day recovery window by default for deleted `PushSecret` documents, and
another backend must be checked for Delete support.

## A stack consumer never produces a token

**Cause.** A `GrafanaStackConsumer` starts with one non-creating managed
`Stack` observer. It does not render the AccessPolicy, rotating token, or
PushSecret until that observer reports a positive `status.atProvider.id`. This
can fail because the selected platform profile does not authorize the exact
slug and region, its organization ProviderConfig cannot observe the stack, the
stack does not exist, or management policies were disabled on the provider.

**Diagnosis.** Inspect the consumer and its observer, then check the provider
runtime arguments and environment. Provider v2.14.0 defaults
`--enable-management-policies` to true; disabling it makes an `Observe` policy
invalid rather than safe.

```bash
kubectl get grafanastackconsumer -n <namespace> <profile> -o yaml
kubectl get stacks.cloud.grafana.m.crossplane.io -n <namespace>
kubectl get deployment -n crossplane-system -l pkg.crossplane.io/provider=provider-grafana -o yaml
```

**Fix.** Correct the platform-owned profile or provider configuration, then
wait for the observer's provider-assigned ID. Do not add a request-supplied ID,
change the realm to an organization, or create an ordinary Stack as a
workaround. The source semantics are documented in
[existing-stack ownership and cross-cluster consumption](migration-1.0.md#existing-stack-ownership-and-cross-cluster-consumption).

## Credential documents conflict during a stack handoff

**Cause.** A source and target full-stack Composition, or a source stack
credential and a consumer credential, are writing the same remote secret path
with `PushSecret` `Replace` behavior. They can overwrite each other's document
even while both Kubernetes resources appear healthy.

**Fix.** Freeze promotion and identify every writer and remote key. Keep
consumer credentials on their distinct profile-owned output path. Retire the
source writer and wait for its finalizer before starting a target writer for an
existing path; otherwise move consumers to a new verified path first. Follow
the ordered [full ownership transfer](migration-1.0.md#transfer-full-stack-ownership-between-clusters) procedure before resuming reconciliation.

## Running the validation gate

Run the complete local gate before opening a change:

```bash
just check
```

It performs public-release scanning, Go formatting and module consistency checks, race-enabled
unit tests with coverage, `go vet`, YAML syntax parsing, and Kustomize rendering for the
platform, AWS examples, and comprehensive catalog base. See
[Security → public-release scanning](security.md#public-release-scanning) for what the scan
itself covers.

## Validation

Run the complete local gate:

~~~bash
just check
~~~

It performs:

- public-release scanning of the working tree and reachable Git history for source identifiers, credential prefixes, private keys, local paths, private endpoints, account IDs, JWT-like values, Kubernetes Secret manifests, sensitive file names, and tracked archives/key containers;
- Go formatting and module consistency checks;
- race-enabled unit tests with coverage;
- go vet;
- YAML syntax parsing;
- Kustomize rendering for the platform, AWS examples, and comprehensive catalog base.

The unit tests pin the desired-resource contracts, deterministic external identities, gating behavior for observed IDs, rotating-token parameters, least-privilege scopes, output-document shape, reconciliation modes, OAuth and SAML rendering, SSO Secret references, incident resources, the pinned Role initializer workaround, Team membership, administration and preferences, three-segment custom and fixed-role assignments, whole-target content ACLs, and safe composite status.

Before making the repository public, also review repository settings, issues, workflow logs, releases, packages, and commit-author metadata. The automated history scan covers reachable local Git objects, but it cannot inspect deleted remote refs or external artifacts that are no longer present in a checkout.
## Known limitations

- The upstream Synthetic Monitoring Installation resource can report Ready/Synced without configuring the product. This module therefore requires an independently observed disabled Check through the derived credential. The full bootstrap chain and the disabled verifier's zero-execution behavior still require deployed validation; no live verification is claimed here.

- The Grafana provider is experimental and may lag the Terraform provider.
- Provider schemas and Grafana APIs may expose fields that do not round-trip cleanly; test drift rather than assuming.
- The pinned provider requires the optional Role autoIncrementVersion field to be present because of an initializer defect; this reference pins it to false and omits version.
- The pinned v2.14.0 release is digest-pinned and signed by the provider's tag workflow; future upgrades must move the tag-scoped certificate identity and both digest occurrences together.
- AccessPolicy realm is a Block List at the pinned build where v2.13.0 generated a Block Set. This reference emits exactly one realm entry, so element ordering is not load-bearing here; a multi-realm policy would need to treat order as significant.
- Stack status gained per-service allowlist URL fields at the pinned build. They are endpoint references for retrieving source IP addresses to allow, not a means of restricting inbound access to a stack.
- The reference has no one-command destructive workflow; the authorized Delete path still requires
  three reviewed stages and a readiness wait.
- It does not provision Kubernetes, AWS infrastructure, DNS, identity providers, or incident relays.
- It uses generic starter dashboards rather than a full observability content library.
- Report, Enterprise, OnCall, plugin, and other resources require the relevant Grafana Cloud capabilities.
- SSO profiles are examples and must be replaced with reviewed identity settings.
- Built-in Viewer, Editor, and Admin definitions cannot be globally rewritten through the current provider; use SSO mapping, fixed/custom roles, and content ACLs.
- Fixed role UIDs and available RBAC actions vary by Grafana version, edition, and entitlement; inventory and test them before assignment.
- FolderPermission, DashboardPermission, RoleAssignment, NotificationPolicy, and similar whole-set APIs need exactly one declarative owner per external target.
- Crossplane readiness reports only the child resources currently desired; an optional disabled domain is not health-checked.
- Secret-store publication is eventually consistent with the configured ESO refresh interval.
- A successful render or unit test does not prove acceptance by a specific Grafana Cloud region or account. Use a disposable stack for live acceptance.
- This is Grafana Cloud only; self-managed Grafana feature toggles and deployment variants are not
  supported.
- Adaptive Metrics, Logs, Traces, and Profiles are deliberately out of scope. Use the UI and
  ticket-based routes until a separately designed module is adopted.
- Datasource LBAC requires Grafana 11.5 or later, a Cloud or Enterprise entitlement, basic auth,
  and governance of inherited/fixed/independently managed grants that can bypass its rules.
- Fleet usage groups are UI-only and Advanced-tier. Agent Observability plugin availability and
  permissions, and Assistant terms, remain environment prerequisites.


## Next steps

- [Architecture](architecture.md) — the full reconciliation-mode table.
- [Secrets](secrets.md) — the credential flow this page's secret-related problems reference.
- [FAQ](faq.md) — shorter, less diagnostic questions.
