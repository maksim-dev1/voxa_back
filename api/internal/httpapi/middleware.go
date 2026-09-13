package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/maksim/voxa-api/internal/auth"
)

type ctxKey string

const userIDKey ctxKey = "userID"

// requireAuth verifies the Bearer token and puts the user id in the
// request context. Every /jobs/* handler relies on this to scope queries
// to the caller — never trust a user_id passed in the request body/query.
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		token, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || token == "" {
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}

		userID, err := auth.VerifyToken(s.cfg.JWTSecret, token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next(w, r.WithContext(ctx))
	}
}

func userIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(userIDKey).(string)
	return id
}
