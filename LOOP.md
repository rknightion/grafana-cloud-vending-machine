# Loop: grafana-cloud-vending-machine
tier: guarded
gate: just check
ci-required: ci-success
release-on-push: yes
deploy-on-push: no
receiver: https://loopwatch.m7kni.com
grafana-stack: none

## Credentials
- The repo carries no source-environment identity, no credentials and no live requests by design;
  examples are inert. `enabled/` is the only Argo-watched live-request directory and starts empty.
- Release token minting uses a short-lived, repository-scoped broker token through OpenBao. If minting
  fails, treat it as a broker problem, not a validation result, and never add a long-lived credential.
- Lanes are network-read-only at most: no live Grafana Cloud contact, no live cluster contact, no
  estate action. A lane may read pinned upstream provider or module source at an exact revision.

## Traps
- `just check` runs `scripts/validate.sh`, byte for byte what the hosted validation workflow runs.
  `just public-release-scan` is its first stage. It scans the tree and every reachable revision, so a
  banned literal that reaches a commit is a permanent failure; published history is never rewritten.
- Absolute macOS home-directory paths fail the scan case-sensitively, in this file too. Derive paths
  from `git rev-parse --show-toplevel` or use relative ones.
- Cluster tooling is always scoped: `KUBECONFIG=/dev/null` or the envtest kubeconfig, with
  `KUBEBUILDER_ASSETS` taken from `just envtest`'s printed export. An unset value is an environment
  gap, not a default. `just envtest` verifies pinned assets; never transcribe its checksum by hand.
  `XDG_CACHE_HOME` must resolve outside the repository tree.
- `just envtest-process-check` and `just envtest-process-reap` find and reap leaked envtest processes.
- A worktree isolates files only: not the Go caches, the envtest assets, the process table or Git refs.
- The provider controller is a carried build: `platform/provider/provider-grafana.yaml` overrides the
  package-runtime image. A provider pin bump (Renovate included) is red until the carry is rebuilt from
  the new tag, with image digest, verifier digest and tag-scoped identity moving together. Never merge
  a pin bump without that rebuild (`docs/installation.md`).
- A new ProviderRevision or FunctionRevision gets a 10-minute grace to report `RuntimeHealthy=True`;
  still False after it, or any True to False flip, is terminal-unhealthy. An image pull error, an Argo
  `Failed` or `Degraded` operation or an XRD not `Established` is an immediate stop. A
  content-identical platform pin does not rerun verifier hooks; fresh hook proof needs one explicitly
  granted one-shot sync.
- Release-please merges at most one release pull request, only as a loop's last publishing act and only
  when the goal grants it. Otherwise leave it for the owner. No comment, label, review, rebase request
  or Renovate-PR merge is authorised outside that merge.
- Completion claims carry the completing SHA and the hosted validation run id.
- Bare `backlog task edit --notes` / `--plan` replace the whole section; use `--append-notes` /
  `--append-plan`.

## Mutexes
- The host's envtest process table and pinned assets path are shared: a lane needing isolation uses a
  lane-private copy.
- Publication (commit or push to main, the release merge, tags, releases) is root-only.
