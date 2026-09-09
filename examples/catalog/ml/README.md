# Grafana Machine Learning

This inert example selects the platform-owned `standard` ML profile for a previously vended stack. The profile owns the job queries, detector algorithms, holiday periods, datasource identifiers, and the cap on continuously running jobs and detectors.

## Files

- `ml.yaml` selects the profile and refers to the stack request.
- `kustomization.yaml` makes the directory directly renderable with Kustomize.

## Values to replace

- Replace `platform.example.org` with your API group.
- Replace `replacewithunique01` with the name of an already-vended stack request.
- Replace `standard` only with an approved platform-owned ML profile. Do not add ML workload details or raise a cap in this request.

The request name must equal `stackRef.name`, and the stack reference is immutable. This establishes one declarative owner for this surface per stack.
