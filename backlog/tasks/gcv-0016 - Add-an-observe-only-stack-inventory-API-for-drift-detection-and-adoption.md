---
id: GCV-0016
title: Add an observe-only stack inventory API for drift detection and adoption
status: To Do
assignee: []
created_date: '2026-08-21 12:14'
labels: []
dependencies: []
ordinal: 16000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The provider ships a third API-group family alongside the cluster-scoped and namespaced ones: an observe-only family generated from Terraform data sources, carrying roughly 50 read-only kinds across 11 groups. It includes set kinds such as FolderSet, DashboardSet, TeamSet, UserSet, LibraryPanelSet, ProbeSet, CollectorSet and OrganizationUsers. Nothing in this repository uses any of it.

Add a read-only composite that composes these set kinds for a referenced stack and publishes to status what actually exists versus what the platform declared. The output is the thing no current mechanism produces: a continuously reconciled list of objects created out of band that Crossplane does not manage.

This also replaces hand work in the migration runbook, which currently asks an operator to inventory an existing stack manually before handing over ownership. The same object then serves as the ongoing drift detector, which addresses the known limitation that Crossplane readiness only reports the children currently desired.

Nothing here can mutate a tenant, which makes it unusually safe to ship early. Note the agento11y family has no observe-only variant because upstream ships no corresponding data sources.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The API is read-only by construction and cannot render a mutating managed resource
- [ ] #2 It gates on the referenced stack the same way the existing access claims do
- [ ] #3 Status distinguishes declared, observed-and-managed, and observed-but-unmanaged objects
- [ ] #4 The migration section of the README points at this API instead of manual inventory steps
- [ ] #5 New observe-only kinds are added to the activation policy and a catalog example renders
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
