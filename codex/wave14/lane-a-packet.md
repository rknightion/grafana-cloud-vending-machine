# Wave 14 Lane A: rotation implementation packet

Status: design repair submitted for root acceptance, not implementation approval.
This packet supersedes the wave-13 packet where they differ. Findings 1-5 of
`codex/wave13/lane-f-review.md` are binding. Finding 6 is outside this packet.
Inspected repository baseline: `52308041d25f465e21e711add0dfcf7578f94866`.
The rejected wave-13 rotation implementation is absent from the current source.

## Invariant and scope

Retain a previously serving generation and its connection Secret until a distinct
readable, healthy, non-expired successor has been selected for publication,
publication is acknowledged at that selector generation, all applicable local
consumer handover checks pass, and retirement intent is durable. Never rotate by
updating the external identity or deleting the serving token first. Every retained
member, including failed and pending candidates, appears in every fresh desired
map. Preserve existing non-token identities, output documents, remote paths and
scope checks. No token bytes or hashes enter status, annotations, logs or errors.

| Site | Kind and group | Minting provider | Additional handover |
| --- | --- | --- | --- |
| `fn.go:renderStack` administrator | `StackServiceAccountRotatingToken`, cloud | organization | stack ProviderConfig `auth`; observed administrator status reference |
| `access.go:addTelemetryAccess` | `AccessPolicyRotatingToken`, cloud | organization | observed telemetryPublisher status reference |
| `fleet.go:addFleetAccess` | `AccessPolicyRotatingToken`, cloud | organization | stack ProviderConfig `fleet_management_auth` |
| `stackconsumer.go:renderStackConsumer` / `stackConsumerToken` | `AccessPolicyRotatingToken`, cloud | platform-authorized organization profile | publication only |
| `serviceaccounts.go:renderServiceAccounts` | `ServiceAccountRotatingToken`, oss | referenced stack | publication only |

All five require publication acknowledgement. The administrator and OSS kinds use
`attribute.key`; the three access-policy sites use `attribute.token`. The stable
legacy keys remain `stack-token`, `telemetry-token`, `fleet-management-token`,
`stack-consumer-token`, and `service-account-<account>-token`. Existing generations
are adopted without renaming. Never recreate a legacy child after promotion.

## Pinned evidence

The repository pins crossplane-provider-grafana v2.14.0 with artifact digest
`3f078cf9f0fa4affbf65a5d9e288d3a33fced0b8e79354589763f1e7bd7c7048`
in `platform/provider/provider-grafana.yaml`. The established source revision is
`dc795606df97a72dce81a0c953e0ec0750e0b489`; its
[go.mod](https://github.com/grafana/crossplane-provider-grafana/blob/dc795606df97a72dce81a0c953e0ec0750e0b489/go.mod#L12-L13)
was fetched again and selects Terraform provider v4.45.1.

The third kind was independently rechecked in the pinned Terraform source,
[OSS rotating token](https://github.com/grafana/terraform-provider-grafana/blob/7f3311691b0e124c55347f7204501fba77f61453/internal/resources/grafana/resource_service_account_rotating_token.go#L20-L95):
`ready_for_rotation` is Computed and ForceNew; CustomizeDiff sets it after expiry
or entry into the early window; UpdateContext invokes Read. Creation appends an
expiry-second suffix to namePrefix, so our generation must add a distinct prefix.
Deletion defaults to leaving the token to expire. This is the same replacement
requirement, not an assumption based on the cloud kinds. The
[access-policy implementation](https://github.com/grafana/terraform-provider-grafana/blob/7f3311691b0e124c55347f7204501fba77f61453/internal/resources/cloud/resource_cloud_access_policy_rotating_token.go)
was also re-fetched. The wave-13 packet records the corresponding stack-token and
Upjet evidence. These are source facts, not a new live reproduction.

## 1. Complete retention and a reachable publication transition

Implement a shared family reconciler in `access.go`. Its explicit result contains
retained desired members, selected source Secret, phase/reason, timing for every
member, required Secret selectors, and exact retired key/UID pairs. Descriptors
carry the legacy key, GVK, namespace, stable publisher identity/output binding,
new-generation template, minting provider, optional consuming provider, and
optional active-status-reference purpose. No private summary may discard fields.

Inventory first, then validate. Preserve each observed member's logical key,
name, namespace, external-name, relevant annotations and full owned spec. Strip
server-owned metadata and status when making desired objects; do not copy status
back as desired. Inventory is independent of Ready/Synced and parent-ID presence.

Assign new children renderer-owned family, ordinal and predecessor-UID metadata.
Their deterministic suffix incorporates predecessor UID and ordinal, within name
limits, and is included in logical name, Kubernetes name, connection-secret name
and namePrefix. Do not copy the predecessor's external-name or external ID.
One outstanding child per predecessor; no retry-generated names. Missing UID or
conflicting identity blocks new creation, not retention.

Use the following transitions, always rendering the full retained inventory:

1. With a valid observed publisher selector P and a due P, render P and one new
   candidate C. Desired publisher remains P while C is absent, pending, failed,
   lacks readable credentials or has unknown/insufficient remaining validity.
2. Once C is observed Ready=True and Synced=True, not deleting, with a provider
   identity and valid owner-bound connection Secret, emit desired selector C
   **before** requiring acknowledgement of C. Keep P and C. Write non-secret
   handover annotations on that same publisher identifying predecessor token UID,
   candidate token UID and candidate Secret UID. No credential digest is stored.
3. Until the publisher is observed selecting C, keep requesting C while C remains
   qualified. After C is observed selected, retain that selector permanently across
   transient errors. Never fall back automatically to P or the legacy name.
   Require the observed handover annotations to match the actual UIDs before
   accepting acknowledgement. A recreated Secret or token blocks retirement.
4. Accept push acknowledgement only for observed selector C, Ready=True, the
   expected store and remote target present in successful publication state, and
   syncedResourceVersion prefixed by the current observed metadata generation
   plus a hyphen. A stale Ready condition is insufficient. Require C still readable
   and valid at retirement. The pinned ESO generation/hash contract and source
   links are in the wave-13 packet; do not substitute a Secret resourceVersion.
5. Complete the applicable consumer/status gates in section 3. Then record exact
   predecessor logical key and UID as retirement intent on the surviving publisher,
   retaining P for this response. Omit P only after that intent is observed and
   all handover gates still pass. Recompute the decision after restart. While P
   remains observed, retain the intent and omit only that authorized UID/key;
   a conflicting UID blocks. No unrelated incoming desired key is pruned.
6. After P is absent, the selected C is the baseline. Keep enough lineage on C
   and the publisher to distinguish completed promotion from an unvended family.
   Do not mint another successor until retiring P is absent. Replace completed
   retirement intent only as part of the next acknowledged handover, never by
   inferring that a missing legacy resource should be reconstructed.

Token Secret validation requires the expected GVK, namespace, name, non-deleting
metadata, nonempty UID, owner reference matching the token UID and nonempty valid
base64-decoded value at the correct key. Same-named leftovers do not qualify.
External consumers are not proven to reload by PushSecret acknowledgement.

## 2. Malformed lineage and missing clock preserve observed state

Inventory must survive validation failures. Use observed composition ownership
and token GVK to conservatively collect all potentially related members before
parsing family/ordinal metadata. Invalid annotations, duplicate ordinals, unknown
selectors, missing members, conflicting UIDs or ambiguous attribution cause
RotationBlocked. Preserve all potentially related observed tokens at their exact
keys/specs, together with the observed publisher selector and handover annotations.
Do not invent a replacement publisher or select the legacy Secret. A family that
has observed children but no observed publisher is blocked and retains those
children; it does not infer a safe new selector.

Missing injected reconcile time follows exactly this preservation path. Do not
use time.Now inside the helper. Never return an empty inventory or Healthy on an
unknown clock. The same rules apply under an already armed Delete lifecycle:
retention preserves observed deletion settings; no omission, automatic revocation,
new Delete arming or deletion-prepared verdict follows a validation failure.

Generation-aware preservation must run before existing parent-ID fallbacks and
admission stages could omit a child. If stack identity is unavailable, do not
relax authorization to mint; preserve the observed tree and report blocked.
Fatal responses alone are not the replay proof: test the actual emitted desired
inventory on nonfatal waiting/blocked paths with no incoming desired state.

## 3. Separate minting identity, consumer identity and status handover

Never compare the consuming ProviderConfig with the minting provider. Represent
both explicitly, including kind/name/namespace. Administrator and fleet tokens
use the organization provider; their consumer is the distinct stack provider at
`provider-config`. OSS uses that stack provider to mint but has no local consuming
ProviderConfig gate of its own. Telemetry and standalone consumers likewise do
not invent a consuming ProviderConfig.

For administrator/fleet, validate the observed stack ProviderConfig against its
stable expected identity, URL and Secret binding. Fetch the exact Secret referenced
by its observed credentials.secretRef, requiring the expected namespace/name/key,
non-deleting metadata and ownership by the observed `instance-credentials`
ExternalSecret UID. Validate that ExternalSecret's stable target/store/remote
bindings too. Decode `credentials` JSON in memory and compare the relevant auth
field to C's credential. ExternalSecret Ready alone is insufficient. Test genuinely
distinct organization and stack provider names and reject either being substituted.

Root adds optional `status.tokenConnectionSecrets.administrator` and
`.telemetryPublisher` references, each `{name,key}`, to `desiredStackStatus` and
the stack schema. Advance the desired reference after C's publication acknowledgement
and applicable materialized consumer check. Retain P until the **observed composite
status** contains C's validated reference. It is not enough to have just emitted
that status in the same response. Preserve old references on blocked observations.
This introduces one deliberate extra reconcile before retirement and no Ready
circularity: status handover does not require composite Ready=True.

`bootstrap.go:serviceBootstrapConfig` uses these references for K6 and Synthetic
Monitoring. Legacy fallback is allowed only when the relevant field is absent;
malformed present fields block. Enforce same namespace and the fixed purpose key.
Retirement proves that future resolvers will not choose P, not that all existing
external consumers reloaded it. Fleet has no separate direct bootstrap reference;
its materialized stack ProviderConfig is the applicable active-reference evidence.

## 4. Immutable observed timing, preservation and honest unknowns

For existing members, preserve observed forProvider creation parameters, parent
bindings, providerConfigRef, writeConnectionSecretToRef, managementPolicies and
deleteOnDestroy. Current platform lifetime/window applies only to a new generation.
It cannot rewrite P or an already created C. Preserve absent fields as absent;
never silently fill them from today's defaults. Explicit authorized lifecycle
arming is a separate operation: only after identity/timing/handover resolution may
it change the deletion fields, and deletionPrepared requires those changes observed.
Rotation itself never enables deletion. Under Preserve, external old tokens expire
naturally. Under an already armed Delete policy, retirement can revoke P only
after the identical handover gates.

Compute each member's schedule from its own observed secondsToLive and
 earlyRotationWindowSeconds (administrator/OSS), or expireAfter and
 earlyRotationWindow (access-policy). Parse durations strictly. Prefer a valid
provider expiration/expiresAt. Otherwise a renderer-created, non-imported token
with verifiable unchanged creation parameters may use creationTimestamp + its own
lifetime as an explicitly estimated conservative deadline. Kubernetes creation
precedes minting, so that bound is early. Imported/adopted external identity without
credible mint timing is unknown. Missing timestamps, missing/invalid parameters,
clock skew into the future, or inconsistent evidence are unknown, never Healthy.

Start preparation at expiry minus early window minus min(24h, early window/2).
A confirmed replace-only rotation refusal, readyForRotation or hasExpired makes
replacement due immediately even if the computed deadline differs. Unrelated
Synced=False does not prove the replacement mechanism. Unknown P timing blocks
healthy claims; where identity and creation authorization are sound it may trigger
one recovery candidate with known new settings. Unknown C timing never qualifies
C for publication or retirement. A candidate already in its preparation window
cannot become serving. Short lifetimes and delayed reconciliation may make progress
impossible; report insufficient remaining time rather than weaken the gates.

Return and retain per-member expiry, preparation deadline, rotation-window start,
source (provider or creation estimate), and validity-known flag. Expose no estimated
value as provider-observed tokenExpiries. Conditions retain the deadline/provenance
in diagnostic messages without credentials. Existing lifetime increases must not
move P's deadline; test this with access-policy status entirely absent.

## 5. Exact integration and family deletion seams

Root owns `fn.go:RunFunction` integration in this order:

- After observed inventory/config resolution, collect named same-namespace Secret
  requirements for all retained candidate/selected generations and observed stack
  ProviderConfig consumers. Merge requirements without replacing unrelated ones.
  Missing required observations produce retained resources and a pending condition.
- Pass validated observations and one injected clock to all five family sites.
  Preserve the config alias used by expiry status; do not replace the map and lose
  `_expiryStatus`. Keep explicit rotation results through desiredStackStatus.
- Run generation-aware preservation through stack/content readiness gates. Prune
  only explicit retired key/UID decisions from the response's incoming desired
  resources before SetDesiredComposedResources, which merges rather than replaces.
  Also omit those keys from the newly rendered desired map. Retention cannot depend
  on this incoming map containing anything.
- After all normal readiness helpers and status generation, apply final
  `CredentialRotationHealthy` and composite Ready=False for any pending, blocked,
  expired or window-reached family. Do not fabricate child conditions or force
  Ready=True when other prerequisites fail. Healthy requires all configured
  families accounted for, known valid timing, and no unresolved handover.
  Reasons include ReplacementPending, PublicationPending, HandoverPending,
  RotationBlocked, RotationWindowReached and CredentialExpired. Choose deterministic
  severity/order while preserving family-specific diagnostics and timing provenance.

Replace the legacy-only deletion helper used by `desiredStackStatus` and
`expiry.go:expiryDeletionState` with a family-inventory helper. An enabled family
needs a recognized current selected member, complete unambiguous inventory and no
unresolved publication/consumer/status handover or pending retirement. For every
extant member, including one with deletionTimestamp, verify observed policy `*`
and deleteOnDestroy=true. Check publisher deletion preparation independently.
The absent legacy key is not required after acknowledged promotion; an absent
current member or never-vended required family is not vacuous success. A disabled
family is exempt only when no members or unresolved retained handover remain.
Malformed lineage or missing clock returns not prepared. This is preparation
configuration evidence, never proof that an external revocation executed.

Update selected-generation checks in stackconsumer token/credential validators
without relaxing profile, policy, namespace, provider, scope or remote-output
identity. Update all legacy-only child-presence guards and expiry/status lookups.
Service-account parent failures must recognize generation keys too.

The checked `platform/rbac/composition-rbac.yaml` grants ESO resource access and
contains no explicit Secret rule. Root must verify the pinned Crossplane installation's
existing Secret-read authority offline; if missing, add the narrow required read
permission in the root-owned RBAC seam. Do not infer permission from an existing
helper, add Secret writes, or call a live cluster. The stack rotating-token CRD is
absent from the current provider fixture; any fixture addition must use the pinned
upstream CRD, not a handwritten approximation.

## Acceptance replay and ownership boundary

Use an offline controller replay with a fresh desired map on every invocation.
Start with only a legacy observed family and publisher. The simulated API applies
the preceding desired response to construct the next observation, increments the
publisher generation on selector changes, and preserves assigned resource UIDs.
Only provider readiness/identity, Secret creation, ESO acknowledgement and consumer
refresh are independently advanced. Never hand-edit the observed selector to C.
Derive observed composite status and retirement annotations from preceding desired
responses as well. Shuffle observed map order and restart the helper between steps.

The mandatory trace is: due P -> C created -> C pending retained -> C failed retained
-> C readable -> desired selector C while observed selector P -> observed selector C
with stale Ready -> current push acknowledgement with old consumer bytes -> new
consumer bytes but old stack status -> desired new status -> observed new status
-> retirement intent emitted -> intent observed -> P omitted -> P absent -> C only,
without legacy recreation. Missing or changed candidate Secret UID blocks at each
applicable point. The publication-only sites omit inapplicable consumer/status
steps, never publication acknowledgement.

Table-test all three kinds across all five descriptors. Add bounded adversarial
cases for malformed lineage/duplicate ordinal/missing clock under Preserve and
Delete; lifetime increase with absent access-token status; unknown C timing; distinct
mint/consumer providers; UID replacement; and stale incoming desired retired keys.
Assert exact retained spec/selector equivalence on blocked paths, final composite
conditions and Ready=False, legitimate deletion readiness after promotion, and
not-prepared during unresolved handover. Keep existing expiry-alias/bootstrap tests.

This lane writes only this packet. Root acceptance is required before implementing;
no self-approval is claimed. No blocker was found in the pinned provider's replacement
capability. Source evidence does not prove uninterrupted availability, external
consumer reload, cluster permissions, or live deletion. Those remain unproven.
The protocol can prevent controller-caused premature withdrawal; it cannot extend
an expired token or survive an unbounded outage. Safe reversal disables scheduling
while retaining generation support, never reinstalls a legacy-only renderer.
