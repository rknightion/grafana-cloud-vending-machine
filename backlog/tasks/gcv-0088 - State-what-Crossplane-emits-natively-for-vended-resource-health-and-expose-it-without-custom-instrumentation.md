---
id: GCV-0088
title: >-
  State what Crossplane emits natively for vended resource health and expose it
  without custom instrumentation
status: To Do
assignee: []
created_date: '2026-09-18 16:45'
updated_date: '2026-09-18 16:47'
labels:
  - needs-triage
dependencies: []
ordinal: 88000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Asked by the owner while splitting GCV-0075: it is not established what, if anything, this platform can emit as telemetry through Crossplane's own surfaces. The answer decides whether fleet-level health is available at all, and it is currently unknown rather than absent.

The constraint is the point of the task. Use only what Crossplane provides out of the box. No custom instrumentation of the composition function, no metric this repository invents, no exporter this repository ships.

What is already established, read from the pinned crossplane-runtime module rather than from documentation, so a lane does not spend a cycle rediscovering it. Seven managed-resource metric families exist under the crossplane subsystem. Four are histograms from the managed reconciler: first_time_to_reconcile_seconds, first_time_to_readiness_seconds, deletion_seconds, and drift_seconds, the last marked ALPHA upstream. Three are gauges from the state-metrics recorder: managed_resource_exists, managed_resource_ready counting children in Ready=True, and managed_resource_synced counting children in Synced=True.

The decisive limitation, and the reason this task is not simply a scrape config: every one of those seven carries exactly one label, gvk. There is no name, no namespace, no composite and no expiry label on any of them. So native telemetry can answer how many resources of a kind are unsynced, which is a real and currently missing fleet signal, but it cannot identify which object is stuck or how close a token is to its rotation window. Identifying the object is conditions work and belongs to GCV-0087, not here. Stating that boundary precisely is part of the deliverable so the gap is not rediscovered as a surprise.

Two things are unverified and must be settled from pinned artifacts rather than assumed. The state-metrics gauges come from a recorder a provider has to start, unlike the four histograms which the managed reconciler records once it is constructed, so whether the pinned provider actually starts it is an open question with a real answer in the package. And the Crossplane core chart is already installed here with metrics enabled, while the provider's own runtime config carries no scrape path and this repository contains no PodMonitor, ServiceMonitor or scrape annotation anywhere. The provider pod is where the vended kinds' metrics are emitted, so that is the gap that matters.

This repository is a portable public reference and must not assume a monitoring stack. Anything shipped here is inert by default and enabled by a consumer.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The managed-resource metric families the pinned crossplane-runtime emits are enumerated from the pinned module source with their labels, and the fleet questions they can and cannot answer are stated
- [ ] #2 Whether the pinned provider starts the state-metrics recorder is settled from the pinned provider package rather than assumed, and the answer is recorded either way
- [ ] #3 The provider metrics endpoint has a scrape path a consumer can enable, inert by default in this reference and requiring no monitoring stack to be present for the gate to pass
- [ ] #4 What an operator cannot learn from native telemetry alone is recorded, naming the per-object and expiry questions it cannot answer and where they are answered instead
- [ ] #5 No instrumentation is added to the composition function and no metric is invented by this repository
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
