package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthCheck(t *testing.T) {
	h := NewHandler(nil, nil)

	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()

	h.HealthCheck(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var res TelemetryResponse
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.Status != "healthy" {
		t.Errorf("expected status 'healthy', got '%s'", res.Status)
	}
	if res.Version == "" {
		t.Errorf("expected non-empty version")
	}
	if res.System.GoVersion == "" {
		t.Errorf("expected non-empty go_version")
	}
}

func TestLivenessReadiness(t *testing.T) {
	h := NewHandler(nil, nil)

	// Test Liveness
	reqLive := httptest.NewRequest("GET", "/livez", nil)
	recLive := httptest.NewRecorder()
	h.Liveness(recLive, reqLive)
	if recLive.Code != http.StatusOK {
		t.Errorf("expected liveness 200, got %d", recLive.Code)
	}

	// Test Readiness
	reqReady := httptest.NewRequest("GET", "/readyz", nil)
	recReady := httptest.NewRecorder()
	h.Readiness(recReady, reqReady)
	if recReady.Code != http.StatusOK {
		t.Errorf("expected readiness 200, got %d", recReady.Code)
	}
}
