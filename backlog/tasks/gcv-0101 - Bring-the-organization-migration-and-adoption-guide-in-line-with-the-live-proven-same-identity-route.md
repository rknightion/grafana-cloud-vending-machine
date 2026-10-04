---
id: GCV-0101
title: >-
  Bring the organization migration and adoption guide in line with the
  live-proven same-identity route
status: To Do
assignee: []
created_date: '2026-10-04 20:18'
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
- [ ] #1 The Migration order section describes the organization move as the same-identity route: keep external deletion protection on, attach the existing managed children rather than deleting or recreating them, never delete a managed resource or its connection Secret during the move, retain old-path output documents, and cut readers over to the new path before declaring completion
- [ ] #2 The Existing-slug result section records that adoption by external name was observed live on one real stack with the same external identity retained, states which parts remain source-derived, and keeps the advice to validate in a disposable environment first
- [ ] #3 The guide names the witness an adopter must collect before declaring the move complete: current-generation Synced and Ready on the request and dependents, fresh publication status at the new path, an accessible reader census with no old-path reader, and an empty in-stack orphan census
- [ ] #4 No estate, stack, account, organisation identifier, host or repository name appears in the change, and just check passes
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
