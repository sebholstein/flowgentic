package systeminfo

import (
	"context"

	"github.com/sebastianm/flowgentic/internal/worker/systeminfo/agentinfo"
)

// SystemInfoService provides system information about the worker node.
type SystemInfoService struct {
	agents agentinfo.AgentInfo
}

func NewSystemInfoService(agents agentinfo.AgentInfo) *SystemInfoService {
	return &SystemInfoService{agents: agents}
}

// Ping is a lightweight health-check that confirms the worker is reachable.
func (s *SystemInfoService) Ping() {}

// ListAgents returns all discovered coding agents.
func (s *SystemInfoService) ListAgents(ctx context.Context, disableCache bool) ([]agentinfo.Agent, error) {
	return s.agents.DiscoverAgents(ctx, disableCache)
}
