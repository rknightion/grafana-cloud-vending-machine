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

The 1.0 candidate remains held by owner decision. The original admission defects
are historical: the agent-observability `collectionRefs` set-list items now
have the required atomic map declaration, and every XRD installs into the pinned
real API-server harness.

The explicit-null boundary is measured separately. Under the shipped
`nullable: true` schema, Kubernetes persists `spec.scim: null` with the key
present. CEL `has()` does not see that null-valued field, so admission accepts
it. The renderer sees the persisted key, refuses the request and emits no
children. Omitted `scim` stays absent. This reconcile-time fail-closed boundary
was accepted by the owner on 2026-09-08; it is not an unresolved schema repair
or supported SCIM configuration.

The owner holds 1.0 until GCV-0061 lands. Release-please PR #32 remains the
release candidate; feature delivery does not itself authorize merging that PR,
creating a release, or creating a Git tag. The steps below describe migration
to the unreleased current API.

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
| SCIM compatibility input | No declared `spec.scim`; an unknown field was pruned by the structural schema. | `spec.scim` is preserved for the intended presence rejection, `!has(self.scim)`. It is not supported configuration. Remove it and use external-group mappings. | The old attempted input was **silently changed** by pruning. Non-null input is **rejected**. Explicit null is admitted and persisted with its key present, then **rejected at reconcile** with zero children. This is the accepted fail-closed boundary, not supported SCIM configuration. |
| Retention | No retention selection. | Optional `spec.retention`, but if selected it requires `class`. Its presence cannot be added or removed after creation and the value is immutable. Choose it only for a newly created replacement request. | **Unaffected** when omitted. An existing request cannot be updated to add it; that update is **rejected**. |
| Expiry | No expiry selection. | Optional `spec.expiry`, but if selected it requires RFC3339 `expiresAt`. Presence is creation-time-only; `expiresAt` is immutable. `extensions` defaults to `[]`, requires `extendedTo`, `reason`, `requestedBy`, and `recordedAt` per entry, and is append-only. Create a replacement request if an existing stack needs expiry. | **Unaffected** when omitted. Adding or removing it, changing the original deadline, or removing/changing an extension is **rejected**. |

The creation-time-only rules are intentionally stricter than ordinary
immutability. `retention` and `expiry` cannot be selected later, so a failed
attempt cannot be repaired with a second update. Plan a replacement request,
including any required data or credential handoff, before making either
selection.

## New request APIs

The following APIs illustrate additions since the baseline. The complete current
inventory is in [Request schema](reference/request-schema.md). They do not alter an existing
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

## Existing-stack ownership and cross-cluster consumption

This section has two deliberately separate operations. A
`GrafanaStackConsumer` creates a new, profile-owned credential for a stack that
another cluster already owns. It does **not** transfer ownership of the stack
or reuse the source stack's credentials. Moving the `GrafanaCloudStackRequest`
itself transfers ownership and must have no overlapping full-stack writers.

### Existing-slug result

**Design position from pinned source: adopt and reconcile; do not duplicate.**
A newly reconciled `GrafanaCloudStackRequest` renders a managed `Stack` with
`crossplane.io/external-name` set to its slug. The pinned Terraform provider
reads that identity through `GetInstance`, so Crossplane can observe an active
existing stack before it considers its Create branch. A found stack therefore
is adopted into reconciliation; if its desired fields differ, the normal
management policy permits an update. The Terraform resource's direct Create
path also rejects an already-taken slug, so a Create branch that is reached
fails as a conflict rather than making a second stack.

This is a source-derived design position and is **unproven against live
behaviour**. Do not use a production stack to turn it into a test. Validate the
provider version, provider configuration, and rendered external name in a
disposable environment before a production handoff.

The conclusion depends on the pinned provider version and its enabled
management-policy support:

- Provider v2.14.0 pins Crossplane Runtime v2.1.0, Upjet v2.2.0, and Terraform
  Provider Grafana v4.45.1 in its [module definition](https://github.com/grafana/crossplane-provider-grafana/blob/v2.14.0/go.mod#L5-L18).
- Its namespaced Stack controller wires management policies into both the
  connector and reconciler when the feature is enabled
  ([controller](https://github.com/grafana/crossplane-provider-grafana/blob/v2.14.0/internal/controller/namespaced/cloud/stack/zz_controller.go#L44-L64)).
  Provider v2.14.0 defaults `--enable-management-policies` to `true` and also
  accepts `ENABLE_MANAGEMENT_POLICIES`; keep that setting enabled
  ([provider entry point](https://github.com/grafana/crossplane-provider-grafana/blob/v2.14.0/cmd/provider/main.go#L50-L58), [feature activation](https://github.com/grafana/crossplane-provider-grafana/blob/v2.14.0/cmd/provider/main.go#L107-L134)).
- With the feature disabled, Runtime v2.1.0 rejects a non-default policy. With
  it enabled, `Observe` alone has no Create action
  ([validation](https://github.com/crossplane/crossplane-runtime/blob/v2.1.0/pkg/reconciler/managed/policies.go#L148-L170), [action resolution](https://github.com/crossplane/crossplane-runtime/blob/v2.1.0/pkg/reconciler/managed/policies.go#L183-L221)).
- Runtime observes before it reaches Create. In Observe-only mode a missing
  external resource is reported as an error, while Create is entered only when
  `ShouldCreate` is true
  ([observe and missing-resource path](https://github.com/crossplane/crossplane-runtime/blob/v2.1.0/pkg/reconciler/managed/reconciler.go#L1118-L1145), [Create gate](https://github.com/crossplane/crossplane-runtime/blob/v2.1.0/pkg/reconciler/managed/reconciler.go#L1289-L1312)).
  A normal managed Stack updates observed drift when Update is allowed
  ([update gate](https://github.com/crossplane/crossplane-runtime/blob/v2.1.0/pkg/reconciler/managed/reconciler.go#L1451-L1511)).
- The provider explicitly defaults Cloud Stack import identity to the slug,
  uses `ExternalNameAsID`, and disables the name initializer
  ([Cloud configuration](https://github.com/grafana/crossplane-provider-grafana/blob/v2.14.0/config/grafana/cloud.go#L62-L80)).
  Upjet puts that external identity into Terraform parameters, reconstructs a
  missing state with that ID, and refreshes it before reporting ResourceExists
  ([ID parameters](https://github.com/crossplane/upjet/blob/v2.2.0/pkg/controller/external_tfpluginsdk.go#L126-L153),
  [initial state](https://github.com/crossplane/upjet/blob/v2.2.0/pkg/controller/external_tfpluginsdk.go#L242-L306),
  [refresh](https://github.com/crossplane/upjet/blob/v2.2.0/pkg/controller/external_tfpluginsdk.go#L473-L493)).
- The Terraform Stack resource defines the provider-assigned ID and importer
  ([resource schema](https://github.com/grafana/terraform-provider-grafana/blob/v4.45.1/internal/resources/cloud/resource_cloud_stack.go#L67-L92)), reads a stack through its identity
  ([read path](https://github.com/grafana/terraform-provider-grafana/blob/v4.45.1/internal/resources/cloud/resource_cloud_stack.go#L559-L577)), and rejects a taken active slug on Create
  ([Create path](https://github.com/grafana/terraform-provider-grafana/blob/v4.45.1/internal/resources/cloud/resource_cloud_stack.go#L349-L430)).

### Consume a stack without transferring it

Use `GrafanaStackConsumer` only after the platform has approved a profile whose
`providerConfigName`, target slug and region, scopes, consumer name, and output
secret path are all fixed. The request supplies the target slug, region, and
profile; it never supplies a stack ID, provider configuration, scopes, consumer
identity, or output path. Its namespace must equal the profile namespace and its
name must equal the profile name. The stack and profile are immutable.

The consumer first renders the namespaced mutating `Stack` with
`managementPolicies: ["Observe"]` and an external name equal to the slug. It
must emit no AccessPolicy, rotating token, or PushSecret until
`status.atProvider.id` is present. That ID is provider-observed input to an
AccessPolicy realm of type `stack`; an organization realm is not an equivalent
fallback.

1. Confirm the target profile authorizes the exact slug and region and that its
   organization `ProviderConfig` is healthy. Confirm the provider's
   management-policy feature remains enabled.
2. Apply the consumer request and wait for the observer to report a positive
   `status.atProvider.id`. A missing stack is a reconciliation error, not
   permission to create one.
3. Verify that the rendered AccessPolicy realm uses that observed ID, then wait
   for the rotating-token child and its PushSecret to become Ready and Synced.
4. Consume the **new profile-owned output path** only after its document has
   been verified. The consumer profile's name and output path must be unique in
   every cluster targeting that organization and must not be the source stack
   credential path. Admission enforces uniqueness within the local profile list;
   the platform owner must enforce the cross-cluster boundary.

This operation leaves the source `GrafanaCloudStackRequest`, its external
Stack, and its credential documents in place. It is appropriate when a second
cluster needs a separate credential, not when that cluster will own the stack.

### Transfer full stack ownership between clusters

This is a controlled ownership handoff, not a consumer rollout. A source and
target full-stack Composition must never concurrently reconcile the same slug,
and two `PushSecret` resources using `Replace` must never concurrently write
the same remote key.

1. Freeze source and target promotion. Inventory the source request, all
   composed-resource external names, non-secret provider IDs, dependent
   requests, and every remote credential document path. Render the target
   request and compare its proposed ownership and output paths before either
   cluster reconciles it.
2. On the source request, confirm `spec.lifecycle.externalResources: Retain`
   and that deletion is not armed. `Retain` is the default, but an earlier
   approved Delete path must be disarmed before continuing. Record the source
   stack's Ready and Synced conditions and the evidence that the external stack
   exists.
3. If a target cluster needs credentials during the handoff, use the consumer
   procedure above with a distinct profile-owned consumer name and output path.
   Move that consumer to its new document and verify it. Do not point it at, or
   let it replace, the source stack credential document.
4. Remove the source dependent requests and their writers through reviewed
   GitOps changes first, after verifying each managed resource uses a
   non-deleting policy. The Stack request's `Retain` field does not configure
   independent dependent requests. Wait for their Kubernetes objects and
   finalizers to disappear, including whole-set and output-document writers.
   Then remove the source full-stack request, wait for its composite and
   composed-resource finalizers to disappear, and prove the external Stack
   still exists. Do not continue while any source writer remains active.
5. Only after step 4, apply the target `GrafanaCloudStackRequest` with the same
   slug and reviewed organization, region, usage, and lifecycle values. Watch
   the managed Stack's external name and conditions. Stop on any Create event,
   changed identity, unexpected update, or external deletion.
6. Check the entire rendered target graph. The full-stack Composition can
   create core service accounts, tokens and content automatically after Stack
   observation; it has no generic one-family-at-a-time adoption switch. Review
   those identities and effects before step 5. Separately owned dependent APIs
   can be reintroduced one at a time after their former writers are gone.
   Provider-assigned credential identities are not inferred from Kubernetes
   names: new credentials may be minted, and retained source credentials must
   be retired through the procedure in
   [Retire the source-held credentials](#retire-the-source-held-credentials)
   once consumers have moved.
7. Handle the generated-credentials path explicitly. A retained source
   PushSecret document can remain in the secret store, but it is no longer a
   current writer. If the target keeps that path, start its writer only after
   the source writer is gone. If the target uses a new path, migrate every
   consumer and verify the new document before retiring access to the old one.
8. Resume promotion only after the target reports Ready and Synced, the final
   inventory matches, each consumer uses its intended path, and rollback has
   been rehearsed without deleting the external Stack.

### Retire the source-held credentials

Step 8 ends with the target owning the stack. It does not end with the source
cluster holding nothing. Under `Retain`, removing a Kubernetes object is
deliberately not a revocation: the external AccessPolicy, its rotating token
and any delivered secret-store document all survive the cluster that vended
them, and they survive with their original scopes. Retiring them is a separate,
ordered step, and it is the last one in the handoff.

Four classes of retained credential are in scope. The first three carry
provider-assigned identities that are never derivable from a Kubernetes name;
the fourth is the delivered copy of a credential rather than the credential
itself:

- the source `AccessPolicy`, keyed `region:policyID`;
- **every** `AccessPolicyRotatingToken` that policy issued, each keyed
  `region:tokenID`. These are the live secrets. Do not assume one token per
  policy: inventory them from the provider side rather than from the source
  cluster's rendered children, because a token issued out of band is invisible
  to the cluster and is exactly the one that outlives the migration;
- any `StackServiceAccount` the source full-stack Composition created
  automatically, keyed `stackSlug:serviceAccountID`, together with its
  every `StackServiceAccountRotatingToken` issued against it. The service
  account is an identity; its tokens are the administrator credentials the
  stack-local provider configuration authenticates with. Every rotating token
  authenticates on its own, so each is a live secret, and retiring one covers
  none of the others. Track each token and its own consumers separately through
  the replacement proof and the cleanup;
- the delivered copies: the remote secret-store document the source `PushSecret`
  wrote, and the in-cluster connection secret it read from.

Inventory the first three from the managed resource external-name annotations
and provider status before anything is removed. The fourth is not a managed
resource and has no external name, so record it separately from the `PushSecret`
itself: its namespace and name, each
`spec.data[].match.remoteRef.remoteKey` it writes, the `spec.secretStoreRefs`
entry that key is written through together with the backend and tenant that
store resolves to, and the namespace and name of the source Secret in
`spec.selector.secret.name`.

Record the store identity with every key, and compare on the pair. A
`remoteKey` is only unique within its store: the same key string in two
different stores is two different documents, so comparing keys alone will
either spare a stale source document or delete a live target one. Removing the
`PushSecret` is what makes all of these references unrecoverable, so record them
first and use the recorded values in the cleanup step.

**Check every source `PushSecret` for `spec.deletionPolicy: None` here, during
the inventory, before step 4 of the transfer procedure above removes the
output-document writers.** That is the last moment the check is possible. This
repository renders `Delete` whenever the request has deletion armed, and
removing a `PushSecret` under `Delete` deletes the remote document with it,
destroying the credential delivery before any retirement check has run and
destroying it for the target as well if the target adopted that path. Correct
the policy, or disarm deletion, before any writer is removed.

A credential you did not record here cannot be revoked from any cluster
inventory afterwards, because no cluster names it any more. The Grafana Cloud
organization's own listing is then the authoritative record of its identity, and
the only place you can go to revoke it. That is separate from its *value*, which
may still sit in remote documents, in a CI secret, or in some other out-of-band
copy that no inventory will show you.

**Prove the replacement before retiring anything.** Every one of these must
hold. They are not all observed in the same place, and the difference matters:
points 1 to 5 read the target cluster and the provider identities it reports,
while point 6 can only be answered by the consumers themselves and by
provider-side authentication records. Nothing this repository ships observes any
of it, so treat the whole list as an operator prerequisite. A target composite
reporting Ready and Synced says the target minted its own credential; it says
nothing whatever about whether any consumer has adopted it.

1. the target composite reports Ready and Synced;
2. the target `AccessPolicy` reports a provider-assigned `status.atProvider`
   policy ID that is **different** from the inventoried source policy ID. An
   identical ID means the target adopted the source policy rather than minting
   its own. That retires the *policy* from this procedure and nothing else: its
   rotating tokens, any service-account tokens, and every delivered copy are
   still source-held and still go through the rest of these checks. Skip the
   policy revocation in the retirement steps, and record explicitly that you did
   and why, so the next operator does not read the gap as an omission;
3. **every** target `AccessPolicyRotatingToken` for that policy reports Ready
   and Synced, and each of their provider-assigned token IDs differs from
   **every** source token ID inventoried for the policy. Both sides are sets:
   check all target tokens against all source tokens, not one against one. The
   policy comparison in point 2 does not cover this, because a token is a
   separate provider identity, and a match on any pair means that source token
   is still the live credential. Proceed only when all target tokens are Ready
   and Synced and no pair matches;
4. the target `PushSecret` reports the ESO condition `Ready=True` with reason
   `Synced`. That is ESO's own condition, not the Crossplane `Ready` and
   `Synced` pair used elsewhere in this guide. Then read the remote document and
   the source Secret directly and confirm they hold the target credential;
   neither is a Crossplane resource and neither carries conditions to check;
5. for every retained `StackServiceAccount`, and for **every** rotating token
   inventoried against it, the target reports its own provider-assigned
   service-account ID and token IDs, and each of those target IDs differs from
   **every** inventoried source ID for that identity, not from a
   position-matched one. Do not revoke while any target ID matches any source
   ID. These are separate identities from the access policy and they need their
   own comparison; a Ready target composite does not imply they were replaced;
6. every active consumer has demonstrably **reloaded** the target credential.
   Reading the target path is a configuration fact, not a runtime one: a process
   that loaded the source token at start-up keeps presenting it until it
   restarts or refreshes, and it will keep succeeding right up to the moment you
   revoke. A successful call after the write is not enough on its own either,
   because the call succeeds identically on the old credential. Take either an
   explicit per-consumer reload confirmation, or provider-side evidence that the
   successful operation authenticated as the **target** token ID. A configured
   path, a healthy target cluster and a green request are together still not
   this evidence.

**Then retire, in this order.** The order is not cosmetic: removing the
Kubernetes owner first is what stops the source Composition from immediately
re-minting whatever you revoke.

1. Confirm the source `PushSecret` writer is gone. Step 4 of the transfer
   procedure above will normally have removed it already; remove it here if it
   has not, through a reviewed GitOps change. Either way it must have carried
   `spec.deletionPolicy: None` when it was removed, per the inventory check
   above. Under `None` the remote document survives the removal and still holds
   a working token, which is what the rest of this procedure assumes. If a
   writer was removed under `Delete`, stop: the document is already gone and
   you are recovering a delivery, not retiring a credential.
2. Remove **every** inventoried source rotating token object for that policy,
   then the AccessPolicy object, from the source cluster, still under
   non-deleting policies, and wait for their finalizers to clear. Repeat step 1
   for each token that had a writer of its own. Nothing has been revoked yet at
   this point.
3. Revoke at the provider: every inventoried token for that policy first, one at
   a time, and the policy itself only once none remain. Revoking the policy
   invalidates its tokens, so the reverse order leaves a window where the policy
   is gone and each token's failure mode is harder to attribute. A token you
   inventoried from the provider but never saw in the source cluster is revoked
   here like any other; it is the one most likely to be missed.
4. Delete the stale remote secret-store document and the in-cluster connection
   secret, using the `remoteKey` and source Secret references recorded in the
   inventory.

    **Check ownership of each recorded `remoteKey` first, and delete only the
    source-only ones.** Step 7 of the transfer procedure permits the target to
    keep the source output path. Where it did, that path is now the *target's*
    live document with the target's writer behind it, and deleting it destroys a
    working credential delivery. Compare each recorded key against the target's
    own `PushSecret` `remoteKey` set: delete a key only when it appears in the
    source inventory and has no remaining live writer of any kind, matching on
    the store identity and the key together rather than on the key alone. The
    target's `PushSecret` set is the usual second owner but it is not the only
    possible one: a pair still claimed by any writer stays.

    The in-cluster connection Secret needs its **own, separate** check: the
    `remoteKey` comparison says nothing about it. Before deleting it, look for
    any remaining local owner or consumer of that namespace and name, including
    another `PushSecret` selector and any workload mounting it. Keep the Secret
    while anything still references it, whatever the remote comparison said.

    This deletion authenticates with the configured `SecretStore`'s own workload
    identity for the selected backend, never with the Grafana token just
    revoked, so confirm that identity can delete the recorded key before
    starting step 3. A revoked token left in a source-only document is an
    operational trap: it will be found, tried, and its failure misread as a
    target-cluster fault.
5. Then, for **each** retained `StackServiceAccount` individually, and only
   once point 5 of the replacement proof holds for that specific service
   account:
   1. remove any writer delivering its token, and confirm it is gone. The same
      retention precondition applies: it must carry `spec.deletionPolicy: None`,
      or removing it takes the remote document with it. Where the target adopted
      that same remote path, the document is the target's and must survive until
      the source-only ownership comparison in point 4 below has cleared it;
   2. remove every inventoried `StackServiceAccountRotatingToken` for it, then
      the `StackServiceAccount` itself, from the source cluster under
      non-deleting policies, and wait for the finalizers to clear. Nothing is
      revoked yet;
   3. revoke at the provider: every one of its tokens first, one at a time, then
      the service account once none remain;
   4. delete each of those tokens' delivered copies, remote and in-cluster,
      using their own recorded references and the same source-only ownership
      comparison as step 4 above. A service-account token path the target
      adopted is the target's document now.

   Do not batch these across service accounts and do not carry the access
   policy's replacement evidence across to any of them. Each one is a separate
   provider identity with its own replacement to prove.

**Revoking too early** takes production down with no rollback. A provider
identity is assigned, not chosen, so a revoked policy or token cannot be
restored, and its replacement necessarily has a different ID. Any consumer still
holding the old value fails closed and stays failed until it is repointed by
hand. This is why the replacement evidence above is a precondition and not a
checklist to fill in afterwards.

**Never revoking** is the more common outcome and the worse one. The credential
outlives the cluster that vended it, the GitOps repository that described it and
the reviewers who approved its scopes. Because the source Kubernetes objects are
gone, it appears in no cluster inventory at all: only the Grafana Cloud
organization still knows it exists. A handoff that stops at step 8 leaves a
full-scope standing credential behind on every migration, permanently.

**Operator prerequisites.** This repository makes no live Grafana Cloud, cluster
or source-environment contact, so none of the following is exercised or proven
here, and each is the operator's to carry out and verify:

- confirming provider-side revocation actually took effect, rather than
  inferring it from the Kubernetes object being gone;
- confirming through the Grafana Cloud organization's own access-policy
  inventory that no retained policy or token from a previous owner remains;
- establishing that no out-of-band copy of a token was taken before revocation,
  which no cluster can determine;
- the rehearsal itself. Rehearse this procedure against a disposable stack
  before running it against one carrying traffic.

### Why `GrafanaStackInventory` is not the handoff mechanism

GCV-0016's `GrafanaStackInventory` is explicitly ruled out for this operation.
Its required `spec.stackRef.name` points to a local
`GrafanaCloudStackRequest`, and its provider sets observe objects inside that
stack: folders, dashboards, teams, users, library panels, probes, collectors,
and organization users. It neither observes the Cloud Stack resource nor
supplies its provider-assigned stack ID. Use it for in-stack drift inventory,
not cross-cluster identity or claim-side adoption.

### General existing-resource adoption

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
