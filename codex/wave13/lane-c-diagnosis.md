# Lane C diagnosis: composed-resource spec churn

## Finding

The churn is spec-writer contention, not list ordering. The composition emits reference fields while
the generated provider resolver writes the corresponding resolved scalar into the same
`spec.forProvider` object. The next composition render omits that scalar, so the two controllers can
alternate writes and increment `metadata.generation` without any user change.

This explains the AccessPolicy discriminator exactly. The two per-stack policy constructors emit
`realm[].stackRef`; with two stacks that produces the four churning policies. The three
stack-consumer policies emit `realm[].identifier` directly and are the three quiet policies. The
pinned provider's generated resolver copies a Stack's computed `id` into
`spec.forProvider.realm[].identifier`. Scope order cannot explain this split and has already been
falsified by the live evidence.

FolderPermission and DashboardPermission have the same contention at more fields:

- `folderRef` resolves into `folderUid`, or `dashboardRef` resolves into `dashboardUid`;
- each `permissions[].teamRef` resolves into `permissions[].teamId`; and
- the Terraform provider reads the effective `orgId` into state, which can be late-initialized into
  spec because that field is optional and computed.

The permission kinds therefore have more independent opportunities to rewrite spec than
AccessPolicy, consistent with their roughly 2.6 times higher generation count. The composition fix
in this lane keeps the declared references but, once the provider has resolved them, preserves the
observed UID, team ID and organization ID in the desired object. Direct actor IDs and direct target
UIDs remain unchanged.

## Evidence and origin

The repository pins crossplane-provider-grafana v2.14.0. Its tag resolves to revision
`dc795606df97a72dce81a0c953e0ec0750e0b489` and depends on terraform-provider-grafana v4.45.1 at
revision `7f3311691b0e124c55347f7204501fba77f61453`.

The pinned CRDs expose all of the fields above in `forProvider`. The generated v2.14.0 resolvers
assign `Identifier`, `FolderUID`, `DashboardUID`, and nested permission `TeamID` directly in spec.
The Terraform v4.45.1 bulk-permission implementation reads the effective folder or dashboard UID,
organization ID, and permission actor IDs into state. Inactive permission actor fields are null and
are not a normalization mismatch; the earlier provider's zero-value behavior does not apply to the
pinned version.

Repository renderers write these fields:

| Kind | Composition-written API fields | Fields that another controller can add to spec |
| --- | --- | --- |
| AccessPolicy | `displayName`, `name`, `realm[].type`, either `realm[].stackRef` or `realm[].identifier`, `region`, `scopes`, optional `conditions[].allowedSubnets` | `realm[].identifier` when `stackRef` is used |
| FolderPermission | `folderRef` or `folderUid`; `permissions[].permission` and exactly one of `role`, `teamRef`, or `userId` | `folderUid`, `permissions[].teamId`, `orgId` |
| DashboardPermission | `dashboardRef` or `dashboardUid`; `permissions[].permission` and exactly one of `role`, `teamRef`, or `userId` | `dashboardUid`, `permissions[].teamId`, `orgId` |

The driver originates in the composition shape: the provider resolver and late initializer are
performing their documented jobs, while the composition repeatedly renders a sparse pre-resolution
shape. The provider needs no change for these resources.

The repository has no duplicate declarative owners for these kinds. There is one
GrafanaContentAccessPolicy renderer and no raw FolderPermission or DashboardPermission manifest.
The three AccessPolicy constructors use distinct child identities. A consumer can still create two
GrafanaContentAccessPolicy requests for one external target, but that would make two external ACL
writers; it would not explain the observed Kubernetes generation growth. The competing writers here
are the composition and generated provider resolver on one managed resource.

## Ranked alternatives

1. Resolved reference scalars omitted by the composition: high confidence. It is present in all
   three affected kinds and the AccessPolicy direct-identifier split matches the four-versus-three
   discriminator.
2. Optional/computed `orgId` late initialization on permission kinds: contributing field. It cannot
   explain AccessPolicy, but it is another permission-spec field the sparse renderer was dropping.
3. Empty versus omitted AccessPolicy `conditions` or `labelPolicy`: low confidence. It could affect
   selected policies, but it does not match the direct-identifier discriminator.
4. API list ordering: rejected. Two high-churn policies already match API scope order while a quiet
   policy does not, and the CRD declares scopes as a set.

## Fix and remaining proof

`roles.go` now renders provider-normalized FolderPermission and DashboardPermission fields from the
observed child. Root must wire the existing observed map into that renderer using the packet below.
The AccessPolicy packet replaces references with an observed Stack ID as soon as one exists, while
retaining the reference during initial staging and as a last-resort transient fallback.

Live reconciliation is not exercised, and is not exercisable here. After publishing and pinning the
function, the operator should compare the same objects' generations over several burst windows and
confirm that the composite watch circuit closes. Until then the bounded cost remains slower
event-driven convergence, repeated provider/API work, and an uninformative `Responsive=False`
condition. If generation still advances, the settling evidence is a field-by-field diff of a quiet
and churning object's `spec`, `status.atProvider`, and `metadata.managedFields`; no further guessed
normalization should be coded.

## Root wiring packet

In the `GrafanaContentAccessPolicy` registry entry in `platform/function/fn.go`, pass the observed
map through:

```go
render: func(xr map[string]any, observed map[resource.Name]resource.ObservedComposed, _ map[string]any) (map[resource.Name]*resource.DesiredComposed, error) {
	return renderContentAccessPolicy(xr, observed)
},
```

## Exact AccessPolicy edit packet

Add this helper to `platform/function/access.go`:

```go
func stackAccessPolicyRealm(observed map[resource.Name]resource.ObservedComposed, policy resource.Name, stackName string) []any {
	identifier := observedString(observed, "stack", "status.atProvider.id")
	if identifier == "" {
		identifier = observedString(observed, policy, "spec.forProvider.realm[0].identifier")
	}
	if identifier != "" {
		return []any{map[string]any{"identifier": identifier, "type": "stack"}}
	}
	return []any{map[string]any{"stackRef": map[string]any{"name": stackName}, "type": "stack"}}
}
```

In `addTelemetryAccess`, replace the literal `realm` value with:

```go
"realm": stackAccessPolicyRealm(observed, "telemetry-access-policy", slug),
```

In `addFleetAccess` in `platform/function/fleet.go`, replace its literal `realm` value with:

```go
"realm": stackAccessPolicyRealm(observed, "fleet-management-access-policy", slug),
```

Add focused tests that assert both constructors emit `realm[].identifier` when the observed Stack
has `status.atProvider.id`, retain an already resolved identifier from the observed policy when the
Stack ID is temporarily absent, and use `stackRef` only before either observed value exists.
