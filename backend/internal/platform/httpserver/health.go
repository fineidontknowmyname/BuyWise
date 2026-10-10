package httpserver

import (
	"context"
	"net/http"
	"time"

	"selfemi/internal/platform/httpjson"
)

const readinessTimeout = 2 * time.Second

type Pinger interface {
	Ping(ctx context.Context) error
}

type Module interface {
	Register(mux *http.ServeMux)
}

type healthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database,omitempty"`
}

func NewRouter(db Pinger, modules ...Module) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", liveness)
	mux.HandleFunc("GET /readyz", readiness(db))
	for _, module := range modules {
		module.Register(mux)
	}
	return mux
}

func liveness(w http.ResponseWriter, _ *http.Request) {
	httpjson.Write(w, http.StatusOK, healthResponse{Status: "up"})
}

func readiness(db Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			httpjson.Write(w, http.StatusServiceUnavailable, healthResponse{Status: "degraded", Database: "down"})
			return
		}
		httpjson.Write(w, http.StatusOK, healthResponse{Status: "up", Database: "up"})
	}
}
