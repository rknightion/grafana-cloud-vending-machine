---
id: GCV-0025
title: Add expiring sandbox stacks with a recorded extension trail
status: To Do
assignee: []
created_date: '2026-08-21 12:17'
labels: []
dependencies: []
ordinal: 25000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Grafana Cloud has no concept of an expiring stack, so a time-to-live has to be built. Crossplane suits this well because the reconcile loop already runs continuously.

Add an expiry to the stack request, have the function compare it against the clock, warn through the existing contact point or incident relay ahead of expiry, then drive deletion through the armed-delete path that already exists rather than inventing a second destructive route. Record extensions on the object so they are auditable rather than argued about.

The hard part is already built: the three-stage reviewed decommission path and the Retain-by-default lifecycle with an explicitly authorized Delete. This task adds the clock and the audit trail, and must not weaken either control. An expiry must not become a way to bypass the deletion authorization.

Deliberate scope limit: expiry applies to sandbox usage classes only, selected by platform configuration, so a production stack cannot acquire one.

Note the function currently derives deterministic values without wall-clock input; introducing time as an input affects reproducibility of renders and needs a testable seam rather than a direct clock read.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Expiry is available only to platform-approved sandbox usage classes
- [ ] #2 Expiry drives the existing armed-delete path and cannot bypass its authorization
- [ ] #3 Time is injected through a testable seam rather than read directly, and render reproducibility is preserved in tests
- [ ] #4 Warnings are emitted ahead of expiry through an already-vended delivery path
- [ ] #5 Extensions are recorded on the object with who and when
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
