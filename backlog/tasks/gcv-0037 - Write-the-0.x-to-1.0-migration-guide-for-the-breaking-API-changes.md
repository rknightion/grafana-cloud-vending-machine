---
id: GCV-0037
title: Write the 0.x to 1.0 migration guide for the breaking API changes
status: To Do
assignee: []
created_date: '2026-09-08 17:02'
labels: []
dependencies: []
ordinal: 37000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Two breaking changes are queued for the first release: multi-organization vending, and the wave-3 governed vending surfaces with bounded token lifetimes. Release-please folds both into 1.0.0 in PR 29, which the owner is holding open. An adopter running an 0.x checkout has no statement anywhere of what changed in the request schema, which fields became required, which became creation-time-only or immutable, and what happens to an existing request that no longer validates. The breaking-change footers name the commits but not the migration, and the changelog is a commit list rather than an upgrade path. This has to exist before the release, not after it, because an adopter who applies 1.0.0 to an existing request finds out by rejection.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Every breaking schema change between 0.1.0 and the 1.0.0 candidate is enumerated with its before and after shape
- [ ] #2 Each entry states what an adopter must do, including whether an existing request is rejected, silently changed, or unaffected
- [ ] #3 Newly required, newly immutable and newly creation-time-only fields are called out separately from added optional fields
- [ ] #4 The guide is published in the docs site and reachable from the release notes, not only present as a file
- [ ] #5 Every example given is inert and carries no source-environment identifier, verified by the publication scan
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
