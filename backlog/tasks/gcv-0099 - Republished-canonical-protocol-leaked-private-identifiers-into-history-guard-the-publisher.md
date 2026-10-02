---
id: GCV-0099
title: >-
  Republished canonical protocol leaked private identifiers into history; guard
  the publisher
status: In Progress
assignee: []
created_date: '2026-10-02 16:48'
updated_date: '2026-10-02 17:56'
labels: []
dependencies: []
priority: high
type: bug
ordinal: 99000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Canonical publication introduced refused identifier classes into public history. Source rewording, a public-consumer publisher guard and exact history-only allowances are required without widening working-tree exemptions.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Canonical source is reworded and public-consumer guard rejects planted refused terms before any push
- [ ] #2 Canonical protocol is republished and exact history-only allowances retain negative working-tree controls
- [ ] #3 Integrated local gate and hosted validation pass with completing SHA and run ID
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Source reword and fail-closed public-consumer guard; root republication; independently reviewed exact history-only scan allowances; integrated local gate and exact main hosted Validate before finalization.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Loop23 source guard landed at 2e668bdcf256bf15d382b8c5e914d1e9dc55ad06, local gate passed with 86 tests and public guard, completed code review had zero unresolved findings. Expected STALE-only drift run 37040932935 had current=7 stale=44 missing=0 unreadable=0. Root publication completed for 44 consumers and remote readbacks contain all pushed commits; this checkout received cc9d3f318b6d55d3c77d7b3bc19967e9814182ec. Exact history-only scan candidate and independent review remain before green main.
<!-- SECTION:NOTES:END -->
