---
id: GCV-0049
title: Complete k6 vending with load tests and schedules
status: In Progress
assignee:
  - '@codex-wave7'
created_date: '2026-09-08 22:36'
updated_date: '2026-09-09 09:41'
labels: []
dependencies: []
type: feature
ordinal: 49000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
GCV-0021 vended the k6 Project, ProjectLimits and ProjectAllowedLoadZones as a blast-radius cap and deliberately stopped there: it provisions no tests and no schedules, and that exclusion is recorded in its acceptance evidence rather than being an oversight. The cap is now in place, so the surface it was capping can be vended.

The provider carries k6 loadtests, loadtestsets, schedules and schedulesets at v2.14.0, none of them emitted. The point of doing this after the cap rather than before is that every test and schedule must land inside the existing platform limits, so the interesting work is proving the cap actually binds them rather than adding the resources.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Load tests and schedules are vended per project from platform-controlled input, with an explicit external name on every deterministic child
- [ ] #2 Admission refuses declared load-test VUs, browser VUs, duration, or load zones outside the Composition profile with its own message, proven against the real API server; reconciliation refuses usage that differs from the referenced stack's observed usage and withholds dynamic children until current observed ProjectLimits and allowed load zones match desired
- [ ] #3 A scheduled test cannot outlive its project, and the deletion path is proven, not asserted
- [x] #4 Every emitted kind appears in the provider activation map and the XRD/renderer registry, and the gate fails by path if one is missing
- [x] #5 A catalog example renders inert and is covered by the catalog README and Kustomization checks
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 6: implement the commissioned surface under the frozen goal and root-owned integration; prove admission and renderer boundaries with required negative controls, then just check and exact-SHA hosted Validate before finalization.

Wave 7 settlement: amend only the criterion whose cross-object admission mechanism does not exist, retain all caps, prove deletion behavior at the machine-controlled render boundary, then close with integrated local and exact-SHA hosted evidence.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Final root integration narrows new workloads to generated structured HTTPS GET scripts with explicit constant-vus scenarios and zero graceful-stop time; arbitrary JavaScript and browser workloads are excluded. API admission proves declared-profile limits; trusted referenced-stack usage remains a reconcile-time check, so AC2 is not fully satisfied. Dynamic child Delete policies are emitted, but remote project/schedule deletion is unproven. Cap transitions now preserve prior configured workloads while publishing revised caps, keeping the composite not Ready until the current Synced generation is observed. GCV-0049 remains incomplete pending exact cross-object admission and deletion acceptance.

Wave 7 acceptance amendment under the owner settlement. Before AC2: A load test or schedule that would breach the existing ProjectLimits or leave the allowed load zones is refused with its own message, proven against the real API server. After AC2: Admission refuses declared load-test VUs, browser VUs, duration, or load zones outside the Composition profile with its own message, proven against the real API server; reconciliation refuses usage that differs from the referenced stack's observed usage and withholds dynamic children until current observed ProjectLimits and allowed load zones match desired. Source evidence: the binding has one Composition paramRef, API-server tests prove declared-profile refusals, and renderer tests prove observed-usage mismatch plus current-cap withholding. The original cross-object admission mechanism does not exist; existing caps are unchanged. Root materiality: HIGH because this corrects a public acceptance claim before 1.0.0.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
PARTIAL: structured HTTPS GET load tests and schedules land with observed identities, explicit constant-vus and gracefulStop 0s. Declared-profile admission and cap weaken/admit/restore passed. Current cap observation gates new loads while prior requested workload specs are preserved so revised caps can converge. AC2 remains incomplete because actual referenced-stack usage is checked only at reconcile; AC3 remote schedule/project lifetime and deletion behavior is unproven. Resume: commission cross-object admission or generated policy lifecycle without changing caps, then prove the remote deletion boundary. Completing source/pin SHA: 187b03ea40ee32fcea890e40c138f00a8c73bd5f. Hosted Validate run 34296536930: success. Local just check passed with 23 real API-server tests, zero skips.
<!-- SECTION:FINAL_SUMMARY:END -->
