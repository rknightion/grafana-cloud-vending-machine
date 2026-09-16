---
id: GCV-0076
title: >-
  A plugin request pinned to the newest version never converges, because the
  provider treats version as replace-only
status: To Do
assignee: []
created_date: '2026-09-16 16:48'
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
