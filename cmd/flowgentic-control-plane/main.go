package main

import (
	"os"

	"github.com/sebastianm/flowgentic/internal/controlplane/server"
)

func main() {

	srv := server.New()
	if err := srv.Start(); err != nil {
		os.Exit(1)
	}
}
