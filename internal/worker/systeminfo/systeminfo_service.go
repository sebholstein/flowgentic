package systeminfo

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/sebastianm/flowgentic/internal/worker/driver"
	v2 "github.com/sebastianm/flowgentic/internal/worker/driver/v2"
	"github.com/sebastianm/flowgentic/internal/worker/systeminfo/agentinfo"
)

var (
	ErrUnsupportedAgent = errors.New("unsupported agent")
	ErrModelDiscovery   = errors.New("model discovery failed")
)

// AgentModels describes available/default model metadata for one agent.
type AgentModels struct {
	Models       []string
	DefaultModel string
}

// SystemInfoService provides system information about the worker node.
type SystemInfoService struct {
	agents agentinfo.AgentInfo

	modelProbeCwd string
	drivers       map[driver.AgentType]v2.Driver
	modelCache    map[driver.AgentType]cachedModels
	mu            sync.Mutex
	now           func() time.Time
}

type cachedModels struct {
	value     AgentModels
	expiresAt time.Time
}

func NewSystemInfoService(agents agentinfo.AgentInfo, drivers []v2.Driver, modelProbeCwd string) *SystemInfoService {
	driverMap := make(map[driver.AgentType]v2.Driver, len(drivers))
	for _, d := range drivers {
		driverMap[driver.AgentType(d.Agent())] = d
	}

	return &SystemInfoService{
		agents:        agents,
		modelProbeCwd: modelProbeCwd,
		drivers:       driverMap,
		modelCache:    make(map[driver.AgentType]cachedModels),
		now:           time.Now,
	}
}

// Ping is a lightweight health-check that confirms the worker is reachable.
func (s *SystemInfoService) Ping() {}

// ListAgents returns all discovered coding agents.
func (s *SystemInfoService) ListAgents(ctx context.Context, disableCache bool) ([]agentinfo.Agent, error) {
	return s.agents.DiscoverAgents(ctx, disableCache)
}

// GetAgentModels returns available/default model metadata for one agent.
func (s *SystemInfoService) GetAgentModels(ctx context.Context, agent driver.AgentType, disableCache bool) (AgentModels, error) {
	d, ok := s.drivers[agent]
	if !ok {
		return AgentModels{}, fmt.Errorf("%w: %s", ErrUnsupportedAgent, agent)
	}

	now := s.now()
	if !disableCache {
		s.mu.Lock()
		if cached, ok := s.modelCache[agent]; ok && now.Before(cached.expiresAt) {
			s.mu.Unlock()
			return cloneAgentModels(cached.value), nil
		}
		s.mu.Unlock()
	}

	inv, err := d.DiscoverModels(ctx, s.modelProbeCwd)
	if err != nil {
		return AgentModels{}, fmt.Errorf("%w: %v", ErrModelDiscovery, err)
	}
	if len(inv.Models) == 0 || inv.DefaultModel == "" {
		return AgentModels{}, fmt.Errorf("%w: incomplete metadata", ErrModelDiscovery)
	}

	value := AgentModels{
		Models:       append([]string(nil), inv.Models...),
		DefaultModel: inv.DefaultModel,
	}
	s.mu.Lock()
	s.modelCache[agent] = cachedModels{
		value:     value,
		expiresAt: now.Add(time.Minute),
	}
	s.mu.Unlock()
	return cloneAgentModels(value), nil
}

func cloneAgentModels(v AgentModels) AgentModels {
	return AgentModels{
		Models:       append([]string(nil), v.Models...),
		DefaultModel: v.DefaultModel,
	}
}
