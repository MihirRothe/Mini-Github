package ci

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

func (h *Handler) ListRuns(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")

	var currentUserID string
	var isSiteAdmin bool
	if u := auth.GetUserFromContext(r.Context()); u != nil {
		currentUserID = u.ID
		isSiteAdmin = u.IsAdmin
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	filter := RunFilter{
		Status:   r.URL.Query().Get("status"),
		Branch:   r.URL.Query().Get("branch"),
		Event:    r.URL.Query().Get("event"),
		Page:     page,
		PageSize: pageSize,
	}

	runs, total, err := h.svc.ListRuns(r.Context(), currentUserID, isSiteAdmin, owner, repoSlug, filter)
	if err != nil {
		if errors.Is(err, ErrAccessDenied) {
			apiErrors.RespondWithError(w, apiErrors.Forbidden("Access denied to repository"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to list pipeline runs"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"runs":  runs,
		"total": total,
	})
}

func (h *Handler) TriggerRun(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")

	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	var req TriggerRunRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	run, err := h.svc.TriggerWorkflow(r.Context(), u.ID, u.IsAdmin, owner, repoSlug, req)
	if err != nil {
		if errors.Is(err, ErrAccessDenied) {
			apiErrors.RespondWithError(w, apiErrors.Forbidden("Write permission required to trigger workflows"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "Workflow run queued successfully",
		"run":     run,
	})
}

func (h *Handler) GetRun(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	runID := chi.URLParam(r, "run_id")

	var currentUserID string
	var isSiteAdmin bool
	if u := auth.GetUserFromContext(r.Context()); u != nil {
		currentUserID = u.ID
		isSiteAdmin = u.IsAdmin
	}

	detail, err := h.svc.GetRun(r.Context(), currentUserID, isSiteAdmin, owner, repoSlug, runID)
	if err != nil {
		if errors.Is(err, ErrRunNotFound) {
			apiErrors.RespondWithError(w, apiErrors.NotFound("Pipeline run not found"))
			return
		}
		if errors.Is(err, ErrAccessDenied) {
			apiErrors.RespondWithError(w, apiErrors.Forbidden("Access denied"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to get pipeline run"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"run": detail,
	})
}

func (h *Handler) CancelRun(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	runID := chi.URLParam(r, "run_id")

	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	if err := h.svc.CancelRun(r.Context(), u.ID, u.IsAdmin, owner, repoSlug, runID); err != nil {
		if errors.Is(err, ErrRunNotFound) {
			apiErrors.RespondWithError(w, apiErrors.NotFound("Pipeline run not found"))
			return
		}
		if errors.Is(err, ErrRunAlreadyFinished) {
			apiErrors.RespondWithError(w, apiErrors.Conflict("Run is already finished"))
			return
		}
		if errors.Is(err, ErrAccessDenied) {
			apiErrors.RespondWithError(w, apiErrors.Forbidden("Write permission required"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "Pipeline run cancelled successfully",
	})
}

func (h *Handler) Rerun(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	runID := chi.URLParam(r, "run_id")

	u := auth.GetUserFromContext(r.Context())
	if u == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("Authentication required"))
		return
	}

	run, err := h.svc.Rerun(r.Context(), u.ID, u.IsAdmin, owner, repoSlug, runID)
	if err != nil {
		apiErrors.RespondWithError(w, apiErrors.Internal(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"message": "Workflow rerun queued successfully",
		"run":     run,
	})
}

func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	jobID := chi.URLParam(r, "job_id")

	var currentUserID string
	var isSiteAdmin bool
	if u := auth.GetUserFromContext(r.Context()); u != nil {
		currentUserID = u.ID
		isSiteAdmin = u.IsAdmin
	}

	job, err := h.svc.GetJob(r.Context(), currentUserID, isSiteAdmin, owner, repoSlug, jobID)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			apiErrors.RespondWithError(w, apiErrors.NotFound("Pipeline job not found"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to get job"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"job": job,
	})
}

func (h *Handler) GetJobLogs(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	jobID := chi.URLParam(r, "job_id")

	var currentUserID string
	var isSiteAdmin bool
	if u := auth.GetUserFromContext(r.Context()); u != nil {
		currentUserID = u.ID
		isSiteAdmin = u.IsAdmin
	}

	afterLine, _ := strconv.Atoi(r.URL.Query().Get("after"))

	logs, err := h.svc.GetJobLogs(r.Context(), currentUserID, isSiteAdmin, owner, repoSlug, jobID, afterLine)
	if err != nil {
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to get logs"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"logs": logs,
	})
}

func (h *Handler) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	ref := r.URL.Query().Get("ref")

	var currentUserID string
	var isSiteAdmin bool
	if u := auth.GetUserFromContext(r.Context()); u != nil {
		currentUserID = u.ID
		isSiteAdmin = u.IsAdmin
	}

	workflows, err := h.svc.ListWorkflows(r.Context(), currentUserID, isSiteAdmin, owner, repoSlug, ref)
	if err != nil {
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to list workflows"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"workflows": workflows,
	})
}
