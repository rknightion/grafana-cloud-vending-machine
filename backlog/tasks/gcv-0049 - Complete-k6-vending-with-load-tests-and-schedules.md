---
id: GCV-0049
title: Complete k6 vending with load tests and schedules
status: In Progress
assignee: []
created_date: '2026-09-08 22:36'
updated_date: '2026-09-08 22:48'
labels: []
dependencies: []
type: feature
ordinal: 49000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
GCV-0021 vended the k6 Project, ProjectLimits and ProjectAllowedLoadZones as a blast-radius cap and deliberately stopped there: it provisions no tests and no schedules, and that exclusion is recorded in its acceptance evidence rather than being an oversight. The cap is now in place, so the surface it was capping can be vended.

The provider carries k6 loadtests, loadtestsets, schedules and schedulesets at v2.14.0, none of them emitted. The point of doing this after the cap rather than before is that every test and schedule must land inside the existing platform limits, so the interesting work is proving the cap actually binds them rather than adding the resources.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Load tests and schedules are vended per project from platform-controlled input, with an explicit external name on every deterministic child
- [ ] #2 A load test or schedule that would breach the existing ProjectLimits or leave the allowed load zones is refused with its own message, proven against the real API server
- [ ] #3 A scheduled test cannot outlive its project, and the deletion path is proven, not asserted
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
