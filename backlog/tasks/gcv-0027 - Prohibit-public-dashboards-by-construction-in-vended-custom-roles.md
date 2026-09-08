---
id: GCV-0027
title: Prohibit public dashboards by construction in vended custom roles
status: Done
assignee: []
created_date: '2026-08-21 12:17'
updated_date: '2026-09-08 13:48'
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
- [x] #1 The public-dashboard write action cannot be granted through a request field
- [x] #2 Allowing it requires an explicit platform profile, consistent with the existing profile pattern
- [x] #3 The README states the control covers custom roles and does not rewrite built-in basic roles
- [x] #4 A test asserts the action is absent from a default-profile rendering
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 ./scripts/validate.sh passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 2: verify the exact public-dashboard action, filter it by platform profile in vended custom roles, and add focused tests; root integrates and gates.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Wave 1 lane I disposition, 2026-09-08: Not started and Parked because the mandatory root pre-fan-out pass did not produce a pushed seam SHA after route metadata was unavailable. Resume after GCV-0032 completes the section 5.0 pass, then spawn EXECUTION on gpt-5.6-luna at max effort with fork_turns none and the pushed pre-pass SHA. No acceptance criterion or Definition of Done item was checked.

Wave 2 verification: focused role tests proved dashboards.public:write is stripped by default and retained only for a platform-authorized referenced-stack profile. Integrated local gate and hosted Validate run 34233686654 passed.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Delivered platform-profile-controlled public-dashboard protection for vended custom roles without exposing a request-side grant or rewriting built-in roles. Completing SHA 83f81afee7526fd6e7c4ec0a47675774d00036b8; hosted Validate run 34233686654 succeeded.
<!-- SECTION:FINAL_SUMMARY:END -->
