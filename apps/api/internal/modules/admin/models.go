package admin

import (
	"encoding/json"
	"time"
)

// AdminStats holds aggregated platform metrics and Go runtime telemetry.
type AdminStats struct {
	TotalUsers          int       `json:"total_users"`
	ActiveUsers         int       `json:"active_users"`
	SuspendedUsers      int       `json:"suspended_users"`
	AdminUsers          int       `json:"admin_users"`
	TotalOrganizations  int       `json:"total_organizations"`
	TotalRepositories   int       `json:"total_repositories"`
	PublicRepositories  int       `json:"public_repositories"`
	PrivateRepositories int       `json:"private_repositories"`
	TotalIssues         int       `json:"total_issues"`
	OpenIssues          int       `json:"open_issues"`
	ClosedIssues        int       `json:"closed_issues"`
	TotalPullRequests   int       `json:"total_pull_requests"`
	OpenPullRequests    int       `json:"open_pull_requests"`
	MergedPullRequests  int       `json:"merged_pull_requests"`
	TotalPipelines      int       `json:"total_pipelines"`
	QueuedPipelines     int       `json:"queued_pipelines"`
	RunningPipelines    int       `json:"running_pipelines"`
	TotalWebhooks       int       `json:"total_webhooks"`
	GoroutinesCount     int       `json:"goroutines_count"`
	AllocatedMemBytes   uint64    `json:"allocated_mem_bytes"`
	SysMemBytes         uint64    `json:"sys_mem_bytes"`
	GCCycles            uint32    `json:"gc_cycles"`
	UptimeSeconds       int64     `json:"uptime_seconds"`
	Timestamp           time.Time `json:"timestamp"`
}

// AuditLogEntry represents a recorded administrative or system event.
type AuditLogEntry struct {
	ID            string          `json:"id"`
	ActorID       *string         `json:"actor_id,omitempty"`
	ActorUsername string          `json:"actor_username,omitempty"`
	Action        string          `json:"action"`
	TargetType    string          `json:"target_type"`
	TargetID      string          `json:"target_id"`
	IPAddress     string          `json:"ip_address"`
	UserAgent     string          `json:"user_agent"`
	Metadata      json.RawMessage `json:"metadata"`
	CreatedAt     time.Time       `json:"created_at"`
}

// AuditLogFilter specifies parameters for querying the audit stream.
type AuditLogFilter struct {
	ActorUsername string `json:"actor"`
	Action        string `json:"action"`
	TargetType    string `json:"target_type"`
	Limit         int    `json:"limit"`
	Offset        int    `json:"offset"`
}

// AdminUserItem represents a user with administrative and suspension controls.
type AdminUserItem struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	AvatarURL   string    `json:"avatar_url"`
	IsAdmin     bool      `json:"is_admin"`
	IsSuspended bool      `json:"is_suspended"`
	ReposCount  int       `json:"repos_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UpdateUserAdminRequest allows toggling admin rights or account suspension.
type UpdateUserAdminRequest struct {
	IsAdmin     *bool `json:"is_admin,omitempty"`
	IsSuspended *bool `json:"is_suspended,omitempty"`
}

// AdminOrgItem represents an organization summary for admin inventory.
type AdminOrgItem struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	MemberCount int       `json:"member_count"`
	ReposCount  int       `json:"repos_count"`
	CreatedAt   time.Time `json:"created_at"`
}

// AdminRepoItem represents a repository summary for admin inventory.
type AdminRepoItem struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	OwnerName     string    `json:"owner_name"`
	OwnerType     string    `json:"owner_type"` // "user" or "org"
	Visibility    string    `json:"visibility"`
	DefaultBranch string    `json:"default_branch"`
	IsArchived    bool      `json:"is_archived"`
	CreatedAt     time.Time `json:"created_at"`
}
