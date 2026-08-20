# Examples

Everything under `catalog/` is inert reference material. The supplied Argo CD ApplicationSet watches only the repository's top-level `enabled/` directory, which is empty by default. Cloning or installing the platform cannot create a Grafana Cloud stack until an edited request is deliberately placed there.

## Catalog

| Example | Start here when you need | Important prerequisites |
| --- | --- | --- |
| [minimal](catalog/minimal/) | A safe baseline with rotating administrator and telemetry credentials | Organization ProviderConfig, external secret store, unique slug, and real region |
| [comprehensive](catalog/comprehensive/) | One renderable example covering every public vending API kind | Approved SSO and incident profiles, entitlements, identity inventory, and reviewed RBAC identifiers |
| [access-and-rbac](catalog/access-and-rbac/) | Teams, Team Sync, direct membership, fixed/custom roles, and content ACLs | Existing users, IdP groups, fixed-role inventory, and approved role scopes |
| [sso-create-only](catalog/sso-create-only/) | Initializing OAuth/OIDC before handing later changes to stack administrators | Approved OAuth profile and client-secret ExternalSecret |
| [sso-azuread](catalog/sso-azuread/) | Enforced Azure AD OAuth with group-based role mapping | Tenant application, group claims, and client-secret ExternalSecret |
| [sso-saml](catalog/sso-saml/) | Enforced SAML with metadata and role-value mapping | IdP metadata, matching attributes, and a tested administrator login path |

Each directory README explains what its manifests own, the expected reconciliation behavior, and every value that must be replaced.

## Enable an example

Install and verify Crossplane, the Grafana provider, the vending function, External Secrets Operator, the external secret store, and the organization ProviderConfig first. Then:

1. Copy one catalog directory to a uniquely named subdirectory of top-level `enabled/`.
2. Replace `platform.example.org`, all `replacewithunique...` values, the example region, and environment-specific profile or identity values. Keep `spec.usage` in the platform-owned vocabulary (`development` or `production` in this reference); it is immutable and forms part of the generated path `{outputSecretPrefix}/{region}/{usage}/{slug}`.
3. Remove optional resources and fields whose prerequisites or entitlements are not available.
4. Render and review the directory before committing it.
5. Commit it to the Git repository watched by the ApplicationSet, or apply it directly only for a disposable evaluation.

For example:

~~~bash
cp -R examples/catalog/minimal enabled/my-stack
kubectl apply --dry-run=server -k enabled/my-stack
kubectl apply -k enabled/my-stack
~~~

Grafana Cloud stack slugs are globally unique. `metadata.name` and `spec.slug` must remain identical. A production deployment should keep real enabled requests in a private GitOps repository rather than publishing stack identities, recipients, users, groups, or environment profile names in a public fork.

Deleting an enabled request uses `spec.lifecycle.externalResources: Retain` by default and prunes
Kubernetes objects while orphaning external resources. `Delete` requires an exact request
namespace/name/UID/profile tuple in platform-owned `deletionAuthorizations`, empty by default. The decommission
has three reviewed stages: arm Delete and wait for `status.deletionReady=true`; remove dependent
access claims and merge/sync until their Kubernetes objects and finalizers are gone while the Stack
still exists; then remove the request. See the [decommission runbook](../README.md#decommission-runbook).
