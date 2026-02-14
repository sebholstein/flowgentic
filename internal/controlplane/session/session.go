package session

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/sebastianm/flowgentic/internal/proto/gen/controlplane/v1/controlplanev1connect"
)

var storeFactory func(db *sql.DB) Store

func RegisterStoreFactory(f func(db *sql.DB) Store) {
	storeFactory = f
}

type StartDeps struct {
	Mux                *http.ServeMux
	Log                *slog.Logger
	DB                 *sql.DB
	Registry           WorkerRegistry
	ThreadTopicUpdater ThreadTopicUpdater
}

type Feature struct {
	Service *SessionService
	Store   Store
}

func Start(ctx context.Context, d StartDeps) *Feature {
	st := storeFactory(d.DB)
	reconciler := NewReconciler(d.Log, st, d.Registry)
	svc := NewSessionService(st, reconciler, d.Registry)
	h := &sessionServiceHandler{log: d.Log, svc: svc, store: st, threadTopicUpdater: d.ThreadTopicUpdater}
	d.Mux.Handle(controlplanev1connect.NewSessionServiceHandler(h))

	go reconciler.Run(ctx)

	return &Feature{Service: svc, Store: st}
}
