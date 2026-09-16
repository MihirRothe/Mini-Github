package repos

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	apiErrors "forgehub/apps/api/internal/errors"
	"forgehub/apps/api/internal/git"
	"forgehub/apps/api/internal/modules/auth"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) CreateRepo(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	var req CreateRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON request body"))
		return
	}

	repo, err := h.svc.CreateRepository(r.Context(), currentUser.ID, currentUser.IsAdmin, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidRepoSlug):
			apiErrors.RespondWithError(w, apiErrors.BadRequest(err.Error()))
		case errors.Is(err, ErrRepoSlugTaken):
			apiErrors.RespondWithError(w, apiErrors.Conflict("A repository with this name already exists"))
		default:
			apiErrors.RespondWithError(w, apiErrors.BadRequest(err.Error()))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message":    "Repository created successfully",
		"repository": repo,
	})
}

func (h *Handler) ListRepos(w http.ResponseWriter, r *http.Request) {
	var currentUserID string
	var isSiteAdmin bool
	if currentUser := auth.GetUserFromContext(r.Context()); currentUser != nil {
		currentUserID = currentUser.ID
		isSiteAdmin = currentUser.IsAdmin
	}

	filter := RepoFilter{
		OwnerSlug: r.URL.Query().Get("owner"),
		Query:     r.URL.Query().Get("q"),
	}

	repos, err := h.svc.ListRepositories(r.Context(), currentUserID, isSiteAdmin, filter)
	if err != nil {
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to list repositories"))
		return
	}
	if repos == nil {
		repos = []*Repository{}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"repositories": repos,
	})
}

func (h *Handler) GetRepo(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	if owner == "" || repoSlug == "" {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Owner and repository name are required"))
		return
	}

	var currentUserID string
	var isSiteAdmin bool
	if currentUser := auth.GetUserFromContext(r.Context()); currentUser != nil {
		currentUserID = currentUser.ID
		isSiteAdmin = currentUser.IsAdmin
	}

	repo, err := h.svc.GetRepository(r.Context(), currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			apiErrors.RespondWithError(w, apiErrors.NotFound("Repository not found"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to retrieve repository"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"repository": repo,
	})
}

func (h *Handler) UpdateRepo(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	if owner == "" || repoSlug == "" {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Owner and repository name are required"))
		return
	}

	var req UpdateRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON request body"))
		return
	}

	repo, err := h.svc.UpdateRepository(r.Context(), currentUser.ID, currentUser.IsAdmin, owner, repoSlug, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrRepoNotFound):
			apiErrors.RespondWithError(w, apiErrors.NotFound("Repository not found"))
		case errors.Is(err, ErrInsufficientRepoPerm):
			apiErrors.RespondWithError(w, apiErrors.Forbidden(err.Error()))
		default:
			apiErrors.RespondWithError(w, apiErrors.BadRequest(err.Error()))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message":    "Repository updated successfully",
		"repository": repo,
	})
}

func (h *Handler) DeleteRepo(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	if owner == "" || repoSlug == "" {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Owner and repository name are required"))
		return
	}

	err := h.svc.DeleteRepository(r.Context(), currentUser.ID, currentUser.IsAdmin, owner, repoSlug)
	if err != nil {
		switch {
		case errors.Is(err, ErrRepoNotFound):
			apiErrors.RespondWithError(w, apiErrors.NotFound("Repository not found"))
		case errors.Is(err, ErrInsufficientRepoPerm):
			apiErrors.RespondWithError(w, apiErrors.Forbidden(err.Error()))
		default:
			apiErrors.RespondWithError(w, apiErrors.Internal("Failed to delete repository"))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "Repository deleted successfully",
	})
}

func (h *Handler) GetBranches(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")

	var currentUserID string
	var isSiteAdmin bool
	if currentUser := auth.GetUserFromContext(r.Context()); currentUser != nil {
		currentUserID = currentUser.ID
		isSiteAdmin = currentUser.IsAdmin
	}

	branches, err := h.svc.GetBranches(r.Context(), currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			apiErrors.RespondWithError(w, apiErrors.NotFound("Repository not found"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to list branches"))
		return
	}
	if branches == nil {
		branches = []git.BranchInfo{}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"branches": branches,
	})
}

func (h *Handler) GetCommits(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	ref := r.URL.Query().Get("ref")
	limitStr := r.URL.Query().Get("limit")
	limit := 30
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	var currentUserID string
	var isSiteAdmin bool
	if currentUser := auth.GetUserFromContext(r.Context()); currentUser != nil {
		currentUserID = currentUser.ID
		isSiteAdmin = currentUser.IsAdmin
	}

	commits, err := h.svc.GetCommits(r.Context(), currentUserID, isSiteAdmin, owner, repoSlug, ref, limit)
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			apiErrors.RespondWithError(w, apiErrors.NotFound("Repository not found"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to list commits"))
		return
	}
	if commits == nil {
		commits = []git.CommitInfo{}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"commits": commits,
	})
}

func (h *Handler) GetTree(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	ref := chi.URLParam(r, "ref")
	subPath := chi.URLParam(r, "*")

	var currentUserID string
	var isSiteAdmin bool
	if currentUser := auth.GetUserFromContext(r.Context()); currentUser != nil {
		currentUserID = currentUser.ID
		isSiteAdmin = currentUser.IsAdmin
	}

	tree, err := h.svc.GetTree(r.Context(), currentUserID, isSiteAdmin, owner, repoSlug, ref, subPath)
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			apiErrors.RespondWithError(w, apiErrors.NotFound("Repository not found"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to list tree"))
		return
	}
	if tree == nil {
		tree = []git.TreeEntry{}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"tree": tree,
	})
}

func (h *Handler) GetBlob(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	ref := chi.URLParam(r, "ref")
	filePath := chi.URLParam(r, "*")
	if filePath == "" {
		filePath = r.URL.Query().Get("path")
	}

	var currentUserID string
	var isSiteAdmin bool
	if currentUser := auth.GetUserFromContext(r.Context()); currentUser != nil {
		currentUserID = currentUser.ID
		isSiteAdmin = currentUser.IsAdmin
	}

	blob, err := h.svc.GetBlob(r.Context(), currentUserID, isSiteAdmin, owner, repoSlug, ref, filePath)
	if err != nil {
		switch {
		case errors.Is(err, ErrRepoNotFound) || errors.Is(err, git.ErrObjectNotFound):
			apiErrors.RespondWithError(w, apiErrors.NotFound("File not found"))
		default:
			apiErrors.RespondWithError(w, apiErrors.Internal("Failed to read file blob"))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"blob": blob,
	})
}

func (h *Handler) GetReadme(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	ref := r.URL.Query().Get("ref")

	var currentUserID string
	var isSiteAdmin bool
	if currentUser := auth.GetUserFromContext(r.Context()); currentUser != nil {
		currentUserID = currentUser.ID
		isSiteAdmin = currentUser.IsAdmin
	}

	blob, err := h.svc.GetReadme(r.Context(), currentUserID, isSiteAdmin, owner, repoSlug, ref)
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) || errors.Is(err, git.ErrObjectNotFound) {
			apiErrors.RespondWithError(w, apiErrors.NotFound("README not found"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to read README"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"readme": blob,
	})
}
