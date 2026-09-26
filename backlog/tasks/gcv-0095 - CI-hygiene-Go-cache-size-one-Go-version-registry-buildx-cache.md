---
id: GCV-0095
title: 'CI hygiene: Go cache size, one Go version, registry buildx cache'
status: To Do
assignee: []
created_date: '2026-09-26 17:11'
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
- [ ] #1 Explain why each setup-go cache entry was ~690MB (go test -race against the controller-runtime/k8s.io/crossplane dependency tree) and reduce the footprint where cheap and safe: cache writes only on push to main (setup-go cache: false on pull_request runs)
- [ ] #2 validate.yml and publish-function.yml resolve Go from one shared source (go-version-file: platform/function/go.mod) so both workflows use the same setup-go cache key instead of two
- [ ] #3 publish-function.yml's docker/build-push-action uses a GHCR registry cache (type=registry, mode=max on cache-to) instead of type=gha, with cache-to only on push to main, and the build job holds packages: write for it
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->
