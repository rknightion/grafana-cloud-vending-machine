# Examples

Everything under `catalog/` is inert reference material. The supplied Argo CD ApplicationSet watches only the repository's top-level `enabled/` directory, which is empty by default. Cloning or installing the platform cannot create a Grafana Cloud stack until an edited request is deliberately placed there.

## Catalog

| Example | Start here when you need | Important prerequisites |
| --- | --- | --- |
| [minimal](catalog/minimal/) | A safe baseline with rotating administrator and telemetry credentials | Registered organization ProviderConfig, external secret store, unique slug, and permitted region |
| [comprehensive](catalog/comprehensive/) | The original stack and access API surface in one renderable base | Approved SSO and incident profiles, entitlements, identity inventory, and reviewed RBAC identifiers |
| [access-and-rbac](catalog/access-and-rbac/) | Teams, Team Sync, direct membership, fixed/custom roles, and content ACLs | Existing users, IdP groups, fixed-role inventory, and approved role scopes |
| [stack-inventory](catalog/stack-inventory/) | Observe-only inventory for migration, drift detection, and adoption review | An existing Ready stack and its healthy per-stack ProviderConfig |
| [fleet-pipelines](catalog/fleet-pipelines/) | A platform-owned Fleet Management pipeline baseline | Fleet entitlement and an approved platform profile |
| [alerting-bundle](catalog/alerting-bundle/) | Stack-scoped alerting with an explicit UI-provenance choice | Reviewed rules, contact points, folder UID, and provenance mode |
| [agent-observability](catalog/agent-observability/) | Separate platform guards and workload-owned evaluation resources | Plugin/permission prerequisites and reviewed workload policy |
| [assistant-governance](catalog/assistant-governance/) | Terms-gated Assistant rules and MCP server allow-list | Accepted terms, platform rule profile, endpoint, and Secret-backed headers |
| [datasource-access](catalog/datasource-access/) | One datasource's Query grants and aggregated LBAC rules | Observed team UID/numeric ID, basic-auth connection, entitlement, and rules |
| [observability-products](catalog/observability-products/) | Application, Kubernetes, and Database Observability activation toggles | Product-specific configuration in its Helm/onboarding surface |
| [provisioning-repository](catalog/provisioning-repository/) | Preview Git-backed dashboard subtree provisioning | Existing Grafana Connection and exclusive content ownership |
| [sso-create-only](catalog/sso-create-only/) | Initializing OAuth/OIDC before handing later changes to stack administrators | Approved OAuth profile and client-secret ExternalSecret |
| [sso-azuread](catalog/sso-azuread/) | Enforced Azure AD OAuth with group-based role mapping | Tenant application, group claims, and client-secret ExternalSecret |
| [sso-saml](catalog/sso-saml/) | Enforced SAML with metadata and role-value mapping | IdP metadata, matching attributes, and a tested administrator login path |

Each directory README explains what its manifests own, the expected reconciliation behavior, and every value that must be replaced.

## Enable an example

Install and verify Crossplane, the Grafana provider, the vending function, External Secrets Operator, the external secret store, and the ProviderConfig for every registered organization in the example's target namespace first. Repeat the namespaced SecretStore, credential Secret, and same-named ProviderConfigs before placing requests in another namespace. Then:

1. Copy one catalog directory to a uniquely named subdirectory of top-level `enabled/`.
2. Replace `platform.example.org`, all `replacewithunique...` values, organization, region, and environment-specific profile or identity values. The immutable organization and usage must be allowed by that organization's platform registry entry, and the selected region must appear in the same entry's allowed regions. Generated credentials use `{outputSecretPrefix}/{organization}/{usage}/{slug}`.
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
still exists; then remove the request. See the [decommission runbook](../docs/governance.md#decommission-runbook).
