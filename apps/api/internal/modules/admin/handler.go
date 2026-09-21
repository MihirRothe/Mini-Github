package admin

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

// GetStats handles GET /api/v1/admin/stats
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())
	stats, err := h.svc.GetStats(r.Context(), currentUser)
	if err != nil {
		if errors.Is(err, ErrUnauthorizedAdmin) {
			apiErrors.RespondWithError(w, apiErrors.Forbidden("Site administrator privileges required"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to load admin statistics: "+err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"stats": stats,
	})
}

// ListAuditLogs handles GET /api/v1/admin/audit-logs
func (h *Handler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	filter := AuditLogFilter{
		ActorUsername: r.URL.Query().Get("actor"),
		Action:        r.URL.Query().Get("action"),
		TargetType:    r.URL.Query().Get("target_type"),
		Limit:         limit,
		Offset:        offset,
	}

	logs, total, err := h.svc.ListAuditLogs(r.Context(), currentUser, filter)
	if err != nil {
		if errors.Is(err, ErrUnauthorizedAdmin) {
			apiErrors.RespondWithError(w, apiErrors.Forbidden("Site administrator privileges required"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to list audit logs: "+err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"audit_logs": logs,
		"total":      total,
		"limit":      filter.Limit,
		"offset":     filter.Offset,
	})
}

// CreateAuditLog handles POST /api/v1/admin/audit-logs
func (h *Handler) CreateAuditLog(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser == nil || !currentUser.IsAdmin {
		apiErrors.RespondWithError(w, apiErrors.Forbidden("Site administrator privileges required"))
		return
	}

	var req struct {
		Action     string `json:"action"`
		TargetType string `json:"target_type"`
		TargetID   string `json:"target_id"`
		Metadata   any    `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON request body"))
		return
	}

	ip := r.RemoteAddr
	ua := r.UserAgent()
	actorID := currentUser.ID

	if err := h.svc.LogAction(r.Context(), &actorID, currentUser.Username, req.Action, req.TargetType, req.TargetID, ip, ua, req.Metadata); err != nil {
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to record audit log: "+err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "Audit event recorded successfully",
	})
}

// ListUsers handles GET /api/v1/admin/users
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	query := r.URL.Query().Get("q")

	users, total, err := h.svc.ListUsers(r.Context(), currentUser, query, limit, offset)
	if err != nil {
		if errors.Is(err, ErrUnauthorizedAdmin) {
			apiErrors.RespondWithError(w, apiErrors.Forbidden("Site administrator privileges required"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to list users: "+err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"users":  users,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// UpdateUserStatus handles PATCH /api/v1/admin/users/{id}
func (h *Handler) UpdateUserStatus(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())
	targetUserID := chi.URLParam(r, "id")

	var req UpdateUserAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON request body"))
		return
	}

	ip := r.RemoteAddr
	ua := r.UserAgent()

	updatedUser, err := h.svc.UpdateUserStatus(r.Context(), currentUser, targetUserID, req, ip, ua)
	if err != nil {
		if errors.Is(err, ErrUnauthorizedAdmin) {
			apiErrors.RespondWithError(w, apiErrors.Forbidden("Site administrator privileges required"))
			return
		}
		if errors.Is(err, ErrCannotDemoteSelf) {
			apiErrors.RespondWithError(w, apiErrors.BadRequest("You cannot revoke your own administrator privileges"))
			return
		}
		if errors.Is(err, ErrUserNotFound) {
			apiErrors.RespondWithError(w, apiErrors.NotFound("User not found"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to update user status: "+err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"user": updatedUser,
	})
}

// ListOrganizations handles GET /api/v1/admin/orgs
func (h *Handler) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	orgs, total, err := h.svc.ListOrganizations(r.Context(), currentUser, limit, offset)
	if err != nil {
		if errors.Is(err, ErrUnauthorizedAdmin) {
			apiErrors.RespondWithError(w, apiErrors.Forbidden("Site administrator privileges required"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to list organizations: "+err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"organizations": orgs,
		"total":         total,
	})
}

// ListRepositories handles GET /api/v1/admin/repos
func (h *Handler) ListRepositories(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	repos, total, err := h.svc.ListRepositories(r.Context(), currentUser, limit, offset)
	if err != nil {
		if errors.Is(err, ErrUnauthorizedAdmin) {
			apiErrors.RespondWithError(w, apiErrors.Forbidden("Site administrator privileges required"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to list repositories: "+err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"repositories": repos,
		"total":        total,
	})
}
