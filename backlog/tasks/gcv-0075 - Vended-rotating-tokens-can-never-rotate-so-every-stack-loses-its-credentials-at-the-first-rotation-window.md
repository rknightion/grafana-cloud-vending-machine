---
id: GCV-0075
title: >-
  Vended rotating tokens can never rotate, so every stack loses its credentials
  at the first rotation window
status: In Progress
assignee: []
created_date: '2026-09-16 16:48'
updated_date: '2026-09-18 09:24'
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
- [ ] #1 The failure is reproduced against the provider version this repository pins, with the resource kinds, the exact provider message and the timing relative to the early-rotation window recorded as evidence
- [ ] #2 The chosen handling of rotation is recorded as a decision with its trade-offs, covering both rotating token kinds the composition emits
- [ ] #3 The chosen handling works for an estate that has already passed a rotation window, not only for a stack vended after the change
- [ ] #4 An operator can distinguish a stuck token from a healthy one from resource conditions alone, without reading provider logs
- [ ] #5 The composite no longer reports Ready=True while a token it owns cannot rotate, or the reason it still does is recorded
- [ ] #6 The upstream position is recorded: whether a provider or Upjet change is required, and whether it has been raised
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
<!-- SECTION:NOTES:END -->
