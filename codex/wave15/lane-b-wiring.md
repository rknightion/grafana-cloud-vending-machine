# Lane B wiring packet

Apply these two edits in `platform/function/fn.go`. They use the same request
config map passed to `renderer.render`; do not wrap, copy, or rebuild it.

1. After the existing `productWithdrawalError` block and before the
   `GrafanaK6Project` readiness check, add the feature registration:

```go
if err == nil {
	addCredentialHealth(content, observed, config)
}
```

This is deliberately after rendering and withdrawal validation. It derives
expected token families from the composite configuration, and observed token
health by GVK and the child `Synced` condition only.

2. After `markObservedResourcesReady(desired, observed)` and before
   `response.SetDesiredComposedResources`, apply the result after every
   existing product and access readiness gate:

```go
if err := applyCredentialHealthReadiness(rsp, config); err != nil {
	response.Fatal(rsp, errors.Wrap(err, "cannot set credential health readiness"))
	return rsp, nil
}
```

`addCredentialHealth` has the signature:

```go
func addCredentialHealth(
	xr map[string]any,
	observed map[resource.Name]resource.ObservedComposed,
	config map[string]any,
)
```

`applyCredentialHealthReadiness` has the signature:

```go
func applyCredentialHealthReadiness(
	rsp *fnv1.RunFunctionResponse,
	config map[string]any,
) error
```

An unhealthy configured family sets only the owning composite to `Ready=False`.
Healthy children leave normal Crossplane readiness aggregation unchanged; the
helper never writes a child condition or readiness value.
