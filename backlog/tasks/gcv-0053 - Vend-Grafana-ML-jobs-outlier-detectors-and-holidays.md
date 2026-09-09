---
id: GCV-0053
title: 'Vend Grafana ML jobs, outlier detectors and holidays'
status: Done
assignee: []
created_date: '2026-09-08 22:36'
updated_date: '2026-09-09 00:53'
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
- [x] #1 ML jobs, outlier detectors and holidays are vended per stack from platform-controlled input, with an explicit external name on every deterministic child
- [x] #2 A platform-owned cap bounds how many continuously running ML resources a stack may have, and an over-budget request is refused with its own message, proven against the real API server
- [x] #3 Every emitted kind appears in the provider activation map and the XRD/renderer registry, and the gate fails by path if one is missing
- [x] #4 A catalog example renders inert and is covered by the catalog README and Kustomization checks
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
Vended ML jobs, outlier detectors and holidays with platform profile caps and real API-server weaken/admit/restore. Singleton stack ownership prevents multiple profile owners; duplicate names are rejected before dependency waits, and withdrawal guards avoid orphaning retained running resources. Complete provider schemas and inert catalog checks passed; no live recurring spend census is claimed. Completing source/pin SHA: 187b03ea40ee32fcea890e40c138f00a8c73bd5f. Hosted Validate run 34296536930: success. Local just check passed with 23 real API-server tests, zero skips.
<!-- SECTION:FINAL_SUMMARY:END -->
