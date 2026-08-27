---
id: GCV-0028
title: Decide the SCIM and Team Sync direction before the tenant count grows
status: To Do
assignee: []
created_date: '2026-08-21 12:18'
updated_date: '2026-08-27 09:27'
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
- [ ] #1 A decision is recorded on whether SCIM is in scope, with the migration cost for existing tenants stated
- [ ] #2 If in scope, SCIM and the SAML assertion attribute are modelled as one coupled feature
- [ ] #3 The mutual exclusivity with external-group mapping is enforced or documented as an admission rule
- [ ] #4 The plan floor and deprovisioning semantics are resolved and recorded
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

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
<!-- SECTION:NOTES:END -->
