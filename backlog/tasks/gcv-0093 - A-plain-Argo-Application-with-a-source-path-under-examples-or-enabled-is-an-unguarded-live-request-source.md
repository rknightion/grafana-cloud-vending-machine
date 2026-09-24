---
id: GCV-0093
title: >-
  A plain Argo Application with a source path under examples/ or enabled/ is an
  unguarded live-request source
status: Done
assignee:
  - '@codex'
created_date: '2026-09-22 17:44'
updated_date: '2026-09-24 10:25'
labels:
  - needs-triage
  - ci
dependencies:
  - GCV-0091
type: bug
ordinal: 93000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Found in the wave 16 review while checking GCV-0091's candidate. GCV-0091 discovers every ApplicationSet and requires each to be the single git generator over enabled/*. The same class has a second shape the candidate does not cover: an ordinary Argo CD Application (kind Application, not ApplicationSet) whose spec.source.path or spec.sources[].path points at examples/, a catalog directory or enabled/ itself. deploy/argocd/platform.yaml is the existing instance of the kind, with sources at platform and deploy/aws, and nothing asserts which paths an Application may source. A tracked Application sourcing examples/catalog/<dir> would make an inert example a live request with the gate green. Scope is the assertion only, in scripts/validate.sh, reusing GCV-0091's discovery once it lands: enumerate every Application and fail by path and object name when any source path is under examples/ or enabled/, or replace that with an explicit allow-list of permitted source paths. Which of those two is the decision. Depends on GCV-0091 landing, because both edit the same validator block.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Every tracked Application is discovered with the same representation boundary GCV-0091 uses (YAML, YML, JSON, List items)
- [x] #2 An Application sourcing any path under examples/ or enabled/ fails the gate by path and object name, proven by a negative control
- [x] #3 The existing platform Application still passes, and the choice between a deny-list and an allow-list of source paths is recorded
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Loop 18 lane A is implementing the exact Application source-path allow-list; independent AR review and hosted validation are required.

Loop 18 chose exact allow-list platform and deploy/aws. Seven negative controls, chart-only and unchanged positives passed; AR first rejected mixed source/sources bypass, A corrected it and AR approved blob 154b7549141df38760ea3ac82ee6abdaaf2fc8cc. Separate ApplicationSet template gap tracked as GCV-0094. Exact local just check and hosted Validate passed.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added a plain-Application source-path allow-list using the existing YAML/YML/JSON/List discovery. AR approved the corrected mixed-field control. Completing pushed SHA ba1475df2ea802a0bbec678986fba80807d5f603, hosted Validate 35984785494; delivered pin SHA 886887d0315a2a6ae5182732f184cc873a5d1fa2, hosted Validate 35986358148.
<!-- SECTION:FINAL_SUMMARY:END -->
