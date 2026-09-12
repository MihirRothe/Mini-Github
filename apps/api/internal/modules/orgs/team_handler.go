package orgs

import (
	"encoding/json"
	"errors"
	"net/http"

	apiErrors "forgehub/apps/api/internal/errors"
	"forgehub/apps/api/internal/modules/auth"

	"github.com/go-chi/chi/v5"
)

// ----------------------------------------------------------------------------
// Team Endpoints
// ----------------------------------------------------------------------------

func (h *Handler) ListTeams(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "org")
	if slug == "" {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Organization slug is required"))
		return
	}

	var currentUserID string
	if currentUser := auth.GetUserFromContext(r.Context()); currentUser != nil {
		currentUserID = currentUser.ID
	}

	teams, err := h.svc.ListTeams(r.Context(), currentUserID, slug)
	if err != nil {
		if errors.Is(err, ErrOrgNotFound) {
			apiErrors.RespondWithError(w, apiErrors.NotFound("Organization not found"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to list teams"))
		return
	}
	if teams == nil {
		teams = []*Team{}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"teams": teams,
	})
}

func (h *Handler) CreateTeam(w http.ResponseWriter, r *http.Request) {
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

	var req CreateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON request body"))
		return
	}

	team, err := h.svc.CreateTeam(r.Context(), currentUser.ID, currentUser.IsAdmin, slug, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrOrgNotFound):
			apiErrors.RespondWithError(w, apiErrors.NotFound("Organization not found"))
		case errors.Is(err, ErrInsufficientPrivilege):
			apiErrors.RespondWithError(w, apiErrors.Forbidden(err.Error()))
		case errors.Is(err, ErrInvalidTeamSlug):
			apiErrors.RespondWithError(w, apiErrors.BadRequest(err.Error()))
		case errors.Is(err, ErrTeamSlugTaken):
			apiErrors.RespondWithError(w, apiErrors.Conflict("Team slug already exists in this organization"))
		default:
			apiErrors.RespondWithError(w, apiErrors.BadRequest(err.Error()))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "Team created successfully",
		"team":    team,
	})
}

func (h *Handler) GetTeam(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "org")
	teamSlug := chi.URLParam(r, "team")
	if slug == "" || teamSlug == "" {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Organization and team slugs are required"))
		return
	}

	var currentUserID string
	if currentUser := auth.GetUserFromContext(r.Context()); currentUser != nil {
		currentUserID = currentUser.ID
	}

	team, err := h.svc.GetTeam(r.Context(), currentUserID, slug, teamSlug)
	if err != nil {
		switch {
		case errors.Is(err, ErrOrgNotFound) || errors.Is(err, ErrTeamNotFound):
			apiErrors.RespondWithError(w, apiErrors.NotFound("Team not found"))
		default:
			apiErrors.RespondWithError(w, apiErrors.Internal("Failed to retrieve team"))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"team": team,
	})
}

func (h *Handler) UpdateTeam(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	slug := chi.URLParam(r, "org")
	teamSlug := chi.URLParam(r, "team")
	if slug == "" || teamSlug == "" {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Organization and team slugs are required"))
		return
	}

	var req UpdateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON request body"))
		return
	}

	team, err := h.svc.UpdateTeam(r.Context(), currentUser.ID, currentUser.IsAdmin, slug, teamSlug, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrOrgNotFound) || errors.Is(err, ErrTeamNotFound):
			apiErrors.RespondWithError(w, apiErrors.NotFound("Team not found"))
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
		"message": "Team updated successfully",
		"team":    team,
	})
}

func (h *Handler) DeleteTeam(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	slug := chi.URLParam(r, "org")
	teamSlug := chi.URLParam(r, "team")
	if slug == "" || teamSlug == "" {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Organization and team slugs are required"))
		return
	}

	err := h.svc.DeleteTeam(r.Context(), currentUser.ID, currentUser.IsAdmin, slug, teamSlug)
	if err != nil {
		switch {
		case errors.Is(err, ErrOrgNotFound) || errors.Is(err, ErrTeamNotFound):
			apiErrors.RespondWithError(w, apiErrors.NotFound("Team not found"))
		case errors.Is(err, ErrInsufficientPrivilege):
			apiErrors.RespondWithError(w, apiErrors.Forbidden(err.Error()))
		default:
			apiErrors.RespondWithError(w, apiErrors.Internal("Failed to delete team"))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "Team deleted successfully",
	})
}

// ----------------------------------------------------------------------------
// Team Members
// ----------------------------------------------------------------------------

func (h *Handler) ListTeamMembers(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "org")
	teamSlug := chi.URLParam(r, "team")
	if slug == "" || teamSlug == "" {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Organization and team slugs are required"))
		return
	}

	var currentUserID string
	if currentUser := auth.GetUserFromContext(r.Context()); currentUser != nil {
		currentUserID = currentUser.ID
	}

	members, err := h.svc.ListTeamMembers(r.Context(), currentUserID, slug, teamSlug)
	if err != nil {
		switch {
		case errors.Is(err, ErrOrgNotFound) || errors.Is(err, ErrTeamNotFound):
			apiErrors.RespondWithError(w, apiErrors.NotFound("Team not found"))
		default:
			apiErrors.RespondWithError(w, apiErrors.Internal("Failed to list team members"))
		}
		return
	}
	if members == nil {
		members = []*TeamMember{}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"members": members,
	})
}

func (h *Handler) AddTeamMember(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	slug := chi.URLParam(r, "org")
	teamSlug := chi.URLParam(r, "team")
	if slug == "" || teamSlug == "" {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Organization and team slugs are required"))
		return
	}

	var req AddTeamMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON request body"))
		return
	}

	member, err := h.svc.AddTeamMember(r.Context(), currentUser.ID, currentUser.IsAdmin, slug, teamSlug, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrOrgNotFound) || errors.Is(err, ErrTeamNotFound):
			apiErrors.RespondWithError(w, apiErrors.NotFound("Team not found"))
		case errors.Is(err, ErrInsufficientPrivilege):
			apiErrors.RespondWithError(w, apiErrors.Forbidden(err.Error()))
		case errors.Is(err, ErrUserNotOrgMember):
			apiErrors.RespondWithError(w, apiErrors.BadRequest(err.Error()))
		case errors.Is(err, ErrMemberAlreadyExists):
			apiErrors.RespondWithError(w, apiErrors.Conflict("User is already a member of this team"))
		default:
			apiErrors.RespondWithError(w, apiErrors.BadRequest(err.Error()))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "Team member added successfully",
		"member":  member,
	})
}

func (h *Handler) UpdateTeamMemberRole(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	slug := chi.URLParam(r, "org")
	teamSlug := chi.URLParam(r, "team")
	username := chi.URLParam(r, "username")
	if slug == "" || teamSlug == "" || username == "" {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Organization slug, team slug, and username are required"))
		return
	}

	var req UpdateTeamMemberRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON request body"))
		return
	}

	err := h.svc.UpdateTeamMemberRole(r.Context(), currentUser.ID, currentUser.IsAdmin, slug, teamSlug, username, req.Role)
	if err != nil {
		switch {
		case errors.Is(err, ErrOrgNotFound) || errors.Is(err, ErrTeamNotFound) || errors.Is(err, ErrMemberNotFound):
			apiErrors.RespondWithError(w, apiErrors.NotFound("Member or team not found"))
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
		"message": "Team member role updated successfully",
	})
}

func (h *Handler) RemoveTeamMember(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	slug := chi.URLParam(r, "org")
	teamSlug := chi.URLParam(r, "team")
	username := chi.URLParam(r, "username")
	if slug == "" || teamSlug == "" || username == "" {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Organization slug, team slug, and username are required"))
		return
	}

	err := h.svc.RemoveTeamMember(r.Context(), currentUser.ID, currentUser.IsAdmin, slug, teamSlug, username)
	if err != nil {
		switch {
		case errors.Is(err, ErrOrgNotFound) || errors.Is(err, ErrTeamNotFound) || errors.Is(err, ErrMemberNotFound):
			apiErrors.RespondWithError(w, apiErrors.NotFound("Member or team not found"))
		case errors.Is(err, ErrInsufficientPrivilege):
			apiErrors.RespondWithError(w, apiErrors.Forbidden(err.Error()))
		default:
			apiErrors.RespondWithError(w, apiErrors.Internal("Failed to remove team member"))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "Team member removed successfully",
	})
}
