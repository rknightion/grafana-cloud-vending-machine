---
id: GCV-0037
title: Write the 0.x to 1.0 migration guide for the breaking API changes
status: Done
assignee: []
created_date: '2026-09-08 17:02'
updated_date: '2026-09-08 18:39'
labels: []
dependencies: []
ordinal: 37000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Two breaking changes are queued for the first release: multi-organization vending, and the wave-3 governed vending surfaces with bounded token lifetimes. Release-please folds both into 1.0.0 in PR 29, which the owner is holding open. An adopter running an 0.x checkout has no statement anywhere of what changed in the request schema, which fields became required, which became creation-time-only or immutable, and what happens to an existing request that no longer validates. The breaking-change footers name the commits but not the migration, and the changelog is a commit list rather than an upgrade path. This has to exist before the release, not after it, because an adopter who applies 1.0.0 to an existing request finds out by rejection.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Every breaking schema change between 0.1.0 and the 1.0.0 candidate is enumerated with its before and after shape
- [x] #2 Each entry states what an adopter must do, including whether an existing request is rejected, silently changed, or unaffected
- [x] #3 Newly required, newly immutable and newly creation-time-only fields are called out separately from added optional fields
- [x] #4 The guide is published in the docs site and reachable from the release notes, not only present as a file
- [x] #5 Every example given is inert and carries no source-environment identifier, verified by the publication scan
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 4: implement the commissioned lane after the pushed root harness pre-pass; preserve frozen schemas and ownership; return acceptance evidence and required negative controls; root integrates, reviews, validates locally and at the exact hosted SHA, then reconciles status.
<!-- SECTION:PLAN:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Published the schema-led migration guide using the explicit unreleased baseline 85c4344a5146eea98b4bfa9fb1c110858cd1f152 because no 0.1.0 tag exists. It distinguishes required, immutable, creation-time-only and optional changes across four existing and ten new APIs, explains adopter actions/outcomes, preserves the original non-destructive adoption runbook, and states both known admission defects. Public page https://m7kni.io/grafana-cloud-vending-machine/migration-1.0/ was fetched successfully with the expected title, baseline, caveat and adoption section; the published docs home links to it. Documentation deployment run 34263373022 succeeded. Held release PR 29 links to the guide through repository-controlled release-header configuration. Publication scan passed. Completing checkpoint SHA 5c482ffcf2100714cfe1c3751733d805a501489a; hosted Validate 34263860085 success. Implementation SHA 9c559d105c5cd7761db3c9ca290150d35c936175; hosted Validate 34263352393 success. Local just check passed with 85.4% coverage. Main still has zero real admission cases and one skipped placeholder; these tasks do not claim admission completion.
<!-- SECTION:FINAL_SUMMARY:END -->
