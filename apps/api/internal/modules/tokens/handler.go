package tokens

import (
	"encoding/json"
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

func (h *Handler) CreateToken(w http.ResponseWriter, r *http.Request) {
	currentUser := authModule.GetUserFromContext(r.Context())
	if currentUser == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	var req authModule.CreateTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON body"))
		return
	}

	tokenResp, err := h.svc.CreateAPIToken(r.Context(), currentUser.ID, req)
	if err != nil {
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to generate token"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"token":   tokenResp,
		"message": "Token generated successfully. Make sure to copy it now as it cannot be shown again.",
	})
}

func (h *Handler) ListTokens(w http.ResponseWriter, r *http.Request) {
	currentUser := authModule.GetUserFromContext(r.Context())
	if currentUser == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	tokens, err := h.svc.ListAPITokens(r.Context(), currentUser.ID)
	if err != nil {
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to fetch tokens"))
		return
	}

	if tokens == nil {
		tokens = make([]*authModule.APIToken, 0)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"tokens": tokens,
	})
}

func (h *Handler) DeleteToken(w http.ResponseWriter, r *http.Request) {
	currentUser := authModule.GetUserFromContext(r.Context())
	if currentUser == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	tokenID := chi.URLParam(r, "id")
	if tokenID == "" {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Token ID is required"))
		return
	}

	if err := h.svc.DeleteAPIToken(r.Context(), currentUser.ID, tokenID); err != nil {
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to revoke token"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "Token revoked successfully",
	})
}
