package issues

import (
	"time"

	"forgehub/apps/api/internal/modules/auth"
)

type IssueState string

const (
	StateOpen   IssueState = "open"
	StateClosed IssueState = "closed"
)

type Issue struct {
	ID            string            `json:"id"`
	RepositoryID  string            `json:"repository_id"`
	Number        int               `json:"number"`
	Title         string            `json:"title"`
	Body          string            `json:"body"`
	State         IssueState        `json:"state"`
	AuthorID      *string           `json:"author_id,omitempty"`
	Author        *auth.PublicUser  `json:"author,omitempty"`
	MilestoneID   *string           `json:"milestone_id,omitempty"`
	Milestone     *Milestone        `json:"milestone,omitempty"`
	Labels        []*Label          `json:"labels"`
	Assignees     []*auth.PublicUser `json:"assignees"`
	CommentsCount int               `json:"comments_count"`
	ClosedAt      *time.Time        `json:"closed_at,omitempty"`
	ClosedByID    *string           `json:"closed_by_id,omitempty"`
	ClosedBy      *auth.PublicUser  `json:"closed_by,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

type IssueDetail struct {
	Issue
	Comments []*IssueComment `json:"comments"`
}

type IssueComment struct {
	ID        string           `json:"id"`
	IssueID   string           `json:"issue_id"`
	AuthorID  *string          `json:"author_id,omitempty"`
	Author    *auth.PublicUser `json:"author,omitempty"`
	Body      string           `json:"body"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

type Label struct {
	ID           string    `json:"id"`
	RepositoryID string    `json:"repository_id"`
	Name         string    `json:"name"`
	Color        string    `json:"color"`
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"created_at"`
}

type Milestone struct {
	ID                string     `json:"id"`
	RepositoryID      string     `json:"repository_id"`
	Title             string     `json:"title"`
	Description       string     `json:"description"`
	State             IssueState `json:"state"`
	DueDate           *time.Time `json:"due_date,omitempty"`
	OpenIssuesCount   int        `json:"open_issues_count"`
	ClosedIssuesCount int        `json:"closed_issues_count"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	ClosedAt          *time.Time `json:"closed_at,omitempty"`
}

type CreateIssueRequest struct {
	Title       string   `json:"title"`
	Body        string   `json:"body"`
	MilestoneID *string  `json:"milestone_id,omitempty"`
	LabelIDs    []string `json:"label_ids,omitempty"`
	AssigneeIDs []string `json:"assignee_ids,omitempty"`
}

type UpdateIssueRequest struct {
	Title       *string     `json:"title,omitempty"`
	Body        *string     `json:"body,omitempty"`
	State       *IssueState `json:"state,omitempty"`
	MilestoneID *string     `json:"milestone_id,omitempty"`
	LabelIDs    *[]string   `json:"label_ids,omitempty"`
	AssigneeIDs *[]string   `json:"assignee_ids,omitempty"`
}

type CreateCommentRequest struct {
	Body string `json:"body"`
}

type UpdateCommentRequest struct {
	Body string `json:"body"`
}

type CreateLabelRequest struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

type UpdateLabelRequest struct {
	Name        *string `json:"name,omitempty"`
	Color       *string `json:"color,omitempty"`
	Description *string `json:"description,omitempty"`
}

type CreateMilestoneRequest struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	DueDate     *time.Time `json:"due_date,omitempty"`
}

type UpdateMilestoneRequest struct {
	Title       *string     `json:"title,omitempty"`
	Description *string     `json:"description,omitempty"`
	State       *IssueState `json:"state,omitempty"`
	DueDate     *time.Time  `json:"due_date,omitempty"`
}

type IssueFilter struct {
	State            string `json:"state"` // "open", "closed", or "all"
	LabelName        string `json:"label"`
	MilestoneID      string `json:"milestone"`
	AssigneeUsername string `json:"assignee"`
	AuthorUsername   string `json:"author"`
	Query            string `json:"q"`
}
