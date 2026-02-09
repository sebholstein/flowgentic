package agentinfo

import (
	"context"
	"sync"
)

// Agent describes a coding agent installed on the worker.
type Agent struct {
	ID      string
	Name    string
	Version string
}

// AgentInfo finds coding agents installed on this worker.
type AgentInfo interface {
	DiscoverAgents(ctx context.Context, disableCache bool) ([]Agent, error)
}

// detector checks whether a single coding agent is installed.
type detector interface {
	detect(ctx context.Context) (Agent, bool)
}

// Discoverer aggregates multiple detectors and returns all found agents.
// Only one DiscoverAgents call runs at a time; results are cached until
// a caller explicitly requests a fresh discovery with disableCache=true.
type Discoverer struct {
	detectors []detector

	mu           sync.Mutex
	cached       bool
	cachedAgents []Agent
}

// NewDiscoverer returns a Discoverer that checks for all known coding agents.
func NewDiscoverer() *Discoverer {
	v := runVersionCommand
	return &Discoverer{
		detectors: []detector{
			claudeCodeDetector{version: v},
			codexDetector{version: v},
			opencodeDetector{version: v},
			ampDetector{version: v},
			geminiDetector{version: v},
		},
	}
}

// DiscoverAgents runs each detector concurrently and returns the agents that are installed.
// Only one discovery runs at a time. Results are cached and returned on subsequent calls
// unless disableCache is true, in which case a fresh discovery is performed and the cache
// is updated with the new result.
func (d *Discoverer) DiscoverAgents(ctx context.Context, disableCache bool) ([]Agent, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.cached && !disableCache {
		return d.cachedAgents, nil
	}

	var (
		mu     sync.Mutex
		wg     sync.WaitGroup
		agents []Agent
	)

	for _, det := range d.detectors {
		wg.Add(1)
		go func(det detector) {
			defer wg.Done()
			if a, ok := det.detect(ctx); ok {
				mu.Lock()
				agents = append(agents, a)
				mu.Unlock()
			}
		}(det)
	}

	wg.Wait()

	d.cachedAgents = agents
	d.cached = true

	return agents, nil
}
