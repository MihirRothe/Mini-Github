package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	internalAuth "forgehub/apps/api/internal/auth"
	"forgehub/apps/api/internal/config"
)

func setupTestHandler() (*Handler, *Service, Repository) {
	repo := NewMemoryRepository()
	cfg := &config.Config{
		SessionCookieName:   "forgehub_session",
		SessionCookieSecure: false,
		SessionTTL:          24 * time.Hour,
	}
	svc := NewService(repo, cfg)
	// Fast params for unit tests
	svc.params = &internalAuth.HashParams{
		Memory:      16 * 1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}
	handler := NewHandler(svc, cfg)
	return handler, svc, repo
}

func TestRegistrationAndLoginFlow(t *testing.T) {
	handler, _, _ := setupTestHandler()

	// 1. Successful Registration
	regBody, _ := json.Marshal(RegisterRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "SuperSecretPassword123!",
	})
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(regBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.Register(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected registration status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "forgehub_session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil || sessionCookie.Value == "" {
		t.Fatalf("expected session cookie to be set")
	}

	// 2. Reject Duplicate Registration
	recDup := httptest.NewRecorder()
	reqDup := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(regBody))
	reqDup.Header.Set("Content-Type", "application/json")
	handler.Register(recDup, reqDup)
	if recDup.Code != http.StatusConflict {
		t.Errorf("expected duplicate registration 409, got %d", recDup.Code)
	}

	// 3. Reject Weak Password
	weakBody, _ := json.Marshal(RegisterRequest{
		Username: "bob",
		Email:    "bob@example.com",
		Password: "weak",
	})
	recWeak := httptest.NewRecorder()
	reqWeak := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(weakBody))
	handler.Register(recWeak, reqWeak)
	if recWeak.Code != http.StatusBadRequest {
		t.Errorf("expected weak password 400, got %d", recWeak.Code)
	}

	// 4. Successful Login
	loginBody, _ := json.Marshal(LoginRequest{
		Login:    "alice",
		Password: "SuperSecretPassword123!",
	})
	recLogin := httptest.NewRecorder()
	reqLogin := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginBody))
	handler.Login(recLogin, reqLogin)
	if recLogin.Code != http.StatusOK {
		t.Fatalf("expected login 200, got %d: %s", recLogin.Code, recLogin.Body.String())
	}

	// 5. Reject Invalid Password
	badLoginBody, _ := json.Marshal(LoginRequest{
		Login:    "alice",
		Password: "IncorrectPassword123!",
	})
	recBadLogin := httptest.NewRecorder()
	reqBadLogin := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(badLoginBody))
	handler.Login(recBadLogin, reqBadLogin)
	if recBadLogin.Code != http.StatusUnauthorized {
		t.Errorf("expected bad credentials 401, got %d", recBadLogin.Code)
	}

	// 6. Logout
	recLogout := httptest.NewRecorder()
	reqLogout := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	reqLogout.AddCookie(sessionCookie)
	handler.Logout(recLogout, reqLogout)
	if recLogout.Code != http.StatusOK {
		t.Errorf("expected logout 200, got %d", recLogout.Code)
	}
}
