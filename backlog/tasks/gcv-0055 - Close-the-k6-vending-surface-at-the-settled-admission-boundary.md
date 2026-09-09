---
id: GCV-0055
title: Close the k6 vending surface at the settled admission boundary
status: Done
assignee:
  - '@codex-wave7'
created_date: '2026-09-09 08:09'
updated_date: '2026-09-09 10:53'
labels: []
dependencies: []
ordinal: 55000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
GCV-0049 is Parked with AC2 and AC3 unmet. Both were written expecting admission to read the referenced stack@s actual usage, which one ValidatingAdmissionPolicy cannot do: a policy takes a single paramRef and cannot also read a second object. Wave 6 established that closing the gap needs either a deployed validating webhook with TLS, RBAC and cert rotation, or a generated per-stack policy lifecycle owned by the function.

The repository owner settled this on 2026-09-09: neither route is taken. Crossplane already refuses at reconcile, the catalog README already tells adopters the budget is not a tenant quota, and a webhook would duplicate the function@s logic in a second place that can drift. The acceptance criteria are rewritten to the boundary that actually holds, and the admission gap becomes a recorded design decision rather than carried debt.

This task closes GCV-0049. It does not weaken any existing cap.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 GCV-0049 acceptance criteria state the enforcement boundary that actually holds: declared profile limits at admission, referenced-stack usage at reconciliation
- [x] #2 A Backlog decision records why cross-object admission is not pursued, naming both rejected routes and their costs, so it is not re-litigated
- [x] #3 The scheduled-test and project deletion path is proven at the boundary the machine actually controls, not asserted
- [x] #4 Public documentation for the k6 surface states which guarantees are admission-time and which are reconcile-time, with no claim the machine does not meet
- [x] #5 GCV-0049 reaches Done with its remaining evidence, or the reason it cannot is a new fact rather than the admission gap this task settles
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Apply the frozen cross-object admission settlement, record the decision, prove scheduled-test/project deletion semantics at the render boundary, reconcile public k6 documentation, and close GCV-0049 without weakening caps.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Recorded accepted decision decision-0001 for the settled admission/reconciliation boundary. Extended TestK6ProjectDynamicChildrenAreDeleteManaged to prove LoadTest and Schedule carry Delete, both disappear from desired state when withdrawn, and the retained Project remains. Removing Delete made the focused test fail for both children; restored focused test and just check passed at 84.3% with zero skips. CodeRabbit reviewed platform/function/k6_test.go and raised 0 issues. Root scope amendment: corrected docs/governance.md because it claimed scripts pass through unchanged, while generatedK6Script emits a structured HTTPS GET workload; leaving that sentence would preserve a false public claim.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Closed GCV-0049 at the owner-settled admission and reconciliation boundary without weakening caps. decision-0001 records the rejected webhook and generated-policy routes; machine-controlled deletion semantics are proven. Completing SHA ed4f3fd93103d8fc84261b1e578699eed1e3cece; hosted Validate public reference run 34336742523 succeeded; local just check passed at 84.3% with zero skips.
<!-- SECTION:FINAL_SUMMARY:END -->
