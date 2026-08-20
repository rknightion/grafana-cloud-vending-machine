---
id: GCV-0009
title: Reconsider typed request structs as the API grows
status: To Do
assignee: []
created_date: '2026-08-20 16:10'
labels:
  - function
  - tech-debt
dependencies: []
ordinal: 9000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The function reads composites as map[string]any and discards the type assertion at every level, in the style metadata, _ := xr['metadata'].(map[string]any). A field of the wrong type becomes a zero value rather than an error.

This is not currently a defect. The XRD validates every field the function reads, so a malformed document does not reach it, and the explicit checks catch empty required values. The risk is that it fails silently rather than loudly if the XRD and the function ever disagree -- which is exactly what a growing API makes likelier, as new optional and nested fields arrive faster than the validation that covers them.

Not worth doing on the current surface: it would touch every render path for no behavioural change. Revisit when the API next grows a nested optional structure, and if the answer is still no, record that too so it stops being reopened.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A decision is recorded either way, with the trigger that would change it
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
