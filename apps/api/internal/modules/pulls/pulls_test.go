package pulls

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"forgehub/apps/api/internal/config"
	"forgehub/apps/api/internal/git"
	"forgehub/apps/api/internal/modules/auth"
	"forgehub/apps/api/internal/modules/orgs"
	"forgehub/apps/api/internal/modules/repos"
)

func setupPullsTest(t *testing.T) (Service, repos.Service, *auth.Service, git.Storage, string, func()) {
	t.Helper()
	ctx := context.Background()
	_ = ctx

	tempDir, err := os.MkdirTemp("", "forgehub-pulls-test-*")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
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
		t.Fatalf("git storage: %v", err)
	}
	gitReader := git.NewReader()

	reposSvc := repos.NewService(repoStore, gitStorage, gitReader, authRepo, orgsRepo, cfg)

	pullsStore := NewMemoryRepositoryStore(authRepo)
	pullsSvc := NewService(pullsStore, reposSvc, gitStorage, gitReader, authRepo)

	cleanup := func() {
		_ = os.RemoveAll(tempDir)
	}

	return pullsSvc, reposSvc, authSvc, gitStorage, tempDir, cleanup
}

func createCommitOnBranch(t *testing.T, diskPath, branch, filename, content, msg string) string {
	t.Helper()

	// 1. Hash object
	cmd := exec.Command("git", "hash-object", "-w", "--stdin")
	cmd.Dir = diskPath
	cmd.Stdin = strings.NewReader(content)
	blobOut, err := cmd.Output()
	if err != nil {
		t.Fatalf("hash-object: %v", err)
	}
	blobSHA := strings.TrimSpace(string(blobOut))

	// 2. Read existing tree or create new tree
	cmd = exec.Command("git", "rev-parse", "refs/heads/"+branch+"^{tree}")
	cmd.Dir = diskPath
	baseTreeOut, _ := cmd.Output()
	baseTreeSHA := strings.TrimSpace(string(baseTreeOut))

	var mktreeInput string
	if baseTreeSHA != "" {
		// List existing tree items
		cmd = exec.Command("git", "ls-tree", baseTreeSHA)
		cmd.Dir = diskPath
		existingTree, _ := cmd.Output()
		mktreeInput = string(existingTree)
	}
	mktreeInput += fmt.Sprintf("100644 blob %s\t%s\n", blobSHA, filename)

	cmd = exec.Command("git", "mktree")
	cmd.Dir = diskPath
	cmd.Stdin = strings.NewReader(mktreeInput)
	treeOut, err := cmd.Output()
	if err != nil {
		t.Fatalf("mktree: %v", err)
	}
	treeSHA := strings.TrimSpace(string(treeOut))

	// 3. Parent commit
	cmd = exec.Command("git", "rev-parse", "refs/heads/"+branch)
	cmd.Dir = diskPath
	parentOut, _ := cmd.Output()
	parentSHA := strings.TrimSpace(string(parentOut))

	args := []string{"commit-tree", treeSHA, "-m", msg}
	if parentSHA != "" {
		args = append(args, "-p", parentSHA)
	}
	cmd = exec.Command("git", args...)
	cmd.Dir = diskPath
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test Committer",
		"GIT_AUTHOR_EMAIL=test@forgehub.local",
		"GIT_COMMITTER_NAME=Test Committer",
		"GIT_COMMITTER_EMAIL=test@forgehub.local",
	)
	commitOut, err := cmd.Output()
	if err != nil {
		t.Fatalf("commit-tree: %v", err)
	}
	commitSHA := strings.TrimSpace(string(commitOut))

	// 4. Update ref
	cmd = exec.Command("git", "update-ref", "refs/heads/"+branch, commitSHA)
	cmd.Dir = diskPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("update-ref: %v", err)
	}

	return commitSHA
}

func TestPullRequestFullWorkflow(t *testing.T) {
	ctx := context.Background()
	pullsSvc, reposSvc, authSvc, gitStorage, _, cleanup := setupPullsTest(t)
	defer cleanup()

	// 1. Create Alice and Bob
	alice, _, err := authSvc.Register(ctx, auth.RegisterRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "Password123!",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("register alice: %v", err)
	}

	bob, _, err := authSvc.Register(ctx, auth.RegisterRequest{
		Username: "bob",
		Email:    "bob@example.com",
		Password: "Password123!",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("register bob: %v", err)
	}

	// 2. Create Repository initialized with README
	_, err = reposSvc.CreateRepository(ctx, alice.ID, false, repos.CreateRepoRequest{
		Name:           "Calculator App",
		Slug:           "calculator-app",
		Description:    "Math calculations repository",
		Visibility:     "public",
		InitWithReadme: true,
	})
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}

	diskPath, err := gitStorage.GetDiskPath("users", "alice", "calculator-app")
	if err != nil {
		t.Fatalf("get disk path: %v", err)
	}

	// 3. Create feature branch "feature/add-math" pointing to main
	cmd := exec.Command("git", "branch", "feature/add-math", "main")
	cmd.Dir = diskPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("create branch feature/add-math: %v", err)
	}

	// 4. Commit a new file calc.go on feature branch
	calcContent := "package main\n\nfunc Add(a, b int) int {\n\treturn a + b\n}\n"
	commitSHA := createCommitOnBranch(t, diskPath, "feature/add-math", "calc.go", calcContent, "feat: add calculator function")
	if commitSHA == "" {
		t.Fatalf("expected valid commit SHA")
	}

	// 5. Compare branches main...feature/add-math
	diff, commits, canMerge, err := pullsSvc.CompareBranches(ctx, alice.ID, false, "alice", "calculator-app", "main", "feature/add-math")
	if err != nil {
		t.Fatalf("compare branches: %v", err)
	}
	if !canMerge {
		t.Errorf("expected branches to be cleanly mergeable")
	}
	if len(commits) != 1 {
		t.Errorf("expected 1 commit between main and feature, got %d", len(commits))
	}
	if len(diff.Files) != 1 || diff.Files[0].NewPath != "calc.go" {
		t.Errorf("expected diff to contain calc.go, got %+v", diff.Files)
	}

	// 6. Create Pull Request
	pr, err := pullsSvc.CreatePullRequest(ctx, alice.ID, false, "alice", "calculator-app", CreatePRRequest{
		Title:        "Implement basic calculator",
		Body:         "Adds addition support in calc.go",
		SourceBranch: "feature/add-math",
		TargetBranch: "main",
	})
	if err != nil {
		t.Fatalf("create PR: %v", err)
	}

	if pr.Number != 1 {
		t.Errorf("expected PR number 1, got %d", pr.Number)
	}
	if pr.State != StateOpen {
		t.Errorf("expected PR state 'open', got '%s'", pr.State)
	}

	// 7. Bob submits a review approving the pull request
	review, err := pullsSvc.CreateReview(ctx, bob.ID, false, "alice", "calculator-app", pr.Number, CreateReviewRequest{
		State: ReviewApproved,
		Body:  "LGTM! Clean implementation.",
	})
	if err != nil {
		t.Fatalf("create review: %v", err)
	}
	if review.State != ReviewApproved {
		t.Errorf("expected review state APPROVED, got %s", review.State)
	}

	// 8. Bob adds a comment
	comment, err := pullsSvc.CreateComment(ctx, bob.ID, false, "alice", "calculator-app", pr.Number, CreatePRCommentRequest{
		Body: "Tested with unit tests, works like a charm.",
	})
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}
	if comment.Body == "" {
		t.Errorf("expected non-empty comment body")
	}

	// 9. Fetch PR details
	detail, err := pullsSvc.GetPullRequest(ctx, alice.ID, false, "alice", "calculator-app", pr.Number)
	if err != nil {
		t.Fatalf("get PR detail: %v", err)
	}
	if len(detail.Reviews) != 1 || detail.Reviews[0].State != ReviewApproved {
		t.Errorf("expected 1 approved review in detail, got %+v", detail.Reviews)
	}
	if len(detail.Comments) != 1 {
		t.Errorf("expected 1 comment in detail, got %+v", detail.Comments)
	}

	// 10. Merge Pull Request (Alice merges using standard 3-way merge commit)
	mergedPR, err := pullsSvc.MergePullRequest(ctx, alice.ID, false, "alice", "calculator-app", pr.Number, MergePRRequest{
		Method:        "merge",
		CommitMessage: "Merge pull request #1 from feature/add-math",
	})
	if err != nil {
		t.Fatalf("merge PR: %v", err)
	}

	if mergedPR.State != StateMerged {
		t.Errorf("expected merged state, got %s", mergedPR.State)
	}
	if mergedPR.MergedAt == nil {
		t.Errorf("expected non-nil MergedAt")
	}
	if mergedPR.MergeCommitSHA == nil || *mergedPR.MergeCommitSHA == "" {
		t.Errorf("expected merge commit SHA to be populated")
	}

	// 11. Verify main branch tree now contains calc.go
	tree, err := reposSvc.GetTree(ctx, alice.ID, false, "alice", "calculator-app", "main", "")
	if err != nil {
		t.Fatalf("get tree of main: %v", err)
	}

	foundCalc := false
	for _, item := range tree {
		if item.Name == "calc.go" {
			foundCalc = true
			break
		}
	}
	if !foundCalc {
		t.Fatalf("expected calc.go to be in main branch tree after merge, got %+v", tree)
	}
}

func TestPullRequestSquashMerge(t *testing.T) {
	ctx := context.Background()
	pullsSvc, reposSvc, authSvc, gitStorage, _, cleanup := setupPullsTest(t)
	defer cleanup()

	// 1. Create User
	user, _, err := authSvc.Register(ctx, auth.RegisterRequest{
		Username: "carol",
		Email:    "carol@example.com",
		Password: "Password123!",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("register user: %v", err)
	}

	// 2. Create Repository
	_, err = reposSvc.CreateRepository(ctx, user.ID, false, repos.CreateRepoRequest{
		Name:           "Squash Demo",
		Slug:           "squash-demo",
		Visibility:     "public",
		InitWithReadme: true,
	})
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}

	diskPath, _ := gitStorage.GetDiskPath("users", "carol", "squash-demo")

	// 3. Create branch
	cmd := exec.Command("git", "branch", "patch-1", "main")
	cmd.Dir = diskPath
	_ = cmd.Run()

	createCommitOnBranch(t, diskPath, "patch-1", "file1.txt", "line1", "commit 1")
	createCommitOnBranch(t, diskPath, "patch-1", "file2.txt", "line2", "commit 2")

	// 4. Create PR
	pr, err := pullsSvc.CreatePullRequest(ctx, user.ID, false, "carol", "squash-demo", CreatePRRequest{
		Title:        "Two commits to squash",
		SourceBranch: "patch-1",
		TargetBranch: "main",
	})
	if err != nil {
		t.Fatalf("create PR: %v", err)
	}

	// 5. Squash merge
	mergedPR, err := pullsSvc.MergePullRequest(ctx, user.ID, false, "carol", "squash-demo", pr.Number, MergePRRequest{
		Method:        "squash",
		CommitMessage: "Squash merge patch-1 into main (#1)",
	})
	if err != nil {
		t.Fatalf("squash merge: %v", err)
	}

	if mergedPR.State != StateMerged {
		t.Errorf("expected merged state, got %s", mergedPR.State)
	}

	// 6. Verify main commits: should have 2 commits total (Initial + 1 squash commit)
	commits, err := reposSvc.GetCommits(ctx, user.ID, false, "carol", "squash-demo", "main", 10)
	if err != nil {
		t.Fatalf("get commits: %v", err)
	}
	if len(commits) != 2 {
		t.Errorf("expected 2 commits on main after squash merge, got %d", len(commits))
	}
}
