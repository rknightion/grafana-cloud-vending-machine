---
id: GCV-0029
title: >-
  Decide how to reach Adaptive Metrics, the largest cost lever with no
  Crossplane coverage
status: To Do
assignee: []
created_date: '2026-08-21 12:18'
updated_date: '2026-08-27 09:27'
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

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
## Feasibility investigation of the Upjet provider route

Verified against grafana/terraform-provider-grafana-adaptive-metrics at v0.3.6 (MPL-2.0, actively pushed).

### Surface is small and flat
Six resources (rule, ruleset, segment, policy, exemption, recommendations_config) and one data source (recommendations). About 2,900 lines of non-test Go. Each resource carries two to eight attributes with almost no nesting. This is an easy Upjet target compared with the main Grafana provider.

Plugin-framework only (terraform-plugin-framework v1.14.1; SDKv2 is an indirect dependency). Upjet handles that path well: 67 of the 121 resources in the main Grafana Crossplane provider are plugin-framework and generate cleanly, so the mechanism is heavily exercised rather than theoretical.

The provider is published in the Terraform registry as grafana/grafana-adaptive-metrics, so the standard schema generation step used by the existing Crossplane provider works unchanged.

### Two upstream blockers, both small, both must land first
1. The module is not importable by any path. go.mod declares module github.com/hashicorp/terraform-provider-grafana-adaptive-metrics while the repository is github.com/grafana/terraform-provider-grafana-adaptive-metrics. Requiring the grafana path fails with a declared-path mismatch; requiring the hashicorp path fails because that repository does not exist. Both directions were tested. This is leftover from the HashiCorp scaffolding template.
2. The provider constructor New(version, commit) lives in internal/provider, so it is unreachable from outside the module even once the path is fixed. Upjet needs a live framework provider instance, so an exported shim is required.

Together these are roughly twenty lines of upstream change. Risk is low precisely because nothing can depend on the module today, so a module path rename breaks no consumer. The main Grafana provider already exposes pkg/provider for exactly this reason, so there is precedent.

### Design seam worth keeping
Segment is the natural per-team partition: it carries a selector, name, fallback_to_default, auto_apply.enabled and policy_id, and both rules and rulesets are scoped to a segment. One composite per segment is therefore the clean ownership boundary, and auto_apply.enabled is a genuine platform governance knob.

The ruleset resource states it replaces all rules in its segment, and upstream issue 79 (open since 2025-01-24) confirms it overwrites rules created individually. Resolve it the same way as the LBAC aggregation in GCV-0017: pick ruleset as the sole owner per segment and never mix it with individual rule resources.

External-name work is six configurations. Five resources implement ImportState. recommendations_config does not and is a singleton carrying only keep_labels and auto_apply.enabled, so it needs deliberate handling.

### Correction to the option list
Calling the Adaptive Metrics HTTP API from the composition function should be struck. Composition functions are deterministic renderers that run on every reconcile; they give no managed-resource lifecycle, no external-name semantics and no drift correction, and making external calls from one is an anti-pattern.

The better middle option is crossplane-contrib/provider-http, which is active and ships namespaced variants suitable for this repository. It gives real managed-resource lifecycle without maintaining a generated provider, at the cost of hand-written request bodies and weak typing.

### Auth maps cleanly
Provider config is url plus api_key in the form tenant-id:token, with optional http_headers, retries and debug. The vending machine already mints stack access policy tokens and knows the numeric tenant identifier, so composing the credential and mirroring it to the secret store follows the existing pattern.

## Decision, taken by the repository owner: out of scope, documented plainly

Adaptive Metrics is not vended here, and neither are adaptive logs, traces or profiles. The Upjet
route stays feasible and the feasibility findings above remain accurate, but neither generating a
second provider nor adopting provider-http is work this reference takes on. The README must say so
explicitly rather than leaving the largest cost lever unmentioned, and must name the ticket-based and
UI routes that remain.

Reversal cost is low and unchanged: nothing here depends on the decision, and the two upstream
blockers are still about twenty lines whenever someone wants them.

Remaining work on this task is documentation only: satisfy AC 1, 3 and 4. AC 2, the ruleset versus
individual-rule merge strategy, falls away with adoption and should be recorded as not applicable
rather than answered.
<!-- SECTION:NOTES:END -->
