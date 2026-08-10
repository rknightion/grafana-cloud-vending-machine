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
[Teams, basic roles, fixed roles, custom roles, and content ACLs](https://github.com/rknightion/grafana-cloud-vending-machine#teams-basic-roles-fixed-roles-custom-roles-and-content-acls)
in the project README.

## What happens if I delete a request from Git?

Argo CD prunes the Kubernetes composite and composed objects, but the external Grafana stack and
its credential documents are orphaned, not destroyed — deletion protection, the omitted `Delete`
management policy, `deleteOnDestroy: false` on rotating tokens, and `PushSecret`'s
`deletionPolicy: None` all combine to make this intentional. See
[Security → non-destructive lifecycle](security.md#non-destructive-lifecycle).

## Can I point this at an existing Grafana Cloud stack?

Not casually. Adoption is a change of controller ownership, not just a manifest deployment, and
needs a rehearsed inventory of the stack's current identity, service accounts, tokens, SSO
provider, plugins, and content before any resource is created under this platform's control. See
the adoption and migration guidance in the project
[README](https://github.com/rknightion/grafana-cloud-vending-machine#migration-and-adoption).

## Why does a new request take more than one reconciliation to become Ready?

Rotating-token resources (`StackServiceAccountRotatingToken`, `AccessPolicyRotatingToken`) are
deliberately rendered one reconciliation after Grafana reports the parent service-account or
access-policy ID, because that ID does not exist until the parent resource itself has been
created and observed. See [Architecture](architecture.md).

## How many Grafana Cloud resource kinds does this actually manage?

The pinned provider release exposes 111 namespaced external managed-resource kinds across 16
Grafana API families. This reference activates only the subset its Compositions emit, deliberately
leaving alerting rule groups, data sources, cloud integrations, SLOs, Synthetic Monitoring,
OnCall schedules, Fleet Management, k6, ML, Asserts, and Assistant as separately owned domains.
See [Architecture → baseline and optional resources](architecture.md#baseline-and-optional-resources).

## Where do I report a bug or ask a question?

Open an issue on
[GitHub](https://github.com/rknightion/grafana-cloud-vending-machine/issues). If it is a
reconciliation problem, include the output of `kubectl describe` on the affected composite and
its composed resources — see [Troubleshooting](troubleshooting.md).
