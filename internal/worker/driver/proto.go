package driver

import (
	"fmt"

	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
)

// AgentType is a string identifier for an agent (e.g. "claude-code", "codex").
type AgentType string

const (
	AgentTypeClaudeCode AgentType = "claude-code"
	AgentTypeCodex      AgentType = "codex"
	AgentTypeOpenCode   AgentType = "opencode"
	AgentTypeAmp        AgentType = "amp"
	AgentTypeGemini     AgentType = "gemini"
)

var agentTypeToProto = map[AgentType]workerv1.Agent{
	AgentTypeClaudeCode: workerv1.Agent_AGENT_CLAUDE_CODE,
	AgentTypeCodex:      workerv1.Agent_AGENT_CODEX,
	AgentTypeOpenCode:   workerv1.Agent_AGENT_OPENCODE,
	AgentTypeAmp:        workerv1.Agent_AGENT_AMP,
	AgentTypeGemini:     workerv1.Agent_AGENT_GEMINI,
}

var protoToAgentType = map[workerv1.Agent]AgentType{
	workerv1.Agent_AGENT_CLAUDE_CODE: AgentTypeClaudeCode,
	workerv1.Agent_AGENT_CODEX:      AgentTypeCodex,
	workerv1.Agent_AGENT_OPENCODE:   AgentTypeOpenCode,
	workerv1.Agent_AGENT_AMP:        AgentTypeAmp,
	workerv1.Agent_AGENT_GEMINI:     AgentTypeGemini,
}

// ProtoAgent converts an AgentType to its proto enum value.
func (a AgentType) ProtoAgent() workerv1.Agent {
	if v, ok := agentTypeToProto[a]; ok {
		return v
	}
	return workerv1.Agent_AGENT_UNSPECIFIED
}

// AgentTypeFromProto converts a proto Agent enum to an AgentType.
func AgentTypeFromProto(a workerv1.Agent) (AgentType, error) {
	if v, ok := protoToAgentType[a]; ok {
		return v, nil
	}
	return "", fmt.Errorf("unknown proto agent: %v", a)
}

// ParseProtoAgent converts a string agent name (e.g. "claude-code") to the proto enum.
func ParseProtoAgent(name string) (workerv1.Agent, error) {
	at := AgentType(name)
	if v, ok := agentTypeToProto[at]; ok {
		return v, nil
	}
	return workerv1.Agent_AGENT_UNSPECIFIED, fmt.Errorf("unknown agent: %s", name)
}
