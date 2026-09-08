---
id: GCV-0014
title: Allow a platform-controlled IP allow-list on vended access policies
status: Done
assignee: []
created_date: '2026-08-21 12:14'
updated_date: '2026-09-08 16:43'
labels: []
dependencies: []
ordinal: 14000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
AccessPolicy supports a conditions object carrying allowedSubnets, which restricts the network locations a token issued under that policy may be used from. This repository already vends a stack-realm telemetry publisher access policy, so the field has an existing owner and needs no new kind.

Name the control precisely in the API and the docs. It restricts where a token may be used from. It is not inbound filtering of a Grafana stack, and it will be misread as such. No mechanism was found for restricting inbound client IPs to a stack.

The allow-list belongs to the platform, not the request author, so it should be selected by profile rather than supplied as free-form CIDRs in a request.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The allow-list is selected from platform-owned configuration, not supplied verbatim by a request author
- [x] #2 The rendered AccessPolicy carries conditions.allowedSubnets only when the selected profile defines one
- [x] #3 The README distinguishes token-use restriction from inbound stack filtering and states the latter is unavailable
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 ./scripts/validate.sh passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 3: root pushes fail-closed seams; assigned lane implements owned files test-first; root audits ownership, integrates documentation and wiring, reviews and validates, verifies signed package publication, pins both references, then finalizes with exact-SHA hosted validation.
<!-- SECTION:PLAN:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Platform-selected token-use subnet profiles render conditions only when configured; explicitly empty restrictions fail closed. README distinguishes token-use restriction from inbound stack filtering. Profile-present and profile-absent race tests passed. Completing delivery SHA bec9551c3c2abb009a4a50412b33efe47b07520c; hosted Validate 34252640140 success. Root just check passed (85.7% coverage). Signed multi-platform function digest sha256:09ff21ddf5436d0f0165ac7849d86ab4c22a6633551d91ab6aab4edc48f88652 is pinned in both locations. No live provider or deployment proof is claimed.
<!-- SECTION:FINAL_SUMMARY:END -->
