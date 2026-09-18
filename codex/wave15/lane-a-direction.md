# Wave 15 lane A: credential-rotation direction

Status: design complete; operational gap acceptance remains an owner decision.
GCV-0075 remains Parked. No acceptance criterion is satisfied by this packet, no
implementation attempt is consumed, and this packet gates no other lane.

## Recommendation

Choose controlled replacement with an explicitly accepted interruption, separately
opted into for each consumer. Keep operator-controlled replacement as the current
fallback. Do not authorize automatic replacement, a universal outage budget, or a
fifth attempt against the overlapping-generation packet on this evidence.

This selects a simpler direction, not a claim that its outage is acceptable for
every workload. The administrator path is a reasonable candidate for a planned
reconciliation pause. Collectors require an accepted loss/recovery budget. In-stack
consumers require their own availability decision. Where a consumer requires
uninterrupted authentication, pursue a provider-owned credential lifecycle contract;
do not reinstate composition-owned overlapping generations to meet that requirement.

Prefer making the generation-specific failure modes structurally impossible over
answering them one by one. Controlled replacement removes successor selectors,
predecessor retirement and generation references. It still needs trustworthy
recovery evidence, independent health accounting and explicit destructive authority.

## Evidence and limits

Read in full: `codex/wave14/lane-f-review.md`, including the fresh review, prior
disposition table and historical first review; `codex/wave14/lane-a-packet.md`; and
GCV-0075 through `backlog task view GCV-0075 --plain`. Read section 2.5 and the lane A
contract in `codex/goal-2026-09-18-wave15.md`. Reused root's Backlog overview.

| Input | SHA-256 at inspection |
|---|---|
| Wave 14 lane F review | `11493e953e5345cd48605a95fa81693c13a1366bdc9bbfdec8a367ac74b062a8` |
| Wave 14 lane A packet | `f52fdcfe4e9641fd03c7c201621332c9bc5d70c706b45d484c8d06b60fb31051` |
| Wave 15 goal | `bf65ef85742573647fab346b909f8f76d576bed39727fc94510be920be8fee90` |

The frozen owner finding is a repeated design failure: four authorized attempts
are exhausted and all five hard boundaries remain unsatisfied. The review's local
success for bounded naming does not reopen the rejected design or alter that decision.
The historical F labels and fresh R labels identify different findings; both are
covered below rather than treating R6 as the historical naming finding.

Recorded evidence, not a new live verification: GCV-0075 says managed-resource
delete-and-recreate restored token reconciliation on 2026-09-16 at the pinned
provider. The source evidence records Terraform ForceNew rotation and Upjet's
refusal to replace an external resource inside Update; the prior packet verified
the third, OSS token kind too. These establish a replacement mechanism and a past
recovery, not a maximum interruption, consumer reload, data preservation or a
repeatable automated procedure. All five emit sites and all three kinds matter.
`AccessPolicyRotatingToken` has no provider expiry fields. This packet does not
re-fetch documentation or re-derive the frozen provider evidence.

## Candidate directions and failure modes

**A. Controlled replacement, accepting an interruption.** Replace one explicitly
identified managed token at a time, keeping its logical resource and publication
bindings stable. There is no concurrent managed successor, selector promotion or
old-generation retirement protocol. Replacement deliberately gives up uninterrupted
authentication; it must not advertise a bounded duration without evidence. Minting
failure, finalizers, delayed secret propagation and consumers retaining old bytes
can extend the interruption indefinitely. A same-named Secret can contain stale
credentials. Stable naming does not prove current contents or successful use.

Replacement authority must identify the exact object incarnation and preserve the
existing external lifecycle policy. Under Preserve, deleting the managed resource
may leave the old external credential valid until expiry; that is neither revocation
proof nor a reliable no-gap mechanism. Under Delete, revocation can start the outage
before a replacement exists. This direction does not silently change Preserve to
Delete. It also must not confuse permission to replace one token with permission to
delete a stack. An unfinished replacement must be resolved before another starts.

**B. Provider-owned credential renewal behind one managed identity.** Seek an explicit
provider contract for repeatable renewal and connection-secret update, with accurate
expiry/progress evidence. If uninterrupted service is required, that contract must
also define old-credential validity and acknowledgement by the real consumer; moving
minting into the provider does not itself prove handover. An upstream design must
respect Crossplane lifecycle ownership, rather than simply removing Upjet's ForceNew
guard. This is a legitimate longer-term direction, but its API, availability and
delivery are unproven here. Failure modes include provider restart during renewal,
lost one-time credential material, stale published bytes, and premature revocation
outside the provider's view of consumers.

**C. Composition-owned overlapping generations, the rejected control.** This requires
the full create-observe-publish-consumer-status-retire protocol in the old packet.
Partial observation, restart, unknown timing and renderer fallthrough remain coupled
to destructive decisions. All seven fresh defects remain open for that direction.
It is compared to explain rejection, not offered as another authorized attempt.

Lengthening lifetime is not a fourth solution: it delays the same boundary and
extends credential exposure without establishing rotation.

## Test against all seven fresh Lane F findings

Labels describe design consequences only. **Structurally impossible** means the
specific failed transition does not exist; **answerable** means a narrower obligation
remains and requires later evidence; **still open** means this evidence cannot settle
the contract. No label is an implementation pass.

| Fresh finding | A: controlled replacement | B: provider-owned renewal | C: rejected overlap |
|---|---|---|---|
| R1: completed lineage blocks the second window | **Structurally impossible** for predecessor/selected-generation deadlock: there is no lineage or retirement state. A second replacement and restart recovery still require proof. | **Answerable** by a provider contract for repeated renewal and restart; no such tested contract is present. | **Still open**: completed lineage blocks the next window. |
| R2: consumer renderer rejects successor before wiring | **Structurally impossible** for selected-successor rejection: bindings stay stable and there is no successor selector. Adoption/configuration changes remain separate validation concerns. | **Structurally impossible** at the composition boundary if the promised single identity and bindings remain stable; provider compatibility is still unproven. | **Still open**: the actual renderer rejects its promoted child. |
| R3: ambiguous evidence arms deletion | **Answerable**: explicit replacement authority is required; ambiguity cannot create destructive authority or rewrite lifecycle policy. Deletion is intentional here, so the general hazard is not impossible. | **Still open**: the provider's lifecycle/revocation contract is unspecified. | **Still open**: preservation can rewrite deletion settings. |
| R4: acknowledgement is not bound to authorized output | **Answerable**: exact store, output, token/Secret incarnation and consumer recovery evidence are still needed. Stale success cannot close an interruption. | **Still open**: provider secret update does not establish remote publication or reload. | **Still open**: acknowledgement can describe the wrong output. |
| R5: deletion readiness bypasses timing and handover | **Answerable**: explicit acceptance permits replacement before handover; it does not assert safe no-gap retirement or stack deletion readiness. Unknown expiry remains unknown and blocks automatic timing claims. | **Still open**: old-token retirement and consumer acknowledgement are unspecified. | **Still open**: unknown timing and bypassed handover can authorize retirement. |
| R6: active status advances before materialized handover | **Structurally impossible** for generation-reference promotion: the reference never changes. Claiming recovered service before actual consumer agreement remains prohibited under R4. | **Structurally impossible** for generation-reference promotion with stable bindings; equivalent early success remains an open provider/consumer issue. | **Still open**: status can point ahead of materialized credentials. |
| R7: missing renderer output disappears from health | **Answerable**: inventory configured families independently and report absence/interruption. Visibility work remains independently owned by GCV-0087. | **Answerable** at platform level; provider success cannot account for absent configured children. | **Still open**: omitted families can look healthy. |

The historical seven findings add concerns not fully represented by the fresh labels:

| Historical finding | A | B | C |
|---|---|---|---|
| F1: retirement/future windows | **Structurally impossible** for retirement lineage; repeatability remains answerable. | **Answerable** through provider renewal contract. | **Still open**, R1. |
| F2: selector rollback | **Structurally impossible**, one stable selector. | **Structurally impossible** with stable binding contract. | **Still open** as an overall preservation invariant, R3. |
| F3: Secret/publication witnesses | **Answerable**, R4. | **Still open**, R4. | **Still open**, R4. |
| F4: false timing/deadlines | **Answerable** for operator-controlled recovery without invented expiry; automatic scheduling remains **still open** for unknown timing. | **Still open** pending trustworthy expiry/provenance. | **Still open**, R1/R5. |
| F5: deletion predicate | **Answerable**, explicit gap acceptance is distinct from safe retirement and stack deletion. | **Still open**, lifecycle contract. | **Still open**, R5. |
| F6: generation suffix collision | **Structurally impossible**, no generated successor names. | **Structurally impossible** in composition; provider-side identity uniqueness is **answerable** upstream. | **Answerable**: fresh review records the local naming correction; it does not repair other failures. |
| F7: missing all-site integration | **Answerable**, all five sites still need recovery/visibility evidence. | **Still open** for provider coverage of all three kinds and end-to-end consumption. | **Still open**, R2/R6/R7. |

## Is a credential gap acceptable for each consumer?

**Administrator: conditionally yes for a planned reconciliation pause, subject to an
owner-approved duration and recovery requirement.** The token feeds the per-stack
ProviderConfig through PushSecret, the secret store and ExternalSecret. Invalid
credentials prevent in-stack resources from reconciling; pending repairs and changes
wait, including minting OSS in-stack tokens that use this provider. Existing workload
execution is not thereby proven unaffected. Recovery means the materialized provider
credentials work again, not simply that the managed token is Ready. Do not overlap
administrator replacement with dependent in-stack replacement. The recorded minting
identity is the organization provider, distinct from the affected consuming provider;
that avoids one apparent dependency cycle but proves no recovery deadline.

**Telemetry and fleet: no blanket acceptance.** `AccessPolicyRotatingToken` authenticates
collectors at telemetry, fleet and standalone-consumer sites. Invalid credentials can
stop ingestion or fleet authentication; buffers can fill and data can be lost. Fleet's
local provider credential field is an additional consumption path, not proof that
external collectors reloaded. Accept interruption only for consumers whose operator
explicitly accepts the resulting loss/recovery budget, or provides evidence of adequate
buffering and reload behavior. No such evidence exists here. Missing provider expiry
fields forbid pretending a known safe rotation deadline; an operator's deliberate
recovery decision is not an expiry estimate.

**In-stack tokens: undecided per declared consumer; default no automatic gap.**
`ServiceAccountRotatingToken` authenticates each declared in-stack consumer, whose
availability and reload behavior are not supplied. An API client may tolerate retries;
a critical integration may not. Publication-only treatment in the old packet does not
make that external impact disappear. Require its consumer owner to accept interruption
and define recovery, or route its uninterrupted requirement to direction B.

Thus the remaining question is concrete: which consumers accept interruption, what
maximum interruption and data loss do they accept, and what recovery evidence ends the
incident? This packet cannot grant those business/operational permissions. Lack of an
answer keeps replacement unautomated; it does not revive direction C.

## Discriminating acceptance check and reversal cost

A future authorized evaluation should demonstrate two successive controlled replacement
cycles, including an already-window-passed initial token, with one stable managed
identity and publication binding per family across all five sites. An offline lifecycle
trace must derive observations from prior actions, inject a restart and failed mint,
and show: interruption declared, exact-incarnation replacement authorized, recreation,
fresh credential publication, applicable consumer recovery, and return to service.
Missing clock, stale Secret, wrong remote acknowledgement or absent configured family
must not produce a recovered verdict. Do not substitute a manually fabricated token
absence for the replacement action. This is an acceptance contract, not source diffs
or permission to perform destructive operations.

The decisive discriminator from C is that success requires no simultaneous managed
generations, successor-selector switch, observed active-generation status promotion
or predecessor retirement gate. If those are needed to claim availability, the proposal
has returned to the rejected design. Conversely, an offline trace cannot prove a
real gap fits a consumer's budget. Consumer-owner evidence of interruption, reload and
loss is a separate prerequisite before operational adoption, obtained outside this
repository's inert workflow under separate authority. No live procedure is proposed
for this repository to execute.

Reversal before replacement is cheap: withhold replacement authority and retain current
resources. After revocation it is not rollback: a deleted credential cannot be restored
by reverting code, and recovery requires new valid credentials and consumer reload.
Stable logical bindings reduce schema/selector migration cost, not outage cost. Moving
later to B requires a reviewed provider contract and migration evidence; switching back
to C would require a new owner decision and is not the recommended fallback.

## Unsettled facts and next action

Unproven: maximum outage; revocation behavior under each existing lifecycle setting;
actual collector buffering/loss; every consumer's reload; automated repeated replacement;
exact safe timing for imported access-policy tokens; and any future provider guarantee.
The recorded live recovery is not evidence for these. Provider-native renewal may require
new APIs or controller behavior; this packet neither asserts that such support exists
nor asks Upjet to abandon its ownership rule.

Root should present direction A with these per-consumer decisions for owner acceptance.
If uninterrupted authentication is mandatory for a consumer, pursue B's contract there
and keep GCV-0075 Parked. No further repair attempt against C is recommended. Visibility
and native fleet-health work remain independent.

Only this packet was written. No source, tests, tracker, acceptance criteria or shared
state were changed; no commit, push, delegation, cluster tooling, Grafana contact,
CodeRabbit, package gate or hosted validation was performed. Tests were intentionally
not added for this design-only artifact.

Validation: reread the complete written packet against the brief; all seven fresh
and all seven historical findings have dispositions for each candidate. Inspected
the complete public-release scan script before execution: it uses local working-tree
and Git-history searches plus temporary negative controls, with no cluster or Grafana
contact. `just public-release-scan` exited 0 and printed `Public-release scan passed.`
Its refused-identifier negative controls and permitted-boundary controls passed.
This result covers the shared working tree at scan time, before this result paragraph
was appended; it is publication scanning, not rotation or hosted validation evidence.
