---
id: GCV-0040
title: 'Reject structurally invalid XRD schemas in the gate, without an API server'
status: Done
assignee: []
created_date: '2026-09-08 19:15'
updated_date: '2026-09-08 21:01'
labels: []
dependencies: []
priority: high
type: bug
ordinal: 40000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The agent-observability XRD ships a collectionRefs array declared x-kubernetes-list-type: set whose items are objects with no x-kubernetes-map-type: atomic. Kubernetes requires set-list entries to be scalars or atomic maps, so the derived CRD is invalid and that XRD cannot install into any cluster. The current gate parses and renders every XRD and still passes, because nothing checks structural list-type constraints. GCV-0034's envtest harness would catch this, but only where an ephemeral control plane can run, and it is the heaviest possible way to find a defect that is decidable from the document alone. A static structural check closes the whole class in the existing gate, with no control-plane dependency, and keeps working on a machine or runner where envtest assets are unavailable. A sweep of the current tree found 16 set-lists, 18 map-lists and exactly one violation, so the check starts green after the pre-pass repair.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 scripts/validate.sh rejects a set-list whose items are objects without x-kubernetes-map-type: atomic, naming the file and the schema path
- [x] #2 scripts/validate.sh rejects a map-list with no x-kubernetes-list-map-keys, and a map-list whose declared keys are not required on its items
- [x] #3 The check walks every document in every tracked XRD file, so a second CompositeResourceDefinition in one file cannot escape it
- [x] #4 The check runs inside just check with no Kubernetes control plane, envtest asset or network access
- [x] #5 A negative control is recorded for each rule: the check fails when the constraint is broken in a scratch copy, and passes again when it is restored
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 5: add the frozen static set-list and map-list structural walk to scripts/validate.sh; prove all three rules with negative controls, including the second XRD document in access-v1beta1.yaml; root integrates and validates hosted execution.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Wave 5 added the recursive structural list-schema walk to scripts/validate.sh. Negative controls rejected an object set without atomic items, a map list without keys, and a map key absent from items.required, each with file and full schema path. A fourth control proved document 2 in access-v1beta1.yaml is visited. Local just check and hosted Validate run 34277495122 passed at 510de1c193c00795d95448a705964707d1aad81f.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Wave 5 added static structural validation for Kubernetes set and map lists across every XRD document and proved all three rules with fail-and-restore controls.
<!-- SECTION:FINAL_SUMMARY:END -->
