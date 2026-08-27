---
id: GCV-0010
title: >-
  Bump the provider pin to the release carrying Agent Observability and the new
  App Platform kinds
status: To Do
assignee: []
created_date: '2026-08-21 12:13'
updated_date: '2026-08-27 09:26'
labels: []
dependencies: []
references:
  - 'https://github.com/grafana/crossplane-provider-grafana/pull/658'
ordinal: 10000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The pinned Grafana Crossplane provider (v2.13.0) is generated against Terraform provider v4.40.1 and exposes 111 of the 121 upstream resources. The ten absent resources are the five Agent Observability kinds plus Alerting RoutingtreeV1Beta1 and RulesequenceV0Alpha1, oss QueryV1, cloud ComponentV1Alpha1, and assistant TermsAcceptance.

Cause: the provider's generator panics on an upstream resource category it does not recognise, and 'Agent Observability' was added upstream in Terraform provider v4.43.0 without a matching entry. An upstream PR adds that entry and regenerates; it is green but awaiting review. Once a provider release carries it, this repository moves its pin.

Moving the pin is not just a digest change. It touches the cosign verification identity, the Provider package reference, and the ManagedResourceActivationPolicy, which currently activates 19 kinds by name and must gain any newly emitted kind. The README provider-family table is the extension index and records the ownership decision per family, so a new family needs a row.

Do not activate kinds speculatively. Activation is driven by what the Compositions actually emit.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 The provider package digest and the cosign --certificate-identity tag are updated together and the signature verification job passes
- [x] #2 ManagedResourceActivationPolicy activates only kinds that a Composition emits after this change
- [x] #3 The README provider-family table gains a row for the agento11y family stating its ownership decision
- [x] #4 Known limitations records the provider version and the resource-count parity reached
- [ ] #5 ./scripts/validate.sh passes and the hosted validation run is green on the completing commit
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 ./scripts/validate.sh passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
## Pinned to a main-branch build rather than a release

The release that GCV-0010 was waiting for does not exist. PR 658, the reference on this task, was
closed; PR 654 superseded it, merged 2026-08-25, and moved the provider to Terraform provider 4.45.1
with the generator entry that was missing. config/groups.go on main now carries
`"Agent Observability": {"agento11y", 1}`. The newest tag is still v2.13.0 from 2026-07-03, and
releases are cut only on a v* tag push, so a release is a human action nobody has taken.

Decision, taken by the repository owner: pin the main build now and let Renovate track it, rather
than wait for a tag.

### Why that is not a downgrade of the supply-chain control, verified live

ci_tag.yaml runs on `push` to `main` and `release-*` as well as on a v* tag. It publishes to
xpkg.upbound.io and ghcr.io and cosign-signs both, so a main build is published and signed by the
same workflow file as a release. Verified against the registries, not inferred:

- xpkg carries one tag per main commit, `v2.13.0-1.g…` through `v2.13.0-13.gdc79560`, matching main's
  commit sequence. The pinned build is main HEAD, digest
  `sha256:3f442e480da05f80be443663976cfd433d33d1d4acd30653a71b6be6eb157766`, byte-identical to the
  GHCR copy.
- `cosign v3.1.2 verify --certificate-identity=…ci_tag.yaml@refs/heads/main` against that digest
  passes: claims validated, transparency-log entry verified, certificate chain verified. Run locally
  with the same pinned cosign image the PreSync Job uses.
- The signing certificate SAN is
  `https://github.com/grafana/crossplane-provider-grafana/.github/workflows/ci_tag.yaml@refs/heads/main`,
  OIDC issuer `https://token.actions.githubusercontent.com`, trigger `push`.

The identity is branch-scoped, so any main build satisfies it and the digest is the real control.
That is the same posture this repository already uses for its own function package, which verifies
against `publish-function.yml@refs/heads/main`.

Signatures are stored as OCI referrers under the fallback tag `sha256-<digest>`, not as a
`sha256-<digest>.sig` tag. A `.sig` lookup 404s on both registries for the release digest too, so an
absent `.sig` is not evidence of an unsigned package. xpkg needs a bearer token even for anonymous
pulls, from `https://xpkg.upbound.io/service/token`, and exposes no referrers API.

### The reference now carries a tag as well as a digest

`xpkg.upbound.io/grafana/provider-grafana:v2.13.0-13.gdc79560@sha256:3f442e48…` in both spec.package
and the verification argv. The registry publishes no floating tag, no `latest` and no `main`, both
404, so a digest-only reference gives Renovate no version to compare and it cannot see a newer build.
cosign resolves the digest and ignores the tag; verified with both reference forms.

renovate.json gains regex versioning on the provider rule:
`^v(?<major>\d+)\.(?<minor>\d+)\.(?<patch>\d+)(?:-(?<build>\d+)\.g[0-9a-f]+)?$`. Default docker
versioning reads the git-describe suffix as a prerelease of the release it is ahead of and would
offer v2.13.0 as an upgrade. The build number orders main builds monotonically, and a real v2.14.0
still wins on minor because its release array is shorter and higher. Do not capture the short sha as
`revision`: regex versioning runs `Number.parseInt(revision, 10)` on it, which is NaN for any sha
starting with a letter.

### Provider-family parity

121 namespaced kinds across 17 families, up from 111 across 16; agento11y is the new family. 50
observe-only kinds. The ten previously absent resources are all present: the five agento11y kinds,
alerting RoutingtreeV1Beta1 and RulesequenceV0Alpha1, oss QueryV1, cloud ComponentV1Alpha1, and
assistant TermsAcceptance.

### CRD diff across the 19 activated kinds

Sixteen are byte-identical, Role included, so the autoIncrementVersion initializer workaround still
applies. Three changed:

- Team gained an `admins` set and `members` narrowed to ordinary membership. Fixed in this change:
  GrafanaTeamAccess gains `spec.team.administrators`, rendered as `admins` only when the request
  declares the field. Emitting an empty set demotes every administrator including UI-created ones, so
  an omitted field must not render the key. Test pins all three cases.
- AccessPolicy realm became a Block List where it was a Block Set. The function emits one realm entry,
  so ordering is not load-bearing here.
- Stack status gained per-service `*AllowlistUrl` fields, purely additive. These are endpoint
  references for retrieving source IPs to allow, not inbound filtering; noted on GCV-0014.

No new kind is activated: the activation policy still lists exactly what the Compositions emit.

### Upstream behaviour changes across 4.40.1 to 4.45.1 that do not affect this reference

grafana_user update skipping, appplatform folder-uid import, k6 publisher_token added then removed,
oncall integration inbound_email, saved queries, servicemodel components. `secret_manager_enabled`
reached no CRD in the generated provider. Fleet Management pipelines gained double canonicalization of
Alloy configs, which matters to GCV-0020 rather than here.

### Evidence

Local `./scripts/validate.sh` green: public-release scan, gofmt, go mod tidy, race tests at 90.7%
coverage, vet, YAML parse, four Kustomize renders, example READMEs, ApplicationSet watch path.
CodeRabbit review raised two major findings, both fixed: the stale "exact tag workflow identity"
wording in docs/security.md and the `revision` capture group above.
<!-- SECTION:NOTES:END -->
