---
id: GCV-0020
title: Vend baseline Fleet Management pipelines as a telemetry hygiene control
status: To Do
assignee: []
created_date: '2026-08-21 12:15'
updated_date: '2026-08-27 09:27'
labels: []
dependencies: []
ordinal: 20000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Fleet Management is the cleanest area surveyed: generally available, included on all tiers, complete resource and data-source coverage, corresponding provider kinds present at the current pin, and no traps found. Two kinds matter: Collector, carrying remoteAttributes, and Pipeline, carrying matchers in Alertmanager selector syntax evaluated against those attributes. Each collector polls and receives one merged configuration.

The strategic value is that it routes around an unreachable problem. Drop rules, cardinality limits and relabeling have no Grafana-Cloud-side declarative resource, but pipelines matched on a per-stack attribute enforce the same hygiene at source across every collector with no tenant action. That converts a governance gap into a governance control.

Fold in the attribution-label control here. Usage groups, which perform cost attribution, are UI-only and require the Advanced plan, so they cannot be vended. They group on labels that must already be present in the data. Enforcing team, cost-centre and environment labels through baseline pipelines makes chargeback possible later even though the grouping itself cannot be configured now.

Credential note: the collector authenticates with Fleet Management basic auth using the instance identifier as the username, not the stack service account token. That is a distinct credential surface from the per-stack ProviderConfig, minted from an access policy with the Fleet Management read scope, so this API needs its own credential path.

Keep matchers and label values generic; real attribute values are environment-owned.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Baseline pipelines are platform-owned and selected by profile, not authored in a request
- [ ] #2 The Fleet Management credential is minted and mirrored to the secret store rather than embedded
- [ ] #3 Attribution labels for team, cost centre and environment are enforced by a baseline pipeline
- [ ] #4 The README states that usage groups themselves are UI-only and Advanced-tier, so only the label half is vendable
- [ ] #5 A catalog example renders with generic matchers and no real attribute values
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
## Decision, taken by the repository owner: pipelines only, platform-owned

Crossplane owns baseline Pipelines selected by profile. Collectors are read through the observe-only
family for inventory and never written. Collectors self-register when the agent starts, so a written
Collector means Crossplane contests the attributes the collector reports about itself on every
reconcile, and a collector that never starts leaves a phantom object. Full Terraform parity minus the
one resource that fights back.

## Provider coverage verified at the current pin, so no bump is needed for this

`fleetmanagement.grafana.m.crossplane.io` Collectors and Pipelines are present at v2.13.0 and at the
pinned main build, plus observe-only Collector and CollectorSet. Pipeline field parity with the
Terraform resource is exact: name, contents, matchers, enabled, configType (ALLOY or OTEL) and
terraformSourceNamespace. Collector carries collectorType, enabled and remoteAttributes.

## Half the credential plumbing already exists

The provider merges the stack connection secret into the ProviderConfig credential map, and
`fleet_management_url` is one of the keys it lands. The per-stack ProviderConfig already sets
`stackSecretRef`, so the URL is present today with no change. What is missing is
`fleet_management_auth`, which is `<stack id>:<token>` where the token comes from an access policy
carrying `fleet-management:read` and `fleet-management:write`. Both scope names verified against the
Terraform resource documentation.

That is an exact clone of the existing telemetry-publisher chain in platform/function/access.go:
stack-realm AccessPolicy, AccessPolicyRotatingToken, PushSecret, then one extra key in the
`instance-credentials` ExternalSecret template. Reuse the 720h expiry and 168h early-rotation window
rather than inventing new ones.

Upstream note for whoever implements this: Terraform provider 4.43.0 double-canonicalizes Alloy
configs for multiline consistency, so pipeline contents round-trip differently than they did at the
old pin. Diff a rendered pipeline against the API before assuming drift is real.
<!-- SECTION:NOTES:END -->
