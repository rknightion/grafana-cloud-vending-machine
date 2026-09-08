---
id: GCV-0025
title: Add expiring sandbox stacks with a recorded extension trail
status: Done
assignee: []
created_date: '2026-08-21 12:17'
updated_date: '2026-09-08 16:43'
labels: []
dependencies: []
ordinal: 25000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Grafana Cloud has no concept of an expiring stack, so a time-to-live has to be built. Crossplane suits this well because the reconcile loop already runs continuously.

Add an expiry to the stack request, have the function compare it against the clock, warn through the existing contact point or incident relay ahead of expiry, then drive deletion through the armed-delete path that already exists rather than inventing a second destructive route. Record extensions on the object so they are auditable rather than argued about.

The hard part is already built: the three-stage reviewed decommission path and the Retain-by-default lifecycle with an explicitly authorized Delete. This task adds the clock and the audit trail, and must not weaken either control. An expiry must not become a way to bypass the deletion authorization.

Deliberate scope limit: expiry applies to sandbox usage classes only, selected by platform configuration, so a production stack cannot acquire one.

Note the function currently derives deterministic values without wall-clock input; introducing time as an input affects reproducibility of renders and needs a testable seam rather than a direct clock read.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Expiry is available only to platform-approved sandbox usage classes
- [x] #2 Expiry drives the existing armed-delete path and cannot bypass its authorization
- [x] #3 Time is injected through a testable seam rather than read directly, and render reproducibility is preserved in tests
- [x] #4 Warnings are emitted ahead of expiry through an already-vended delivery path
- [x] #5 Extensions are recorded on the object with who and when
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
Approved sandbox expiry uses one injected clock, append-only declared extension provenance, and the existing incident contact. Effective deadline delays Review 1 arming while preserving exact Delete authorization before and after expiry. Reviews 2 and 3 remain separate reviewed changes. Actual deletion-field boundary and reproducibility tests passed. Authenticated provenance requires audit-log correlation; no live deletion or warning delivery was exercised. Completing delivery SHA bec9551c3c2abb009a4a50412b33efe47b07520c; hosted Validate 34252640140 success. Root just check passed (85.7% coverage). Signed multi-platform function digest sha256:09ff21ddf5436d0f0165ac7849d86ab4c22a6633551d91ab6aab4edc48f88652 is pinned in both locations. No live provider or deployment proof is claimed.
<!-- SECTION:FINAL_SUMMARY:END -->
