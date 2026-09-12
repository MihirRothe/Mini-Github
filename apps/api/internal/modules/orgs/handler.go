package orgs

import (
	"encoding/json"
	"errors"
	"net/http"

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

// ----------------------------------------------------------------------------
// Organization Endpoints
// ----------------------------------------------------------------------------

func (h *Handler) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	var req CreateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON request body"))
		return
	}

	org, err := h.svc.CreateOrganization(r.Context(), currentUser.ID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidOrgSlug) || errors.Is(err, ErrReservedOrgSlug):
			apiErrors.RespondWithError(w, apiErrors.BadRequest(err.Error()))
		case errors.Is(err, ErrOrgSlugTaken):
			apiErrors.RespondWithError(w, apiErrors.Conflict("Organization slug is already in use"))
		default:
			apiErrors.RespondWithError(w, apiErrors.BadRequest(err.Error()))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message":      "Organization created successfully",
		"organization": org,
	})
}

func (h *Handler) ListUserOrganizations(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	orgs, err := h.svc.ListUserOrganizations(r.Context(), currentUser.ID)
	if err != nil {
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to list organizations"))
		return
	}
	if orgs == nil {
		orgs = []*Organization{}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"organizations": orgs,
	})
}

func (h *Handler) GetOrganization(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "org")
	if slug == "" {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Organization slug is required"))
		return
	}

	var currentUserID string
	if currentUser := auth.GetUserFromContext(r.Context()); currentUser != nil {
		currentUserID = currentUser.ID
	}

	org, err := h.svc.GetOrganization(r.Context(), currentUserID, slug)
	if err != nil {
		if errors.Is(err, ErrOrgNotFound) {
			apiErrors.RespondWithError(w, apiErrors.NotFound("Organization not found"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to retrieve organization"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"organization": org,
	})
}

func (h *Handler) UpdateOrganization(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	slug := chi.URLParam(r, "org")
	if slug == "" {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Organization slug is required"))
		return
	}

	var req UpdateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON request body"))
		return
	}

	org, err := h.svc.UpdateOrganization(r.Context(), currentUser.ID, currentUser.IsAdmin, slug, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrOrgNotFound):
			apiErrors.RespondWithError(w, apiErrors.NotFound("Organization not found"))
		case errors.Is(err, ErrInsufficientPrivilege):
			apiErrors.RespondWithError(w, apiErrors.Forbidden(err.Error()))
		default:
			apiErrors.RespondWithError(w, apiErrors.BadRequest(err.Error()))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message":      "Organization updated successfully",
		"organization": org,
	})
}

func (h *Handler) DeleteOrganization(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	slug := chi.URLParam(r, "org")
	if slug == "" {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Organization slug is required"))
		return
	}

	err := h.svc.DeleteOrganization(r.Context(), currentUser.ID, currentUser.IsAdmin, slug)
	if err != nil {
		switch {
		case errors.Is(err, ErrOrgNotFound):
			apiErrors.RespondWithError(w, apiErrors.NotFound("Organization not found"))
		case errors.Is(err, ErrInsufficientPrivilege):
			apiErrors.RespondWithError(w, apiErrors.Forbidden(err.Error()))
		default:
			apiErrors.RespondWithError(w, apiErrors.Internal("Failed to delete organization"))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "Organization deleted successfully",
	})
}

// ----------------------------------------------------------------------------
// Member Endpoints
// ----------------------------------------------------------------------------

func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "org")
	if slug == "" {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Organization slug is required"))
		return
	}

	var currentUserID string
	if currentUser := auth.GetUserFromContext(r.Context()); currentUser != nil {
		currentUserID = currentUser.ID
	}

	members, err := h.svc.ListMembers(r.Context(), currentUserID, slug)
	if err != nil {
		if errors.Is(err, ErrOrgNotFound) {
			apiErrors.RespondWithError(w, apiErrors.NotFound("Organization not found"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to list members"))
		return
	}
	if members == nil {
		members = []*OrgMember{}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"members": members,
	})
}

func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	slug := chi.URLParam(r, "org")
	if slug == "" {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Organization slug is required"))
		return
	}

	var req AddOrgMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON request body"))
		return
	}

	member, err := h.svc.AddMember(r.Context(), currentUser.ID, currentUser.IsAdmin, slug, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrOrgNotFound):
			apiErrors.RespondWithError(w, apiErrors.NotFound("Organization not found"))
		case errors.Is(err, ErrInsufficientPrivilege):
			apiErrors.RespondWithError(w, apiErrors.Forbidden(err.Error()))
		case errors.Is(err, ErrMemberAlreadyExists):
			apiErrors.RespondWithError(w, apiErrors.Conflict("User is already a member of this organization"))
		default:
			apiErrors.RespondWithError(w, apiErrors.BadRequest(err.Error()))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "Member added successfully",
		"member":  member,
	})
}

func (h *Handler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	slug := chi.URLParam(r, "org")
	username := chi.URLParam(r, "username")
	if slug == "" || username == "" {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Organization slug and username are required"))
		return
	}

	var req UpdateOrgMemberRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON request body"))
		return
	}

	err := h.svc.UpdateMemberRole(r.Context(), currentUser.ID, currentUser.IsAdmin, slug, username, req.Role)
	if err != nil {
		switch {
		case errors.Is(err, ErrOrgNotFound) || errors.Is(err, ErrMemberNotFound):
			apiErrors.RespondWithError(w, apiErrors.NotFound(err.Error()))
		case errors.Is(err, ErrInsufficientPrivilege):
			apiErrors.RespondWithError(w, apiErrors.Forbidden(err.Error()))
		case errors.Is(err, ErrCannotRemoveLastOwner):
			apiErrors.RespondWithError(w, apiErrors.Conflict("Cannot demote the last owner of the organization"))
		default:
			apiErrors.RespondWithError(w, apiErrors.BadRequest(err.Error()))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "Member role updated successfully",
	})
}

func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	slug := chi.URLParam(r, "org")
	username := chi.URLParam(r, "username")
	if slug == "" || username == "" {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Organization slug and username are required"))
		return
	}

	err := h.svc.RemoveMember(r.Context(), currentUser.ID, currentUser.IsAdmin, slug, username)
	if err != nil {
		switch {
		case errors.Is(err, ErrOrgNotFound) || errors.Is(err, ErrMemberNotFound):
			apiErrors.RespondWithError(w, apiErrors.NotFound(err.Error()))
		case errors.Is(err, ErrInsufficientPrivilege):
			apiErrors.RespondWithError(w, apiErrors.Forbidden(err.Error()))
		case errors.Is(err, ErrCannotRemoveLastOwner):
			apiErrors.RespondWithError(w, apiErrors.Conflict("Cannot remove the last owner of the organization"))
		default:
			apiErrors.RespondWithError(w, apiErrors.Internal("Failed to remove member"))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "Member removed successfully",
	})
}
