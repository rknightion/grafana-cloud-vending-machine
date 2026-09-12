---
id: GCV-0073
title: Define ordered decommission for Git Sync credentials
status: To Do
assignee: []
created_date: '2026-09-12 17:31'
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
- [ ] #1 The API has an explicit retain-default lifecycle contract for the external Connection and Securevalue
- [ ] #2 Any delete path is platform-authorized and orders Connection deletion before Securevalue deletion
- [ ] #3 Credential revocation and the ExternalSecret-owned Kubernetes Secret lifecycle are documented
- [ ] #4 Completion is observable without relying on disappearance of Kubernetes objects alone
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
