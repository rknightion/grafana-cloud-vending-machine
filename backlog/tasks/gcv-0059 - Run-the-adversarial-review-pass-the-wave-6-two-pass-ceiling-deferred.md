---
id: GCV-0059
title: Run the adversarial review pass the wave 6 two-pass ceiling deferred
status: To Do
assignee: []
created_date: '2026-09-09 08:10'
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
