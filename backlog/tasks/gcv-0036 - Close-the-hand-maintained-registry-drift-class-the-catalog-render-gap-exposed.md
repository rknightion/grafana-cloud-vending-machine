---
id: GCV-0036
title: Close the hand-maintained registry drift class the catalog render gap exposed
status: Done
assignee: []
created_date: '2026-09-08 17:02'
updated_date: '2026-09-08 18:39'
labels: []
dependencies: []
ordinal: 36000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Four catalog directories shipped with no kustomization.yaml and failed to render while the gate stayed green, because scripts/validate.sh listed fourteen render paths by hand and there were eighteen directories. That was fixed in 28b4832de3bfafd5e3c0faa3a2bff31dc4bffbad by enumerating instead. The same shape exists elsewhere and has not been checked: platform/kustomization.yaml, the ManagedResourceActivationPolicy kind list and platform/rbac/composition-rbac.yaml are all append-only registries maintained by hand, and the wave operating model already names them as pre-assigned wiring-pass entries precisely because nothing derives them. A registry that silently stops covering a new resource is the same defect that shipped once with examples/catalog/minimal and again with these four, so the question is which of the remaining lists can drift and what would be observable when one does.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Every hand-maintained registry in the repository is inventoried, each recorded as either derivable-and-now-asserted or as deliberately manual with the reason
- [x] #2 For each registry that can drift, the gate asserts its coverage against the thing it is supposed to enumerate
- [x] #3 Each new assertion is proven by a negative control: an entry is removed, the gate fails naming it, and the removal is reverted
- [x] #4 A registry left deliberately manual carries a comment at its definition saying so, so the next agent does not silently automate it
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 just check passes locally
- [x] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 4: implement the commissioned lane after the pushed root harness pre-pass; preserve frozen schemas and ownership; return acceptance evidence and required negative controls; root integrates, reviews, validates locally and at the exact hosted SHA, then reconciles status.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Wave 4 registry implementation delivered at 9c559d105c5cd7761db3c9ca290150d35c936175; hosted Validate 34263352393 success, local just check passed at 85.4% coverage.

| Registry | Decision and evidence |
|---|---|
| Platform Kustomization resources | Recursively enumerate platform YAML/YML; exclude only the Kustomization itself and OCI package metadata; missing entries fail by path. |
| XRD composite kinds and renderer registry | Compare every XRD YAML document with registry keys: 14 kinds from 13 API paths, including both access APIs. |
| Catalog bases and sibling resources | Enumerate all 18 directories, require README and Kustomization, require every sibling manifest, render every directory. |
| Signed package/verifier pairs | Discover every Function/Provider install under platform, require one paired verification Job and exact digest equality. |
| ManagedResourceActivationPolicy | Selection stays deliberately manual. Coverage derives 54 emitted provider-managed GVKs from actual Go constructors and resolves them against a compact map verified from all 297 CRDs in the exact pinned provider artifact. Current 55 activations retain the separately operated Connection resource. |
| Inventory kind/plural table | Existing actual Go table contributes the full dynamic input set; pinned-provider mapping verifies the plural for each emitted kind. The shell checker independently checks the eight explicit inventory plurals. |
| Composition support RBAC | Deliberately manual and commented at definition: this role grants External Secrets support access, whereas provider controller access is supplied through provider CRDs. Rendered GVKs do not mechanically define this role. |
| AWS base resource selection | Deliberately manual and commented at definition: optional profile-secret manifests are opt-in, not automatic sibling resources. |
| ApplicationSet directories | Existing exact policy assertion remains `[{path: enabled/*}]`; no other watch path is introduced and enabled stays empty. |

The compact CRD map is an extracted provider schema artifact, not an independently maintained list of what the renderer emits. The Go census derives the latter and requires exact set agreement. A changed provider pin fails until the map is regenerated from its corresponding artifact; no network request is hidden inside the gate.

The production census is 65 calls to newDesired, yielding 57 unique GVKs: 54 provider-managed targets and three explicit exclusions (ProviderConfig, ExternalSecret and PushSecret). A separate direct core Secret constructor is verified and excluded with its reason. Dynamic kind sources use the actual inventory tables, product table and content-access enum; unresolved forms fail by file and line. The checker tolerates unrelated source-line shifts and rejects non-call newDesired references and unsupported direct DesiredComposed literals.

Root independently compared every one of the 54 compact-map entries to the exact retained provider artifact: all matched. The one retained activation beyond those 54 is the separately configured Connection resource.

Negative controls are preserved in codex/wave4/C-negative-controls.log, C3-* logs and C-root-* logs. They cover missing platform/API/catalog entries, missing verifier, missing inventory and k6 activation, provider-map pin mismatch, constructor alias/direct literal/package initializer escapes, and malformed resources or missing source-file diagnostics. Each control fails by the affected path/kind and the restored sources pass. CodeRabbit returned four minor findings: three fixed for clear failures or unused code; the read-only Close return wrapper was left because it changes no behavior or configured gate.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Closed the registry drift class with recursive manifest/catalog discovery, XRD/renderer agreement, discovered package/verifier matching, and a source-derived census of 65 constructor calls / 54 provider-managed GVKs verified against the exact pinned CRD artifact. Manual selection boundaries are commented. Removed-entry, pin-mismatch and unsupported-constructor controls fail by path/kind; restored checks pass. Completing checkpoint SHA 5c482ffcf2100714cfe1c3751733d805a501489a; hosted Validate 34263860085 success. Implementation SHA 9c559d105c5cd7761db3c9ca290150d35c936175; hosted Validate 34263352393 success. Local just check passed with 85.4% coverage. Main still has zero real admission cases and one skipped placeholder; these tasks do not claim admission completion.
<!-- SECTION:FINAL_SUMMARY:END -->
