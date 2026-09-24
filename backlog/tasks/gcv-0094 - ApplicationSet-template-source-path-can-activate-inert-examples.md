---
id: GCV-0094
title: ApplicationSet template source path can activate inert examples
status: Done
assignee:
  - '@codex'
created_date: '2026-09-24 09:10'
updated_date: '2026-09-24 13:01'
labels:
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
- [x] #1 An ApplicationSet with a fixed template source path under examples/ or enabled/ fails the gate by file and object name, including a List-wrapped or JSON representation where applicable
- [x] #2 The shipped generator and template continue to pass, and existing GCV-0091 negative controls still fail
- [x] #3 The resulting control covers all ApplicationSet template sources, including multi-source entries or documents why a form is impossible
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Prove the pre-change fixed-path escape against the real validator, add the exact ApplicationSet template allow-list in the existing walker, replay negative and positive controls, obtain independent blob review, then land with exact-SHA local and hosted validation.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Loop 19 selected 2026-09-24: exact ApplicationSet template source-path allow-list. Lane A implements and AR independently reviews its blob. Frozen allowed form: goTemplate true; one template source mapping with path exactly {{.path.path}}; no sources or templatePatch. Generator/template repoURL and targetRevision equality is outside this task.

Loop 19 owner release decision, 2026-09-24: leave the release-please pull request opened by the fix push for the owner. This run makes no pull-request write; report its number, head and checks at close.

Loop 19 landed validator commit 1687bc711c568089365ac5732dcb71076c3b1800. Prechange fixed examples/catalog/minimal path passed; corrected control rejects examples/, enabled/, List, JSON, untracked, multi-source, mixed fields, templatePatch, false or absent goTemplate, whitespace variants and chart even with the allowed path. AR approved exact blob e513adef77c1317026a5423d647e871ba20f6256; review SHA-256 1dc425f1f5a53dcc45619e8e8f7b984bf2bbe7029aa8984c48cce22b0dfabfad. Exact-SHA detached just check passed with 85.5% coverage and Validation passed. Hosted Validate public reference run 36002036869 succeeded on the completing SHA. CodeRabbit pass 1 caught a chart-with-path gap; pass 2 reviewed scripts/validate.sh with zero findings.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Closed the ApplicationSet template source-path gap with an exact directory-path allow-list. Completing SHA 1687bc711c568089365ac5732dcb71076c3b1800; detached local just check passed at 85.5% coverage; hosted Validate public reference run 36002036869 succeeded on that SHA. Independent AR approved the corrected blob, and CodeRabbit pass 2 completed with zero findings.
<!-- SECTION:FINAL_SUMMARY:END -->
