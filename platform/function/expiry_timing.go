package main

import (
	"github.com/crossplane/function-sdk-go/errors"
	"time"
)

// expiryArmingPermitted is only an additional timing gate. The caller must
// validate the existing explicit Delete intent and exact authorization first.
func expiryArmingPermitted(xr, config map[string]any) (bool, error) {
	spec, _ := xr["spec"].(map[string]any)
	raw, requested := spec["expiry"]
	if !requested {
		return true, nil
	}
	expiry, ok := raw.(map[string]any)
	if !ok {
		return false, errors.New("spec.expiry must be an object")
	}
	if _, err := expiryPolicyForUsage(config, stringValue(spec, "usage", "")); err != nil {
		return false, err
	}
	effective, _, err := effectiveExpiry(expiry)
	if err != nil {
		return false, err
	}
	now, ok := config[reconcileTimeConfigKey].(time.Time)
	if !ok {
		return false, errors.New("expiry requires the injected reconcile time")
	}
	return !now.Before(effective), nil
}
