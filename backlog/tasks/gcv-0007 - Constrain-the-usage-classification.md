---
id: GCV-0007
title: Constrain the usage classification
status: In Progress
assignee:
  - '@codex'
created_date: '2026-08-20 16:10'
updated_date: '2026-08-20 17:42'
labels:
  - api
dependencies: []
ordinal: 7000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
usage is a free-form string matching ^[a-z0-9-]+$ and is interpolated straight into the generated document path as {prefix}/{region}/{usage}/{slug}. spec.slug is immutable through a CEL validation; usage is not, and it has the same effect on external identity.

Two consequences. A typo is accepted and silently mints a whole separate credential path, with no error and nothing to compare against. And changing it on an existing request moves every future document to a new path while leaving the old ones behind: on a live control plane a poc to dev change orphaned four documents holding credentials for the previous path, which had to be found and deleted by hand.

Options, not exclusive: add an allowed set to the platform-owned Composition input, the same shape as ssoProfiles, so the platform decides the vocabulary and the function rejects anything else; or add a CEL immutability rule next to the one on slug; or both, since they solve different halves. If usage stays mutable, the reconciliation section should say plainly that changing it orphans previously published documents.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 usage is validated against a platform-owned set, or made immutable, or both
- [ ] #2 The consequence of changing it is documented wherever the output path is described
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Map the usage identity contract and every output-path description.
2. Freeze the platform-owned vocabulary and immutability behavior, then implement with focused validation/tests.
3. Update documentation, run the integrated gate and record exact-SHA hosted proof.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Design frozen 2026-08-20: use both controls. Add platform-owned allowedUsages with reference defaults development and production, reject any request value outside it, and make spec.usage immutable with CEL because it is part of external credential identity. Document the full prefix/region/usage/slug path and that immutability prevents orphaned documents at an old path.

Local implementation evidence 2026-08-20: allowedUsages validation, usage immutability, path regression coverage, documentation updates, and the integrated ./scripts/validate.sh gate passed with 89.3% statement coverage. Hosted exact-SHA evidence remains pending.
<!-- SECTION:NOTES:END -->
