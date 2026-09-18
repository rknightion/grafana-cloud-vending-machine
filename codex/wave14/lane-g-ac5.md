# GCV-0078 AC5 completing-revision delta

Derived from frozen baseline `85c4344a5146eea98b4bfa9fb1c110858cd1f152` to the
wave-14 pinning revision `1f2af4a0b37c244a799e1fe0cdfcea2fa5abd82e`. The runtime
function published from source revision `7b3598efa68d980947e63cc01bc233ed74cafc7a` is
`sha256:b59b3cded495d40983869fd926c04f776fe0b9e77d0312338529336627d0a41c`.
Hosted source validation is run `35363219843`; pinning-revision validation run `35364595851`
also passed at the exact pinning revision.

## Artifact identities

| Artifact | Frozen baseline | Wave-14 pinning revision |
| --- | --- | --- |
| `platform/` tree | `4f8f9180440f43ea4912b8d8fc27bb915f898c48` | `ff4f4735f259a2ce56f8d72fec4ad659f8a73ef5` |
| `platform/apis/` tree | `1a4acc6a14607c0dbddab4e6783d8a59ca536a2c` | `42b7195e68a41d4f365de36139c7b4b6ba7f60b2` |
| `platform/function/` tree | `e96cb8f9453f0cc362fb5100ef2acef0208c690a` | `265229ecc71d677ece223434f1fdc1d82d21ff90` |
| `platform/provider/` tree | `f4469ec9e61888bc53a83cd3012ccfeb857edefa` | `9703637e177b92315683a257708a859f43d81aae` |
| `deploy/argocd/` tree | `03e347d08f53804b51b5bc4fba50ae5e44433d24` | `03e347d08f53804b51b5bc4fba50ae5e44433d24` |
| minimal catalogue tree | `04804ab098726abc39f345ae427ad6f121e1c7f4` | `5c1b015cff81defc9ce086d57dff97560a0e2a80` |
| minimal request input SHA-256 | `643fcc059438b2c40cc126fc0b5235dc80e1feda242f5f6e1c0d94096dea2b13` | `1af123507ba76d7d767e7941eb97d10b945ad06307099de211ab24c02e54697e` |
| provider manifest SHA-256 | `d2e5041e7fcf8cd77499489ee8d306d1f339c00a736276c6e798c3e827f8dd91` | `04076df1b3e7d353cd680f07013a6ac4e50ad45346cc395e494e7074d040cb2e` |
| function install SHA-256 | `326cff56e51d9ecad5103267099bd82a3349c70a83fd5d6ade73eb02dbd266b3` | `036ac437a89bc8180fa4723da169161b4b2fb2e1dd096e95f5967330137ebc43` |

The Argo delivery inputs are byte-identical across the comparison. The platform input is not:
the provider moves from the pre-release v2.13 build to signed v2.14.0, and the function digest
moves from `sha256:fb5e86a7a664572ef3383da16e85f1468c6d13ac8fd9abff61268daeb5bc44b8`
to the signed wave-14 digest above.

## Schema and composition delta

The baseline contains four XRDs and four Compositions in two API files. The target contains 24 XRDs
and 24 Compositions in 23 API files. The original four surfaces remain: stack request, team access,
content access and custom-role binding. Twenty target surfaces are additive, including inventory,
consumer, service-account, provisioning, observability-product, alerting, synthetic-monitoring and
k6 request kinds. The cutover inventory must select only the kinds actually active for the request;
the repository cannot infer that live set.

For `GrafanaCloudStackRequest`, the target adds required `spec.organization`, makes it immutable,
and adds the `expiry`, `products`, `retention` and refused compatibility `scim` fields. The existing
required `displayName`, `slug`, `region` and `usage` fields remain, and `slug`, `usage` and `profile`
retain their immutability rules. The pinned API-server test proves that an old stored request cannot
gain `organization`, whether the update changes only that field or also changes a mutable sibling.
Replacement is therefore required; the released transition rule is unchanged.

The minimal catalogue request adds only `spec.organization`. The frozen delivery generator must
also preserve its existing slug, region, profile, lifecycle and byte-for-byte usage value; add that
usage to both the platform-wide and selected-organization `allowedUsages`; retain every active
usage-keyed profile; and fail if its selector no longer matches the completing catalogue input.
The generator and its estate-specific overlay are outside this public repository, so their rendered
output has no honest repository hash. Applying those patches cleanly and hashing that render is an
AC6 pre-state check, not evidence claimed here.

## Migration disposition

This re-derived delta satisfies AC5 against the actual wave-14 pinning revision. It does not satisfy
AC6. No live request, cluster, stack, credential, remote document, consumer or generator was
contacted or changed. The cutover also must not start on this pin: GCV-0075 is Parked and the shipped
credentials are not self-renewing. Resume with the lane-G pre-state inventory only after a safe
rotation implementation has landed and its function digest is the intended target pin.
