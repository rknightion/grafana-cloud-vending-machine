---
id: GCV-0045
title: Vend on-call schedules and shifts
status: Done
assignee: []
created_date: '2026-09-08 22:35'
updated_date: '2026-09-09 00:53'
labels: []
dependencies: []
priority: high
type: feature
ordinal: 45000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Of 26 OnCall kinds in the pinned provider, this machine emits five, and effectively only OutgoingWebhook carries on-call meaning. A vended stack therefore gets alerting with nobody on the other end of it. Schedules and shifts are the base layer: escalation chains and routes reference a schedule, so this task has to land before them.

The provider carries oncall schedules, oncallshifts, usergroups, users and usernotificationrules at v2.14.0. Note the identity problem this repository has already hit twice: OnCall user and user-group identities are assigned by the provider and cannot be derived from a request. Read the provider-assigned-IDs rule in the Wave operating model document before designing the reference shape - the renderTeamAccess wait-for-observed-ID pattern is the one to copy.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A per-stack on-call schedule with rotating shifts is vended from platform-controlled input, with an explicit external name so a re-adopt does not attempt a create
- [x] #2 No provider-assigned OnCall user, user-group or schedule ID is guessed; anything that depends on one emits nothing until the ID is observed, and a test proves the wait
- [x] #3 A schedule with no reachable responder is refused by the XRD with its own message, proven against the real API server
- [x] #4 Every emitted kind appears in the provider activation map and the XRD/renderer registry, and the gate fails by path if one is missing
- [x] #5 A catalog example renders inert and is covered by the catalog README and Kustomization checks
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
Vended OnCall users observed by provider identity, rotating shifts and schedules with explicit UTC start and singleton responder rotation groups. No assigned ID is guessed; dependent resources wait and preserve observed external names. Assigned IDs have no fabricated external name at first creation. Real XRD and complete provider-CRD admission/readback passed. Completing source/pin SHA: 187b03ea40ee32fcea890e40c138f00a8c73bd5f. Hosted Validate run 34296536930: success. Local just check passed with 23 real API-server tests, zero skips.
<!-- SECTION:FINAL_SUMMARY:END -->
