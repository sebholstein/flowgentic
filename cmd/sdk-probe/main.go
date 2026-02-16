// sdk-probe connects to Claude CLI via the Go SDK and exercises control protocol methods.
// Usage: go run ./cmd/sdk-probe
//
//	go run ./cmd/sdk-probe -discover   # test DiscoverModels via ACP driver
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	claudecode "github.com/sebastianm/flowgentic/internal/claude-agent-sdk-go"
	claudeacp "github.com/sebastianm/flowgentic/internal/worker/driver/claude/acp"
	v2 "github.com/sebastianm/flowgentic/internal/worker/driver/v2"
)

func main() {
	discover := flag.Bool("discover", false, "test DiscoverModels via ACP driver")
	flag.Parse()

	if *discover {
		runDiscoverModels()
		return
	}
	runSDKProbe()
}

func runSDKProbe() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Fprintln(os.Stderr, "=== sdk-probe: connecting to Claude CLI ===")

	client := claudecode.NewClient(
		claudecode.WithPermissionMode(claudecode.PermissionModeBypassPermissions),
		claudecode.WithStderrCallback(func(line string) {
			fmt.Fprintf(os.Stderr, "[stderr] %s\n", line)
		}),
		claudecode.WithDebugWriter(io.Discard),
	)

	if err := client.Connect(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL connect: %v\n", err)
		os.Exit(1)
	}
	defer client.Disconnect()
	fmt.Fprintln(os.Stderr, "OK   connected")

	// Give the CLI a moment to start up
	time.Sleep(2 * time.Second)

	// 1. SupportedModels
	fmt.Fprintln(os.Stderr, "\n--- SupportedModels ---")
	modelsCtx, modelsCancel := context.WithTimeout(ctx, 10*time.Second)
	defer modelsCancel()
	models, err := client.SupportedModels(modelsCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL SupportedModels: %v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "OK   %d models\n", len(models))
		for _, m := range models {
			fmt.Fprintf(os.Stderr, "     value=%s  display=%s  desc=%s\n", m.Value, m.DisplayName, m.Description)
		}
		b, _ := json.MarshalIndent(models, "", "  ")
		fmt.Println(string(b))
	}

	// 2. SupportedCommands
	fmt.Fprintln(os.Stderr, "\n--- SupportedCommands ---")
	cmdsCtx, cmdsCancel := context.WithTimeout(ctx, 10*time.Second)
	defer cmdsCancel()
	cmds, err := client.SupportedCommands(cmdsCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL SupportedCommands: %v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "OK   %d commands\n", len(cmds))
		for i, c := range cmds {
			if i >= 5 {
				fmt.Fprintf(os.Stderr, "     ... and %d more\n", len(cmds)-5)
				break
			}
			fmt.Fprintf(os.Stderr, "     /%s  %s\n", c.Name, c.Description)
		}
	}

	fmt.Fprintln(os.Stderr, "\n=== sdk-probe: done ===")
}

func runDiscoverModels() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cfg := v2.ClaudeCodeConfig
	cfg.AdapterFactory = claudeacp.NewAdapter

	drv := v2.NewDriver(log, cfg)

	fmt.Fprintln(os.Stderr, "=== DiscoverModels via ACP driver ===")
	inv, err := drv.DiscoverModels(ctx, ".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL DiscoverModels: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "OK   default=%s  models=%v\n", inv.DefaultModel, inv.Models)
	b, _ := json.MarshalIndent(inv, "", "  ")
	fmt.Println(string(b))

	fmt.Fprintln(os.Stderr, "\n=== done ===")
}
