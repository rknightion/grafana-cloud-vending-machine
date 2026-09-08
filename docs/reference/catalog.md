---
title: Catalog Reference
description: Every example under examples/catalog, what it demonstrates, and what an adopter must change before use
---

# Catalog Reference

Nothing in `examples/catalog` is live. The supplied ApplicationSet watches only top-level
`enabled/*`, which starts empty. Each catalog directory is a renderable Kustomize base with a
README describing its ownership boundary.

## Catalog directories

| Directory | What it demonstrates | What an adopter changes or verifies |
| --- | --- | --- |
| [minimal](../../examples/catalog/minimal/) | Safe stack baseline, rotating credentials, create-only content, no SSO | Organization, slug, region, permitted usage, API group, and secret backend |
| [comprehensive](../../examples/catalog/comprehensive/) | The original stack and access API surface in one renderable base | Approved profiles, entitlements, identities, role UIDs/actions, and ACL targets |
| [access-and-rbac](../../examples/catalog/access-and-rbac/) | Teams, Team Sync, direct membership, roles, and content ACLs | Existing users, IdP groups, reviewed role scopes, and targets |
| [stack-inventory](../../examples/catalog/stack-inventory/) | Observe-only declared/managed/unmanaged inventory for migration and adoption | A Ready stack and healthy per-stack ProviderConfig; exact declared selectors |
| [fleet-pipelines](../../examples/catalog/fleet-pipelines/) | Selection of a platform-owned Fleet baseline pipeline profile | Fleet entitlement, approved profile, and stack reference |
| [alerting-bundle](../../examples/catalog/alerting-bundle/) | Stack-scoped alert rules, contact points, mute timings, templates, and inhibitions | Folder UID, recipient, rules, and the deliberate provenance mode |
| [agent-observability](../../examples/catalog/agent-observability/) | Separate platform guards and workload-owned evaluation resources | Plugin/permission prerequisites and reviewed workload policy |
| [assistant-governance](../../examples/catalog/assistant-governance/) | Terms-gated Assistant rules and MCP allow-list | Accepted terms, reviewed platform profiles, endpoint, and Secret-backed headers |
| [datasource-access](../../examples/catalog/datasource-access/) | One datasource's authoritative team Query grants and aggregated LBAC tree | Observed team UID and numeric ID, basic-auth connection Secret, entitlement, and rules |
| [observability-products](../../examples/catalog/observability-products/) | Stack-request product activation toggles | Organization, stack identity, and product-specific configuration outside this API |
| [provisioning-repository](../../examples/catalog/provisioning-repository/) | Preview Git-provisioned dashboard subtree | Existing Grafana Connection, Git URL/branch/path, and exclusive subtree ownership |
| [sso-create-only](../../examples/catalog/sso-create-only/) | OAuth initialization followed by administrator ownership | Approved OAuth profile and handoff policy |
| [sso-azuread](../../examples/catalog/sso-azuread/) | Enforced Azure AD OAuth with group role mapping | Tenant/application values, group claims, and client-secret path |
| [sso-saml](../../examples/catalog/sso-saml/) | Enforced SAML metadata and role mapping | Metadata, attributes, signing requirements, and a tested administrator login path |

Each directory's own `README.md` explains what its manifests own, the expected reconciliation
behaviour, and every value that must be replaced.

The `comprehensive` directory remains a renderable base for the original stack/access surface; the
specialist modules intentionally have their own catalog directories because each has a different
prerequisite and lifecycle. A catalog example demonstrates an activation or ownership choice, not
a production configuration.

## Enabling an example

Install Crossplane, the Grafana provider and vending function, ESO, the secret store, and the
ProviderConfigs for every registered organization first. Then:

1. Copy one catalog directory to a uniquely named subdirectory of top-level `enabled/`.
2. Replace `platform.example.org`, all placeholders, the stack organization, region, and usage.
   The organization registry must recognize the organization and permit both the selected region
   and usage.
3. Remove optional resources whose entitlement, connection, plugin, profile, or other prerequisite
   is unavailable.
4. Render and review the directory before committing it.

```bash
cp -R examples/catalog/minimal enabled/my-stack
kubectl apply --dry-run=server -k enabled/my-stack
kubectl apply -k enabled/my-stack
```

Grafana Cloud stack slugs are globally unique. `metadata.name` and `spec.slug` must remain
identical. A stack is in one organization, but the platform installation may serve several
registry entries. Generated credentials use
`{outputSecretPrefix}/{organization}/{usage}/{slug}` so secret-store IAM can be scoped to an
organization. Keep real enabled requests and organization credential Secrets in a private GitOps
repository rather than publishing stack identities, recipients, users, groups, or environment
profile names in a public fork.

Deleting an enabled request uses `spec.lifecycle.externalResources: Retain` by default: it prunes
Kubernetes objects and orphans external resources. `Delete` requires an exact request
namespace/name/UID/profile entry in platform-owned `deletionAuthorizations` (empty by default). The decommission has three
reviewed stages: arm Delete and reach `status.deletionReady=true`; remove dependent access claims
and merge/sync until their Kubernetes objects and finalizers are gone while the Stack still exists;
then remove the request. See the decommission runbook in the project
[README](https://github.com/rknightion/grafana-cloud-vending-machine#decommission-runbook).

## Governance catalog bases

- `golden-slo`: platform ratio objective and a stack using its usage profile.
- `k6-project`: project with platform-capped limits and allowed load zones.
- `synthetic-monitoring`: explicit consuming-team checks and independent installation verification.
- `promotion-ladder`: immutable rung identities with explicit branch-promotion direction.

Each is an inert Kustomize base rendered by the repository gate. See [Governance](../governance.md) and each catalog README for prerequisites.
