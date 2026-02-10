package workload

import (
	"context"
	"fmt"

	"github.com/sebastianm/flowgentic/internal/worker/driver"
)

// LaunchResult describes the outcome of launching a workload.
type LaunchResult struct {
	Accepted       bool
	Message        string
	AgentRunID     string
	AgentID        string
	AgentSessionID string
	Status         string
	Mode           string
}

// WorkloadService orchestrates workload scheduling on the worker.
type WorkloadService struct {
	mgr *AgentRunManager
}

func NewWorkloadService(mgr *AgentRunManager) *WorkloadService {
	return &WorkloadService{mgr: mgr}
}

// ListAgentRuns returns info for all active agent runs.
func (s *WorkloadService) ListAgentRuns(_ context.Context) []SessionListEntry {
	return s.mgr.ListSessions()
}

// Schedule launches a workload via the AgentRunManager.
func (s *WorkloadService) Schedule(ctx context.Context, agentRunID, agentID string, opts driver.LaunchOpts) (LaunchResult, error) {
	sess, err := s.mgr.Launch(ctx, agentRunID, agentID, opts, nil)
	if err != nil {
		return LaunchResult{
			Accepted: false,
			Message:  fmt.Sprintf("launch failed: %v", err),
		}, nil
	}

	info := sess.Info()
	return LaunchResult{
		Accepted:       true,
		Message:        fmt.Sprintf("agent run launched on agent %s", agentID),
		AgentRunID:     agentRunID,
		AgentID:        info.AgentID,
		AgentSessionID: info.AgentSessionID,
		Status:         string(info.Status),
		Mode:           string(info.Mode),
	}, nil
}
