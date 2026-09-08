# Stack inventory

Use this inert example to observe objects in an existing Grafana Cloud stack and
separate objects declared by the platform from objects returned by the provider
that have no declaration.

## Purpose

`GrafanaStackInventory` composes read-only provider data sources for folders,
dashboards, teams, users, library panels, Synthetic Monitoring probes, and Fleet
Management collectors. It waits for the referenced `GrafanaCloudStackRequest`
to report `Ready=True`, then publishes `status.declared`,
`status.observedAndManaged`, and `status.observedButUnmanaged`.

## Deliberate omissions

- This directory declares no live stack and contains no credentials, endpoints,
  users, dashboards, or other source-environment identity.
- The inventory never creates, updates, or deletes a Grafana object. It only
  uses the provider's `.o.` observe-only API groups.
- Optional domain families such as k6, OnCall, cloud integrations, and AWS
  scrape jobs are not inferred from a stack request. Add a separately owned
  domain inventory when its credentials and lifecycle are approved.
- `OrganizationUser` is queried only when a declaration supplies an email or
  login selector; the provider does not expose it as an all-users data source.

## Values and caveats

- Replace `replacewithstack01` with the existing stack request name in this
  namespace. The stack and its per-stack `ProviderConfig` must already be
  healthy before the inventory can observe anything.
- Populate `spec.declared` with generic object references only after reviewing
  the target stack. Use the exact API version, kind, and external name where
  available; use `email` or `login` only for an `OrganizationUser` lookup.
- The catalog is not watched by Argo. Copy and adapt it into the deliberate
  live-request path only after reviewing provider credentials, entitlements,
  and the resulting read-only children.
