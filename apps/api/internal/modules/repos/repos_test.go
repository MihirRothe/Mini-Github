package repos

import (
	"context"
	"os"
	"testing"

	"forgehub/apps/api/internal/config"
	"forgehub/apps/api/internal/git"
	"forgehub/apps/api/internal/modules/auth"
	"forgehub/apps/api/internal/modules/orgs"
)

func TestRepositoryLifecycleAndGit(t *testing.T) {
	ctx := context.Background()

	tempDir, err := os.MkdirTemp("", "forgehub-repos-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		GitRootDir:       tempDir,
		GitDefaultBranch: "main",
		BaseURL:          "http://localhost:8080",
	}

	authRepo := auth.NewMemoryRepository()
	orgsRepo := orgs.NewMemoryRepository(authRepo)
	repoStore := NewMemoryRepositoryStore()

	gitStorage, err := git.NewStorage(tempDir)
	if err != nil {
		t.Fatalf("git storage: %v", err)
	}
	gitReader := git.NewReader()

	user := &auth.User{Username: "johndoe", Email: "john@example.com", DisplayName: "John Doe"}
	_ = authRepo.CreateUser(ctx, user)

	svc := NewService(repoStore, gitStorage, gitReader, authRepo, orgsRepo, cfg)

	// 1. Create personal repository with README
	repo, err := svc.CreateRepository(ctx, user.ID, false, CreateRepoRequest{
		Name:           "Awesome App",
		Slug:           "awesome-app",
		Description:    "A test app on ForgeHub",
		Visibility:     "public",
		InitWithReadme: true,
	})
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}

	if repo.Slug != "awesome-app" {
		t.Errorf("expected slug 'awesome-app', got '%s'", repo.Slug)
	}
	if repo.OwnerName != "johndoe" {
		t.Errorf("expected owner 'johndoe', got '%s'", repo.OwnerName)
	}

	// 2. Verify branches
	branches, err := svc.GetBranches(ctx, user.ID, false, "johndoe", "awesome-app")
	if err != nil {
		t.Fatalf("get branches: %v", err)
	}
	if len(branches) == 0 || branches[0].Name != "main" {
		t.Errorf("expected main branch, got %+v", branches)
	}

	// 3. Verify commits
	commits, err := svc.GetCommits(ctx, user.ID, false, "johndoe", "awesome-app", "main", 10)
	if err != nil {
		t.Fatalf("get commits: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected 1 initial commit, got %d", len(commits))
	}

	// 4. Verify tree
	tree, err := svc.GetTree(ctx, user.ID, false, "johndoe", "awesome-app", "main", "")
	if err != nil {
		t.Fatalf("get tree: %v", err)
	}
	if len(tree) != 1 || tree[0].Name != "README.md" {
		t.Errorf("expected README.md in tree, got %+v", tree)
	}

	// 5. Verify Readme blob
	readme, err := svc.GetReadme(ctx, user.ID, false, "johndoe", "awesome-app", "main")
	if err != nil {
		t.Fatalf("get readme: %v", err)
	}
	if readme == nil || readme.Content == "" {
		t.Errorf("expected non-empty readme content")
	}

	// 6. Delete repository
	err = svc.DeleteRepository(ctx, user.ID, false, "johndoe", "awesome-app")
	if err != nil {
		t.Fatalf("delete repo: %v", err)
	}

	// 7. Verify repo no longer exists
	_, err = svc.GetRepository(ctx, user.ID, false, "johndoe", "awesome-app")
	if err == nil {
		t.Errorf("expected error after repository deletion")
	}
}
