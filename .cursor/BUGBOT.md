# Grafana Cloud vending review guide

## What this is

This public Crossplane reference reconciles Grafana Cloud stack and product requests. `platform/apis/` declares the request APIs. `platform/function/main.go` serves the Go composition function, and `platform/function/fn.go` dispatches and renders managed resources.

## Invariants

- `deploy/argocd/requests-applicationset.yaml` must continue to watch only `enabled/*`. Catalog examples are inert, renderable Kustomize bases. Public code, examples and review evidence must use fictional values.
- Keep deterministic child names and external-name annotations stable for adoption. Read provider-assigned account, team and policy identifiers from observations. `platform/function/roles.go` waits for an observed team ID before emitting its assignment. Give each whole-set ACL or role assignment a single declarative owner.
- Preserve released request semantics. Tightening a CEL condition, required field or immutability rule on a released kind needs a compatibility and migration review. Check the matching renderer as well as the XRD.

## Where to look hardest

- Follow lifecycle changes through `platform/function/fn.go`, `observation.go`, `oncall.go`, `oncallidentity.go` and `expiry.go`. Check ownership, recovered identities, decommission order and behavior under partial observations. Preserve the released inbound-email path for requests without an integration type.
- Review access scope and credential handling from XRD inputs through the renderer, RBAC and runtime manifests. Keep credentials and bearer links out of public examples, diagnostics and claim status. Check the provider pin, override image, verification digest and tag-scoped identity together when the carried provider changes.
- For a new resource, check the renderer dispatch, platform Kustomize inventory, composition RBAC, and relevant API and renderer tests for agreement.

## Checks already covered

The gate scans the working tree and reachable history for publication issues. It also checks formatting, module tidiness, race tests, vet, Kustomize rendering and admission with pinned envtest assets. Hosted validation adds workflow security and pull request evidence checks.

## Verification

Run `just setup`, provision verified assets with `just envtest`, then run `just check </dev/null>` with the reported assets and `KUBECONFIG=/dev/null`. Use renderer tests for resource emission, envtest for API-server admission, and pinned upstream source for provider-contract claims. Live Grafana Cloud and cluster testing is outside this repository's loop.
