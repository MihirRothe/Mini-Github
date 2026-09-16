package repos

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"forgehub/apps/api/internal/config"
	"forgehub/apps/api/internal/git"
	"forgehub/apps/api/internal/modules/auth"
	"forgehub/apps/api/internal/modules/orgs"

	"github.com/go-chi/chi/v5"
)

func setupTestGitServer(t *testing.T) (http.Handler, Service, *auth.Service, string, func()) {
	tempDir, err := os.MkdirTemp("", "forgehub-githttp-test-*")
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
	repoStore := NewMemoryRepositoryStore()

	gitStorage, err := git.NewStorage(tempDir)
	if err != nil {
		t.Fatalf("git storage: %v", err)
	}
	gitReader := git.NewReader()

	reposSvc := NewService(repoStore, gitStorage, gitReader, authRepo, orgsRepo, cfg)
	gitHTTPHandler := NewGitHTTPHandler(reposSvc, authSvc)

	r := chi.NewRouter()
	r.Get("/{owner}/{repo}.git/info/refs", gitHTTPHandler.InfoRefs)
	r.Post("/{owner}/{repo}.git/{service:(git-upload-pack|git-receive-pack)}", gitHTTPHandler.ServiceRPC)
	r.Get("/{owner}/{repo}/info/refs", gitHTTPHandler.InfoRefs)
	r.Post("/{owner}/{repo}/{service:(git-upload-pack|git-receive-pack)}", gitHTTPHandler.ServiceRPC)

	cleanup := func() {
		_ = os.RemoveAll(tempDir)
	}

	return r, reposSvc, authSvc, tempDir, cleanup
}

func TestGitSmartHTTP_InfoRefs_Public(t *testing.T) {
	ctx := context.Background()
	handler, reposSvc, authSvc, _, cleanup := setupTestGitServer(t)
	defer cleanup()

	// 1. Create a user
	user, _, err := authSvc.Register(ctx, auth.RegisterRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "Password123!",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// 2. Create public repo with readme
	_, err = reposSvc.CreateRepository(ctx, user.ID, false, CreateRepoRequest{
		Name:           "Hello World",
		Slug:           "hello-world",
		Visibility:     "public",
		InitWithReadme: true,
	})
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}

	// 3. GET /alice/hello-world.git/info/refs?service=git-upload-pack
	req := httptest.NewRequest(http.MethodGet, "/alice/hello-world.git/info/refs?service=git-upload-pack", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/x-git-upload-pack-advertisement" {
		t.Errorf("expected upload-pack-advertisement, got '%s'", contentType)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "# service=git-upload-pack") {
		t.Errorf("expected service=git-upload-pack header in body, got: %s", body)
	}
	if !strings.Contains(body, "refs/heads/main") {
		t.Errorf("expected refs/heads/main in ref advertisement, got: %s", body)
	}
}

func TestGitSmartHTTP_PrivateRepo_AuthRequired(t *testing.T) {
	ctx := context.Background()
	handler, reposSvc, authSvc, _, cleanup := setupTestGitServer(t)
	defer cleanup()

	user, _, err := authSvc.Register(ctx, auth.RegisterRequest{
		Username: "bob",
		Email:    "bob@example.com",
		Password: "Password123!",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	_, err = reposSvc.CreateRepository(ctx, user.ID, false, CreateRepoRequest{
		Name:           "Secret Code",
		Slug:           "secret-code",
		Visibility:     "private",
		InitWithReadme: true,
	})
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}

	// 1. Unauthenticated request to private repo should return 401
	req := httptest.NewRequest(http.MethodGet, "/bob/secret-code.git/info/refs?service=git-upload-pack", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rec.Code)
	}

	// 2. Request with invalid credentials should return 401
	req = httptest.NewRequest(http.MethodGet, "/bob/secret-code.git/info/refs?service=git-upload-pack", nil)
	req.SetBasicAuth("bob", "WrongPassword!")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for bad password, got %d", rec.Code)
	}

	// 3. Request with valid credentials should return 200
	req = httptest.NewRequest(http.MethodGet, "/bob/secret-code.git/info/refs?service=git-upload-pack", nil)
	req.SetBasicAuth("bob", "Password123!")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with valid credentials, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGitSmartHTTP_PersonalAccessTokenAuth(t *testing.T) {
	ctx := context.Background()
	handler, reposSvc, authSvc, _, cleanup := setupTestGitServer(t)
	defer cleanup()

	user, _, err := authSvc.Register(ctx, auth.RegisterRequest{
		Username: "carol",
		Email:    "carol@example.com",
		Password: "Password123!",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// Create a PAT token for carol
	tokenResp, err := authSvc.CreateAPIToken(ctx, user.ID, auth.CreateTokenRequest{
		Name:          "CI Token",
		Scopes:        []string{"repo"},
		ExpiresInDays: 30,
	})
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	_, err = reposSvc.CreateRepository(ctx, user.ID, false, CreateRepoRequest{
		Name:           "Private Project",
		Slug:           "private-project",
		Visibility:     "private",
		InitWithReadme: true,
	})
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}

	// Request with PAT in Basic Auth (username: carol, password: <token>)
	req := httptest.NewRequest(http.MethodGet, "/carol/private-project.git/info/refs?service=git-upload-pack", nil)
	req.SetBasicAuth("carol", tokenResp.Token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with PAT auth, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGitSmartHTTP_EndToEnd_CloneAndPush(t *testing.T) {
	ctx := context.Background()
	handler, reposSvc, authSvc, _, cleanup := setupTestGitServer(t)
	defer cleanup()

	// Spin up httptest server
	server := httptest.NewServer(handler)
	defer server.Close()

	// Create user
	user, _, err := authSvc.Register(ctx, auth.RegisterRequest{
		Username: "developer",
		Email:    "dev@example.com",
		Password: "SecretPassword123!",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// Create personal repo with readme
	_, err = reposSvc.CreateRepository(ctx, user.ID, false, CreateRepoRequest{
		Name:           "Project Alpha",
		Slug:           "project-alpha",
		Visibility:     "public",
		InitWithReadme: true,
	})
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}

	// Create a local client directory to perform git clone
	clientDir, err := os.MkdirTemp("", "forgehub-gitclient-*")
	if err != nil {
		t.Fatalf("client temp dir: %v", err)
	}
	defer os.RemoveAll(clientDir)

	// Clone URL with basic auth embedded or anonymous
	cloneURL := fmt.Sprintf("%s/developer/project-alpha.git", server.URL)

	// Step 1: Run `git clone <cloneURL>`
	cloneCmd := exec.Command("git", "clone", cloneURL, clientDir)
	cloneOut, err := cloneCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git clone failed: %v\nOutput:\n%s", err, string(cloneOut))
	}

	// Verify README exists in cloned repo
	readmePath := filepath.Join(clientDir, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		t.Fatalf("README.md not found in cloned repo")
	}

	// Step 2: Make a change, commit, and push
	newFilePath := filepath.Join(clientDir, "feature.txt")
	if err := os.WriteFile(newFilePath, []byte("New cool feature!"), 0644); err != nil {
		t.Fatalf("write feature.txt: %v", err)
	}

	// git add feature.txt
	addCmd := exec.Command("git", "-C", clientDir, "add", "feature.txt")
	if out, err := addCmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, string(out))
	}

	// git commit -m "Add feature.txt"
	commitCmd := exec.Command("git", "-C", clientDir, "commit", "-m", "Add feature.txt")
	commitCmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Developer",
		"GIT_AUTHOR_EMAIL=dev@example.com",
		"GIT_COMMITTER_NAME=Developer",
		"GIT_COMMITTER_EMAIL=dev@example.com",
	)
	if out, err := commitCmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, string(out))
	}

	// Step 3: git push with Basic Auth in remote URL
	pushURL := strings.Replace(server.URL, "http://", "http://developer:SecretPassword123!@", 1) + "/developer/project-alpha.git"
	setRemoteCmd := exec.Command("git", "-C", clientDir, "remote", "set-url", "origin", pushURL)
	if out, err := setRemoteCmd.CombinedOutput(); err != nil {
		t.Fatalf("remote set-url: %v\n%s", err, string(out))
	}

	pushCmd := exec.Command("git", "-C", clientDir, "push", "origin", "main")
	pushOut, err := pushCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git push failed: %v\nOutput:\n%s", err, string(pushOut))
	}

	// Step 4: Verify repository now has 2 commits on main branch via reposSvc
	commits, err := reposSvc.GetCommits(ctx, user.ID, false, "developer", "project-alpha", "main", 10)
	if err != nil {
		t.Fatalf("get commits: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("expected 2 commits after push, got %d", len(commits))
	}
	if commits[0].Message != "Add feature.txt" {
		t.Errorf("expected top commit message 'Add feature.txt', got '%s'", commits[0].Message)
	}

	// Step 5: Verify feature.txt is now in the Git tree
	tree, err := reposSvc.GetTree(ctx, user.ID, false, "developer", "project-alpha", "main", "")
	if err != nil {
		t.Fatalf("get tree: %v", err)
	}
	foundFeature := false
	for _, entry := range tree {
		if entry.Name == "feature.txt" {
			foundFeature = true
			break
		}
	}
	if !foundFeature {
		t.Errorf("expected feature.txt in tree, got %+v", tree)
	}
}
