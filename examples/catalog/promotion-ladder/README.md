# Multi-stack promotion ladder

This inert catalog example vends two independent Grafana Cloud stacks and one
Git provisioning repository per stack. The declared `promotionDirection` makes
the content flow visible: `forward` follows the listed rungs and `reverse`
follows the opposite order. The composition only configures each repository to
read its branch. It never writes Git, merges branches, or promotes content.

## Files

- `ladder.yaml` declares a development-to-production ladder.
- `kustomization.yaml` makes the directory a directly renderable Kustomize base.

## Preconditions

- Keep the organization, region, rung names, rung order, and rung slugs stable.
  Region changes require stack replacement, so the API rejects them rather than
  silently recreating a stack.
- All ladder stacks belong to one Grafana Cloud organization and one region.
  Multi-organization isolation is not available in a ladder.
- The free tier permits one stack and self-service paid plans permit three. A
  ladder with more rungs needs a negotiated contractual limit. Multi-stack data
  sources are limited to one region and were capped at ten stacks in preview.
- A separately managed `ConnectionV0Alpha1` exists for each rung. It holds the
  reviewed Git credential; this request contains no credential material.
- The selected branches and repository path contain reviewed dashboard content.

## Values to replace

- Replace `platform.example.org`, the ladder name, every rung slug, the
  organization, region, display names, usage values, branches, repository URL,
  and path with approved values.
- Keep each `connectionRef.name` distinct and replace it with the existing
  Connection for that rung. Do not add a Secret or credential to this manifest.
- Choose `forward` or `reverse` explicitly according to the reviewed delivery
  flow. Rung names do not infer direction.

## Topology and reconciliation

Shared-stack and stack-per-tenant topologies are both deliberately supported
models. Grafana recommends a single production stack when teams, folders, RBAC,
data source permissions, and LBAC provide sufficient isolation. Separate stacks
fit a development/staging ladder or complete departmental isolation. The stack
request API is shaped for stack-per-tenant, while the team-access and
content-access APIs model slices of a shared stack.

The ladder status reports each rung's branch, stack readiness, source rung, and
observed repository version. It marks a target `InSync` or `Drifted` only when
both adjacent rungs report a version; otherwise it reports `Unknown`. That
status observes branch content only. Promotion remains an approved Git action.

The catalog is not watched by the example ApplicationSet. Copy it into the
deliberate live-request path only after reviewing capacity, organization,
connection ownership, branch protections, and the resulting stack resources.
