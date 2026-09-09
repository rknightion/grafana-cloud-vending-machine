---
id: GCV-0059
title: Run the adversarial review pass the wave 6 two-pass ceiling deferred
status: In Progress
assignee:
  - '@codex-wave7'
created_date: '2026-09-09 08:10'
updated_date: '2026-09-09 10:47'
labels: []
dependencies: []
ordinal: 59000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Wave 6 shipped nine new public XRDs and took the machine from 54 to 80 emitted provider kinds in a single run. It ran two CodeRabbit integration passes and stopped there, correctly respecting its own ceiling: the first found one critical, three major and five minor, the second found three minor. Both rounds were finding real defects when the ceiling stopped them, and the last one was found by the report audit rather than by review.

A consolidation wave is where the deferred pass belongs, against integrated code rather than lane output, and before the 1.0.0 boundary freezes these APIs under a compatibility promise. The highest-value targets are the ones the root itself graded HIGH: cross-resource identity and trust (`surfacecontext.go`), credential client routing for Cloud, PDC and Faro, PDC token renewal and connection-Secret ownership, the whole-set ownership singletons, and the admission policies whose paramRef binds a named Composition.

This is a review task, not a rewrite task. Findings become fixes or recorded decisions; a finding left unfixed states why.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The nine wave 6 surfaces are reviewed as integrated code, with the review scope and what was excluded both stated
- [ ] #2 Every finding is read and dispositioned: fixed, or left with a stated reason; no severity band is dismissed unread
- [ ] #3 Any finding that changes a public schema is treated as a compatibility question and recorded before 1.0.0, not applied silently
- [ ] #4 Fixes carry a control that failed before and passes after, and the full gate passes on the completing commit
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Review all nine wave 6 surfaces as integrated code, disposition every issue, route schema-affecting questions to root, and require a failing control before any fix plus a fresh review after corrections.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Lane F two-pass adversarial review covered the complete platform surface, including all nine wave 6 XRDs and 37 function files; GCV-0060 asserts files and all section 5.7 exclusions were outside scope.

Pass 1: four major and one minor findings. Fixed all valid findings: cloud integration jobs now fail closed when an observed account ID disappears; k6 cap transitions reconstruct only platform-owned fields, restore dynamic management policies and current ProviderConfig, and preserve only the observed external name; service-account dependants fail closed when an observed account ID disappears; optional Synthetic Monitoring alert fields are omitted instead of rendered as empty strings. Rejected the minor request to retain existing CheckAlerts when alerts are omitted because the public contract says only an explicit nonempty alerts list owns CheckAlerts.

Pass 2: six findings. Fixed receiver-status unpublished handling so alerting waits without error; fixed Frontend Observability profile transitions that would orphan an observed Faro app; corrected the public k6 schema to enforce the already-published one-schedule-per-load-test promise. The k6 correction was escalated to root under goal section 6.5 and accepted before 1.0.0: before correction, two differently named schedules targeting one load test were admitted; after correction the API server refused the request with 'spec: Invalid value: each load test may have at most one schedule'. Root materiality: HIGH because this closes a public compatibility mismatch before release.

Rejected the ML optional-list guard because envtest admitted a Composition profile with jobs and omitted outlierDetectors before any source change. Rejected the Synthetic Monitoring empty-alert suggestion because minItems: 1, renderer behavior and the public README all define omission as no CheckAlerts; the finding also contained unrelated text. Left alerting-routing negative-control polling unchanged because no failure signature was observed and the test-only hardening had no required failing control.

All accepted repairs carry behavioral controls that failed against the prior behavior and pass after restoration. The integrated gate passed after pass 2: Validation passed, function coverage 84.4%, zero skips.
<!-- SECTION:NOTES:END -->
