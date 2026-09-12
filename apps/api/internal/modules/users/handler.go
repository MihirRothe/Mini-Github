package users

import (
	"encoding/json"
	"errors"
	"net/http"

	apiErrors "forgehub/apps/api/internal/errors"
	authModule "forgehub/apps/api/internal/modules/auth"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc *authModule.Service
}

func NewHandler(svc *authModule.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	if username == "" {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Username parameter is required"))
		return
	}

	profile, err := h.svc.GetUserProfile(r.Context(), username)
	if err != nil {
		if errors.Is(err, authModule.ErrUserNotFound) {
			apiErrors.RespondWithError(w, apiErrors.NotFound("User not found"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to retrieve user profile"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"user": profile,
	})
}

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	currentUser := authModule.GetUserFromContext(r.Context())
	if currentUser == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	var req authModule.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON body"))
		return
	}

	updated, err := h.svc.UpdateProfile(r.Context(), currentUser.ID, req)
	if err != nil {
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to update profile"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"user":    updated,
		"message": "Profile updated successfully",
	})
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	currentUser := authModule.GetUserFromContext(r.Context())
	if currentUser == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	var req authModule.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON body"))
		return
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Both current_password and new_password are required"))
		return
	}

	if err := h.svc.ChangePassword(r.Context(), currentUser.ID, req.CurrentPassword, req.NewPassword); err != nil {
		if err.Error() == "current password does not match" {
			apiErrors.RespondWithError(w, apiErrors.New(http.StatusUnauthorized, apiErrors.ErrUnauthorized, "Current password does not match"))
			return
		}
		if errors.Is(err, authModule.ErrWeakPassword) {
			apiErrors.RespondWithError(w, apiErrors.New(http.StatusBadRequest, apiErrors.ErrValidation, err.Error()))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to update password"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "Password changed successfully. Please log in again with your new credentials.",
	})
}
