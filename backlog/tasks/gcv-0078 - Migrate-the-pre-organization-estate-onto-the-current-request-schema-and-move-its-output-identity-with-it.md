---
id: GCV-0078
title: >-
  Migrate the pre-organization estate onto the current request schema and move
  its output identity with it
status: Parked
assignee: []
created_date: '2026-09-16 17:10'
updated_date: '2026-09-24 08:37'
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
- [x] #1 Whether an already-stored stack request can gain spec.organization in place is settled against a real API server using this repository's admission fixture, with the evidence recorded
- [x] #2 If it cannot be migrated in place, the replacement path is recorded, stating what happens to the vended stack, its service account and its tokens, and whether the external stack survives the replacement
- [x] #3 The output-identity move names every consumer of the old remote path and states the cutover order, including when the old path stops being read
- [x] #4 The estate's non-standard usage value is preserved, or the migration records why it can change given usage is immutable and forms part of the output identity
- [x] #5 The schema delta is re-derived against the platform revision current at pickup rather than any delta recorded in this task
- [ ] #6 The estate reaches the current request schema with its composite Synced and Ready, no in-stack resource orphaned by the move, and its delivery patches applying cleanly against the example they rewrite
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 14: settle in-place organization migration on the pinned ephemeral API server and produce the AC1-AC5 cutover plan; keep AC6 open for the human-operated live cutover.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
2026-09-18 deferral reasoning corrected, and the task split for wave 14.

AC1 IS NOT LIVE WORK, and every prior wave that deferred this task said it was. Wave 13's selection
table recorded GCV-0078 as "needs a live admission answer / an API-server experiment" and deferred
it on that basis. AC1's own wording is "against a real API server using THIS REPOSITORY'S admission
fixture" - that is the pinned ephemeral envtest server the gate already stands up on every `just
check` run, not a live cluster. The load-bearing unknown is answerable here with zero live contact.
Do not re-defer this task on the belief that AC1 needs an estate.

The shape to copy is the create-persist-upgrade-update sequence wave 13 used to defeat Kubernetes
validation ratcheting on the provisioning API; `platform/function/provisioning_test.go`'s CRD
upgrade cases are the worked example. Test BOTH an update that touches no sibling field and one that
changes a sibling, because ratcheting is live in this harness and the migration guide's instruction
is the former while a real cutover is likely the latter.

AC1 THROUGH AC5 ARE COMMISSIONED IN WAVE 14. AC6 IS NOT. The cutover is an authorised action on a
real estate, which the wave forbids, so this task will not reach Done in wave 14 even if its lane
fully succeeds. It is also not Parked if the lane succeeds, because nothing is blocked - the cutover
is simply the next human step.

WHY THE CUTOVER IS AFTER THE WAVE RATHER THAN BEFORE OR DURING IT, so this is not re-litigated:

1. The frozen estate cannot take the current pin at all. Its stored request carries no
   spec.organization and the current schema makes that field required, so the request is refused the
   moment a newer revision lands. This is not a pin bump and no rollout ordering makes it one. It
   also means this estate is irrelevant to GCV-0077's clean-signal problem, which only ever
   concerned the estate that is not frozen.
2. AC5 requires the delta re-derived at pickup, and wave 14 itself moves two XRDs (the stack
   consumer API and the provisioning connection API). A delta derived before that wave is stale by
   the end of it.
3. Cutting over after wave 14 lands the estate on a pin that can rotate, once GCV-0075 is in it.
   Migrating onto the current un-rotatable pin and re-rolling a week later is two cutovers on the
   most fragile estate in the fleet, and the first would hand it credentials that cannot rotate.

A released-API change is a blocker, not a lane edit. If AC1's answer implies relaxing the
spec.organization transition rule or making the field optional again, that is a change to
platform/apis/stack-v1beta1.yaml, which shipped in v1.0.0, v1.0.1 and v2.0.0. Return the exact edit
and let the owner decide; a fail-closed loosening on a released API is a breaking change however
safe it looks.

Wave 14 AC1-AC5 evidence is in codex/wave14/lane-g-plan.md and codex/wave14/lane-g-ac5.md. The pinned API server refused both organization-only and organization-plus-mutable-sibling updates, so replacement is required. Before AC6: wait for GCV-0075 to land safely; freeze promotion; capture the source revision and generator revision; record the stored request, output path, UID, conditions and immutable fields; inventory every composed object, external identity, management policy, credential generation, remote writer and reader; render the replacement against the completing catalogue while preserving usage and both allowed-usage layers. During AC6: verify Retain and non-deleting policies; remove the old request through its delivery mechanism; prove the external stack survives with unchanged identity; apply the replacement and stop on any Create or identity change; wait for the replacement core graph and dependent requests; move every remote-path consumer one at a time and prove reload onto replacement credentials; retire source tokens before policies or service accounts; delete only source-only documents; resume promotion only after the orphan, duplicate and stale-reader census is empty. AC6 remains open because no live estate, cluster, credential, consumer or generator was contacted.

2026-09-24: AC6 is the human-operated replacement cutover in the task notes, after GCV-0075 lands and the estate runs a pin carrying it. No one is presently executing it.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
AC1-AC5 completed against pinning revision 1f2af4a0b37c244a799e1fe0cdfcea2fa5abd82e and hosted validation run 35364595851. AC6 remains open for the human-operated replacement cutover; status remains In Progress.
<!-- SECTION:FINAL_SUMMARY:END -->
