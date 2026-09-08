---
id: GCV-0053
title: 'Vend Grafana ML jobs, outlier detectors and holidays'
status: To Do
assignee: []
created_date: '2026-09-08 22:36'
labels: []
dependencies: []
type: feature
ordinal: 53000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The ml group is eight CRDs in the pinned provider - jobs, outlierdetectors, alerts and holidays - with nothing vended. It is the lowest-priority gap on the survey and is tracked so the inventory is complete rather than because it blocks anything.

The reason it is worth vending at all rather than leaving to users is cost: forecasting jobs and outlier detectors run continuously, so they are a recurring spend the platform should be able to cap the same way it caps synthetic checks and k6 load zones.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 ML jobs, outlier detectors and holidays are vended per stack from platform-controlled input, with an explicit external name on every deterministic child
- [ ] #2 A platform-owned cap bounds how many continuously running ML resources a stack may have, and an over-budget request is refused with its own message, proven against the real API server
- [ ] #3 Every emitted kind appears in the provider activation map and the XRD/renderer registry, and the gate fails by path if one is missing
- [ ] #4 A catalog example renders inert and is covered by the catalog README and Kustomization checks
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
