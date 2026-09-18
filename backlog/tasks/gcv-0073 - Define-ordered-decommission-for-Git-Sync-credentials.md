---
id: GCV-0073
title: Define ordered decommission for Git Sync credentials
status: Done
assignee: []
created_date: '2026-09-12 17:31'
updated_date: '2026-09-18 15:54'
labels:
  - needs-triage
dependencies: []
type: feature
ordinal: 73000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
GrafanaProvisioningConnection uses the common non-destructive management policies, so removing the claim removes its Kubernetes management objects while retaining the external Grafana Connection and Securevalue. The existing stack decommission contract does not define ordering or evidence for these separately vended children. Blindly switching both to Delete could remove the secure value before the connection or revoke a credential without review. Settle an explicit owner-authorized lifecycle covering Connection-before-Securevalue ordering, credential revocation, the ExternalSecret-owned Kubernetes Secret boundary, and observable completion evidence.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The API has an explicit retain-default lifecycle contract for the external Connection and Securevalue
- [x] #2 Any delete path is platform-authorized and orders Connection deletion before Securevalue deletion
- [x] #3 Credential revocation and the ExternalSecret-owned Kubernetes Secret lifecycle are documented
- [x] #4 Completion is observable without relying on disappearance of Kubernetes objects alone
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 14: define and implement retain-by-default Git Sync credential lifecycle, platform-authorized ordered deletion, and an observable completion signal without changing a released API incompatibly.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Wave 14 implemented retain-default external lifecycle, platform-bound deletion authorization, observed phase witnesses, Connection-before-Securevalue withdrawal, and composite decommission status. Local just check passed with pinned envtest and KUBECONFIG=/dev/null. No live deletion or credential revocation was exercised.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Completed at 1f2af4a0b37c244a799e1fe0cdfcea2fa5abd82e; hosted Validate public reference run 35364595851 passed at that exact SHA. Ordered decommission is schema-backed, renderer-enforced, documented, and covered by focused plus integrated tests.
<!-- SECTION:FINAL_SUMMARY:END -->
