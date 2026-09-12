package orgs

import (
	"time"
)

// OrgRole represents the role of a user within an organization.
type OrgRole string

const (
	OrgRoleOwner          OrgRole = "owner"
	OrgRoleAdmin          OrgRole = "admin"
	OrgRoleMember         OrgRole = "member"
	OrgRoleBillingManager OrgRole = "billing_manager"
)

// TeamRole represents the role of a user within a team.
type TeamRole string

const (
	TeamRoleMaintainer TeamRole = "maintainer"
	TeamRoleMember     TeamRole = "member"
)

// Permission represents granular repository access permissions.
type Permission string

const (
	PermNone     Permission = ""
	PermRead     Permission = "read"
	PermTriage   Permission = "triage"
	PermWrite    Permission = "write"
	PermMaintain Permission = "maintain"
	PermAdmin    Permission = "admin"
)

// Organization represents a ForgeHub organization account.
type Organization struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	AvatarURL   string    `json:"avatar_url"`
	Website     string    `json:"website"`
	Location    string    `json:"location"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	MemberCount int       `json:"member_count,omitempty"`
	TeamCount   int       `json:"team_count,omitempty"`
	RepoCount   int       `json:"repo_count,omitempty"`
	Role        OrgRole   `json:"role,omitempty"` // populated when queried for a specific user
}

// OrgMember represents a user's membership within an organization.
type OrgMember struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	UserID         string    `json:"user_id"`
	Username       string    `json:"username"`
	DisplayName    string    `json:"display_name"`
	AvatarURL      string    `json:"avatar_url"`
	Email          string    `json:"email"`
	Role           OrgRole   `json:"role"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Team represents a collaborative group within an organization.
type Team struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	Description    string    `json:"description"`
	Privacy        string    `json:"privacy"` // "visible" or "secret"
	MemberCount    int       `json:"member_count"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	CurrentUserRole TeamRole `json:"current_user_role,omitempty"`
}

// TeamMember represents a user's membership within a team.
type TeamMember struct {
	ID          string    `json:"id"`
	TeamID      string    `json:"team_id"`
	UserID      string    `json:"user_id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	AvatarURL   string    `json:"avatar_url"`
	Role        TeamRole  `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
}

// TeamRepoPermission defines repository access granted to a team.
type TeamRepoPermission struct {
	ID           string     `json:"id"`
	TeamID       string     `json:"team_id"`
	RepositoryID string     `json:"repository_id"`
	Permission   Permission `json:"permission"`
	CreatedAt    time.Time  `json:"created_at"`
}

// DTOs

type CreateOrgRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Website     string `json:"website"`
	Location    string `json:"location"`
	AvatarURL   string `json:"avatar_url"`
}

type UpdateOrgRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Website     string `json:"website"`
	Location    string `json:"location"`
	AvatarURL   string `json:"avatar_url"`
}

type AddOrgMemberRequest struct {
	Username string  `json:"username"`
	Role     OrgRole `json:"role"`
}

type UpdateOrgMemberRoleRequest struct {
	Role OrgRole `json:"role"`
}

type CreateTeamRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Privacy     string `json:"privacy"`
}

type UpdateTeamRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Privacy     string `json:"privacy"`
}

type AddTeamMemberRequest struct {
	Username string   `json:"username"`
	Role     TeamRole `json:"role"`
}

type UpdateTeamMemberRoleRequest struct {
	Role TeamRole `json:"role"`
}
