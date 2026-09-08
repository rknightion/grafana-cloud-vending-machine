---
title: Migrating to 1.0
description: Schema-led migration from the unreleased pre-breaking checkout to the 1.0 API surface
---

# Migrating to 1.0

This guide compares the explicit pre-breaking checkout baseline
`85c4344a5146eea98b4bfa9fb1c110858cd1f152` with the 1.0 candidate's
`platform/apis/` schemas. That baseline is an unreleased 0.x checkout, not a
`v0.1.0` tag or a published release: neither exists. The parent of the first
breaking commit already contains the multi-organization schema seam, so this
comparison deliberately includes it rather than treating that seam as a
compatible starting point.

The candidate is held and is not admission-ready. Real API-server checks found
two unresolved schema defects: the agent-observability CRD's `collectionRefs`
set-list items lack the required atomic map declaration, and an explicit
`spec.scim: null` is admitted because CEL treats null as absent. The migration
steps below describe the intended adoption sequence after those defects are
resolved and the complete admission gate passes. Do not apply this candidate
as a validated release.

The guide uses these outcomes precisely:

- **Rejected** means the current schema or request renderer refuses a create,
  apply, or reconciliation until the request is changed.
- **Silently changed** means the old schema accepted an input but pruned it;
  it never represented an active request setting.
- **Unaffected** means an existing object can retain its old shape. An added
  API kind has no existing object to migrate.

## Migration order

1. Inventory `GrafanaCloudStackRequest` objects and the platform organization
   registry before applying the new XRDs.
2. Choose one registered organization key for every existing stack request and
   add it in a reviewed change. The key must permit that request's existing
   region and usage.
3. Remove any attempted `spec.scim` input. Use the supported external-group
   mapping APIs instead.
4. Decide whether the old stack must be replaced to gain retention or expiry.
   Those choices cannot be added after creation.
5. Apply the new XRDs, then re-apply each migrated request and check the
   resulting output-document location. It is now organization-segmented.

This inert shape illustrates the required addition; `example-primary` is a
placeholder registry key, not a deployable identity.

```yaml
apiVersion: platform.example.org/v1beta1
kind: GrafanaCloudStackRequest
metadata:
  name: examplestack01
  namespace: example-vending
spec:
  displayName: Example stack
  slug: examplestack01
  region: prod-us-central-0
  usage: development
  organization: example-primary
```

## Existing request kinds

The baseline already exposed `GrafanaCloudStackRequest`,
`GrafanaCustomRoleBinding`, `GrafanaTeamAccess`, and
`GrafanaContentAccessPolicy`. The custom-role definition moved from the old
combined stack file into its own file, but its kind and schema are unchanged.
`GrafanaTeamAccess` and `GrafanaContentAccessPolicy` are also unchanged, so
their existing requests are **unaffected**.

### Stack request changes

| Change | Before shape | 1.0 shape and adopter action | Existing 0.x request outcome |
| --- | --- | --- | --- |
| Organization | `spec` required `displayName`, `slug`, `region`, and `usage`. | `spec.organization` is additionally required, is 1-63 lowercase DNS-label characters, and is immutable. Add a permitted registry key before the next reconciliation; changing that key later requires a replacement request. | **Rejected** while absent: the current schema requires it and the renderer also refuses a stack claim without it. |
| Output identity consequence | Output identity was based on `{prefix}/{region}/{usage}/{slug}`. | The organization requirement makes the output identity `{prefix}/{organization}/{usage}/{slug}`. Review external-secret access and consumers after migration. | **Unaffected** until the reviewed organization migration is applied; that deliberate update then selects a different output location. |
| Product switches | No `products` object. | Optional `spec.products.applicationObservability`, `kubernetesObservability`, and `databaseObservability`, each defaulting to `false`. Set only the switches the platform has approved. | **Unaffected** when omitted; all three default to disabled. |
| SCIM compatibility input | No declared `spec.scim`; an unknown field was pruned by the structural schema. | `spec.scim` is preserved for the intended presence rejection, `!has(self.scim)`. It is not supported configuration. Remove it and use external-group mappings. | The old attempted input was **silently changed** by pruning. Non-null input is **rejected**. Explicit null is currently admitted: this is an unresolved admission defect, not supported behavior. |
| Retention | No retention selection. | Optional `spec.retention`, but if selected it requires `class`. Its presence cannot be added or removed after creation and the value is immutable. Choose it only for a newly created replacement request. | **Unaffected** when omitted. An existing request cannot be updated to add it; that update is **rejected**. |
| Expiry | No expiry selection. | Optional `spec.expiry`, but if selected it requires RFC3339 `expiresAt`. Presence is creation-time-only; `expiresAt` is immutable. `extensions` defaults to `[]`, requires `extendedTo`, `reason`, `requestedBy`, and `recordedAt` per entry, and is append-only. Create a replacement request if an existing stack needs expiry. | **Unaffected** when omitted. Adding or removing it, changing the original deadline, or removing/changing an extension is **rejected**. |

The creation-time-only rules are intentionally stricter than ordinary
immutability. `retention` and `expiry` cannot be selected later, so a failed
attempt cannot be repaired with a second update. Plan a replacement request,
including any required data or credential handoff, before making either
selection.

## New request APIs

These ten kinds did not exist in the baseline. They do not alter an existing
0.x object and are therefore **unaffected** until an adopter deliberately
creates one. The required shape below is the create contract; nested required
fields apply only when their optional enclosing list or object is supplied.

| New kind | 1.0 create contract | Adopter action and existing-object outcome |
| --- | --- | --- |
| `GrafanaStackInventory` | `stackRef.name` is required. Each optional `declared` entry requires `apiVersion` and `kind`, and must identify an object with `externalName`, `name`, `email`, or `login`. | Create only for reviewed, observe-only inventory. No legacy kind exists, so existing requests are **unaffected**. |
| `GrafanaFleetPipelines` | `stackRef.name` and `profile` are required. | Select a platform-owned profile; it is a new request and existing requests are **unaffected**. |
| `GrafanaAlertingBundle` | `stackRef.name` and `provenance` are required. Optional contact points, mute timings, templates, inhibition rules, and rule groups have their own required names and payload fields. | Choose `enforced` or `createOnly` provenance before creating it. Existing requests are **unaffected**. |
| `GrafanaAgentObservability` | `stackRef.name` is required. Optional guard and workload lists have required identities and, for evaluators, `evaluatorId`, `kind`, `config`, `outputKeys`, and `version`. | Create a separate opt-in request; do not expect a stack request to infer workload rules. Existing requests are **unaffected**. |
| `GrafanaAssistantGovernance` | `stackRef.name` and `termsAcceptance.accepted` are required. Rules and MCP servers default to empty lists; supplied entries require their names and scopes. | Record acceptance, then create this separate request. Existing requests are **unaffected**. |
| `GrafanaDatasourceAccess` | `stackRef.name`, `datasource.uid`, `datasource.name`, `datasource.type`, `datasource.authMode`, and `teams` are required. `metadata.name` must equal `datasource.uid`; the UID and stack reference are immutable. | Create one namespace-local owner for a datasource. The immutability rule rejects later UID or stack-reference changes, but there is no 0.x object to migrate. |
| `GrafanaProvisioningRepository` | `stackRef.name` and `repository.uid`, `title`, `url`, `branch`, `path`, and `connectionRef.name` are required. A `dashboard` field is rejected. | Create it for a Git-provisioned subtree and keep classic Dashboard ownership on the stack request. Existing requests are **unaffected**. |
| `GrafanaK6Project` | `stackRef.name`, `grafanaUser`, and `allowedLoadZones` are required. | Create a dedicated project request after the referenced stack is ready. Existing requests are **unaffected**. |
| `GrafanaSyntheticMonitoring` | `stackRef.name` is required and `metadata.name` must equal it. Each supplied check requires `name`, `type`, `target`, `frequencySeconds`, and `probeNames`; HTTP checks require only `http.method`, and browser checks require only `browser.script`. The stack reference is immutable. | Create a separate check-set owner. Invalid check-mode combinations and later stack-reference changes are **rejected**; existing requests are **unaffected**. |
| `GrafanaStackLadder` | `organization`, `region`, `promotionDirection`, `rungs`, and `repository` are required. A repository requires `url` and `path`; each rung requires `name`, `slug`, `displayName`, `usage`, `branch`, and `connectionRef.name`. Organization, region, rung names/order, and rung slugs are immutable. | Define the complete ladder at creation. Later changes to those identities are **rejected** and require a separately reviewed migration; existing requests are **unaffected**. |

## What is newly required, immutable, and optional

The distinctions matter during adoption:

- **Newly required on the existing stack kind:** `spec.organization`. It is
  also immutable, so validate the selected registry key before applying it.
- **Newly creation-time-only on the existing stack kind:** the presence of
  `spec.retention` and `spec.expiry`. Within them, `retention.class` and
  `expiry.expiresAt` are immutable; expiry extension records can only be
  appended.
- **Newly optional on the existing stack kind:** `spec.products` and its three
  false-defaulted switches. Leaving them absent does not enable a product.
- **Newly immutable on newly introduced kinds:** datasource UID and stack
  reference, synthetic-monitoring stack reference, and ladder organization,
  region, rung identities, and rung order. These constrain updates only after
  the adopter has chosen to create that new kind.

## Verification after migration

Use the current XRDs as the contract, then check the request rather than only
the apply exit code:

```bash
kubectl apply --dry-run=server -f migrated-stack.yaml
kubectl get grafanacloudstackrequest -n example-vending examplestack01 -o yaml
```

Confirm that the selected organization resolves in the platform registry, that
the rendered output location uses the organization segment, and that no
rejected compatibility or creation-time-only change remains in the request.
Keep all examples outside the live-request directory until that review is
complete.

## Adopting existing external resources

Do not point this platform at existing stacks casually. Adoption is a change of controller ownership, not just a manifest deployment.

A safe adoption rehearsal should:

1. render and apply the `GrafanaStackInventory` example for the existing stack, then review its `status.declared`, `status.observedAndManaged`, and `status.observedButUnmanaged` output;
2. back up the existing configuration through the supported Grafana APIs or its current IaC state;
3. render the request and inspect every desired managed resource before applying;
4. begin with create-only or observe-only modes where available;
5. verify that the Stack external name resolves to the intended existing identity;
6. confirm no unrelated service account, token, SSO provider, plugin, role, or dashboard would be claimed;
7. hand over one resource family at a time;
8. retain a tested rollback that removes Kubernetes ownership without deleting external resources.

The inert `examples/catalog/stack-inventory` catalog is the migration entry point. Copy and adapt it
into the deliberate live-request path only after the referenced stack and its per-stack
`ProviderConfig` are healthy. It composes only the provider's observe-only data sources, so it cannot
create, update, or delete Grafana objects. Use its observed lists to identify out-of-band objects
before adding declarations or moving a resource family into an owning Composition.

The provider imports external objects through the crossplane.io/external-name annotation. This reference emits an identity only when it can derive the provider's import key without consulting a live environment:

| Resource | Deterministic external name |
| --- | --- |
| Stack | request slug |
| Baseline Folder | declared Folder UID |
| Baseline Dashboard | UID embedded in the declared dashboard JSON |
| Custom Role | declared role UID |
| Whole-set RoleAssignment | declared role UID |
| RoleAssignmentItem | declared role UID, literal actor type, and the observed bare Team ID: roleUID:team:teamID |
| FolderPermission or DashboardPermission with a direct UID target | declared target UID |

Do not infer the remaining import keys from Kubernetes names, display names, list order, or another resource's UID. Inventory them from the existing managed resource annotation and provider status or from the supported Grafana inventory API. The pinned provider uses keys such as stackSlug:serviceAccountID for StackServiceAccount, region:policyID for AccessPolicy, orgID:teamID for Team, region:tokenID for AccessPolicyRotatingToken, and orgID for OrganizationPreferences. A rotating service-account token and any resource without a documented importer must be recorded verbatim and rehearsed against the pinned provider; do not manufacture an ID. Whole-set content permissions that target another managed resource by reference also require the resolved target UID to be inventoried.

For a non-destructive orphan-and-adopt transition:

1. pause automated promotion and confirm Delete is absent from every external managed resource involved;
2. export a mapping of logical composed-resource name, kind, external-name annotation, relevant non-secret status IDs, and external object URL or UID;
3. render the replacement Composition and compare that mapping before applying it;
4. let this function supply deterministic identities, and add provider-assigned identities through a reviewed adoption field or dedicated migration Composition;
5. move one resource family at a time, requiring the original identity plus Ready and Synced before continuing;
6. stop on any Create attempt for an inventoried object, identity change, duplicate external object, or deletion event;
7. keep provider-assigned adoption inputs durable until the standard Composition can preserve the same identity, then remove migration machinery only after another steady-state check;
8. resume automated promotion after the complete inventory matches and rollback has been tested.

Extend the request API or use a dedicated migration Composition for provider-assigned inputs. Do not patch generated managed resources by hand: the Composition will overwrite the patch and can turn an adoption into an attempted create.
