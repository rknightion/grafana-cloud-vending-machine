---
id: GCV-0046
title: 'Vend on-call escalation chains, routes and integrations'
status: In Progress
assignee: []
created_date: '2026-09-08 22:36'
updated_date: '2026-09-08 23:59'
labels: []
dependencies:
  - GCV-0045
priority: high
type: feature
ordinal: 46000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The half of IRM that turns a firing alert into a page. Escalation chains, escalations, routes and integrations are all present in the pinned provider and none is vended. Without them a vended schedule is decorative: nothing routes an alert onto it.

This is the seam where the alerting bundle and IRM meet, so the integration boundary matters more than either surface alone. An integration receives alerts, a route matches them, an escalation chain decides who is woken and in what order. A vending machine that must fail closed cannot ship a chain with an unreachable final step, and cannot let a route silently match nothing.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 An integration, a route and an escalation chain are vended per stack so that an ordinary alert rule owned by GrafanaAlertingRouting reaches the vended schedule through its routing tree and contact point, with every configured link proven at the real API server; existing GrafanaAlertingBundle direct-notification semantics remain unchanged
- [ ] #2 An escalation chain whose final step reaches nobody is refused by the XRD with its own message, proven against the real API server
- [ ] #3 A route that matches no vended alert, and a chain that references an unvended schedule, are both refused with their own messages
- [ ] #4 Every emitted kind appears in the provider activation map and the XRD/renderer registry, and the gate fails by path if one is missing
- [ ] #5 A catalog example renders inert and is covered by the catalog README and Kustomization checks
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

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Root section-8 correction, HIGH materiality: the original AC1 named the legacy alerting bundle. Pinned source proves its per-rule notificationSettings bypass the routing tree, contrary to the goal starting-state narrative. The joined-path design therefore uses ordinary rules in the new routing surface while preserving frozen legacy files. Original criterion: An integration, a route and an escalation chain are vended per stack so that an alert from the vended alerting bundle reaches the vended schedule end to end, with the linkage proven rather than asserted. This corrects the subject of the API-server proof; it does not claim live evaluation, email delivery or paging. Reversal requires changing a new public surface or reopening the frozen legacy seam.
<!-- SECTION:NOTES:END -->
