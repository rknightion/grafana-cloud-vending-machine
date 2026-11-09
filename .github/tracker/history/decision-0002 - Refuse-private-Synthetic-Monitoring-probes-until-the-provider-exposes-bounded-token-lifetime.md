---
id: decision-0002
title: >-
  Refuse private Synthetic Monitoring probes until the provider exposes bounded
  token lifetime
date: '2026-09-09 09:49'
status: accepted
---
## Context

The pinned Grafana Crossplane provider at v2.14.0 exposes a Synthetic Monitoring Probe CRD, but that resource has no token or token-lifetime field. Source evidence is the pinned provider schema represented by platform/provider/managed-kind-map.json and the refusal in platform/apis/synthetic-monitoring-v1beta1.yaml plus platform/function/syntheticmonitoring.go. A vended private probe would therefore issue a credential whose lifetime the shared composition token ceiling cannot bound.

## Decision

The repository owner decided on 2026-09-09 that private probes are refused as the shipped design. GrafanaSyntheticMonitoring admits no privateProbes field and renders no Probe or probe token. The refusal remains fail-closed until the pinned provider exposes a provider-enforced token lifetime that the composition can set within the existing ceiling.

## Consequences

Private targets must use probes provisioned outside this API. The public request, rendered children, status, examples, and published credentials contain no probe token. Every provider-pin bump must re-check the Probe schema and token lifecycle; the refusal can change only when the remote token lifetime is enforceable and its value can remain out of status and public examples.

## Alternatives

Expiring a Kubernetes Secret was rejected because deleting or expiring the local Secret does not expire the remote probe token. Keeping the feature as unfinished debt was rejected because the current fail-closed refusal is complete and intentional for the provider contract that is actually pinned.
