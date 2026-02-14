package server

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"connectrpc.com/grpcreflect"
	"github.com/sebastianm/flowgentic/internal/config"
	"github.com/sebastianm/flowgentic/internal/controlplane/session"
	_ "github.com/sebastianm/flowgentic/internal/controlplane/session/store"        // registers store factory
	"github.com/sebastianm/flowgentic/internal/controlplane/embeddedworker"
	_ "github.com/sebastianm/flowgentic/internal/controlplane/embeddedworker/store" // registers store factory
	"github.com/sebastianm/flowgentic/internal/controlplane/project"
	_ "github.com/sebastianm/flowgentic/internal/controlplane/project/store" // registers store factory
	"github.com/sebastianm/flowgentic/internal/controlplane/relay"
	"github.com/sebastianm/flowgentic/internal/controlplane/task"
	_ "github.com/sebastianm/flowgentic/internal/controlplane/task/store" // registers store factory
	"github.com/sebastianm/flowgentic/internal/controlplane/thread"
	_ "github.com/sebastianm/flowgentic/internal/controlplane/thread/store" // registers store factory
	"github.com/sebastianm/flowgentic/internal/controlplane/worker"
	_ "github.com/sebastianm/flowgentic/internal/controlplane/worker/store" // registers store factory
	controlplanev1 "github.com/sebastianm/flowgentic/internal/proto/gen/controlplane/v1"
	controlplanev1connect "github.com/sebastianm/flowgentic/internal/proto/gen/controlplane/v1/controlplanev1connect"
)

type featureSet struct {
	embeddedSvc  *embeddedworker.EmbeddedWorkerService
	serverCancel context.CancelFunc
}

func (s *Server) startFeatures(mux *http.ServeMux, db *sql.DB, cp config.ControlPlaneConfig) *featureSet {
	// Start relay first (empty registry — workers are added by the worker feature).
	registry := relay.Start(relay.StartDeps{
		Mux: mux,
		Log: s.log,
	})

	// Wire up worker feature (seeds config workers, populates relay registry).
	workerFeature := worker.Start(worker.StartDeps{
		Mux:          mux,
		Log:          s.log,
		DB:           db,
		Registry:     registry,
		PingRegistry: registry,
		SeedWorkers:  cp.Workers,
	})

	// Wire up project feature.
	project.Start(project.StartDeps{
		Mux: mux,
		Log: s.log,
		DB:  db,
	})

	// Wire up session feature (must come before thread, which needs SessionCreator).
	serverCtx, serverCancel := context.WithCancel(context.Background())
	sessionFeature := session.Start(serverCtx, session.StartDeps{
		Mux:      mux,
		Log:      s.log,
		DB:       db,
		Registry: registry,
	})

	// Wire up task feature.
	task.Start(task.StartDeps{
		Mux: mux,
		Log: s.log,
		DB:  db,
	})

	// Wire up thread feature.
	threadSvc := thread.Start(thread.StartDeps{
		Mux:             mux,
		Log:             s.log,
		DB:              db,
		SessionCreator: sessionFeature.Service,
	})

	// Start state sync watchers for all configured workers.
	stateSyncHandler := session.NewStateSyncHandler(s.log, sessionFeature.Store, threadSvc, sessionFeature.Service)
	for _, w := range cp.Workers {
		watcher := session.NewStateSyncWatcher(s.log, w.ID, w.URL, w.Secret, stateSyncHandler)
		go watcher.Run(serverCtx)
	}

	// Resolve config path for the embedded worker.
	configPath := os.Getenv("FLOWGENTIC_CONFIG")
	if configPath == "" {
		configPath = "flowgentic.json"
	}

	// Wire up embedded worker feature.
	var embeddedSvc *embeddedworker.EmbeddedWorkerService
	if cp.EmbeddedWorker.Enabled {
		embeddedSvc = embeddedworker.Start(embeddedworker.StartDeps{
			Mux:        mux,
			Log:        s.log,
			DB:         db,
			Config:     cp.EmbeddedWorker,
			Registry:   registry,
			Store:      workerFeature.Store,
			ConfigPath: configPath,
		})

		// Watch embedded worker status and start/stop a state sync watcher
		// when it comes up or goes down.
		go watchEmbeddedWorkerStateSync(serverCtx, s.log, embeddedSvc, stateSyncHandler)
	}

	// Register gRPC reflection for all control plane services.
	reflector := grpcreflect.NewStaticReflector(
		controlplanev1connect.WorkerServiceName,
		controlplanev1connect.ProjectServiceName,
		controlplanev1connect.TaskServiceName,
		controlplanev1connect.ThreadServiceName,
		controlplanev1connect.SessionServiceName,
		controlplanev1connect.EmbeddedWorkerServiceName,
	)
	mux.Handle(grpcreflect.NewHandlerV1(reflector))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))

	return &featureSet{
		embeddedSvc:  embeddedSvc,
		serverCancel: serverCancel,
	}
}

// watchEmbeddedWorkerStateSync subscribes to embedded worker status changes
// and starts/stops a StateSyncWatcher when the worker comes up or goes down.
func watchEmbeddedWorkerStateSync(
	ctx context.Context,
	log *slog.Logger,
	svc *embeddedworker.EmbeddedWorkerService,
	handler session.StateSyncHandler,
) {
	ch := svc.Subscribe()
	defer svc.Unsubscribe(ch)

	var cancel context.CancelFunc

	for {
		select {
		case <-ctx.Done():
			if cancel != nil {
				cancel()
			}
			return
		case <-ch:
			status, _, _, addr := svc.GetStatus()

			if status == controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_RUNNING && cancel == nil {
				secret := svc.Secret()
				url := fmt.Sprintf("http://%s", addr)
				watchCtx, watchCancel := context.WithCancel(ctx)
				cancel = watchCancel
				watcher := session.NewStateSyncWatcher(log, "local", url, secret, handler)
				go watcher.Run(watchCtx)
				log.Info("started state sync watcher for embedded worker", "addr", addr)
			} else if status != controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_RUNNING && cancel != nil {
				cancel()
				cancel = nil
				log.Info("stopped state sync watcher for embedded worker")
			}
		}
	}
}
