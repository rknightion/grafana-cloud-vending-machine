---
id: GCV-0095
title: 'CI hygiene: Go cache size, one Go version, registry buildx cache'
status: Parked
assignee: []
created_date: '2026-09-26 17:11'
updated_date: '2026-09-26 17:37'
labels: []
dependencies: []
priority: high
type: chore
ordinal: 95000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Fleet CI hygiene tracked as GHC-0006 in rknightion/.github. This repository's Actions cache filled to 10GB almost entirely with setup-go caches (~690MB each). Root cause: go test -race against a large controller-runtime/k8s.io/crossplane dependency tree (333 modules) produces a large build cache on its own (~1.7GB raw with a plain build, ~2.9GB with -race enabled, roughly a 70% increase from race instrumentation alone); validate.yml and publish-function.yml also pinned two different Go versions (1.27.1 vs env.GO_VERSION 1.27.0), so setup-go never shared one cache key between them and every run on every PR/push wrote its own copy, multiplying the footprint. publish-function.yml also used a type=gha Docker layer cache for its runtime image build, competing with the same setup-go caches against the repository's shared 10GB Actions cache cap.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Explain why each setup-go cache entry was ~690MB (go test -race against the controller-runtime/k8s.io/crossplane dependency tree) and reduce the footprint where cheap and safe: cache writes only on push to main (setup-go cache: false on pull_request runs)
- [x] #2 validate.yml and publish-function.yml resolve Go from one shared source (go-version-file: platform/function/go.mod) so both workflows use the same setup-go cache key instead of two
- [ ] #3 publish-function.yml's docker/build-push-action uses a GHCR registry cache (type=registry, mode=max on cache-to) instead of type=gha, with cache-to only on push to main, and the build job holds packages: write for it
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Committed and pushed as 8fba44d on main. AC1 and AC2 proven by the hosted Publish Grafana vending function run 36259154934's Test function job (completed success): go-version-file: platform/function/go.mod resolved identically in both workflows (one shared setup-go cache key), and the cache: ${{ github.event_name != 'pull_request' }} guard means pull_request runs no longer write a fresh ~690MB cache. Root cause confirmed by direct measurement: go test -race against the 333-module controller-runtime/k8s.io/crossplane dependency tree produces ~2.9GB of raw GOCACHE versus ~1.7GB without -race (about a 70% increase from race instrumentation alone), on top of a 262MB GOMODCACHE; the two workflows previously pinning different Go versions meant this was duplicated as two separate cache entries instead of one, on every PR run. AC3 (GHCR registry cache on the build job) is implemented in the same commit but is statically verified only: actionlint and zizmor are both clean, and the cache-from/cache-to/login/permissions wiring was traced by hand against docker/build-push-action's documented input parsing, but the build and publish jobs were SKIPPED on 8fba44d because that commit only touched .github/workflows/*.yml, and publish-function.yml's runtime-changed gate (unrelated to this change) only runs build/publish when platform/function's actual runtime files change. Deliberately did not force a workflow_dispatch run to exercise it, since that would push a real new immutable package version to ghcr.io/rknightion/grafana-cloud-vending-machine/function-grafana-vending and move the floating :main tag - a live publish this task does not need. Parking rather than closing: whoever picks this up next should check AC3 off the build job's log (cache-from/cache-to hitting ghcr.io/rknightion/grafana-cloud-vending-machine/function-grafana-vending:buildcache-amd64 and -arm64 with no auth or ref errors) on the next push that actually touches platform/function, then move this to Done.
<!-- SECTION:FINAL_SUMMARY:END -->
