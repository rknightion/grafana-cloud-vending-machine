---
id: GCV-0087
title: Make a stuck rotating token visible from composite conditions alone
status: Done
assignee:
  - '@codex'
created_date: '2026-09-18 16:44'
updated_date: '2026-09-18 19:02'
labels: []
dependencies: []
ordinal: 87000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Split out of GCV-0075. When a vended rotating token cannot rotate, nothing surfaces it: the composite reports Synced=True and Ready=True, each stuck token reports Ready=True, and only the child's Synced condition carries the failure. On the estate where this was first found, thousands of CannotUpdateExternalResource events accumulated against a token while every composite looked healthy.

This is the half of GCV-0075 that carries no credential-safety invariant and does not depend on how rotation is eventually performed. It makes the failure visible whether rotation is fixed by controlled replacement, by a longer lifetime, or by an operator acting out of band. It is therefore worth landing before the rotation direction is settled, and it is the only thing that would have caught the original incident.

The constraint, set by the owner: derive the signal only from what Crossplane already provides. Observed child conditions and the status this repository already receives. No new provider field, no custom instrumentation of the composition function, no metric this repository invents.

One kind makes this harder than it looks. StackServiceAccountRotatingToken publishes expiration, hasExpired, secondsToLive and earlyRotationWindowSeconds in status.atProvider. AccessPolicyRotatingToken publishes none of them; all four read null on live objects, so there is nothing on the resource from which to compute how close it is to its window. Whatever satisfies this has to work without that field rather than assume it.

The composition emits three rotating-token kinds, not two: StackServiceAccountRotatingToken, AccessPolicyRotatingToken and ServiceAccountRotatingToken. A configured family that renders no token at all must be accounted for rather than silently disappearing from the health roll-up, which is the defect the wave 14 review found in the rejected implementation.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A composite owning a rotating token that cannot rotate does not report Ready=True, or the exact reason it still does is recorded
- [x] #2 A stuck token is distinguishable from a healthy one from resource conditions alone, with no access to provider logs
- [x] #3 The signal works for AccessPolicyRotatingToken, which publishes no expiration, secondsToLive or earlyRotationWindowSeconds in status.atProvider
- [x] #4 All three emitted rotating-token kinds are covered, and a configured family that renders no token is accounted for rather than dropping out of the health roll-up
- [x] #5 The signal is derived only from observed child conditions and status this repository already receives, with no new provider field and no instrumentation added to the composition function
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 15 lane B: implement condition-only rotating-token health in two new feature files; root applies the fn.go registration; security review and integrated gate follow.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Wave 15 implemented exact-current-child credential health by GVK and identity. Missing children, missing Synced conditions and Synced values other than True now make the owning composite not Ready. AccessPolicyRotatingToken needs no expiry fields. All three emitted rotating-token kinds and configured-but-unrendered families are covered from observed child conditions only. CodeRabbit and Lane E security findings were corrected before landing. Local just check passed with 85.3 percent statement coverage; source SHA fda11ea36184af9c7ebfdd1899db136738fedc4f and completing pin SHA 1c13c27039068f4eacede61f6664dd498cff0d9a both passed hosted validation.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Completed condition-only rotating-token health across all three emitted kinds. Exact current child membership prevents retired or unrelated tokens from masking or poisoning health, and configured missing children fail closed. Verified by local just check and hosted Validate run 35382961277 at 1c13c27039068f4eacede61f6664dd498cff0d9a.
<!-- SECTION:FINAL_SUMMARY:END -->
