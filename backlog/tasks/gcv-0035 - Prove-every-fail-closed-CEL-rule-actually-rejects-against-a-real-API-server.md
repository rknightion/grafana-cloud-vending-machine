---
id: GCV-0035
title: 'Prove every fail-closed CEL rule actually rejects, against a real API server'
status: To Do
assignee: []
created_date: '2026-09-08 17:02'
labels: []
dependencies:
  - GCV-0034
ordinal: 35000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The vending machine's safety story rests on rules that are asserted to reject: SCIM presence, creation-time-only retention and expiry, immutable ladder identity and ordering, append-only extension records, explicitly empty subnet restrictions, omitted k6 zone intent. Each is currently proven only by a Go unit test over the renderer or by reading the YAML. Admission is the layer that actually enforces them, and a rule that compiles is not a rule that rejects: a mistyped field path, a pruned subtree or a rule attached at the wrong level all admit the request the platform intended to refuse. The negative case is the one that matters here, because every one of these rules exists to be the last line before a credential, a deletion or a billing surface.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Each fail-closed rule has a table-driven case that submits the forbidden request to the real apiserver and asserts admission is refused
- [ ] #2 Each rejection case asserts on the rule's own message, so a request refused for an unrelated reason cannot pass as proof
- [ ] #3 The paired allowed case is admitted for every rule, so no case passes by rejecting everything
- [ ] #4 Immutability rules are proven by admitting a create then submitting the forbidden update, not by a create alone
- [ ] #5 A deliberately weakened rule is shown to fail the harness, recorded as a negative control in the task summary
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
