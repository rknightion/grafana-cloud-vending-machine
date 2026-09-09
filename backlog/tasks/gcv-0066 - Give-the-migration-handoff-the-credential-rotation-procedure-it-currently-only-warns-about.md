---
id: GCV-0066
title: >-
  Give the migration handoff the credential rotation procedure it currently only
  warns about
status: In Progress
assignee:
  - '@claude'
created_date: '2026-09-09 21:28'
updated_date: '2026-09-09 22:42'
labels: []
dependencies: []
ordinal: 66000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The migration guide tells the operator that credentials retained under non-deleting management policies need a rotation or revocation plan after the target cluster takes ownership, and then stops. There is no procedure. That is the point in the migration where the source cluster still holds live credentials it no longer owns and nothing tells the operator what to do with them, which is the highest-consequence gap left in the handoff: an unrevoked retained access policy outlives the cluster that vended it. The plan has to work for the retained policy, its rotating token and any delivered output secret, and it has to say what order those come apart in relative to the target cluster proving it has minted its own.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The migration guide carries an ordered procedure for retiring source-held credentials after target ownership is proven
- [ ] #2 The procedure names each retained object class it covers and states what evidence proves the target has replaced it before anything is revoked
- [ ] #3 The procedure states the failure mode of revoking too early and of never revoking at all
- [ ] #4 Any step that cannot be verified from this repository is marked as an operator prerequisite rather than presented as proven
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Read the ownership-transfer procedure and identify every credential class the source still holds once the target owns the stack.
2. Write the retirement procedure as an ordered subsection of that transfer section, and make step 6 point at it instead of only warning.
3. State both failure modes explicitly, since revoking too early and never revoking are opposite errors with the same root cause.
4. Mark every step this repository cannot verify as an operator prerequisite rather than presenting it as proven.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
CodeRabbit review record for this procedure. It went through ten passes, which is far past the usual ceiling; the reason is worth recording because it is a property of the work and not of the tool. The procedure is a credential-destruction runbook written from scratch, so almost every finding was of the form "this instruction destroys something under a case you did not name". Those are exactly the findings worth taking, and each fix lengthened the prose the next pass then read.

Command each time: `coderabbit review --agent --base main`. Every pass emitted a `complete` event.

Severity totals across all passes: 0 critical, 24 major, 4 minor. Applied 25; not applied 3, each recorded below with its reason.

The substantive defects it caught, all of which were mine and all of which were real:

- Step 4 deleted the remote secret-store document unconditionally, while step 7 of the transfer procedure above explicitly permits the target to keep the source output path. On that path the instruction deleted the target live credential document. This was the worst one.
- The `PushSecret` deletion-policy trap: this repository renders `deletionPolicy: Delete` whenever deletion is armed (platform/function/access.go:47-51), so removing the writer takes the remote document with it. The check has to happen during inventory, because step 4 of the transfer procedure removes output-document writers before the retirement section is reached.
- Consumer evidence: reading the target path is a configuration fact. A process that loaded the source token at start-up keeps presenting it, and keeps succeeding, right up to revocation. A post-write successful call proves nothing on its own either.
- Plurality throughout. One policy can have several rotating tokens, one service account several tokens, and a token issued out of band never appears in the source cluster at all. Every comparison is set against set.
- `remoteKey` is unique only within its store, so the ownership comparison is on the (store identity, key) pair.
- ESO `PushSecret` reports `Ready=True` with reason `Synced`, which is not the Crossplane Ready/Synced pair used elsewhere in this guide, and neither the remote document nor the source Secret carries conditions at all.
- An adopted policy still leaves its tokens and delivered copies to retire; the earlier text said there was nothing to retire.

Not applied, with reasons:

1. Replace the documented key formats (`region:tokenID`, `stackSlug:serviceAccountID`) with a general instruction to read the external-name annotation. The formats are correct, verified against pinned provider source, and this guide already documents them in the adoption section. The inventory instruction already says to read the actual annotations rather than construct keys, so this would remove accurate information without changing any action.
2. Per-executable checksums for the envtest binaries (raised on GCV-0065). Upstream publishes one archive sidecar and no per-binary digests; the archive hash already fail-closes on any tampered member.
3. An etcd version assertion in the cached-asset reuse path (GCV-0065). The justfile pins the Kubernetes version only; there is no etcd pin to assert against and inventing one would be a fabricated control.
<!-- SECTION:NOTES:END -->
