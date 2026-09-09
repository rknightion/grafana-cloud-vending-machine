---
id: GCV-0042
title: Reconcile held-candidate documentation after the admission gate lands
status: Done
assignee: []
created_date: '2026-09-08 20:11'
updated_date: '2026-09-09 00:53'
labels:
  - needs-triage
dependencies:
  - GCV-0034
  - GCV-0035
  - GCV-0041
type: docs
ordinal: 42000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The current public documentation says the 1.0 candidate is held pending two unresolved admission defects and that explicit SCIM null is admitted. Once the commissioned admission repairs and hosted gate complete, those statements become historical rather than current. This wave does not carry authority to redefine the release hold or broaden its four-task queue, so the documentation correction needs a separate owner-reviewed task that preserves the still-active no-release decision.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Every current-status statement about the collectionRefs and explicit SCIM null admission defects matches the proven final API-server behavior
- [x] #2 The standing 1.0.0 hold remains explicit without claiming that either repaired defect is still unresolved
- [x] #3 The migration guide keeps the before-state and migration instructions while clearly distinguishing historical defects from current behavior
- [x] #4 Documentation links and the public documentation render validate successfully
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
Reconciled current admission and SCIM-null behavior, migration history and the unchanged release hold. Documentation rendered successfully; 156 relative links checked, zero missing. SCIM post-disable behavior remains an owner statement; provider ScimConfig exists but remains out of scope. Completing source/pin SHA: 187b03ea40ee32fcea890e40c138f00a8c73bd5f. Hosted Validate run 34296536930: success. Local just check passed with 23 real API-server tests, zero skips.
<!-- SECTION:FINAL_SUMMARY:END -->
