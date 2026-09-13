// Package httpapi wires up voxa-api's HTTP routes.
package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/maksim/voxa-api/internal/config"
	"github.com/maksim/voxa-api/internal/db"
	"github.com/maksim/voxa-api/internal/relay"
)

type Server struct {
	cfg   config.Config
	db    *db.DB
	relay *relay.Client
}

func New(cfg config.Config, database *db.DB) *Server {
	return &Server{cfg: cfg, db: database, relay: relay.NewClient(cfg.StorageRelayURL)}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("POST /auth/register", s.handleRegister)
	mux.HandleFunc("POST /auth/login", s.handleLogin)

	mux.HandleFunc("POST /jobs", s.requireAuth(s.handleCreateJob))
	mux.HandleFunc("GET /jobs", s.requireAuth(s.handleListJobs))
	mux.HandleFunc("GET /jobs/{id}", s.requireAuth(s.handleGetJob))
	mux.HandleFunc("GET /jobs/{id}/result", s.requireAuth(s.handleGetResult))

	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
