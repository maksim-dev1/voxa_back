// Command voxa-storage-relay runs on nexus. It receives audio files
// streamed from voxa-api (on Pi), writes them to local disk where the
// GPU worker can read them, and enqueues a transcription job.
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/maksim/voxa-storage-relay/internal/config"
	"github.com/maksim/voxa-storage-relay/internal/ingest"
	"github.com/maksim/voxa-storage-relay/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	st, err := store.Connect(ctx, cfg.DatabaseURL, cfg.RedisURL)
	if err != nil {
		log.Fatalf("store connect: %v", err)
	}
	defer st.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("/ingest", ingest.New(st, cfg.RecordingsDir))

	log.Printf("voxa-storage-relay listening on %s (recordings dir: %s)", cfg.Addr, cfg.RecordingsDir)
	if err := http.ListenAndServe(cfg.Addr, mux); err != nil {
		log.Fatal(err)
	}
}
