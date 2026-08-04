# Enforced SAML SSO

Use this example for a SAML integration whose metadata, assertion attributes, role values, and signing policy are controlled by the platform.

## Prerequisites

- The minimal platform and organization ProviderConfig prerequisites are met.
- The IdP metadata URL is reachable from Grafana Cloud and its entity, SSO, certificate, NameID, and signature settings have been reviewed.
- Assertion attributes and role values match the Composition profile.
- Any certificate or private-key flow uses Secret references populated from an external secret manager.
- A tested break-glass or alternate administrator login exists before SAML is enabled.

## Values to replace

- Replace `platform.example.org`, every `replacewithunique05` occurrence, the region, usage, profile, and display name.
- Replace `example-saml` with an approved Composition profile containing the real metadata and mapping. Do not put signing keys in the request or Git.

## Reconciliation

The `enforced` mode makes the selected SAML settings authoritative. Crossplane polls Grafana and restores out-of-band UI changes to managed SSO fields. Moving to `createOnly` hands later settings changes to administrators; moving to `observeOnly` removes write authority while retaining observation.
