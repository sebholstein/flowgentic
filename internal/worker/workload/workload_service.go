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
	WorkloadID     string
	AgentID        string
	AgentSessionID string
	Status         string
	Mode           string
}

// WorkloadService orchestrates workload scheduling on the worker.
type WorkloadService struct {
	mgr *WorkloadManager
}

func NewWorkloadService(mgr *WorkloadManager) *WorkloadService {
	return &WorkloadService{mgr: mgr}
}

// Schedule launches a workload via the WorkloadManager.
func (s *WorkloadService) Schedule(ctx context.Context, workloadID, agentID string, opts driver.LaunchOpts) (LaunchResult, error) {
	sess, err := s.mgr.Launch(ctx, workloadID, agentID, opts, nil)
	if err != nil {
		return LaunchResult{
			Accepted: false,
			Message:  fmt.Sprintf("launch failed: %v", err),
		}, nil
	}

	info := sess.Info()
	return LaunchResult{
		Accepted:       true,
		Message:        fmt.Sprintf("workload launched on agent %s", agentID),
		WorkloadID:     workloadID,
		AgentID:        info.AgentID,
		AgentSessionID: info.AgentSessionID,
		Status:         string(info.Status),
		Mode:           string(info.Mode),
	}, nil
}
