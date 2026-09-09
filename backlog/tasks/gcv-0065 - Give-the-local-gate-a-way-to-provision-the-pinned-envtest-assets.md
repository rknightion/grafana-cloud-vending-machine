---
id: GCV-0065
title: Give the local gate a way to provision the pinned envtest assets
status: In Progress
assignee:
  - '@claude'
created_date: '2026-09-09 21:28'
updated_date: '2026-09-09 22:42'
labels: []
dependencies: []
ordinal: 65000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The admission harness needs a pinned kube-apiserver and etcd, and hosted CI downloads them from the controller-tools release before running the gate. Locally there is no equivalent: `just setup` asserts KUBEBUILDER_ASSETS is already exported and fails when it is not, and no recipe fetches the assets. A contributor or agent running the documented gate for the first time gets a run where every admission test fails with `KUBEBUILDER_ASSETS is not set` while the rest of the suite passes and coverage silently drops, which reads like a broken repository rather than a missing prerequisite. The hosted workflow already carries the download, version assertion and layout logic worth reusing.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A just recipe provisions the pinned envtest assets for the local platform and is discoverable from `just --list`
- [ ] #2 The recipe verifies the downloaded archive against the publisher checksum sidecar and fails closed when it does not match, so no unverified binary is ever extracted
- [ ] #3 The recipe verifies the extracted kube-apiserver reports the version the justfile pins, and fails by name when it does not
- [ ] #4 Assets land outside the repository tree, so neither the gate YAML walk nor the publication scan forbidden-filename walk can reach them
- [ ] #5 The contributor documentation states how to provision the assets before running the gate for the first time
- [ ] #6 A gate run that is missing the assets fails with a message naming the provisioning recipe rather than only the raw environment-variable error
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 just check passes locally
- [ ] #2 hosted Validate workflow passes on the completing commit
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Reuse the hosted workflow download, sidecar-checksum and version-assertion logic, generalised for the local platform pair rather than linux-amd64 only.
2. Land the assets outside the repository tree, under the XDG cache, so neither the gate walk nor the publication scan can ever see them.
3. Make the recipe idempotent: an already-provisioned directory that reports the pinned version is reused, not re-downloaded.
4. Have the gate fail fast and by recipe name when the assets are absent, instead of after the suite has already run without them.
5. Point setup and the contributor documentation at the recipe.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Acceptance criteria tightened during implementation, with the reasons, so the record matches what was actually required rather than what was guessed at creation:

- The original criterion allowed assets to land in an ignored path inside the repository. Rejected on evidence: `scripts/validate.sh` walks every YAML document under the working directory and `scripts/public-release-scan.sh` walks it for forbidden filenames with `find`, and neither consults gitignore. An ignored directory is invisible to git and fully visible to both of those. Assets now land under the XDG cache, outside the tree.
- Checksum verification was implemented but was not a stated criterion, which left the strongest control in this recipe unpinned. It is now one.

Not applied, with reason: a review finding asked for per-executable checksum verification of kube-apiserver and etcd individually. Upstream publishes one `.sha512` sidecar for the archive and no per-binary digests, so there is nothing to verify against. The archive checksum is verified before extraction and already fail-closes on any tampered member, so per-binary digests would add no control even if they existed.

Containment guard on the cache location, added after review: `XDG_CACHE_HOME` is caller-controlled, so the recipe resolves the deepest existing ancestor of the cache path and the repository root with `pwd -P`, which follows symlinks on both sides, and refuses when the cache resolves to the repository root or any descendant of it. The check runs before anything is created, so a refusal leaves nothing behind.

Negative control, run: `XDG_CACHE_HOME=<repo root>/.badcache just envtest` exits 1 with `Refusing to provision envtest assets inside the repository at ...`, and no `.badcache` directory exists afterwards. The normal external cache path is unaffected and still provisions.

Not applied, with reason: a review finding asked the cached-asset reuse path to verify etcd reports the version the justfile pins. The justfile pins `ENVTEST_KUBERNETES_VERSION` only, which is the Kubernetes version and governs kube-apiserver; etcd is whatever version that controller-tools release bundles, and no pin exists to check it against. The reuse path does require both binaries present and executable, and the archive checksum is verified before extraction, so a truncated or tampered pair fails closed already. Adding an invented etcd version to assert against would be a fabricated control.

Delivered: a `just envtest` recipe in the `dev` group. It resolves the local platform pair from `uname`, refuses a machine type with no published asset, downloads the release archive and its `.sha512` sidecar, parses the sidecar strictly (the same awk contract the hosted workflow uses, so a changed release convention fails rather than being guessed at), verifies with `sha512sum` or `shasum -a 512` depending on what the host has, extracts, and asserts the extracted kube-apiserver reports `Kubernetes v${ENVTEST_KUBERNETES_VERSION}`. It is idempotent: an already-provisioned directory reporting the pinned version is reused without a download. It prints the export line with `%q` so a cache path containing a space stays copy-pasteable.

`scripts/validate.sh` now fails fast, before the publication scan, when `KUBEBUILDER_ASSETS` has no kube-apiserver and etcd, naming `just envtest`. That replaces the previous behaviour, which was to run the whole suite and let the admission tests fail individually while the run still reported the rest as passing and coverage dropped from 84.9% to 84.5%. Hosted CI provisions and exports the path before calling the script, so the guard cannot fire there. `just setup` names the recipe in its error too.

Controls run:
- cold cache: removed the cache directory, ran the recipe, downloaded and verified, `kube-apiserver: Kubernetes v1.37.0`, `etcd: etcd Version: 3.7.0`.
- warm cache: `Pinned envtest assets already present.`, no download.
- absent pin: `just ENVTEST_KUBERNETES_VERSION=9.99.0 envtest` fails at download with curl 404 rather than proceeding.
- cache inside the repository: refused with the path named, and nothing created.
- missing assets at the gate: `env -u KUBEBUILDER_ASSETS ./scripts/validate.sh` exits 1 naming `just envtest`.
- full gate with the recipe-provided path: `Validation passed.`, coverage 84.9%.
<!-- SECTION:NOTES:END -->
