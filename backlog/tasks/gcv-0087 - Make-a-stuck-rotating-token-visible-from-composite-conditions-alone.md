---
id: GCV-0087
title: Make a stuck rotating token visible from composite conditions alone
status: To Do
assignee: []
created_date: '2026-09-18 16:44'
updated_date: '2026-09-18 16:47'
labels:
  - needs-triage
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
- [ ] #1 A composite owning a rotating token that cannot rotate does not report Ready=True, or the exact reason it still does is recorded
- [ ] #2 A stuck token is distinguishable from a healthy one from resource conditions alone, with no access to provider logs
- [ ] #3 The signal works for AccessPolicyRotatingToken, which publishes no expiration, secondsToLive or earlyRotationWindowSeconds in status.atProvider
- [ ] #4 All three emitted rotating-token kinds are covered, and a configured family that renders no token is accounted for rather than dropping out of the health roll-up
- [ ] #5 The signal is derived only from observed child conditions and status this repository already receives, with no new provider field and no instrumentation added to the composition function
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
