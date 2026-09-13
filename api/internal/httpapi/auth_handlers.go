package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/maksim/voxa-api/internal/auth"
	"github.com/maksim/voxa-api/internal/db"
)

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string `json:"token"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" || len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "email required, password must be at least 8 characters")
		return
	}

	if _, err := s.db.UserByEmail(r.Context(), req.Email); err == nil {
		writeError(w, http.StatusConflict, "email already registered")
		return
	} else if !errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	userID := uuid.NewString()
	if err := s.db.CreateUser(r.Context(), userID, req.Email, hash); err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	token, err := auth.IssueToken(s.cfg.JWTSecret, s.cfg.JWTTTL, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, authResponse{Token: token})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	user, err := s.db.UserByEmail(r.Context(), req.Email)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token, err := auth.IssueToken(s.cfg.JWTSecret, s.cfg.JWTTTL, user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, authResponse{Token: token})
}
