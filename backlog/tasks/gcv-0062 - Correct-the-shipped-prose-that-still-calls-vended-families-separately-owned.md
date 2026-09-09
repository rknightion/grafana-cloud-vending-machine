---
id: GCV-0062
title: Correct the shipped prose that still calls vended families separately owned
status: To Do
assignee: []
created_date: '2026-09-09 12:48'
updated_date: '2026-09-09 12:48'
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
- [ ] #1 Every prose sentence in docs/ that names which provider families are or are not vended agrees with the compositeRenderers registry, checked family by family
- [ ] #2 Either the gate fails on this class by path, or the prose is restructured so the claim lives only in an already-checked inventory and the reason that choice was made is recorded
- [ ] #3 docs/architecture.md's per-family treatment table agrees with what is actually vended
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
