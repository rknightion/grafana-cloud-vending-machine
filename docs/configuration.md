---
title: Configuration
description: Platform policy, profiles, and the organization registry for the vending machine
---

# Configuration

!!! warning "`platform.example.org` is a documentation placeholder"
    Every request, XRD, and Composition in this repository uses the API group
    `platform.example.org`. It is not a production API group. Replace it everywhere in the XRDs,
    Compositions, examples, Argo CD health configuration, and your own request manifests before
    adopting this API in production. See [Request Schema Reference](reference/request-schema.md)
    for the per-API fields and admission rules.

This page describes the platform-owned policy boundary: registered organizations, profiles,
allowed values, and the public-base/private-environment split. For custom-resource fields and
ownership units, use the [Request Schema Reference](reference/request-schema.md).
## SSO patterns and ownership

The provider supports OAuth settings for github, gitlab, google, azuread, okta, and generic_oauth, plus SAML. This reference accepts those Grafana Cloud-relevant providers. The upstream schema also contains LDAP settings, but LDAP is a self-managed Grafana integration rather than a portable Grafana Cloud stack profile, so the function rejects it.

Platform owners define complete profiles in the Composition input. Request authors can select only a profile name and ownership mode. OAuth client secrets must be LocalSecretKeySelector references; literal client-secret fields are not copied by the function. SAML can use an IdP metadata URL, or certificateSecretRef and privateKeySecretRef when the chosen flow requires key material.

| Pattern | Important profile fields | Role behavior |
| --- | --- | --- |
| Generic OAuth/OIDC | authUrl, tokenUrl, apiUrl, clientId, scopes, claim paths | roleAttributePath maps claims to Viewer, Editor, Admin, or None |
| Azure AD | tenant-specific auth/token URLs, clientId, group claim path | group-aware roleAttributePath; account for group overage behavior in the IdP design |
| GitHub, GitLab, Google, or Okta | provider-specific organizations/domains/groups and client credentials | Use allowed groups/organizations as an admission gate, then map the resulting role |
| SAML | idpMetadataUrl, assertion attributes, roleValues fields, signature settings | IdP role values map to Grafana basic roles |

roleAttributeStrict=true rejects a login when no valid role can be derived. skipOrgRoleSync=true has the opposite ownership implication: Grafana stops updating the user's organization basic role from the IdP. Use it only when another reviewed process owns organization roles. allowAssignGrafanaAdmin is far more privileged than organization Admin and should remain false unless server-administrator assignment is explicitly required and supported.

The four reconciliation modes apply to every supported profile type:

- enforced keeps the selected settings in forProvider and repairs UI drift;
- createOnly puts the settings in initProvider and preserves later administrator changes;
- observeOnly supplies only providerName with Observe permission;
- disabled emits no SSO managed resource.

Keep providerName stable when switching ownership. A change from generic_oauth to saml is an identity-provider migration, not a routine mode toggle, and needs a tested login and rollback plan.


## Platform configuration

The Composition input (a `GrafanaVendingConfig` object embedded in
`platform/apis/stack-v1beta1.yaml`) is the platform-owned policy boundary. It controls:

| Field | Description |
| --- | --- |
| `organizations` | Registry entries keyed by organization name. Each supplies an organization ProviderConfig name, resolved in the request namespace, plus its allowed regions and usages. There is no fallback entry. |
| `allowedUsages` | Additional platform-wide usage allow-list. A request must pass this list as well as the selected organization's list; the reference values are `development` and `production`. |
| `outputSecretPrefix` | The external path prefix for generated per-stack documents. |
| `publicDashboardProfiles` | Platform-owned profiles allowed to retain the exact custom-role action `dashboards.public:write`; every other profile has that action stripped. Built-in Viewer/Editor/Admin roles are unchanged. |
| `deletionAuthorizations` | Platform-owned list of exact request `namespace`, `name`, Kubernetes `uid`, and immutable `profile` tuples authorized for `spec.lifecycle.externalResources: Delete`. It is empty by default; UID binding prevents an authorization from applying to a later request that reuses the same name. |
| `secretStoreRef` | Either a namespaced `SecretStore` or a `ClusterSecretStore`. |
| `fleetPipelineProfiles` | Required by `GrafanaFleetPipelines`. Each selectable profile supplies at least one matcher plus platform-owned `team`, `cost-centre`, and `environment` labels; the shipped `standard` profile is defined in `platform/apis/fleet-v1beta1.yaml`. Requests select a name and cannot supply pipeline contents or attribution values. |
| `ssoProfiles` | Approved OAuth or SAML settings and Secret references. |
| `incidentProfiles` | Approved relay URLs and authorization Secret references. |

Every organization-plane child uses the selected registry ProviderConfig in the request namespace.
The v2 ProviderConfig is namespaced, so every namespace allowed to contain requests must carry a
same-named credential Secret and ProviderConfig for each registry entry. Consumers select a
profile by name in `spec.sso.profile` or `spec.incidentIntegration.profile`.
They cannot supply an arbitrary identity endpoint, client secret, incident URL, or authorization
value directly in a stack request — see [SSO](sso.md).

The organization registry fails closed: an unknown organization, a region absent from its allowed
list, or a usage absent from its allowed list rejects the request. Generated documents use
`{outputSecretPrefix}/{organization}/{usage}/{slug}`, allowing secret-store IAM to be scoped by
organization. `spec.lifecycle.externalResources` defaults to `Retain`. `Delete` is accepted only
when the request's namespace, name, Kubernetes UID, and immutable `spec.profile` match an entry in
`deletionAuthorizations`; the list is empty by default, so a consumer cannot authorize deletion by
selecting a profile.

The API group is not a runtime setting. Replace every occurrence of `platform.example.org` in the
XRDs, Compositions, examples, tests, Argo CD health configuration, and documentation when making
a production fork.

## Public base, private environment overlay

The reusable implementation and a live environment have different publication boundaries:

| Public reference owns | Private environment owns |
| --- | --- |
| Provider and function packages at immutable digests | Approved public-reference Git commit |
| XRD schemas and Composition behaviour | Production API group under a controlled domain |
| Retain-by-default lifecycle and platform-controlled Delete authorization | Secret-store kind/name, cloud region, and workload identity |
| Placeholder SSO and incident profile shapes | Real endpoints and `ExternalSecret` remote paths |
| Comprehensive request with reserved example identities | Globally unique stack slug, intended recipients, user IDs, groups, and verified fixed-role UIDs |

An Argo CD `Application` can source `platform/` directly from this repository at an immutable
commit. Its Kustomize patches must replace every XRD group-qualified name, every XRD `spec.group`,
every Composition `spec.compositeTypeRef.apiVersion`, and the stack Composition input `apiVersion`.
A second source in the same `Application` can hold the environment `SecretStore`, `ExternalSecrets`,
and one ProviderConfig/credential Secret per registered organization in every request namespace.
The copies keep the registry's ProviderConfig name so namespaced references resolve locally. This keeps one Argo owner
while avoiding a copied platform implementation.

The comprehensive request can be consumed the same way from
`examples/catalog/comprehensive`. Apply private Kustomize patches for every custom kind you use
rather than editing generated managed resources:

1. replace the request `metadata.name`, `spec.slug`, display name, region, immutable usage
   (`development` or `production` in the reference vocabulary), and request references;
2. replace every `spec.stackRef.name` and any baseline managed-resource names containing the
   example slug;
3. replace example Team names, members, external groups, role permissions/scopes, and fixed-role
   UIDs with reviewed values;
4. disable SSO unless its selected profile and Secret exist;
5. disable incident delivery unless its relay endpoint and authorization Secret exist;
6. disable the report unless the recipients are intentional and the stack has the required
   capability;
7. retain the plugin only after approving its slug/version and entitlement.

Do not put a patched live request in this public repository's `enabled/` directory — that
publishes a real cloud-resource identity and couples a production deployment to mutable public
data. Keep enabled requests or overlays in a private GitOps repository, and pin public platform
and catalog sources to reviewed commits.

## Next steps

- [Request Schema Reference](reference/request-schema.md) — every per-API field and admission rule.
- [Reference → Catalog](reference/catalog.md) — every worked example and what it demonstrates.
- [SSO](sso.md) — the four OAuth/SAML providers and the four reconciliation modes.
- [Architecture](architecture.md) — how platform policy flows through Crossplane into managed
  resources.
