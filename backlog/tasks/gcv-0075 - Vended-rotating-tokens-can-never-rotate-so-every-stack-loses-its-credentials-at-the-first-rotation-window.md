---
id: GCV-0075
title: >-
  Vended rotating tokens can never rotate, so every stack loses its credentials
  at the first rotation window
status: Parked
assignee: []
created_date: '2026-09-16 16:48'
updated_date: '2026-09-18 19:02'
labels: []
dependencies: []
references:
  - >-
    https://github.com/grafana/terraform-provider-grafana/blob/main/internal/resources/cloud/resource_cloud_stack_service_account_rotating_token.go
  - >-
    https://github.com/grafana/terraform-provider-grafana/blob/main/internal/resources/cloud/resource_cloud_access_policy_rotating_token.go
priority: high
type: bug
ordinal: 75000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Every stack this platform vends gets a StackServiceAccountRotatingToken, and any stack with telemetry or fleet access gets AccessPolicyRotatingToken as well. Neither can ever rotate under Crossplane, so both eventually expire and are never replaced.

The upstream resources implement rotation as a Terraform replace. `ready_for_rotation` is declared Computed and ForceNew, and a CustomizeDiff sets it to true once the clock passes `expiration - early_rotation_window`. Terraform reacts by destroying and recreating the token, and the replacement IS the rotation. Upjet never replaces a managed resource; it refuses the update and reports CannotUpdateExternalResource with 'refuse to update the external resource because the following update requires replacing it: cannot change the value of the argument "ready_for_rotation" from "" to "true"'. The field is provider-internal and absent from the CRD, so nothing in this repository can set, suppress or patch it.

Observed live on 2026-09-16 across two estates running this platform. On the first estate both rotating tokens went Synced=False the minute their early-rotation window opened, four days before expiry, and had accumulated 5610 CannotUpdateExternalResource events by the time this was found. The second estate has not reached its window yet and is on a newer provider with an identical CRD schema, so it is expected to fail the same way about two weeks later.

Nothing surfaces this. The composite reports Synced=True and Ready=True, and each stuck token reports Ready=True. Only the child's Synced condition carries the failure, so a stack looks healthy right up to the moment its credentials stop working.

The consequence is the credential handover this platform is built on. The administrator token feeds the per-stack ProviderConfig through PushSecret, the secret store and ExternalSecret, so when it expires every in-stack resource stops reconciling. The telemetry and fleet tokens are what collectors authenticate with, so those expiring stops ingestion.

This needs a decision, not a patch, and every option is unattractive: delete and recreate each token out of band before its window opens, lengthen the lifetime so the window is never reached in practice and accept weaker hygiene, or get the behaviour changed upstream. A future agent cannot recover any of this from the code, because the repository only ever sets expireAfter, secondsToLive and the early rotation window.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The failure is reproduced against the provider version this repository pins, with the resource kinds, the exact provider message and the timing relative to the early-rotation window recorded as evidence
- [x] #2 The chosen handling of rotation is recorded as a decision with its trade-offs, covering both rotating token kinds the composition emits
- [ ] #3 The chosen handling works for an estate that has already passed a rotation window, not only for a stack vended after the change
- [ ] #4 An operator can distinguish a stuck token from a healthy one from resource conditions alone, without reading provider logs
- [ ] #5 The composite no longer reports Ready=True while a token it owns cannot rotate, or the reason it still does is recorded
- [x] #6 The upstream position is recorded: whether a provider or Upjet change is required, and whether it has been raised
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 13: security design packet first; root acceptance gates renderer implementation across all three rotating-token kinds, followed by adversarial security review and integrated evidence.

Root accepted Lane A packet: overlapping token generations with create-observe-publish-consumer-handover-retire ordering. Goal ownership repaired to include additive active-secret status refs and bootstrap consumers; root retains shared RunFunction, status-condition, RBAC, and validation wiring.

Wave 14: repair the accepted rotation design against security findings 1-5, implement the replayed handover across all five emit sites, integrate root-owned wiring, and obtain adversarial review before the gate.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
2026-09-18 live re-verification and three corrections. Read this before starting.

THE UPSTREAM MECHANISM IS CONFIRMED, from source rather than from inference.

In the provider's Terraform resource, ready_for_rotation is Computed AND ForceNew, and CustomizeDiff calls SetNew('ready_for_rotation', true) once either has_expired is true or Now() is after expiration minus early_rotation_window. UpdateContext is wired to the READ function, so an update changes only Terraform state and never touches Grafana. Rotation is therefore expressible ONLY as a Terraform replacement.

Upjet refuses exactly that. In pkg/controller/external_tfpluginsdk.go the Update path calls assertNoForceNew() and returns 'refuse to update the external resource because the following update requires replacing it' before it ever applies. That is a deliberate design position, not a bug: a Crossplane managed resource's external lifecycle belongs to Crossplane, and Upjet will not delete an external resource from inside Update.

So 'it cannot rotate under Crossplane' is precise about the UPDATE path and misleading as a general statement. Replacement is available one layer up: deleting and recreating the managed resource creates a new token, which is what Terraform's destroy-and-create does. That is the shape of the fix, and it is already proven - see below.

CORRECTION 1: THREE KINDS, NOT TWO. The description says 'both rotating token kinds'. The composition emits three:
  StackServiceAccountRotatingToken   platform/function/fn.go:632, the administrator token
  AccessPolicyRotatingToken          platform/function/access.go:86, fleet.go:172, stackconsumer.go:213
  ServiceAccountRotatingToken        platform/function/serviceaccounts.go:111, the in-stack tokens from GCV-0051
The third is in the oss group rather than cloud and was missed entirely. Whether its upstream resource carries the same ForceNew ready_for_rotation field has NOT been verified - check it rather than assuming, in either direction.

CORRECTION 2: THE STUCK TOKENS WERE ALREADY REMEDIATED, BY RECREATION, AND IT WORKED. Every rotating token on both estates now reports Synced=True with ReconcileSuccess and Ready=True, and there is not one CannotUpdateExternalResource event against any token kind on either cluster. The administrator token on the estate that was stuck has a creationTimestamp of 2026-09-16T16:56:43Z, which is the same minute this task was filed, so it was deleted and recreated as the immediate remediation. The 5610 accumulated events are gone with the object that produced them.

This is the single most useful fact available for choosing a direction: delete-and-recreate of the managed resource DOES rotate the token, observed, on a live estate, at the pinned provider. The question is no longer whether Crossplane can do it but whether the platform does it deliberately or an operator keeps doing it by hand.

CORRECTION 3: THE DEADLINE IS NOT IMMINENT, and any plan that assumes it is will be wrong. Live values, both estates, read 2026-09-18:
  secondsToLive 2592000, thirty days
  earlyRotationWindowSeconds 604800, seven days
  estate A administrator token expires 2026-10-10, window opens 2026-10-03
  estate B administrator token expires 2026-10-16, window opens 2026-10-09
So the next failure is roughly two weeks out, not days. There is room to do this properly. The recreation on 2026-09-16 bought that room and it will run out again on the same thirty-day cycle.

AC4 IS HARDER THAN IT LOOKS for one of the three kinds. StackServiceAccountRotatingToken publishes expiration, hasExpired, secondsToLive and earlyRotationWindowSeconds in status.atProvider. AccessPolicyRotatingToken publishes NONE of them - all read null on live objects - so there is nothing on the resource from which to compute how close it is to its window. Whatever satisfies AC4 has to work without that field, or has to get it published.

Provider versions differ across the two estates, v2.14.0 on one and a v2.13.0 build on the other, and the behaviour and CRD schema were identical on both, so this is not version-specific.

Lane A verified the third ServiceAccountRotatingToken has the same Computed ForceNew ready_for_rotation mechanism at the provider pin. Design packet: codex/wave13/lane-a-packet.md.

Wave 13 security review found material safe-ordering defects in the attempted A2 implementation: publication deadlock, candidate withdrawal, unsafe lineage fallback, wrong consumer ProviderConfig identity, and dishonest unknown-expiry handling. The A2 code was removed. Resume by correcting the accepted design packet against lane-f-review.md, then reimplement and replay the complete desired-to-observed lifecycle before any publish. AC2 applies to all three emitted rotating-token kinds, including ServiceAccountRotatingToken.

Wave 13 terminal reconciliation: the A2 implementation was rejected and removed after lane-f-review.md found material safe-ordering defects. Resume only after repairing the design for publication ordering, observed candidate retention, lineage, consumer identity, unknown expiry, and root wiring across all three token kinds.

Wave 14 exhausted all four authorized implementation attempts: wave 13 attempt 1, lane B attempts 2 and 3, and root attempt 4. The fresh adversarial review still found seven high-impact defects: second-window completion is blocked; the standalone consumer rejects the selected successor before wiring; ambiguous evidence can arm deletion; publication acknowledgement is not bound to the authorized output; deletion readiness accepts unknown timing and bypassed handover; status can advance before materialized consumer handover; and health accounting omits unrendered families. The rejected rotation source and documentation were restored to the integration base and are absent from the landing slice. Resume only with explicit authorization for another attempt and implement the exact owner corrections in codex/wave14/lane-f-review.md.

2026-09-18 OWNER DECISION after wave 14: the task is SPLIT and the rotation direction is RE-OPENED FOR DESIGN. No fifth implementation attempt is authorised against the accepted overlapping-generation packet.

WHY, because this is the expensive finding and it must not be re-litigated. Three attempts were made against that packet: lane B twice and the root once. Lane F's prior-finding disposition table records the outcome precisely - all seven of its first-review findings came back 'Partially corrected, still FAIL', and all five wave-13 hard boundaries remained unsatisfied after each attempt. That is a repeating failure signature against the DESIGN, not three independent implementation failures. A fifth attempt at the same zero-gap overlapping-generation handover is expected to fail the same way.

THE SPLIT.
- The operator-visibility half, formerly AC4 and AC5, moves to GCV-0087. It carries no credential-safety invariant, does not depend on how rotation is eventually performed, and is the only thing that would have caught the original incident. It lands first.
- Fleet-level health through Crossplane's own telemetry surfaces becomes GCV-0088, scoped by the owner to what Crossplane provides out of the box and nothing this repository invents.
- AC3, rotation that works for an estate already past its window, stays here and is design-only until a direction is accepted.

THE DIRECTION TO EVALUATE FIRST, and it is already evidenced in this task rather than speculative: delete-and-recreate of the managed resource DOES rotate the token, observed live on 2026-09-16 at the pinned provider, when the stuck tokens on both estates were remediated that way and every one now reports Synced=True with no CannotUpdateExternalResource events. Controlled replacement accepts a brief credential gap. The overlapping-generation design existed to eliminate that gap, and eliminating it is what has failed three times. Whether the gap is acceptable is the decision, and it has not been taken.

The seven Lane F findings in codex/wave14/lane-f-review.md are no longer a repair checklist. They are the test any candidate direction must survive, and a direction that cannot produce them as non-problems is a better answer than one that answers them individually.

Timing, so no plan assumes a cliff: the windows reopen on the thirty-day cycle the 2026-09-16 recreation restarted. Manual delete-and-recreate remains the proven fallback and needs no code.

Wave 15 design packet recommends controlled replacement only where the owner explicitly accepts interruption for that consumer and token type. Keep operator-controlled replacement as the proven fallback. Where uninterrupted authentication is mandatory, pursue provider-owned renewal instead. Do not automate universally and do not retry the rejected overlapping-generation design. Owner decision still required: which consumers accept interruption, the maximum acceptable interruption or telemetry loss, and the recovery evidence required before automation. GCV-0075 remains Parked and no acceptance criterion was satisfied.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Parked after design and adversarial review. Provider replacement behavior, all three emitted token kinds, the overlapping-generation decision, and the upstream Upjet boundary are recorded. No rotation implementation remains in the tree. Resume from codex/wave13/lane-f-review.md. The integrated repository passed just check locally and hosted Validate run 35333659147 at b47831ddddfd5ec10e1e699d4d8608886261becd, but AC3 through AC5 remain unproven.

Parked after four of four authorized attempts. No rotation implementation landed. The accepted design packet, implementation evidence and two-phase adversarial review preserve the precise seven-edit resume boundary.

Wave 15 reopened the rotation direction without an implementation attempt. Recommendation: controlled replacement with explicit per-consumer interruption acceptance; operator-controlled fallback; provider-owned renewal when interruption is unacceptable.
<!-- SECTION:FINAL_SUMMARY:END -->
