---
id: GCV-0076
title: >-
  A plugin request pinned to the newest version never converges, because the
  provider treats version as replace-only
status: In Progress
assignee: []
created_date: '2026-09-16 16:48'
updated_date: '2026-09-18 09:24'
labels: []
dependencies: []
type: bug
ordinal: 76000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
PluginInstallation carries `version` as a ForceNew argument. The composition function defaults it to the literal string "latest" when a request omits it (platform/function/plugins.go:16), and the provider records the concrete version it actually installed. The two never match, so the managed resource holds a permanent diff and the provider refuses the update for the same reason rotating tokens fail: it would require replacing the external resource.

Observed live on 2026-09-16: a plugin installation requesting "latest" against a recorded concrete version, Synced=False since the day it was created three weeks earlier, Ready=True throughout. It will never converge on its own and no later reconcile can fix it.

This is the default path rather than an unusual request, because a caller who names a plugin without naming a version gets "latest" from the function. Same ForceNew class as the rotating token failure, but a much smaller fix, so it is tracked separately rather than folded in.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A request that names a plugin without naming a version produces a managed resource that reaches Synced=True
- [ ] #2 The behaviour when a caller explicitly asks for the newest version is defined, and either converges or is rejected at admission rather than left permanently out of sync
- [ ] #3 A plugin installation already stuck in this state has a stated remediation an operator can follow
- [ ] #4 A function test covers the version-defaulting path and fails if the default reintroduces a value the provider cannot converge on
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Wave 13: write a failing version-defaulting test, choose the additive convergence behavior against the pinned CRD and released API, implement it, and record existing-object remediation.

Selected additive convergence path: omitted or explicit latest adopts the observed concrete installed version after first installation; an explicit numeric version remains pinned. The released API is not tightened. An already-stuck PluginInstallation must be deleted and recreated because its ForceNew diff cannot self-repair.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
2026-09-18 live re-confirmation. Still failing, still accumulating, and it is now the only rotating-or-ForceNew failure actually firing on either estate.

One PluginInstallation on the estate that vends plugins carries 7988 CannotUpdateExternalResource events, most recent at 2026-09-18T07:49:15Z, minutes before this note. The rotating-token failures that this task was filed alongside have been remediated by recreation and produce no events now, so this is the live instance of the ForceNew class rather than a historical one.

The defaulting line is platform/function/plugins.go:16, stringValue(plugin, 'version', 'latest'). Confirmed present at the current pin.

Same upstream mechanism as GCV-0075, verified from source: Upjet's Update path calls assertNoForceNew() in pkg/controller/external_tfpluginsdk.go and refuses before applying, so a ForceNew field whose stored value can never equal the requested one holds a permanent diff. Here 'latest' is the requested value and a concrete version is what the provider records, so the two can never converge and no reconcile will fix it.

AC3's remediation is already evidenced by the neighbouring task: deleting and recreating the managed resource is what cleared the equivalent state on the rotating tokens. State it explicitly as the operator step, because the object will not repair itself.

AC2 is the real decision: whether asking for the newest version is rejected at admission, or resolved to a concrete version at render time, or accepted with the churn documented. Rejecting it is the smallest change and the one that cannot be wrong later, but it removes a convenience a caller currently has - note that spec.repository style version pinning is not comparable here because this API is not released under the same constraint. Check whether the plugin API is already released before adding a rule that refuses an input it previously accepted.
<!-- SECTION:NOTES:END -->
