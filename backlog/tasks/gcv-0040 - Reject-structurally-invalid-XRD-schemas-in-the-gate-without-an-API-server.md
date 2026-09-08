---
id: GCV-0040
title: 'Reject structurally invalid XRD schemas in the gate, without an API server'
status: In Progress
assignee: []
created_date: '2026-09-08 19:15'
updated_date: '2026-09-08 20:06'
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
- [ ] #1 scripts/validate.sh rejects a set-list whose items are objects without x-kubernetes-map-type: atomic, naming the file and the schema path
- [ ] #2 scripts/validate.sh rejects a map-list with no x-kubernetes-list-map-keys, and a map-list whose declared keys are not required on its items
- [ ] #3 The check walks every document in every tracked XRD file, so a second CompositeResourceDefinition in one file cannot escape it
- [ ] #4 The check runs inside just check with no Kubernetes control plane, envtest asset or network access
- [ ] #5 A negative control is recorded for each rule: the check fails when the constraint is broken in a scratch copy, and passes again when it is restored
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 5: add the frozen static set-list and map-list structural walk to scripts/validate.sh; prove all three rules with negative controls, including the second XRD document in access-v1beta1.yaml; root integrates and validates hosted execution.
<!-- SECTION:PLAN:END -->
