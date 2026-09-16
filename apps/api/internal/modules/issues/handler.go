package issues

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

func (h *Handler) ListIssues(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")

	var currentUserID string
	var isSiteAdmin bool
	if u := auth.GetUserFromContext(r.Context()); u != nil {
		currentUserID = u.ID
		isSiteAdmin = u.IsAdmin
	}

	filter := IssueFilter{
		State:            r.URL.Query().Get("state"),
		LabelName:        r.URL.Query().Get("label"),
		MilestoneID:      r.URL.Query().Get("milestone"),
		AssigneeUsername: r.URL.Query().Get("assignee"),
		AuthorUsername:   r.URL.Query().Get("author"),
		Query:            r.URL.Query().Get("q"),
	}

	issuesList, openCount, closedCount, err := h.svc.ListIssues(r.Context(), currentUserID, isSiteAdmin, owner, repoSlug, filter)
	if err != nil {
		if errors.Is(err, ErrAccessDenied) {
			apiErrors.RespondWithError(w, apiErrors.Forbidden("You do not have access to this repository"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to list issues"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"issues":       issuesList,
		"open_count":   openCount,
		"closed_count": closedCount,
	})
}

func (h *Handler) CreateIssue(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")

	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	var req CreateIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON body"))
		return
	}

	issue, err := h.svc.CreateIssue(r.Context(), u.ID, u.IsAdmin, owner, repoSlug, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmptyTitle):
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
		"message": "Issue created successfully",
		"issue":   issue,
	})
}

func (h *Handler) GetIssue(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	numberStr := chi.URLParam(r, "number")

	number, err := strconv.Atoi(numberStr)
	if err != nil || number <= 0 {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid issue number"))
		return
	}

	var currentUserID string
	var isSiteAdmin bool
	if u := auth.GetUserFromContext(r.Context()); u != nil {
		currentUserID = u.ID
		isSiteAdmin = u.IsAdmin
	}

	detail, err := h.svc.GetIssue(r.Context(), currentUserID, isSiteAdmin, owner, repoSlug, number)
	if err != nil {
		if errors.Is(err, ErrIssueNotFound) {
			apiErrors.RespondWithError(w, apiErrors.NotFound("Issue not found"))
			return
		}
		if errors.Is(err, ErrAccessDenied) {
			apiErrors.RespondWithError(w, apiErrors.Forbidden("Access denied"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to get issue"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"issue": detail,
	})
}

func (h *Handler) UpdateIssue(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	numberStr := chi.URLParam(r, "number")

	number, err := strconv.Atoi(numberStr)
	if err != nil || number <= 0 {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid issue number"))
		return
	}

	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	var req UpdateIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON body"))
		return
	}

	issue, err := h.svc.UpdateIssue(r.Context(), u.ID, u.IsAdmin, owner, repoSlug, number, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrIssueNotFound):
			apiErrors.RespondWithError(w, apiErrors.NotFound("Issue not found"))
		case errors.Is(err, ErrAccessDenied):
			apiErrors.RespondWithError(w, apiErrors.Forbidden("You do not have permission to modify this issue"))
		case errors.Is(err, ErrEmptyTitle):
			apiErrors.RespondWithError(w, apiErrors.BadRequest(err.Error()))
		default:
			apiErrors.RespondWithError(w, apiErrors.Internal(err.Error()))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "Issue updated successfully",
		"issue":   issue,
	})
}

func (h *Handler) DeleteIssue(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	numberStr := chi.URLParam(r, "number")

	number, err := strconv.Atoi(numberStr)
	if err != nil || number <= 0 {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid issue number"))
		return
	}

	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	if err := h.svc.DeleteIssue(r.Context(), u.ID, u.IsAdmin, owner, repoSlug, number); err != nil {
		if errors.Is(err, ErrIssueNotFound) {
			apiErrors.RespondWithError(w, apiErrors.NotFound("Issue not found"))
			return
		}
		if errors.Is(err, ErrAccessDenied) {
			apiErrors.RespondWithError(w, apiErrors.Forbidden("You do not have permission to delete this issue"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "Issue deleted successfully",
	})
}

// ============================================================================
// Comments Handlers
// ============================================================================

func (h *Handler) CreateComment(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	numberStr := chi.URLParam(r, "number")

	number, err := strconv.Atoi(numberStr)
	if err != nil || number <= 0 {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid issue number"))
		return
	}

	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	var req CreateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON body"))
		return
	}

	comment, err := h.svc.CreateComment(r.Context(), u.ID, u.IsAdmin, owner, repoSlug, number, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmptyComment):
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
		"message": "Comment added successfully",
		"comment": comment,
	})
}

func (h *Handler) UpdateComment(w http.ResponseWriter, r *http.Request) {
	commentID := chi.URLParam(r, "comment_id")
	if commentID == "" {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Comment ID required"))
		return
	}

	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	var req UpdateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON body"))
		return
	}

	comment, err := h.svc.UpdateComment(r.Context(), u.ID, u.IsAdmin, commentID, req)
	if err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			apiErrors.RespondWithError(w, apiErrors.NotFound("Comment not found"))
			return
		}
		if errors.Is(err, ErrAccessDenied) {
			apiErrors.RespondWithError(w, apiErrors.Forbidden("Forbidden"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"comment": comment,
	})
}

func (h *Handler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	commentID := chi.URLParam(r, "comment_id")
	if commentID == "" {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Comment ID required"))
		return
	}

	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	if err := h.svc.DeleteComment(r.Context(), u.ID, u.IsAdmin, commentID); err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			apiErrors.RespondWithError(w, apiErrors.NotFound("Comment not found"))
			return
		}
		if errors.Is(err, ErrAccessDenied) {
			apiErrors.RespondWithError(w, apiErrors.Forbidden("Forbidden"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "Comment deleted successfully",
	})
}

// ============================================================================
// Labels Handlers
// ============================================================================

func (h *Handler) ListLabels(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")

	var currentUserID string
	var isSiteAdmin bool
	if u := auth.GetUserFromContext(r.Context()); u != nil {
		currentUserID = u.ID
		isSiteAdmin = u.IsAdmin
	}

	labels, err := h.svc.ListLabels(r.Context(), currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to list labels"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"labels": labels,
	})
}

func (h *Handler) CreateLabel(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")

	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	var req CreateLabelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON body"))
		return
	}

	label, err := h.svc.CreateLabel(r.Context(), u.ID, u.IsAdmin, owner, repoSlug, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmptyLabelName):
			apiErrors.RespondWithError(w, apiErrors.BadRequest(err.Error()))
		case errors.Is(err, ErrLabelNameTaken):
			apiErrors.RespondWithError(w, apiErrors.Conflict("A label with this name already exists"))
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
		"label": label,
	})
}

func (h *Handler) UpdateLabel(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	labelID := chi.URLParam(r, "label_id")

	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	var req UpdateLabelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON body"))
		return
	}

	label, err := h.svc.UpdateLabel(r.Context(), u.ID, u.IsAdmin, owner, repoSlug, labelID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrLabelNotFound):
			apiErrors.RespondWithError(w, apiErrors.NotFound("Label not found"))
		case errors.Is(err, ErrLabelNameTaken):
			apiErrors.RespondWithError(w, apiErrors.Conflict("A label with this name already exists"))
		case errors.Is(err, ErrAccessDenied):
			apiErrors.RespondWithError(w, apiErrors.Forbidden(err.Error()))
		default:
			apiErrors.RespondWithError(w, apiErrors.Internal(err.Error()))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"label": label,
	})
}

func (h *Handler) DeleteLabel(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	labelID := chi.URLParam(r, "label_id")

	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	if err := h.svc.DeleteLabel(r.Context(), u.ID, u.IsAdmin, owner, repoSlug, labelID); err != nil {
		if errors.Is(err, ErrLabelNotFound) {
			apiErrors.RespondWithError(w, apiErrors.NotFound("Label not found"))
			return
		}
		if errors.Is(err, ErrAccessDenied) {
			apiErrors.RespondWithError(w, apiErrors.Forbidden(err.Error()))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "Label deleted successfully",
	})
}

// ============================================================================
// Milestones Handlers
// ============================================================================

func (h *Handler) ListMilestones(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")

	var currentUserID string
	var isSiteAdmin bool
	if u := auth.GetUserFromContext(r.Context()); u != nil {
		currentUserID = u.ID
		isSiteAdmin = u.IsAdmin
	}

	milestones, err := h.svc.ListMilestones(r.Context(), currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to list milestones"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"milestones": milestones,
	})
}

func (h *Handler) CreateMilestone(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")

	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	var req CreateMilestoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON body"))
		return
	}

	m, err := h.svc.CreateMilestone(r.Context(), u.ID, u.IsAdmin, owner, repoSlug, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmptyMilestoneTitle):
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
		"milestone": m,
	})
}

func (h *Handler) UpdateMilestone(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	milestoneID := chi.URLParam(r, "milestone_id")

	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	var req UpdateMilestoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON body"))
		return
	}

	m, err := h.svc.UpdateMilestone(r.Context(), u.ID, u.IsAdmin, owner, repoSlug, milestoneID, req)
	if err != nil {
		if errors.Is(err, ErrMilestoneNotFound) {
			apiErrors.RespondWithError(w, apiErrors.NotFound("Milestone not found"))
			return
		}
		if errors.Is(err, ErrAccessDenied) {
			apiErrors.RespondWithError(w, apiErrors.Forbidden(err.Error()))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"milestone": m,
	})
}

func (h *Handler) DeleteMilestone(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	milestoneID := chi.URLParam(r, "milestone_id")

	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	if err := h.svc.DeleteMilestone(r.Context(), u.ID, u.IsAdmin, owner, repoSlug, milestoneID); err != nil {
		if errors.Is(err, ErrMilestoneNotFound) {
			apiErrors.RespondWithError(w, apiErrors.NotFound("Milestone not found"))
			return
		}
		if errors.Is(err, ErrAccessDenied) {
			apiErrors.RespondWithError(w, apiErrors.Forbidden(err.Error()))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "Milestone deleted successfully",
	})
}
