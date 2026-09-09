---
id: GCV-0056
title: >-
  Close the synthetic monitoring surface and record the private-probe refusal as
  design
status: To Do
assignee: []
created_date: '2026-09-09 08:09'
labels: []
dependencies: []
ordinal: 56000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
GCV-0050 is Parked with AC1, AC3 and AC4 unmet. AC4 is the same cross-object admission limit that GCV-0049 hits and is settled the same way. AC1 is different: the pinned provider@s Probe CRD carries no token or token-lifetime field, so a vended private probe cannot meet the composition token expiry ceiling, and the XRD therefore refuses private probes outright with that exact reason, proven at the real API server.

The repository owner settled this on 2026-09-09: the refusal is the shipped design, not a gap. It is implemented, tested and already documented in the catalog README with its real cause. Carrying it as an unmet acceptance criterion makes a finished surface look like a failed lane. AC3, the alert-path delivery proof, is a separate question and is judged on its own merits rather than folded into this settlement.

Do not substitute Secret expiry for provider token expiry. That was considered and rejected in wave 6: they are not the same lifetime and the remote token would remain unbounded.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The private-probe refusal is recorded as a Backlog decision with the provider-source evidence, and GCV-0050 no longer carries it as an unmet criterion
- [ ] #2 The check-budget criterion states the reconcile-time boundary that actually holds, consistent with the k6 settlement
- [ ] #3 The CheckAlerts alert-path question is either proven at the boundary the machine controls or split into its own task with a concrete resume boundary; it is not closed by assertion
- [ ] #4 Public documentation for the synthetic monitoring surface makes no claim the machine does not meet, and the private-probe reason survives a provider-pin bump review
- [ ] #5 GCV-0050 reaches Done or Parked on a stated remaining fact, with the settled questions removed from its criteria
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
