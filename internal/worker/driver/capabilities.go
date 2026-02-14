package driver

// Capability describes an optional feature a driver supports.
type Capability string

const (
	CapStreaming      Capability = "streaming"
	CapSessionResume  Capability = "session_resume"
	CapCostTracking   Capability = "cost_tracking"
	CapCustomModel    Capability = "custom_model"
	CapSystemPrompt   Capability = "system_prompt"
	CapPermissionRequest Capability = "permission_request"
	CapFileSystem        Capability = "file_system"
	CapTerminal          Capability = "terminal"
)

// Capabilities describes what a driver supports.
type Capabilities struct {
	Agent     string       `json:"agent"`
	Supported []Capability `json:"supported"`
}

// Has returns true if the capability is in the supported list.
func (c Capabilities) Has(cap Capability) bool {
	for _, s := range c.Supported {
		if s == cap {
			return true
		}
	}
	return false
}
