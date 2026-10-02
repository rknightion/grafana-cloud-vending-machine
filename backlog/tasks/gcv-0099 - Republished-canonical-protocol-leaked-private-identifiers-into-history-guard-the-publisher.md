---
id: GCV-0099
title: >-
  Republished canonical protocol leaked private identifiers into history; guard
  the publisher
status: To Do
assignee: []
created_date: '2026-10-02 16:48'
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
