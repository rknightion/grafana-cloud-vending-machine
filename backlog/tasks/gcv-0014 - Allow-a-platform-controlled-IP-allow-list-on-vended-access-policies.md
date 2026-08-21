---
id: GCV-0014
title: Allow a platform-controlled IP allow-list on vended access policies
status: To Do
assignee: []
created_date: '2026-08-21 12:14'
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
- [ ] #1 The allow-list is selected from platform-owned configuration, not supplied verbatim by a request author
- [ ] #2 The rendered AccessPolicy carries conditions.allowedSubnets only when the selected profile defines one
- [ ] #3 The README distinguishes token-use restriction from inbound stack filtering and states the latter is unavailable
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
