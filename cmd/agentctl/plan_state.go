package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

const (
	agentCtlSessionIDEnv = "AGENTCTL_SESSION_ID"
	planRootSuffix       = ".agentflow/plans"
	stateDirName         = ".agentctl"
)

type planAllocation struct {
	ThreadID string `json:"thread_id"`
	PlanDir  string `json:"plan_dir"`
}

type planState struct {
	Current    planAllocation   `json:"current"`
	Additional []planAllocation `json:"additional"`
}

func loadOrInitPlanState() (*planState, error) {
	sessionID := os.Getenv(agentCtlSessionIDEnv)
	if sessionID == "" {
		return nil, fmt.Errorf("%s env not set", agentCtlSessionIDEnv)
	}

	root, err := planRootDir()
	if err != nil {
		return nil, err
	}
	statePath := filepath.Join(root, stateDirName, sessionID+".json")

	b, err := os.ReadFile(statePath)
	if err == nil {
		var st planState
		if err := json.Unmarshal(b, &st); err != nil {
			return nil, fmt.Errorf("parse plan state: %w", err)
		}
		if st.Current.ThreadID == "" || st.Current.PlanDir == "" {
			return nil, fmt.Errorf("invalid plan state: missing current allocation")
		}
		return &st, nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read plan state: %w", err)
	}

	// Current thread is anchored to the internal session id.
	currentDir := filepath.Join(root, sessionID)
	st := &planState{
		Current: planAllocation{
			ThreadID: sessionID,
			PlanDir:  currentDir,
		},
	}
	if err := ensurePlanDir(currentDir); err != nil {
		return nil, err
	}
	if err := savePlanState(st); err != nil {
		return nil, err
	}
	return st, nil
}

func savePlanState(st *planState) error {
	sessionID := os.Getenv(agentCtlSessionIDEnv)
	if sessionID == "" {
		return fmt.Errorf("%s env not set", agentCtlSessionIDEnv)
	}
	root, err := planRootDir()
	if err != nil {
		return err
	}
	stateDir := filepath.Join(root, stateDirName)
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}
	statePath := filepath.Join(stateDir, sessionID+".json")
	tmpPath := statePath + ".tmp"

	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal plan state: %w", err)
	}
	if err := os.WriteFile(tmpPath, b, 0o644); err != nil {
		return fmt.Errorf("write temp plan state: %w", err)
	}
	if err := os.Rename(tmpPath, statePath); err != nil {
		return fmt.Errorf("replace plan state: %w", err)
	}
	return nil
}

func planRootDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	root := filepath.Join(home, planRootSuffix)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", fmt.Errorf("create plan root: %w", err)
	}
	return root, nil
}

func ensurePlanDir(path string) error {
	if err := os.MkdirAll(filepath.Join(path, "tasks"), 0o755); err != nil {
		return fmt.Errorf("create plan dir %q: %w", path, err)
	}
	return nil
}

func allocateAdditionalPlanDir(st *planState) (planAllocation, error) {
	root, err := planRootDir()
	if err != nil {
		return planAllocation{}, err
	}

	threadID := uuid.Must(uuid.NewV7()).String()
	internalSessionID := uuid.Must(uuid.NewV7()).String()
	path := filepath.Join(root, internalSessionID)

	a := planAllocation{
		ThreadID: threadID,
		PlanDir:  path,
	}
	if err := ensurePlanDir(path); err != nil {
		return planAllocation{}, err
	}
	st.Additional = append(st.Additional, a)
	if err := savePlanState(st); err != nil {
		return planAllocation{}, err
	}
	return a, nil
}
