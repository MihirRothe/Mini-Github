package pulls

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

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

func (h *Handler) ListPulls(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")

	var currentUserID string
	var isSiteAdmin bool
	if u := auth.GetUserFromContext(r.Context()); u != nil {
		currentUserID = u.ID
		isSiteAdmin = u.IsAdmin
	}

	filter := PRFilter{
		State:          r.URL.Query().Get("state"),
		AuthorUsername: r.URL.Query().Get("author"),
		SourceBranch:   r.URL.Query().Get("source_branch"),
		TargetBranch:   r.URL.Query().Get("target_branch"),
		Query:          r.URL.Query().Get("q"),
	}

	prs, openC, closedC, mergedC, err := h.svc.ListPullRequests(r.Context(), currentUserID, isSiteAdmin, owner, repoSlug, filter)
	if err != nil {
		if errors.Is(err, ErrAccessDenied) {
			apiErrors.RespondWithError(w, apiErrors.Forbidden("Access denied to repository"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to list pull requests"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"pull_requests": prs,
		"open_count":    openC,
		"closed_count":  closedC,
		"merged_count":  mergedC,
	})
}

func (h *Handler) CreatePull(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")

	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	var req CreatePRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON body"))
		return
	}

	pr, err := h.svc.CreatePullRequest(r.Context(), u.ID, u.IsAdmin, owner, repoSlug, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmptyTitle):
			apiErrors.RespondWithError(w, apiErrors.BadRequest(err.Error()))
		case errors.Is(err, ErrSameBranches):
			apiErrors.RespondWithError(w, apiErrors.BadRequest(err.Error()))
		case errors.Is(err, ErrBranchNotFound):
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
		"message":      "Pull request created successfully",
		"pull_request": pr,
	})
}

func (h *Handler) GetPull(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	numberStr := chi.URLParam(r, "number")

	number, err := strconv.Atoi(numberStr)
	if err != nil || number <= 0 {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid pull request number"))
		return
	}

	var currentUserID string
	var isSiteAdmin bool
	if u := auth.GetUserFromContext(r.Context()); u != nil {
		currentUserID = u.ID
		isSiteAdmin = u.IsAdmin
	}

	detail, err := h.svc.GetPullRequest(r.Context(), currentUserID, isSiteAdmin, owner, repoSlug, number)
	if err != nil {
		if errors.Is(err, ErrPRNotFound) {
			apiErrors.RespondWithError(w, apiErrors.NotFound("Pull request not found"))
			return
		}
		if errors.Is(err, ErrAccessDenied) {
			apiErrors.RespondWithError(w, apiErrors.Forbidden("Access denied"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to get pull request"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"pull_request": detail,
	})
}

func (h *Handler) UpdatePull(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	numberStr := chi.URLParam(r, "number")

	number, err := strconv.Atoi(numberStr)
	if err != nil || number <= 0 {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid pull request number"))
		return
	}

	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	var req UpdatePRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON body"))
		return
	}

	pr, err := h.svc.UpdatePullRequest(r.Context(), u.ID, u.IsAdmin, owner, repoSlug, number, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrPRNotFound):
			apiErrors.RespondWithError(w, apiErrors.NotFound("Pull request not found"))
		case errors.Is(err, ErrPRAlreadyMerged):
			apiErrors.RespondWithError(w, apiErrors.Conflict(err.Error()))
		case errors.Is(err, ErrAccessDenied):
			apiErrors.RespondWithError(w, apiErrors.Forbidden("Forbidden"))
		default:
			apiErrors.RespondWithError(w, apiErrors.Internal(err.Error()))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"pull_request": pr,
	})
}

func (h *Handler) MergePull(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	numberStr := chi.URLParam(r, "number")

	number, err := strconv.Atoi(numberStr)
	if err != nil || number <= 0 {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid pull request number"))
		return
	}

	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	var req MergePRRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	pr, err := h.svc.MergePullRequest(r.Context(), u.ID, u.IsAdmin, owner, repoSlug, number, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrPRNotFound):
			apiErrors.RespondWithError(w, apiErrors.NotFound("Pull request not found"))
		case errors.Is(err, ErrPRAlreadyMerged):
			apiErrors.RespondWithError(w, apiErrors.Conflict("Pull request is already merged"))
		case errors.Is(err, ErrPRClosed):
			apiErrors.RespondWithError(w, apiErrors.Conflict("Pull request is closed and cannot be merged"))
		case errors.Is(err, ErrHasConflicts):
			apiErrors.RespondWithError(w, apiErrors.Conflict("Cannot merge: conflicts exist between branches"))
		case errors.Is(err, ErrAccessDenied):
			apiErrors.RespondWithError(w, apiErrors.Forbidden("Write permission required to merge pull request"))
		default:
			apiErrors.RespondWithError(w, apiErrors.Internal(err.Error()))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message":      "Pull request merged successfully",
		"pull_request": pr,
	})
}

func (h *Handler) GetPullDiff(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	numberStr := chi.URLParam(r, "number")

	number, err := strconv.Atoi(numberStr)
	if err != nil || number <= 0 {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid pull request number"))
		return
	}

	var currentUserID string
	var isSiteAdmin bool
	if u := auth.GetUserFromContext(r.Context()); u != nil {
		currentUserID = u.ID
		isSiteAdmin = u.IsAdmin
	}

	diff, err := h.svc.GetPRDiff(r.Context(), currentUserID, isSiteAdmin, owner, repoSlug, number)
	if err != nil {
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to calculate diff"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"diff": diff,
	})
}

func (h *Handler) GetPullCommits(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	numberStr := chi.URLParam(r, "number")

	number, err := strconv.Atoi(numberStr)
	if err != nil || number <= 0 {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid pull request number"))
		return
	}

	var currentUserID string
	var isSiteAdmin bool
	if u := auth.GetUserFromContext(r.Context()); u != nil {
		currentUserID = u.ID
		isSiteAdmin = u.IsAdmin
	}

	commits, err := h.svc.GetPRCommits(r.Context(), currentUserID, isSiteAdmin, owner, repoSlug, number)
	if err != nil {
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to fetch commits"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"commits": commits,
	})
}

func (h *Handler) CreateReview(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	numberStr := chi.URLParam(r, "number")

	number, err := strconv.Atoi(numberStr)
	if err != nil || number <= 0 {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid pull request number"))
		return
	}

	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	var req CreateReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON body"))
		return
	}

	review, err := h.svc.CreateReview(r.Context(), u.ID, u.IsAdmin, owner, repoSlug, number, req)
	if err != nil {
		if errors.Is(err, ErrAccessDenied) {
			apiErrors.RespondWithError(w, apiErrors.Forbidden("Access denied"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"review": review,
	})
}

func (h *Handler) ListReviews(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	numberStr := chi.URLParam(r, "number")

	number, err := strconv.Atoi(numberStr)
	if err != nil || number <= 0 {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid pull request number"))
		return
	}

	var currentUserID string
	var isSiteAdmin bool
	if u := auth.GetUserFromContext(r.Context()); u != nil {
		currentUserID = u.ID
		isSiteAdmin = u.IsAdmin
	}

	reviews, err := h.svc.ListReviews(r.Context(), currentUserID, isSiteAdmin, owner, repoSlug, number)
	if err != nil {
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to list reviews"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"reviews": reviews,
	})
}

func (h *Handler) CreateComment(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	numberStr := chi.URLParam(r, "number")

	number, err := strconv.Atoi(numberStr)
	if err != nil || number <= 0 {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid pull request number"))
		return
	}

	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	var req CreatePRCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON body"))
		return
	}

	comment, err := h.svc.CreateComment(r.Context(), u.ID, u.IsAdmin, owner, repoSlug, number, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmptyCommentBody):
			apiErrors.RespondWithError(w, apiErrors.BadRequest(err.Error()))
		case errors.Is(err, ErrAccessDenied):
			apiErrors.RespondWithError(w, apiErrors.Forbidden("Access denied"))
		default:
			apiErrors.RespondWithError(w, apiErrors.Internal(err.Error()))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"comment": comment,
	})
}

func (h *Handler) ListComments(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	numberStr := chi.URLParam(r, "number")

	number, err := strconv.Atoi(numberStr)
	if err != nil || number <= 0 {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid pull request number"))
		return
	}

	var currentUserID string
	var isSiteAdmin bool
	if u := auth.GetUserFromContext(r.Context()); u != nil {
		currentUserID = u.ID
		isSiteAdmin = u.IsAdmin
	}

	comments, err := h.svc.ListComments(r.Context(), currentUserID, isSiteAdmin, owner, repoSlug, number)
	if err != nil {
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to list comments"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"comments": comments,
	})
}

func (h *Handler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	commentID := chi.URLParam(r, "comment_id")

	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	if err := h.svc.DeleteComment(r.Context(), u.ID, u.IsAdmin, commentID); err != nil {
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to delete comment"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "Comment deleted successfully",
	})
}

func (h *Handler) Compare(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	spec := chi.URLParam(r, "spec") // e.g. "main...feature"

	parts := strings.Split(spec, "...")
	if len(parts) != 2 {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid compare spec: must be in format base...head"))
		return
	}
	baseRef := strings.TrimSpace(parts[0])
	headRef := strings.TrimSpace(parts[1])

	var currentUserID string
	var isSiteAdmin bool
	if u := auth.GetUserFromContext(r.Context()); u != nil {
		currentUserID = u.ID
		isSiteAdmin = u.IsAdmin
	}

	diff, commits, canMerge, err := h.svc.CompareBranches(r.Context(), currentUserID, isSiteAdmin, owner, repoSlug, baseRef, headRef)
	if err != nil {
		apiErrors.RespondWithError(w, apiErrors.Internal(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"base_ref":  baseRef,
		"head_ref":  headRef,
		"diff":      diff,
		"commits":   commits,
		"can_merge": canMerge,
	})
}
