---
id: GCV-0078
title: >-
  Migrate the pre-organization estate onto the current request schema and move
  its output identity with it
status: To Do
assignee: []
created_date: '2026-09-16 17:10'
labels: []
dependencies: []
documentation:
  - docs/migration-1.0.md
type: feature
ordinal: 78000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
One estate still runs a platform revision that predates the multi-organization seam. Its stack request carries no `spec.organization`, and the current schema makes that field required and immutable. The estate cannot take any newer platform revision until this is resolved, so it is frozen on an old function, an old provider build and a four-kind API surface while the other estate moves on.

Two facts are already recorded in docs/migration-1.0.md and are the reason this is a migration rather than a pin bump. `spec.organization` is rejected while absent, so the request is refused the moment the newer schema lands. And adding it moves the output identity from `{prefix}/{region}/{usage}/{slug}` to `{prefix}/{organization}/{usage}/{slug}`, so every consumer reading the old remote path breaks unless it is moved in the same cutover.

The load-bearing unknown is whether an already-stored request can gain the field at all. The transition rule compares `self.spec.organization` to `oldSelf.spec.organization`, and the stored object has no such field. An optional field becoming required underneath a transition rule is exactly the case that errors rather than passing, so the guide's instruction to add the key may not be executable against a live object. Settle that against a real API server with this repository's admission fixture before planning anything else, because the answer decides whether this is an edit or a replacement.

Two more constraints that are easy to miss. The estate's `usage` value is outside the shipped allowed set, it is immutable, and it forms part of the output identity, so the overlay patch that widens the allowed set has to survive the migration or the request is refused on a different rule. And the estate has no written request anywhere: its delivery patches the repository's own catalogue example at sync time through a generator, so every schema change has to be expressed as a patch and kept in step with the example it rewrites.

The version gap will be larger by the time this is picked up, and the schema may have moved again. Re-derive the delta against whatever the platform is at pickup. Do not trust a delta recorded before the work starts.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Whether an already-stored stack request can gain spec.organization in place is settled against a real API server using this repository's admission fixture, with the evidence recorded
- [ ] #2 If it cannot be migrated in place, the replacement path is recorded, stating what happens to the vended stack, its service account and its tokens, and whether the external stack survives the replacement
- [ ] #3 The output-identity move names every consumer of the old remote path and states the cutover order, including when the old path stops being read
- [ ] #4 The estate's non-standard usage value is preserved, or the migration records why it can change given usage is immutable and forms part of the output identity
- [ ] #5 The schema delta is re-derived against the platform revision current at pickup rather than any delta recorded in this task
- [ ] #6 The estate reaches the current request schema with its composite Synced and Ready, no in-stack resource orphaned by the move, and its delivery patches applying cleanly against the example they rewrite
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
