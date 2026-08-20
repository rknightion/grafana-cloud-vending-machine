# Enforced Azure AD OAuth SSO

Use this example for Azure AD OAuth with group-aware Grafana role mapping controlled by platform policy. It omits `spec.lifecycle.externalResources`, so external-resource handling remains the safe `Retain` default.

## Prerequisites

- The minimal platform and organization ProviderConfig prerequisites are met.
- An Azure application registration has approved redirect URIs, tenant-specific authorization/token endpoints, and the required claims and consent.
- The Composition's `example-azuread` profile has been replaced with real tenant/application settings and reviewed group-to-role expressions.
- The application client secret is materialized into the profile's referenced Kubernetes Secret through the external secret manager.
- Group overage behavior is accounted for and a tested break-glass or alternate administrator login exists.

## Values to replace

- Replace `platform.example.org`, every `replacewithunique06` occurrence, the region, usage, profile, and display name. In the reference, `usage` must be the immutable platform vocabulary `development` or `production` and forms part of `{outputSecretPrefix}/{region}/{usage}/{slug}`.
- Replace `example-azuread` with the approved Composition profile. Keep credentials and tenant endpoints in platform policy and external Secrets, not in the request.

## Reconciliation

The `enforced` mode makes the selected Azure AD settings authoritative. Crossplane polls Grafana and restores out-of-band UI changes to managed SSO fields. Changing the profile or provider is an identity migration and should be tested with an explicit login and rollback plan.

With the default `Retain` lifecycle, removing this request orphans external resources. A platform-
authorized Delete decommission requires three reviewed stages: arm
`spec.lifecycle.externalResources: Delete` and wait for `status.deletionReady=true`; remove
dependent access claims and wait for their Kubernetes objects and finalizers to be gone while the
Stack still exists; then remove the request. Arming alone never deletes anything.
