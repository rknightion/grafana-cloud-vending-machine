---
id: GCV-0094
title: ApplicationSet template source path can activate inert examples
status: To Do
assignee: []
created_date: '2026-09-24 09:10'
labels:
  - needs-triage
  - ci
dependencies: []
type: bug
ordinal: 94000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
During loop 18 GCV-0093 CodeRabbit review, the existing GCV-0091 ApplicationSet control was found to constrain the git generator directories to enabled/* but not spec.template.spec.source.path. The current requests template uses {{.path.path}}, yet a new or edited ApplicationSet with a fixed examples/catalog path can pass the generator check and make an inert catalog example live. This is distinct from GCV-0093, which covers plain Applications only; that loop may not widen its commissioned control. Inspect all ApplicationSet source forms and prove the gap against the real validator before choosing the repair.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 An ApplicationSet with a fixed template source path under examples/ or enabled/ fails the gate by file and object name, including a List-wrapped or JSON representation where applicable
- [ ] #2 The shipped generator and template continue to pass, and existing GCV-0091 negative controls still fail
- [ ] #3 The resulting control covers all ApplicationSet template sources, including multi-source entries or documents why a form is impossible
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
