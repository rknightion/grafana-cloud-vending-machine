# Grafana Cloud vending machine with Crossplane

This repository is a portable reference architecture for vending Grafana Cloud stacks through a
small, declarative Kubernetes API. Argo CD owns the objects in Git, Crossplane renders and
reconciles Grafana Cloud resources, and External Secrets Operator (ESO) moves credentials between
Kubernetes and an external secret store. It supports Grafana Cloud only; self-managed Grafana is
out of scope.

One vending-machine installation can serve several registered Grafana Cloud organizations. Each
stack selects one immutable organization, region, usage, and platform profile. The baseline gives a
stack rotating administrator and telemetry credentials, deletion protection, starter content, SSO,
reports, plugins, incident relay resources, teams, roles, and content access. The top-level
`enabled/` directory starts empty, so copying or installing this repository cannot create a stack
until a user copies an inert catalog example, edits it, reviews the render, and commits the request.

> `platform.example.org` is a documentation placeholder. Replace it with a domain your organization
> controls in the XRDs, Compositions, examples, Argo CD health configuration, and requests before
> adopting this API.

## Quickstart

The next release is a breaking one. `GrafanaProvisioningRepository` requests written against 1.x must be restructured; see [Migrating to 2.0](docs/migration-1.0.md#migrating-to-20). No other request kind changes.

Read [Getting started](docs/getting-started.md) for prerequisites and the copy-edit-review-commit
path to a first request. Use [Installation](docs/installation.md) first when the controllers,
provider, secret store, and organization `ProviderConfig` objects are not already healthy.

## Documentation map

| Page | Covers |
| --- | --- |
| [Documentation home](docs/index.md) | Who this is for, the request-to-stack flow, baseline capabilities, and the full reading map |
| [Getting started](docs/getting-started.md) | Prerequisites and the first request |
| [Installation](docs/installation.md) | Direct and Argo CD installation, pins, signatures, releases, and upgrades |
| [Configuration](docs/configuration.md) | Platform policy, profiles, and the organization registry |
| [Architecture](docs/architecture.md) | Composition model, renderer registry, resource graph, reconciliation, and provider ownership |
| [Request schema](docs/reference/request-schema.md) | Per-API fields, CompositeResourceDefinitions, short names, admission guards, and status |
| [Catalog](docs/reference/catalog.md) | Every inert catalog example and its adaptation boundary |
| [Secrets](docs/secrets.md) | Organization credentials, rotating tokens, ESO, and AWS permissions |
| [SSO](docs/sso.md) | OAuth and SAML profiles and reconciliation modes |
| [Security](docs/security.md) | Credential boundaries, token models, RBAC, and network restrictions |
| [Governance](docs/governance.md) | Token ceilings, bounded product APIs, retention, tenancy, expiry, SCIM scope, and reviewed decommission |
| [Migration and adoption](docs/migration-1.0.md) | Breaking schema changes per release boundary, and inventory-first, non-destructive adoption |
| [Troubleshooting](docs/troubleshooting.md) | Recurring reconciliation and validation failures |
| [FAQ](docs/faq.md) | Short answers to common questions |

Catalog directories are documented in [examples/README.md](examples/README.md). The project is
available under the [Apache License 2.0](LICENSE).
