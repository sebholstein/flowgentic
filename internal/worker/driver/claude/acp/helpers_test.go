package acp

import (
	"context"
	"log/slog"
	"os"
	"sync"

	acpsdk "github.com/coder/acp-go-sdk"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// fakeUpdateSender captures ACP session notifications for assertion.
type fakeUpdateSender struct {
	mu      sync.Mutex
	updates []acpsdk.SessionNotification
}

func (f *fakeUpdateSender) SessionUpdate(_ context.Context, n acpsdk.SessionNotification) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updates = append(f.updates, n)
	return nil
}

func (f *fakeUpdateSender) allUpdates() []acpsdk.SessionNotification {
	f.mu.Lock()
	defer f.mu.Unlock()
	dst := make([]acpsdk.SessionNotification, len(f.updates))
	copy(dst, f.updates)
	return dst
}

// newTestAdapter creates an Adapter wired to a fakeUpdateSender.
func newTestAdapter() (*Adapter, *fakeUpdateSender) {
	fake := &fakeUpdateSender{}
	a := &Adapter{
		log:     testLogger(),
		updater: fake,
	}
	return a, fake
}
