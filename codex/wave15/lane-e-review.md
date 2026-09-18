# Wave 15 lane E security review

Status: FAIL. Single review pass completed; no implementation corrections made.

- GCV-0087: FAIL, source accounting defects E1 and E2.
- GCV-0069: FAIL, source defects E3 and E4, frozen-contract issue E5, and the separately identified root renderer blocker W1.
- Review base: `bc3e99a41e031e5c9a10b824cbc2ef21cdd9e779` plus the frozen working-tree candidate below. This is not a completing SHA.
- Sole reviewer-written file: `codex/wave15/lane-e-review.md`.

## Candidate and evidence boundary

Reviewed B's `credentialhealth.go`, `credentialhealth_test.go`, and exact `lane-b-wiring.md` packet; D's `oncall.go`, `oncall_test.go`, `alertingjoin.go`, `alertingjoin_test.go`, `oncall-v1beta1.yaml`, and `alerting-routing-v1beta1.yaml`. Source and test filenames are relative to `platform/function/`; API filenames are relative to `platform/apis/`; the wiring packet is under `codex/wave15/`.

Read the two tasks, goal sections 2.5, 2.6, 4.2, 4.4, 4.5 and 6, released API controls using `git show v3.0.0:...`, and relevant unchanged renderer, readiness, withdrawal and token emit code. Tracker was read-only; root's overview was reused. No memory was needed for this self-contained review.

Candidate SHA-256 values:

| File | SHA-256 |
|---|---|
| credentialhealth.go | `1bcb45e564e84c7391c8c13716a0db9838b34a4afc73ac2037a81c6fccac8539` |
| credentialhealth_test.go | `84819f7a260d9e843916d6da74f7ffe7efe19462cff0b18f531dc7e6fb72af60` |
| lane-b-wiring.md | `aa31ace25bde22e90bf0c218f2448af1ec047450c352864d9a2415b6fa975d4e` |
| oncall.go | `6c1839f2a4d3127ef674532c1058341be324dfb19f4a5daacbf6041e508aff5e` |
| oncall_test.go | `878e81136156f964b51c8aaf45a03b103b36eb1c5bc91ca37ed389f4fbe04634` |
| alertingjoin.go | `a193dee0a0085fa12392c7a85fd0b9517d6dfe4193fe7762aa2b0f2d48176338` |
| alertingjoin_test.go | `9dfa01af0f3953c5da506d7071ddea97a15e16344e161cf6100cd696f756ec34` |
| oncall-v1beta1.yaml | `74f49d2f6d9152a28f70e893606b32707f0ce0a37119a44e8cc940a7e42b16db` |
| alerting-routing-v1beta1.yaml | `ecc31c77b7ddcdedcf5785b8e1be36c7cfdadc202b1dd8dc105b7e173f40d239` |
| unchanged alertingrouting.go | `9ba4412f3907ae87b8504c6e751dfab4db27ed183e8ba52b0416936ad65b848f` |

One focused local execution from `platform/function`:

```text
go test ./... -run 'TestCredentialHealth|TestRenderOnCallSelectableIntegrationTypePreservesDefaultIdentity|TestRenderOnCallSlackChannelReferenceIsObservedAndRequired|TestRenderOnCallUnresolvedSlackChannelFailsLoudly|TestRenderOnCallOpaqueSlackChannelIDDoesNotCreateObserver|TestResolveAlertingReceiversProjectsObservedIntegrationURLIntoIRMContactPoint|TestIRMShapesJoinFromOneClaimSet' -count=1
--- FAIL: TestResolveAlertingReceiversProjectsObservedIntegrationURLIntoIRMContactPoint (0.00s)
    alertingjoin_test.go:334: contactPoints[0] must set name and email.addresses
--- FAIL: TestIRMShapesJoinFromOneClaimSet (0.00s)
    alertingjoin_test.go:428: contactPoints[0] must set name and email.addresses
FAIL
FAIL github.com/rknightion/grafana-cloud-vending-machine/platform/function 0.315s
FAIL
```

Exit 1. These were the only failures reported by the selected tests. Tool execution reference: session `17005`, completion chunk `89ad5b`. Counterexamples below are source traces, not newly executed regression tests: the reviewer was allowed to write only this packet. No integrated gate, API-server admission or hosted validation was run by this reviewer.

## Findings

### E1 - major - false healthy through interchangeable counts

Owner: Lane B correction attempt, or root after explicit ownership transfer. Location: `credentialhealth.go:32-49,87-92`.

Counterexample: change a service-account request from account `old` to account `new`. On the first reconciliation after the edit, observed composed resources still contain the old `ServiceAccountRotatingToken` with `Synced=True`; the newly configured account has no observed token. Expected count is one, found count is one, and the helper writes healthy=true. No observation says the new credential is healthy or even exists. Account-list changes are admitted; `productWithdrawalError` does not handle `GrafanaServiceAccounts`. The same masking occurs whenever an extra same-GVK child substitutes for a missing configured token.

This proves a false health verdict. It does not independently prove a Ready=True composite on that exact first reconcile, because normal desired-child readiness may separately hold it false. The new safety signal nevertheless loses its configured-but-absent guarantee, precisely when another readiness path must rescue it.

Exact correction: retain GVK discovery, but correlate each counted observation to the current configured credential identity using existing object/spec identity, not an unqualified family count. Derive expected membership from configuration even when the renderer emits no token. Do not merely require equality of counts: one old and one missing new token still have equal cardinality. Add the old-to-new replacement counterexample as a failing test before repair. No emit-site edit or new provider field is needed.

### E2 - major - false unhealthy from a retired credential

Owner: same as E1. Location: `credentialhealth.go:34-42`.

Counterexample: a request previously selected accounts `keep` and `retire`; it now selects only `keep`. Its current token has `Synced=True`; the retired token remains in the observed set during removal with `Synced=False`. Expected count is one, found count is two, but the unconditional scan sets healthy=false and the proposed final readiness hook overrides the current composite to Ready=False. The retired credential is no longer a configured destination. An unexpected rotating-token GVK similarly poisons an unrelated configured family.

Exact correction: after matching observations to current configured membership, aggregate that membership only. Preserve explicit missing-member failure and test a retired failed token alongside a current healthy token. This must be repaired together with E1; filtering by GVK alone or ignoring all excess children cannot identify which observation belongs to the current request.

### E3 - major - opaque-ID escape hatch still transmits display names

Owner: root owns disposition after D's two exhausted attempts; no automatic third lane attempt. Location: `oncall.go:202-217,237-238`.

Counterexample: set `route.channelId: operations-channel` on an otherwise valid request with observed prerequisite identities. The API admits it and the renderer puts exactly that display-name string in `slack[0].channelId`, with no observer and no Required reference. The task records that the vendor silently drops a display name supplied in this field. Calling it an opaque ID in the description does not validate it. Existing tests use another arbitrary placeholder and check pass-through only.

Exact correction: reject display-name input in the new renderer branch using the pinned provider/vendor opaque-ID contract, and withhold successful destination readiness until observed Route status confirms the requested destination. A syntactically plausible nonexistent ID also needs a mismatch/missing-destination refusal; syntax validation alone is insufficient. Keep the valid opaque-ID escape hatch and the Required-reference path. Do not add a released-API CEL or required field to achieve this. If the pinned observations cannot distinguish a dropped destination, return that precise blocker rather than assert AC3. No live probe is authorized.

### E4 - major - existing inbound-email references lose their released behavior

Owner: root after D ownership transfer. Location: `alertingjoin.go:84-137` and the modified default-path tests.

Counterexample: an existing default `inbound_email` OnCall receiver is Ready, current-generation, stack-bound, and publishes a valid `alertReceiver.address` and integration ID. Its Integration has no `status.atProvider.link`. The released resolver emits an email contact point from that receiver. The candidate demands the link and returns ready=false indefinitely. Even when a link exists, the candidate switches the existing contact point from email to OnCall without an input opt-in. The default integration's preserved names do not preserve this consumer behavior.

Exact correction: branch on the resolved integration type. Preserve the released email parsing/projection for `inbound_email`, and use the new internal Integration URL resolution for `grafana_alerting`. Restore a regression case with the released address-only observations and add the separate new-type URL case. Do not solve this by requiring a new field or changing the default.

### E5 - minor - literal frozen rule breached by a nested required field

Owner: root contract disposition; API file ownership transfers from D. Location: `oncall-v1beta1.yaml`, new `route.channelRef.required: [name]`.

Counterexample to the frozen edit rule: `git diff v3.0.0` contains a newly added `required: [name]`. Sections 4.4 and 6.3 explicitly classify adding any required field on either released API as a returned blocker; this candidate added it directly.

This is not evidence that an existing admitted input is newly rejected: the parent field is new and optional. That distinction matters. No existing required list, CEL rule, immutability rule or default was tightened. The precise conflict is with the literal frozen instruction, not a demonstrated breaking schema change.

Exact correction within this wave: remove the new nested `required` declaration and retain the existing renderer refusal for an empty `channelRef.name`, or return this exact nested-schema edit for the owner's explicit decision. Do not label the whole API structurally breaking solely from this optional nested constraint.

### W1 - major - known root renderer wiring blocker, not a new D source defect

Owner: root wiring pass. Location: unchanged `alertingrouting.go:34-39,59-62`.

Counterexample: successful resolution produces a contact point containing `oncall` and no `email`; the renderer rejects it with `contactPoints[0] must set name and email.addresses`. The focused run reproduces both AC4 and AC5 failures.

Exact correction: validate exactly one internal email or oncall shape, validate the oncall URL and nonempty `oncallIntegrationRef.name`, and emit only that selected provider block. Reject both/neither, malformed maps and invalid URLs with value-free errors. Do not format the URL or its parse error into response errors, logs or status; URL parser errors can include the original bearer value. Preserve ordinary email rendering. This proposed correction is suitable in principle, but no corrected code was available to approve.

## Eight review priorities

| Priority | Disposition |
|---|---|
| 1. False healthy and false unhealthy | FAIL: E1 and E2. Exact GVK comparison is sound, but membership is reduced to counts and all extras affect the verdict. Missing, malformed and Unknown Synced conditions fail closed for an observed matching child. Duplicate contradictory conditions are first-match; no admitted real-object counterexample was established, so no separate blocking finding. |
| 2. Configured absent family | PARTIAL: an entirely empty expected GVK is counted missing independently of rendering. A missing current member can still be hidden by a retired same-GVK child, E1. |
| 3. AccessPolicy with no expiry | PASS for the isolated condition logic. No expiry field is read; Synced=True succeeds, absent/False/Unknown fails. Family-accounting defects remain separate. |
| 4. Channel resolution | PARTIAL: channelRef creates observe-only SlackChannel and Required Route reference; an observed channel with empty slackID errors, and initial absence cannot resolve the Required reference. Opaque channelId has E3. Both optional fields together error. Neither remains allowed for released schedule-only requests; requiring a new channel globally would break those inputs. |
| 5. Integration identity | PASS for unchanged default names and non-reuse across types. Child suffix and display suffix remain byte-identical; external identity is reused only when observed metadata.name matches the selected identity. A type change with an existing route fails closed rather than reusing its old integration identity. |
| 6. Released API | No tightening of existing required lists, CEL, immutability or defaults; optional integrationType defaults to inbound_email. E5 records the new nested required-field instruction conflict. E4 is a behavioral compatibility failure despite the mostly additive schema diff. |
| 7. URL secrecy and root blocker | Current join deep-copies the claim and projects the link only into internal renderer input. No reviewed URL-to-claim status, error, log, name or annotation path was found. Intended provider ContactPoint spec is necessarily a durable control-plane resource, not a public consumer surface. W1 must preserve value-free errors and exclusive shape selection. The correction is not yet verified. |
| 8. One-claim-set proof and activation | Current AC5 test fails at W1. Its selected-type Integration, EscalationChain and regex policy setup can cover the four requested shapes after wiring, but does not prove provider behavior or bad-ID refusal. SlackChannel activation and the known contact renderer edit remain required; no additional renderer registration or activation requirement was identified from this slice. E3 and E4 need their own negative/regression evidence. |

B's proposed hook ordering is appropriate: calculate after successful render/withdrawal checks, then apply the unhealthy override after ordinary access/product readiness and observed-child readiness. It does not mutate child readiness. Both hook calls must share the same map. The packet says the map is the one passed to the renderer, but literal `config` differs in name from the renderer's `resolvedConfig`; its actual proposed code is consistent with itself because both new calls use `config` and the health helper does not consume resolved configuration. No blocking counterexample arises from that wording.

## Return and next action

Proven: source counterexamples E1-E4, exact released-default identity preservation, condition-only no-expiry behavior, the schema diff, and the two focused renderer failures. Not proven: corrections, integrated/hosted gates, final provider admission of the combined new shape, or published behavior. Live reconciliation is not exercised, and not exercisable here.

No source/tests/APIs/tracker/root-state/wiring files were edited, no corrections implemented, no agents spawned, no cluster tooling or Grafana Cloud contact used, and no commit, push, release, tag or PR performed.

Recommended next action: return B's accounting findings for its remaining bounded correction attempt; root must disposition D's exhausted-attempt findings under section 6.2 before further repair. Apply W1 only as authorized root wiring, with secrecy-preserving validation. Do not mark either task Done from the current candidate. This review consumes its one permitted pass; subsequent verification belongs to the root and the remaining prescribed gates.
