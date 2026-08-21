---
id: GCV-0017
title: Add a datasource access API that aggregates LBAC rules per datasource
status: To Do
assignee: []
created_date: '2026-08-21 12:15'
labels: []
dependencies: []
ordinal: 17000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Add a platform-owned composite, one per datasource, owning the datasource, its permission items, and the single aggregated LBAC rules resource.

The design constraint is a shape mismatch, not a provider defect. DataSourceConfigLBACRules is one resource per datasource whose rules field is a JSON-encoded map keyed by team UID, and its provider documentation states it manages the entire LBAC rules tree and overwrites existing rules. The natural consumer-facing shape is per-team, so if the API exposes a per-team claim, several claims map onto one whole-tree resource and the last writer wins. The function must therefore fold every team claim into one resource, and admission must reject a second composite targeting the same datasource. This is the same pattern the existing content-access API already uses for folder and dashboard ACLs, extended to the one place where the resource looks per-team and is not.

Two security facts make this more than plumbing. LBAC is opt-in per team: a team with no rules on a datasource can query all of it, so omitting rules is an allow-all default rather than a deny. And LBAC is bypassed by anyone holding broader datasource query permission, so Grafana guidance is to strip default Viewer and Editor query access first. An API exposing LBAC without owning the datasource permissions ships a false sense of isolation.

Other verified constraints: LBAC applies only to datasources using basic auth; it requires Grafana 11.5.0 or later and an Enterprise or Cloud entitlement; prefer the permission item form over the whole-set permission form for everything else. Private networking is one field, private_data_source_connect_network_id, top-level on the datasource and Cloud-only, but the network and its token are Cloud-plane objects while the binding is stack-plane, so that combination needs two ProviderConfigs.

Keep endpoints and credentials out of this repository. Datasource connection details are environment-owned.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Exactly one composite may own a given datasource; a second claimant is rejected at admission
- [ ] #2 All team claims are aggregated into a single LBAC rules resource; no rendering path can emit two owners for one datasource
- [ ] #3 When any team claim exists, default broad query permission is removed rather than left in place
- [ ] #4 Permission item forms are used throughout; no whole-set permission resource is rendered
- [ ] #5 The API refuses or clearly warns when LBAC is requested on a datasource whose auth mode cannot support it
- [ ] #6 A catalog example renders with two team claims on one datasource and no credentials or endpoints
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
