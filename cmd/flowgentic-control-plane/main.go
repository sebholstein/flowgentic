package main

import (
	"flag"
	"os"

	"github.com/sebastianm/flowgentic/internal/controlplane/server"
)

func main() {
	listenAddr := flag.String("listen-addr", "", "Address to listen on (e.g. :8080)")
	flag.Parse()

	srv := server.New(server.Opts{
		ListenAddr: *listenAddr,
	})
	if err := srv.Start(); err != nil {
		os.Exit(1)
	}
}
