package webhooks

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	apiErrors "forgehub/apps/api/internal/errors"
	"forgehub/apps/api/internal/modules/auth"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ListWebhooks(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")

	var currentUserID string
	var isSiteAdmin bool
	if u := auth.GetUserFromContext(r.Context()); u != nil {
		currentUserID = u.ID
		isSiteAdmin = u.IsAdmin
	}

	hooks, err := h.svc.ListWebhooks(r.Context(), currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		if errors.Is(err, ErrAccessDenied) {
			apiErrors.RespondWithError(w, apiErrors.Forbidden(err.Error()))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to list webhooks"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"webhooks": hooks,
	})
}

func (h *Handler) CreateWebhook(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")

	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	var req CreateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON body"))
		return
	}

	hook, err := h.svc.CreateWebhook(r.Context(), u.ID, u.IsAdmin, owner, repoSlug, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidWebhookURL):
			apiErrors.RespondWithError(w, apiErrors.BadRequest(err.Error()))
		case errors.Is(err, ErrAccessDenied):
			apiErrors.RespondWithError(w, apiErrors.Forbidden(err.Error()))
		default:
			apiErrors.RespondWithError(w, apiErrors.Internal(err.Error()))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "Webhook created successfully",
		"webhook": hook,
	})
}

func (h *Handler) GetWebhook(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	hookID := chi.URLParam(r, "hook_id")

	var currentUserID string
	var isSiteAdmin bool
	if u := auth.GetUserFromContext(r.Context()); u != nil {
		currentUserID = u.ID
		isSiteAdmin = u.IsAdmin
	}

	hook, err := h.svc.GetWebhook(r.Context(), currentUserID, isSiteAdmin, owner, repoSlug, hookID)
	if err != nil {
		if errors.Is(err, ErrWebhookNotFound) {
			apiErrors.RespondWithError(w, apiErrors.NotFound("Webhook not found"))
			return
		}
		if errors.Is(err, ErrAccessDenied) {
			apiErrors.RespondWithError(w, apiErrors.Forbidden("Access denied"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to get webhook"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"webhook": hook,
	})
}

func (h *Handler) UpdateWebhook(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	hookID := chi.URLParam(r, "hook_id")

	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	var req UpdateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON body"))
		return
	}

	hook, err := h.svc.UpdateWebhook(r.Context(), u.ID, u.IsAdmin, owner, repoSlug, hookID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrWebhookNotFound):
			apiErrors.RespondWithError(w, apiErrors.NotFound("Webhook not found"))
		case errors.Is(err, ErrInvalidWebhookURL):
			apiErrors.RespondWithError(w, apiErrors.BadRequest(err.Error()))
		case errors.Is(err, ErrAccessDenied):
			apiErrors.RespondWithError(w, apiErrors.Forbidden("Access denied"))
		default:
			apiErrors.RespondWithError(w, apiErrors.Internal(err.Error()))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "Webhook updated successfully",
		"webhook": hook,
	})
}

func (h *Handler) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	hookID := chi.URLParam(r, "hook_id")

	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	if err := h.svc.DeleteWebhook(r.Context(), u.ID, u.IsAdmin, owner, repoSlug, hookID); err != nil {
		if errors.Is(err, ErrWebhookNotFound) {
			apiErrors.RespondWithError(w, apiErrors.NotFound("Webhook not found"))
			return
		}
		if errors.Is(err, ErrAccessDenied) {
			apiErrors.RespondWithError(w, apiErrors.Forbidden("Access denied"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to delete webhook"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "Webhook deleted successfully",
	})
}

func (h *Handler) TestPing(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	hookID := chi.URLParam(r, "hook_id")

	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	delivery, err := h.svc.TestPing(r.Context(), u.ID, u.IsAdmin, owner, repoSlug, hookID)
	if err != nil {
		apiErrors.RespondWithError(w, apiErrors.Internal(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message":  "Ping test delivered",
		"delivery": delivery,
	})
}

func (h *Handler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	hookID := chi.URLParam(r, "hook_id")

	var currentUserID string
	var isSiteAdmin bool
	if u := auth.GetUserFromContext(r.Context()); u != nil {
		currentUserID = u.ID
		isSiteAdmin = u.IsAdmin
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	deliveries, err := h.svc.ListDeliveries(r.Context(), currentUserID, isSiteAdmin, owner, repoSlug, hookID, limit)
	if err != nil {
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to list deliveries"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"deliveries": deliveries,
	})
}

func (h *Handler) GetDelivery(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	hookID := chi.URLParam(r, "hook_id")
	deliveryID := chi.URLParam(r, "delivery_id")

	var currentUserID string
	var isSiteAdmin bool
	if u := auth.GetUserFromContext(r.Context()); u != nil {
		currentUserID = u.ID
		isSiteAdmin = u.IsAdmin
	}

	del, err := h.svc.GetDelivery(r.Context(), currentUserID, isSiteAdmin, owner, repoSlug, hookID, deliveryID)
	if err != nil {
		apiErrors.RespondWithError(w, apiErrors.NotFound("Delivery not found"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"delivery": del,
	})
}

func (h *Handler) Redeliver(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	hookID := chi.URLParam(r, "hook_id")
	deliveryID := chi.URLParam(r, "delivery_id")

	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	del, err := h.svc.Redeliver(r.Context(), u.ID, u.IsAdmin, owner, repoSlug, hookID, deliveryID)
	if err != nil {
		apiErrors.RespondWithError(w, apiErrors.Internal(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message":  "Redelivered successfully",
		"delivery": del,
	})
}
