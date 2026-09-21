package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"forgehub/apps/api/internal/modules/auth"

	"github.com/go-chi/chi/v5"
)

func TestAdminServiceStats(t *testing.T) {
	repo := NewMemoryRepository(nil)
	svc := NewService(repo)
	ctx := context.Background()

	adminUser := &auth.User{
		ID:       "admin-1",
		Username: "sysadmin",
		IsAdmin:  true,
	}
	regularUser := &auth.User{
		ID:       "user-1",
		Username: "regular",
		IsAdmin:  false,
	}

	// Regular user must be rejected
	_, err := svc.GetStats(ctx, regularUser)
	if err != ErrUnauthorizedAdmin {
		t.Fatalf("Expected ErrUnauthorizedAdmin, got %v", err)
	}

	// Admin user must receive stats
	stats, err := svc.GetStats(ctx, adminUser)
	if err != nil {
		t.Fatalf("Failed to get stats as admin: %v", err)
	}

	if stats.GoroutinesCount <= 0 {
		t.Errorf("Expected positive goroutines count, got %d", stats.GoroutinesCount)
	}
	if stats.AllocatedMemBytes <= 0 {
		t.Errorf("Expected non-zero allocated memory, got %d", stats.AllocatedMemBytes)
	}
	if stats.TotalUsers <= 0 {
		t.Errorf("Expected total users >= 0, got %d", stats.TotalUsers)
	}
}

func TestAdminAuditLogs(t *testing.T) {
	repo := NewMemoryRepository(nil)
	svc := NewService(repo)
	ctx := context.Background()

	adminUser := &auth.User{
		ID:       "admin-1",
		Username: "sysadmin",
		IsAdmin:  true,
	}

	// Log an action
	actorID := adminUser.ID
	err := svc.LogAction(ctx, &actorID, adminUser.Username, "repo.delete", "repository", "repo-123", "127.0.0.1", "test-agent", map[string]string{
		"repo_name": "test-repo",
	})
	if err != nil {
		t.Fatalf("Failed to log action: %v", err)
	}

	logs, total, err := svc.ListAuditLogs(ctx, adminUser, AuditLogFilter{Limit: 10})
	if err != nil {
		t.Fatalf("Failed to list audit logs: %v", err)
	}

	if total != 1 || len(logs) != 1 {
		t.Fatalf("Expected 1 audit log entry, got total=%d, len=%d", total, len(logs))
	}

	if logs[0].Action != "repo.delete" {
		t.Errorf("Expected action repo.delete, got %s", logs[0].Action)
	}
	if logs[0].ActorUsername != "sysadmin" {
		t.Errorf("Expected actor sysadmin, got %s", logs[0].ActorUsername)
	}
}

func TestAdminUserStatusUpdate(t *testing.T) {
	repo := NewMemoryRepository(nil)
	svc := NewService(repo)
	ctx := context.Background()

	adminUser := &auth.User{
		ID:       "admin-1",
		Username: "sysadmin",
		IsAdmin:  true,
	}

	// Self-demotion should fail
	demoteFalse := false
	_, err := svc.UpdateUserStatus(ctx, adminUser, adminUser.ID, UpdateUserAdminRequest{
		IsAdmin: &demoteFalse,
	}, "127.0.0.1", "test")
	if err != ErrCannotDemoteSelf {
		t.Fatalf("Expected ErrCannotDemoteSelf, got %v", err)
	}

	// Target user promotion should succeed
	promoteTrue := true
	targetUserID := "target-user-2"
	updated, err := svc.UpdateUserStatus(ctx, adminUser, targetUserID, UpdateUserAdminRequest{
		IsAdmin: &promoteTrue,
	}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("Failed to promote user: %v", err)
	}

	if !updated.IsAdmin {
		t.Errorf("Expected user to be promoted to admin")
	}

	// Check that audit log was created
	logs, _, _ := svc.ListAuditLogs(ctx, adminUser, AuditLogFilter{})
	if len(logs) == 0 || logs[0].Action != "admin.user_promoted" {
		t.Errorf("Expected audit log for user promotion, got %v", logs)
	}
}

func TestAdminHTTPHandlers(t *testing.T) {
	repo := NewMemoryRepository(nil)
	svc := NewService(repo)
	h := NewHandler(svc)

	adminUser := &auth.User{
		ID:       "admin-1",
		Username: "sysadmin",
		IsAdmin:  true,
	}
	nonAdminUser := &auth.User{
		ID:       "user-2",
		Username: "dev",
		IsAdmin:  false,
	}

	r := chi.NewRouter()
	r.Route("/api/v1/admin", func(adminRouter chi.Router) {
		adminRouter.Get("/stats", h.GetStats)
		adminRouter.Get("/audit-logs", h.ListAuditLogs)
		adminRouter.Get("/users", h.ListUsers)
		adminRouter.Patch("/users/{id}", h.UpdateUserStatus)
	})

	// 1. Non-admin access to /stats should return 403
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", nil)
	req = req.WithContext(auth.ContextWithUser(req.Context(), nonAdminUser))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden for non-admin, got %d", rec.Code)
	}

	// 2. Admin access to /stats should return 200 OK
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", nil)
	req = req.WithContext(auth.ContextWithUser(req.Context(), adminUser))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for admin stats, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. Admin access to /users should return 200 OK
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	req = req.WithContext(auth.ContextWithUser(req.Context(), adminUser))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for admin users, got %d: %s", rec.Code, rec.Body.String())
	}

	// 4. Admin update user status
	isSuspendedTrue := true
	updateBody, _ := json.Marshal(UpdateUserAdminRequest{
		IsSuspended: &isSuspendedTrue,
	})
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/target-123", bytes.NewReader(updateBody))
	req = req.WithContext(auth.ContextWithUser(req.Context(), adminUser))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for user status patch, got %d: %s", rec.Code, rec.Body.String())
	}
}
