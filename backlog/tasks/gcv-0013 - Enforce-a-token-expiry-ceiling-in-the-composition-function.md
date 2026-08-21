---
id: GCV-0013
title: Enforce a token expiry ceiling in the composition function
status: To Do
assignee: []
created_date: '2026-08-21 12:14'
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
- [ ] #1 The platform maximum lifetime is a Composition input, not a request field, so a request author cannot raise it
- [ ] #2 A rendered token always carries a bounded lifetime; the function fails closed rather than emitting an unbounded token
- [ ] #3 The composite status publishes each rendered token expiry
- [ ] #4 Tests cover the refusal path and the capping path
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
