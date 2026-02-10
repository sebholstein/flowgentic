package server

import (
	"context"
	"database/sql"
	"net/http"
	"os"

	"connectrpc.com/grpcreflect"
	"github.com/sebastianm/flowgentic/internal/config"
	"github.com/sebastianm/flowgentic/internal/controlplane/agentrun"
	_ "github.com/sebastianm/flowgentic/internal/controlplane/agentrun/store" // registers store factory
	"github.com/sebastianm/flowgentic/internal/controlplane/embeddedworker"
	"github.com/sebastianm/flowgentic/internal/controlplane/project"
	_ "github.com/sebastianm/flowgentic/internal/controlplane/project/store" // registers store factory
	"github.com/sebastianm/flowgentic/internal/controlplane/relay"
	"github.com/sebastianm/flowgentic/internal/controlplane/thread"
	_ "github.com/sebastianm/flowgentic/internal/controlplane/thread/store" // registers store factory
	"github.com/sebastianm/flowgentic/internal/controlplane/worker"
	_ "github.com/sebastianm/flowgentic/internal/controlplane/worker/store" // registers store factory
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

	// Wire up agent run feature (must come before thread, which needs AgentRunCreator).
	serverCtx, serverCancel := context.WithCancel(context.Background())
	agentRunSvc := agentrun.Start(serverCtx, agentrun.StartDeps{
		Mux:      mux,
		Log:      s.log,
		DB:       db,
		Registry: registry,
	})

	// Wire up thread feature.
	thread.Start(thread.StartDeps{
		Mux:             mux,
		Log:             s.log,
		DB:              db,
		AgentRunCreator: agentRunSvc,
	})

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
			Config:     cp.EmbeddedWorker,
			Registry:   registry,
			Store:      workerFeature.Store,
			ConfigPath: configPath,
		})
	}

	// Register gRPC reflection for all control plane services.
	reflector := grpcreflect.NewStaticReflector(
		controlplanev1connect.WorkerServiceName,
		controlplanev1connect.ProjectServiceName,
		controlplanev1connect.ThreadServiceName,
		controlplanev1connect.AgentRunServiceName,
		controlplanev1connect.EmbeddedWorkerServiceName,
	)
	mux.Handle(grpcreflect.NewHandlerV1(reflector))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))

	return &featureSet{
		embeddedSvc:  embeddedSvc,
		serverCancel: serverCancel,
	}
}
