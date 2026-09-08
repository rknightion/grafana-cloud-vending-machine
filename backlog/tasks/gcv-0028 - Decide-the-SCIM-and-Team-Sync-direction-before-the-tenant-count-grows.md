---
id: GCV-0028
title: Decide the SCIM and Team Sync direction before the tenant count grows
status: Done
assignee: []
created_date: '2026-08-21 12:18'
updated_date: '2026-09-08 16:43'
labels: []
dependencies: []
ordinal: 28000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
SCIM group synchronisation and legacy Team Sync are mutually exclusive: enabling group sync conflicts with the external-group mapping this repository vends today through its team-access and custom-role-binding APIs. SCIM group sync can create and delete teams from identity-provider group changes, whereas Team Sync only maps groups onto teams that already exist.

Enabling SCIM later is therefore a breaking migration for every existing tenant, not an additive feature. The cost of reversing this decision rises with every vended stack, which is why it wants deciding now rather than when someone asks for it.

SCIM is also not independent of SSO. It requires SAML specifically, and the SCIM configuration is inert without an SSO settings block carrying an external-uid assertion attribute matching the identity provider SCIM external identifier. If it is ever exposed it must be one coupled feature, not three independent knobs.

Two open questions to resolve as part of the decision: the exact Cloud plan floor, where sources disagree between Pro-and-above and Advanced-only; and the deprovisioning semantics when user sync is disabled after provisioning, specifically whether accounts are removed or frozen.

The identity-provider half of SCIM has no declarative coverage at all, so any decision to adopt it accepts manual console configuration per tenant.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A decision is recorded on whether SCIM is in scope, with the migration cost for existing tenants stated
- [x] #2 If in scope, SCIM and the SAML assertion attribute are modelled as one coupled feature
- [x] #3 The mutual exclusivity with external-group mapping is enforced or documented as an admission rule
- [x] #4 The plan floor and deprovisioning semantics are resolved and recorded
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

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
## Decision, taken by the repository owner: in scope, as one coupled feature

SCIM is exposed, and it is exposed as a single indivisible feature: SCIM configuration, the SAML SSO
block, and the external-uid assertion attribute move together. Three independent knobs is the failure
mode to avoid, because SCIM is inert without a SAML assertion attribute matching the identity
provider's SCIM external identifier.

Admission must reject external-group mapping and SCIM group sync on the same stack. They are mutually
exclusive: group sync creates and deletes teams from identity-provider group changes, while Team Sync
only maps groups onto teams that already exist. Enabling SCIM on a tenant that already has vended
external-group mapping is a breaking migration for that tenant, not an additive change, so the
migration path has to be written before the first tenant gets it.

Still to resolve inside this task, as AC 4 already requires: the exact Cloud plan floor, where sources
disagree between Pro-and-above and Advanced-only, and the deprovisioning semantics when user sync is
disabled after provisioning. Resolve both against a live tenant or Grafana documentation before
building, and record what was found.

The identity-provider half has no declarative coverage, so this decision accepts manual console
configuration per tenant. State that in the README next to the API rather than only here.

Correction for wave 3: the binding goal supersedes the older in-scope note. SCIM is out of scope; external-group mapping remains the supported identity model. The lane records the frozen decision and admission enforcement. Plan floor and disabled-user-sync semantics remain explicitly unresolved under the no-network decision brief; evidence required to settle them will be recorded.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Binding wave 3 decision supersedes the historical in-scope note: SCIM is out of scope. Team Sync remains supported; schema and defensive function admission reject SCIM and mixed external-group mapping. Future adoption requires coupled SAML and external UID plus a breaking tenant migration and manual identity-provider setup. AC2 is conditional and not applicable to the out-of-scope decision. Under goal section 5.3 lane H, AC4 is satisfied by explicitly recording both unresolved questions, not by claiming resolution: plan floor and disabled-user-sync semantics are absent from the pinned provider schema. Current authoritative product evidence or controlled tenant tests are required to settle them. SCIM admission race tests and schema validation passed. Completing delivery SHA bec9551c3c2abb009a4a50412b33efe47b07520c; hosted Validate 34252640140 success. Root just check passed (85.7% coverage). Signed multi-platform function digest sha256:09ff21ddf5436d0f0165ac7849d86ab4c22a6633551d91ab6aab4edc48f88652 is pinned in both locations. No live provider or deployment proof is claimed.
<!-- SECTION:FINAL_SUMMARY:END -->
