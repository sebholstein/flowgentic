package server

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"connectrpc.com/grpcreflect"
	"github.com/sebastianm/flowgentic/internal/config"
	"github.com/sebastianm/flowgentic/internal/controlplane/embeddedworker"
	"github.com/sebastianm/flowgentic/internal/controlplane/project"
	projectstore "github.com/sebastianm/flowgentic/internal/controlplane/project/store"
	"github.com/sebastianm/flowgentic/internal/controlplane/relay"
	"github.com/sebastianm/flowgentic/internal/controlplane/thread"
	threadstore "github.com/sebastianm/flowgentic/internal/controlplane/thread/store"
	"github.com/sebastianm/flowgentic/internal/controlplane/worker"
	"github.com/sebastianm/flowgentic/internal/controlplane/worker/store"
	"github.com/sebastianm/flowgentic/internal/database"
	controlplanev1connect "github.com/sebastianm/flowgentic/internal/proto/gen/controlplane/v1/controlplanev1connect"
	"github.com/sebastianm/flowgentic/internal/tsnetutil"
)

// Opts holds optional CLI overrides for the control plane server.
type Opts struct {
	ListenAddr string
}

type Server struct {
	log  *slog.Logger
	cfg  *config.Config
	ln   *tsnetutil.Listener
	opts Opts
}

func New(opts Opts) *Server {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil)).With("component", "flowgentic-control-plane")
	return &Server{
		log:  log,
		opts: opts,
	}
}

func (s *Server) Start() error {
	cfg, err := config.Parse()
	if err != nil {
		s.log.Error("config error", "error", err)
		return fmt.Errorf("config error: %w", err)
	}
	s.cfg = cfg

	cp := cfg.ControlPlane
	s.log.Info("Starting flowgentic-control-plane", "tailscale_enabled", cp.Tailscale.Enabled)

	listenAddr := s.opts.ListenAddr
	if listenAddr == "" {
		listenAddr = fmt.Sprintf(":%d", cp.Port)
	}

	ln, err := tsnetutil.ListenAddr(listenAddr, cp.Tailscale)
	if err != nil {
		s.log.Error("listen failed", "addr", listenAddr, "error", err)
		return err
	}
	s.ln = ln
	defer s.ln.Close()

	// Open database.
	dbPath := cp.DatabasePath
	if dbPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("determining home directory: %w", err)
		}
		dbPath = filepath.Join(home, ".flowgentic", "flowgentic.db")
	}

	db, err := database.Open(context.Background(), dbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()
	s.log.Info("database opened", "path", dbPath)

	// Create store and seed config-file workers.
	workerStore := store.NewSQLiteStore(db)
	s.seedWorkers(workerStore, cp.Workers)

	// Load all workers from DB for the relay.
	dbWorkers, err := workerStore.ListWorkers(context.Background())
	if err != nil {
		return fmt.Errorf("loading workers from database: %w", err)
	}
	relayWorkers := make([]config.WorkerEndpoint, len(dbWorkers))
	for i, w := range dbWorkers {
		relayWorkers[i] = config.WorkerEndpoint{
			ID:     w.ID,
			URL:    w.URL,
			Secret: w.Secret,
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "flowgentic-control-plane OK")
	})
	if s.ln.LC != nil {
		mux.HandleFunc("/whoami", s.handleWhoAmI)
	}

	registry := relay.Start(relay.StartDeps{
		Mux:     mux,
		Log:     s.log,
		Workers: relayWorkers,
	})

	worker.Start(worker.StartDeps{
		Mux:          mux,
		Log:          s.log,
		Store:        workerStore,
		Registry:     registry,
		PingRegistry: registry,
	})

	// Wire up project management feature.
	projectStore := projectstore.NewSQLiteStore(db)
	project.Start(project.StartDeps{
		Mux:   mux,
		Log:   s.log,
		Store: projectStore,
	})

	// Wire up thread management feature.
	threadStore := threadstore.NewSQLiteStore(db)
	thread.Start(thread.StartDeps{
		Mux:   mux,
		Log:   s.log,
		Store: threadStore,
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
			Store:      workerStore,
			ConfigPath: configPath,
		})
	}

	// Register gRPC reflection for all control plane services.
	reflector := grpcreflect.NewStaticReflector(
		controlplanev1connect.WorkerManagementServiceName,
		controlplanev1connect.ProjectManagementServiceName,
		controlplanev1connect.ThreadManagementServiceName,
		controlplanev1connect.EmbeddedWorkerServiceName,
	)
	mux.Handle(grpcreflect.NewHandlerV1(reflector))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))

	handler := corsMiddleware(mux)

	s.log.Info("HTTP server listening",
		"addr", s.ln.Addr().String(),
		"tailscale_enabled", cp.Tailscale.Enabled,
		"https", cp.Tailscale.HTTPS,
	)

	// Auto-start embedded worker after server is listening.
	if embeddedSvc != nil {
		go func() {
			time.Sleep(100 * time.Millisecond)
			if err := embeddedSvc.Start(context.Background()); err != nil {
				s.log.Error("embedded worker auto-start failed", "error", err)
			}
		}()
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		s.log.Info("received signal, shutting down", "signal", sig)
		if embeddedSvc != nil {
			if err := embeddedSvc.Stop(context.Background()); err != nil {
				s.log.Warn("error stopping embedded worker during shutdown", "error", err)
			}
		}
		s.ln.Close()
	}()

	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)

	srv := &http.Server{Handler: handler, Protocols: protocols}
	if err := srv.Serve(s.ln); err != nil {
		s.log.Error("serve error", "error", err)
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

// seedWorkers idempotently inserts config-file workers that don't already exist
// in the database.
func (s *Server) seedWorkers(st *store.SQLiteStore, cfgWorkers []config.WorkerEndpoint) {
	ctx := context.Background()

	for _, w := range cfgWorkers {
		s.seedWorker(ctx, st, worker.Worker{
			ID:     w.ID,
			Name:   w.ID,
			URL:    w.URL,
			Secret: w.Secret,
		})
	}
}

func (s *Server) seedWorker(ctx context.Context, st *store.SQLiteStore, w worker.Worker) {
	existing, err := st.GetWorker(ctx, w.ID)
	if err == nil {
		// Worker exists — update URL/secret in case config changed.
		if existing.URL != w.URL || existing.Secret != w.Secret {
			w.Name = existing.Name
			if _, err := st.UpdateWorker(ctx, w); err != nil {
				s.log.Error("failed to update seeded worker", "id", w.ID, "error", err)
			}
		}
		return
	}
	if _, err := st.CreateWorker(ctx, w); err != nil {
		s.log.Error("failed to seed worker", "id", w.ID, "error", err)
	}
}

// handleWhoAmI uses the Tailscale LocalClient to identify the caller.
func (s *Server) handleWhoAmI(w http.ResponseWriter, r *http.Request) {
	who, err := s.ln.LC.WhoIs(r.Context(), r.RemoteAddr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	login := html.EscapeString(who.UserProfile.LoginName)
	firstLabel, _, _ := strings.Cut(who.Node.ComputedName, ".")
	node := html.EscapeString(firstLabel)
	fmt.Fprintf(w, "You are %s from %s (%s)\n", login, node, r.RemoteAddr)
}
