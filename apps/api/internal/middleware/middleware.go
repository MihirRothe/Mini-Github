package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"forgehub/apps/api/internal/errors"
	"forgehub/apps/api/internal/logger"
	authModule "forgehub/apps/api/internal/modules/auth"

	"github.com/google/uuid"
)

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (rw *responseWriterWrapper) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriterWrapper) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
		}

		w.Header().Set("X-Request-ID", reqID)
		ctx := logger.WithRequestID(r.Context(), reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func StructuredLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapper := &responseWriterWrapper{ResponseWriter: w}

		next.ServeHTTP(wrapper, r)

		latency := time.Since(start)
		log := logger.FromContext(r.Context())

		// Do not log noisy healthcheck hits at Info level unless errored
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" || r.URL.Path == "/livez" {
			if wrapper.statusCode >= 400 {
				log.Warn("Health check failed",
					"path", r.URL.Path,
					"status", wrapper.statusCode,
					"latency_ms", latency.Milliseconds(),
				)
			}
			return
		}

		log.Info("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapper.statusCode,
			"latency_ms", latency.Milliseconds(),
			"bytes", wrapper.bytesWritten,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)
	})
}

func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log := logger.FromContext(r.Context())
				log.Error("Panic recovered in HTTP handler",
					"error", fmt.Sprintf("%v", rec),
					"stack", string(debug.Stack()),
				)

				errors.RespondWithError(w, errors.Internal("An unexpected internal server error occurred"))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		next.ServeHTTP(w, r)
	})
}

// Authenticate extracts session cookie or Bearer token and associates user with context
func Authenticate(authSvc *authModule.Service, cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// 1. Check Session Cookie
			if cookie, err := r.Cookie(cookieName); err == nil && cookie.Value != "" {
				if user, err := authSvc.ValidateSession(ctx, cookie.Value); err == nil && user != nil {
					ctx = authModule.ContextWithUser(ctx, user)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			// 2. Check Authorization: Bearer <token> (PAT)
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				rawToken := strings.TrimPrefix(authHeader, "Bearer ")
				if strings.HasPrefix(rawToken, "fh_pat_") {
					if user, _, err := authSvc.ValidateAPIToken(ctx, rawToken); err == nil && user != nil {
						ctx = authModule.ContextWithUser(ctx, user)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				} else {
					// Also support raw session token via Bearer
					if user, err := authSvc.ValidateSession(ctx, rawToken); err == nil && user != nil {
						ctx = authModule.ContextWithUser(ctx, user)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth blocks unauthenticated requests with 401 Unauthorized
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := authModule.GetUserFromContext(r.Context())
		if user == nil {
			errors.RespondWithError(w, errors.Unauthorized("Authentication is required to access this resource"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAdmin blocks non-administrator accounts with 403 Forbidden
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := authModule.GetUserFromContext(r.Context())
		if user == nil {
			errors.RespondWithError(w, errors.Unauthorized("Authentication is required to access this resource"))
			return
		}
		if !user.IsAdmin {
			errors.RespondWithError(w, errors.Forbidden("Administrative privileges are required for this action"))
			return
		}
		next.ServeHTTP(w, r)
	})
}
