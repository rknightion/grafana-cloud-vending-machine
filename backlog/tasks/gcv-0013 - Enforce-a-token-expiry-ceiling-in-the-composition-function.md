---
id: GCV-0013
title: Enforce a token expiry ceiling in the composition function
status: Done
assignee: []
created_date: '2026-08-21 12:14'
updated_date: '2026-09-08 16:43'
labels: []
dependencies: []
ordinal: 13000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Grafana Cloud has no organization-level control requiring tokens to expire. On AccessPolicyToken the expires_at field is optional, and on StackServiceAccountToken secondsToLive is optional; omitting either yields a token that never expires. There is no product setting that forbids this.

The composition function is therefore the only place in the system where a token lifetime policy can be enforced, which is an argument for the platform existing rather than merely a feature of it.

Add a platform-controlled token policy: refuse to render any token without a bounded lifetime, cap the requested lifetime at a platform maximum carried in the Composition input rather than the request, and publish the resulting expiry to the composite status so a fleet-wide credential-age view is possible from Kubernetes alone.

This repository already uses the rotating token variants, which is the correct baseline; this task adds the ceiling and the observability, not rotation.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The platform maximum lifetime is a Composition input, not a request field, so a request author cannot raise it
- [x] #2 A rendered token always carries a bounded lifetime; the function fails closed rather than emitting an unbounded token
- [x] #3 The composite status publishes each rendered token expiry
- [x] #4 Tests cover the refusal path and the capping path
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

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Mandatory Composition maximum bounds administrator, telemetry and Fleet rotating tokens; missing or invalid policy fails closed. Provider-observed expiries are published to status. Refusal, capping and status branches passed race tests. Completing delivery SHA bec9551c3c2abb009a4a50412b33efe47b07520c; hosted Validate 34252640140 success. Root just check passed (85.7% coverage). Signed multi-platform function digest sha256:09ff21ddf5436d0f0165ac7849d86ab4c22a6633551d91ab6aab4edc48f88652 is pinned in both locations. No live provider or deployment proof is claimed.
<!-- SECTION:FINAL_SUMMARY:END -->
