package search

import (
	"encoding/json"
	"net/http"
	"strconv"

	"forgehub/apps/api/internal/errors"
	"forgehub/apps/api/internal/modules/auth"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	var currentUserID string
	var isSiteAdmin bool
	if u := auth.GetUserFromContext(r.Context()); u != nil {
		currentUserID = u.ID
		isSiteAdmin = u.IsAdmin
	}

	q := r.URL.Query().Get("q")
	searchType := SearchType(r.URL.Query().Get("type"))
	if searchType == "" {
		searchType = SearchTypeRepositories
	}

	page := 1
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	perPage := 20
	if perPageStr := r.URL.Query().Get("per_page"); perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 && pp <= 100 {
			perPage = pp
		}
	}

	results, err := h.svc.Search(r.Context(), currentUserID, isSiteAdmin, q, searchType, page, perPage)
	if err != nil {
		errors.RespondWithError(w, errors.Internal("Failed to perform search"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(results)
}

func (h *Handler) QuickSearch(w http.ResponseWriter, r *http.Request) {
	var currentUserID string
	var isSiteAdmin bool
	if u := auth.GetUserFromContext(r.Context()); u != nil {
		currentUserID = u.ID
		isSiteAdmin = u.IsAdmin
	}

	q := r.URL.Query().Get("q")
	results, err := h.svc.QuickSearch(r.Context(), currentUserID, isSiteAdmin, q)
	if err != nil {
		errors.RespondWithError(w, errors.Internal("Failed to perform quick search"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(results)
}
