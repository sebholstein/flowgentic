package server

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"

	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	"connectrpc.com/validate"
	"github.com/sebastianm/flowgentic/internal/config"
	workerv1connect "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1/workerv1connect"
	"github.com/sebastianm/flowgentic/internal/tsnetutil"
	"github.com/sebastianm/flowgentic/internal/worker/agentctl"
	"github.com/sebastianm/flowgentic/internal/worker/driver"
	"github.com/sebastianm/flowgentic/internal/worker/driver/claude"
	"github.com/sebastianm/flowgentic/internal/worker/driver/codex"
	"github.com/sebastianm/flowgentic/internal/worker/driver/gemini"
	"github.com/sebastianm/flowgentic/internal/worker/driver/opencode"
	"github.com/sebastianm/flowgentic/internal/worker/interceptors"
	"github.com/sebastianm/flowgentic/internal/worker/systeminfo"
	"github.com/sebastianm/flowgentic/internal/worker/systeminfo/agentinfo"
	"github.com/sebastianm/flowgentic/internal/worker/workload"
)

// Opts holds optional CLI overrides for the worker server.
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
	log := slog.New(slog.NewTextHandler(os.Stdout, nil)).With("component", "flowgentic-worker")
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

	w := cfg.Worker
	s.log.Info("Starting flowgentic-worker", "tailscale_enabled", w.Tailscale.Enabled)

	// --- Public listener (Tailscale-aware) ---

	listenAddr := s.opts.ListenAddr
	if listenAddr == "" {
		listenAddr = fmt.Sprintf(":%d", w.Port)
	}

	ln, err := tsnetutil.ListenAddr(listenAddr, w.Tailscale)
	if err != nil {
		s.log.Error("listen failed", "addr", listenAddr, "error", err)
		return err
	}
	s.ln = ln
	defer s.ln.Close()

	secret := os.Getenv("FLOWGENTIC_WORKER_SECRET")
	if secret == "" {
		s.log.Error("FLOWGENTIC_WORKER_SECRET environment variable is required")
		return fmt.Errorf("FLOWGENTIC_WORKER_SECRET environment variable is required")
	}

	validateInterceptor := validate.NewInterceptor()

	publicAuth := connect.WithInterceptors(interceptors.NewAuth(secret), validateInterceptor)

	publicMux := http.NewServeMux()

	systeminfo.Start(systeminfo.StartDeps{
		Mux:          publicMux,
		Log:          s.log,
		Interceptors: publicAuth,
		Agents:       agentinfo.NewDiscoverer(),
	})

	// --- Private CTL listener (localhost-only) ---

	ctlSecret, err := generateCtlSecret()
	if err != nil {
		return fmt.Errorf("generate ctl secret: %w", err)
	}

	ctlLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		s.log.Error("ctl listen failed", "error", err)
		return fmt.Errorf("ctl listen: %w", err)
	}
	defer ctlLn.Close()

	ctlAuth := connect.WithInterceptors(interceptors.NewAuth(ctlSecret), validateInterceptor)
	ctlMux := http.NewServeMux()

	ctlURL := fmt.Sprintf("http://%s", ctlLn.Addr().String())

	// Create drivers and the AgentRunManager via workload.Start().
	mgr := workload.Start(workload.StartDeps{
		Mux:          publicMux,
		Log:          s.log,
		Interceptors: publicAuth,
		Drivers: []driver.Driver{
			claude.NewDriver(claude.DriverDeps{Log: s.log}),
			codex.NewDriver(codex.DriverDeps{Log: s.log}),
			opencode.NewDriver(opencode.DriverDeps{Log: s.log}),
			gemini.NewDriver(gemini.DriverDeps{Log: s.log}),
		},
		CtlURL:    ctlURL,
		CtlSecret: ctlSecret,
	})

	// Wire agentctl RPC handlers, passing the AgentRunManager as EventHandler.
	agentctl.Start(agentctl.StartDeps{
		Mux:          ctlMux,
		Log:          s.log,
		Interceptors: ctlAuth,
		Handler:      mgr,
	})

	publicReflector := grpcreflect.NewStaticReflector(
		workerv1connect.SystemServiceName,
		workerv1connect.WorkerServiceName,
	)
	publicMux.Handle(grpcreflect.NewHandlerV1(publicReflector))
	publicMux.Handle(grpcreflect.NewHandlerV1Alpha(publicReflector))

	ctlReflector := grpcreflect.NewStaticReflector(
		workerv1connect.HookCtlServiceName,
		workerv1connect.AgentCtlServiceName,
	)
	ctlMux.Handle(grpcreflect.NewHandlerV1(ctlReflector))
	ctlMux.Handle(grpcreflect.NewHandlerV1Alpha(ctlReflector))

	// --- Start both servers ---

	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)

	s.log.Info("HTTP server listening",
		"public_addr", s.ln.Addr().String(),
		"ctl_addr", ctlLn.Addr().String(),
		"tailscale_enabled", w.Tailscale.Enabled,
		"https", w.Tailscale.HTTPS,
	)

	// Run private CTL server in background goroutine.
	ctlSrv := &http.Server{Handler: ctlMux, Protocols: protocols}
	go func() {
		if err := ctlSrv.Serve(ctlLn); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.log.Error("ctl serve error", "error", err)
		}
	}()

	// Run public server (blocking).
	publicSrv := &http.Server{Handler: publicMux, Protocols: protocols}
	if err := publicSrv.Serve(s.ln); err != nil {
		s.log.Error("serve error", "error", err)
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

// generateCtlSecret returns a 32-byte random hex string for the private CTL listener.
func generateCtlSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("crypto/rand: %w", err)
	}
	return hex.EncodeToString(b), nil
}
