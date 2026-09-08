---
id: GCV-0051
title: Vend in-stack service accounts under the token expiry ceiling
status: In Progress
assignee: []
created_date: '2026-09-08 22:36'
updated_date: '2026-09-08 22:48'
labels: []
dependencies: []
type: feature
ordinal: 51000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
GCV-0013 enforces a composition-level token expiry ceiling, and today that ceiling covers the cloud StackServiceAccount and its rotating tokens. The in-stack side is not vended at all: oss serviceaccounts, serviceaccountrotatingtokens, serviceaccounttokens, serviceaccountpermissions and serviceaccountpermissionitems are all present in the pinned provider and none is emitted.

That means the ceiling covers roughly half the tokens a real platform hands out. A team that needs an in-stack service account for a Terraform run or a CI job creates one by hand today, outside the ceiling, outside the rotation policy and outside the decommission path - which is precisely the class of credential the machine exists to stop existing.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 In-stack service accounts and their rotating tokens are vended from platform-controlled input, with an explicit external name on every deterministic child
- [ ] #2 The existing composition token expiry ceiling binds in-stack tokens exactly as it binds cloud tokens, and a request exceeding it is refused with its own message, proven against the real API server
- [ ] #3 Service account permissions have exactly one declarative owner, consistent with the whole-set resources rule, and that ownership is tested
- [ ] #4 No token value reaches status or a rendered example; only derived credentials are published, and a test proves it
- [ ] #5 Every emitted kind appears in the provider activation map and the XRD/renderer registry, and the gate fails by path if one is missing
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 6: implement the commissioned surface under the frozen goal and root-owned integration; prove admission and renderer boundaries with required negative controls, then just check and exact-SHA hosted Validate before finalization.
<!-- SECTION:PLAN:END -->
