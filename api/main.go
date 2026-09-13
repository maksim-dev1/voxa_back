// Command voxa-api is the app-tier HTTP service (deployed on Pi/Dokploy).
// It handles auth, accepts recording uploads, and exposes job status —
// no ML work happens here, that's the worker on nexus.
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/maksim/voxa-api/internal/config"
	"github.com/maksim/voxa-api/internal/db"
	"github.com/maksim/voxa-api/internal/httpapi"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	database, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer database.Close()

	server := httpapi.New(cfg, database)

	log.Printf("voxa-api listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, server.Routes()); err != nil {
		log.Fatal(err)
	}
}
