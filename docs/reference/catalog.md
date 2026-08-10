---
title: Catalog Reference
description: Every example under examples/catalog, what it demonstrates, and what an adopter must change before use
---

# Catalog Reference

Nothing in `examples/catalog` is live. Each directory demonstrates one ownership decision and
uses reserved example values. The supplied `ApplicationSet` watches only the repository's
top-level `enabled/` directory, which is empty by default — see
[Getting started](../getting-started.md) for the copy-edit-review-commit path to using one of
these.

## Catalog directories

| Directory | What it demonstrates | What an adopter changes |
| --- | --- | --- |
| [minimal](https://github.com/rknightion/grafana-cloud-vending-machine/tree/main/examples/catalog/minimal) | Safe stack baseline, rotating credentials, create-only content, no SSO | Slug, region, usage, API group, secret backend |
| [comprehensive](https://github.com/rknightion/grafana-cloud-vending-machine/tree/main/examples/catalog/comprehensive) | Every public API kind: enforced content and OAuth SSO, report, plugin, incident relay, compact custom-role binding, direct and synchronized Team access, multiple custom/fixed roles, preferences, and folder/dashboard ACLs | All profile names, endpoints, recipients, identities, verified fixed-role UIDs, plugins, role actions/scopes, and ACL targets |
| [sso-create-only](https://github.com/rknightion/grafana-cloud-vending-machine/tree/main/examples/catalog/sso-create-only) | Platform initializes OAuth, then stack administrators own later SSO edits | Approved OAuth profile and handoff policy |
| [sso-azuread](https://github.com/rknightion/grafana-cloud-vending-machine/tree/main/examples/catalog/sso-azuread) | Azure AD OAuth profile selected from platform policy | Tenant/application IDs, group claims, role expression, client-secret path |
| [sso-saml](https://github.com/rknightion/grafana-cloud-vending-machine/tree/main/examples/catalog/sso-saml) | SAML metadata and role-value mapping | Metadata URL, attributes, signing requirements, role values |
| [access-and-rbac](https://github.com/rknightion/grafana-cloud-vending-machine/tree/main/examples/catalog/access-and-rbac) | Direct and directory Team membership, preferences, custom/fixed roles, folder/dashboard ACLs | Team/group names, verified role UIDs, actions/scopes, ACL targets |

Each directory's own `README.md` explains what its manifests own, the expected reconciliation
behaviour, and every value that must be replaced.

The `comprehensive` directory is a renderable Kustomize base. It intentionally contains every
current public API kind so it can be used for schema validation, platform evaluation, and
consumer overlays. It is not a claim that every feature should be enabled for every stack — a
production catalog normally offers a few reviewed profiles (such as `standard`, `regulated`, and
`administrator-owned-SSO`) rather than exposing raw provider fields to request authors.

## Enabling an example

Install and verify Crossplane, the Grafana provider, the vending function, ESO, the external
secret store, and the organization `ProviderConfig` first (see [Installation](../installation.md)).
Then:

1. Copy one catalog directory to a uniquely named subdirectory of top-level `enabled/`.
2. Replace `platform.example.org`, all `replacewithunique...` values, the example region, and
   environment-specific profile or identity values.
3. Remove optional resources and fields whose prerequisites or entitlements are not available.
4. Render and review the directory before committing it.
5. Commit it to the Git repository watched by the `ApplicationSet`, or apply it directly only for
   a disposable evaluation.

```bash
cp -R examples/catalog/minimal enabled/my-stack
kubectl apply --dry-run=server -k enabled/my-stack
kubectl apply -k enabled/my-stack
```

Grafana Cloud stack slugs are globally unique. `metadata.name` and `spec.slug` must remain
identical. A production deployment should keep real enabled requests in a private GitOps
repository rather than publishing stack identities, recipients, users, groups, or environment
profile names in a public fork — see
[Configuration → Public base, private environment overlay](../configuration.md#public-base-private-environment-overlay).

Deleting an enabled request prunes its Kubernetes objects, but the reference intentionally omits
provider `Delete` permission and enables stack deletion protection. This orphans external
resources; it is not a complete decommission workflow — see the decommission runbook in the
project [README](https://github.com/rknightion/grafana-cloud-vending-machine#decommission-runbook).
