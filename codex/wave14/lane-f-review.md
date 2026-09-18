# Wave 14 lane F fresh review after root correction

**Current verdict: FAIL.** This section supersedes the earlier verdict's source identity and
finding dispositions. The original review is retained below as historical evidence. Root reports
this correction consumed the fourth and final authorized implementation attempt. This review
provides no permission for another attempt; root must apply the goal/protocol stop and extension
rules without weakening acceptance.

## Exact reviewed slice and executed check

Reviewed the corrected accumulated uncommitted tree over
`52308041d25f465e21e711add0dfcf7578f94866`, against the unchanged accepted lane A packet.
Read the current diffs and source for access/helper tests, fn wiring/status/tests, bootstrap/tests,
expiry, stack schema, registry audit and affected docs, plus unchanged consumer and account
renderers. This is not an exact completing-SHA, hosted validation or deployment verdict.

| File | Current SHA-256 |
| --- | --- |
| `platform/function/access.go` | `d46c286fdb28ccede2fd73f11fa2952f90bd19fcd52b99a4158f8c2a42d0ec49` |
| `platform/function/access_test.go` | `b4c84aca66627f0bcd6121b11e5a99f66b2170f1572815e2e2dfad9e53d5aa43` |
| `platform/function/fn.go` | `cf1f73f43151cd1e7868b2fd7cc8da6ad9f6e3e5a893d52b7364334f167b5b36` |
| `platform/function/bootstrap.go` | `40fa0935a348c3b59b4e2dc35e61a2d2e4ad97aa7e076f1da9fc888461730da9` |
| `platform/function/expiry.go` | `306bf450bf82d58cc286eb8f8931e6a0ab102ef809e1660892970a0a6f243345` |
| `platform/apis/stack-v1beta1.yaml` | `cb845c05b28a302b1ea68ebeb8f0feea549ef4f89be786133144b406b4ee5dc7` |
| `platform/function/stackconsumer.go` | `1a55954be14fdfe50fc87c0a124e7e3dc5e4f5a581841252475eb2ec791d0724` |

Executed once, exit 0:

```text
$ just test 'TestRotatingTokenFamily|TestRotationWiring|TestRotatingTokenTiming|TestBoundedRotationName|TestBootstrapUsesObservedSelectedGeneration'
cd platform/function && go test -race -cover -run 'TestRotatingTokenFamily|TestRotationWiring|TestRotatingTokenTiming|TestBoundedRotationName|TestBootstrapUsesObservedSelectedGeneration' ./...
ok  github.com/rknightion/grafana-cloud-vending-machine/platform/function 1.675s coverage: 19.6% of statements
```

This proves those tests pass, not the mandatory full desired-to-observed lifecycle. The replay
ends immediately after first predecessor absence. The new real-site test discovers descriptors;
it never runs the five sites through successor selection, acknowledgement, consumer/status
handover, retirement, restart and a second window.

Root continues to own current-phase route receipt verification. The route-receipts file read
in this phase has no F row. Requested SECURITY route is not substituted for runtime evidence;
provider attestation remains unknown and is not required. Reconcile this phase specifically.

## Prior finding dispositions

| Prior finding | Current disposition |
| --- | --- |
| F1 retirement deadlock and future windows | **Partially corrected, still FAIL.** Exact observed predecessor omission and UID pruning now exist. Completed-intent/readability ordering still blocks the next due window; observed annotation mutation is not durable cleanup. R1. |
| F2 serving selector rollback | **Selector regression corrected on the inspected blocked branches; invariant still FAIL.** Observed publisher is restored, but blocked preservation now rewrites deletion policies from the fresh base, including missing-clock/malformed states. R3. |
| F3 Secret/publication evidence | **Partially corrected, still FAIL.** Required Secrets validate GVK/name/namespace/owner/UID/data, fallback bytes were removed, deleting candidates are refused, and candidate UID witnesses are compared. Exact successful remote target/store entries are still not checked; desired-vs-observed spec comparison can compare an observed copy to itself. R4. |
| F4 timing | **Partially corrected.** Future estimates and candidates inside preparation time are rejected; renderer provenance marking exists. Completed generation still passes through new-candidate time qualification before scheduling, and deletion helper no longer checks timing at all. R1/R5. |
| F5 deletion predicate | **Partially corrected, still FAIL.** Outstanding direct successor and independent boolean checks were added. Unknown timing can now be prepared; stable-legacy bypass suppresses consumer/status requirements. R5. |
| F6 bounded names | **Satisfied in source and focused check.** The base is shortened before suffix concatenation; Kubernetes name, Secret and minting prefix retain the discriminator. |
| F7 missing integration | **Partially corrected, still FAIL.** All five normal emit sites are discovered and invoked; required Secrets, bootstrap references, status schema, pruning and aggregate health override now exist. Renderer refusal, status promotion sequencing and incomplete-inventory health remain. R2/R6/R7. |

Wave-13 hard boundaries 1 through 5 remain **not satisfied overall**, respectively because of
R1/R2, R3, R4/R6, R1/R5, and R2/R5/R7. Improvements above are real and retained; none is a
waiver for the unresolved acceptance boundary.

## Remaining material findings and exact owner corrections

### R1 - HIGH: completed-generation state still cannot survive the second window

`access.go:239-285` validates the selected generation using `validatedRotationSecretUID`
before recognizing predecessor absence. That calls candidate qualification, requiring
`Now < PreparationDeadline`. When C reaches its next preparation deadline, completed
retirement becomes RotationBlocked before `rotatingTokenDue` can schedule D. The path
without retirement annotations has the same early qualification at `302-308` while C's
immutable predecessor UID remains present.

At `278-284`, cleanup deletes from `publisherAnnotations`, which aliases the **observed**
publisher. Desired publisher was cloned at `227`; its annotations are unchanged. Thus
cleanup is neither a durable desired transition nor a safe immutable observation. On
blocked paths after P is absent, the base legacy token can remain in desired as well.
The new replay has no second-window assertion to expose this.

Owner: root or a newly authorized implementation owner. Distinguish completed lineage
from a candidate awaiting retirement before applying candidate time gates. Preserve the
completed lineage durably on desired, do not mutate observed input, suppress absent
legacy recreation on every branch, and permit C to schedule D at its deadline while
keeping C selected. Require an apply-of-prior-desired replay through two full generations
and expired/unknown C after completed P retirement.

### R2 - HIGH: standalone consumer rotation stops at its own pre-wiring validator

`stackconsumer.go:104,314-320` validates the entire observed publisher spec against the
base publisher, whose selector still names the legacy Secret. After the helper selects C
and that selector is observed, the next renderer invocation returns an error from
`stackConsumerObservedChildMatches` (`348` onward). `fn.go:173-177` only invokes rotation
when the renderer succeeded. It therefore cannot acknowledge C or retire P at this site.
The same unchanged legacy-token full-spec comparison also rejects a lifetime configuration
change before immutable-generation handling can preserve the observed token.

Owner: root/assigned consumer renderer owner, subject to attempt authority. Make validators
recognize validated retained/selected generations without relaxing profile, namespace,
provider, policy, scope or remote-output binding. Run the **actual renderer plus wiring**
through observed C selection and subsequent retirement; descriptor discovery is insufficient.

### R3 - HIGH: malformed or untimed evidence now arms deletion during preservation

`preserveRotatingTokenInventory` at `access.go:495-504` copies each observed token then
calls `applyRotationTokenLifecycle`, which overwrites managementPolicies and deleteOnDestroy
from the current base. Missing-clock and malformed-lineage branches call this helper.
`preserveBlockedRotatingTokenFamily` repeats that update and copies publisher deletionPolicy.
`fn.go:267-287` unconditionally reapplies those lifecycle changes after both passes.
An observed Preserve member can therefore become Delete under ambiguous evidence, violating
the accepted packet's exact-spec preservation and no-new-arming rule.

Owner: root/assigned helper owner. Preserve observed policy and deleteOnDestroy byte-for-byte
on every unknown/malformed/unresolved handover path. Permit an explicit authorized lifecycle
transition only after identity/timing/handover resolution; remove unconditional post-pass
arming. Test Preserve observations with current Delete request and missing clock/duplicate
lineage, asserting exact specs and publisher deletion policy.

### R4 - HIGH: publication acknowledgement remains insufficiently bound

`observedRotatingTokenPublisherAcknowledged` (`access.go:1009-1036`) still accepts any
nonempty syncedPushSecrets. It does not require the expected store and remote target's
successful entry. Its full-spec equality does not fix this: the selected branch already
replaces desired publisher with an observed clone (`225-229`), so the comparison normally
compares observed spec with itself rather than the authorized base output binding.
Candidate UID and Secret UID checks are improvements, but predecessor UID/key handover
and exact output acknowledgement remain incomplete.

Owner: root/assigned helper owner. Keep a separate authorized immutable publisher binding
from the renderer. Compare observed store, remote path, template and relevant selector
against it, and inspect the matching successful publication entries. Bind predecessor,
candidate and Secret witnesses consistently before retirement. A nonmatching nonempty
publication map must fail a focused test.

### R5 - HIGH: deletionPrepared accepts unknown timing and bypassed handover

`rotatingTokenFamilyDeletionPrepared` (`access.go:1054-1123`) no longer calls the timing
helper or validates the current Secret witness. A legacy token with no lifetime/creation/
provider expiry can be prepared. `fn.go:293-296` forces both handover booleans true for
stableLegacyRotationFamily, even though its definition validates only inventory, selector
and absence of annotations. Fallback descriptors in `expiry.go` also have no required
handover flags and pass true booleans. This does not satisfy the packet's missing/unknown
timing and active-reference requirements. Changed test fixtures omit timing and now pass.

Separately, `fn.go:304` replaces the expiry deletionReady value with only deletionArmed
and family readiness, dropping the stack delete-protection and publisher checks that
`expiryDeletionState` includes. `mergeStackExpiryStatus` can carry this weaker result.

Owner: root. Restore complete family timing/current-Secret/handover validation, remove
unverified stable-legacy bypasses, and share one complete deletion-readiness result with
expiry/status without discarding protection requirements. Require unknown timing and
still-enabled delete protection to remain not prepared.

### R6 - HIGH: active status reference advances before materialized consumer handover

`publishTokenConnectionSecretStatus` (`fn.go:1204`) promotes every HandoverPending result.
The helper returns that phase for **both** missing materialized consumer refresh and
missing observed status reference (`access.go:317-327`). Thus administrator status can
point to C while the stack ProviderConfig Secret still contains P. The accepted packet
requires C publication acknowledgement **and materialized consumer agreement** before
desired status changes; only its later observed status is the extra retirement gate.

Owner: root. Carry explicit publication/consumer qualification separately from generic
phase and publish the new status reference only when the applicable materialized consumer
gate passes. Replay old consumer bytes -> new bytes/old status -> desired new status ->
observed new status, using the real wiring. Bootstrap's selected-reference lookup and
malformed-present refusal are useful but do not prove that upstream sequence.

### R7 - HIGH: health and retention depend on the base renderer emitting a family

`renderedRotatingTokenDescriptors` (`fn.go:322-377`) skips a family whenever the base
renderer lacks either legacy token or publisher. Inventory is therefore not independent
of parent-ID/readiness gates. An enabled, never-vended or temporarily omitted family is
absent from the aggregate; a family with zero observed members returns Healthy at
`access.go:172-175`, without known timing. `applyCredentialRotationCondition` can consequently
report Healthy for incomplete families. The packet explicitly requires all configured
families accounted for and preservation before parent-ID fallback, including nonfatal waits.

Owner: root with renderer owners. Derive configured family inventory independently of
current emitted children, retain potentially related observed generations before renderer
fallthrough, and mark missing required observations pending/blocked. Assert real responses
for missing parent IDs, missing publisher and zero observed token at every affected site.

## What is proven, what is not, and scope boundaries

Proven by source and the executed focused tests: suffix preservation, the selected Secret
bootstrap lookup, normal-path discovery of all five sites/three kinds, owner-bound required
Secret decoding, final aggregate condition application after ordinary readiness, and an
exact-key/UID pruning mechanism. The aggregate override does not fabricate child readiness.
These are component facts, not the accepted end-to-end failed-replacement invariant.

That invariant, safe lifecycle arming/deletion, second-window rotation, all-site replay and
GCV-0075 AC3/AC4/AC5 remain unsatisfied by this corrected slice. Lane C's unchanged bounded
profile remains acceptable for scope breadth, namespace/target isolation, unique shipped
output path and allowed identifiers; it does not repair the consumer rotation defect.

No new source or test was written. Only this report was updated, retaining its first
review below. I did not run the full gate, CodeRabbit, a production build or hosted checks
in this phase, and did not edit the tracker, stage/commit/push, delegate, or perform any
release/tag/PR or live Grafana/cluster action. Live minting, ESO/provider reconciliation,
consumer reload and revocation remain **not exercised, and not exercisable here**.

---

# Historical first review (superseded by the fresh review above)

# Wave 14 lane F security review

Verdict: **FAIL for lane B; no material security finding in lane C's bounded profile change.**
The failed-replacement invariant is not satisfied on every path. Missing integration is separate
from the helper defects below; applying the wiring packet alone cannot resolve this verdict.

## Reviewed slice and provenance

Reviewed the uncommitted working tree over `52308041d25f465e21e711add0dfcf7578f94866`,
not a completing commit. Binding packet: `codex/wave14/lane-a-packet.md`, SHA-256
`f52fdcfe4e9641fd03c7c201621332c9bc5d70c706b45d484c8d06b60fb31051`.
Read lane B's wiring/evidence packets, lane C's docs packet, the goal, AGENTS.md,
Backlog overview, canonical fan-out protocol including Appendix A, and Wave operating model.

Primary source identities at review:

| File | SHA-256 |
| --- | --- |
| `platform/function/access.go` | `ea03a58c1d71928c2b0f6f8cab5ddaced641851b5bfa9e4a21deaffd632e0f73` |
| `platform/function/access_test.go` | `0d9dec0a3d08cc6dcd2f758bd4101d0742ffa62e92d15480c28c9f2938f66b47` |
| `platform/function/fn.go` | `e4778276a14df7aad596c35f7b13834fcb1299d56a2235cce3d2b64c92e18320` |
| `platform/apis/stack-consumer-v1beta1.yaml` | `48e9efd3d18bc88bb974e5670c24c51ede127680989e12f63192e7d7bdb9d3cd` |
| `platform/function/stackconsumer_admission_test.go` | `2d8cf87d9423847683621acf5d7903ecf2349c5df91dade31ceeb2d87f2fb9d6` |

Also inspected the existing consumer renderer/authorization, bootstrap and expiry seams,
consumer catalog diff, and relevant profile declarations. Unrelated lane D/E/G changes
are outside this verdict. Root owns the exact own-child session/current-phase runtime
receipt and instructed this reviewer to continue without waiting for metadata. Requested
route is SECURITY, `gpt-6-astra` / `medium`; requested route alone is not a verified receipt.
Reconcile this packet with the root's route receipt before using it as the required review.
Provider attestation is unknown and is not required by the goal.

## Wave 13 findings 1 through 5

1. **Not satisfied: publication ordering and complete lifecycle.** The helper does select a
   readable candidate before acknowledgement (`access.go:393`), and preserves pending members.
   But retirement never initiates predecessor omission, the test manually removes that member,
   and a completed intent prevents every subsequent rotation. See F1 and F2.
2. **Not satisfied: malformed evidence preserves the serving selector.** The malformed-lineage
   branch copies the observed publisher, but the missing-clock branch does not replace an existing
   base publisher. Selected-candidate failure and retirement branches also leave the legacy
   base selector in desired. Tokens may remain while the publisher rolls back. See F2.
3. **Not satisfied: distinct minting/consuming identities and active status reference.** The
   descriptor contains separate fields and compares the minting binding. Consumer/status gates
   are unverified booleans; no production caller resolves the materialized consumer or observed
   composite reference. UID acknowledgement is also incomplete. See F3 and F7.
4. **Not satisfied: immutable timing and honest unknowns.** Observed member specs are copied,
   and lifetime calculation uses observed parameters. However imported tokens and future-created
   objects receive unsupported creation estimates; completed intent bypasses validity entirely;
   candidates inside preparation time can qualify. See F1 and F4.
5. **Not satisfied: integration and deletion seams.** No production call invokes the helper;
   Secret requirements, selected status/bootstrap references, final condition/Ready override,
   and generation-aware deletion are absent. The unused deletion helper also has independent
   false-ready/false-block cases. See F5 and F7.

## Material findings and precise owner edits

### F1 - HIGH: retirement cannot complete through desired state, then disables future rotation

`access.go:219-260` preserves P, sees observed retirement intent, and keeps P in desired
while P exists. `pruneRotatingTokenRetiredDesired` at `476-487` likewise continues without
pruning when the exact observed UID exists. Nothing causes P to disappear. The test at
`access_test.go:282-298` explicitly expects P retained, then supplies a later observation
without P. That removal is not derived from any preceding desired response.

If P disappears externally, the same branch returns Healthy before evaluating selected
timing/readiness or scheduling another candidate. Persistent completed intent therefore
makes an expired C Healthy forever and prevents the second rotation.

Owner: lane B, or a root-assigned repair owner after explicit ownership transfer. After
observed exact intent, recheck all handover gates and omit exactly P's key/UID from both
fresh desired and incoming desired while P is still observed. Wait for observed absence;
then continue normal scheduling for C, preserving durable completed-lineage evidence.
Replace the manual-removal replay with apply-of-prior-desired retirement, and carry the
replay through a second due window plus expired/unknown-timing completed-intent cases.

### F2 - HIGH: blocked and completed paths roll publication back to legacy

`preserveRotatingTokenInventory` (`access.go:461-470`) copies the observed publisher only
when desired lacks it. Normal renderers already supply a legacy-selector publisher.
Missing-clock handling (`158-161`), selected-successor unreadability (`273-279`), and the
retirement-intent branch (`225-259`) return without replacing that base selector. Thus an
observed publisher selecting C can emit desired P on error or after retirement; if P has
been removed, publication points to a nonexistent Secret. Handover annotations are also
lost when later responses start from a fresh base publisher.

Owner: lane B/assigned repair owner. Initialize the durable selector and handover state
from the observed publisher before every transition. Never restore the legacy selector
once C is observed selected. On uncertain evidence preserve the whole observed publisher
and every retained token exactly. Test missing clock, failed C and completed intent with
observed C, a fresh legacy base map, and both Preserve and armed Delete settings; assert
selector and annotations, not only token presence or phase.

### F3 - HIGH: publication and Secret evidence is not bound to the acknowledged generation

`observedRotatingTokenPublisherAcknowledged` (`access.go:917-931`) accepts any nonempty
`syncedPushSecrets`, without checking the configured store and remote target or matching
handover token/Secret UIDs. `setRotatingTokenPublisherTransition` obtains the Secret UID
from invented token status (`candidateSecretUID`, `657-659`), not the validated required
Secret observation. `rotationSecretEvidence` (`899-914`) even falls back to status
credential bytes and defaults missing owner UID to the token UID. The checked provider
fixture contains no `connectionSecret`/`connectionDetails` status fields. The test's
`secret-c-old` witness is written into that synthetic status, so its UID-replacement case
does not establish the required real Secret witness.

A recreated candidate Secret with current owner/name can therefore reuse stale publisher
acknowledgement; an unrelated publication entry can satisfy the success check. Deleting
token metadata is not checked by `rotatingTokenCandidateReadable` either.

Owner: lane B/assigned repair owner for the helper, root for required-resource collection.
Use only validated observed Secrets, persist their actual UID in publisher handover intent,
and compare observed predecessor/candidate/Secret UIDs before retirement. Check the exact
expected store/remote output entries and current generation acknowledgement. Reject
nonempty token deletionTimestamp. Remove status-byte fallback and synthetic owner defaults;
use provider-shaped token fixtures plus separate Secret observations. Keep token bytes in
memory only. Tests must invalidate each UID, output binding, and deletion witness independently.

### F4 - HIGH: unknown timing can become a falsely safe deadline

`rotatingTokenTimingForObserved` (`access.go:783-820`) estimates every token's expiry from
creationTimestamp plus lifetime without verifying renderer-created versus imported identity,
unchanged creation parameters, or a timestamp in the future. Imported credentials may have
expired well before their Kubernetes object was created. `rotatingTokenCandidateReadable`
(`868-871`) accepts candidates up to rotation-window start, while the packet forbids a
candidate already inside its earlier preparation window. F1 also bypasses all timing checks.

Owner: lane B/assigned repair owner. Gate estimates on credible mint provenance and
immutable creation settings, reject future/inconsistent times, and keep unsupported timing
unknown. Qualify C only before its preparation deadline. Retain the observed full spec and
provider/deletion bindings. Test imported/future creation and preparation-window candidates.

### F5 - HIGH: deletion-prepared does not mean resolved family handover

`rotatingTokenFamilyDeletionPrepared` (`access.go:940-1012`) allows a non-due selected legacy
member with an extant pending successor: no general outstanding-candidate check exists.
That state can arise during early preparation or recovery. For a promoted member it checks
absence of predecessor and a retirement UID, but not full key/candidate/Secret intent.
The condition at `1005-1008` requires both booleans whenever either gate is enabled; a
fleet family needing only ConsumerReady is falsely blocked by StatusReferenceReady=false.

Owner: lane B/assigned repair owner, with root switching callers. Reject every unresolved
candidate/publication/retirement state, validate exact complete selected lineage and
publisher binding, and independently gate `(RequireConsumer && !ConsumerReady)` and
`(RequireStatusReference && !StatusReferenceReady)`. Cover legitimate legacy-free promotion,
pending early candidate, malformed intent, and consumer-only/status-only descriptors.

### F6 - HIGH: generation suffix can be truncated out of the Kubernetes name

`boundedRotationName` (`access.go:751-758`) concatenates then truncates at 63 bytes.
`addRotatingTokenCandidate` appends its suffix to existing names. For a maximum-length
legacy Kubernetes name the entire generation suffix is discarded, so distinct logical
keys refer to the same managed object. Long Secret names/namePrefix values have the same
collision risk, violating distinct minting before retirement.

Owner: lane B/assigned repair owner. Reserve space for the generation discriminator before
truncating the base, use a deterministic bounded prefix, and assert distinct object,
connection-Secret and minting names at boundary lengths across consecutive generations.

### F7 - HIGH: no integration exists for the five real emit sites

Source search finds no production `reconcileRotatingTokenFamily` call outside its definition.
`fn.go:828-830,1043` and `expiry.go:169-171` still use legacy-only deletion readiness.
`bootstrap.go` and `fn.go` contain no `tokenConnectionSecrets`/`CredentialRotationHealthy`
implementation. Fleet, standalone consumer and service-account renderers are unchanged.
All-site tests at `access_test.go:301-321` pass five labels to the helper with
PublicationOnly=true and the same access-policy-shaped fields; they do not execute the
five real render sites or administrator/fleet consumer and status transitions.

Owner: root for its wiring/status/bootstrap/schema/RBAC verification seams; lane B/assigned
repair owner for its renderer, expiry and test files. Implement the accepted packet, not
just the incomplete wiring prose. Preserve generation inventory before parent-ID/readiness
fallthrough, validate generation-aware consumer selectors, gather required Secrets, bind
consumer ProviderConfig and ExternalSecret output to actual bytes and owner UID, publish
and observe status references, prune retired incoming keys, then set final condition/Ready.
Verify the pinned offline Secret-read authority instead of accepting the packet's assertion.
The wiring instruction to emit one condition of the same type per family should become
one deterministic aggregate condition with all family diagnostics. Provide real all-site
replay evidence, including old consumer bytes and old observed composite status.

## Lane C scope and blast-radius verdict

The three profile scopes are bounded to the consumer: `metrics:write`, `logs:write`, and
`traces:write` each correspond to its documented telemetry signal. No read, stack-inventory,
admin or unrelated signal scope was added. The fleet-wide telemetry list was not widened.
`stackConsumerPolicy` binds the policy realm to the observed authorized stack identity.

The shipped profile fixes namespace `telemetry-consumers`, target `robk` in
`prod-gb-south-1`, and platform-owned provider/output settings. Admission checks exact
namespace at `stack-consumer-v1beta1.yaml:148`, exact target immediately after it,
fails closed when parameters are absent, and retains renderer-side checks in
`configuredStackConsumerProfile`. An unintended namespace cannot use this profile via
an ordinary request. Cluster administrators able to change the Composition remain trusted.

The output path `/platform/grafana-cloud/consumers/robk-telemetry` has no other shipped
profile collision in the inspected manifests. Admission and renderer both reject duplicate
configured output paths/consumer names. This is repository configuration evidence, not
an inventory of any external secret store. Slug, region, and generic cluster-local names
fit the frozen publication boundary; no refused numeric identity, private repository or
estate/project identity was introduced by the inspected lane C diff.

Lane C may be accepted independently for this bounded security review. It does not fix
rotation. Its tests' synthetic provider identities prove renderer shape, not live minting.

## Evidence limits and handoff

This is a source review, with exact file hashes and concrete counterexample traces. I did
not run tests or create a second implementation. Lane B's recorded green focused command
and successful production build are worker evidence only; the replay flaw prevents them
from proving the accepted invariant. No failing-then-passing test claim is made by this lane.

GCV-0075 AC3, AC4 and AC5 are not satisfied by the reviewed slice. Root must assign repair
ownership, retain consumed attempts, obtain focused failing-then-passing replay evidence,
complete the missing integration, and request fresh review of the corrected accumulated
slice before accepting rotation. The ordinary full/hosted gates remain root-owned.

I wrote only this review file. I did not implement corrections, edit tracker/source,
stage, commit, push, delegate, run CodeRabbit or the full gate, touch any live Grafana or
cluster, or perform release/tag/PR actions. Live minting, provider/ESO reconciliation,
consumer reload and external revocation are **not exercised, and not exercisable here**.
