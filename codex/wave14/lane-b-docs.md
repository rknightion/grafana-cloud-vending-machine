# Wave 14 Lane B documentation packet

This packet names the documentation targets and finished prose. The source
pages remain root-owned; no page was edited by this lane.

## `docs/security.md`

Insert after the current `### Rotating administrator token` material, before
the telemetry section, under the anchor `## Overlapping credential rotation`:

> Rotating credentials use overlapping generations. The function keeps the
> serving token, its connection Secret, and the existing PushSecret while it
> creates and observes a successor. A successor receives a new composition key,
> Kubernetes name, connection Secret name, and provider name prefix. The
> predecessor is never renamed or deleted to make room for it.
>
> The successor must be Ready and Synced, have a provider-assigned identity,
> and have a readable Secret owned by that token. The function then selects the
> successor in the existing PushSecret. It waits for the observed publisher
> generation to acknowledge that selector, including the successful publication
> generation, before checking any local consumer handover. A stale Ready
> condition does not acknowledge a new selector.
>
> Administrator and Fleet Management credentials also wait for the materialized
> stack ProviderConfig Secret to contain the successor credential. The stack
> composite status records the selected administrator and telemetry Secret
> references after their handovers are observed. K6 and Synthetic Monitoring
> resolve those references when present and use the legacy names only while the
> additive fields are absent during migration.
>
> The predecessor remains desired until the handover and retirement intent are
> observed. After the exact predecessor key and UID are absent, the function
> omits that generation and does not recreate the legacy child. Failed, pending,
> malformed, or ambiguously identified candidates remain desired alongside the
> serving generation. Secret data and credential digests never enter status,
> annotations, logs, or errors.

## `docs/secrets.md`

Insert after the existing rotating-token lifecycle section under the anchor
`## Generation-specific Secret ownership`:

> Every token generation owns a distinct connection Secret. A Secret qualifies
> for publication only when its namespace, name, fixed data key, UID, and token
> owner UID match the observed generation. A same-named Secret left by an older
> generation is not evidence. The publisher selector changes in place so its
> remote path, output shape, store, and deletion policy remain stable.
>
> The function retains observed Secret and token members on every fresh desired
> map, including a failed or pending successor. Missing Secret observations,
> changed Secret UIDs, unknown token timing, and malformed generation lineage
> block rotation visibly and preserve the currently serving credential.

## `docs/troubleshooting.md`

Add under the existing credential troubleshooting heading at anchor
`## CredentialRotationHealthy`:

> Read the composite `CredentialRotationHealthy` condition together with the
> child Ready and Synced conditions. `ReplacementPending` means a successor is
> being created or its owned Secret is not readable. `PublicationPending` means
> the desired publisher selector has not been acknowledged by the observed
> PushSecret generation. `HandoverPending` means publication succeeded but the
> materialized consumer or composite status still references the predecessor.
> `RotationBlocked` means identity, lineage, timing, or Secret evidence is
> ambiguous. `RotationWindowReached` and `CredentialExpired` indicate that the
> serving token has reached its replacement deadline.
>
> The composite stays Ready=False for all of these states, even when a provider
> child still reports Ready=True. The condition message includes the family,
> deadline, and whether timing came from provider status or a conservative
> creation estimate. It never includes token bytes or hashes.

## Acceptance boundary

These pages describe the local/source controller contract. They do not claim
live Grafana availability, external consumer reload, provider revocation, or
cluster permission isolation. The implementation proves overlap and ordering in
the offline replay; live external behavior remains an operational verification
item.
