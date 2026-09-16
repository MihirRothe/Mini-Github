package router

import (
	"net/http"
	"time"

	internalAuth "forgehub/apps/api/internal/auth"
	"forgehub/apps/api/internal/config"
	"forgehub/apps/api/internal/database"
	"forgehub/apps/api/internal/errors"
	"forgehub/apps/api/internal/git"
	"forgehub/apps/api/internal/middleware"
	authModule "forgehub/apps/api/internal/modules/auth"
	"forgehub/apps/api/internal/modules/ci"
	"forgehub/apps/api/internal/modules/health"
	"forgehub/apps/api/internal/modules/issues"
	"forgehub/apps/api/internal/modules/orgs"
	"forgehub/apps/api/internal/modules/pulls"
	"forgehub/apps/api/internal/modules/repos"
	"forgehub/apps/api/internal/modules/search"
	"forgehub/apps/api/internal/modules/tokens"
	"forgehub/apps/api/internal/modules/users"
	"forgehub/apps/api/internal/modules/webhooks"
	"forgehub/apps/api/internal/redis"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

type RouterOptions struct {
	Config *config.Config
	DB     *database.DB
	Redis  *redis.Client
}

func New(opts RouterOptions) *chi.Mux {
	r := chi.NewRouter()

	// Global Core Middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.StructuredLogger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.SecurityHeaders)

	// Cross-Origin Resource Sharing (CORS)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:3000", "http://127.0.0.1:5173", "http://localhost:8080"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link", "X-Request-ID", "X-Total-Count"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Domain Services & Repositories
	authRepo := authModule.NewRepository(opts.DB)
	authSvc := authModule.NewService(authRepo, opts.Config)
	orgsRepo := orgs.NewRepository(opts.DB, authRepo)
	orgsSvc := orgs.NewService(orgsRepo, authRepo)

	// Git Storage & Repositories (Phase 4)
	gitStorage, err := git.NewStorage(opts.Config.GitRootDir)
	if err != nil {
		panic("failed to initialize git storage: " + err.Error())
	}
	gitReader := git.NewReader()
	repoStore := repos.NewRepositoryStore(opts.DB)
	reposSvc := repos.NewService(repoStore, gitStorage, gitReader, authRepo, orgsRepo, opts.Config)

	// Issues & Collaboration (Phase 5)
	issuesStore := issues.NewRepositoryStore(opts.DB, authRepo)
	issuesSvc := issues.NewService(issuesStore, reposSvc, authRepo)
	issuesHandler := issues.NewHandler(issuesSvc)

	// Pull Requests & Reviews (Phase 6)
	pullsStore := pulls.NewRepositoryStore(opts.DB, authRepo)
	pullsSvc := pulls.NewService(pullsStore, reposSvc, gitStorage, gitReader, authRepo)
	pullsHandler := pulls.NewHandler(pullsSvc)

	// CI/CD Pipelines & Container Runners (Phase 7)
	ciStore := ci.NewRepositoryStore(opts.DB, authRepo)
	ciExecutor := ci.NewLocalExecutor(ciStore)
	ciSvc := ci.NewService(ciStore, reposSvc, gitStorage, gitReader, authRepo, ciExecutor)
	ciHandler := ci.NewHandler(ciSvc)

	// Webhooks & Automation (Phase 8)
	webhooksStore := webhooks.NewRepositoryStore(opts.DB)
	webhooksDispatcher := webhooks.NewDispatcher(webhooksStore)
	webhooksSvc := webhooks.NewService(webhooksStore, webhooksDispatcher, reposSvc, authRepo)
	webhooksHandler := webhooks.NewHandler(webhooksSvc)

	// Global Search & Code Navigation (Phase 9)
	searchStore := search.NewSearchStore(opts.DB)
	searchSvc := search.NewService(searchStore, gitReader)
	searchHandler := search.NewHandler(searchSvc)

	// Authentication Context Middleware
	r.Use(middleware.Authenticate(authSvc, opts.Config.SessionCookieName))

	// Auth Rate Limiter: 10 attempts per minute
	authRateLimiter := internalAuth.NewRateLimiter(10, 1*time.Minute)

	// Handlers
	authHandler := authModule.NewHandler(authSvc, opts.Config)
	userHandler := users.NewHandler(authSvc)
	tokenHandler := tokens.NewHandler(authSvc)
	orgsHandler := orgs.NewHandler(orgsSvc)
	reposHandler := repos.NewHandler(reposSvc)
	gitHTTPHandler := repos.NewGitHTTPHandler(reposSvc, authSvc)
	healthHandler := health.NewHandler(opts.DB, opts.Redis)

	// Git Smart HTTP Protocol Endpoints (clone, fetch, push)
	r.Get("/{owner}/{repo}.git/info/refs", gitHTTPHandler.InfoRefs)
	r.Post("/{owner}/{repo}.git/{service:(git-upload-pack|git-receive-pack)}", gitHTTPHandler.ServiceRPC)
	r.Get("/{owner}/{repo}/info/refs", gitHTTPHandler.InfoRefs)
	r.Post("/{owner}/{repo}/{service:(git-upload-pack|git-receive-pack)}", gitHTTPHandler.ServiceRPC)

	// Custom 404 & 405 error handlers
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		errors.RespondWithError(w, errors.NotFound("The requested route does not exist"))
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		errors.RespondWithError(w, errors.New(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed"))
	})

	// Top-level Health Probes
	r.Get("/healthz", healthHandler.HealthCheck)
	r.Get("/readyz", healthHandler.Readiness)
	r.Get("/livez", healthHandler.Liveness)

	// API v1 Routing Tree
	r.Route("/api/v1", func(v1 chi.Router) {
		v1.Get("/health", healthHandler.HealthCheck)
		v1.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"message":"pong"}`))
		})

		// Global Search (Phase 9)
		v1.Get("/search", searchHandler.Search)
		v1.Get("/search/quick", searchHandler.QuickSearch)

		// Auth Endpoints
		v1.Route("/auth", func(authRouter chi.Router) {
			authRouter.With(authRateLimiter.Middleware).Post("/register", authHandler.Register)
			authRouter.With(authRateLimiter.Middleware).Post("/login", authHandler.Login)
			authRouter.Post("/logout", authHandler.Logout)
			authRouter.With(middleware.RequireAuth).Get("/me", authHandler.Me)
		})

		// User Profiles
		v1.Route("/users", func(userRouter chi.Router) {
			userRouter.Get("/{username}", userHandler.GetProfile)
			userRouter.With(middleware.RequireAuth).Patch("/me", userHandler.UpdateProfile)
			userRouter.With(middleware.RequireAuth).Put("/me/password", userHandler.ChangePassword)
		})

		// Personal Access Tokens
		v1.Route("/tokens", func(tokenRouter chi.Router) {
			tokenRouter.Use(middleware.RequireAuth)
			tokenRouter.Post("/", tokenHandler.CreateToken)
			tokenRouter.Get("/", tokenHandler.ListTokens)
			tokenRouter.Delete("/{id}", tokenHandler.DeleteToken)
		})

		// Organizations & Teams (Phase 3)
		v1.Route("/orgs", func(orgRouter chi.Router) {
			orgRouter.With(middleware.RequireAuth).Post("/", orgsHandler.CreateOrganization)
			orgRouter.With(middleware.RequireAuth).Get("/", orgsHandler.ListUserOrganizations)
			orgRouter.Get("/{org}", orgsHandler.GetOrganization)
			orgRouter.With(middleware.RequireAuth).Patch("/{org}", orgsHandler.UpdateOrganization)
			orgRouter.With(middleware.RequireAuth).Delete("/{org}", orgsHandler.DeleteOrganization)

			// Organization Members
			orgRouter.Get("/{org}/members", orgsHandler.ListMembers)
			orgRouter.With(middleware.RequireAuth).Post("/{org}/members", orgsHandler.AddMember)
			orgRouter.With(middleware.RequireAuth).Patch("/{org}/members/{username}", orgsHandler.UpdateMemberRole)
			orgRouter.With(middleware.RequireAuth).Delete("/{org}/members/{username}", orgsHandler.RemoveMember)

			// Organization Teams
			orgRouter.Get("/{org}/teams", orgsHandler.ListTeams)
			orgRouter.With(middleware.RequireAuth).Post("/{org}/teams", orgsHandler.CreateTeam)
			orgRouter.Get("/{org}/teams/{team}", orgsHandler.GetTeam)
			orgRouter.With(middleware.RequireAuth).Patch("/{org}/teams/{team}", orgsHandler.UpdateTeam)
			orgRouter.With(middleware.RequireAuth).Delete("/{org}/teams/{team}", orgsHandler.DeleteTeam)

			// Team Members
			orgRouter.Get("/{org}/teams/{team}/members", orgsHandler.ListTeamMembers)
			orgRouter.With(middleware.RequireAuth).Post("/{org}/teams/{team}/members", orgsHandler.AddTeamMember)
			orgRouter.With(middleware.RequireAuth).Patch("/{org}/teams/{team}/members/{username}", orgsHandler.UpdateTeamMemberRole)
			orgRouter.With(middleware.RequireAuth).Delete("/{org}/teams/{team}/members/{username}", orgsHandler.RemoveTeamMember)
		})

		// Repositories & Git Data (Phase 4)
		v1.Route("/repos", func(repoRouter chi.Router) {
			repoRouter.With(middleware.RequireAuth).Post("/", reposHandler.CreateRepo)
			repoRouter.Get("/", reposHandler.ListRepos)
			repoRouter.Get("/{owner}/{repo}", reposHandler.GetRepo)
			repoRouter.With(middleware.RequireAuth).Patch("/{owner}/{repo}", reposHandler.UpdateRepo)
			repoRouter.With(middleware.RequireAuth).Delete("/{owner}/{repo}", reposHandler.DeleteRepo)

			// Git Tree, Commits & Blobs
			repoRouter.Get("/{owner}/{repo}/branches", reposHandler.GetBranches)
			repoRouter.Get("/{owner}/{repo}/commits", reposHandler.GetCommits)
			repoRouter.Get("/{owner}/{repo}/tree/{ref}", reposHandler.GetTree)
			repoRouter.Get("/{owner}/{repo}/tree/{ref}/*", reposHandler.GetTree)
			repoRouter.Get("/{owner}/{repo}/blob/{ref}/*", reposHandler.GetBlob)
			repoRouter.Get("/{owner}/{repo}/readme", reposHandler.GetReadme)

			// Issues & Collaboration (Phase 5)
			repoRouter.Get("/{owner}/{repo}/issues", issuesHandler.ListIssues)
			repoRouter.With(middleware.RequireAuth).Post("/{owner}/{repo}/issues", issuesHandler.CreateIssue)
			repoRouter.Get("/{owner}/{repo}/issues/{number}", issuesHandler.GetIssue)
			repoRouter.With(middleware.RequireAuth).Patch("/{owner}/{repo}/issues/{number}", issuesHandler.UpdateIssue)
			repoRouter.With(middleware.RequireAuth).Delete("/{owner}/{repo}/issues/{number}", issuesHandler.DeleteIssue)

			// Issue Comments
			repoRouter.With(middleware.RequireAuth).Post("/{owner}/{repo}/issues/{number}/comments", issuesHandler.CreateComment)
			repoRouter.With(middleware.RequireAuth).Patch("/{owner}/{repo}/issues/{number}/comments/{comment_id}", issuesHandler.UpdateComment)
			repoRouter.With(middleware.RequireAuth).Delete("/{owner}/{repo}/issues/{number}/comments/{comment_id}", issuesHandler.DeleteComment)

			// Issue Labels
			repoRouter.Get("/{owner}/{repo}/labels", issuesHandler.ListLabels)
			repoRouter.With(middleware.RequireAuth).Post("/{owner}/{repo}/labels", issuesHandler.CreateLabel)
			repoRouter.With(middleware.RequireAuth).Patch("/{owner}/{repo}/labels/{label_id}", issuesHandler.UpdateLabel)
			repoRouter.With(middleware.RequireAuth).Delete("/{owner}/{repo}/labels/{label_id}", issuesHandler.DeleteLabel)

			// Milestones
			repoRouter.Get("/{owner}/{repo}/milestones", issuesHandler.ListMilestones)
			repoRouter.With(middleware.RequireAuth).Post("/{owner}/{repo}/milestones", issuesHandler.CreateMilestone)
			repoRouter.With(middleware.RequireAuth).Patch("/{owner}/{repo}/milestones/{milestone_id}", issuesHandler.UpdateMilestone)
			repoRouter.With(middleware.RequireAuth).Delete("/{owner}/{repo}/milestones/{milestone_id}", issuesHandler.DeleteMilestone)

			// Pull Requests & Code Review (Phase 6)
			repoRouter.Get("/{owner}/{repo}/pulls", pullsHandler.ListPulls)
			repoRouter.With(middleware.RequireAuth).Post("/{owner}/{repo}/pulls", pullsHandler.CreatePull)
			repoRouter.Get("/{owner}/{repo}/pulls/{number}", pullsHandler.GetPull)
			repoRouter.With(middleware.RequireAuth).Patch("/{owner}/{repo}/pulls/{number}", pullsHandler.UpdatePull)
			repoRouter.With(middleware.RequireAuth).Post("/{owner}/{repo}/pulls/{number}/merge", pullsHandler.MergePull)
			repoRouter.Get("/{owner}/{repo}/pulls/{number}/diff", pullsHandler.GetPullDiff)
			repoRouter.Get("/{owner}/{repo}/pulls/{number}/commits", pullsHandler.GetPullCommits)
			repoRouter.Get("/{owner}/{repo}/pulls/{number}/reviews", pullsHandler.ListReviews)
			repoRouter.With(middleware.RequireAuth).Post("/{owner}/{repo}/pulls/{number}/reviews", pullsHandler.CreateReview)
			repoRouter.Get("/{owner}/{repo}/pulls/{number}/comments", pullsHandler.ListComments)
			repoRouter.With(middleware.RequireAuth).Post("/{owner}/{repo}/pulls/{number}/comments", pullsHandler.CreateComment)
			repoRouter.With(middleware.RequireAuth).Delete("/{owner}/{repo}/pulls/{number}/comments/{comment_id}", pullsHandler.DeleteComment)
			repoRouter.Get("/{owner}/{repo}/compare/{spec}", pullsHandler.Compare)

			// CI/CD Actions & Pipelines (Phase 7)
			repoRouter.Get("/{owner}/{repo}/actions/runs", ciHandler.ListRuns)
			repoRouter.With(middleware.RequireAuth).Post("/{owner}/{repo}/actions/runs", ciHandler.TriggerRun)
			repoRouter.Get("/{owner}/{repo}/actions/runs/{run_id}", ciHandler.GetRun)
			repoRouter.With(middleware.RequireAuth).Post("/{owner}/{repo}/actions/runs/{run_id}/cancel", ciHandler.CancelRun)
			repoRouter.With(middleware.RequireAuth).Post("/{owner}/{repo}/actions/runs/{run_id}/rerun", ciHandler.Rerun)
			repoRouter.Get("/{owner}/{repo}/actions/jobs/{job_id}", ciHandler.GetJob)
			repoRouter.Get("/{owner}/{repo}/actions/jobs/{job_id}/logs", ciHandler.GetJobLogs)
			repoRouter.Get("/{owner}/{repo}/actions/workflows", ciHandler.ListWorkflows)

			// Webhooks & Automation (Phase 8)
			repoRouter.Get("/{owner}/{repo}/settings/hooks", webhooksHandler.ListWebhooks)
			repoRouter.With(middleware.RequireAuth).Post("/{owner}/{repo}/settings/hooks", webhooksHandler.CreateWebhook)
			repoRouter.Get("/{owner}/{repo}/settings/hooks/{hook_id}", webhooksHandler.GetWebhook)
			repoRouter.With(middleware.RequireAuth).Patch("/{owner}/{repo}/settings/hooks/{hook_id}", webhooksHandler.UpdateWebhook)
			repoRouter.With(middleware.RequireAuth).Delete("/{owner}/{repo}/settings/hooks/{hook_id}", webhooksHandler.DeleteWebhook)
			repoRouter.With(middleware.RequireAuth).Post("/{owner}/{repo}/settings/hooks/{hook_id}/tests", webhooksHandler.TestPing)
			repoRouter.Get("/{owner}/{repo}/settings/hooks/{hook_id}/deliveries", webhooksHandler.ListDeliveries)
			repoRouter.Get("/{owner}/{repo}/settings/hooks/{hook_id}/deliveries/{delivery_id}", webhooksHandler.GetDelivery)
			repoRouter.With(middleware.RequireAuth).Post("/{owner}/{repo}/settings/hooks/{hook_id}/deliveries/{delivery_id}/redeliver", webhooksHandler.Redeliver)
		})
	})

	return r
}
