---
id: GCV-0050
title: Complete synthetic monitoring with private probes and check alerts
status: Done
assignee:
  - '@codex-wave7'
created_date: '2026-09-08 22:36'
updated_date: '2026-09-09 10:53'
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
- [x] #1 No probe token value reaches status or a rendered example; only derived credentials are published, and a test proves it
- [x] #2 Check alerts are vended so a failing vended check reaches the vended alerting path, with the linkage proven rather than asserted
- [x] #3 The existing check budget rejects declared values outside the Composition profile with its own message; reconciliation selects that profile from the referenced stack's observed usage, and direct provider writes remain outside this managed budget
- [x] #4 Every emitted kind appears in the provider activation map and the XRD/renderer registry, and the gate fails by path if one is missing
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 6: implement the commissioned surface under the frozen goal and root-owned integration; prove admission and renderer boundaries with required negative controls, then just check and exact-SHA hosted Validate before finalization.

Wave 7 settlement: record private-probe refusal as owner-approved design, amend only criteria describing unavailable mechanisms, prove the CheckAlerts linkage at the machine-controlled configuration boundary, then close with integrated local and exact-SHA hosted evidence.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Wave 7 owner settlement removed former AC1: Private probes are vended per stack from platform-controlled input, with probe tokens bounded by the existing composition token expiry ceiling. Provider evidence shows Probe has no token-lifetime field, so that mechanism does not exist; accepted decision decision-0002 records the shipped refusal and the provider-bump condition for reconsideration. Former AC4 before: The existing check budget still binds after the addition, and an over-budget request is still refused with its own message. After: The existing check budget rejects declared values outside the Composition profile with its own message; reconciliation selects that profile from the referenced stack's observed usage, and direct provider writes remain outside this managed budget. Source evidence is decision-0001, renderer budget tests, and the observed referenced-stack context. Root materiality: HIGH because both changes correct public compatibility claims before 1.0.0. AC3 proof is configuration-bound: the renderer accepts only a positive numeric provider-observed Check ID, emits CheckAlerts with that ID and the Synthetic Monitoring ProviderConfig, and the pinned provider CRD admits the exact rendered child. Rule evaluation and notification delivery remain unproven because CheckAlerts exposes no contact-point, policy, route, receiver, or label linkage.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
DONE: Synthetic Monitoring CheckAlerts are linked at the rendered, provider-admitted configuration boundary; evaluation and notification delivery remain explicitly unproven. Private probes are intentionally refused until the provider exposes bounded token lifetime, recorded in decision-0002. Completing source SHA b1ceaa36d63dc8b449215b9b362103d7da9b1dbf; hosted Validate public reference run 34337220901 succeeded; local just check passed with zero skips.
<!-- SECTION:FINAL_SUMMARY:END -->
