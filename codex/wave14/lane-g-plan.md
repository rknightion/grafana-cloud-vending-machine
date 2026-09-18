# GCV-0078 migration answer and post-wave cutover plan

## Scope and evidence identity

This lane covers acceptance criteria 1 through 5 only. It made no live cluster
or Grafana Cloud contact. The experiment ran against the repository's pinned
Kubernetes 1.37.0 envtest API server from base revision
`52308041d25f465e21e711add0dfcf7578f94866`, using this exact test:

`TestPreOrganizationStackRequestOrganizationMigrationAcrossSchemaUpgrade`
in `platform/function/organization_migration_test.go`.

The test reconstructs the pre-organization XRD by reversing only the released
organization delta: the field property, its `required` entry, and its
transition rule. It installs that schema, creates and persists two requests
without `spec.organization`, replaces the CRD with the current schema, reads
both objects back from the API server, and attempts both commissioned updates.
The test fails closed if any of those three reverse-delta matches stops being
unique.

The API server returned these responses verbatim:

```text
GrafanaCloudStackRequest.platform.example.org "migrationonly01" is invalid: <nil>: Invalid value: "object": no such key: organization evaluating rule: spec.organization is immutable
```

```text
GrafanaCloudStackRequest.platform.example.org "migrationsibling01" is invalid: <nil>: Invalid value: "object": no such key: organization evaluating rule: spec.organization is immutable
```

The first response is from adding only `spec.organization`. The second is from
adding `spec.organization` while changing the mutable `spec.displayName`
sibling. Both are admission refusals from the real API server. Client-side dry
run was not used.

## AC1 decision

An already-stored pre-organization request cannot gain `spec.organization` in
place under the released current rule. Evaluation of
`self.spec.organization == oldSelf.spec.organization` fails because the
persisted `oldSelf` has no organization key. Kubernetes validation ratcheting
does not rescue either update shape. The cutover must use a replacement
request.

If the owner requires an in-place route instead, the exact released-API edit
needed is to replace:

```yaml
- rule: self.spec.organization == oldSelf.spec.organization
  message: spec.organization is immutable
```

with:

```yaml
- rule: "!has(oldSelf.spec.organization) || self.spec.organization == oldSelf.spec.organization"
  message: spec.organization is immutable
```

That edit admits the one absent-to-present transition and keeps later changes
immutable. It changes a released API validation contract and is therefore a
breaking-change blocker. This lane did not apply it. The replacement branch
below needs no released API change.

## AC2 replacement sequence and survival contract

The human-operated cutover must use the wave's completing revision and function
digest. All observations and mutations below are scoped to the one frozen
estate.

### Pre-state witnesses

1. Freeze automated promotion. Record the deployed source revision, function
   digest, XRD and Composition hashes, and the rendered delivery output. Record
   the source example revision and the generator/patch revision that rewrites
   it.
2. Record the stored request YAML, UID, generation, immutable slug, region,
   exact usage, profile, lifecycle, Ready and Synced conditions, and
   `status.outputSecretPath`. Confirm organization is absent.
3. Inventory every composed Kubernetes object, its external-name annotation,
   non-secret provider identity, management policies, deletion flags,
   conditions, and owning composite resource. Inventory all dependent requests
   that name this stack through `spec.stackRef.name`.
4. Inventory each credential identity and delivered copy separately: the core
   stack service account, every rotating token issued for it, fleet and
   telemetry access policies, every token issued for those policies, every
   additional service account and token, every source Secret, and every
   `PushSecret` store-and-remote-key pair. Provider-assigned IDs are sets, not
   names inferred from Kubernetes objects.
5. Prove `spec.lifecycle.externalResources` resolves to `Retain`, deletion is
   not armed, the managed Stack has delete protection enabled, rotating tokens
   have `deleteOnDestroy: false`, and every credential `PushSecret` involved in
   this move has `deletionPolicy: None`. Stop if any Delete path is active.
6. Read and preserve the old administrator, fleet, telemetry and active
   dependent documents before mutation. Record their store identity as well as
   their remote key. Confirm each active process's configured path and current
   token identity so a later reload can be distinguished from continued use of
   the old token.
7. Render the replacement from the completing revision without applying it.
   It must keep the same metadata name, slug, region, usage, profile and
   lifecycle, add the reviewed organization key, and produce the expected new
   organization-segmented path. Review the entire desired graph and compare
   every proposed external identity with the pre-state inventory.

### Ordered mutations

1. Keep dependent requests present. The current function preserves their
   existing composed resources when the referenced stack is temporarily absent
   or not Ready and observed children already exist. Do not delete those
   requests as part of the stack replacement. Keep promotion frozen so no
   unrelated desired-state change is introduced during this preservation
   window.
2. Remove the old `GrafanaCloudStackRequest` through its delivery mechanism and
   wait for its composite and owned Kubernetes children to finish deletion.
   Continue only after the external Grafana Cloud Stack is independently shown
   to exist with the same slug and provider identity.
3. Apply the replacement request with the same name and the reviewed
   organization. The new managed Stack must carry the same external name.
   Watch provider events and stop on a Stack Create, a changed external
   identity, or an external deletion. Adoption/observation of the retained
   stack is required.
4. Allow the replacement's core graph to reconcile. The full-stack Composition
   may mint a new core service account and new rotating tokens. Treat those as
   replacement identities. Do not assume that matching Kubernetes names mean
   that a provider-assigned identity was adopted.
5. Wait for the replacement composite and its Stack, core service account,
   replacement rotating tokens, new-path `PushSecret` writers, per-stack
   `ExternalSecret`, and ProviderConfig to reach their documented healthy
   conditions. Then allow preserved dependent composites to re-resolve the
   replacement stack and verify their existing provider identities and desired
   resources against the pre-state inventory.
6. Cut remote-path consumers over in the AC3 order below. Prove the replacement
   credentials and consumer reloads before revoking anything.
7. Retire source credentials only after replacement proof. Remove the old
   Kubernetes owner/writer first, then revoke every old token, then its policy
   or service account, using the recorded provider identities. Delete an old
   delivered document only when its exact `(store identity, remote key)` pair
   has no remaining live writer or reader. A replacement writer that adopted a
   key owns that document, so that key is not source-only.
8. Resume promotion only after all post-state checks pass and the old path has
   crossed the last-reader boundary below.

### What survives and what changes

The external Grafana Cloud Stack survives removal of the old request because
the Retain branch uses management policies without `Delete`, keeps Stack delete
protection enabled, and sets provider deletion flags false. The replacement
must adopt that same stack by its unchanged external-name slug; a Create event
is a stop condition.

The old core stack service account and its tokens also survive Kubernetes
object removal under the same non-deleting policies. Their remote documents
survive because `PushSecret` deletion policy is `None`. They remain live
credentials until explicitly revoked. The replacement may mint a different
service account and tokens because these identities are provider-assigned.
Therefore the safe contract is overlap, consumer cutover, verified reload, and
then source-token-first retirement. A Ready replacement alone is not proof that
any consumer stopped using a retained source token.

## AC3 remote-path consumer map and cutover

Let `OLD` be `{prefix}/{region}/{usage}/{slug}` and `NEW` be
`{prefix}/{organization}/{usage}/{slug}`. Inventory uses exact store-and-key
pairs; a key alone is not an identity.

Repository-owned path relationships are:

| Path | Writer or reader | Cutover evidence |
| --- | --- | --- |
| `OLD` / `NEW` | Core administrator `PushSecret` writes `vending.json`. The per-stack `instance-credentials` `ExternalSecret` reads the service-account token and stack URL from it. External automation may also read it. | New writer ESO `Ready=True`, reason `Synced`; direct new-document readback; new `ExternalSecret` spec and synced version; external consumer reload proof. |
| `OLD/fleet-management` / `NEW/fleet-management` | Fleet `PushSecret` writes `fleet-management.json`; `instance-credentials` reads it into the per-stack ProviderConfig credential Secret. External collectors may read it. | New writer and direct document readback, followed by current ProviderConfig Secret evidence and collector authentication as the replacement token. |
| `OLD/telemetry-publisher` / `NEW/telemetry-publisher` | Telemetry `PushSecret` writes `telemetry.json`; the base administrator document publishes this path. Telemetry collectors are external readers. | New writer and document readback, then per-collector config change, reload, and authentication as the replacement token. |
| `OLD/service-accounts/{account}` / `NEW/service-accounts/{account}` | Each `GrafanaServiceAccounts` `PushSecret` writes one service-account document. Workloads using each named account are external readers. | One replacement service-account and token identity per account, new writer/readback, then per-workload reload proof. |
| `OLD/k6` / `NEW/k6` | `GrafanaK6Project` writes `k6.json` and its own `ExternalSecret` reads that document into the k6 ProviderConfig Secret. External readers, if any, share the same inventory boundary. | New writer/readback, new remote key in the `ExternalSecret`, synced target Secret, and retained project identity checks. |
| `OLD/synthetic-monitoring` / `NEW/synthetic-monitoring` | `GrafanaSyntheticMonitoring` writes the derived credential document. Any out-of-band reader must be inventoried. | New writer/readback and consumer reload proof where a reader exists. |
| `status.outputSecretPath` and `status.telemetrySecretPath` | The stack status feeds required-resource context for service accounts, k6, synthetic monitoring, and other stack-referenced renderers. The delivery generator and operational tooling may read status. | Replacement status equals `NEW`; every dependent composite has re-resolved the new stack UID/path while preserving or deliberately replacing its external identities. |

Before mutation, search the estate delivery source, generated manifests,
cluster `ExternalSecret` objects, workload configuration, CI/CD secret
references, and secret-store access policy for `OLD` and every descendant.
That inventory is the complete set of readers to check off. Source inspection
cannot prove that out-of-band set.

Cut over in this order:

1. Keep all old documents and readers active.
2. Establish the replacement stack and replacement credential identities.
3. Start and verify every `NEW` writer. Read each remote document directly.
4. Let repository-owned `ExternalSecret` readers move to `NEW` and verify their
   target Secrets and dependent ProviderConfigs.
5. Move external consumers one at a time: administrator automation, fleet
   collectors, telemetry collectors, named service-account workloads, k6, and
   synthetic-monitoring readers. After each configuration change, force or
   observe a reload and prove provider-side use of the replacement token
   identity. Merely reading `NEW` in configuration is insufficient.
6. Re-run the exhaustive reference search. `OLD` stops being read at the exact
   moment the last inventoried reader has both removed the old store/key from
   configuration and demonstrated a reload onto the replacement credential,
   and the repository-owned `ExternalSecret` set contains no `OLD` key. Record
   that timestamp and last reader.
7. Only after that boundary, revoke old credentials and remove source-only old
   documents. Keep any store/key pair that still has a live writer or reader.

## AC4 immutable non-standard usage

The stored `spec.usage` value must be copied byte-for-byte into the replacement.
It cannot change: the released XRD makes it immutable, it selects platform
policy, and it remains a segment in both output identities.

The estate's delivery patches a repository catalogue example through a
generator, so the completing-revision overlay must do all of the following in
one reviewed render:

1. add `spec.organization` while preserving the request's existing
   `spec.usage`, slug, region, profile and lifecycle;
2. retain the non-standard usage in the stack Composition input's top-level
   `spec.allowedUsages`;
3. retain the same value in the selected organization's `allowedUsages`;
4. retain or add every active usage-keyed platform profile needed by the
   estate's dependent requests, such as expiry, k6, synthetic-monitoring or
   product-specific budgets; and
5. rebase the generator patches against the exact completing catalogue example
   and fail the delivery if a selector no longer matches. The rendered request,
   XRD/Composition inputs, and generated diff are the evidence, not the patch
   source by itself.

The request renderer checks both the platform-wide and selected-organization
allow-lists. Widening only one still refuses the unchanged usage. No step in
this plan renames or normalizes the value.

## AC5 completing-revision collection contract

AC5 is awaiting the root's integrated completing-tree identity. Lane C's
stack-consumer XRD and lane E's provisioning-connection XRD were still dirty
shared-checkout work when this plan was written, so
`52308041d25f465e21e711add0dfcf7578f94866` is not the required AC5 identity.

After lanes C and E are integrated, the root must record the exact integrated
SHA and function digest, then re-derive the migration delta from the frozen
estate's deployed revision plus its generator overlays to that SHA. The delta
must cover every XRD and Composition, required/defaulted/immutable fields,
validation policies, provider package and CRD versions, function package,
Argo delivery inputs, and every active dependent request kind. Hash the old
render, new render, and overlay output; identify additions, refusals,
replacements, and unaffected fields from those artifacts. Do not reuse a delta
captured before the wave. AC5 is satisfied only when that integrated SHA and
artifact set are attached to the task evidence.

## AC6 human resume boundary

AC6 is explicitly not done in this lane. No request was replaced, no estate was
contacted, and no live stack, service account, token, remote document, or
consumer was changed.

The human resumes after wave 14 has a completing SHA, published function digest,
green exact-SHA validation, and the AC5 delta above. Execute the pre-state
witnesses, ordered mutations, and post-state checks in this plan. AC6 remains
open until the replacement composite is Ready and Synced, the external Stack
identity is unchanged, every dependent resource is accounted for with no
unintended orphan or duplicate, all generator patches apply cleanly to the
completing example, every consumer has reloaded from the intended new path, and
the source credential retirement census is complete.
