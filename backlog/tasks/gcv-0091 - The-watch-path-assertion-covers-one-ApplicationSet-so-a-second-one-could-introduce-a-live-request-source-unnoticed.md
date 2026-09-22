---
id: GCV-0091
title: >-
  The watch-path assertion covers one ApplicationSet, so a second one could
  introduce a live-request source unnoticed
status: Done
assignee:
  - '@codex'
created_date: '2026-09-22 15:59'
updated_date: '2026-09-22 19:18'
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
- [x] #1 Every tracked ApplicationSet is discovered rather than the one path being named, so a new object is covered without a validator edit
- [x] #2 A second ApplicationSet introducing any generator other than the known git generator over enabled/* fails the gate by path and object name, proven by a negative control
- [x] #3 The existing widened-glob and second-generator-block refusals still fail, proven by their negative controls
- [x] #4 Whether a non-request ApplicationSet is permitted at all is recorded as a decision rather than left implicit in the check
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 16 lane A extends ApplicationSet discovery and refusal controls in scripts/validate.sh; preserve and quote every pre-existing negative control before root acceptance.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Wave 17 phase-1 security review approved exact validator blob e67f11cdd822678d677900f6329a958f3bc3ef41 after exhaustive replay. It discovers direct YAML/YML/JSON and recursive List ApplicationSets, including nonignored untracked files; refuses alternate generators, mixed files/directories, widened globs and second generator blocks by path/object; and allows multiple independently compliant objects. Owner decision: non-request ApplicationSets have no exception; every represented ApplicationSet must use the single known directory-only git generator over exactly enabled/*. Plain Applications remain GCV-0093. Source commit ac465f9a842089d8588259ee72ff8125ba236405 is contained in completing main SHA 5f672c09a9fc162fcfbe28ad42d342475cb2a798; hosted Validate run 35771159008 succeeded.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Wave 16 lane A expanded discovery to tracked and nonignored-untracked YAML, YML and JSON, including recursively embedded List items, and demonstrated direct, second-generator, second-ApplicationSet, List, JSON and untracked counterexamples. The source remains uncommitted because lane E stopped on the repeated cross-stack AC5 failure before certifying every pre-existing validator negative control. Resume with that exhaustive replay and integrated plus hosted validation; do not accept a green just check alone.

Expanded the watch-path assertion from one named object to every directly represented ApplicationSet and enforced the exact directory-only enabled/* generator shape with adversarial negative-control replay. Completed by source commit ac465f9a842089d8588259ee72ff8125ba236405 within main SHA 5f672c09a9fc162fcfbe28ad42d342475cb2a798; hosted Validate run 35771159008 passed.
<!-- SECTION:FINAL_SUMMARY:END -->
