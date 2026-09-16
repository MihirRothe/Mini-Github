package webhooks

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"forgehub/apps/api/internal/config"
	"forgehub/apps/api/internal/git"
	"forgehub/apps/api/internal/modules/auth"
	"forgehub/apps/api/internal/modules/orgs"
	"forgehub/apps/api/internal/modules/repos"
)

func TestSignPayloadAndVerify(t *testing.T) {
	secret := "k9$2mP#8z!forgehub-secret"
	payload := []byte(`{"action":"opened","issue":{"id":"123","title":"Bug found"}}`)

	sig := SignPayload(secret, payload)
	if sig == "" {
		t.Fatalf("expected non-empty signature")
	}

	if !VerifySignature(secret, payload, sig) {
		t.Errorf("expected signature verification to succeed")
	}

	// Tampered payload
	tampered := []byte(`{"action":"closed","issue":{"id":"123","title":"Bug found"}}`)
	if VerifySignature(secret, tampered, sig) {
		t.Errorf("expected tampered signature verification to fail")
	}

	// Wrong secret
	if VerifySignature("wrong-secret", payload, sig) {
		t.Errorf("expected wrong secret verification to fail")
	}
}

func setupWebhooksTest(t *testing.T) (Service, repos.Service, *auth.Service, func()) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "forgehub-webhooks-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	cfg := &config.Config{
		GitRootDir:       tempDir,
		GitDefaultBranch: "main",
		BaseURL:          "http://localhost:8080",
	}

	authRepo := auth.NewMemoryRepository()
	authSvc := auth.NewService(authRepo, cfg)
	orgsRepo := orgs.NewMemoryRepository(authRepo)
	repoStore := repos.NewMemoryRepositoryStore()

	gitStorage, err := git.NewStorage(tempDir)
	if err != nil {
		t.Fatalf("create git storage: %v", err)
	}
	gitReader := git.NewReader()

	reposSvc := repos.NewService(repoStore, gitStorage, gitReader, authRepo, orgsRepo, cfg)

	hookStore := NewMemoryRepositoryStore()
	dispatcher := NewDispatcher(hookStore)
	hookSvc := NewService(hookStore, dispatcher, reposSvc, authRepo)

	cleanup := func() {
		_ = os.RemoveAll(tempDir)
	}

	return hookSvc, reposSvc, authSvc, cleanup
}

func TestWebhookLifecycleAndPing(t *testing.T) {
	ctx := context.Background()
	hookSvc, reposSvc, authSvc, cleanup := setupWebhooksTest(t)
	defer cleanup()

	// 1. Create a mock destination server
	var mu sync.Mutex
	var receivedHeaders http.Header
	var receivedBody []byte

	destServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		receivedHeaders = r.Header.Clone()
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"received"}`))
	}))
	defer destServer.Close()

	// 2. Register user & create repo
	user, _, err := authSvc.Register(ctx, auth.RegisterRequest{
		Username: "webhook-owner",
		Email:    "owner@forgehub.local",
		Password: "Password123!",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("register user: %v", err)
	}

	repo, err := reposSvc.CreateRepository(ctx, user.ID, false, repos.CreateRepoRequest{
		Name:           "Webhook Target Repo",
		Slug:           "webhook-target-repo",
		Visibility:     "public",
		InitWithReadme: true,
	})
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}

	// 3. Create Webhook
	secret := "super-secure-token"
	hook, err := hookSvc.CreateWebhook(ctx, user.ID, false, "webhook-owner", "webhook-target-repo", CreateWebhookRequest{
		URL:         destServer.URL,
		ContentType: "application/json",
		Secret:      secret,
		Events:      []string{EventPush, EventIssues},
	})
	if err != nil {
		t.Fatalf("create webhook: %v", err)
	}

	if hook.URL != destServer.URL {
		t.Errorf("expected URL '%s', got '%s'", destServer.URL, hook.URL)
	}

	// 4. Test Ping Delivery
	del, err := hookSvc.TestPing(ctx, user.ID, false, "webhook-owner", "webhook-target-repo", hook.ID)
	if err != nil {
		t.Fatalf("test ping: %v", err)
	}

	if !del.IsSuccess {
		t.Errorf("expected successful ping delivery")
	}
	if del.ResponseStatusCode == nil || *del.ResponseStatusCode != 200 {
		t.Errorf("expected status code 200, got %v", del.ResponseStatusCode)
	}

	// Verify headers received by test server
	mu.Lock()
	eventHeader := receivedHeaders.Get("X-Forge-Event")
	sigHeader := receivedHeaders.Get("X-Forge-Signature-256")
	body := receivedBody
	mu.Unlock()

	if eventHeader != EventPing {
		t.Errorf("expected X-Forge-Event 'ping', got '%s'", eventHeader)
	}
	if !VerifySignature(secret, body, sigHeader) {
		t.Errorf("signature verification of received ping payload failed")
	}

	// 5. List Deliveries
	deliveries, err := hookSvc.ListDeliveries(ctx, user.ID, false, "webhook-owner", "webhook-target-repo", hook.ID, 10)
	if err != nil {
		t.Fatalf("list deliveries: %v", err)
	}
	if len(deliveries) < 1 {
		t.Fatalf("expected at least 1 delivery logged, got %d", len(deliveries))
	}

	// 6. Redeliver previous delivery
	redel, err := hookSvc.Redeliver(ctx, user.ID, false, "webhook-owner", "webhook-target-repo", hook.ID, deliveries[0].ID)
	if err != nil {
		t.Fatalf("redeliver: %v", err)
	}
	if !redel.IsSuccess {
		t.Errorf("expected redelivery success")
	}

	// 7. Test Broadcast Event
	hookSvc.BroadcastEvent(ctx, repo.ID, EventPush, "commit", map[string]string{
		"ref": "refs/heads/main",
		"sha": "1234567890abcdef",
	})

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	broadcastEvent := receivedHeaders.Get("X-Forge-Event")
	mu.Unlock()

	if broadcastEvent != EventPush {
		t.Errorf("expected broadcast X-Forge-Event 'push', got '%s'", broadcastEvent)
	}

	// 8. Delete Webhook
	err = hookSvc.DeleteWebhook(ctx, user.ID, false, "webhook-owner", "webhook-target-repo", hook.ID)
	if err != nil {
		t.Fatalf("delete webhook: %v", err)
	}

	// Ensure list is now empty
	hooks, err := hookSvc.ListWebhooks(ctx, user.ID, false, "webhook-owner", "webhook-target-repo")
	if err != nil {
		t.Fatalf("list webhooks after delete: %v", err)
	}
	if len(hooks) != 0 {
		t.Errorf("expected 0 webhooks after deletion, got %d", len(hooks))
	}
}
