package pulls

import (
	"time"

	"forgehub/apps/api/internal/git"
	"forgehub/apps/api/internal/modules/auth"
)

type PRState string

const (
	StateOpen   PRState = "open"
	StateClosed PRState = "closed"
	StateMerged PRState = "merged"
)

type ReviewState string

const (
	ReviewApproved         ReviewState = "approved"
	ReviewChangesRequested ReviewState = "changes_requested"
	ReviewCommented        ReviewState = "commented"
)

type PullRequest struct {
	ID             string           `json:"id"`
	RepositoryID   string           `json:"repository_id"`
	Number         int              `json:"number"`
	Title          string           `json:"title"`
	Body           string           `json:"body"`
	State          PRState          `json:"state"`
	SourceBranch   string           `json:"source_branch"`
	TargetBranch   string           `json:"target_branch"`
	AuthorID       *string          `json:"author_id,omitempty"`
	Author         *auth.PublicUser `json:"author,omitempty"`
	IsDraft        bool             `json:"is_draft"`
	MergeCommitSHA *string          `json:"merge_commit_sha,omitempty"`
	MergedByID     *string          `json:"merged_by_id,omitempty"`
	MergedBy       *auth.PublicUser `json:"merged_by,omitempty"`
	MergedAt       *time.Time       `json:"merged_at,omitempty"`
	ClosedByID     *string          `json:"closed_by_id,omitempty"`
	ClosedBy       *auth.PublicUser `json:"closed_by,omitempty"`
	ClosedAt       *time.Time       `json:"closed_at,omitempty"`
	CommentsCount  int              `json:"comments_count"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

type PullRequestDetail struct {
	PullRequest
	Reviews      []*PullRequestReview  `json:"reviews"`
	Comments     []*PullRequestComment `json:"comments"`
	Commits      []git.CommitInfo      `json:"commits"`
	Diff         *git.DiffResult       `json:"diff,omitempty"`
	CanMerge     bool                  `json:"can_merge"`
	HasConflicts bool                  `json:"has_conflicts"`
}

type PullRequestReview struct {
	ID            string           `json:"id"`
	PullRequestID string           `json:"pull_request_id"`
	ReviewerID    string           `json:"reviewer_id"`
	Reviewer      *auth.PublicUser `json:"reviewer,omitempty"`
	State         ReviewState      `json:"state"`
	Body          string           `json:"body"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

type PullRequestComment struct {
	ID            string           `json:"id"`
	PullRequestID string           `json:"pull_request_id"`
	ReviewID      *string          `json:"review_id,omitempty"`
	AuthorID      *string          `json:"author_id,omitempty"`
	Author        *auth.PublicUser `json:"author,omitempty"`
	FilePath      *string          `json:"file_path,omitempty"`
	LineNumber    *int             `json:"line_number,omitempty"`
	Body          string           `json:"body"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

type CreatePRRequest struct {
	Title        string `json:"title"`
	Body         string `json:"body"`
	SourceBranch string `json:"source_branch"`
	TargetBranch string `json:"target_branch"`
	IsDraft      bool   `json:"is_draft"`
}

type UpdatePRRequest struct {
	Title        *string  `json:"title,omitempty"`
	Body         *string  `json:"body,omitempty"`
	State        *PRState `json:"state,omitempty"`
	TargetBranch *string  `json:"target_branch,omitempty"`
	IsDraft      *bool    `json:"is_draft,omitempty"`
}

type MergePRRequest struct {
	Method        string `json:"method"` // "merge" or "squash"
	CommitMessage string `json:"commit_message"`
}

type CreateReviewRequest struct {
	State ReviewState `json:"state"` // "approved", "changes_requested", "commented"
	Body  string      `json:"body"`
}

type CreatePRCommentRequest struct {
	Body       string  `json:"body"`
	FilePath   *string `json:"file_path,omitempty"`
	LineNumber *int    `json:"line_number,omitempty"`
	ReviewID   *string `json:"review_id,omitempty"`
}

type PRFilter struct {
	State          string `json:"state"` // "open", "closed", "merged", "all"
	AuthorUsername string `json:"author"`
	SourceBranch   string `json:"source_branch"`
	TargetBranch   string `json:"target_branch"`
	Query          string `json:"q"`
}
