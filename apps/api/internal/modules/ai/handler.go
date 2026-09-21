package ai

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

// Explain handles POST /api/v1/ai/explain
func (h *Handler) Explain(w http.ResponseWriter, r *http.Request) {
	var req ExplainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON request body"))
		return
	}

	providerID := r.URL.Query().Get("provider")
	resp, err := h.svc.ExplainCode(r.Context(), req, providerID)
	if err != nil {
		if errors.Is(err, ErrEmptyInput) {
			apiErrors.RespondWithError(w, apiErrors.BadRequest("Code content cannot be empty"))
			return
		}
		if errors.Is(err, ErrInputTooLarge) {
			apiErrors.RespondWithError(w, apiErrors.BadRequest("Code snippet exceeds maximum permissible size"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to analyze code: "+err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"explanation": resp,
	})
}

// Review handles POST /api/v1/ai/review
func (h *Handler) Review(w http.ResponseWriter, r *http.Request) {
	var req ReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON request body"))
		return
	}

	providerID := r.URL.Query().Get("provider")
	resp, err := h.svc.ReviewDiff(r.Context(), req, providerID)
	if err != nil {
		if errors.Is(err, ErrEmptyInput) {
			apiErrors.RespondWithError(w, apiErrors.BadRequest("Diff content cannot be empty"))
			return
		}
		if errors.Is(err, ErrInputTooLarge) {
			apiErrors.RespondWithError(w, apiErrors.BadRequest("Diff exceeds maximum permissible size"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to review diff: "+err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"review": resp,
	})
}

// GenerateTests handles POST /api/v1/ai/generate-tests
func (h *Handler) GenerateTests(w http.ResponseWriter, r *http.Request) {
	var req GenerateTestsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON request body"))
		return
	}

	providerID := r.URL.Query().Get("provider")
	resp, err := h.svc.GenerateTests(r.Context(), req, providerID)
	if err != nil {
		if errors.Is(err, ErrEmptyInput) {
			apiErrors.RespondWithError(w, apiErrors.BadRequest("Code content cannot be empty"))
			return
		}
		if errors.Is(err, ErrInputTooLarge) {
			apiErrors.RespondWithError(w, apiErrors.BadRequest("Code snippet exceeds maximum permissible size"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to generate tests: "+err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"test_suite": resp,
	})
}

// Chat handles POST /api/v1/ai/chat
func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON request body"))
		return
	}

	providerID := r.URL.Query().Get("provider")
	resp, err := h.svc.Chat(r.Context(), req, providerID)
	if err != nil {
		if errors.Is(err, ErrEmptyInput) {
			apiErrors.RespondWithError(w, apiErrors.BadRequest("Chat messages cannot be empty"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("AI chat failed: "+err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// Providers handles GET /api/v1/ai/providers
func (h *Handler) Providers(w http.ResponseWriter, r *http.Request) {
	resp := h.svc.GetProvidersStatus()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// ReviewPullRequest handles POST /api/v1/repos/{owner}/{repo}/pulls/{number}/ai-review
func (h *Handler) ReviewPullRequest(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repo := chi.URLParam(r, "repo")
	numStr := chi.URLParam(r, "number")

	number, err := strconv.Atoi(numStr)
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

	postComment := r.URL.Query().Get("post_comment") == "true"
	providerID := r.URL.Query().Get("provider")

	resp, err := h.svc.ReviewPullRequest(r.Context(), currentUserID, isSiteAdmin, owner, repo, number, postComment, providerID)
	if err != nil {
		if errors.Is(err, ErrEmptyDiff) {
			apiErrors.RespondWithError(w, apiErrors.BadRequest("Pull request has no changes to review"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to generate AI review: "+err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"review": resp,
	})
}

// ExplainBlob handles POST /api/v1/repos/{owner}/{repo}/ai/explain-blob
func (h *Handler) ExplainBlob(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repo := chi.URLParam(r, "repo")
	ref := r.URL.Query().Get("ref")
	path := r.URL.Query().Get("path")

	if ref == "" || path == "" {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("ref and path query parameters are required"))
		return
	}

	var currentUserID string
	var isSiteAdmin bool
	if u := auth.GetUserFromContext(r.Context()); u != nil {
		currentUserID = u.ID
		isSiteAdmin = u.IsAdmin
	}

	providerID := r.URL.Query().Get("provider")
	resp, err := h.svc.ExplainBlob(r.Context(), currentUserID, isSiteAdmin, owner, repo, ref, path, providerID)
	if err != nil {
		if errors.Is(err, ErrBinaryFileNotSupported) {
			apiErrors.RespondWithError(w, apiErrors.BadRequest("Binary files cannot be analyzed by AI"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to explain file: "+err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"explanation": resp,
	})
}

// GenerateBlobTests handles POST /api/v1/repos/{owner}/{repo}/ai/generate-blob-tests
func (h *Handler) GenerateBlobTests(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repo := chi.URLParam(r, "repo")
	ref := r.URL.Query().Get("ref")
	path := r.URL.Query().Get("path")

	if ref == "" || path == "" {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("ref and path query parameters are required"))
		return
	}

	var currentUserID string
	var isSiteAdmin bool
	if u := auth.GetUserFromContext(r.Context()); u != nil {
		currentUserID = u.ID
		isSiteAdmin = u.IsAdmin
	}

	providerID := r.URL.Query().Get("provider")
	resp, err := h.svc.GenerateBlobTests(r.Context(), currentUserID, isSiteAdmin, owner, repo, ref, path, providerID)
	if err != nil {
		if errors.Is(err, ErrBinaryFileNotSupported) {
			apiErrors.RespondWithError(w, apiErrors.BadRequest("Binary files cannot have tests generated"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to generate tests: "+err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"test_suite": resp,
	})
}
