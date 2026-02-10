package server

import (
	"context"
	"flag"
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

	"github.com/sebastianm/flowgentic/internal/config"
	"github.com/sebastianm/flowgentic/internal/database"
	"github.com/sebastianm/flowgentic/internal/tsnetutil"
)

// opts holds optional CLI overrides for the control plane server.
type opts struct {
	ListenAddr string
}

type Server struct {
	log  *slog.Logger
	cfg  *config.Config
	ln   *tsnetutil.Listener
	opts opts
}

func New() *Server {
	listenAddr := flag.String("listen-addr", "127.0.0.1:8420", "Address to listen on (e.g. :8080)")
	flag.Parse()
	log := slog.New(slog.NewTextHandler(os.Stdout, nil)).With("component", "control-plane")

	return &Server{
		log: log,
		opts: opts{
			ListenAddr: *listenAddr,
		},
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
	s.log.Info("Starting control-plane", "tailscale_enabled", cp.Tailscale.Enabled)

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

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "flowgentic-control-plane OK")
	})
	if s.ln.LC != nil {
		mux.HandleFunc("/whoami", s.handleWhoAmI)
	}

	features := s.startFeatures(mux, db, cp)
	defer features.serverCancel()

	handler := corsMiddleware(mux)

	s.log.Info("HTTP server listening",
		"addr", s.ln.Addr().String(),
		"tailscale_enabled", cp.Tailscale.Enabled,
		"https", cp.Tailscale.HTTPS,
	)

	// Auto-start embedded worker after server is listening.
	if features.embeddedSvc != nil {
		go func() {
			time.Sleep(100 * time.Millisecond)
			if err := features.embeddedSvc.Start(context.Background()); err != nil {
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
		features.serverCancel() // stop reconciler and other background goroutines
		if features.embeddedSvc != nil {
			if err := features.embeddedSvc.Stop(context.Background()); err != nil {
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
