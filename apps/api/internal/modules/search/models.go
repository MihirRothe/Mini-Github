package search

import (
	"time"

	"forgehub/apps/api/internal/git"
	"forgehub/apps/api/internal/modules/auth"
)

type SearchType string

const (
	SearchTypeAll          SearchType = "all"
	SearchTypeRepositories SearchType = "repositories"
	SearchTypeIssues       SearchType = "issues"
	SearchTypePulls        SearchType = "pulls"
	SearchTypeCode         SearchType = "code"
	SearchTypeUsers        SearchType = "users"
)

type ParsedQuery struct {
	Raw        string `json:"raw"`
	CleanQuery string `json:"clean_query"`
	RepoOwner  string `json:"repo_owner,omitempty"`
	RepoSlug   string `json:"repo_slug,omitempty"`
	Author     string `json:"author,omitempty"`
	State      string `json:"state,omitempty"` // "open", "closed", "merged"
	IsPR       *bool  `json:"is_pr,omitempty"`
	IsIssue    *bool  `json:"is_issue,omitempty"`
	Language   string `json:"language,omitempty"`
	Sort       string `json:"sort,omitempty"` // "best", "updated", "stars", "newest"
}

type RepoResultItem struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	Description   string    `json:"description"`
	OwnerName     string    `json:"owner_name"`
	OwnerType     string    `json:"owner_type"`
	Visibility    string    `json:"visibility"`
	DefaultBranch string    `json:"default_branch"`
	StarCount     int       `json:"star_count"`
	ForkCount     int       `json:"fork_count"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type IssueResultItem struct {
	ID            string           `json:"id"`
	Number        int              `json:"number"`
	Title         string           `json:"title"`
	BodySnippet   string           `json:"body_snippet"`
	State         string           `json:"state"`
	RepoOwner     string           `json:"repo_owner"`
	RepoSlug      string           `json:"repo_slug"`
	Author        *auth.PublicUser `json:"author,omitempty"`
	CommentsCount int              `json:"comments_count"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

type PRResultItem struct {
	ID            string           `json:"id"`
	Number        int              `json:"number"`
	Title         string           `json:"title"`
	BodySnippet   string           `json:"body_snippet"`
	State         string           `json:"state"`
	IsDraft       bool             `json:"is_draft"`
	SourceBranch  string           `json:"source_branch"`
	TargetBranch  string           `json:"target_branch"`
	RepoOwner     string           `json:"repo_owner"`
	RepoSlug      string           `json:"repo_slug"`
	Author        *auth.PublicUser `json:"author,omitempty"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

type CodeResultItem struct {
	RepoOwner string          `json:"repo_owner"`
	RepoSlug  string          `json:"repo_slug"`
	FilePath  string          `json:"file_path"`
	Matches   []git.GrepMatch `json:"matches"`
}

type UserResultItem struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Bio         string `json:"bio"`
	AvatarURL   string `json:"avatar_url"`
	IsOrg       bool   `json:"is_org"`
}

type SearchCounts struct {
	Repositories int `json:"repositories"`
	Issues       int `json:"issues"`
	Pulls        int `json:"pulls"`
	Code         int `json:"code"`
	Users        int `json:"users"`
}

type SearchResults struct {
	Query        string            `json:"query"`
	Type         SearchType        `json:"type"`
	TotalCount   int               `json:"total_count"`
	Page         int               `json:"page"`
	PerPage      int               `json:"per_page"`
	Counts       SearchCounts      `json:"counts"`
	Repositories []RepoResultItem  `json:"repositories,omitempty"`
	Issues       []IssueResultItem `json:"issues,omitempty"`
	Pulls        []PRResultItem    `json:"pulls,omitempty"`
	Code         []CodeResultItem  `json:"code,omitempty"`
	Users        []UserResultItem  `json:"users,omitempty"`
}

type QuickSearchResponse struct {
	Query        string            `json:"query"`
	Repositories []RepoResultItem  `json:"repositories"`
	Issues       []IssueResultItem `json:"issues"`
	Pulls        []PRResultItem    `json:"pulls"`
	Code         []CodeResultItem  `json:"code"`
	Users        []UserResultItem  `json:"users"`
}
