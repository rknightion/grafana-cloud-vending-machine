# Lane E documentation packet

This packet is not an edit to public documentation. Apply it only after the
documentation owner is terminal.

## Target: `docs/troubleshooting.md`, anchor: `## Known limitations`

Add a subsection headed `### Permanently refused provider request shapes`.

A provider child rejected permanently with a 4xx does not become valid through
another reconcile. Crossplane keeps reconciling the desired child, so the
refusal repeats indefinitely and creates continuous load on the vendor tenant.
Inspect the child resource's `Synced` and `Ready` conditions and its events;
the composite's conditions summarize the composition and can remain healthy
while one child is permanently refused. Do not wait for retries to repair a
known invalid shape. Correct the rejected fields and reapply the claim. Use the
reviewed controlled-deletion path only if correction cannot withdraw the child,
then confirm the child is gone and the provider calls have stopped.

Never probe a vendor request shape on a stack that anything depends on. The
vendor secret API ignores `dryRun`; two objects submitted with `dryRun=All`
persisted as real resources and had to be removed. Assume every vendor API call
mutates. Use a disposable stack, record every object created during the probe,
inspect the result and conditions, and remove all probe residue before using the
finding elsewhere. Kubernetes server-side dry run and the vendor secret API
have different semantics; the former does not make the latter safe.

## Target: `docs/reference/request-schema.md`, anchor: `## Status conditions`

After the existing status-condition paragraph, add one sentence: `Composite
conditions are not a substitute for each composed child's conditions: diagnose
a provider refusal on the child object and its events, because a composite can
remain healthy while a child is retried.`

## GCV-0082 AC6 decision record

Keep the connection renderer's standalone secure value. It deliberately
duplicates the credential inside the vendor because the expected restoration of
the secure-value reference form is tracked separately; retaining the child
makes that restoration a one-line renderer change instead of a redesign. This
packet records no migration because the rendered behaviour does not change.
