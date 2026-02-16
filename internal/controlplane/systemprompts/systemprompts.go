package systemprompts

import (
	"bytes"
	_ "embed"
	"fmt"
	"path/filepath"
	"text/template"
)

//go:embed ORCHESTRATED_PLAN_MODE.md
var orchestratedPlanModeTemplateText string

const planRootDir = "~/.Flowgentic/plans"

type PlanDir struct {
	ThreadID string
	Path     string
}

type OrchestratedPlanModeData struct {
	CurrentPlanDir     string
	AdditionalPlanDirs []PlanDir
}

var orchestratedPlanModeTemplate = template.Must(template.New("orchestrated-plan-mode").Parse(orchestratedPlanModeTemplateText))

func DefaultPlanDirForSession(sessionID string) string {
	return filepath.Join(planRootDir, sessionID)
}

func RenderOrchestratedPlanMode(data OrchestratedPlanModeData) (string, error) {
	var b bytes.Buffer
	if err := orchestratedPlanModeTemplate.Execute(&b, data); err != nil {
		return "", fmt.Errorf("render orchestrated plan mode prompt: %w", err)
	}
	return b.String(), nil
}
