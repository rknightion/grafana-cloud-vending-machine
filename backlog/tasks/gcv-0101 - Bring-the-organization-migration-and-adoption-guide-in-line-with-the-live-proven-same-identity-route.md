---
id: GCV-0101
title: >-
  Bring the organization migration and adoption guide in line with the
  live-proven same-identity route
status: Done
assignee: []
created_date: '2026-10-04 20:18'
updated_date: '2026-10-04 20:43'
labels:
  - docs
dependencies: []
priority: medium
type: chore
ordinal: 101000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
docs/migration-1.0.md tells an adopter to apply the new XRDs and then re-apply each stack request, and its Existing-slug result section calls adoption a source-derived position that is unproven against live behaviour. GCV-0078 (migrate the pre-organization estate onto the current request schema) has since moved a real estate onto the organization schema on the same external stack identity: the external stack kept deletion protection on, the original managed children were re-attached rather than deleted or recreated, no managed resource or connection Secret was deleted, old-path output documents were retained under unchanged retention policies, and readers were cut over to the organization-segmented path and witnessed. A plain re-apply was not the route that worked. The guide must describe the proven route generically, with no estate, stack, account or repository identity.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The Migration order section describes the organization move as the same-identity route: keep external deletion protection on, attach the existing managed children rather than deleting or recreating them, never delete a managed resource or its connection Secret during the move, retain old-path output documents, and cut readers over to the new path before declaring completion
- [x] #2 The Existing-slug result section records that adoption by external name was observed live on one real stack with the same external identity retained, states which parts remain source-derived, and keeps the advice to validate in a disposable environment first
- [x] #3 The guide names the witness an adopter must collect before declaring the move complete: current-generation Synced and Ready on the request and dependents, fresh publication status at the new path, an accessible reader census with no old-path reader, and an empty in-stack orphan census
- [x] #4 No estate, stack, account, organisation identifier, host or repository name appears in the change, and just check passes
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Update the migration guide from the generic live witness, preserving the distinction between observed and source-derived behaviour; validate offline.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
loop29: one implementation attempt; accepted and landed, independent review had no findings.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Migration guide now records the scoped live-observed same-identity route, non-destructive child attachment, old-path retention, reader cutover and completion witness, separately from source-derived branches. Completing SHA d77ec23e2d181ee07f33e0edd94c50b41452f830; hosted Validate public reference run 37232538508 succeeded. Isolated composed offline just check passed with Validation passed; independent exact-range review found no defects. CodeRabbit skipped for docs only.
<!-- SECTION:FINAL_SUMMARY:END -->
