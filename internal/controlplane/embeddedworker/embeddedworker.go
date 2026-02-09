package embeddedworker

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"

	"github.com/sebastianm/flowgentic/internal/config"
	"github.com/sebastianm/flowgentic/internal/proto/gen/controlplane/v1/controlplanev1connect"
)

// StartDeps holds the dependencies needed by the embedded worker feature.
type StartDeps struct {
	Mux        *http.ServeMux
	Log        *slog.Logger
	Config     config.EmbeddedWorkerConfig
	Registry   RegistryUpdater
	Store      Store
	ConfigPath string
}

// Start resolves config defaults, creates the service, and registers the RPC
// handler. The returned *EmbeddedWorkerService can be used to auto-start the
// worker after the HTTP server is listening.
func Start(d StartDeps) *EmbeddedWorkerService {
	binaryPath := d.Config.BinaryPath
	if binaryPath == "" {
		binaryPath = "flowgentic-worker"
	}

	secret := d.Config.Secret
	if secret == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			d.Log.Error("failed to generate embedded worker secret", "error", err)
		}
		secret = hex.EncodeToString(b)
	}

	svc := NewEmbeddedWorkerService(
		d.Log,
		binaryPath,
		secret,
		d.ConfigPath,
		d.Registry,
		d.Store,
	)

	h := &embeddedWorkerServiceHandler{log: d.Log, svc: svc}
	d.Mux.Handle(controlplanev1connect.NewEmbeddedWorkerServiceHandler(h))

	return svc
}
