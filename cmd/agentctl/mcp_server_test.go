package main

import (
	"bytes"
	"context"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPServerHandshake(t *testing.T) {
	srv := newMCPServer()
	session := newMCPTestSession(t, srv)

	initResult := session.InitializeResult()
	require.NotNil(t, initResult)
	require.NotNil(t, initResult.ServerInfo)
	assert.Equal(t, "agentctl", initResult.ServerInfo.Name)
	assert.NotEmpty(t, initResult.ProtocolVersion)
}

func TestMCPServerToolsList(t *testing.T) {
	srv := newMCPServer()
	session := newMCPTestSession(t, srv)

	res, err := session.ListTools(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, res)

	names := make([]string, 0, len(res.Tools))
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
	}

	assert.ElementsMatch(t, []string{
		"set_topic",
		"ask_question",
		"plan_get_current_dir",
		"plan_request_thread_dir",
		"plan_remove_thread",
		"plan_clear_current",
		"plan_commit",
	}, names)
}

func TestMCPServerToolCall_SetTopic(t *testing.T) {
	srv := newMCPServer()
	var gotTopic string
	srv.setTopicFn = func(_ context.Context, topic string) error {
		gotTopic = topic
		return nil
	}
	session := newMCPTestSession(t, srv)

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "set_topic",
		Arguments: map[string]any{"topic": "MCP rollout"},
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.False(t, res.IsError)
	assert.Equal(t, "MCP rollout", gotTopic)
	assert.Contains(t, firstTextContent(res), "Topic set successfully")
}

func TestMCPServerToolCall_ValidationError(t *testing.T) {
	srv := newMCPServer()
	session := newMCPTestSession(t, srv)

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "plan_remove_thread",
		Arguments: map[string]any{},
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "thread_id")
	assert.Nil(t, res)
}

func TestMCPServerToolCall_AskQuestion(t *testing.T) {
	srv := newMCPServer()
	srv.askQuestionFn = func(_ context.Context, question string) (string, error) {
		return "mocked: " + question, nil
	}
	session := newMCPTestSession(t, srv)

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "ask_question",
		Arguments: map[string]any{"question": "What genre should we target?"},
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.False(t, res.IsError)
	assert.Equal(t, "mocked: What genre should we target?", firstTextContent(res))
}

func newMCPTestSession(t *testing.T, srv *mcpServer) *mcp.ClientSession {
	t.Helper()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)

	go func() {
		errCh <- srv.server.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "agentctl-test-client",
		Version: "1.0.0",
	}, nil)

	session, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = session.Close()
		cancel()
		_ = <-errCh
	})

	return session
}

func firstTextContent(res *mcp.CallToolResult) string {
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			return strings.TrimSpace(tc.Text)
		}
	}
	return ""
}

func TestAdaptInput_ContentLengthFraming(t *testing.T) {
	payload := `{"jsonrpc":"2.0","id":1,"method":"ping","params":{}}`
	in := "Content-Length: " + strconv.Itoa(len(payload)) + "\r\n\r\n" + payload + "\n"

	pr, pw := io.Pipe()
	modeCh := make(chan bool, 1)
	errCh := make(chan error, 1)

	go func() {
		errCh <- adaptInput(strings.NewReader(in), pw, modeCh)
	}()

	mode := <-modeCh
	assert.True(t, mode)

	out, err := io.ReadAll(pr)
	require.NoError(t, err)
	assert.Equal(t, payload+"\n", string(out))
	err = <-errCh
	if err != nil {
		assert.ErrorIs(t, err, io.EOF)
	}
}

func TestAdaptOutput_ContentLengthFraming(t *testing.T) {
	var out bytes.Buffer
	err := adaptOutput(strings.NewReader("{\"ok\":true}\n"), &out, true)
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Content-Length:")
	assert.Contains(t, out.String(), "{\"ok\":true}")
}
