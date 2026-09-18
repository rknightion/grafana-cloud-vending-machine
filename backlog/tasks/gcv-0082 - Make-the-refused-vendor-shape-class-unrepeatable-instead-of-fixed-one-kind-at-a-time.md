---
id: GCV-0082
title: >-
  Make the refused-vendor-shape class unrepeatable, instead of fixed one kind at
  a time
status: To Do
assignee: []
created_date: '2026-09-18 07:46'
labels:
  - needs-triage
  - vendor-defect
dependencies: []
priority: high
type: feature
ordinal: 82000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The Git Sync 403 was not a one-field mistake, it was a class, and the repository so far has fixed one instance of it. GCV-0081 is the second instance, found by reading rather than by any control. This task is the control.

THE CLASS. This platform composes request shapes against a vendor API it cannot exercise here, Crossplane retries a rejected shape indefinitely, and a refused shape therefore becomes continuous load on a live vendor tenant. The first occurrence paged the vendor's on-call at a weekend over a secure value their provisioning app could not read, once every three minutes, because a claim existed. Three things were missing then and are still missing.

1. NO RECORD OF WHAT THE VENDOR REFUSES, at a level anything can check. The Git Sync evidence lives in a renderer doc comment, a catalog README and a Backlog task - all prose, all discoverable only by someone already looking at that kind. A sibling API was designed against the refused shape and CEL-hardened to accept nothing else, and every gate stayed green, because no artefact states the prohibition in a form the gate can assert. What is needed is one place that names each refused shape, the exact error, the date and how it was observed, and an assertion in the gate that no composed child emits one. The existing connection test asserts the create form for that one field; the assertion needed is over every emitted child.

2. NO STATED CONSEQUENCE OF A PERMANENTLY REFUSED CHILD. The management policies are non-destructive by design, so a child that can never succeed is retried forever and no condition on the composite says so - the composite reported Synced and Ready throughout. GCV-0077 is the same family seen from the other end: this platform generates continuous vendor API load and nothing surfaces it. An operator needs to be told that the remedy for a 4xx child is to remove the claim, not to wait, and needs to be able to see that a child is in that state.

3. NO PROBING PROTOCOL, which is why the first occurrence left residue on someone's stack. dryRun=All is IGNORED by the vendor's secret API: two objects posted with it persisted as real resources and had to be deleted by hand. kubectl apply --dry-run=server against a real API server IS honoured, so the two behave oppositely with no indication from the vendor side, and an agent that assumes the Kubernetes semantics mutates a live tenant. Live verification is outside this repository's evidence boundary, so what belongs here is the written protocol a consumer estate follows: never probe on a stack anything depends on, assume every call mutates, record and remove residue.

ONE DECISION TO SETTLE WHILE HERE. The connection renderer still emits a standalone secure value that duplicates the credential inside the vendor, and its stated justification is that restoring the reference form later would then be a one-line change. GCV-0074 was subsequently amended to record the create form as PERMANENT design with the vendor defect never to be filed, which removes that justification. So the platform currently writes a second copy of a private key into a live tenant for a revert that is not planned - and an unreferenced secure value whose only decrypter is the provisioning app is the same object class that appeared in the vendor's alert. Decide whether it keeps being rendered. Removing it is a live-behaviour change on existing installations, so it is a decision with a migration note, not a deletion.

Scope boundary: this task builds the control and records the class. Fixing the instance it already found belongs to GCV-0081.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 One artefact in this repository names every vendor request shape known to be refused, with the exact error string, the date, and whether it was observed or inferred
- [ ] #2 The gate fails if any composed child emits a shape on that list, checked across every emitted kind rather than per-field in one test
- [ ] #3 The behaviour of a permanently refused child is documented: that it is retried indefinitely, that it generates continuous load on the vendor tenant, and what an operator should do about it
- [ ] #4 An operator can tell from resource conditions that a child is failing in a way no reconcile will fix, without reading provider logs
- [ ] #5 A written protocol for verifying a vendor shape against a live stack exists, stating that the vendor's secret API ignores dryRun and that every call must be assumed to mutate
- [ ] #6 A decision is recorded on whether the connection renderer keeps duplicating the credential as a standalone secure value, with a migration note if it stops
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
