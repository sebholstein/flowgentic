package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// TailscaleConfig contains settings for exposing a service as a Tailscale / tsnet node.
type TailscaleConfig struct {
	// Enabled toggles whether the service should start with tsnet and register a Tailscale service.
	Enabled bool `json:"enabled"`

	// Hostname is the device name that will appear in your tailnet for this embedded tsnet node.
	Hostname string `json:"hostname"`

	// AuthKey is an optional Tailscale auth key used for unattended login.
	// If empty, tsnet falls back to TS_AUTHKEY / TS_AUTH_KEY env vars,
	// then prompts for interactive login on first start.
	AuthKey string `json:"authKey"`

	// Ephemeral controls whether this node is ephemeral in the tailnet.
	Ephemeral bool `json:"ephemeral"`

	// ControlURL optionally overrides the Tailscale control server URL (advanced / testing only).
	ControlURL string `json:"controlURL"`

	// Dir overrides the directory where tsnet stores its persistent state.
	// Defaults to the user config directory under tsnet-<hostname>.
	Dir string `json:"dir"`

	// HTTPS enables automatic TLS via Tailscale-managed Let's Encrypt certificates.
	// Only effective when Enabled is true.
	HTTPS bool `json:"https"`

	// ServiceName is the logical name of the Tailscale service (for Tailscale Services / Serve).
	ServiceName string `json:"serviceName"`
}

// WorkerEndpoint describes a remote worker that the control plane can relay
// requests to.
type WorkerEndpoint struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Secret string `json:"secret"`
}

// EmbeddedWorkerConfig holds settings for the embedded worker managed by the
// control plane as a child process.
type EmbeddedWorkerConfig struct {
	// Enabled toggles whether the control plane should manage an embedded worker.
	Enabled bool `json:"enabled"`

	// BinaryPath is the path to the worker binary. Defaults to "flowgentic-worker"
	// next to the control plane binary.
	BinaryPath string `json:"binaryPath"`

	// Secret is a shared secret between the CP and embedded worker.
	// Auto-generated if empty.
	Secret string `json:"secret"`
}

// ControlPlaneConfig holds configuration for the flowgentic control plane.
type ControlPlaneConfig struct {
	Port           int                  `json:"port"`
	Tailscale      TailscaleConfig      `json:"tailscale"`
	Workers        []WorkerEndpoint     `json:"workers"`
	DatabasePath   string               `json:"databasePath"`
	EmbeddedWorker EmbeddedWorkerConfig `json:"embeddedWorker"`
}

// WorkerConfig holds configuration for the flowgentic worker.
type WorkerConfig struct {
	Port      int             `json:"port"`
	Tailscale TailscaleConfig `json:"tailscale"`
}

// Config is the top-level configuration for the flowgentic system.
type Config struct {
	ControlPlane ControlPlaneConfig `json:"controlPlane"`
	Worker       WorkerConfig       `json:"worker"`
}

// Parse reads a JSON config file and returns the parsed Config.
// The file path is taken from FLOWGENTIC_CONFIG env var, defaulting to "flowgentic.json".
func Parse() (*Config, error) {
	path := os.Getenv("FLOWGENTIC_CONFIG")
	if path == "" {
		path = "flowgentic.json"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	cfg := &Config{
		ControlPlane: ControlPlaneConfig{
			Port:           8080,
			EmbeddedWorker: EmbeddedWorkerConfig{Enabled: true},
		},
		Worker: WorkerConfig{Port: 8081},
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	return cfg, nil
}
