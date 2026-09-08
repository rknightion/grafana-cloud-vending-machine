---
id: GCV-0042
title: Reconcile held-candidate documentation after the admission gate lands
status: To Do
assignee: []
created_date: '2026-09-08 20:11'
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
- [ ] #1 Every current-status statement about the collectionRefs and explicit SCIM null admission defects matches the proven final API-server behavior
- [ ] #2 The standing 1.0.0 hold remains explicit without claiming that either repaired defect is still unresolved
- [ ] #3 The migration guide keeps the before-state and migration instructions while clearly distinguishing historical defects from current behavior
- [ ] #4 Documentation links and the public documentation render validate successfully
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
