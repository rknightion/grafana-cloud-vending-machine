# Wave 14 root evidence

## Window and starting state

- Run census start: `2026-09-18T14:14:35Z`
- Integration base: `52308041d25f465e21e711add0dfcf7578f94866`
- Open release-please pull request observed: number 35, `chore: release 3.0.0`; no mutation authorised or attempted

## Function package pre-state witness

Outgoing function digest before any replacement: `sha256:5fa57e4288fa2b53a25585ae8a9b89f4d5ac84de03c0dac4f878cbdc300b7e9e`.

Occurrences to replace together after the exact source publish completes:

1. `platform/function/install.yaml`, Cosign verification argument
2. `platform/function/install.yaml`, Function package reference
3. `docs/installation.md`, pinned-version table and its inline package reference

The unrelated Cosign tool image digest in `platform/function/install.yaml` is outside the function pin.

## Root-judgement record

Expected rows so far: 7.

| Decision | Evidence | Alternatives rejected | Reversal cost | Root materiality |
|---|---|---|---|---|
| Serialize final package-level GREEN collection after shared-checkout writers stabilize | Lane C and then Lane E each hit compile errors from another lane's in-progress owned Go file; Go package tests compile the whole package even with `-run` | Editing another lane's WIP; counting environment interference as a failed attempt; adding late worktrees that would require transferring dirty untracked files | Low: final focused tests can run once the shared package compiles | Low scheduling repair; no product contract changed |
| Root rescue keeps Securevalue Delete policy armed across the `SecurevalueDeleting` witness | New assertion failed with `got ["Create","Observe","Update","LateInitialize"], want Delete to remain armed`; the corrected focused suite passed | Accepting a status-only witness while desired state reverted; sending a third worker correction after its two implementation cycles | Low: one condition in the terminating renderer and one regression assertion | Medium lifecycle correctness; reversing it reintroduces a use-after-policy race |
| Admit reserve R1 before the integrated gate | Lane B was under Lane F review; every committed lane had returned or was running with prerequisites met; the integrated gate had not begun | Leaving the stale GCV-0069 blocker untouched; admitting reserve work after the observable cutoff | Low: packet and tracker note only | Low campaign scheduling decision; produced a fit brief without source mutation |
| Transfer GCV-0075 repair ownership to root for attempt 4 of 4 | Lane B consumed its two worker attempts and returned partial; Lane F rejected all five standing findings with seven precise counterexamples; one authorized campaign attempt remained | A prohibited third Lane B attempt; accepting green helper tests over the security rejection; seeking an unauthorized fifth attempt | High: root integrated the five emit sites and repaired lifecycle, evidence binding, timing, deletion and naming contracts in one final attempt | High credential-availability and revocation safety |
| Park GCV-0075 and remove the rejected rotation source from the landing slice | Lane F's fresh review found seven remaining high-impact defects after attempt 4, including second-window blocking, pre-wiring consumer rejection and unsafe deletion readiness; all four attempts are consumed | A prohibited fifth attempt; landing known unsafe rotation logic; weakening the accepted packet | Medium: the rejected source was restored while retaining all packets and review evidence | High credential-availability and revocation safety |
| Fix two CodeRabbit findings and retain two reviewed contracts | The first pass found one search-error propagation defect, two duplicate non-renewal documentation findings, a request to neutralize the intentionally real profile, and a claim that the implemented lifecycle was only proposed; the scan fix and lifecycle source were verified, and the second pass returned zero findings | Ignoring the valid scan defect; replacing the commissioned real profile; weakening accurate lifecycle documentation | Low: one helper status check and one bounded README paragraph | Medium publication-gate reliability and operator safety |
| Complete GCV-0078 AC5 from public source evidence while deferring the estate-specific generator render to AC6 | The frozen and pinning revisions provide exact tree, manifest and catalogue hashes plus the complete schema delta; the estate-specific generator and overlay are deliberately absent from this public repository | Fabricating an overlay hash; contacting the live estate; leaving AC5 tied to a stale pre-wave revision | Low: the delta is a standalone evidence artifact and can be re-derived from the two revisions | Medium migration correctness; AC6 still requires the real rendered overlay and live pre-state |

## Integrated gate

- First invocation: stopped before validation because `KUBEBUILDER_ASSETS` was unset.
- Causal correction: set the repository's pinned envtest assets and retained `KUBECONFIG=/dev/null`.
- Corrected invocation: passed. The public scan, pinned Kubernetes 1.37 API-server admission suite, full Go test suite and 85.2% statement coverage completed successfully.

## Publication evidence

- Source revision: `7b3598efa68d980947e63cc01bc233ed74cafc7a`.
- Hosted source validation: run `35363219843`, success.
- Function publication: run `35363219707`, success.
- Published digest: `sha256:b59b3cded495d40983869fd926c04f776fe0b9e77d0312338529336627d0a41c`.
- Cosign verification: claims, transparency-log inclusion and certificate chain passed for the exact `publish-function.yml@refs/heads/main` identity.
- Pinning revision: `1f2af4a0b37c244a799e1fe0cdfcea2fa5abd82e`; all three references moved together.
- Hosted pin validation: run `35364595851`, success.
- Release pull request 35 remained open and received only the repository automation side effect from the authorized main pushes; the root did not open, edit, merge or close it.
- CodeRabbit was skipped for the digest-only and tracker/evidence-only commits as required by the review policy.
