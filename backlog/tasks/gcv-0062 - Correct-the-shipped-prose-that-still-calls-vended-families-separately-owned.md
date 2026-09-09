---
id: GCV-0062
title: Correct the shipped prose that still calls vended families separately owned
status: Done
assignee:
  - campaign-root
created_date: '2026-09-09 12:48'
updated_date: '2026-09-09 14:56'
labels: []
dependencies: []
ordinal: 62000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
`docs/faq.md` says "Cloud integrations, OnCall schedules, ML and Asserts remain separately owned." ML is vended - `platform/apis/ml-v1beta1.yaml`, `platform/function/ml.go`, GCV-0053 Done in wave 6 - and cloud integrations and OnCall are too, as `GrafanaCloudIntegrations` and `GrafanaOnCall` in the `compositeRenderers` registry. Three of the four named families are wrong.

This is the drift class GCV-0057 deliberately did not cover. That gate check binds documents that declare a COMPLETE inventory under a known heading; a prose sentence naming families in passing declares nothing, so nothing catches it. The question this task has to answer is whether that class is checkable at all, or whether the honest answer is to stop writing per-family prose inventories and point at the one checked inventory instead.

Scope is the shipped public documentation set, not the tracker.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Every prose sentence in docs/ that names which provider families are or are not vended agrees with the compositeRenderers registry, checked family by family
- [x] #2 Either the gate fails on this class by path, or the prose is restructured so the claim lives only in an already-checked inventory and the reason that choice was made is recorded
- [x] #3 docs/architecture.md's per-family treatment table agrees with what is actually vended
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 8 lane D audits shipped family-ownership claims against actual renderers, corrects the three owned prose documents, and implements either path-specific drift checking via root wiring or consolidation into checked inventories with a recorded rationale. Root integrates the frozen asserts name and verifies local and exact-SHA hosted gates.
<!-- SECTION:PLAN:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Wave 8 completed at 3faf9a8b02a92d3777f42753c212e2bed346cf7f; hosted Validate public reference run 34366337713 succeeded and local just check passed with zero skips.

Removed false separately-owned family claims and the remaining duplicate FAQ module list. FAQ links the checked XRD inventory and architecture family table. The 19-row provider-family table now agrees with reachable constructors from registered renderers, including delegated ladder rendering and product ProviderConfigs. The Go check derives associations from source rather than a second hand-maintained mapping. Wrong family status and wrong composite controls fail at docs/architecture.md; missing managed map, activation and renderer controls fail by their exact paths. Every temporary mutation was byte-restored. Governance was checked and required no edit. k6 load-test/schedule and additional-service-account treatment claims were also corrected. Independent review approved; all root decision grades upheld.
<!-- SECTION:FINAL_SUMMARY:END -->
