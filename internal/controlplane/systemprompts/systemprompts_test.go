package systemprompts

import (
	"strings"
	"testing"
)

func TestRenderOrchestratedPlanMode_EmptyData(t *testing.T) {
	t.Parallel()

	out, err := RenderOrchestratedPlanMode(OrchestratedPlanModeData{})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if !strings.Contains(out, "Flowgentic Agent") {
		t.Fatalf("missing header in output: %q", out)
	}
}

func TestRenderOrchestratedPlanMode_ContainsExpectedSections(t *testing.T) {
	t.Parallel()

	out, err := RenderOrchestratedPlanMode(OrchestratedPlanModeData{
		CurrentPlanDir: "/tmp/current-plan",
	})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	for _, want := range []string{
		"Flowgentic Agent",
		"Flowgentic MCP",
		"set_topic",
		"Guidelines",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %q", want, out)
		}
	}
}

func TestDefaultPlanDirForSession(t *testing.T) {
	t.Parallel()

	got := DefaultPlanDirForSession("sess-123")
	want := "~/.agentflow/plans/sess-123"
	if got != want {
		t.Fatalf("unexpected default plan dir: got %q want %q", got, want)
	}
}
