---
title: FAQ
description: Short answers to recurring questions about the Grafana Cloud vending machine
---

# FAQ

## Does installing this create any Grafana Cloud stacks?

No. The top-level `enabled/` directory starts empty and nothing under `examples/catalog` is
applied by the supplied `ApplicationSet`. A user must copy an example, edit every placeholder,
review the rendered output, and commit it before a stack is created. See
[Getting started](getting-started.md).

## Can I use a secret backend other than AWS Secrets Manager?

Yes, in principle. The composition function only emits `SecretStore` references — it does not
hard-code AWS. `deploy/aws` is the example integration layer, and can be replaced with the
equivalent tooling for another ESO provider as long as it supports `ExternalSecret` and
`PushSecret` with the required structured-value behaviour. See [Secrets](secrets.md).

## Do I have to use Argo CD?

No. `platform/` and `deploy/aws` can be applied directly with `kubectl apply -k` — see
[Installation → Direct installation](installation.md#direct-installation). Argo CD is the
GitOps layer this reference ships an example integration for, not a hard dependency of the
Crossplane/ESO core.

## Can I rewrite the built-in Viewer, Editor, and Admin roles?

No. The current Grafana provider offers no resource that rewrites the global definitions of the
built-in basic roles. The supported customization points are SSO basic-role mapping, fixed/custom
role assignments, and per-folder or per-dashboard permissions — see
[Teams, roles, and content ACLs](reference/request-schema.md#teams-roles-and-content-acls)
in the Request Schema Reference.

## What happens if I delete a request from Git?

Unless the request was deliberately armed, `spec.lifecycle.externalResources` is `Retain` and
Argo CD prunes the Kubernetes composite and composed objects while the external Grafana stack and
credential documents are orphaned, not destroyed. Stack-local Grafana content remains
retain/orphan because deleting the Stack destroys it. `Delete` is accepted only for an exact
request namespace/name/UID/profile tuple in platform-owned `deletionAuthorizations`, which is empty by default.

Decommissioning has three reviewed stages: first set `externalResources: Delete` and wait for
`status.deletionArmed=true` followed by `status.deletionReady=true` (observed Stack
`deleteProtection=false`, deletion-managed rotating tokens, plus finalized and currently synced
credential PushSecrets); then remove dependent access claims and merge/sync until their
Kubernetes objects and finalizers are gone while the Stack still exists; finally remove the request.
The armed path deletes only the Stack, administrator service account/token, telemetry access
policy/token, and administrator/telemetry credential documents. AWS Secrets Manager uses a 30-day
recovery window by default for deleted `PushSecret` documents, and other backends must be checked
for Delete support. See [Security → external-resource lifecycle](security.md#external-resource-lifecycle).

The request's `spec.organization` and `spec.usage` are immutable and must be allowed by the
platform-owned organization registry; generated documents use
`{outputSecretPrefix}/{organization}/{usage}/{slug}`.

## Can I point this at an existing Grafana Cloud stack?

Not casually. Adoption is a change of controller ownership, not just a manifest deployment, and
needs a rehearsed inventory of the stack's current identity, service accounts, tokens, SSO
provider, plugins, and content before any resource is created under this platform's control. See
the adoption and migration guidance in the project
[Migration and adoption](migration-1.0.md).

## Why does a new request take more than one reconciliation to become Ready?

Rotating-token resources (`StackServiceAccountRotatingToken`, `AccessPolicyRotatingToken`) are
deliberately rendered one reconciliation after Grafana reports the parent service-account or
access-policy ID, because that ID does not exist until the parent resource itself has been
created and observed. See [Architecture](architecture.md).

## How many Grafana Cloud resource kinds does this actually manage?

The pinned provider exposes 121 namespaced external managed-resource kinds across 17 families,
plus 50 observe-only kinds. This reference activates only the subset its Compositions emit. It now
has explicit modules for inventory, Fleet pipelines, alerting, Agent Observability, Assistant
governance, datasource access, Git provisioning repositories, product activation, golden SLOs,
bounded k6 and Synthetic Monitoring, and promotion ladders. Cloud integrations, OnCall schedules,
ML and Asserts remain separately owned. See [Governance](governance.md). See [Architecture → baseline and optional resources](architecture.md#baseline-and-optional-resources).

## Does this manage Adaptive Metrics or other adaptive products?

No. Adaptive Metrics, Logs, Traces, and Profiles are deliberately out of scope, and the stack API
has no hidden adaptive toggle. Use the Grafana Cloud UI and the applicable ticket-based route. The
decision is inexpensive to reverse because no current request or composition depends on it.

## Where do I report a bug or ask a question?

Open an issue on
[GitHub](https://github.com/rknightion/grafana-cloud-vending-machine/issues). If it is a
reconciliation problem, include the output of `kubectl describe` on the affected composite and
its composed resources — see [Troubleshooting](troubleshooting.md).
