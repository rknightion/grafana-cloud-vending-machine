# Private Data Source Connect

This inert example vends one PDC network and attaches one datasource to it. The
network ID is provider-assigned, so the token and datasource wait until it is
observed. The token value stays in its derived connection Secret and is never
published to the composite, the catalog, or an external secret store.
Each bounded renewal window has its own connection Secret. Consumers refresh
from the current token child reference; the example does not configure a live
agent or claim automatic credential rotation.

## Values to replace

- Replace `platform.example.org` with your API group.
- Replace `replacewithunique01` with the name of an already-ready stack request.
- Select only an approved platform-owned PDC profile and use a token lifetime
  no greater than that Composition's `maximumTokenLifetime`.
- Replace the reserved invalid datasource hostname in an environment overlay.

## Ownership

This request exclusively owns the rendered datasource and derives its UID from
the PDC composite. Do not create a `GrafanaDatasourceAccess` claim for the same
datasource. Use that API for a datasource it owns itself.
