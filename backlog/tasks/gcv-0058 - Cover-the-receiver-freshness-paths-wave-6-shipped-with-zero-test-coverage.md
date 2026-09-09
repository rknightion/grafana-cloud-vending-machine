---
id: GCV-0058
title: Cover the receiver-freshness paths wave 6 shipped with zero test coverage
status: Done
assignee:
  - '@codex-wave7'
created_date: '2026-09-09 08:10'
updated_date: '2026-09-09 10:53'
labels: []
dependencies: []
ordinal: 58000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Wave 6@s root graded the receiver and freshness guards HIGH on the grounds that a stale Ready flag must never authorize a joined alerting path. Measured against the completing tree at 84.3% overall coverage, two functions on exactly that path execute in no test at all: `alertingjoin.go:90 onCallReceiverStatus` and `observation.go:78 observedDesiredReady`, both 0.0%. `alertingjoin.go:18 resolveAlertingReceivers` is at 31.5%.

The end-to-end alert-to-responder proof passes, so the covered path works. The uncovered branches are the refusal paths - what happens when an observation is stale, absent or malformed - which is where a fail-closed machine either holds or silently opens. A helper at 0.0% is also worth checking for being unreachable rather than merely untested; either finding is useful.

Scope is the wave 6 alerting-join and observation code. Do not retro-fit tests onto surfaces this task did not touch; report a coverage gap elsewhere rather than filling it.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The zero-coverage functions on the receiver and freshness path are either exercised by tests that pin their refusal behaviour, or removed as unreachable with evidence that nothing calls them
- [x] #2 Each added test fails against the pre-change behaviour for the right reason before it passes, and that is shown rather than claimed
- [x] #3 The refusal branches are covered, not just the success path: stale, absent and malformed observations each have a named case
- [x] #4 No test asserts on an implementation detail that a refactor of the same behaviour would break
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Establish red-before controls for stale, absent, and malformed receiver observations; exercise or remove zero-coverage helpers based on call evidence; keep assertions at behavioral boundaries.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Lane E evidence: added named stale, absent, and malformed receiver-observation cases at behavioral boundaries. Each case failed when its matching freshness guard was weakened and passed after restoration. Removed observedDesiredReady after call-site search proved it unreachable. CodeRabbit reviewed the two changed function files with zero findings. Integrated just check passed with 84.3% coverage and zero skips before the lane commit.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Pinned stale, absent, and malformed receiver-observation refusals with named behavioral tests; each failed under its matching weakened guard and passed after restoration. Removed unreachable observedDesiredReady after call-site proof. Completing SHA d86b40713a55136506cac824c1ea7a320c141412; hosted Validate public reference run 34338537668 succeeded; local just check passed at 84.3% with zero skips. Integrated review SHA d1713e78d0969ad58569e9d207a6ecf5c8dbedd3 also passed hosted run 34342127436.
<!-- SECTION:FINAL_SUMMARY:END -->
