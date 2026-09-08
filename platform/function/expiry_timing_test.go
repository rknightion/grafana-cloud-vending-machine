package main

import (
	"testing"
	"time"
)

func TestExpiryTimingBoundary(t *testing.T) {
	deadline := time.Date(2030, 1, 2, 0, 0, 0, 0, time.UTC)
	xr := map[string]any{"spec": map[string]any{"usage": "development", "expiry": map[string]any{"expiresAt": deadline.Format(time.RFC3339)}}}
	config := map[string]any{"spec": map[string]any{"expiryPolicies": []any{map[string]any{"usage": "development", "warningBefore": "24h", "warningFolderUID": "baseline"}}}}
	for _, delta := range []time.Duration{-time.Second, 0, time.Second} {
		config[reconcileTimeConfigKey] = deadline.Add(delta)
		due, err := expiryArmingPermitted(xr, config)
		if err != nil || due != (delta >= 0) {
			t.Fatalf("delta=%v due=%v err=%v", delta, due, err)
		}
	}
	delete(config, reconcileTimeConfigKey)
	if _, err := expiryArmingPermitted(xr, config); err == nil {
		t.Fatal("accepted absent clock")
	}
}
