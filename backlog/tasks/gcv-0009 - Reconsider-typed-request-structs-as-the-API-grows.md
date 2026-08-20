---
id: GCV-0009
title: Reconsider typed request structs as the API grows
status: Done
assignee:
  - '@codex'
created_date: '2026-08-20 16:10'
updated_date: '2026-08-20 17:50'
labels:
  - function
  - tech-debt
dependencies: []
ordinal: 9000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The function reads composites as map[string]any and discards the type assertion at every level, in the style metadata, _ := xr['metadata'].(map[string]any). A field of the wrong type becomes a zero value rather than an error.

This is not currently a defect. The XRD validates every field the function reads, so a malformed document does not reach it, and the explicit checks catch empty required values. The risk is that it fails silently rather than loudly if the XRD and the function ever disagree -- which is exactly what a growing API makes likelier, as new optional and nested fields arrive faster than the validation that covers them.

Not worth doing on the current surface: it would touch every render path for no behavioural change. Revisit when the API next grows a nested optional structure, and if the answer is still no, record that too so it stops being reopened.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A decision is recorded either way, with the trigger that would change it
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 ./scripts/validate.sh passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Reassess unstructured request parsing against the nested API added in this wave.
2. Record the typed-struct decision and the observable trigger that would reverse it.
3. Validate the integrated repository and record exact-SHA hosted proof.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Decision 2026-08-20: do not migrate all request parsing to repository-owned typed structs in this wave. The five render paths and XRD admission already protect existing fields, so a full rewrite has no current behavioral return. Parse the new spec.lifecycle.externalResources safety boundary with explicit type/path errors rather than silent defaults. Reverse this decision when lifecycle requires two or more nested fields that must be validated atomically, or a test reproduces an XRD/function mismatch that silently selects an unsafe lifecycle action or materially wrong resources; then add a single typed conversion boundary and migrate the renderers together. Cross-reference: GCV-0006.

Security hardening did not cross the typed-migration reversal threshold: lifecycle remains one request field, and the additional authorization and observed-resource readiness checks are platform input/observed-state contracts rather than multiple lifecycle fields requiring atomic request parsing.

Local implementation evidence 2026-08-20: malformed lifecycle tests prove explicit type/path errors, the no-migration decision and reversal trigger are recorded, and integrated ./scripts/validate.sh passed with 89.3% statement coverage. Hosted exact-SHA evidence remains pending.

Published package evidence 2026-08-20: source/merge SHA d2343aef13dac0a41505d35da6ca2545ae1ed7df passed hosted Validate public reference run 32399213024. Publish Grafana vending function run 32399213015 succeeded; immutable amd64/arm64 package digest sha256:fb5e86a7a664572ef3383da16e85f1468c6d13ac8fd9abff61268daeb5bc44b8 verified with Cosign against https://github.com/rknightion/grafana-cloud-vending-machine/.github/workflows/publish-function.yml@refs/heads/main and is being pinned in the delivery commit.

Completion evidence: SHA d8d8ee9d90d714249178e468f908721e805a6af8; hosted Validate public reference run 32399726607 passed.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Recorded the decision to retain unstructured parsing, added explicit lifecycle type/path errors, and documented the two observable triggers for a future typed migration. Integrated validation passed locally and on the completing SHA.
<!-- SECTION:FINAL_SUMMARY:END -->
