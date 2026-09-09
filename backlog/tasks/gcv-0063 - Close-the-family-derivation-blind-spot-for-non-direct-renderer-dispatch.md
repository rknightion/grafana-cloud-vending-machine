---
id: GCV-0063
title: Close the family-derivation blind spot for non-direct renderer dispatch
status: Done
assignee:
  - campaign-root
created_date: '2026-09-09 16:27'
updated_date: '2026-09-09 18:22'
labels:
  - needs-triage
dependencies: []
priority: medium
ordinal: 63000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
`platform/function/family_docs_test.go` derives the documented provider-family ownership table by walking the AST from each entry in `compositeRenderers` and collecting every reachable `newDesired` call. That derivation is what replaced the hand-maintained ownership map, and it is what closed GCV-0062's drift class.

It only follows one call shape. The traversal recurses on `call.Fun.(*ast.Ident)` - a plain call to a top-level function in the package. A renderer reached through a method call on a receiver, or through a func-typed value or struct field, contributes its `newDesired` kinds invisibly: the derivation sees no constructors on that path, so `docs/architecture.md` can omit a whole composite or understate a family's status while the gate stays green.

No renderer uses that shape at HEAD. The only methods in the non-test package are `Function.RunFunction`, three `platformSettings` accessors, `inventoryDeclaration.statusMap`, `inventoryObject.statusMap` and `managedGVK.key`, and every renderer is a plain top-level function, so the check is complete today and the shipped table is correct. The gap is future-facing: the first feature area that dispatches through a method or a func value silently reopens the drift class the check exists to close, and it fails open rather than by path.

The same traversal shape is used for the dynamic-constructor path via `dynamicNewDesiredKinds`, so check whether that inherits the limitation too rather than assuming only the direct path is affected.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The family-ownership derivation follows renderer reachability through method calls and func-typed values, not only direct calls to top-level functions
- [x] #2 A renderer deliberately dispatched through a shape the current traversal cannot see is proven to fail the gate by path, and the source is restored byte-identically
- [x] #3 dynamicNewDesiredKinds is assessed for the same limitation and either fixed or explicitly ruled out with a reason
- [x] #4 The derivation fails closed: a call shape it cannot resolve is an error naming the unresolved construct, never a silently empty contribution
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 9 lane E follows method and func-value renderer reachability, proves hidden-dispatch negative controls and fails closed on unresolved calls; root integrates, reviews and verifies local plus exact-SHA hosted gate.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Wave 9 security review approved with historical findings and no unresolved code finding. Two CodeRabbit passes returned five minor findings: four fixed, one redundant omitted-ceiling case declined with accessor-equivalence evidence. One temporary file-ownership violation was restored byte-identically; one unscoped client dry-run has unknown schema-read contact and zero reported server mutations. Review hash-method limitations are preserved in the wave 9 report. Live behavior remains unproven.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Family ownership derivation follows local methods, pointer receivers, function values and possible assignments. Imported terminal origins are proved; unresolved local dispatch fails with source path. Dynamic constructors share the corrected reachability traversal. Deliberately hidden dispatch and twelve independent adversarial controls resolve the Cloud family or refuse by path; temporary mutations were restored byte-identically. Local factory, type-switch, shadowing, call-argument and result-position blind spots were exposed by failing controls and repaired. Acceptance completing commit 3314fb91bed68f4a4d3fe48b5c469a45a270f85a; hosted Validate public reference run 34388031402 succeeded. Local just check passed: 505 assertions, zero failures, zero skips, 84.9% coverage. Published function source 15b4df59f51391b1f492110276211d0b929c4c9d, verified digest sha256:e5b10d21e462e8892f2865a26cd263e4b576278ad4fa842c485ccd12233db2e9; wrong source SHA rejected before exact-source verification. Final tracker bookkeeping SHA is certified separately in the wave 9 terminal report.
<!-- SECTION:FINAL_SUMMARY:END -->
