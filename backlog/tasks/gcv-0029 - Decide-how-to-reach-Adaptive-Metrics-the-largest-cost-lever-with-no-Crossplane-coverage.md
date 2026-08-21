---
id: GCV-0029
title: >-
  Decide how to reach Adaptive Metrics, the largest cost lever with no
  Crossplane coverage
status: To Do
assignee: []
created_date: '2026-08-21 12:18'
labels: []
dependencies: []
ordinal: 29000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Adaptive Metrics is the single largest cost lever in Grafana Cloud and is unreachable from the pinned provider. It is not part of the Grafana Terraform provider at all: it is a separate first-party Terraform provider carrying rule, ruleset, recommendations-config and exemption resources, and there is no corresponding Crossplane provider. Verified: the pinned provider contains no adaptive resources of any kind.

Its credential shape also differs from everything else vended here: basic auth directly against the Mimir endpoint, with the numeric tenant identifier as the username and an access policy token as the password. That is a per-stack credential of exactly the kind this repository already mints and mirrors to a secret store, which makes the option unusually cheap here specifically.

Three routes to evaluate: generate a second Crossplane provider from the upstream Terraform provider; call the HTTP API from the existing composition function, which already owns credential handling; or declare it out of scope and document that plainly.

Known upstream defect to account for either way: the ruleset resource overwrites anything applied by individual rules or by the recommendations engine, so if tenants both accept auto-recommendations and add custom rules, an explicit merge strategy is required rather than assumed.

The sibling adaptive products are worse off and should probably be scoped out in the same decision: adaptive logs, traces and profiles are all generally available with no declarative route found, and for logs and traces no API documentation was locatable.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A decision is recorded choosing one of the three routes, with the reasoning and the reversal cost
- [ ] #2 If adopted, the ruleset versus individual-rule merge conflict has a stated strategy
- [ ] #3 The adaptive logs, traces and profiles position is stated in the same decision
- [ ] #4 The README records whichever routes are declared out of scope, rather than leaving them unmentioned
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
