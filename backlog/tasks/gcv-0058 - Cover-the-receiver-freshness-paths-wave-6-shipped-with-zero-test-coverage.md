---
id: GCV-0058
title: Cover the receiver-freshness paths wave 6 shipped with zero test coverage
status: To Do
assignee: []
created_date: '2026-09-09 08:10'
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
- [ ] #1 The zero-coverage functions on the receiver and freshness path are either exercised by tests that pin their refusal behaviour, or removed as unreachable with evidence that nothing calls them
- [ ] #2 Each added test fails against the pre-change behaviour for the right reason before it passes, and that is shown rather than claimed
- [ ] #3 The refusal branches are covered, not just the success path: stale, absent and malformed observations each have a named case
- [ ] #4 No test asserts on an implementation detail that a refactor of the same behaviour would break
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
