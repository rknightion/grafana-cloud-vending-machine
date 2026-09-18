# Wave 14 reserve R1 triage - GCV-0069

## Scope and source identity

This packet is the reserve R1 re-triage required by wave 14 section 4.8. It reviewed the
committed working tree at the wave-14 base, including `platform/function/testdata/provider-crds.json`,
the provider manifest, the renderer and the existing GCV-0069 task. No live Grafana or Kubernetes
contact was made.

The pinned provider artifact is `v2.14.0` at the immutable package digest recorded in
`platform/provider/provider-grafana.yaml`. The fixture comment identifies the imported package
source revision as `dc795606df97a72dce81a0c953e0ec0750e0b489`. The fixture contains 46 CRDs,
including the 42 pre-existing entries described by the Wave operating model. The provenance
contract is semantic agreement with the pinned package, allowing the generated empty top-level
`status` stanza; 12 of those 42 entries carry that stanza and 30 do not. The old byte-equality
control against one neighbour was therefore unsatisfiable by construction and did not establish a
fixture-provenance failure.

## GVKs re-derived from the pinned artifact

The following values are read from `platform/function/testdata/provider-crds.json`; none is copied
from the task's frozen wording.

| CRD name | Group / version | Kind | Plural | Scope |
| --- | --- | --- | --- | --- |
| `integrations.oncall.grafana.m.crossplane.io` | `oncall.grafana.m.crossplane.io/v1alpha1` | `Integration` | `integrations` | Namespaced |
| `escalationchains.oncall.grafana.m.crossplane.io` | `oncall.grafana.m.crossplane.io/v1alpha1` | `EscalationChain` | `escalationchains` | Namespaced |
| `routes.oncall.grafana.m.crossplane.io` | `oncall.grafana.m.crossplane.io/v1alpha1` | `Route` | `routes` | Namespaced |
| `slackchannels.oncall.grafana.o.crossplane.io` | `oncall.grafana.o.crossplane.io/v1alpha1` | `SlackChannel` | `slackchannels` | Namespaced |
| `contactpoints.alerting.grafana.m.crossplane.io` | `alerting.grafana.m.crossplane.io/v1alpha1` | `ContactPoint` | `contactpoints` | Namespaced |
| `notificationpolicies.alerting.grafana.m.crossplane.io` | `alerting.grafana.m.crossplane.io/v1alpha1` | `NotificationPolicy` | `notificationpolicies` | Namespaced |

The artifact also confirms the spelling and version of the adjacent secure-value kind, if a
future implementation packet needs it: `SecurevalueV1Beta1` (lowercase `v`),
`enterprise.grafana.m.crossplane.io/v1alpha1`, plural `securevaluev1beta1s`, Namespaced.

## Acceptance criteria disposition

All five criteria are implementable. The stale wave-10 fixture blocker is refuted; the remaining
work is ordinary API, renderer, reference-resolution, activation and integrated-test work.

1. **AC1 - fit.** `Integration.spec.forProvider.type` is a string and its pinned schema
   explicitly includes `grafana_alerting` among the supported values. The implementation needs an
   integration-type field and validation on the composite, then must pass that value through the
   renderer in place of the current inbound-email literal. The claim and rendered Integration
   must be tested for both the existing default and `grafana_alerting`.
2. **AC2 - fit, with the recorded amendment.** `Route.spec.forProvider.slack` exposes
   `slackChannelRef` (a required-resolution `SlackChannel` reference) and the opaque `channelId`
   escape hatch. The amendment remains binding: accept exactly one of `channelRef.name` resolved
   through an observe-only `SlackChannel`, or `channelId` as an opaque id. A display name must
   never be sent as `channelId`. The implementation needs the composite field, the observe-only
   child with this artifact's `SlackChannel` GVK, and a renderer test proving the resolved id is
   placed on the Route.
3. **AC3 - fit.** The pinned Route schema gives `slackChannelRef` a `Required` resolution policy
   by default. The composition must preserve that policy and fail readiness loudly when the
   referenced SlackChannel cannot be resolved, with a negative renderer or admission test. No
   route with an absent destination is acceptable.
4. **AC4 - fit.** `ContactPoint.spec.forProvider.oncall` contains `oncallIntegrationRef` with
   required reference resolution and a `url` field. The pinned Integration status carries the
   provider-observed `link` value used by the reference mapping. The implementation must resolve
   that value inside the control plane and render an IRM-typed contact point without accepting a
   consumer-supplied bearer URL. Tests must cover missing and observed Integration state. Live
   read-back is not exercised, and not exercisable here.
5. **AC5 - fit.** One claim set can prove the four linked shapes: Integration (including its
   selected type), EscalationChain, Route (including regex routing and the Slack destination),
   and Alerting ContactPoint plus NotificationPolicy. The implementation needs the cross-resource
   activation/mapping entries, renderer coverage for the combined observed transitions, and an
   integrated admission/render test. The pinned schemas expose the needed `EscalationChain`,
   `Route.routingType`/`routingRegex`, `ContactPoint.oncall`, and `NotificationPolicy` fields.

## Fit brief and remaining boundary

GCV-0069 is fit to commission as an implementation lane. The route/reference work must preserve
required resolution and the AC2 exactly-one rule. The task's existing AC2 amendment is still
correct and should remain in the tracker. The package artifact and fixture provenance are no
longer a blocker. The remaining boundary is implementation plus the normal local and hosted gates;
those are outside this reserve mapping packet.

This lane did not edit source, tests, schemas, activation registries, catalogs or documentation.
It did not contact Grafana Cloud or a cluster, change task status, publish a package, create a
release or tag, open or merge a pull request, or run a live behavioural proof.
