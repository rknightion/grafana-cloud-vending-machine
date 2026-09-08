# Git provisioning repository

Use this opt-in example when a Git repository, rather than a Crossplane Dashboard resource, owns one Grafana folder subtree. It is a preview surface. The classic Crossplane Dashboard route remains the supported default in this wave; Git provisioning repositories are the intended future default once the preview matures.

## Files

- `repository.yaml` declares one Git-provisioned folder subtree.
- `kustomization.yaml` renders the example as a Kustomize base.

## Prerequisites

- The minimal platform and the referenced Grafana Cloud stack are ready.
- A Grafana `ConnectionV0Alpha1` already exists in that stack with the reviewed Git credential. This request references that connection and never embeds a token, private key, or Kubernetes Secret reference.
- The Git repository and branch contain the dashboards and related resources for this subtree.

Grafana Cloud is the only supported deployment target. Self-managed Grafana feature-toggle requirements do not apply to this reference.

## Values to replace

- Replace `platform.example.org`, every `replacewithunique06` occurrence, the Git URL, branch, path, and repository UID.
- Replace `replace-with-existing-grafana-connection` with the existing Grafana connection name. Do not add credentials to this manifest.

## Ownership and reconciliation

The repository `path` is the owned folder subtree. This API renders no Crossplane `Dashboard` resource for it and rejects a request that also declares classic dashboard ownership. Keep each subtree on exactly one route: use this API for Git provisioning, or use `GrafanaCloudStackRequest` for the supported classic Dashboard default.

The Connection remains separately managed so its Git credential never enters this public repository. The Repository references that Connection, then Grafana synchronizes the declared Git subtree.
