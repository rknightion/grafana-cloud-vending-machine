---
id: GCV-0075
title: >-
  Vended rotating tokens can never rotate, so every stack loses its credentials
  at the first rotation window
status: To Do
assignee: []
created_date: '2026-09-16 16:48'
labels: []
dependencies: []
references:
  - >-
    https://github.com/grafana/terraform-provider-grafana/blob/main/internal/resources/cloud/resource_cloud_stack_service_account_rotating_token.go
  - >-
    https://github.com/grafana/terraform-provider-grafana/blob/main/internal/resources/cloud/resource_cloud_access_policy_rotating_token.go
priority: high
type: bug
ordinal: 75000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Every stack this platform vends gets a StackServiceAccountRotatingToken, and any stack with telemetry or fleet access gets AccessPolicyRotatingToken as well. Neither can ever rotate under Crossplane, so both eventually expire and are never replaced.

The upstream resources implement rotation as a Terraform replace. `ready_for_rotation` is declared Computed and ForceNew, and a CustomizeDiff sets it to true once the clock passes `expiration - early_rotation_window`. Terraform reacts by destroying and recreating the token, and the replacement IS the rotation. Upjet never replaces a managed resource; it refuses the update and reports CannotUpdateExternalResource with 'refuse to update the external resource because the following update requires replacing it: cannot change the value of the argument "ready_for_rotation" from "" to "true"'. The field is provider-internal and absent from the CRD, so nothing in this repository can set, suppress or patch it.

Observed live on 2026-09-16 across two estates running this platform. On the first estate both rotating tokens went Synced=False the minute their early-rotation window opened, four days before expiry, and had accumulated 5610 CannotUpdateExternalResource events by the time this was found. The second estate has not reached its window yet and is on a newer provider with an identical CRD schema, so it is expected to fail the same way about two weeks later.

Nothing surfaces this. The composite reports Synced=True and Ready=True, and each stuck token reports Ready=True. Only the child's Synced condition carries the failure, so a stack looks healthy right up to the moment its credentials stop working.

The consequence is the credential handover this platform is built on. The administrator token feeds the per-stack ProviderConfig through PushSecret, the secret store and ExternalSecret, so when it expires every in-stack resource stops reconciling. The telemetry and fleet tokens are what collectors authenticate with, so those expiring stops ingestion.

This needs a decision, not a patch, and every option is unattractive: delete and recreate each token out of band before its window opens, lengthen the lifetime so the window is never reached in practice and accept weaker hygiene, or get the behaviour changed upstream. A future agent cannot recover any of this from the code, because the repository only ever sets expireAfter, secondsToLive and the early rotation window.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 The failure is reproduced against the provider version this repository pins, with the resource kinds, the exact provider message and the timing relative to the early-rotation window recorded as evidence
- [ ] #2 The chosen handling of rotation is recorded as a decision with its trade-offs, covering both rotating token kinds the composition emits
- [ ] #3 The chosen handling works for an estate that has already passed a rotation window, not only for a stack vended after the change
- [ ] #4 An operator can distinguish a stuck token from a healthy one from resource conditions alone, without reading provider logs
- [ ] #5 The composite no longer reports Ready=True while a token it owns cannot rotate, or the reason it still does is recorded
- [ ] #6 The upstream position is recorded: whether a provider or Upjet change is required, and whether it has been raised
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
