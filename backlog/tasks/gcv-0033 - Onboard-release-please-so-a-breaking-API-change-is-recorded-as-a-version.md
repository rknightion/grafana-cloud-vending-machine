---
id: GCV-0033
title: Onboard release-please so a breaking API change is recorded as a version
status: To Do
assignee: []
created_date: '2026-09-08 08:08'
updated_date: '2026-09-08 08:18'
labels: []
dependencies: []
priority: medium
type: chore
ordinal: 33000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The repository has no tags, no release workflow and no version recorded anywhere, so there is no way to say which commit a consumer is pinned to and no way to signal a breaking change. Consumers pin platform/ by commit SHA, which works but carries no compatibility meaning.

Making spec.organization a required field on the served v1beta1 API is a breaking change to every existing stack request, and the decision to take it in place rather than adding a v1beta2 was made on the understanding that the break would at least be recorded as a release.

Follow the OpenBao CI-secrets runbook rather than provisioning a PAT: the release workflow mints a short-lived, repo-scoped GitHub App installation token through the shared broker-token action against a per-repo permission set. RELEASE_PLEASE_TOKEN must never be provisioned. Keep the workflow self-contained.

The OpenBao permission set, policy and JWT role are an external secret-store mutation and are provisioned outside this repository before the workflow can succeed first time.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A self-contained release-please workflow runs on pushes to main and mints its token through the shared broker-token action against a per-repo permission set, with no PAT
- [ ] #2 Release-please config and manifest exist and are seeded at a version that makes the required-field change a major bump
- [ ] #3 Commit-message conventions the release notes depend on are stated in the contributor instructions
- [ ] #4 The workflow is validated without a live run, and the first real run is recorded as evidence separately
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
OpenBao side provisioned and read back 2026-09-08, before any wave work. All three objects are live and
structurally identical to the working autopi-ha consumer, which is the reference to copy in the
workflow:

- permission set release-please-grafana-cloud-vending-machine: installation 152037622 (rknightion),
  repositories scoped to this repository alone, permissions contents=write and pull_requests=write.
- policy gha-release-please-grafana-cloud-vending-machine: create/read/update on that one token path
  and nothing else. No token wildcard.
- JWT role of the same name, binding repository owner id, this repository id, ref refs/heads/main and
  runner_environment github-hosted, with a 5 minute TTL and 3 uses.

Because the permission set and the role share one name, the workflow may default the role and needs
only the permission-set input. Passing only permission-set when the two names diverge is what killed
the first docs-sync rollout with a 400; it is safe here precisely because they match.

TS_WIF_CLIENT_ID and TS_WIF_AUDIENCE already exist as repository secrets and are identifiers rather
than credentials, so this task adds no repository secret. The job needs permissions id-token write or
the OIDC request returns nothing and the mint step fails before anything else.

Releases now depend on camden being up, unsealed and on the tailnet. A failure at the mint step is
infrastructure, not the commit.
<!-- SECTION:NOTES:END -->
