package task

import (
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
	Mux *http.ServeMux
	Log *slog.Logger
	DB  *sql.DB
}

type Feature struct {
	Service *TaskService
}

func Start(d StartDeps) *Feature {
	st := storeFactory(d.DB)
	svc := NewTaskService(st)
	h := &taskServiceHandler{log: d.Log, svc: svc}
	d.Mux.Handle(controlplanev1connect.NewTaskServiceHandler(h))
	return &Feature{Service: svc}
}
