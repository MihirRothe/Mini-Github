package issues

import (
	"context"
	"os"
	"testing"

	"forgehub/apps/api/internal/config"
	"forgehub/apps/api/internal/git"
	"forgehub/apps/api/internal/modules/auth"
	"forgehub/apps/api/internal/modules/orgs"
	"forgehub/apps/api/internal/modules/repos"
)

func setupIssuesTest(t *testing.T) (Service, repos.Service, *auth.Service, func()) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "forgehub-issues-test-*")
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

	issuesStore := NewMemoryRepositoryStore(authRepo)
	issuesSvc := NewService(issuesStore, reposSvc, authRepo)

	_ = ctx
	cleanup := func() {
		_ = os.RemoveAll(tempDir)
	}

	return issuesSvc, reposSvc, authSvc, cleanup
}

func TestIssuesWorkflow(t *testing.T) {
	ctx := context.Background()
	issuesSvc, reposSvc, authSvc, cleanup := setupIssuesTest(t)
	defer cleanup()

	// 1. Create Users
	user1, _, err := authSvc.Register(ctx, auth.RegisterRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "Password123!",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("register user1: %v", err)
	}

	user2, _, err := authSvc.Register(ctx, auth.RegisterRequest{
		Username: "bob",
		Email:    "bob@example.com",
		Password: "Password123!",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("register user2: %v", err)
	}

	// 2. Create Public Repository for alice
	repo, err := reposSvc.CreateRepository(ctx, user1.ID, false, repos.CreateRepoRequest{
		Name:           "Alpha Project",
		Slug:           "alpha-project",
		Visibility:     "public",
		InitWithReadme: true,
	})
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}

	// 3. Create Labels
	labelBug, err := issuesSvc.CreateLabel(ctx, user1.ID, false, "alice", "alpha-project", CreateLabelRequest{
		Name:        "bug",
		Color:       "#d73a4a",
		Description: "Something isn't working",
	})
	if err != nil {
		t.Fatalf("create label bug: %v", err)
	}

	labelFeat, err := issuesSvc.CreateLabel(ctx, user1.ID, false, "alice", "alpha-project", CreateLabelRequest{
		Name:        "feature",
		Color:       "#a2eeef",
		Description: "New capability",
	})
	if err != nil {
		t.Fatalf("create label feature: %v", err)
	}

	labels, err := issuesSvc.ListLabels(ctx, user1.ID, false, "alice", "alpha-project")
	if err != nil || len(labels) < 2 {
		t.Fatalf("list labels error or empty: %v", err)
	}

	// 4. Create Milestone
	milestone, err := issuesSvc.CreateMilestone(ctx, user1.ID, false, "alice", "alpha-project", CreateMilestoneRequest{
		Title:       "v1.0 Launch",
		Description: "First stable release",
	})
	if err != nil {
		t.Fatalf("create milestone: %v", err)
	}

	// 5. Create Issue #1 by bob (authenticated user on public repo)
	iss1, err := issuesSvc.CreateIssue(ctx, user2.ID, false, "alice", "alpha-project", CreateIssueRequest{
		Title: "Crash when clicking header button",
		Body:  "Steps to reproduce: click the top-right button twice rapidly.",
	})
	if err != nil {
		t.Fatalf("create issue 1: %v", err)
	}
	if iss1.Number != 1 {
		t.Errorf("expected issue number 1, got %d", iss1.Number)
	}
	if iss1.State != StateOpen {
		t.Errorf("expected issue state open, got %s", iss1.State)
	}

	// 6. Create Issue #2 by alice (repo owner) with labels and milestone
	iss2, err := issuesSvc.CreateIssue(ctx, user1.ID, false, "alice", "alpha-project", CreateIssueRequest{
		Title:       "Implement dark mode theme",
		Body:        "Add OLED dark background and high contrast accents.",
		MilestoneID: &milestone.ID,
		LabelIDs:    []string{labelFeat.ID},
		AssigneeIDs: []string{user1.ID},
	})
	if err != nil {
		t.Fatalf("create issue 2: %v", err)
	}
	if iss2.Number != 2 {
		t.Errorf("expected issue number 2, got %d", iss2.Number)
	}
	if len(iss2.Labels) != 1 || iss2.Labels[0].Name != "feature" {
		t.Errorf("expected label feature on issue 2, got %+v", iss2.Labels)
	}
	if len(iss2.Assignees) != 1 || iss2.Assignees[0].Username != "alice" {
		t.Errorf("expected assignee alice on issue 2, got %+v", iss2.Assignees)
	}

	// 7. Add Comment to Issue #1
	comment, err := issuesSvc.CreateComment(ctx, user1.ID, false, "alice", "alpha-project", 1, CreateCommentRequest{
		Body: "Thanks for reporting! Fixed in commit 4b2c1d.",
	})
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}
	if comment.Body != "Thanks for reporting! Fixed in commit 4b2c1d." {
		t.Errorf("unexpected comment body: %s", comment.Body)
	}

	// 8. Fetch Issue Detail for #1
	detail1, err := issuesSvc.GetIssue(ctx, user2.ID, false, "alice", "alpha-project", 1)
	if err != nil {
		t.Fatalf("get issue 1 detail: %v", err)
	}
	if len(detail1.Comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(detail1.Comments))
	}
	if detail1.Comments[0].Author.Username != "alice" {
		t.Errorf("expected comment author alice, got %s", detail1.Comments[0].Author.Username)
	}

	// 9. Close Issue #1
	closedState := StateClosed
	updated1, err := issuesSvc.UpdateIssue(ctx, user1.ID, false, "alice", "alpha-project", 1, UpdateIssueRequest{
		State: &closedState,
	})
	if err != nil {
		t.Fatalf("close issue 1: %v", err)
	}
	if updated1.State != StateClosed {
		t.Errorf("expected state closed, got %s", updated1.State)
	}
	if updated1.ClosedAt == nil {
		t.Errorf("expected closed_at timestamp to be set")
	}

	// 10. List issues with state filters
	openIssues, openCount, closedCount, err := issuesSvc.ListIssues(ctx, user1.ID, false, "alice", "alpha-project", IssueFilter{
		State: "open",
	})
	if err != nil {
		t.Fatalf("list open issues: %v", err)
	}
	if len(openIssues) != 1 || openIssues[0].Number != 2 {
		t.Errorf("expected 1 open issue (#2), got %+v", openIssues)
	}
	if openCount != 1 || closedCount != 1 {
		t.Errorf("expected counts (1, 1), got (%d, %d)", openCount, closedCount)
	}

	// 11. Filter by label
	featIssues, _, _, err := issuesSvc.ListIssues(ctx, user1.ID, false, "alice", "alpha-project", IssueFilter{
		LabelName: "feature",
	})
	if err != nil {
		t.Fatalf("list by label feature: %v", err)
	}
	if len(featIssues) != 1 || featIssues[0].Number != 2 {
		t.Errorf("expected issue #2 matching label feature, got %+v", featIssues)
	}

	// 12. Update milestone status
	milestones, err := issuesSvc.ListMilestones(ctx, user1.ID, false, "alice", "alpha-project")
	if err != nil || len(milestones) == 0 {
		t.Fatalf("expected milestones, got: %v", err)
	}

	_ = labelBug
	_ = repo
}

func TestIssuesPermissions(t *testing.T) {
	ctx := context.Background()
	issuesSvc, reposSvc, authSvc, cleanup := setupIssuesTest(t)
	defer cleanup()

	owner, _, _ := authSvc.Register(ctx, auth.RegisterRequest{
		Username: "owner",
		Email:    "owner@example.com",
		Password: "Password123!",
	}, "127.0.0.1", "test-agent")

	stranger, _, _ := authSvc.Register(ctx, auth.RegisterRequest{
		Username: "stranger",
		Email:    "stranger@example.com",
		Password: "Password123!",
	}, "127.0.0.1", "test-agent")

	// Create Private repo
	_, err := reposSvc.CreateRepository(ctx, owner.ID, false, repos.CreateRepoRequest{
		Name:           "Secret Vault",
		Slug:           "secret-vault",
		Visibility:     "private",
		InitWithReadme: true,
	})
	if err != nil {
		t.Fatalf("create private repo: %v", err)
	}

	// Stranger cannot view issues on private repo
	_, _, _, err = issuesSvc.ListIssues(ctx, stranger.ID, false, "owner", "secret-vault", IssueFilter{})
	if err == nil {
		t.Errorf("expected permission error when stranger lists private repo issues")
	}

	// Stranger cannot create issue on private repo
	_, err = issuesSvc.CreateIssue(ctx, stranger.ID, false, "owner", "secret-vault", CreateIssueRequest{
		Title: "I want access",
		Body:  "Please add me",
	})
	if err == nil {
		t.Errorf("expected permission error when stranger creates issue on private repo")
	}

	// Unauthenticated user cannot create issue
	_, err = issuesSvc.CreateIssue(ctx, "", false, "owner", "secret-vault", CreateIssueRequest{
		Title: "Anonymous",
		Body:  "Anon",
	})
	if err == nil {
		t.Errorf("expected error when unauthenticated user creates issue")
	}
}
