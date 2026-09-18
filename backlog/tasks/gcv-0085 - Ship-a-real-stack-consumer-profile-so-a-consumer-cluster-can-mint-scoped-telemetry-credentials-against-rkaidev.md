---
id: GCV-0085
title: >-
  Ship a real stack consumer profile so a consumer cluster can mint scoped
  telemetry credentials against rkaidev
status: To Do
assignee: []
created_date: '2026-09-18 10:13'
labels: []
dependencies:
  - GCV-0075
priority: high
type: feature
ordinal: 85000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
GCV-0061 shipped the GrafanaStackConsumer API and the renderer that mints a scoped consumer credential against a stack the platform does not own, but the only profile that exists is the example-reader placeholder in the shipped Composition, pointing at exampleobservedstack in prod-us-central-0 with providerConfigName grafana-cloud-org-example-primary and a single metrics:read scope. Because the profile owns the ProviderConfig, the authorized targets, the scopes, the consumer identity and the output secret path, and the ValidatingAdmissionPolicy denies any GrafanaStackConsumer whose named profile is not complete and does not authorize its namespace, slug and region, no consumer cluster can actually use the feature yet.

The concrete driver is the Legend AI workshop estate in rkps-awsinfra, which needs telemetry credentials for the rkaidev stack (prod-gb-south-1, org 1390833, stack id 1802885) carrying sigil:write and metrics:read alongside the publisher scopes. Its own attempt to hand-author a standalone AccessPolicy plus AccessPolicyRotatingToken was parked on 2026-09-18 because the AccessPolicyRotatingToken CRD exposes accessPolicyId as a literal string with no Ref or Selector, so a static manifest set cannot order the policy before the token. This renderer already solves exactly that, in observedStackConsumerPolicyID, by reading status.atProvider.policyId off the observed policy before emitting the token, which is why a profile here is the right answer and a hand-authored manifest set in the consumer repo is not.

Two things a future agent cannot recover from the code. The telemetry publisher scope list at platform/function/access.go:69 is deliberately NOT the place to add these scopes: that list is composed into every stack the platform vends, so widening it grants sigil:write fleet-wide. The consumer profile scope list is per-profile and is passed through stackConsumerPolicy with no allowlist, only a non-empty string check, so it is the bounded surface. Separately, consumers of this profile inherit GCV-0075: stackconsumer.go:213 is one of the three AccessPolicyRotatingToken emit sites, so a consumer token cannot rotate until that lands, and rkaidev reaches its early rotation window on 2026-10-09 and expires 2026-10-16.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A named non-placeholder stack consumer profile authorizes the rkaidev stack by slug and region and is served by the shipped Composition
- [ ] #2 The profile carries the scopes a telemetry consumer needs, including sigil:write and metrics:read, and the scope list is justified against what the consumer actually writes
- [ ] #3 The profile names a real organization ProviderConfig and a real consumer namespace, and a GrafanaStackConsumer created in that namespace for that slug and region is admitted rather than denied by the ValidatingAdmissionPolicy
- [ ] #4 The minted credential reaches the platform secret store at the profile's outputSecretPath in the documented telemetry.json shape, verified by reading it back rather than from the PushSecret's own status
- [ ] #5 The example-reader placeholder's fate is recorded: kept as documentation, renamed, or removed, with the reason
- [ ] #6 The interaction with GCV-0075 is stated on this task: whether a consumer profile may ship before rotation is fixed, and what an operator does at the 2026-10-09 window if it ships first
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
