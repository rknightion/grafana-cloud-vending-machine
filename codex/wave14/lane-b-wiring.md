# Wave 14 Lane B wiring packet

Accepted design: `codex/wave14/lane-a-packet.md`, SHA-256
`f52fdcfe4e9641fd03c7c201621332c9bc5d70c706b45d484c8d06b60fb31051`.
Source identity: baseline `52308041d25f465e21e711add0dfcf7578f94866`.

The execution seam in `platform/function/access.go` is:

```go
type rotatingTokenFamilyDescriptor struct {
    Family string
    LegacyKey, PublisherKey resource.Name
    TokenAPIVersion, TokenKind, Namespace string
    LegacySecretName, SecretKey string
    NamePrefix string
    Lifetime, EarlyRotationWindow time.Duration
    MintingProviderRef, ConsumerProviderRef map[string]any
    RequireConsumer, RequireStatusReference, PublicationOnly bool
}

type rotatingTokenRotationInput struct {
    Now time.Time
    Secrets map[string]rotationSecretObservation
    ConsumerReady, StatusReferenceReady bool
}

func reconcileRotatingTokenFamily(
    desired map[resource.Name]*resource.DesiredComposed,
    observed map[resource.Name]resource.ObservedComposed,
    d rotatingTokenFamilyDescriptor,
    in rotatingTokenRotationInput,
) rotatingTokenRotationResult
```

The helper mutates `desired` in place and returns `rotatingTokenRotationResult`.
The result carries `Phase`, `Reason`, `Healthy`, `SelectedKey`,
`SelectedSecret`, `RequiredSecrets`, per-member `Timings`, and exact
`RetirementIntent`/`Retired` key and UID pairs. Do not replace the desired map
or discard the result. Token bytes are accepted only in the in-memory
`rotationSecretObservation` and never enter a desired object, status,
annotation, log or error.

## RunFunction insertion points

In `RunFunction`, after `renderer.render` has produced the complete base map and
after the existing required stack/provider context has been resolved, construct
five descriptors and run the helper once for each enabled family:

| Family | Legacy key | Publisher key | Kind/group | Secret key | Handover |
| --- | --- | --- | --- | --- | --- |
| administrator | `stack-token` | `credentials` | `StackServiceAccountRotatingToken` / cloud | `attribute.key` | stack ProviderConfig plus observed administrator status reference |
| telemetry publisher | `telemetry-token` | `telemetry-credentials` | `AccessPolicyRotatingToken` / cloud | `attribute.token` | observed telemetry publisher status reference |
| Fleet Management | `fleet-management-token` | `fleet-management-credentials` | `AccessPolicyRotatingToken` / cloud | `attribute.token` | materialized stack ProviderConfig |
| stack consumer | `stack-consumer-token` | `stack-consumer-credentials` | `AccessPolicyRotatingToken` / cloud | `attribute.token` | publication acknowledgement |
| in-stack account | `service-account-<account>-token` | `service-account-<account>-credentials` | `ServiceAccountRotatingToken` / oss | `attribute.key` | publication acknowledgement |

Use the configured stack namespace and the existing stable publisher names.
The organization ProviderConfig belongs in `MintingProviderRef` for the
administrator, telemetry, and Fleet families; the stack ProviderConfig belongs
in `ConsumerProviderRef`. For the OSS family the stack ProviderConfig is the
minting reference and no local consuming ProviderConfig gate is required. The
stack-consumer profile's authorized organization ProviderConfig is its minting
reference and it has publication-only handover.

Run the first pass with the injected `config[reconcileTimeConfigKey].(time.Time)`
and the Secret observations available at that reconcile. Add the returned
required selectors to the existing `rsp.Requirements.Resources` map, preserving
all unrelated requirements. Use one deterministic requirement name per
namespace/name/key, for example `rotation-secret-<family>-<short-hash>`; do not
use the token value or a timestamp in the name. A selector is:

```go
&fnv1.ResourceSelector{
    ApiVersion: "v1",
    Kind: "Secret",
    Match: &fnv1.ResourceSelector_MatchName{MatchName: selector.Name},
    Namespace: &selector.Namespace,
}
```

On the next invocation, read each selected Secret through
`request.GetRequiredResource`. Reject a resource unless its kind/apiVersion,
namespace, name, non-deleting metadata, non-empty UID, fixed key, and owner UID
match the observed token. Decode the Secret data in memory and populate
`rotationSecretObservation`. Same-named leftovers do not qualify. Continue to
render and retain every observed token member when the required resource is
absent or invalid.

The helper must be called after the normal renderers have constructed the
publisher and token template and before `productWithdrawalError`, readiness
gating, `desiredStackStatus`, and `SetDesiredComposedResources`. On a fresh vend
the helper has no observed family and leaves the base map untouched. On a
rotation it retains every observed generation and changes only the existing
publisher selector and non-secret handover annotations.

## Desired/status wiring

Keep the original `config` map alias used by `addStackExpiry`; add a private
`_rotationResults` map to that same map in place. Do not make a replacement map
for the result or the `_expiryStatus` side effect. Pass the same map to
`desiredStackStatus`.

Immediately before the final `response.SetDesiredComposedResources`, call:

```go
if !pruneRotatingTokenRetiredDesired(rsp.Desired.Resources, observed, retired) {
    response.Fatal(rsp, errors.New("rotation retirement UID conflicts with observed resource"))
    return rsp, nil
}
```

`retired` is the concatenation of the helper results' exact key/UID pairs. This
prunes only an observed, authorized predecessor after the intent is observed;
it never sweeps a kind or name prefix and never relies on the incoming desired
map containing the predecessor.

After normal status generation and readiness helpers, apply one deterministic
composite condition per configured family. The condition type is
`CredentialRotationHealthy`, status `True`/reason `Healthy` only when all
enabled families have known valid timing and no transition. Use `False` and
the helper phase/reason for `ReplacementPending`, `PublicationPending`,
`HandoverPending`, `RotationBlocked`, `RotationWindowReached`,
`CredentialExpired`, or `RetirementPending`. Include the family, phase,
deadline, and timing provenance in the message; never include Secret data or a
credential digest. Any rotation condition other than Healthy sets the desired
composite Ready to `False` while leaving child provider conditions untouched.

Add optional status fields to the stack composite at the same point that
`desiredStackStatus` publishes `tokenExpiries`:

```yaml
status:
  tokenConnectionSecrets:
    administrator: {name: <same-namespace Secret>, key: attribute.key}
    telemetryPublisher: {name: <same-namespace Secret>, key: attribute.token}
```

Publish the selected generation's observed connection Secret only after its
publication acknowledgement and materialized handover gate pass. Retain the
previous observed reference until the composite status itself reports the new
reference on a later invocation. Validate same namespace and fixed purpose key.
Fleet uses its observed materialized stack ProviderConfig and has no separate
status reference.

`bootstrap.go:serviceBootstrapConfig` and its K6/Synthetic Monitoring callers
must resolve these status references when the field is present. A missing field
keeps the existing legacy fallback for migration. A present but malformed field
blocks the consumer and therefore blocks retirement.

## Deletion and fixture seam

Replace the three legacy calls in `desiredStackStatus` and the matching calls in
`expiry.go:expiryDeletionState` with
`rotatingTokenFamilyDeletionPrepared(observed, descriptor, input)`. The helper
requires a complete unambiguous family inventory, an observed current selector,
all members' `managementPolicies: ["*"]` and `deleteOnDestroy: true`, a
prepared publisher, and no unresolved promotion, consumer, status-reference,
or retirement handover. It accepts a promoted family without the legacy key
only after the exact predecessor is absent and retirement intent is observed.
Malformed lineage, missing clock, unknown timing, missing publisher, and a
candidate handover return false. The result is preparation evidence only.

The pinned Crossplane chart already grants controller Secret read access through
its aggregate role. Do not add a duplicate core-Secret RBAC rule. Add no Secret
writes and no live-cluster checks. The stack rotating-token CRD fixture remains
a root-owned fixture change if a schema test requires it; source it from the
pinned provider.

## Renderer-specific seams

The administrator token is constructed in `fn.go:renderStack`; root must pass
its descriptor after the existing token and `credentials` PushSecret are built.
`addTelemetryAccess` and `addFleetAccess` use their existing publisher keys and
distinct organization minting provider references. `renderStackConsumer` uses
the profile-authorized provider and publication-only gates. Each account in
`renderServiceAccounts` receives an independent descriptor and generation key;
the ServiceAccount and Permission parents retain their provider-assigned IDs.
No constructor may derive an external ID or rename a pre-existing legacy child.
