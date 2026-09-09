---
id: GCV-0056
title: >-
  Close the synthetic monitoring surface and record the private-probe refusal as
  design
status: Done
assignee:
  - '@codex-wave7'
created_date: '2026-09-09 08:09'
updated_date: '2026-09-09 10:53'
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
- [x] #1 The private-probe refusal is recorded as a Backlog decision with the provider-source evidence, and GCV-0050 no longer carries it as an unmet criterion
- [x] #2 The check-budget criterion states the reconcile-time boundary that actually holds, consistent with the k6 settlement
- [x] #3 The CheckAlerts alert-path question is either proven at the boundary the machine controls or split into its own task with a concrete resume boundary; it is not closed by assertion
- [x] #4 Public documentation for the synthetic monitoring surface makes no claim the machine does not meet, and the private-probe reason survives a provider-pin bump review
- [x] #5 GCV-0050 reaches Done or Parked on a stated remaining fact, with the settled questions removed from its criteria
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Apply the frozen private-probe and cross-object admission settlements, record the provider-evidenced decision, prove the existing CheckAlerts-to-alerting linkage at the configuration boundary, reconcile public documentation, and close GCV-0050.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Created accepted decision decision-0002 with provider-source evidence, the Secret-expiry rejection, and the exact provider-pin condition that could reopen private probes. Reconciled the catalog README to separate declared-profile admission, observed-stack reconciliation, rendered and provider-admitted CheckAlerts linkage, and unproven rule evaluation or delivery. Focused renderer and provider-admission tests passed; weakening the positive numeric observed Check ID guard made TestSyntheticMonitoringRendersCheckAlertsOnlyFromObservedCheckIDs fail because it rendered CheckAlerts from malformed observation. CodeRabbit was skipped because this lane changed documentation and tracker records only; no branching logic changed.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Closed GCV-0050 with private-probe refusal as shipped design, declared-profile admission versus observed-stack reconciliation stated, and CheckAlerts proven through render and provider admission. decision-0002 carries the provider-bump review condition. Completing SHA b1ceaa36d63dc8b449215b9b362103d7da9b1dbf; hosted Validate public reference run 34337220901 succeeded; local just check passed at 84.3% with zero skips.
<!-- SECTION:FINAL_SUMMARY:END -->
