---
id: GCV-0063
title: Close the family-derivation blind spot for non-direct renderer dispatch
status: To Do
assignee: []
created_date: '2026-09-09 16:27'
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
- [ ] #1 The family-ownership derivation follows renderer reachability through method calls and func-typed values, not only direct calls to top-level functions
- [ ] #2 A renderer deliberately dispatched through a shape the current traversal cannot see is proven to fail the gate by path, and the source is restored byte-identically
- [ ] #3 dynamicNewDesiredKinds is assessed for the same limitation and either fixed or explicitly ruled out with a reason
- [ ] #4 The derivation fails closed: a call shape it cannot resolve is an error naming the unresolved construct, never a silently empty contribution
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
