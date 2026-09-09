# Frontend Observability app

This inert example selects the platform-owned `standard` Frontend Observability app profile for a previously vended stack.

## Files

- `frontend-observability.yaml` selects the profile and refers to the stack request.
- `kustomization.yaml` makes the directory directly renderable with Kustomize.

## Credential boundary

The provider publishes `status.atProvider.collectorEndpoint` for each App. Its URL contains the Faro app key because the browser SDK sends telemetry directly to that endpoint. The key is an ingestion identifier intended to be visible in a browser bundle. It is not accepted from this request, stored in a Kubernetes Secret, or granted privileged Grafana access.

## Values to replace

- Replace `platform.example.org` with your API group.
- Replace `replacewithunique01` with the name of an already-vended stack request.
- Replace `standard` only with an approved platform-owned Frontend Observability profile. Do not add app configuration or any credential to this request.

The request name must equal `stackRef.name`, and the stack reference is immutable. This establishes one declarative owner for this surface per stack.
