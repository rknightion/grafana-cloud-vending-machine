---
id: GCV-0091
title: >-
  The watch-path assertion covers one ApplicationSet, so a second one could
  introduce a live-request source unnoticed
status: To Do
assignee: []
created_date: '2026-09-22 15:59'
labels:
  - needs-triage
  - ci
dependencies: []
type: bug
ordinal: 91000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
scripts/validate.sh asserts the requests ApplicationSet generator's directories equals exactly [{path: enabled/*}]. That catches a widened glob and catches a second generator block added to that same object. It does not catch a wholly separate ApplicationSet, in deploy/ or anywhere else tracked, introducing a second input source for live requests.

Found while settling the external-project-CR trigger boundary for GCV-0079 and GCV-0080. One of the rejected routes was an Argo Plugin generator, and a separate ApplicationSet is exactly the shape it would have taken, so the control that was supposed to refuse it would have stayed green. The boundary is now an owner decision rather than a control, which is the weaker of the two.

The standing constraint this protects is that enabled/ starts empty and inert and examples/ is never watched, so widening what produces live requests is how an inert example becomes a live request. A control that names one object rather than the class leaves that open.

Scope is the assertion, not a new mechanism: discover every ApplicationSet in the tracked tree and require each one's generators to be the single known git generator over enabled/*, failing by path and object name on anything else. An ApplicationSet that legitimately generates something other than requests would need an explicit allowance, and whether any such object should be permitted at all is part of the repair.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Every tracked ApplicationSet is discovered rather than the one path being named, so a new object is covered without a validator edit
- [ ] #2 A second ApplicationSet introducing any generator other than the known git generator over enabled/* fails the gate by path and object name, proven by a negative control
- [ ] #3 The existing widened-glob and second-generator-block refusals still fail, proven by their negative controls
- [ ] #4 Whether a non-request ApplicationSet is permitted at all is recorded as a decision rather than left implicit in the check
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
