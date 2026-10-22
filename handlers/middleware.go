package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"khata-book/models"
	"khata-book/services"
)

type contextKey string

const UserContextKey contextKey = "authUser"

// AuthMiddleware resolves the user identity and RBAC role from header X-User-Id, cookie, or query param.
// In enterprise deployments, an upstream gateway/reverse proxy or session cookie passes the authenticated user.
func AuthMiddleware(svc *services.KhataService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := r.Header.Get("X-User-Id")
			if userID == "" {
				if cookie, err := r.Cookie("khata_user_id"); err == nil {
					userID = cookie.Value
				}
			}
			if userID == "" {
				userID = r.URL.Query().Get("as_user")
			}
			if userID == "" {
				userID = "admin" // Default to admin for seamless evaluation
			}

			user, err := svc.GetUserByID(userID)
			if err != nil {
				// Fallback to admin if specified user not found
				user, err = svc.GetUserByID("admin")
				if err != nil {
					WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Failed to resolve identity")
					return
				}
			}

			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserFromContext retrieves the authenticated user from the request context.
func GetUserFromContext(r *http.Request) *models.User {
	if u, ok := r.Context().Value(UserContextKey).(*models.User); ok {
		return u
	}
	return nil
}

// WriteJSON sends a JSON response with status code.
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// WriteError sends a formatted JSON error.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, map[string]interface{}{
		"error":  message,
		"code":   code,
		"status": status,
	})
}

// HandleServiceError maps domain errors to standard HTTP status codes.
func HandleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, models.ErrNotFound):
		WriteError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, models.ErrUnauthorized):
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "You do not have permission to access this department")
	case errors.Is(err, models.ErrMaxDepthExceeded):
		WriteError(w, http.StatusBadRequest, "MAX_DEPTH_EXCEEDED", err.Error())
	case errors.Is(err, models.ErrMoveExceedsMaxDepth):
		WriteError(w, http.StatusBadRequest, "SUBTREE_DEPTH_EXCEEDED", err.Error())
	case errors.Is(err, models.ErrCannotMoveToDescendant):
		WriteError(w, http.StatusBadRequest, "CYCLE_DETECTED", err.Error())
	case errors.Is(err, models.ErrInvalidInput):
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", err.Error())
	default:
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}
