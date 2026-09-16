package repos

import (
	"net/http"
	"strings"

	"forgehub/apps/api/internal/git"
	"forgehub/apps/api/internal/modules/auth"
	"forgehub/apps/api/internal/modules/orgs"

	"github.com/go-chi/chi/v5"
)

type GitHTTPHandler struct {
	svc     Service
	authSvc *auth.Service
}

func NewGitHTTPHandler(svc Service, authSvc *auth.Service) *GitHTTPHandler {
	return &GitHTTPHandler{
		svc:     svc,
		authSvc: authSvc,
	}
}

func (h *GitHTTPHandler) InfoRefs(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	repoSlug = strings.TrimSuffix(repoSlug, ".git")

	service := r.URL.Query().Get("service")
	if service != "git-upload-pack" && service != "git-receive-pack" {
		http.Error(w, "Unsupported service parameter", http.StatusBadRequest)
		return
	}

	user, repo, perm, ok := h.authenticateAndAuthorize(w, r, owner, repoSlug, service)
	if !ok {
		return
	}
	_ = user
	_ = perm

	git.HandleInfoRefs(w, r, repo.DiskPath, service)
}

func (h *GitHTTPHandler) ServiceRPC(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoSlug := chi.URLParam(r, "repo")
	repoSlug = strings.TrimSuffix(repoSlug, ".git")

	service := chi.URLParam(r, "service")
	if service != "git-upload-pack" && service != "git-receive-pack" {
		http.Error(w, "Unsupported service endpoint", http.StatusBadRequest)
		return
	}

	user, repo, perm, ok := h.authenticateAndAuthorize(w, r, owner, repoSlug, service)
	if !ok {
		return
	}
	_ = user
	_ = perm

	git.HandleServiceRPC(w, r, repo.DiskPath, service)
}

func (h *GitHTTPHandler) authenticateAndAuthorize(
	w http.ResponseWriter,
	r *http.Request,
	owner, repoSlug, serviceName string,
) (*auth.User, *Repository, orgs.Permission, bool) {
	ctx := r.Context()

	// 1. Fetch repository directly from store (bypass permission check initially to read visibility)
	repo, err := h.svc.(*service).repoStore.GetRepoByOwnerAndSlug(ctx, owner, repoSlug)
	if err != nil {
		w.Header().Set("WWW-Authenticate", `Basic realm="ForgeHub"`)
		http.Error(w, "Repository not found", http.StatusNotFound)
		return nil, nil, orgs.PermNone, false
	}

	// 2. Extract Basic Auth
	username, password, hasAuth := r.BasicAuth()
	var user *auth.User

	if hasAuth {
		if strings.HasPrefix(password, "fh_pat_") {
			// Personal Access Token
			u, _, err := h.authSvc.ValidateAPIToken(ctx, password)
			if err == nil && u != nil {
				user = u
			}
		} else {
			// Standard password
			u, err := h.authSvc.AuthenticateCredentials(ctx, username, password)
			if err == nil && u != nil {
				user = u
			}
		}

		if user == nil {
			w.Header().Set("WWW-Authenticate", `Basic realm="ForgeHub"`)
			http.Error(w, "Invalid username, password, or access token", http.StatusUnauthorized)
			return nil, nil, orgs.PermNone, false
		}
	}

	// 3. Evaluate Permissions
	var userID string
	var isSiteAdmin bool
	if user != nil {
		userID = user.ID
		isSiteAdmin = user.IsAdmin
	}

	perm, err := h.svc.ResolveUserPermission(ctx, userID, isSiteAdmin, repo)
	if err != nil {
		http.Error(w, "Failed to resolve permissions", http.StatusInternalServerError)
		return nil, nil, orgs.PermNone, false
	}

	// 4. Enforce Access based on action
	if serviceName == "git-upload-pack" {
		// Read / Clone / Fetch
		if repo.Visibility != "public" && !perm.Includes(orgs.PermRead) {
			w.Header().Set("WWW-Authenticate", `Basic realm="ForgeHub"`)
			http.Error(w, "Authentication required to read repository", http.StatusUnauthorized)
			return nil, nil, orgs.PermNone, false
		}
	} else if serviceName == "git-receive-pack" {
		// Push / Write
		if user == nil {
			w.Header().Set("WWW-Authenticate", `Basic realm="ForgeHub"`)
			http.Error(w, "Authentication required to push to repository", http.StatusUnauthorized)
			return nil, nil, orgs.PermNone, false
		}
		if !perm.Includes(orgs.PermWrite) {
			http.Error(w, "Forbidden: You do not have write permissions to push to this repository", http.StatusForbidden)
			return nil, nil, orgs.PermNone, false
		}
	}

	return user, repo, perm, true
}
