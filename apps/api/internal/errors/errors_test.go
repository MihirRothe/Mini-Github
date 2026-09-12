package errors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStandardErrors(t *testing.T) {
	err := NotFound("Repository not found")
	if err.HTTPStatus != http.StatusNotFound {
		t.Errorf("expected 404, got %d", err.HTTPStatus)
	}
	if err.Code != ErrNotFound {
		t.Errorf("expected NOT_FOUND, got %s", err.Code)
	}

	rec := httptest.NewRecorder()
	RespondWithError(rec, err)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}

	var res ErrorResponse
	if decodeErr := json.NewDecoder(rec.Body).Decode(&res); decodeErr != nil {
		t.Fatalf("failed to decode error response: %v", decodeErr)
	}

	if res.Error.Code != ErrNotFound {
		t.Errorf("expected NOT_FOUND in response, got %s", res.Error.Code)
	}
	if res.Error.Message != "Repository not found" {
		t.Errorf("expected 'Repository not found', got %s", res.Error.Message)
	}
}
