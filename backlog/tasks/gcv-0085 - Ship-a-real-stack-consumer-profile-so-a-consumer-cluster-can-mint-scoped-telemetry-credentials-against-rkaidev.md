---
id: GCV-0085
title: >-
  Ship a real stack consumer profile so a consumer cluster can mint scoped
  telemetry credentials
status: Done
assignee: []
created_date: '2026-09-18 10:13'
updated_date: '2026-09-18 15:54'
labels: []
dependencies:
  - GCV-0075
priority: high
type: feature
ordinal: 85000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
GCV-0061 shipped the GrafanaStackConsumer API and the renderer that mints a scoped consumer credential against a stack the platform does not own, but the only profile that exists is the example-reader placeholder in the shipped Composition at platform/apis/stack-consumer-v1beta1.yaml:96, pointing at exampleobservedstack in prod-us-central-0 with providerConfigName grafana-cloud-org-example-primary and a single metrics:read scope. Because the profile owns the ProviderConfig, the authorized targets, the scopes, the consumer identity and the output secret path, and the ValidatingAdmissionPolicy denies any GrafanaStackConsumer whose named profile is not complete and does not authorize its namespace, slug and region, no consumer cluster can actually use the feature yet.

THE SHIPPED REFERENCE PROFILE TARGETS THE robk STACK IN prod-gb-south-1. Owner decision 2026-09-18: a real stack slug and region are publishable in this repository, so the profile that ships here is a working one rather than another placeholder. A separate consumer estate needs telemetry credentials for its own stack; that estate authors its own profile in its own repository from this one as the template, so no consumer-estate identity enters this repository.

The driving consumer estate's own attempt to hand-author a standalone AccessPolicy plus AccessPolicyRotatingToken was parked on 2026-09-18 because the AccessPolicyRotatingToken CRD exposes accessPolicyId as a literal string with no Ref or Selector, so a static manifest set cannot order the policy before the token. This renderer already solves exactly that, in observedStackConsumerPolicyID, by reading status.atProvider.policyId off the observed policy before emitting the token, which is why a profile here is the right answer and a hand-authored manifest set in a consumer repo is not.

Two things a future agent cannot recover from the code. The telemetry publisher scope list at platform/function/access.go:69 is deliberately NOT the place to add consumer scopes: that list is composed into every stack the platform vends, so widening it grants a write scope fleet-wide. The consumer profile scope list is per-profile and is passed through stackConsumerPolicy with no allowlist, only a non-empty string check, so it is the bounded surface. Separately, consumers of this profile inherit GCV-0075: stackconsumer.go:213 is one of the three AccessPolicyRotatingToken emit sites, so a consumer token cannot rotate until that lands. The driving estate's stack reaches its early rotation window on 2026-10-09 and expires 2026-10-16, which is the deadline this interaction has to be answered against.

IDENTIFIER SCRUB 2026-09-18: this description previously carried a numeric org id, a numeric stack id, a private repository name and an internal project name. All are removed. They remain in reachable history, which cannot be rewritten; GCV-0086 adds the control that stops the class recurring. Stack slugs and regions are permitted here by owner decision; bare numeric ids and private repository or project names are not.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A named non-placeholder stack consumer profile authorizes the robk stack in prod-gb-south-1 by slug and region and is served by the shipped Composition
- [x] #2 The profile carries the scopes a telemetry consumer needs, and every scope is justified against what such a consumer actually writes rather than copied from the fleet-wide publisher list
- [x] #3 The profile names an organization ProviderConfig and a consumer namespace that are not example placeholders, and a GrafanaStackConsumer created in that namespace for that slug and region is admitted rather than denied by the ValidatingAdmissionPolicy, proven at the pinned API server
- [x] #4 The emitted credential path and PushSecret render the documented telemetry.json shape at the profile's outputSecretPath, proven by renderer test; live read-back of the secret store is recorded as not exercisable in this repository and left to the consuming estate
- [x] #5 The example-reader placeholder's fate is recorded: kept as documentation, renamed, or removed, with the reason
- [x] #6 The interaction with GCV-0075 is stated on this task: whether a consumer profile may ship before rotation is fixed, and what an operator does at the 2026-10-09 window if it ships first
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 14: replace the placeholder consumer profile, prove admission and renderer output against the pinned ephemeral API server, document the bounded telemetry scopes, and record the rotation interaction.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Rotation decision (2026-09-18): platform/function/stackconsumer.go is a rotating-token site because it emits AccessPolicyRotatingToken. The consumer profile may ship before GCV-0075 lands, but its credential must be treated as non-renewing until that repair is deployed. If it ships first, the operator must perform an explicit approved credential replacement and publication before the 2026-10-09 early-rotation window, verify the consuming estate has reloaded the replacement, and must not rely on automatic rotation; if that handover cannot be performed, remove the request before the token expiry window.

Wave 14 shipped the named telemetry-writer profile, admitted its inert request on the pinned API server, proved the exact telemetry.json publication shape and fixed output path, removed the unusable placeholder, and documented the non-renewing credential handover. Live secret-store read-back was not exercised, and not exercisable here.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Completed at 1f2af4a0b37c244a799e1fe0cdfcea2fa5abd82e; hosted Validate public reference run 35364595851 passed at that exact SHA. The bounded write-only profile is usable as an inert public reference, with automatic rotation explicitly unavailable while GCV-0075 remains Parked.
<!-- SECTION:FINAL_SUMMARY:END -->
