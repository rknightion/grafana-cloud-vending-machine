---
title: Native Crossplane telemetry
description: The managed-resource metrics Crossplane and the Grafana provider expose without custom instrumentation
---

# Native Crossplane telemetry

The Crossplane runtime and the pinned Grafana provider already expose managed-resource telemetry.
This page records the exact signal contract and the boundary of what that telemetry can tell an
operator. It does not add instrumentation to the composition function and it does not install a
monitoring stack.

## Metric families

The function module pins `github.com/crossplane/crossplane-runtime/v2 v2.4.0`. The runtime source
constructs the four managed reconciler histograms in
[`pkg/reconciler/managed/metrics.go`](https://github.com/crossplane/crossplane-runtime/blob/v2.4.0/pkg/reconciler/managed/metrics.go#L31-L85)
and the three state-recorder gauges in
[`pkg/statemetrics/mr_state_metrics.go`](https://github.com/crossplane/crossplane-runtime/blob/v2.4.0/pkg/statemetrics/mr_state_metrics.go#L35-L59).
All seven use the Prometheus `crossplane` subsystem and carry exactly one label: `gvk`.

| Family | Type | Native meaning | Fleet question it answers |
| --- | --- | --- | --- |
| `crossplane_managed_resource_first_time_to_reconcile_seconds` | Histogram | Time from creation until the controller first detects a resource whose initial observation is captured as `Synced=Unknown` | Are resources of a kind taking longer than usual to enter reconciliation? |
| `crossplane_managed_resource_first_time_to_readiness_seconds` | Histogram | Time from creation until the first `Ready=True` condition for a resource whose initial observation is captured as `Synced=Unknown` | Are resources of a kind taking longer than usual to become ready? |
| `crossplane_managed_resource_deletion_seconds` | Histogram | Time from deletion timestamp until deletion completes | Are deletions of a kind taking longer than usual? |
| `crossplane_managed_resource_drift_seconds` | Histogram, `ALPHA` upstream | Time since the previous successful reconcile when the resource is found out of sync; provider restart is excluded | Is drift detection for a kind delayed? |
| `crossplane_managed_resource_exists` | Gauge | Number of managed resources of a kind | How many resources of a kind exist? |
| `crossplane_managed_resource_ready` | Gauge | Number of managed resources of a kind in `Ready=True` | How many resources of a kind are ready? |
| `crossplane_managed_resource_synced` | Gauge | Number of managed resources of a kind in `Synced=True` | How many resources of a kind are synced? |

For an activated kind, the difference between `crossplane_managed_resource_exists` and
`crossplane_managed_resource_synced` is a fleet-level count of resources that are not currently
`Synced=True`. The label value is the group, version, and kind string; it is not an object identity.

## The provider starts state metrics

The installed provider package is Grafana Crossplane provider `v2.14.0`, pinned by digest in
[`platform/provider/provider-grafana.yaml`](../platform/provider/provider-grafana.yaml). In the
`v2.14.0` source, [`cmd/provider/main.go`](https://github.com/grafana/crossplane-provider-grafana/blob/v2.14.0/cmd/provider/main.go)
creates both `managed.NewMRMetricRecorder()` and `statemetrics.NewMRStateMetrics()`, registers both
with the controller-runtime metrics registry, and passes both through `MetricOptions`. A generated
managed controller then creates `statemetrics.NewMRStateRecorder(...)` and adds it to the manager
when those options are present. The source path is visible, for example, in
[`accesspolicyrotatingtoken/zz_controller.go`](https://github.com/grafana/crossplane-provider-grafana/blob/v2.14.0/internal/controller/namespaced/cloud/accesspolicyrotatingtoken/zz_controller.go#L75-L80).

Therefore the three state-recorder families are part of the provider's native path for activated
managed-resource kinds. SafeStart still controls when each activated controller is registered, so a
kind that is not activated has no resource series to report. This is a pinned-source conclusion;
this repository does not contact a live provider or cluster to exercise it.

## Provider scrape path

The Crossplane Helm chart is pinned to `2.3.4` in
[`deploy/argocd/crossplane.yaml`](../deploy/argocd/crossplane.yaml). Its `metrics.enabled: true`
value adds `/metrics`, port `8080`, and `prometheus.io/*` scrape annotations to the Crossplane and
RBAC Manager pods, and exposes container port `8080` on those two pods. It does not modify the
provider package DeploymentRuntimeConfig or discover the provider pod.

The provider has its own controller-runtime metrics server. The provider source leaves the
manager's metrics options at their defaults; its pinned controller-runtime dependency
([`pkg/metrics/server/server.go`](https://github.com/kubernetes-sigs/controller-runtime/blob/v0.22.0/pkg/metrics/server/server.go))
defaults that server to HTTP `:8080` with the `/metrics` endpoint. The optional
[`deploy/monitoring`](../deploy/monitoring/) Kustomize overlay patches the provider
`DeploymentRuntimeConfig` with these pod annotations:

```yaml
prometheus.io/path: /metrics
prometheus.io/port: "8080"
prometheus.io/scrape: "true"
```

The overlay includes the normal `platform/` base and is not referenced by the default platform
installation. A consumer can select `deploy/monitoring` as the platform Kustomize root, or copy
its patch into an existing private overlay that already includes `platform/`. The default tree
therefore remains inert, and `just check` does not require a Prometheus, PodMonitor, or ServiceMonitor
CRD. No monitoring stack is bundled here; the consumer supplies the scraper and its security
policy.

## What native telemetry cannot answer

The `gvk` label intentionally aggregates across objects. Native telemetry cannot identify the
stuck object's name, namespace, composite owner, stack, or provider configuration. It also has no
token expiry timestamp, rotation window, or remaining lifetime label. A nonzero unsynced count can
trigger fleet-level investigation, but it cannot say which token needs attention or how close that
token is to expiry.

The per-object stuck-token question is answered by the credential-health conditions on the
composite from GCV-0087. Conditions and object inspection provide the identity and reason; native
Crossplane metrics provide the aggregate kind-level signal.

## Scope

This reference adds no composition-function instrumentation, exporter, custom metric, PodMonitor,
ServiceMonitor, or monitoring stack. The only new deployment surface is the consumer-selected pod
annotation patch described above.
