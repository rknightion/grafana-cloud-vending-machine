# External project to stack request

This catalog base shows the consuming-platform mapping contract for one
project CR and its generated `GrafanaCloudStackRequest`. It is deliberately
outside `enabled/`, so it is inert and is not watched by the repository's
ApplicationSet.

## Files

- `project.yaml` is the illustrative external project-CR shape. This
  repository does not install or watch its CRD.
- `stack-request.yaml` is the generated request that a consuming platform
  would submit through a reviewed pull request.
- `kustomization.yaml` renders both public-shape objects as one catalog base.

## Mapping in this example

The source organization is `example-primary`, the constrained usage is
`development`, and the source project name is `demo`. The request identity is
therefore `exampleprimarydevelopmentdemo`, which satisfies the request XRD's
lowercase-alphanumeric slug pattern and appears identically in
`metadata.name` and `spec.slug`.

The region, profile, lifecycle, and optional feature values in the request are
platform-fixed template values. The request explicitly uses
`spec.lifecycle.externalResources: Retain`; removing the source project object
does not delete a stack. See [External project integration](../../../docs/external-project-integration.md)
for the complete mapping, approval seam, failure classification, and reviewed
decommission path.

## Enablement boundary

Do not copy this directory into `enabled/` without adapting the values,
verifying the organization registry and target namespace, and sending the
generated request through the consuming platform's pull-request approval
workflow. The catalog render is structural evidence only; it does not contact
a cluster or Grafana Cloud.
