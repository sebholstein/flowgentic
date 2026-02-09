package tsnetutil

import (
	"crypto/tls"
	"fmt"
	"net"

	"github.com/sebastianm/flowgentic/internal/config"
	"tailscale.com/client/local"
	"tailscale.com/tsnet"
)

// Listener wraps a net.Listener with optional Tailscale resources.
type Listener struct {
	net.Listener

	// TS is the underlying tsnet.Server (nil when Tailscale is disabled).
	TS *tsnet.Server

	// LC is the Tailscale local client (nil when Tailscale is disabled).
	LC *local.Client
}

// Close tears down the listener and, if present, the tsnet server.
func (l *Listener) Close() error {
	err := l.Listener.Close()
	if l.TS != nil {
		if tsErr := l.TS.Close(); tsErr != nil && err == nil {
			err = tsErr
		}
	}
	return err
}

// Listen creates a Listener on the given port, optionally using Tailscale.
// When tsCfg.Enabled is false a plain TCP listener is returned.
func Listen(port int, tsCfg config.TailscaleConfig) (*Listener, error) {
	return ListenAddr(fmt.Sprintf(":%d", port), tsCfg)
}

// ListenAddr creates a Listener on the given address, optionally using Tailscale.
// When tsCfg.Enabled is false a plain TCP listener is returned.
func ListenAddr(addr string, tsCfg config.TailscaleConfig) (*Listener, error) {
	if !tsCfg.Enabled {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("tcp listen on %s: %w", addr, err)
		}
		return &Listener{Listener: ln}, nil
	}

	ts := new(tsnet.Server)
	ts.Hostname = tsCfg.Hostname
	ts.Ephemeral = tsCfg.Ephemeral
	ts.AuthKey = tsCfg.AuthKey
	ts.ControlURL = tsCfg.ControlURL
	if tsCfg.Dir != "" {
		ts.Dir = tsCfg.Dir
	}

	if err := ts.Start(); err != nil {
		return nil, fmt.Errorf("starting tsnet server: %w", err)
	}

	lc, err := ts.LocalClient()
	if err != nil {
		ts.Close()
		return nil, fmt.Errorf("getting tailscale local client: %w", err)
	}

	ln, err := ts.Listen("tcp", addr)
	if err != nil {
		ts.Close()
		return nil, fmt.Errorf("tsnet listen on %s: %w", addr, err)
	}

	// Wrap with TLS for automatic HTTPS via Tailscale-managed Let's Encrypt certificates.
	var netLn net.Listener = ln
	if tsCfg.HTTPS {
		netLn = tls.NewListener(ln, &tls.Config{
			GetCertificate: lc.GetCertificate,
		})
	}

	return &Listener{
		Listener: netLn,
		TS:       ts,
		LC:       lc,
	}, nil
}
