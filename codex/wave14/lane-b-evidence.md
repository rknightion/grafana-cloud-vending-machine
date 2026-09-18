# Wave 14 Lane B evidence

Accepted packet: `codex/wave14/lane-a-packet.md`.

Accepted packet SHA-256: `f52fdcfe4e9641fd03c7c201621332c9bc5d70c706b45d484c8d06b60fb31051`.

Source identity: `52308041d25f465e21e711add0dfcf7578f94866`.

## Replay RED

The baseline source was materialized from the source identity in an isolated
temporary directory. The mandatory replay assertion was added there before the
Lane B helper existed:

```text
$ go test ./... -run '^TestRotationSuccessorIsRetainedAcrossARefresh$' -count=1
--- FAIL: TestRotationSuccessorIsRetainedAcrossARefresh (0.00s)
    rotation_red_test.go:24: due observed token did not produce a retained successor generation
FAIL
FAIL github.com/rknightion/grafana-cloud-vending-machine/platform/function 0.534s
```

This is the missing behavior: a due observed token rendered only its legacy
child and did not retain a distinct successor across refresh.

## Replay GREEN

After the helper and replay were implemented in the shared checkout:

```text
$ just test 'TestRotatingTokenFamily|TestSelectedTokenExpiryStatusDoesNotLabelEstimatesAsProviderObserved'
cd platform/function && go test -race -cover -run 'TestRotatingTokenFamily|TestSelectedTokenExpiryStatusDoesNotLabelEstimatesAsProviderObserved' ./...
ok github.com/rknightion/grafana-cloud-vending-machine/platform/function 1.793s coverage: 7.8% of statements
```

The replay covers fresh desired maps, pending and failed candidates, desired
successor selection before observed acknowledgement, stale publisher status,
current publisher acknowledgement, durable retirement intent, predecessor
absence, and no legacy recreation. Adversarial cases cover malformed lineage,
duplicate ordinals, missing clock, unknown candidate timing, observed lifetime
preservation, distinct minting provider identity, and Secret UID replacement.

## Production build

```text
$ go build ./...
```

The command exited successfully with no output.

## Owned changes

- `platform/function/access.go`: shared generation inventory, deterministic
  successor construction, Secret and publisher acknowledgement gates, immutable
  timing, retirement/pruning results, and family deletion preparation helper.
- `platform/function/access_test.go`: replay, all five site descriptors, and
  adversarial rotation tests.
- `codex/wave14/lane-b-wiring.md`: exact root-owned integration signatures and
  insertion points.
- `codex/wave14/lane-b-docs.md`: target anchors and finished documentation
  prose for security, Secret ownership, and troubleshooting.

## Disposition and limits

The source/helper layer proves the overlapping-generation ordering locally. AC3,
AC4, and AC5 remain root integration claims until the five descriptors are
invoked from `RunFunction`, required Secret observations are wired, composite
status and Ready overrides are applied, bootstrap references are migrated, and
family-aware deletion readiness replaces the legacy calls. No live Grafana,
Kubernetes cluster, provider, or External Secrets system was contacted.

The existing pinned Crossplane Secret-read authority remains the basis for the
wiring packet; no duplicate Secret RBAC was added. No commit, stage, push,
release, tag, tracker edit, or live resource mutation was performed.
