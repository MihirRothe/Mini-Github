package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"forgehub/apps/api/internal/errors"
	"forgehub/apps/api/internal/logger"

	"github.com/google/uuid"
)

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
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
