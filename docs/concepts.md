---
title: Core concepts
description: The Crossplane and GitOps concepts used by this Grafana Cloud vending-machine reference
---

# Core concepts

This page explains the small set of Kubernetes and Crossplane concepts used by this repository.
Read it before changing a request if Crossplane is new to you. It describes the control path in
this repository, not a replacement for the [Crossplane documentation](https://docs.crossplane.io/).

## The short version

A request is a Kubernetes custom resource. Git contains the request, Argo CD applies it, and
Crossplane turns it into managed resources for the Grafana provider. The provider compares those
resources with Grafana Cloud and works toward the requested state. ESO supplies the provider's
organization credential and publishes the generated stack credentials.

A request supplies input. The Composition function derives the managed-resource list from that
request, platform policy, and the observed state of resources it already owns.

## Terms used here

| Term | Meaning in this repository | Where to inspect it |
| --- | --- | --- |
| Custom resource | A Kubernetes object defined by this platform API, such as `GrafanaCloudStackRequest` | `platform/apis/stack-v1beta1.yaml` and [Request Schema Reference](reference/request-schema.md) |
| XRD | Crossplane's definition of a composite API. It supplies the custom-resource schema and binds the API to one Composition. | `platform/apis/` |
| Composition | The Crossplane rule that translates one composite resource into its desired composed resources. Every public API here uses Pipeline mode with one function step. | Each `platform/apis/*-v1beta1.yaml` |
| Composition function | The Go program named `function-grafana-vending`. It reads the observed request and composed resources, then returns the desired resources and status. | `platform/function/fn.go` |
| Managed resource | A Kubernetes object owned by Crossplane that represents one Grafana Cloud object, such as a Stack, service account, or folder. | Function renderer output and provider CRDs |
| Provider | The Grafana Crossplane provider. It observes and changes Grafana Cloud through its APIs. | `platform/provider/provider-grafana.yaml` |
| Reconciliation | The repeated control-loop work that compares observed state with desired state and makes progress toward the latter. It is asynchronous, not an API request that completes immediately. | [Architecture](architecture.md#reconciliation-and-out-of-band-changes) |
| Condition | A status signal such as `Ready` or `Synced`. It explains whether Crossplane has accepted the desired state and whether the provider reports the resource ready. | [Request Schema Reference](reference/request-schema.md#status-conditions) |

## From Git to a ready request

1. An author copies an inert catalog example to one new directory under `enabled/`, replaces every placeholder, and commits it. The supplied ApplicationSet watches only `enabled/*`.
2. Argo CD creates one Application for that directory and applies the request resource to the target namespace.
3. The XRD schema and any applicable admission policy validate the request. A denied request has not reached the provider.
4. Crossplane runs the bound Composition. Its function receives the request, the composed resources Crossplane already observes, and the platform-owned Composition input. It returns a complete desired set of managed resources.
5. The Grafana provider reconciles each managed resource with Grafana Cloud. Some children wait for an observed parent identity, so a new request normally needs more than one reconciliation before it can become ready.

The [request lifecycle figure](diagrams/composition-reconciliation.html) shows those stages and
the point at which a request can legitimately wait.

## Desired state, observed state, and ownership

Desired state is what the function asks Crossplane to maintain. It comes from the request plus
platform-owned configuration. Observed state is what Crossplane has already read from the
Kubernetes resources it manages, including provider-assigned IDs and status. The function needs
both: it cannot safely create a child that requires an ID until its parent has reported that ID.

Argo CD owns the request in Git. Crossplane owns the managed resources it composes from that
request. ESO owns the movement of credential material into and out of Kubernetes. Do not put a
managed resource beside its request in Git, because that would give Argo CD and Crossplane two
writers for the same object. The [ownership figure](diagrams/gitops-ownership.html) shows the
boundary in full.

## What a successful render does and does not prove

Rendering proves the function produced Kubernetes objects that match its tested contract. It does
not prove that an external Grafana Cloud API call completed. The repository's tests deliberately
do not contact a live stack or cluster. A provider `Ready` condition in an environment you control
is the signal that follows successful rendering; see [Getting started](getting-started.md#6-observe-reconciliation).

## Where to go next

- [Getting started](getting-started.md) for the first copy-edit-review-commit request.
- [Installation](installation.md) for the controllers, provider, and ESO prerequisites.
- [Configuration](configuration.md) for the platform-owned organization registry and profiles.
- [Architecture](architecture.md) for reconciliation, lifecycle, and ownership detail.
- [Request Schema Reference](reference/request-schema.md) for fields, admission rules, and status.
