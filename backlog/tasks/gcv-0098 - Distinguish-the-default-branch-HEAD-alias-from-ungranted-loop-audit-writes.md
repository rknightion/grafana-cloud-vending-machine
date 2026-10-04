---
id: GCV-0098
title: Distinguish the default-branch HEAD alias from ungranted loop audit writes
status: Done
assignee: []
created_date: '2026-10-01 00:29'
updated_date: '2026-10-04 12:17'
labels:
  - needs-triage
dependencies: []
priority: low
type: bug
ordinal: 98000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Loop 22 closeout reports the remote symbolic HEAD movement as ungranted while the identical refs/heads/main movement is granted. This is distinct from the correctly flagged release-branch non-fast-forward rewrite and foreign automation pull refs. The extra alias warning obscures attribution in otherwise authorized campaigns. The maintained audit implementation and canonical grant procedure live outside this repository; do not edit generated runtime copies here.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A granted default-branch movement does not independently fail solely because its remote HEAD alias moves to the same commit.
- [x] #2 Ungrantable independent branch changes and non-fast-forward changes remain blocked, with complete raw evidence retained.
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Loop 26 preparation (2026-10-04): already fixed in the maintained audit implementation outside this repository, before this task was picked up. Evidence: the loop 25 closeout audit labelled the remote HEAD alias by the ref it points to - GRANTED derived HEAD beside a granted main move in one checkout, UNGRANTED derived HEAD beside a foreign main push in another - and still flagged every non-fast-forward branch move as ungranted, with raw lines retained in that loop's evidence. No change in this repository; nothing to implement here.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Closed without a change here: the audit tool already judges a remote HEAD alias by its target ref and still blocks ungranted and non-fast-forward moves, as loop 25's closeout audit shows. Tracker-only.
<!-- SECTION:FINAL_SUMMARY:END -->
