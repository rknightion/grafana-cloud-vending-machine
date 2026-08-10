---
title: SSO
description: OAuth and SAML configuration for vended Grafana Cloud stacks, and the platform-owned profile model
---

# SSO

The Grafana provider supports OAuth settings for `github`, `gitlab`, `google`, `azuread`,
`okta`, and `generic_oauth`, plus SAML. This reference accepts those Grafana Cloud-relevant
providers. The upstream schema also contains LDAP settings, but LDAP is a self-managed Grafana
integration rather than a portable Grafana Cloud stack profile, so the composition function
rejects it.

## Profile ownership

Platform owners define complete profiles in the Composition input
(`spec.ssoProfiles` — see [Configuration → Platform configuration](configuration.md#platform-configuration)).
Request authors select only a profile name (`spec.sso.profile`) and a reconciliation mode
(`spec.sso.mode`) — they cannot supply an arbitrary identity endpoint or client secret directly
in a request.

OAuth client secrets must be `LocalSecretKeySelector` references; literal client-secret fields
are not copied by the function. SAML can use an IdP metadata URL, or `certificateSecretRef` and
`privateKeySecretRef` when the chosen flow requires key material.

| Pattern | Important profile fields | Role behaviour |
| --- | --- | --- |
| Generic OAuth/OIDC | `authUrl`, `tokenUrl`, `apiUrl`, `clientId`, `scopes`, claim paths | `roleAttributePath` maps claims to Viewer, Editor, Admin, or None |
| Azure AD | tenant-specific auth/token URLs, `clientId`, group claim path | group-aware `roleAttributePath`; account for group-overage behaviour in the IdP design |
| GitHub, GitLab, Google, or Okta | provider-specific organizations/domains/groups and client credentials | Use allowed groups/organizations as an admission gate, then map the resulting role |
| SAML | `idpMetadataUrl`, assertion attributes, `roleValues` fields, signature settings | IdP role values map to Grafana basic roles |

`roleAttributeStrict: true` rejects a login when no valid role can be derived.
`skipOrgRoleSync: true` has the opposite ownership implication: Grafana stops updating the user's
organization basic role from the IdP — use it only when another reviewed process owns
organization roles. `allowAssignGrafanaAdmin` is far more privileged than organization Admin and
should remain `false` unless server-administrator assignment is explicitly required and
supported.

## Reconciliation modes

Every supported profile type shares the same four modes, selected with `spec.sso.mode`:

| Mode | Effect |
| --- | --- |
| `enforced` | Settings live in `forProvider`; Crossplane repairs UI drift. |
| `createOnly` | Settings live in `initProvider`; later administrator changes are preserved. |
| `observeOnly` | Only `providerName` is supplied, with Observe permission — Crossplane never creates or updates. |
| `disabled` | No SSO managed resource is desired at all. |

Keep `providerName` stable when switching ownership. Changing from `generic_oauth` to `saml` (or
between any two provider types) is an identity-provider migration, not a routine mode toggle —
plan and test the login path and a rollback before doing it.

## Example profiles in this repository

`platform/apis/v1beta1.yaml` embeds three example profiles in its Composition input, matching
the `sso-create-only`, `sso-azuread`, and `sso-saml` catalog examples:

- **`example-oidc`** — a `generic_oauth` profile pointed at `identity.example.com`, mapping
  membership of a `grafana-admins` group to the Admin role and everyone else to Viewer.
- **`example-azuread`** — an `azuread` profile with a two-tier group-based role mapping
  (`example-grafana-admins` → Admin, `example-grafana-editors` → Editor, else Viewer).
- **`example-saml`** — a SAML profile using an IdP metadata URL and a four-way role-value mapping
  (Admin/Editor/Viewer/None).

All three are placeholder identities under `platform.example.org` and `identity.example.com` — do
not point live SSO at them; see [Configuration](configuration.md) for the full placeholder
warning.

Each profile's OAuth client secret or SAML material is read through an `ExternalSecret` — see
[Secrets → SSO and incident profile secrets](secrets.md#sso-and-incident-profile-secrets).

## Which catalog example to start from

| Need | Example |
| --- | --- |
| Platform initializes OAuth once, then hands ownership to stack administrators | [`sso-create-only`](https://github.com/rknightion/grafana-cloud-vending-machine/tree/main/examples/catalog/sso-create-only) |
| Enforced Azure AD OAuth with group-based role mapping | [`sso-azuread`](https://github.com/rknightion/grafana-cloud-vending-machine/tree/main/examples/catalog/sso-azuread) |
| Enforced SAML with IdP metadata and role-value mapping | [`sso-saml`](https://github.com/rknightion/grafana-cloud-vending-machine/tree/main/examples/catalog/sso-saml) |
| Every SSO pattern alongside the rest of the vending API | [`comprehensive`](https://github.com/rknightion/grafana-cloud-vending-machine/tree/main/examples/catalog/comprehensive) |

See [Reference → Catalog](reference/catalog.md) for the complete catalog, including the
non-SSO examples.

## Next steps

- [Configuration](configuration.md) — the `spec.sso.*` request fields and platform-level
  `ssoProfiles` shape.
- [Secrets](secrets.md) — how OAuth client secrets and SAML key material reach the cluster.
- [Architecture → reconciliation](architecture.md#reconciliation-and-out-of-band-changes) — the
  full drift-behaviour table across every resource type, not just SSO.
