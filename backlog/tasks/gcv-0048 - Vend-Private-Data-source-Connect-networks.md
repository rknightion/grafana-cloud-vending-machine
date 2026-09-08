---
id: GCV-0048
title: Vend Private Data source Connect networks
status: To Do
assignee: []
created_date: '2026-09-08 22:36'
labels: []
dependencies: []
type: feature
ordinal: 48000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
cloud privatedatasourceconnectnetworks and privatedatasourceconnectnetworktokens are both present in the pinned provider and neither is vended. PDC is how a vended stack reaches a datasource that is not on the public internet, so without it the machine can only vend stacks for teams whose data is already exposed.

It belongs in a fail-closed machine specifically because it is a network boundary control, not a convenience: the network is the thing that decides what a stack can and cannot reach. The token half is the sensitive half and this repository already has the pattern for it - bounded lifetimes under the composition token expiry ceiling, derived credentials only, nothing published to status.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A PDC network is vended per stack from platform-controlled input, with an explicit external name so a re-adopt does not attempt a create
- [ ] #2 PDC network tokens are bounded by the existing composition token expiry ceiling, and a request that would exceed it is refused with its own message
- [ ] #3 No token value reaches status or a rendered example; only derived credentials are published, and a test proves it
- [ ] #4 A datasource request that names a PDC network the platform did not vend is refused with its own message, proven against the real API server
- [ ] #5 Every emitted kind appears in the provider activation map and the XRD/renderer registry, and the gate fails by path if one is missing
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
