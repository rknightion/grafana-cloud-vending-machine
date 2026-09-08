---
id: GCV-0038
title: Split the 75 KB README so it stops being a single-owner scheduling bottleneck
status: In Progress
assignee: []
created_date: '2026-09-08 17:02'
updated_date: '2026-09-08 17:20'
labels: []
dependencies: []
ordinal: 38000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The wave operating model records the README as a deliberate exception: because it is one large file it has exactly one owner per wave, and where several lanes each need a section the README owner is scheduled last with declared dependencies on all of them. That is a real cost paid every wave, and it grows as surfaces are added. The operating model still describes it as 54 KB; it is now 76534 bytes, which is the growth curve stated as a fact. A documentation site already exists under docs/ with its own navigation, so the content has somewhere to go. The point of splitting is not tidiness, it is removing the serialization: sections that live in separate files can be owned by the lanes that produced them. The README that remains should be what a reader needs in the first minute, and a map to the rest.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 README content is redistributed into docs pages by subject, with no content silently dropped in the move
- [ ] #2 The remaining README orients a new reader and links onward, and is small enough that two lanes editing different subjects do not collide
- [ ] #3 Every moved section is reachable from the docs site navigation, verified by rendering it rather than by reading the config
- [ ] #4 Links that pointed into the README from docs, examples and task history either still resolve or are updated
- [ ] #5 The Wave operating model document is updated to retire the single-owner README exception, since the reason for it is gone
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 4: implement the commissioned lane after the pushed root harness pre-pass; preserve frozen schemas and ownership; return acceptance evidence and required negative controls; root integrates, reviews, validates locally and at the exact hosted SHA, then reconciles status.
<!-- SECTION:PLAN:END -->
