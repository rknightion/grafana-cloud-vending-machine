# Git provisioning repository

Use this opt-in example when a Git repository, rather than a Crossplane Dashboard resource, owns one Grafana folder subtree. It is a preview surface. The classic Crossplane Dashboard route remains the supported default in this wave; Git provisioning repositories are the intended future default once the preview matures.

## Files

- `repository.yaml` declares one Git-provisioned folder subtree.
- `kustomization.yaml` renders the example as a Kustomize base.

## Prerequisites

- The minimal platform and the referenced Grafana Cloud stack are ready.
- Apply a separately vended Grafana `ConnectionV0Alpha1` to the stack before applying this request. This request references that connection and never embeds a token, private key, or Kubernetes Secret reference.
- The Git repository and branch contain the dashboards and related resources for this subtree.

Grafana Cloud is the only supported deployment target. Self-managed Grafana feature-toggle requirements do not apply to this reference.

## Values to replace

- Replace `platform.example.org`, every `replacewithunique06` occurrence, the Git URL, branch, path, and repository UID.
- Replace `replace-with-existing-grafana-connection` with the existing Grafana connection name. Do not add credentials to this manifest.
- Select exactly one provider block matching `repository.type`: `local`, `github`, `githubEnterprise`, `git`, `bitbucket`, or `gitlab`. The local form contains only `local.path`; remote forms carry their URL, branch, and path. The generic Git and Bitbucket forms also require `tokenUser`. Bitbucket PAT authentication conventionally uses `x-token-auth` for that field.
- `sync.target` accepts `folder` or `folderless`. `instance` is deliberately refused: it would claim the whole Grafana instance and conflict with the one-declarative-owner-per-folder-subtree rule.
- **`sync.intervalSeconds` has a Grafana Cloud floor of 300, and nothing reports the override.** A lower value is accepted and then silently raised: a claim asking for 60 is stored by Grafana as 300, while the Crossplane provider reads back its own requested 60 and reports no drift. Neither side flags it, so a request below 300 is a claim that quietly does not describe the live poll interval. Verified live on 2026-09-12. Set 300 or more.
- Keep `workflows: []` for a read-only Git-provisioned subtree. Add only `write` and/or `branch` after reviewing the resulting Grafana write path.
- When the provider needs a repository token, webhook secret, or commit-signing key, use `repository.secure.<key>.name` to reference an externally vended secure value and increment `secureVersion` after rotation. The claim has no secure-value creation control or credential literal field. An S/MIME certificate belongs in `repository.commit.smimeCertificate` because it is public.

## Ownership and reconciliation

The repository `path` is the owned folder subtree. This API renders no Crossplane `Dashboard` resource for it and rejects a request that also declares classic dashboard ownership. Keep each subtree on exactly one route: use this API for Git provisioning, or use `GrafanaCloudStackRequest` for the supported classic Dashboard default.

The Connection remains separately managed so its Git credential never enters this public repository. The Repository references that Connection, then Grafana synchronizes the declared Git subtree. Several repository claims can reference the same stack and separately vended connection while retaining one Git owner for each folder subtree.
