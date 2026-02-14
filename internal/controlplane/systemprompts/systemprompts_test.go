package systemprompts

import (
	"strings"
	"testing"
)

func TestRenderOrchestratedPlanMode_RequiresCurrentPlanDir(t *testing.T) {
	t.Parallel()

	_, err := RenderOrchestratedPlanMode(OrchestratedPlanModeData{})
	if err == nil {
		t.Fatalf("expected error when CurrentPlanDir is empty")
	}
}

func TestRenderOrchestratedPlanMode_NoAdditionalDirs(t *testing.T) {
	t.Parallel()

	out, err := RenderOrchestratedPlanMode(OrchestratedPlanModeData{
		CurrentPlanDir: "/tmp/current-plan",
	})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	if !strings.Contains(out, "Current thread plan directory: `/tmp/current-plan`") {
		t.Fatalf("missing current plan dir in output: %q", out)
	}
	if !strings.Contains(out, "No additional pre-allocated thread plan directories are currently assigned.") {
		t.Fatalf("missing no-additional-dirs message in output: %q", out)
	}
}

func TestRenderOrchestratedPlanMode_WithAdditionalDirs(t *testing.T) {
	t.Parallel()

	out, err := RenderOrchestratedPlanMode(OrchestratedPlanModeData{
		CurrentPlanDir: "/tmp/current-plan",
		AdditionalPlanDirs: []PlanDir{
			{ThreadID: "thread-1", Path: "/tmp/plan-1"},
			{ThreadID: "thread-2", Path: "/tmp/plan-2"},
		},
	})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	if !strings.Contains(out, "Thread `thread-1`: `/tmp/plan-1`") {
		t.Fatalf("missing thread-1 mapping in output: %q", out)
	}
	if !strings.Contains(out, "Thread `thread-2`: `/tmp/plan-2`") {
		t.Fatalf("missing thread-2 mapping in output: %q", out)
	}
	if strings.Contains(out, "No additional pre-allocated thread plan directories are currently assigned.") {
		t.Fatalf("unexpected no-additional-dirs message in output: %q", out)
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
