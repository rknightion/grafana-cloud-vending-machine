---
id: GCV-0036
title: Close the hand-maintained registry drift class the catalog render gap exposed
status: In Progress
assignee: []
created_date: '2026-09-08 17:02'
updated_date: '2026-09-08 17:20'
labels: []
dependencies: []
ordinal: 36000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Four catalog directories shipped with no kustomization.yaml and failed to render while the gate stayed green, because scripts/validate.sh listed fourteen render paths by hand and there were eighteen directories. That was fixed in 28b4832de3bfafd5e3c0faa3a2bff31dc4bffbad by enumerating instead. The same shape exists elsewhere and has not been checked: platform/kustomization.yaml, the ManagedResourceActivationPolicy kind list and platform/rbac/composition-rbac.yaml are all append-only registries maintained by hand, and the wave operating model already names them as pre-assigned wiring-pass entries precisely because nothing derives them. A registry that silently stops covering a new resource is the same defect that shipped once with examples/catalog/minimal and again with these four, so the question is which of the remaining lists can drift and what would be observable when one does.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Every hand-maintained registry in the repository is inventoried, each recorded as either derivable-and-now-asserted or as deliberately manual with the reason
- [ ] #2 For each registry that can drift, the gate asserts its coverage against the thing it is supposed to enumerate
- [ ] #3 Each new assertion is proven by a negative control: an entry is removed, the gate fails naming it, and the removal is reverted
- [ ] #4 A registry left deliberately manual carries a comment at its definition saying so, so the next agent does not silently automate it
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 4: implement the commissioned lane after the pushed root harness pre-pass; preserve frozen schemas and ownership; return acceptance evidence and required negative controls; root integrates, reviews, validates locally and at the exact hosted SHA, then reconciles status.
<!-- SECTION:PLAN:END -->
