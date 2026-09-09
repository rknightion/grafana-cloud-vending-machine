---
id: GCV-0050
title: Complete synthetic monitoring with private probes and check alerts
status: Parked
assignee: []
created_date: '2026-09-08 22:36'
updated_date: '2026-09-09 00:53'
labels: []
dependencies: []
type: feature
ordinal: 50000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
GCV-0022 vended Synthetic Monitoring installation and a budgeted check set, and explicitly excluded private Probes and probe tokens - its AC4 was satisfied by that exclusion rather than by implementation. sm probes and sm checkalerts are both present in the pinned provider and neither is emitted.

Private probes are what let a vended stack check something that is not publicly reachable, which is the same gap PDC closes for datasources. Check alerts are the piece that makes a failing check page somebody rather than just going red on a dashboard, so this task also has a natural seam with the on-call work.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Private probes are vended per stack from platform-controlled input, with probe tokens bounded by the existing composition token expiry ceiling
- [x] #2 No probe token value reaches status or a rendered example; only derived credentials are published, and a test proves it
- [ ] #3 Check alerts are vended so a failing vended check reaches the vended alerting path, with the linkage proven rather than asserted
- [ ] #4 The existing check budget still binds after the addition, and an over-budget request is still refused with its own message
- [x] #5 Every emitted kind appears in the provider activation map and the XRD/renderer registry, and the gate fails by path if one is missing
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 6: implement the commissioned surface under the frozen goal and root-owned integration; prove admission and renderer boundaries with required negative controls, then just check and exact-SHA hosted Validate before finalization.
<!-- SECTION:PLAN:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
PARTIAL: CheckAlerts lands after observed numeric Check ID, with optional alerts preserving existing requests. Private probes remain refused on create/update, so no unbounded probe token is issued. Token non-disclosure is satisfied by exclusion; no private-probe vending claim. Renderer check-budget weaken/admit/restore passed, but exact usage-dependent API admission and actual alert delivery are unproven. Pinned provider/client source exposes no bounded private-probe token expiry. Resume: provider-side expiry contract plus commissioned cross-resource admission and alert-path proof. Do not replace remote token expiry with Secret expiry. Completing source/pin SHA: 187b03ea40ee32fcea890e40c138f00a8c73bd5f. Hosted Validate run 34296536930: success. Local just check passed with 23 real API-server tests, zero skips.
<!-- SECTION:FINAL_SUMMARY:END -->
