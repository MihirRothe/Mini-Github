package repos

import (
	"time"

	"forgehub/apps/api/internal/modules/orgs"
)

type Repository struct {
	ID                    string          `json:"id"`
	Name                  string          `json:"name"`
	Slug                  string          `json:"slug"`
	Description           string          `json:"description"`
	Visibility            string          `json:"visibility"` // "public", "private", "internal"
	DefaultBranch         string          `json:"default_branch"`
	IsArchived            bool            `json:"is_archived"`
	IsFork                bool            `json:"is_fork"`
	ForkedFromID          *string         `json:"forked_from_id,omitempty"`
	OwnerType             string          `json:"owner_type"` // "user" or "org"
	OwnerUserID           *string         `json:"owner_user_id,omitempty"`
	OwnerOrgID            *string         `json:"owner_org_id,omitempty"`
	OwnerName             string          `json:"owner_name"`
	DiskPath              string          `json:"disk_path,omitempty"`
	CloneURL              string          `json:"clone_url"`
	HTTPCloneURL          string          `json:"http_clone_url"`
	StarsCount            int             `json:"stars_count"`
	ForksCount            int             `json:"forks_count"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
	CurrentUserPermission orgs.Permission `json:"current_user_permission,omitempty"`
}

type CreateRepoRequest struct {
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	Description    string `json:"description"`
	Visibility     string `json:"visibility"` // "public", "private", "internal"
	OwnerType      string `json:"owner_type"` // "user" or "org"
	OwnerSlug      string `json:"owner_slug"` // username or org slug
	DefaultBranch  string `json:"default_branch"`
	InitWithReadme bool   `json:"init_with_readme"`
}

type UpdateRepoRequest struct {
	Description   *string `json:"description"`
	Visibility    *string `json:"visibility"`
	DefaultBranch *string `json:"default_branch"`
	IsArchived    *bool   `json:"is_archived"`
}

type RepoFilter struct {
	OwnerType string
	OwnerSlug string
	UserID    string
	Query     string
}
