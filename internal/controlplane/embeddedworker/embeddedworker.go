package embeddedworker

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"log/slog"
	"net/http"

	appconfig "github.com/sebastianm/flowgentic/internal/config"
	"github.com/sebastianm/flowgentic/internal/proto/gen/controlplane/v1/controlplanev1connect"
)

// EmbeddedWorkerConfigStore persists the embedded worker's generated secret.
type EmbeddedWorkerConfigStore interface {
	GetSecret(ctx context.Context) (string, error)
	UpsertSecret(ctx context.Context, secret string) error
}

// embeddedWorkerConfigStoreFactory is registered by the store subpackage.
var embeddedWorkerConfigStoreFactory func(db *sql.DB) EmbeddedWorkerConfigStore

// RegisterEmbeddedWorkerConfigStoreFactory is called by the store subpackage
// to provide a store constructor, avoiding an import cycle.
func RegisterEmbeddedWorkerConfigStoreFactory(f func(db *sql.DB) EmbeddedWorkerConfigStore) {
	embeddedWorkerConfigStoreFactory = f
}

// DefaultPreferredPort is the port the embedded worker tries to bind first.
const DefaultPreferredPort = 19542

// StartDeps holds the dependencies needed by the embedded worker feature.
type StartDeps struct {
	Mux        *http.ServeMux
	Log        *slog.Logger
	DB         *sql.DB
	Config     appconfig.EmbeddedWorkerConfig
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

	configStore := embeddedWorkerConfigStoreFactory(d.DB)
	secret := resolveSecret(context.Background(), d.Log, d.Config, configStore)

	svc := NewEmbeddedWorkerService(
		d.Log,
		binaryPath,
		secret,
		DefaultPreferredPort,
		d.ConfigPath,
		d.Registry,
		d.Store,
	)

	h := &embeddedWorkerServiceHandler{log: d.Log, svc: svc}
	d.Mux.Handle(controlplanev1connect.NewEmbeddedWorkerServiceHandler(h))

	return svc
}

// resolveSecret loads the secret from the database. If none exists, it
// generates a new one and persists it. An explicit secret in the file config
// takes priority.
func resolveSecret(
	ctx context.Context,
	log *slog.Logger,
	fileCfg appconfig.EmbeddedWorkerConfig,
	cs EmbeddedWorkerConfigStore,
) string {
	if fileCfg.Secret != "" {
		return fileCfg.Secret
	}

	secret, err := cs.GetSecret(ctx)
	if err == nil && secret != "" {
		log.Info("loaded embedded worker secret from database")
		return secret
	}

	// First run — generate and persist.
	secret = generateSecret()
	if err := cs.UpsertSecret(ctx, secret); err != nil {
		log.Error("failed to persist embedded worker secret", "error", err)
	} else {
		log.Info("generated and persisted new embedded worker secret")
	}

	return secret
}

func generateSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("failed to generate random secret: " + err.Error())
	}
	return hex.EncodeToString(b)
}
