---
id: GCV-0026
title: >-
  Make retention class a stack-request input, because export does not exist at
  decommission
status: To Do
assignee: []
created_date: '2026-08-21 12:17'
labels: []
dependencies: []
ordinal: 26000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Long-lived telemetry can only be protected at stack creation, not at decommission, so retention class belongs in the request rather than in a runbook.

Verified constraints. Cloud logs export syncs Loki chunks to a customer-owned bucket but only on a rolling window of roughly seven to thirty days, so it extends retention forward rather than dumping history. No equivalent bulk export was found for metrics or traces. Logs retention itself is adjustable through a self-serve API in thirty-day multiples with a one-year maximum, but below thirty days needs a support request, and there is an open upstream issue tracking a declarative resource for it, so today it is not reconcilable. Metrics and traces retention has no self-serve API at all. Stack deletion is permanent.

The consequence is that decommissioning a stack destroys its historical metrics and traces unless a durable sink has been written to since day one. Teams typically discover this during decommissioning, when it is unfixable.

So the vendable half is the fan-out, not the retention setting: a long-term retention class provisions remote-write to a durable sink at creation. State plainly in the README that retention periods themselves cannot be reconciled and name the ticket-based routes, rather than exposing a field that silently does nothing.

Leave room for the logs retention API as an escape hatch if it becomes worth calling directly, but do not build that on this task.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A retention class in the request provisions durable fan-out at creation for the long-term class
- [ ] #2 No spec field implies control over a retention period that cannot actually be reconciled
- [ ] #3 The README states which retention controls are ticket-based or API-only and that metrics and traces have no bulk export
- [ ] #4 The decommission runbook cross-references this as a creation-time decision
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
