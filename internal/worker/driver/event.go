package driver

// HookEvent is the raw event received from an agent hook via RPC.
type HookEvent struct {
	SessionID string `json:"session_id"`
	Agent     string `json:"agent"`
	HookName  string `json:"hook_name"`
	Payload   []byte `json:"payload"`
}
