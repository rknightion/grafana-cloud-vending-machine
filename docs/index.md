---
title: grafana-cloud-vending-machine — declarative Grafana Cloud stack vending
description: A portable reference architecture that vends Grafana Cloud stacks through a declarative Kubernetes API, using Argo CD, Crossplane, and External Secrets Operator.
image: assets/social-card.png
---

# grafana-cloud-vending-machine

**A reference architecture for vending Grafana Cloud stacks through a small, declarative API.**
Argo CD owns what is in Git, Crossplane continuously reconciles Grafana Cloud, and External
Secrets Operator (ESO) moves credentials between Kubernetes and an external secret store.

It is deliberately more than a minimal stack example. The baseline includes rotating
administrator and telemetry credentials, deletion protection, stack-local provider
configuration, starter content, configurable drift behaviour, OAuth and SAML SSO, reports,
plugins, incident relay resources, teams, basic-role ACLs, fixed-role assignments, custom roles,
and role assignments.

The source and issue tracker live on
**[GitHub](https://github.com/rknightion/grafana-cloud-vending-machine)**.

!!! warning "`platform.example.org` is a documentation placeholder"
    Every XRD, Composition, example, and page in this documentation uses the API group
    `platform.example.org`. It is not a production API group — it is not resolvable and it is not
    meant to be used as-is. Replace every occurrence with a domain your organization controls
    before adopting this API. See [Getting started](getting-started.md) and
    [Configuration](configuration.md) for where this matters most.

## Who this is for

Platform teams that already run Argo CD and Crossplane and want to offer Grafana Cloud stacks as
a self-service, GitOps-native product — with rotating credentials, safe defaults, and an explicit
non-destructive lifecycle — rather than hand-running Terraform or the Grafana Cloud API per
request.

## Request-to-stack flow

A request is a `GrafanaCloudStackRequest` custom resource committed to a directory under the
repository's top-level `enabled/`. From there:

1. **Argo CD** watches `enabled/*` through an `ApplicationSet` and creates one Argo CD
   `Application` per directory, applying the request object to the cluster.
2. **Crossplane**, through a Composition pipeline backed by a Go composition function, renders
   the request into a full set of managed resources: a `Stack`, a rotating administrator service
   account and token, an optional rotating telemetry access policy, baseline folders and
   dashboards, and any SSO, plugin, report, incident, team, or access-control resources the
   request enables.
3. **Crossplane's Grafana provider** continuously reconciles those managed resources against
   Grafana Cloud — creating what is missing and repairing drift on anything the request's
   reconciliation mode marks as enforced.
4. **External Secrets Operator** moves the organization credential from an external secret store
   into the cluster so the provider can authenticate, and moves the stack's generated
   administrator and telemetry tokens back out to the external store as structured per-stack
   documents.

Nothing under `examples/catalog` is applied by the supplied `ApplicationSet`, and the top-level
`enabled/` directory starts empty. A user must copy a catalog example, edit every placeholder,
review the rendered output, and commit it before a stack is created. This is a deliberate safety
property: cloning or installing this platform cannot, by itself, create a Grafana Cloud stack.

## What the baseline includes

Every enabled request gets, by default or by opt-in field:

| Capability | What it is |
| --- | --- |
| Rotating administrator credential | A `StackServiceAccount` with the Admin role and a `StackServiceAccountRotatingToken`, rotated automatically rather than issued once as a static token |
| Rotating telemetry credential | A stack-realm `AccessPolicy` scoped to `stacks:read`, `metrics:write`, `logs:write`, `traces:write`, with its own rotating token, when `spec.telemetryAccess.enabled` is true |
| Deletion protection | The `Stack` sets `deleteProtection`, and every composed managed resource omits the `Delete` management policy — normal Git-driven pruning cannot destroy the external stack or its credentials |
| Stack-local provider configuration | A namespaced Grafana `ProviderConfig` scoped to the one stack, built from the generated administrator credential |
| Starter content | Three baseline folder/dashboard pairs occupying the billing/usage, telemetry-endpoints, and stack-home slots, when `spec.baselineDashboards.enabled` is true |
| Configurable drift behaviour | Per-resource `enforced`/`createOnly`/`observeOnly`/`disabled` reconciliation modes that decide whether Crossplane repairs an administrator's UI edit or leaves it alone |
| OAuth and SAML SSO | Platform-defined SSO profiles a request selects by name — `github`, `gitlab`, `google`, `azuread`, `okta`, `generic_oauth`, or `saml` |
| Reports | An optional scheduled monthly usage report (PDF/CSV) |
| Plugins | An optional list of Grafana Cloud plugin installations |
| Incident relay resources | Optional OnCall outgoing webhooks and Alerting contact points that call a platform-owned relay |
| Teams | Directory-synchronized or directly-managed Grafana teams |
| Basic-role ACLs | Folder/dashboard permissions granted to the built-in Viewer/Editor/Admin basic roles |
| Fixed-role assignments | Existing Grafana-managed fixed roles assigned to a team by UID |
| Custom roles | Stack-local roles built from explicit action/scope permission pairs |
| Role assignments | Whole-set or item-level bindings of a role to a team |

## Reading further

| | |
|---|---|
| [Getting started](getting-started.md) | Prerequisites and the copy-edit-review-commit path to your first stack |
| [Installation](installation.md) | Bootstrapping Crossplane, ESO, and the platform components |
| [Configuration](configuration.md) | The request API fields, defaults, and what each does |
| [Architecture](architecture.md) | The three-controller split, reconciliation, and where state lives |
| [Secrets](secrets.md) | ESO wiring and the credential rotation model |
| [SSO](sso.md) | OAuth and SAML configuration |
| [Security](security.md) | Supply-chain controls, secret handling, and non-destructive lifecycle |
| [Troubleshooting](troubleshooting.md) | Common failure modes |
| [FAQ](faq.md) | Short answers to recurring questions |

## Project

grafana-cloud-vending-machine is open source under the Apache 2.0 licence. Issues and pull
requests are welcome on
[GitHub](https://github.com/rknightion/grafana-cloud-vending-machine).
