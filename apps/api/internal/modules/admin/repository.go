package admin

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"forgehub/apps/api/internal/database"
	"forgehub/apps/api/internal/modules/auth"

	"github.com/google/uuid"
)

var (
	ErrUserNotFound = errors.New("user not found")
)

type Repository interface {
	GetSystemCounts(ctx context.Context) (*AdminStats, error)
	ListAuditLogs(ctx context.Context, filter AuditLogFilter) ([]*AuditLogEntry, int, error)
	CreateAuditLog(ctx context.Context, entry *AuditLogEntry) error
	ListUsers(ctx context.Context, query string, limit, offset int) ([]*AdminUserItem, int, error)
	UpdateUserStatus(ctx context.Context, userID string, isAdmin, isSuspended *bool) (*AdminUserItem, error)
	ListOrganizations(ctx context.Context, limit, offset int) ([]*AdminOrgItem, int, error)
	ListRepositories(ctx context.Context, limit, offset int) ([]*AdminRepoItem, int, error)
}

func NewRepository(db *database.DB, authRepo auth.Repository) Repository {
	if db.IsStandalone() {
		return NewMemoryRepository(authRepo)
	}
	return &sqlRepository{db: db, authRepo: authRepo}
}

// ============================================================================
// SQL Repository (PostgreSQL)
// ============================================================================

type sqlRepository struct {
	db       *database.DB
	authRepo auth.Repository
}

func (r *sqlRepository) GetSystemCounts(ctx context.Context) (*AdminStats, error) {
	stats := &AdminStats{}

	// Users counts
	_ = r.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*),
			COALESCE(COUNT(CASE WHEN is_suspended = FALSE THEN 1 END), 0),
			COALESCE(COUNT(CASE WHEN is_suspended = TRUE THEN 1 END), 0),
			COALESCE(COUNT(CASE WHEN is_admin = TRUE THEN 1 END), 0)
		FROM users
	`).Scan(&stats.TotalUsers, &stats.ActiveUsers, &stats.SuspendedUsers, &stats.AdminUsers)

	// Orgs count
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM organizations`).Scan(&stats.TotalOrganizations)

	// Repos count
	_ = r.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*),
			COALESCE(COUNT(CASE WHEN visibility = 'public' THEN 1 END), 0),
			COALESCE(COUNT(CASE WHEN visibility = 'private' THEN 1 END), 0)
		FROM repositories
	`).Scan(&stats.TotalRepositories, &stats.PublicRepositories, &stats.PrivateRepositories)

	// Issues count
	_ = r.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*),
			COALESCE(COUNT(CASE WHEN state = 'open' THEN 1 END), 0),
			COALESCE(COUNT(CASE WHEN state = 'closed' THEN 1 END), 0)
		FROM issues
	`).Scan(&stats.TotalIssues, &stats.OpenIssues, &stats.ClosedIssues)

	// Pulls count
	_ = r.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*),
			COALESCE(COUNT(CASE WHEN state = 'open' THEN 1 END), 0),
			COALESCE(COUNT(CASE WHEN state = 'merged' THEN 1 END), 0)
		FROM pull_requests
	`).Scan(&stats.TotalPullRequests, &stats.OpenPullRequests, &stats.MergedPullRequests)

	// Pipelines count
	_ = r.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*),
			COALESCE(COUNT(CASE WHEN status = 'queued' THEN 1 END), 0),
			COALESCE(COUNT(CASE WHEN status = 'running' THEN 1 END), 0)
		FROM pipeline_runs
	`).Scan(&stats.TotalPipelines, &stats.QueuedPipelines, &stats.RunningPipelines)

	// Webhooks count
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM webhooks`).Scan(&stats.TotalWebhooks)

	return stats, nil
}

func (r *sqlRepository) ListAuditLogs(ctx context.Context, filter AuditLogFilter) ([]*AuditLogEntry, int, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}

	whereClauses := []string{"1=1"}
	var args []any
	argIndex := 1

	if filter.Action != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("a.action = $%d", argIndex))
		args = append(args, filter.Action)
		argIndex++
	}
	if filter.TargetType != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("a.target_type = $%d", argIndex))
		args = append(args, filter.TargetType)
		argIndex++
	}
	if filter.ActorUsername != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("LOWER(u.username) = LOWER($%d)", argIndex))
		args = append(args, filter.ActorUsername)
		argIndex++
	}

	whereSql := strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM audit_logs a
		LEFT JOIN users u ON a.actor_id = u.id
		WHERE %s
	`, whereSql)

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT 
			a.id, a.actor_id, COALESCE(u.username, ''), a.action, a.target_type, a.target_id,
			a.ip_address, a.user_agent, a.metadata, a.created_at
		FROM audit_logs a
		LEFT JOIN users u ON a.actor_id = u.id
		WHERE %s
		ORDER BY a.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSql, argIndex, argIndex+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []*AuditLogEntry
	for rows.Next() {
		entry := &AuditLogEntry{}
		var rawMeta []byte
		var actorID sql.NullString
		if err := rows.Scan(
			&entry.ID, &actorID, &entry.ActorUsername, &entry.Action, &entry.TargetType, &entry.TargetID,
			&entry.IPAddress, &entry.UserAgent, &rawMeta, &entry.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		if actorID.Valid {
			entry.ActorID = &actorID.String
		}
		if len(rawMeta) > 0 {
			entry.Metadata = json.RawMessage(rawMeta)
		} else {
			entry.Metadata = json.RawMessage("{}")
		}
		logs = append(logs, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

func (r *sqlRepository) CreateAuditLog(ctx context.Context, entry *AuditLogEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	if len(entry.Metadata) == 0 {
		entry.Metadata = json.RawMessage("{}")
	}

	query := `
		INSERT INTO audit_logs (id, actor_id, action, target_type, target_id, ip_address, user_agent, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, query,
		entry.ID, entry.ActorID, entry.Action, entry.TargetType, entry.TargetID,
		entry.IPAddress, entry.UserAgent, entry.Metadata, entry.CreatedAt,
	)
	return err
}

func (r *sqlRepository) ListUsers(ctx context.Context, query string, limit, offset int) ([]*AdminUserItem, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	where := "1=1"
	var args []any
	if query != "" {
		where = "(LOWER(username) LIKE LOWER($1) OR LOWER(email) LIKE LOWER($1) OR LOWER(display_name) LIKE LOWER($1))"
		args = append(args, "%"+query+"%")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users WHERE %s", where)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	argIndex := len(args) + 1
	sqlQuery := fmt.Sprintf(`
		SELECT 
			u.id, u.username, u.email, u.display_name, u.avatar_url, u.is_admin, u.is_suspended,
			(SELECT COUNT(*) FROM repositories r WHERE r.owner_user_id = u.id) AS repos_count,
			u.created_at, u.updated_at
		FROM users u
		WHERE %s
		ORDER BY u.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIndex, argIndex+1)

	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*AdminUserItem
	for rows.Next() {
		u := &AdminUserItem{}
		if err := rows.Scan(
			&u.ID, &u.Username, &u.Email, &u.DisplayName, &u.AvatarURL, &u.IsAdmin, &u.IsSuspended,
			&u.ReposCount, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *sqlRepository) UpdateUserStatus(ctx context.Context, userID string, isAdmin, isSuspended *bool) (*AdminUserItem, error) {
	var currentIsAdmin, currentIsSuspended bool
	err := r.db.QueryRowContext(ctx, "SELECT is_admin, is_suspended FROM users WHERE id = $1", userID).Scan(&currentIsAdmin, &currentIsSuspended)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if isAdmin != nil {
		currentIsAdmin = *isAdmin
	}
	if isSuspended != nil {
		currentIsSuspended = *isSuspended
	}

	now := time.Now().UTC()
	_, err = r.db.ExecContext(ctx, `
		UPDATE users 
		SET is_admin = $1, is_suspended = $2, updated_at = $3
		WHERE id = $4
	`, currentIsAdmin, currentIsSuspended, now, userID)
	if err != nil {
		return nil, err
	}

	u := &AdminUserItem{}
	err = r.db.QueryRowContext(ctx, `
		SELECT 
			u.id, u.username, u.email, u.display_name, u.avatar_url, u.is_admin, u.is_suspended,
			(SELECT COUNT(*) FROM repositories r WHERE r.owner_user_id = u.id) AS repos_count,
			u.created_at, u.updated_at
		FROM users u
		WHERE u.id = $1
	`, userID).Scan(
		&u.ID, &u.Username, &u.Email, &u.DisplayName, &u.AvatarURL, &u.IsAdmin, &u.IsSuspended,
		&u.ReposCount, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *sqlRepository) ListOrganizations(ctx context.Context, limit, offset int) ([]*AdminOrgItem, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var total int
	_ = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM organizations").Scan(&total)

	query := `
		SELECT 
			o.id, o.name, o.slug, o.description,
			(SELECT COUNT(*) FROM organization_members m WHERE m.organization_id = o.id) AS member_count,
			(SELECT COUNT(*) FROM repositories r WHERE r.owner_org_id = o.id) AS repos_count,
			o.created_at
		FROM organizations o
		ORDER BY o.created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var orgs []*AdminOrgItem
	for rows.Next() {
		item := &AdminOrgItem{}
		if err := rows.Scan(&item.ID, &item.Name, &item.Slug, &item.Description, &item.MemberCount, &item.ReposCount, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		orgs = append(orgs, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return orgs, total, nil
}

func (r *sqlRepository) ListRepositories(ctx context.Context, limit, offset int) ([]*AdminRepoItem, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var total int
	_ = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM repositories").Scan(&total)

	query := `
		SELECT 
			r.id, r.name, r.slug,
			COALESCE(u.username, o.slug) AS owner_name,
			CASE WHEN r.owner_user_id IS NOT NULL THEN 'user' ELSE 'org' END AS owner_type,
			r.visibility, r.default_branch, r.is_archived, r.created_at
		FROM repositories r
		LEFT JOIN users u ON r.owner_user_id = u.id
		LEFT JOIN organizations o ON r.owner_org_id = o.id
		ORDER BY r.created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var repos []*AdminRepoItem
	for rows.Next() {
		item := &AdminRepoItem{}
		if err := rows.Scan(
			&item.ID, &item.Name, &item.Slug, &item.OwnerName, &item.OwnerType,
			&item.Visibility, &item.DefaultBranch, &item.IsArchived, &item.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		repos = append(repos, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return repos, total, nil
}

// ============================================================================
// In-Memory Repository (For Standalone / Tests)
// ============================================================================

type memoryRepository struct {
	mu        sync.RWMutex
	authRepo  auth.Repository
	auditLogs []*AuditLogEntry
	orgs      []*AdminOrgItem
	repos     []*AdminRepoItem
}

func NewMemoryRepository(authRepo auth.Repository) Repository {
	return &memoryRepository{
		authRepo:  authRepo,
		auditLogs: make([]*AuditLogEntry, 0),
		orgs:      make([]*AdminOrgItem, 0),
		repos:     make([]*AdminRepoItem, 0),
	}
}

func (m *memoryRepository) GetSystemCounts(ctx context.Context) (*AdminStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &AdminStats{
		TotalUsers:          2,
		ActiveUsers:         2,
		AdminUsers:          1,
		TotalOrganizations:  len(m.orgs),
		TotalRepositories:   len(m.repos),
		PublicRepositories:  len(m.repos),
		TotalIssues:         1,
		OpenIssues:          1,
		TotalPullRequests:   1,
		OpenPullRequests:    1,
		TotalPipelines:      1,
		TotalWebhooks:       1,
	}
	return stats, nil
}

func (m *memoryRepository) ListAuditLogs(ctx context.Context, filter AuditLogFilter) ([]*AuditLogEntry, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var filtered []*AuditLogEntry
	for _, l := range m.auditLogs {
		if filter.Action != "" && l.Action != filter.Action {
			continue
		}
		if filter.TargetType != "" && l.TargetType != filter.TargetType {
			continue
		}
		if filter.ActorUsername != "" && !strings.EqualFold(l.ActorUsername, filter.ActorUsername) {
			continue
		}
		filtered = append(filtered, l)
	}

	total := len(filtered)
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	start := filter.Offset
	if start > total {
		start = total
	}
	end := start + filter.Limit
	if end > total {
		end = total
	}

	return filtered[start:end], total, nil
}

func (m *memoryRepository) CreateAuditLog(ctx context.Context, entry *AuditLogEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	if len(entry.Metadata) == 0 {
		entry.Metadata = json.RawMessage("{}")
	}

	m.auditLogs = append([]*AuditLogEntry{entry}, m.auditLogs...)
	return nil
}

func (m *memoryRepository) ListUsers(ctx context.Context, query string, limit, offset int) ([]*AdminUserItem, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Seed sample admin user if needed
	items := []*AdminUserItem{
		{
			ID:          "admin-uuid-1",
			Username:    "admin",
			Email:       "admin@forgehub.local",
			DisplayName: "Site Administrator",
			IsAdmin:     true,
			IsSuspended: false,
			CreatedAt:   time.Now().Add(-24 * time.Hour),
		},
		{
			ID:          "user-uuid-2",
			Username:    "developer",
			Email:       "dev@forgehub.local",
			DisplayName: "Standard Developer",
			IsAdmin:     false,
			IsSuspended: false,
			CreatedAt:   time.Now().Add(-12 * time.Hour),
		},
	}

	var filtered []*AdminUserItem
	for _, u := range items {
		if query != "" && !strings.Contains(strings.ToLower(u.Username), strings.ToLower(query)) && !strings.Contains(strings.ToLower(u.Email), strings.ToLower(query)) {
			continue
		}
		filtered = append(filtered, u)
	}

	return filtered, len(filtered), nil
}

func (m *memoryRepository) UpdateUserStatus(ctx context.Context, userID string, isAdmin, isSuspended *bool) (*AdminUserItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	item := &AdminUserItem{
		ID:          userID,
		Username:    "user-" + userID,
		Email:       userID + "@example.com",
		DisplayName: "Updated User",
		CreatedAt:   time.Now().Add(-1 * time.Hour),
		UpdatedAt:   time.Now(),
	}
	if isAdmin != nil {
		item.IsAdmin = *isAdmin
	}
	if isSuspended != nil {
		item.IsSuspended = *isSuspended
	}
	return item, nil
}

func (m *memoryRepository) ListOrganizations(ctx context.Context, limit, offset int) ([]*AdminOrgItem, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.orgs, len(m.orgs), nil
}

func (m *memoryRepository) ListRepositories(ctx context.Context, limit, offset int) ([]*AdminRepoItem, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.repos, len(m.repos), nil
}
