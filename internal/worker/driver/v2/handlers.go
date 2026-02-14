package v2

import (
	"context"

	acp "github.com/coder/acp-go-sdk"
)

// ClientHandlers holds optional handler implementations for client-side ACP capabilities.
// Nil handlers will return "unsupported" errors.
type ClientHandlers struct {
	FS       FileSystemHandler
	Terminal TerminalHandler
}

// FileSystemHandler handles file system operations requested by the agent.
type FileSystemHandler interface {
	ReadTextFile(ctx context.Context, req acp.ReadTextFileRequest) (acp.ReadTextFileResponse, error)
	WriteTextFile(ctx context.Context, req acp.WriteTextFileRequest) (acp.WriteTextFileResponse, error)
}

// TerminalHandler handles terminal operations requested by the agent.
type TerminalHandler interface {
	CreateTerminal(ctx context.Context, req acp.CreateTerminalRequest) (acp.CreateTerminalResponse, error)
	KillTerminalCommand(ctx context.Context, req acp.KillTerminalCommandRequest) (acp.KillTerminalCommandResponse, error)
	TerminalOutput(ctx context.Context, req acp.TerminalOutputRequest) (acp.TerminalOutputResponse, error)
	ReleaseTerminal(ctx context.Context, req acp.ReleaseTerminalRequest) (acp.ReleaseTerminalResponse, error)
	WaitForTerminalExit(ctx context.Context, req acp.WaitForTerminalExitRequest) (acp.WaitForTerminalExitResponse, error)
}
