// Package ingest handles the /ingest HTTP endpoint: streaming the audio
// file straight to disk (never buffered fully in memory — recordings can
// be an hour+ of audio on an 8GB-RAM box) and registering the job.
package ingest

import (
	"encoding/json"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/maksim/voxa-storage-relay/internal/store"
)

type Handler struct {
	store         *store.Store
	recordingsDir string
}

func New(s *store.Store, recordingsDir string) *Handler {
	return &Handler{store: s, recordingsDir: recordingsDir}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/form-data" {
		writeError(w, http.StatusBadRequest, "expected multipart/form-data")
		return
	}

	reader := multipart.NewReader(r.Body, params["boundary"])

	var userID string
	var audioSaved bool
	jobID := uuid.NewString()
	audioPath := filepath.Join(h.recordingsDir, jobID+".m4a")

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			writeError(w, http.StatusBadRequest, "malformed multipart body")
			return
		}

		switch part.FormName() {
		case "user_id":
			data, _ := io.ReadAll(io.LimitReader(part, 256))
			userID = string(data)
		case "audio":
			if err := h.saveAudio(part, audioPath); err != nil {
				log.Printf("ingest: failed to save audio for job %s: %v", jobID, err)
				writeError(w, http.StatusInternalServerError, "failed to save audio")
				return
			}
			audioSaved = true
		}
		part.Close()
	}

	if userID == "" || !audioSaved {
		_ = os.Remove(audioPath)
		writeError(w, http.StatusBadRequest, "missing user_id or audio part")
		return
	}

	if err := h.store.CreateJob(r.Context(), jobID, userID, audioPath); err != nil {
		log.Printf("ingest: failed to create job %s: %v", jobID, err)
		_ = os.Remove(audioPath)
		writeError(w, http.StatusInternalServerError, "failed to register job")
		return
	}

	log.Printf("ingest: job %s created for user %s -> %s", jobID, userID, audioPath)
	writeJSON(w, http.StatusCreated, map[string]string{"job_id": jobID})
}

func (h *Handler) saveAudio(part *multipart.Part, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, part)
	return err
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
