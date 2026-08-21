---
id: GCV-0027
title: Prohibit public dashboards by construction in vended custom roles
status: To Do
assignee: []
created_date: '2026-08-21 12:17'
labels: []
dependencies: []
ordinal: 27000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
There is no Cloud-compatible configuration flag that disables public dashboards, but withholding the public-dashboard write action from every vended custom role achieves it organization-wide. This repository already owns custom roles through its access APIs, so this is a small, contained change rather than a new surface.

Make it a platform-controlled default rather than a request option: a request author should not be able to grant the action to themselves. Where a tenant genuinely needs public dashboards, that becomes an explicitly authorized platform profile decision, in the same shape as the existing SSO and incident profile pattern.

Note the related constraint already recorded in known limitations: built-in Viewer, Editor and Admin definitions cannot be globally rewritten through this provider, so this control applies to custom roles and must not be described as an organization-wide guarantee covering basic roles.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The public-dashboard write action cannot be granted through a request field
- [ ] #2 Allowing it requires an explicit platform profile, consistent with the existing profile pattern
- [ ] #3 The README states the control covers custom roles and does not rewrite built-in basic roles
- [ ] #4 A test asserts the action is absent from a default-profile rendering
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
