# Wave 13 Lane F security review

Verdict: **material A2 flaws exist; the credential mechanism must not land in
this state.** D also has a material admission-contract gap for previously stored
secure fields. Its renderer refusal is intact. This is a source review of the
uncommitted changes over `e91d3fe2d7600aa1369593ddec349495014272d6`, not approval
of a completing SHA or of subsequent root wiring.

## Findings

1. **Material, A2: publication cannot advance, and an observed candidate is
   withdrawn.** `platform/function/access.go:304-321` renders the successor only
   when it is not observed. On the following invocation the existing-successor
   branch records names but never renders that successor. Lines 337-364 render
   only the predecessor while waiting. A fresh desired map therefore omits the
   pending candidate, including a failed candidate. Separately, lines 342-350
   select the successor only after its publication acknowledgment, while lines
   358-361 restore the old observed selector. Starting with the old selector,
   no invocation asks PushSecret to publish the new Secret. This is a deadlock
   that leaves consumers on the expiring predecessor; omission also makes the
   candidate's lifetime depend on incidental incoming desired state. The test at
   `access_test.go:221` manually changes the observed selector to the successor,
   skipping the missing transition, and does not assert candidate retention.
   Required correction: render every retained member on every invocation;
   explicitly select a readable successor for publication before acknowledgment;
   test the full sequence with each observed selector derived from the preceding
   desired response and a fresh desired map.

2. **Material, A2: malformed lineage can withdraw the serving generation.**
   `access.go:396-410` returns no member inventory for malformed annotations or
   duplicate ordinals. The blocked branch at lines 200-206 then emits only the
   legacy member and returns the legacy Secret selection. If a generation is
   already serving, this drops it and rolls the publisher back, potentially to
   an expired legacy credential. Under an armed Delete lifecycle an omitted
   serving MR can also revoke that credential. Missing-clock handling at
   lines 222-228 similarly retains only one member. This contradicts the
   accepted invariant that ambiguous evidence preserves every observed member
   and the durable selector. The review cannot assert that failed replacement
   leaves serving credentials intact across all paths.

3. **Material, A2: consumer handover compares the wrong ProviderConfig identity.**
   `access.go:614` compares the consuming ProviderConfig name to
   `family.ProviderConfigRef.name`, which is the token's minting provider.
   The administrator descriptor at `fn.go:643` and fleet descriptor in
   `fleet.go` use the organization provider. The actual consuming ProviderConfig
   at `fn.go:649-655` is the stack provider. A correctly identity-validated
   consumer observation therefore cannot pass. The helper test gives both roles
   the same name, concealing the mismatch. Use separate explicit minting and
   consumer identities and exercise distinct names. The helper also has no
   active stack-status-reference check before retirement, required by the packet
   for downstream bootstrap consumers; that must be satisfied by integration.

4. **Material, A2: existing creation parameters and unknown timing are not
   preserved honestly.** `access.go:647-676` rebuilds each observed member from
   current family settings rather than retaining its immutable creation
   parameters, provider binding and deletion settings. The fallback expiry at
   lines 517-525 uses the current family lifetime/window rather than the observed
   token spec. Increasing a platform lifetime can therefore postpone replacement
   beyond an existing access token's actual expiry. Missing creation time returns
   Healthy at lines 520-521, and the successor validator accepts that result at
   lines 568-571. Unknown validity is treated as valid. Required correction:
   schedule from observed immutable parameters, preserve those parameters on
   existing generations, and block unknown or unverifiable timing visibly.

5. **Material integration gap, not a separate A2 ownership accusation: composite
   state and retirement are not yet wired.** The inspected `fn.go:RunFunction`
   has no rotation Secret collection, explicit retired-key pruning, or final
   `CredentialRotationHealthy`/Ready override. `access.go:177-183` records a
   private summary but discards `RetiredKeys` and does not retain deadlines or
   estimated-timing provenance. The family deletion helper at lines 850-873
   requires the synthetic legacy member to exist even after legitimate promotion,
   and does not check unresolved handover. Its use in `expiry.go:169-172` can
   permanently block deletion after legacy retirement, or call deletion prepared
   during unresolved handover when all members are armed. No final composite
   honesty or lifecycle verdict can pass until these seams are integrated and
   tested. Routine root wiring alone does not repair findings 1-4.

6. **Material, D: static CEL refusal does not reject all retained-field updates.**
   `platform/apis/provisioning-v1beta1.yaml:235` uses `false` without `oldSelf`.
   Kubernetes validation ratcheting accepts unchanged invalid subfields on
   updates; non-transition CEL validations participate. See the official
   [validation ratcheting documentation](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#validation-ratcheting).
   The harness pins Kubernetes 1.37.0, where this behavior applies. The new test
   at `provisioning_test.go:332` creates an initially valid object and then adds
   each forbidden field; it does not persist a formerly accepted value, upgrade
   the CRD, and update another field while retaining that value. Therefore the
   migration packet's statement that any update still containing a field is
   rejected is incorrect for this case. Required correction: an admission rule
   that also rejects unchanged pre-existing values, with a real API-server
   upgrade test for all three aliases and a successful field-removal migration.
   Keep the independent renderer refusal.

## D checks that passed source review

The YAML anchor at lines 229-248 applies the same refusal to token, webhookSecret
and commitSigningKey, retaining the field definitions and existing create guard.
`provisioningSecureValues` checks presence before inspecting value shape, so name,
create, empty and malformed values cannot pass through the three supported keys.
The direct repository renderer and internal ladder both use that function;
the repository renderer is the sole constructor of RepositoryV0Alpha1. No second
path emitting the same top-level provider secure fields was found. Malformed
non-map secure input is ignored rather than forwarded. The error and migration
packet correctly qualify the vendor defect as inferred from Connection.

The migration instructions specify removing all three fields, retaining the
Connection reference, and reapplying without deleting the repository. They are
actionable once the retained-update statement is made true and tested. Breaking
commit metadata remains a root publication responsibility, not yet verified.

## Evidence and limits

The focused existing tests passed with `KUBECONFIG=/dev/null`:

```text
just test 'TestRotatingTokenFamilyRetainsServingGenerationUntilHandover|TestProvisioningRepositoryRendererRefusesEverySecureValuePath'
ok  github.com/rknightion/grafana-cloud-vending-machine/platform/function (cached) coverage: 7.1% of statements
```

This proves the existing assertions only. No adversarial test was added because
Lane F owns only this review file. No integrated gate or admission migration
experiment was run by F. Root reports D's ephemeral create and field-addition
update tests passed; retained-field migration remains outside that evidence.
No live Grafana or Kubernetes contact occurred. Live behavior is not exercised,
and not exercisable here.

Reviewed content hashes (SHA-256):

```text
7b1bdd00ab1e3c631316915ce01de01252bf2bb0602212f1dd07c9a5bee72871  platform/function/access.go
35d772b1faba3ce7d7b6857b609c902a5eae125c2d50222fb38d48ddd640dd6d  platform/function/fn.go
ea037101b23bb635006e4e1e2fcfb0f3c6414434af02b3f635e22377ec931418  platform/function/provisioning.go
d1fa3bf1e1165a83b0b7420ce9f2f4a9c001905aa11535b235f7303b1e445f9f  platform/apis/provisioning-v1beta1.yaml
```

Only `codex/wave13/lane-f-review.md` was written. No implementation, tracker,
index, commit, publication or external resource was changed by this review.
Root owns disposition under sections 6.8 and 8, with existing attempt accounting
preserved. No material finding is waived by these passing focused tests.
