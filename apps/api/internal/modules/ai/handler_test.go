package ai

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"forgehub/apps/api/internal/config"
	"forgehub/apps/api/internal/modules/ai/providers"

	"github.com/go-chi/chi/v5"
)

func setupTestRouter() *chi.Mux {
	cfg := &config.Config{AIProvider: "local"}
	mgr := providers.NewManager(cfg)
	svc := NewService(mgr, nil, nil)
	h := NewHandler(svc)

	r := chi.NewRouter()
	r.Route("/api/v1/ai", func(aiRouter chi.Router) {
		aiRouter.Post("/explain", h.Explain)
		aiRouter.Post("/review", h.Review)
		aiRouter.Post("/generate-tests", h.GenerateTests)
		aiRouter.Post("/chat", h.Chat)
		aiRouter.Get("/providers", h.Providers)
	})
	return r
}

func TestHandlerExplain(t *testing.T) {
	r := setupTestRouter()

	body, _ := json.Marshal(ExplainRequest{
		Code: "package main\n\nfunc Hello() string { return \"world\" }",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/explain", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var res map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := res["explanation"]; !ok {
		t.Errorf("Response missing 'explanation' key")
	}
}

func TestHandlerReview(t *testing.T) {
	r := setupTestRouter()

	diff := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,3 @@
 func main() {
+	println("ok")
 }
`
	body, _ := json.Marshal(ReviewRequest{
		Diff: diff,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/review", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var res map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := res["review"]; !ok {
		t.Errorf("Response missing 'review' key")
	}
}

func TestHandlerGenerateTests(t *testing.T) {
	r := setupTestRouter()

	body, _ := json.Marshal(GenerateTestsRequest{
		Code: "func Multiply(a, b int) int { return a * b }",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/generate-tests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var res map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := res["test_suite"]; !ok {
		t.Errorf("Response missing 'test_suite' key")
	}
}

func TestHandlerChat(t *testing.T) {
	r := setupTestRouter()

	body, _ := json.Marshal(ChatRequest{
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello ForgeAI"},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerProviders(t *testing.T) {
	r := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ai/providers", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var res ProvidersResponse
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("Failed to decode providers: %v", err)
	}

	if res.ActiveProvider != "local" {
		t.Errorf("Expected active provider 'local', got %s", res.ActiveProvider)
	}
}
