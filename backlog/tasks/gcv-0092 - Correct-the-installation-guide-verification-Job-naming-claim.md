---
id: GCV-0092
title: Correct the installation guide verification Job naming claim
status: Done
assignee: []
created_date: '2026-09-22 17:20'
updated_date: '2026-09-22 19:18'
labels:
  - needs-triage
dependencies: []
type: docs
ordinal: 92000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The installation guide says the verification Job name contains the digest prefix. The shipped manifest instead uses a stable PreSync hook name with BeforeHookCreation and changes only the two immutable digest references. The stale statement can misdirect operators during a pin roll and should be corrected without changing the manifest.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The installation guide describes the stable verification Job name, BeforeHookCreation behavior and the two digest locations accurately
- [x] #2 Documentation validation rejects a future contradiction between this statement and the manifest where practical
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Wave 17 corrected the guide to name the stable verification Job, PreSync plus BeforeHookCreation behavior and the two package-digest locations. Phase-2 security review first rejected three bypasses, then approved corrected validator blob e67f11cdd822678d677900f6329a958f3bc3ef41 and documentation blob 3d58e6068848c5a9a7f87a36e2ae1f712be281a8 after all 15 cases matched: 11 negatives rejected and four positives passed. Source commit ac465f9a842089d8588259ee72ff8125ba236405 is contained in completing main SHA 5f672c09a9fc162fcfbe28ad42d342475cb2a798; hosted Validate run 35771159008 succeeded.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Corrected the verification-Job documentation and added fail-closed validation for stable naming, hook recreation, exact package-reference placement and statement-to-manifest binding. Completed by source commit ac465f9a842089d8588259ee72ff8125ba236405 within main SHA 5f672c09a9fc162fcfbe28ad42d342475cb2a798; hosted Validate run 35771159008 passed.
<!-- SECTION:FINAL_SUMMARY:END -->
