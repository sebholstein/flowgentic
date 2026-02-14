package systeminfo

import (
	"context"
	"fmt"
	"testing"
	"time"

	acp "github.com/coder/acp-go-sdk"
	"github.com/sebastianm/flowgentic/internal/worker/driver"
	v2 "github.com/sebastianm/flowgentic/internal/worker/driver/v2"
	"github.com/sebastianm/flowgentic/internal/worker/systeminfo/agentinfo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAgentInfo struct{}

func (fakeAgentInfo) DiscoverAgents(context.Context, bool) ([]agentinfo.Agent, error) {
	return nil, nil
}

type fakeModelDriver struct {
	agent string
	inv   v2.ModelInventory
	err   error
	calls int
}

func (d *fakeModelDriver) Agent() string { return d.agent }
func (d *fakeModelDriver) Capabilities() driver.Capabilities {
	return driver.Capabilities{Agent: d.agent}
}
func (d *fakeModelDriver) Launch(context.Context, v2.LaunchOpts, v2.EventCallback) (v2.Session, error) {
	return nil, fmt.Errorf("not implemented")
}
func (d *fakeModelDriver) DiscoverModels(_ context.Context, _ string) (v2.ModelInventory, error) {
	d.calls++
	if d.err != nil {
		return v2.ModelInventory{}, d.err
	}
	return d.inv, nil
}

type fakeSession struct{}

func (fakeSession) Info() v2.SessionInfo { return v2.SessionInfo{} }
func (fakeSession) Prompt(context.Context, []acp.ContentBlock) (*acp.PromptResponse, error) {
	return nil, nil
}
func (fakeSession) Cancel(context.Context) error                                    { return nil }
func (fakeSession) Stop(context.Context) error                                      { return nil }
func (fakeSession) Wait(context.Context) error                                      { return nil }
func (fakeSession) RespondToPermission(context.Context, string, bool, string) error { return nil }
func (fakeSession) SetSessionMode(context.Context, driver.SessionMode) error        { return nil }

func TestSystemInfoService_GetAgentModels_CacheBehavior(t *testing.T) {
	now := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)
	d := &fakeModelDriver{
		agent: string(driver.AgentTypeCodex),
		inv: v2.ModelInventory{
			Models:       []string{"gpt-5", "gpt-5-mini"},
			DefaultModel: "gpt-5",
		},
	}
	svc := NewSystemInfoService(fakeAgentInfo{}, []v2.Driver{d}, "/tmp")
	svc.now = func() time.Time { return now }

	first, err := svc.GetAgentModels(context.Background(), driver.AgentTypeCodex, false)
	require.NoError(t, err)
	assert.Equal(t, []string{"gpt-5", "gpt-5-mini"}, first.Models)
	assert.Equal(t, "gpt-5", first.DefaultModel)
	assert.Equal(t, 1, d.calls)

	second, err := svc.GetAgentModels(context.Background(), driver.AgentTypeCodex, false)
	require.NoError(t, err)
	assert.Equal(t, 1, d.calls)
	assert.Equal(t, first, second)

	_, err = svc.GetAgentModels(context.Background(), driver.AgentTypeCodex, true)
	require.NoError(t, err)
	assert.Equal(t, 2, d.calls)

	now = now.Add(61 * time.Second)
	_, err = svc.GetAgentModels(context.Background(), driver.AgentTypeCodex, false)
	require.NoError(t, err)
	assert.Equal(t, 3, d.calls)
}
