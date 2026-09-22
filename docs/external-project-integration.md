---
title: External project integration
description: A source-grounded contract for turning a consuming platform project CR into a reviewed Grafana Cloud stack request
---

# External project integration

This page defines the boundary for a consuming platform that wants a project
custom resource (CR) to result in a Grafana Cloud stack request. The project CR
shape below is an illustrative adapter contract. This repository does not
install that CRD, watch its cluster, or run the adapter.

The request contract is the `GrafanaCloudStackRequest` XRD in
[`platform/apis/stack-v1beta1.yaml`](../platform/apis/stack-v1beta1.yaml). It
requires `spec.displayName`, `spec.slug`, `spec.region`, `spec.usage`, and
`spec.organization`. The XRD also requires `metadata.name == spec.slug`, and
keeps `slug`, `usage`, `organization`, and `profile` immutable after creation.

## Trigger boundary

The external-CR trigger belongs to the consuming platform. The intended flow
is:

1. The consuming platform observes its own project CR.
2. Its adapter normalizes the project into a candidate
   `GrafanaCloudStackRequest` and a Kustomize directory.
3. The adapter opens a pull request adding that directory below `enabled/`.
4. A person reviews and merges the pull request.
5. The repository's Argo CD ApplicationSet applies the merged directory. Its
   only input is the top-level `enabled/*` Git generator path, as shown in
   [`deploy/argocd/requests-applicationset.yaml`](../deploy/argocd/requests-applicationset.yaml).

There is no external-project controller, generator, webhook, or second watch
path in this repository. The catalog example in
[`examples/catalog/external-project/`](../examples/catalog/external-project/)
is outside `enabled/` and is inert.

### Alternatives considered

The consuming-platform boundary is deliberate:

- A controller in this repository would encode one consumer's internal CRD
  shape in a public reference. It would also add a runtime image, publish and
  signature workflow, digest pin, RBAC, and a second serialized publishing
  bottleneck beside the composition function.
- An Argo Plugin generator is an HTTP service, so it is a runtime component
  under another name. Its claim would not land in Git, so it would not provide
  a reviewable generated artifact before reconciliation.
- A consuming-platform adapter plus a pull request into `enabled/` keeps the
  trigger with the owner of the project CR, gives the generated claim a Git
  review boundary, and preserves this repository as a portable public
  reference.

The absence of a trigger component here is the public-reference decision, not
an unfinished implementation. The consuming platform owns the adapter and
must adapt its real project CR fields to the normalized contract below.

## Normalized project shape

The adapter consumes this minimal shape. `metadata.name` is constrained by the
adapter to lowercase alphanumeric characters, three to sixteen characters. A
consumer with a different source CR uses an explicit field adapter to produce
the same normalized values; it must not guess a field or silently fall back to
a default.

```yaml
apiVersion: tenancy.example.org/v1alpha1
kind: Project
metadata:
  name: demo
  namespace: platform-projects
spec:
  displayName: Example development project
  environment: development
  organization: example-primary
```

`spec.environment` is the external name for the request's usage class. This
contract permits only the constrained enum `development` or `production`.
That is narrower than the XRD's syntactic lowercase-and-hyphen pattern. The
platform-owned Composition input remains the semantic authority: the selected
organization entry must allow the usage and the selected region, and the
global `allowedUsages` list must allow the usage too. A value such as
`preview`, even if it matches the XRD's syntax, is refused by this adapter.

`spec.organization` is required. The adapter copies the exact value into the
request and treats it as immutable. It must resolve to a platform registry
entry with a provider configuration, an allowed region, and an allowed usage.
There is no organization fallback. A missing, malformed, unknown, or changed
organization stops mapping and produces no request.

## Field mapping

The table distinguishes project-CR values from values fixed by the consuming
platform's request template. The example values in the final column are public
shapes, not source-environment identity.

| Project CR field | Request field | Ownership and rule | Example |
| --- | --- | --- | --- |
| `metadata.name` | `metadata.name` and `spec.slug` | CR-sourced identity segment. Build the slug below; never truncate it. The two request fields must be identical. | `demo` |
| `spec.displayName` | `spec.displayName` | CR-sourced, required, and validated against the XRD's 1 to 128 character string limit. | `Example development project` |
| `spec.environment` | `spec.usage` | CR-sourced through the closed enum `development` or `production`; immutable in the request. | `development` |
| `spec.organization` | `spec.organization` | CR-sourced, required, registry-validated, and immutable in the request. | `example-primary` |
| No project field | `metadata.namespace` | Platform-fixed target namespace selected by the consuming platform's Argo destination. | `grafana-vending` |
| No project field | `spec.region` | Platform-fixed by the organization and usage profile. It must match the XRD region pattern and the registry's `allowedRegions`. | `prod-us-central-0` |
| No project field | `spec.profile` | Platform-fixed configuration profile. The XRD makes it immutable. | `standard` |
| No project field | `spec.lifecycle.externalResources` | Fixed to `Retain` for generated requests. The project CR cannot arm external deletion. | `Retain` |
| No project field | baseline and optional feature fields | Platform-fixed template defaults: baseline dashboards and telemetry access enabled; plugins empty; dashboards and home preference `createOnly`; SSO, reports, and incident integration disabled; product toggles false. | See the worked request |

The template's optional fields are still real request fields. If a consumer
chooses to emit them explicitly, the fixed values are:

- `spec.products.applicationObservability`,
  `spec.products.kubernetesObservability`, and
  `spec.products.databaseObservability`: `false`;
- `spec.baselineDashboards.enabled` and `spec.telemetryAccess.enabled`:
  `true`;
- `spec.plugins`: `[]`;
- `spec.reconciliation.dashboards` and
  `spec.reconciliation.homePreference`: `createOnly`;
- `spec.sso.mode`: `disabled`, with no profile;
- `spec.monthlyReport.enabled`: `false`, with no recipients or reply-to;
- `spec.incidentIntegration.enabled`: `false`, with no profile and
  `templateMode: createOnly`.

`spec.retention` and `spec.expiry` are not project inputs. They are
creation-time platform decisions and are omitted from this minimal contract;
if a consuming platform adds either, it must select the value from a reviewed
platform policy rather than the source CR.

The output identity is deterministic and valid for the actual XRD slug
pattern, which permits lowercase alphanumeric characters only:

```text
normalizedOrganization = remove '-' from spec.organization
slug = normalizedOrganization + spec.environment + metadata.name
metadata.name = slug
```

For the worked shape, `example-primary`, `development`, and `demo` produce
`exampleprimarydevelopmentdemo`. The adapter must refuse the candidate when
the result is shorter than three characters, longer than 32 characters, not
`^[a-z0-9]+$`, or already identifies another request in the target namespace.
It must also refuse a normalization collision between two registered
organizations. Truncating the slug, choosing a different separator, or
silently reusing an existing request would break the API's identity contract.

The adapter does not copy provider IDs, stack URLs, ProviderConfig names,
credential paths, secret references, tokens, or vendor endpoints from a
project CR. Those values are either provider-observed or platform-owned and
are outside this trigger contract.

### Refusal before a pull request

Mapping is fail-closed. The adapter must reject the candidate and leave no
request under `enabled/` when any of these checks fails:

- a required source value is missing, the display name is outside the XRD
  length, or the source identity cannot produce a valid slug;
- `environment` is not `development` or `production`;
- `organization` is absent, malformed, unknown, or not immutable for the
  existing request;
- the platform-fixed region is outside the XRD pattern or is not allowed by
  the selected organization entry;
- the organization entry does not allow the selected usage, or the global
  usage allow-list does not contain it;
- the generated identity collides, exceeds the XRD's 32-character limit, or
  would change an existing `metadata.name` or immutable request field;
- the source CR tries to override a platform-fixed field such as region,
  profile, lifecycle, provider identity, or credential output.

The refusal is a mapping or schema failure, not a vendor refusal. A rejected
candidate must not be retried against Grafana Cloud and must not be represented
as an applied stack request.

## Review, cap, and failure classification

The pull request is the approval seam and the explicit cap. The consuming
platform must not auto-merge generated requests. The default bounded unit is
one candidate project per pull request, so every new stack entering
reconciliation consumes one human approval. The review checks the normalized
field mapping, identity uniqueness, organization and region policy, the fixed
`Retain` lifecycle, and the requested number of pending projects.

This seam bounds how many generated requests can enter the watched Git tree. It
does not bound the number of project CRs that can be created, vendor quotas or
spend, or the number of requests that were already approved and merged. Those
are consuming-platform policy concerns. A queue, budget, or stricter batch
limit may be added by that platform, but it cannot bypass the review seam in
this contract.

The review artifact and the Kubernetes condition chain make templating errors
distinct from vendor refusals without provider logs:

| Failure class | Where it appears | Meaning and action |
| --- | --- | --- |
| Mapping error | Adapter or pull-request check, before merge | The normalized source values cannot produce a valid request. Do not merge or apply anything. |
| Request schema error | API-server admission or Argo apply status, before provider reconciliation | The generated YAML violates the XRD, for example a missing required field or `metadata.name != spec.slug`. Fix the template and rerun the mapping check. |
| Vendor refusal | An applied request's composed child `status.conditions` and Kubernetes events | The request passed mapping and API admission, but a provider-managed child reports its own `Synced=False` or `Ready=False` condition. Investigate that child and its condition message; provider logs are not required for classification. |
| Credential health signal | The applied composite's conditions, with the expected child condition | The GCV-0087 shape makes a configured rotating-token family with a missing child or a child without `Synced=True` force the owning composite to `Ready=False`. It is an observed health signal, not a mapping error and not a custom provider field. |

The standard condition shape is intentionally small:

```yaml
# Applied GrafanaCloudStackRequest, after a valid mapping
status:
  conditions:
    - type: Ready
      status: "False"

# One observed rotating-token child, when it is the failing witness
status:
  conditions:
    - type: Synced
      status: "False"
      reason: CannotUpdateExternalResource
```

The `CannotUpdateExternalResource` reason is an example of a provider
condition reason when emitted, not a reason the adapter invents. Ordinary
provider failures can leave a composite's own readiness less specific, so the
child condition and event remain part of the diagnosis. GCV-0087 specifically
closes the rotating-token visibility gap by making missing or unsynced expected
children visible through composite `Ready=False`; it does not claim that a
generic `Ready=False` is proof of a template error.

## Decommission and project disappearance

Project-CR disappearance is not a deletion signal. The adapter does not remove
the generated Git request, change its lifecycle, or set `Delete` when the
source CR disappears. A generated request remains in Git until a separate
reviewed change removes or changes it.

The request XRD defaults `spec.lifecycle.externalResources` to `Retain`.
Removing an unarmed request from Git therefore orphans external resources. The
only destructive route is the existing owner-authorised decommission path in
[`docs/governance.md`](governance.md#decommission-runbook):

1. A platform owner adds the exact request namespace, name, Kubernetes UID, and
   immutable profile to platform-owned `deletionAuthorizations`.
2. A first reviewed change sets `spec.lifecycle.externalResources: Delete`.
   Wait for `status.deletionArmed=true` and then
   `status.deletionReady=true`. Readiness requires observed
   `deleteProtection=false`, prepared administrator and telemetry rotating
   tokens, and current successful external-secret sync and finalizer state for
   every enabled credential document.
3. A second reviewed change removes dependent `GrafanaCustomRoleBinding`,
   `GrafanaTeamAccess`, and `GrafanaContentAccessPolicy` claims and waits for
   their Kubernetes objects and finalizers to disappear while the request and
   Stack still exist.
4. A third reviewed change removes the request from Git and verifies the
   resulting external deletion. Stack-local content remains retain/orphan by
   design.

The source project CR cannot perform any of these stages. If its disappearance
should eventually lead to decommission, that is a new, separately reviewed
consuming-platform workflow that must still enter the owner-authorised path.

## Scope and evidence

This repository ships the public request schema, the normalized mapping
contract, the review and cap boundary, lifecycle semantics, and the inert
worked example. It ships no external-project controller, generator runtime,
RBAC, credentials, live request, or widened ApplicationSet watch. A rendered
catalog example proves YAML and Kustomize shape only; it does not prove a live
cluster, provider, vendor, or deployment transaction.
