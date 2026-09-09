---
id: GCV-0047
title: Vend cloud provider integrations and metrics scrape jobs
status: Done
assignee: []
created_date: '2026-09-08 22:36'
updated_date: '2026-09-09 00:53'
labels: []
dependencies: []
type: feature
ordinal: 47000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Three provider groups are entirely unvended: cloudprovider (awsaccounts, awscloudwatchscrapejobs, awsresourcemetadatascrapejobs, azurecredentials), connections (metricsendpointscrapejobs) and cloudintegrations. This is how a vended stack actually acquires cloud telemetry, so a stack vended today arrives empty and a human wires the ingest by hand.

It is also the strongest fit for a vending machine control that does not exist yet: a platform-owned scrape budget. Uncontrolled CloudWatch scrape jobs are the largest ingest cost lever on the list, and this repository already established the budget pattern for synthetic monitoring and k6.

PUBLICATION CONSTRAINT, and it is the reason this task is riskier than its neighbours: the public-release scan rejects AWS ARNs carrying an account ID, and it scans every reachable Git revision, so one careless example permanently red-lights the gate. Every example identifier is synthetic and inert by construction. No live cloud account, role or credential is contacted or recorded.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Cloud provider accounts and scrape jobs are vended per stack from platform-controlled input, with an explicit external name on every deterministic child
- [x] #2 A platform-owned scrape budget caps job count and scrape frequency, and an over-budget request is refused with its own message, proven against the real API server
- [x] #3 No credential value is ever published to status or to a rendered example; only derived or referenced credentials are used, and a test proves no secret material leaks
- [x] #4 Every example identifier is synthetic; the public-release scan passes on the working tree and on the completing revision
- [x] #5 Every emitted kind appears in the provider activation map and the XRD/renderer registry, and the gate fails by path if one is missing
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 6: implement the commissioned surface under the frozen goal and root-owned integration; prove admission and renderer boundaries with required negative controls, then just check and exact-SHA hosted Validate before finalization.
<!-- SECTION:PLAN:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Vended cloud accounts and scrape jobs under the actual Composition count/frequency budget. Dedicated clients consume an observed organization credential Secret reference and an identity-checked managed Stack Cloud Provider endpoint. Ordinary CloudIntegration uses stack credentials. Real provider admission/readback and budget weaken/admit/restore passed. No cloud ingest or live authorization proof. Completing source/pin SHA: 187b03ea40ee32fcea890e40c138f00a8c73bd5f. Hosted Validate run 34296536930: success. Local just check passed with 23 real API-server tests, zero skips.
<!-- SECTION:FINAL_SUMMARY:END -->
