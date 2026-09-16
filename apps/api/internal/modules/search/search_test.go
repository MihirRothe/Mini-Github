package search

import (
	"context"
	"testing"
	"time"

	"forgehub/apps/api/internal/git"
	"forgehub/apps/api/internal/modules/auth"
)

type mockGitReader struct {
	git.Reader
	matches []git.GrepMatch
}

func (m *mockGitReader) Grep(diskPath, ref, query string, maxResults int) ([]git.GrepMatch, error) {
	var matched []git.GrepMatch
	for _, match := range m.matches {
		matched = append(matched, match)
	}
	return matched, nil
}

func TestQueryParser(t *testing.T) {
	raw := "refactor authentication repo:forgehub/core author:alice state:open lang:go sort:stars"
	pq := ParseQuery(raw)

	if pq.CleanQuery != "refactor authentication" {
		t.Errorf("expected clean query 'refactor authentication', got %q", pq.CleanQuery)
	}
	if pq.RepoOwner != "forgehub" || pq.RepoSlug != "core" {
		t.Errorf("expected repo forgehub/core, got %s/%s", pq.RepoOwner, pq.RepoSlug)
	}
	if pq.Author != "alice" {
		t.Errorf("expected author alice, got %s", pq.Author)
	}
	if pq.State != "open" {
		t.Errorf("expected state open, got %s", pq.State)
	}
	if pq.Language != "go" {
		t.Errorf("expected language go, got %s", pq.Language)
	}
	if pq.Sort != "stars" {
		t.Errorf("expected sort stars, got %s", pq.Sort)
	}

	rawPR := "fix bug is:pr is:closed"
	pqPR := ParseQuery(rawPR)
	if pqPR.IsPR == nil || !*pqPR.IsPR {
		t.Errorf("expected IsPR true, got %v", pqPR.IsPR)
	}
	if pqPR.State != "closed" {
		t.Errorf("expected state closed, got %s", pqPR.State)
	}
}

func TestSearchServiceMemoryStore(t *testing.T) {
	ctx := context.Background()
	store := NewMemorySearchStore()

	// Seed repositories: 1 public, 1 private
	store.AddRepo(RepoResultItem{
		ID:          "repo-1",
		Name:        "ForgeHub Core",
		Slug:        "forgehub-core",
		Description: "Production-grade collaboration engine",
		OwnerName:   "forgehub",
		OwnerType:   "org",
		Visibility:  "public",
		StarCount:   42,
		UpdatedAt:   time.Now(),
	}, "/data/git/forgehub/core.git")

	store.AddRepo(RepoResultItem{
		ID:          "repo-2",
		Name:        "Secret Agent",
		Slug:        "secret-agent",
		Description: "Confidential security module",
		OwnerName:   "forgehub",
		OwnerType:   "org",
		Visibility:  "private",
		StarCount:   10,
		UpdatedAt:   time.Now(),
	}, "/data/git/forgehub/secret.git", "user-admin")

	// Seed issues
	store.AddIssue(IssueResultItem{
		ID:            "issue-1",
		Number:        1,
		Title:         "Support OAuth2 and SAML SSO",
		BodySnippet:   "We need to implement SAML enterprise logins for ForgeHub",
		State:         "open",
		RepoOwner:     "forgehub",
		RepoSlug:      "forgehub-core",
		Author:        &auth.PublicUser{ID: "u-1", Username: "alice"},
		CommentsCount: 3,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	})

	// Seed PRs
	store.AddPull(PRResultItem{
		ID:           "pr-1",
		Number:       12,
		Title:        "Implement atomic merge engine",
		BodySnippet:  "Squash and merge support with git cherry-pick",
		State:        "open",
		SourceBranch: "feat/merge",
		TargetBranch: "main",
		RepoOwner:    "forgehub",
		RepoSlug:     "forgehub-core",
		Author:       &auth.PublicUser{ID: "u-2", Username: "bob"},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})

	// Seed users
	store.AddUser(UserResultItem{
		ID:          "u-1",
		Username:    "alice",
		DisplayName: "Alice Developer",
		Bio:         "Distributed systems engineer",
	})
	store.AddUser(UserResultItem{
		ID:          "org-1",
		Username:    "forgehub",
		DisplayName: "ForgeHub Organization",
		Bio:         "Open source developer platform",
		IsOrg:       true,
	})

	mockGit := &mockGitReader{
		matches: []git.GrepMatch{
			{Path: "main.go", LineNumber: 42, LineText: "func StartServer() error {"},
		},
	}

	svc := NewService(store, mockGit)

	t.Run("Public Search Repositories", func(t *testing.T) {
		res, err := svc.Search(ctx, "", false, "forgehub", SearchTypeRepositories, 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res.Repositories) != 1 {
			t.Fatalf("expected 1 public repo for anonymous search, got %d", len(res.Repositories))
		}
		if res.Repositories[0].Slug != "forgehub-core" {
			t.Errorf("expected forgehub-core, got %s", res.Repositories[0].Slug)
		}
	})

	t.Run("Private Repo Access Isolation", func(t *testing.T) {
		// Anonymous search for 'secret' should yield 0 results
		anonRes, err := svc.Search(ctx, "", false, "secret", SearchTypeRepositories, 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(anonRes.Repositories) != 0 {
			t.Errorf("expected 0 repos for unauthorized user, got %d", len(anonRes.Repositories))
		}

		// User with access 'user-admin' should see 'secret-agent'
		authRes, err := svc.Search(ctx, "user-admin", false, "secret", SearchTypeRepositories, 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(authRes.Repositories) != 1 {
			t.Errorf("expected 1 repo for authorized user, got %d", len(authRes.Repositories))
		}
	})

	t.Run("Issue Search with State and Text", func(t *testing.T) {
		res, err := svc.Search(ctx, "", false, "OAuth2 is:open", SearchTypeIssues, 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res.Issues) != 1 {
			t.Fatalf("expected 1 issue match, got %d", len(res.Issues))
		}
		if res.Issues[0].Number != 1 {
			t.Errorf("expected issue #1, got #%d", res.Issues[0].Number)
		}
	})

	t.Run("PR Search", func(t *testing.T) {
		res, err := svc.Search(ctx, "", false, "atomic is:pr", SearchTypePulls, 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res.Pulls) != 1 {
			t.Fatalf("expected 1 PR match, got %d", len(res.Pulls))
		}
		if res.Pulls[0].Title != "Implement atomic merge engine" {
			t.Errorf("unexpected PR title: %s", res.Pulls[0].Title)
		}
	})

	t.Run("User Search", func(t *testing.T) {
		res, err := svc.Search(ctx, "", false, "alice", SearchTypeUsers, 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res.Users) != 1 {
			t.Fatalf("expected 1 user match, got %d", len(res.Users))
		}
		if res.Users[0].Username != "alice" {
			t.Errorf("expected alice, got %s", res.Users[0].Username)
		}
	})

	t.Run("Code Search", func(t *testing.T) {
		res, err := svc.Search(ctx, "", false, "StartServer", SearchTypeCode, 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res.Code) != 1 {
			t.Fatalf("expected 1 code match item, got %d", len(res.Code))
		}
		if res.Code[0].FilePath != "main.go" {
			t.Errorf("expected main.go, got %s", res.Code[0].FilePath)
		}
	})

	t.Run("Quick Search Multi-Entity", func(t *testing.T) {
		quick, err := svc.QuickSearch(ctx, "", false, "forgehub")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(quick.Repositories) == 0 {
			t.Errorf("expected repo match in quick search")
		}
		if len(quick.Users) == 0 {
			t.Errorf("expected org/user match in quick search")
		}
	})
}
