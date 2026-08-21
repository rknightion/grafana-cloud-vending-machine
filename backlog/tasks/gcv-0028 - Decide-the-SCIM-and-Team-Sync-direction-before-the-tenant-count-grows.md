---
id: GCV-0028
title: Decide the SCIM and Team Sync direction before the tenant count grows
status: To Do
assignee: []
created_date: '2026-08-21 12:18'
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
