package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"forgehub/apps/api/internal/config"
	apiErrors "forgehub/apps/api/internal/errors"
)

type Handler struct {
	svc *Service
	cfg *config.Config
}

func NewHandler(svc *Service, cfg *config.Config) *Handler {
	return &Handler{
		svc: svc,
		cfg: cfg,
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON request body"))
		return
	}

	ip, ua := r.RemoteAddr, r.UserAgent()
	user, token, err := h.svc.Register(r.Context(), req, ip, ua)
	if err != nil {
		if errors.Is(err, ErrInvalidUsername) || errors.Is(err, ErrInvalidEmail) || errors.Is(err, ErrWeakPassword) {
			apiErrors.RespondWithError(w, apiErrors.New(http.StatusBadRequest, apiErrors.ErrValidation, err.Error()))
			return
		}
		if errors.Is(err, ErrDuplicateUser) {
			apiErrors.RespondWithError(w, apiErrors.New(http.StatusConflict, apiErrors.ErrConflict, "A user with this username or email already exists"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("Failed to register user"))
		return
	}

	h.setSessionCookie(w, token, h.cfg.SessionTTL)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"user":    user,
		"message": "User registered successfully",
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErrors.RespondWithError(w, apiErrors.BadRequest("Invalid JSON request body"))
		return
	}

	ip, ua := r.RemoteAddr, r.UserAgent()
	user, token, err := h.svc.Login(r.Context(), req, ip, ua)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			apiErrors.RespondWithError(w, apiErrors.New(http.StatusUnauthorized, apiErrors.ErrUnauthorized, "Invalid username/email or password"))
			return
		}
		if errors.Is(err, ErrAccountSuspended) {
			apiErrors.RespondWithError(w, apiErrors.New(http.StatusForbidden, apiErrors.ErrForbidden, "Your account has been suspended"))
			return
		}
		apiErrors.RespondWithError(w, apiErrors.Internal("An error occurred during authentication"))
		return
	}

	h.setSessionCookie(w, token, h.cfg.SessionTTL)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"user":    user,
		"message": "Logged in successfully",
	})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(h.cfg.SessionCookieName)
	if err == nil && cookie.Value != "" {
		_ = h.svc.Logout(r.Context(), cookie.Value, r.RemoteAddr, r.UserAgent())
	}

	h.clearSessionCookie(w)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "Logged out successfully",
	})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		apiErrors.RespondWithError(w, apiErrors.Unauthorized("You are not authenticated"))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"user": user,
	})
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, token string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.SessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(ttl),
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   h.cfg.SessionCookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.SessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cfg.SessionCookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}
