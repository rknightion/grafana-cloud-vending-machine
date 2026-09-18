# Wave 13 Lane A: rotating-token replacement decision

## Decision and invariant

Use overlapping, individually identified token generations, with create, observe,
publish, verify-consumer-handover, then retire ordering. Never perform an in-place
rotation update or delete the serving generation to make room for its replacement.

**Testable invariant:** a previously serving token and its connection Secret remain
desired, with their identities and deletion settings intact, until a distinct,
healthy, non-expired successor has a readable connection Secret, its selected
PushSecret generation has published successfully, and, for administrator and fleet
credentials, the actual Secret referenced by the observed ProviderConfig contains
the successor credential.

This is a controller-caused outage prevention invariant, not an unconditional
availability promise. No finite-lived credential design can preserve validity
through an outage longer than the remaining lifetime, recover an already-expired
administrator credential without an independent credential, or guarantee that an
unmanaged external consumer reloads a published credential. The cloud administrator
token uses the independent organization ProviderConfig, so it can recover after
expiry; the OSS token depends on restoring that administrator path first. Do not
describe any of these limitations as renderer-tested live availability.

## Verified provider evidence

Provider v2.14.0 resolves to commit
`dc795606df97a72dce81a0c953e0ec0750e0b489`. Its `go.mod` selects Terraform
provider v4.45.1, commit `7f3311691b0e124c55347f7204501fba77f61453`.

- [OSS rotating-token implementation](https://github.com/grafana/terraform-provider-grafana/blob/7f3311691b0e124c55347f7204501fba77f61453/internal/resources/grafana/resource_service_account_rotating_token.go):
  `resourceServiceAccountRotatingToken` declares `ready_for_rotation` Computed and
  ForceNew, its CustomizeDiff sets it after expiry or entry into the early window,
  and UpdateContext is `serviceAccountRotatingTokenRead`. The third kind therefore
  has the same replacement requirement. This is verified, not inferred.
- [Stack rotating-token implementation](https://github.com/grafana/terraform-provider-grafana/blob/7f3311691b0e124c55347f7204501fba77f61453/internal/resources/cloud/resource_cloud_stack_service_account_rotating_token.go)
  has the same behavior.
- [Access-policy rotating-token implementation](https://github.com/grafana/terraform-provider-grafana/blob/7f3311691b0e124c55347f7204501fba77f61453/internal/resources/cloud/resource_cloud_access_policy_rotating_token.go)
  uses ForceNew `ready_for_rotation` and an expiry-derived token name. All three
  have `delete_on_destroy=false` by default and document overlapping replacement
  when deletion is enabled.
- The local `platform/function/testdata/provider-crds.json` includes OSS and
  access-policy rotating tokens, but not the stack rotating token. The pinned
  provider's `apis/namespaced/cloud/v1alpha1/zz_stackserviceaccountrotatingtoken_types.go`
  confirms the stack fields, including `expiration`, `hasExpired`, and
  `readyForRotation`. The access-policy CRD *declares* `expiresAt`, `expireAfter`,
  and `earlyRotationWindow`; the supplied live evidence says they are absent.
  Schema permission is not evidence of live publication. Never require those
  access-policy status fields to schedule rotation.

The task's supplied Upjet refusal and successful manual replacement are accepted
starting evidence. No live service or cluster was contacted. An upstream update
would be needed to make in-place Update rotate; this design does not require it.
No upstream issue was filed by this lane.

## State machine and identities

Implement one shared generation helper in `access.go`, used by every token
constructor. A family consists of its legacy logical key plus generation children
carrying renderer-owned annotations for family, ordinal, and predecessor UID.
The observed PushSecret selector is the durable serving-generation pointer.
Annotations are non-secret state, not token values or token hashes.

1. **Adopt.** Existing legacy tokens keep their exact composition resource key,
   Kubernetes name, connection-secret name, external-name annotation, parent IDs,
   provider binding, and immutable creation parameters. No initial migration
   rename. A fresh vend may also start at the legacy name. A configured family
   must not disappear merely because its parent's ID is temporarily absent.
2. **Prepare.** When due, create exactly one successor. Use ordinal `n+1` and a
   bounded suffix derived from the predecessor UID for its logical key,
   Kubernetes name, connection Secret, and `namePrefix`. Derivation is deterministic
   across retries; do not use wall-clock buckets or a new random suffix per render.
   Include the suffix in `namePrefix`: upstream appends only second-resolution
   expiry time, so two unsuffixed generations created in one second can collide.
   No successor is minted while an existing successor for that predecessor is
   outstanding. Missing predecessor UID means wait visibly, never guess.
3. **Observe.** Keep the predecessor and publisher unchanged until successor
   Ready=True and Synced=True, no deletion timestamp, provider-assigned identity
   observed, and its own connection Secret contains a nonempty validly decoded
   `attribute.key` or `attribute.token`. Reject a Secret with mismatched namespace,
   name, deletion timestamp, or owner UID. Require ownership by the observed token
   UID; a same-named leftover Secret is not proof. For known expiry require time
   remaining; a candidate already due for replacement must not become active.
4. **Publish.** Change only the existing PushSecret's source selector to the
   successor Secret. Preserve its name, remote key, output JSON shape, template,
   store, and deletion policy. Keep both tokens. Once the observed selector points
   to the successor, never roll it back automatically on a transient failure. If
   that Secret is recreated under a different UID, report `RotationBlocked` and
   retain both token generations; do not claim preservation or retire either one.
   Recovery requires repairing the successor publication or an explicit controlled
   operator selection of a still-validated predecessor, not an inferred rollback.
5. **Verify.** Require the observed selector to equal the successor Secret,
   PushSecret Ready=True, nonempty successful publication state for the configured
   target, and `status.syncedResourceVersion` prefixed by the *observed current*
   metadata generation plus `-`. A previous generation's Ready=True is not enough.
   This follows the pinned [ESO successful reconcile](https://github.com/external-secrets/external-secrets/blob/cc1ae7fe2927fbe61df9aa87bf2e5075972c79f9/pkg/controllers/pushsecret/pushsecret_controller.go)
   and [resource-version helper](https://github.com/external-secrets/external-secrets/blob/cc1ae7fe2927fbe61df9aa87bf2e5075972c79f9/pkg/controllers/util/util.go), which write
   `<generation>-<metadata hash>`; it is not the source Secret resourceVersion.
   Candidate Secret data is immutable by construction for a generation. If its
   UID changes, invalidate the handover and preserve the old token.
6. **Consumer handover.** For administrator and fleet tokens, additionally fetch
   the actual Secret referenced by the observed `provider-config`, decode its
   `credentials` JSON, and compare `auth` or `fleet_management_auth` with the
   successor bytes. Check expected namespace, Secret name/key, expected URL and
   ownership by the observed `instance-credentials` ExternalSecret. Also require
   the observed ProviderConfig and ExternalSecret bindings to match the desired
   stable bindings. ExternalSecret Ready=True alone is not evidence: it can refer
   to the preceding refresh. Never emit token bytes or hashes in status, conditions,
   errors, logs, or annotations. Required Secret data stays in per-invocation memory.
7. **Retire.** Only after those predicates pass may the old MR leave desired state.
   In the normal preserve lifecycle, keep its existing non-deleting management
   policy and `deleteOnDestroy=false`: the external old token expires naturally,
   retaining the existing overlap for external consumers. Do not enable deletion
   merely to implement rotation. Where the operator has already armed the Delete
   lifecycle, removal may revoke the old token, but only after the same handover
   gate. Never remove the active Secret through an old generation's owner reference.
   Wait until the retiring MR is absent before starting another replacement for
   that family. Never reconstruct the legacy child after promotion.

Generation selection must survive process restart and shuffled observed-map
iteration: correlate the observed selector with validated family children and
predecessor annotations, not whichever map entry appears first. Ambiguous lineage,
duplicate ordinals, unknown selectors, or identity conflicts preserve all observed
family members and return a blocked rotation condition. A successor is a *new*
external resource, not a rename of an observed one. Do not copy the predecessor's
external-name or provider-assigned ID onto it. Existing non-token children,
including accounts, access policies, and permissions, never change identity.
Deterministic import identities elsewhere retain their explicit external names.

## Scheduling and stuck states

Use the injected reconcile time, never an independent wall clock in the renderer.
Compute from each generation's observed creation parameters, not newly edited
platform defaults. Let `L` be lifetime, `W` early window, and preparation margin
`min(24h, W/2)`; initiate at `expiry - W - margin`. For absent expiration use
`metadata.creationTimestamp + L` as a conservative lower bound, explicitly
labelled estimated. Kubernetes creation precedes external minting for newly
managed tokens, so this rotates early rather than late. Imported or unverifiable
legacy timing is immediately due, not silently declared safe. Unknown timestamps
or parameters cause visible diagnostic state and retain the current resource.
A confirmed replace-only rotation condition, `hasExpired=true`, or `readyForRotation=true` starts
replacement immediately regardless of the nominal deadline. An unrelated `Synced=False` condition
does not. Do not await Ready=False.

All three kinds use this scheduling and generation logic. Administrator and OSS
use seconds fields and `attribute.key`; access-policy tokens use duration strings
and `attribute.token`, covering telemetry, fleet, and stack consumers. OSS and
standalone consumers require publication acknowledgment, not a nonexistent local
ProviderConfig handover. Their external applications must reload the remote value;
normal non-revocation preserves the existing token until its ordinary expiry.

Report a custom composite condition `CredentialRotationHealthy`: True/Healthy
when no token is due or stuck; False/ReplacementPending while creating;
False/PublicationPending while publishing; False/HandoverPending while waiting
for the materialized consumer Secret; False/RotationBlocked for malformed
identity/evidence or failed successor; False/RotationWindowReached and
False/CredentialExpired for deadline states. Condition messages identify family,
stage, deadline, and whether the deadline is estimated, never credentials.
Set composite Ready=False for all nonhealthy rotation states, even if the provider
leaves child Ready=True. Do not fabricate provider-owned child conditions; the
child's Ready and Synced remain visible alongside the composite diagnostic.
Crossplane Synced may legitimately remain True while the state machine waits.

Tiny supported lifetimes and arbitrarily slow reconciliation cannot satisfy the
availability aim. Do not add a second breaking admission refusal in this wave;
surface insufficient remaining handover time and preserve the predecessor. The
normal default has days of lead time against hourly ESO refresh. An already
expired generation reports degraded recovery, never uninterrupted service.

## Required implementation seams

The original A2 ownership list alone is insufficient. Root must accept and wire
the following bounded prerequisites before calling this complete.

- `access.go`: shared family descriptor, deterministic generation construction,
  schedule, candidate validation, publication acknowledgment, retention/retirement,
  rotation summary helpers; update `addTelemetryAccess` and
  `publishTokenExpiryStatus` to use the selected generation.
- `fn.go` token block: `renderStack` supplies the administrator family descriptor.
  `fleet.go:addFleetAccess`, `serviceaccounts.go:renderServiceAccounts`, and
  `stackconsumer.go:renderStackConsumer`/`stackConsumerToken` supply the remaining
  descriptors. `validateObservedStackConsumerToken` and
  `validateObservedStackConsumerCredentials` must validate selected generations
  against the same authorized policy/provider/output identity without demanding
  the original selector forever. Do not weaken the non-rotation identity checks.
- `fn.go:observedRotatingTokenDeletionPrepared`, `desiredStackStatus`, and
  `expiry.go:expiryDeletionState`: enumerate all extant family generations instead
  of checking only the three legacy keys. Retain the existing deletion-prepared
  contract for each member, and require no unresolved handover for deletionReady.
  That helper inspects rendered/observed configuration, not provider deletion
  execution; do not relabel it a revocation proof. A predecessor still observed
  while deleting remains part of the inventory.
- **Root wiring, `fn.go:RunFunction`:** collect rotation required-Secret selectors
  and identity-validated observations, invoke shared rotation reconciliation for
  configured families before resource publication, prune only retired family keys
  from incoming desired resources because SDK SetDesiredComposedResources merges,
  and apply custom condition/Ready=False *after* normal readiness helpers. Scope
  pruning to explicitly retired keys, never a kind-wide or name-prefix sweep.
  Missing required resources means wait with predecessors retained, not an empty
  desired response. Merge config additions in place and return the same map:
  expiry.go writes `_expiryStatus` through this alias.
- **Root wiring, `fn.go:desiredStackStatus`, `platform/apis/stack-v1beta1.yaml`,
  `bootstrap.go:serviceBootstrapConfig`:** add optional status
  `tokenConnectionSecrets.administrator` and `.telemetryPublisher` objects with
  `name` and `key`. Publish the observed selected, validated generation reference.
  K6 and Synthetic Monitoring currently hardcode legacy connection-secret names;
  use these refs with legacy fallback only when the new fields are absent.
  Validate same-namespace names and the fixed key for each purpose. This is an
  additive internal-reference status surface, not a user-supplied credential path.
  Do not retire the predecessor until the observed stack status points to its
  successor, so downstream resolvers cannot newly select the retired name.
- **Root wiring, RBAC check:** required Secret reads already have precedent in
  `provisioningConnectionCredentialConfig`. Confirm the existing Crossplane role
  can read these same-namespace Secrets; if not, root owns the narrowly required
  read permission change in `platform/rbac/composition-rbac.yaml`. No new rendered
  kind or activation-policy entry is needed. Do not grant Secret write permission.
- Tests in the matching existing `_test.go` files; root also owns bootstrap/schema
  regression checks for the added seams. Add the missing pinned stack CRD fixture
  through the repository's fixture process if schema tests need it, not a fabricated
  hand-written schema.

## Acceptance tests, alternatives, and reversal

Three scenario groups are sufficient:

1. Replay one administrator lifecycle across reconciles: healthy predecessor,
   due, candidate pending, candidate readable, stale PushSecret Ready, acknowledged
   push with old materialized ProviderConfig data, then matching data/status refs.
   Assert predecessor retained at every earlier step and retired only at the last.
   The negative test gives successor Ready=True while publication is stale or the
   consumer Secret still contains the predecessor; reversing order must fail it.
2. Table-test three kinds, access status absent, already-stuck legacy input,
   identity mismatch, and failed successor. Check deterministic retry, distinct
   external namePrefix, no predecessor renaming, no secret alias sharing, and
   conditions/Ready=False. A failed candidate never revokes the predecessor.
3. Restart/promotion plus lifecycle integration: no legacy recreation, only one
   successor, all generations considered by deletion readiness, both deletion
   policies preserved, K6/SM active reference resolution, and existing expiry
   alias tests remain green. Exercise retirement against incoming desired state.

Rejected: blind delete/recreate (credential gap); merely lengthening TTL (delays
the same failure and weakens hygiene); suppressing Update or observing only
(expiration still happens); copying the old token's external identity onto a new
child (adoption, not replacement); shared connection-secret names (racing writers
and owner-reference deletion); readiness-only handover (stale conditions); clock
bucket renaming (unbounded churn and missed generations); waiting solely for an
upstream change (does not resolve the current recurring failure).

Reversal is not a simple renderer rollback: old code would recreate legacy names
and withdraw generation children. Preserve the current serving generation and
its Secret, move publishers/bootstrap references through the same acknowledgment
gate, and only then retire generation state. Disabling proactive scheduling while
keeping generation support is the safe short-term rollback. Natural overlap costs
additional still-valid tokens until their existing expirations; it does not extend
their TTL or grant additional scope.

Live behavior is not exercised, and is not exercisable in this run. Packet
acceptance must include root ownership of the wiring above and the bounded
availability claim; otherwise A2 must return that missing seam rather than
implement a readiness-only approximation.
