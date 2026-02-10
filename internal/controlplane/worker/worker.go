package worker

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/sebastianm/flowgentic/internal/config"
	"github.com/sebastianm/flowgentic/internal/proto/gen/controlplane/v1/controlplanev1connect"
)

// storeFactory is registered by the store subpackage via RegisterStoreFactory.
var storeFactory func(db *sql.DB) Store

// RegisterStoreFactory is called by the store subpackage to provide a Store
// constructor, avoiding an import cycle.
func RegisterStoreFactory(f func(db *sql.DB) Store) {
	storeFactory = f
}

// Feature holds references that other features may need after Start.
type Feature struct {
	Store Store
}

// StartDeps holds the dependencies needed by the worker feature.
type StartDeps struct {
	Mux          *http.ServeMux
	Log          *slog.Logger
	DB           *sql.DB
	Registry     RegistryUpdater
	PingRegistry WorkerRegistry
	SeedWorkers  []config.WorkerEndpoint
}

// Start creates the worker store, seeds config-file workers, populates the
// relay registry, and registers the WorkerService RPC handler on the mux.
func Start(d StartDeps) *Feature {
	st := storeFactory(d.DB)

	// Seed config-file workers into the database.
	seedWorkers(d.Log, st, d.SeedWorkers)

	// Populate the relay registry from all DB workers.
	populateRegistry(d.Log, st, d.Registry)

	svc := NewWorkerService(st, d.Registry)
	pingSvc := NewPingService(d.PingRegistry)
	h := &workerServiceHandler{log: d.Log, svc: svc, pingSvc: pingSvc}
	d.Mux.Handle(controlplanev1connect.NewWorkerServiceHandler(h))

	return &Feature{Store: st}
}

// seedWorkers idempotently inserts config-file workers that don't already exist
// in the database.
func seedWorkers(log *slog.Logger, st Store, cfgWorkers []config.WorkerEndpoint) {
	ctx := context.Background()
	for _, w := range cfgWorkers {
		seedWorker(ctx, log, st, Worker{
			ID:     w.ID,
			Name:   w.ID,
			URL:    w.URL,
			Secret: w.Secret,
		})
	}
}

func seedWorker(ctx context.Context, log *slog.Logger, st Store, w Worker) {
	existing, err := st.GetWorker(ctx, w.ID)
	if err == nil {
		// Worker exists — update URL/secret in case config changed.
		if existing.URL != w.URL || existing.Secret != w.Secret {
			w.Name = existing.Name
			if _, err := st.UpdateWorker(ctx, w); err != nil {
				log.Error("failed to update seeded worker", "id", w.ID, "error", err)
			}
		}
		return
	}
	if _, err := st.CreateWorker(ctx, w); err != nil {
		log.Error("failed to seed worker", "id", w.ID, "error", err)
	}
}

// populateRegistry loads all workers from the DB and registers them with the
// relay registry so they are immediately routable.
func populateRegistry(log *slog.Logger, st Store, reg RegistryUpdater) {
	ctx := context.Background()
	workers, err := st.ListWorkers(ctx)
	if err != nil {
		log.Error("failed to load workers for relay registry", "error", err)
		return
	}
	for _, w := range workers {
		if err := reg.AddWorker(w.ID, w.URL, w.Secret); err != nil {
			log.Error("failed to register worker in relay", "id", w.ID, "error", err)
		}
	}
}
