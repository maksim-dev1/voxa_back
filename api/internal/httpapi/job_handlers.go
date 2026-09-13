package httpapi

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/maksim/voxa-api/internal/db"
	"github.com/maksim/voxa-api/internal/relay"
)

type jobResponse struct {
	ID        string  `json:"id"`
	Status    string  `json:"status"`
	Error     *string `json:"error,omitempty"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

func toJobResponse(j db.Job) jobResponse {
	return jobResponse{
		ID:        j.ID,
		Status:    j.Status,
		Error:     j.Error,
		CreatedAt: j.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: j.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// handleCreateJob streams the uploaded recording straight through to
// storage-relay on nexus. It reads the multipart body part-by-part
// (never r.ParseMultipartForm, which would buffer the whole file to a
// temp file on Pi first) — Pi stays stateless, the recording touches disk
// only once it lands on nexus.
func (s *Server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())

	reader, err := r.MultipartReader()
	if err != nil {
		writeError(w, http.StatusBadRequest, "expected multipart/form-data")
		return
	}

	var audioPart *multipart.Part
	var filename, contentType string
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			writeError(w, http.StatusBadRequest, "malformed multipart body")
			return
		}
		if part.FormName() == "audio" {
			audioPart = part
			filename = part.FileName()
			contentType = part.Header.Get("Content-Type")
			break
		}
		part.Close()
	}

	if audioPart == nil {
		writeError(w, http.StatusBadRequest, "missing 'audio' file field")
		return
	}
	defer audioPart.Close()

	jobID, err := s.relay.UploadRecording(r.Context(), relay.Upload{
		UserID:      userID,
		Filename:    filename,
		ContentType: contentType,
		Body:        audioPart,
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to reach storage relay")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": jobID})
}

func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	jobs, err := s.db.ListJobs(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	out := make([]jobResponse, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, toJobResponse(j))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	id := r.PathValue("id")

	job, err := s.db.JobByID(r.Context(), id, userID)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "job not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, toJobResponse(job))
}

func (s *Server) handleGetResult(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	id := r.PathValue("id")

	// Confirm ownership first — never fetch a transcript by job id alone.
	job, err := s.db.JobByID(r.Context(), id, userID)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "job not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if job.Status != "done" {
		writeError(w, http.StatusConflict, "job is not done yet: "+job.Status)
		return
	}

	transcript, err := s.db.TranscriptByJobID(r.Context(), id)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "transcript not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// segments is already-valid jsonb from Postgres; write it straight
	// through wrapped in an object, no need to round-trip via a Go struct.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"segments":`))
	_, _ = w.Write(transcript.Segments)
	_, _ = w.Write([]byte(`}`))
}
