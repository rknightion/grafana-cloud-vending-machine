---
id: GCV-0019
title: Vend a golden SLO per usage class
status: Done
assignee: []
created_date: '2026-08-21 12:15'
updated_date: '2026-09-08 16:43'
labels: []
dependencies:
  - GCV-0018
ordinal: 19000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The SLO kind auto-generates recording rules plus fastburn, slowburn and budget-remaining alert rules from one definition, so a single vended object gives a tenant a working error budget on day one instead of an empty alerting page, and gives the platform one comparable reliability metric across every tenant.

Render it in handoff mode so tenants can tune it, using the provenance decision from the alerting bundle task rather than a separate mechanism.

Ordering: the destination datasource must exist first, since the SLO references a Prometheus-compatible datasource uid. Use the observed-resource dependency gates already established rather than a wait.

Query types available are ratio, freeform, threshold and a multi-datasource form. Prefer ratio for a golden template because it is the only one that can be parameterised safely without knowing the workload.

Avoid the alerting enrichment block for now: its assistant investigation type depends on alert enrichment, which is public preview with an explicit breaking-change warning.

Objectives and queries are workload-owned in general; what is vended here is a template per usage class, not an inferred SLO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The SLO is templated per usage class from platform-owned configuration, not inferred from a stack request
- [x] #2 It renders only after its destination datasource is observed, using the existing gating pattern
- [x] #3 It is rendered in handoff mode so a tenant can edit it, consistent with the alerting provenance decision
- [x] #4 No alert enrichment block is rendered while that surface remains in preview
- [x] #5 A catalog example renders with a ratio query and inert metric names
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 ./scripts/validate.sh passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 3: root pushes fail-closed seams; assigned lane implements owned files test-first; root audits ownership, integrates documentation and wiring, reviews and validates, verifies signed package publication, pins both references, then finalizes with exact-SHA hosted validation.
<!-- SECTION:PLAN:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Usage-selected ratio SLO waits for its observed destination datasource, uses handoff initialization and omits enrichment. Golden SLO branch tests and the inert catalog parse/render passed. Completing delivery SHA bec9551c3c2abb009a4a50412b33efe47b07520c; hosted Validate 34252640140 success. Root just check passed (85.7% coverage). Signed multi-platform function digest sha256:09ff21ddf5436d0f0165ac7849d86ab4c22a6633551d91ab6aab4edc48f88652 is pinned in both locations. No live provider or deployment proof is claimed.
<!-- SECTION:FINAL_SUMMARY:END -->
