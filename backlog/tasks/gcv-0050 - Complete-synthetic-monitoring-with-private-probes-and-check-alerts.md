---
id: GCV-0050
title: Complete synthetic monitoring with private probes and check alerts
status: In Progress
assignee: []
created_date: '2026-09-08 22:36'
updated_date: '2026-09-08 22:48'
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
- [ ] #2 No probe token value reaches status or a rendered example; only derived credentials are published, and a test proves it
- [ ] #3 Check alerts are vended so a failing vended check reaches the vended alerting path, with the linkage proven rather than asserted
- [ ] #4 The existing check budget still binds after the addition, and an over-budget request is still refused with its own message
- [ ] #5 Every emitted kind appears in the provider activation map and the XRD/renderer registry, and the gate fails by path if one is missing
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 6: implement the commissioned surface under the frozen goal and root-owned integration; prove admission and renderer boundaries with required negative controls, then just check and exact-SHA hosted Validate before finalization.
<!-- SECTION:PLAN:END -->
