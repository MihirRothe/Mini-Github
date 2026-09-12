package router

import (
	"net/http"
	"time"

	internalAuth "forgehub/apps/api/internal/auth"
	"forgehub/apps/api/internal/config"
	"forgehub/apps/api/internal/database"
	"forgehub/apps/api/internal/errors"
	"forgehub/apps/api/internal/middleware"
	authModule "forgehub/apps/api/internal/modules/auth"
	"forgehub/apps/api/internal/modules/health"
	"forgehub/apps/api/internal/modules/orgs"
	"forgehub/apps/api/internal/modules/tokens"
	"forgehub/apps/api/internal/modules/users"
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

	// Authentication Context Middleware
	r.Use(middleware.Authenticate(authSvc, opts.Config.SessionCookieName))

	// Auth Rate Limiter: 10 attempts per minute
	authRateLimiter := internalAuth.NewRateLimiter(10, 1*time.Minute)

	// Handlers
	authHandler := authModule.NewHandler(authSvc, opts.Config)
	userHandler := users.NewHandler(authSvc)
	tokenHandler := tokens.NewHandler(authSvc)
	orgsHandler := orgs.NewHandler(orgsSvc)
	healthHandler := health.NewHandler(opts.DB, opts.Redis)

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
	})

	return r
}
