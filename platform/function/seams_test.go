package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestUnimplementedWave3KindsFailClosed(t *testing.T) {
	for _, kind := range []string{"GrafanaK6Project", "GrafanaSyntheticMonitoring", "GrafanaStackLadder"} {
		renderer, exists := compositeRenderers[kind]
		if !exists {
			t.Fatalf("%s must be registered behind a readiness flag", kind)
		}
		original := renderer
		renderer.implemented = false
		compositeRenderers[kind] = renderer
		defer func() { compositeRenderers[kind] = original }()
		rsp := callFunction(t, fmt.Sprintf(`{"apiVersion":"platform.example.org/v1beta1","kind":%q}`, kind), nil, "")
		if got := fatalResult(rsp); !strings.Contains(got, "registered but not implemented") {
			t.Fatalf("%s fatal = %q", kind, got)
		}
		if len(rsp.GetDesired().GetResources()) != 0 {
			t.Fatalf("%s emitted resources before implementation", kind)
		}
	}
}

func TestConfiguredGoldenSLOIsNotSilentlyOmitted(t *testing.T) {
	rsp := callFunction(t, stackDocument(nil), nil, `{"spec":{"goldenSLOProfiles":"invalid"}}`)
	if fatalResult(rsp) == "" {
		t.Fatal("configured golden SLO returned empty success")
	}
}
