package repos

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"forgehub/apps/api/internal/database"
	"forgehub/apps/api/internal/modules/orgs"

	"github.com/google/uuid"
)

var (
	ErrRepoNotFound      = errors.New("repository not found")
	ErrRepoSlugTaken     = errors.New("a repository with this name already exists for this owner")
	ErrCollaboratorExists = errors.New("collaborator already exists")
)

type RepositoryStore interface {
	CreateRepo(ctx context.Context, repo *Repository) error
	GetRepoByOwnerAndSlug(ctx context.Context, ownerSlug, repoSlug string) (*Repository, error)
	GetRepoByID(ctx context.Context, id string) (*Repository, error)
	ListRepositories(ctx context.Context, filter RepoFilter) ([]*Repository, error)
	UpdateRepo(ctx context.Context, repo *Repository) error
	DeleteRepo(ctx context.Context, id string) error

	GetCollaboratorPermission(ctx context.Context, repoID, userID string) (*orgs.Permission, error)
	AddCollaborator(ctx context.Context, repoID, userID string, perm orgs.Permission) error
	RemoveCollaborator(ctx context.Context, repoID, userID string) error
}

func NewRepositoryStore(db *database.DB) RepositoryStore {
	if db.IsStandalone() {
		return NewMemoryRepositoryStore()
	}
	return &sqlRepositoryStore{db: db}
}

// ============================================================================
// SQL Repository (PostgreSQL)
// ============================================================================

type sqlRepositoryStore struct {
	db *database.DB
}

func (s *sqlRepositoryStore) CreateRepo(ctx context.Context, r *Repository) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	r.CreatedAt = now
	r.UpdatedAt = now

	query := `
		INSERT INTO repositories (
			id, name, slug, description, visibility, default_branch,
			is_archived, is_fork, forked_from_id, owner_user_id, owner_org_id,
			disk_path, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err := s.db.ExecContext(ctx, query,
		r.ID, r.Name, r.Slug, r.Description, r.Visibility, r.DefaultBranch,
		r.IsArchived, r.IsFork, r.ForkedFromID, r.OwnerUserID, r.OwnerOrgID,
		r.DiskPath, r.CreatedAt, r.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique") {
			return ErrRepoSlugTaken
		}
		return err
	}
	return nil
}

func (s *sqlRepositoryStore) GetRepoByOwnerAndSlug(ctx context.Context, ownerSlug, repoSlug string) (*Repository, error) {
	query := `
		SELECT r.id, r.name, r.slug, r.description, r.visibility, r.default_branch,
		       r.is_archived, r.is_fork, r.forked_from_id, r.owner_user_id, r.owner_org_id,
		       COALESCE(u.username, o.slug) AS owner_name,
		       CASE WHEN r.owner_user_id IS NOT NULL THEN 'user' ELSE 'org' END AS owner_type,
		       r.disk_path, r.created_at, r.updated_at
		FROM repositories r
		LEFT JOIN users u ON r.owner_user_id = u.id
		LEFT JOIN organizations o ON r.owner_org_id = o.id
		WHERE (LOWER(u.username) = LOWER($1) OR LOWER(o.slug) = LOWER($1))
		  AND LOWER(r.slug) = LOWER($2)
	`
	r := &Repository{}
	err := s.db.QueryRowContext(ctx, query, ownerSlug, repoSlug).Scan(
		&r.ID, &r.Name, &r.Slug, &r.Description, &r.Visibility, &r.DefaultBranch,
		&r.IsArchived, &r.IsFork, &r.ForkedFromID, &r.OwnerUserID, &r.OwnerOrgID,
		&r.OwnerName, &r.OwnerType, &r.DiskPath, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRepoNotFound
		}
		return nil, err
	}
	return r, nil
}

func (s *sqlRepositoryStore) GetRepoByID(ctx context.Context, id string) (*Repository, error) {
	query := `
		SELECT r.id, r.name, r.slug, r.description, r.visibility, r.default_branch,
		       r.is_archived, r.is_fork, r.forked_from_id, r.owner_user_id, r.owner_org_id,
		       COALESCE(u.username, o.slug) AS owner_name,
		       CASE WHEN r.owner_user_id IS NOT NULL THEN 'user' ELSE 'org' END AS owner_type,
		       r.disk_path, r.created_at, r.updated_at
		FROM repositories r
		LEFT JOIN users u ON r.owner_user_id = u.id
		LEFT JOIN organizations o ON r.owner_org_id = o.id
		WHERE r.id = $1
	`
	r := &Repository{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&r.ID, &r.Name, &r.Slug, &r.Description, &r.Visibility, &r.DefaultBranch,
		&r.IsArchived, &r.IsFork, &r.ForkedFromID, &r.OwnerUserID, &r.OwnerOrgID,
		&r.OwnerName, &r.OwnerType, &r.DiskPath, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRepoNotFound
		}
		return nil, err
	}
	return r, nil
}

func (s *sqlRepositoryStore) ListRepositories(ctx context.Context, filter RepoFilter) ([]*Repository, error) {
	query := `
		SELECT r.id, r.name, r.slug, r.description, r.visibility, r.default_branch,
		       r.is_archived, r.is_fork, r.forked_from_id, r.owner_user_id, r.owner_org_id,
		       COALESCE(u.username, o.slug) AS owner_name,
		       CASE WHEN r.owner_user_id IS NOT NULL THEN 'user' ELSE 'org' END AS owner_type,
		       r.disk_path, r.created_at, r.updated_at
		FROM repositories r
		LEFT JOIN users u ON r.owner_user_id = u.id
		LEFT JOIN organizations o ON r.owner_org_id = o.id
		WHERE 1=1
	`
	var args []any
	argIdx := 1

	if filter.OwnerSlug != "" {
		query += fmt.Sprintf(" AND (LOWER(u.username) = LOWER($%d) OR LOWER(o.slug) = LOWER($%d))", argIdx, argIdx)
		args = append(args, filter.OwnerSlug)
		argIdx++
	}

	if filter.Query != "" {
		query += fmt.Sprintf(" AND (r.name ILIKE $%d OR r.description ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+filter.Query+"%")
		argIdx++
	}

	query += " ORDER BY r.updated_at DESC LIMIT 50"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*Repository
	for rows.Next() {
		r := &Repository{}
		err := rows.Scan(
			&r.ID, &r.Name, &r.Slug, &r.Description, &r.Visibility, &r.DefaultBranch,
			&r.IsArchived, &r.IsFork, &r.ForkedFromID, &r.OwnerUserID, &r.OwnerOrgID,
			&r.OwnerName, &r.OwnerType, &r.DiskPath, &r.CreatedAt, &r.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func (s *sqlRepositoryStore) UpdateRepo(ctx context.Context, r *Repository) error {
	r.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE repositories
		SET description = $1, visibility = $2, default_branch = $3, is_archived = $4, updated_at = $5
		WHERE id = $6
	`
	res, err := s.db.ExecContext(ctx, query, r.Description, r.Visibility, r.DefaultBranch, r.IsArchived, r.UpdatedAt, r.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrRepoNotFound
	}
	return nil
}

func (s *sqlRepositoryStore) DeleteRepo(ctx context.Context, id string) error {
	query := `DELETE FROM repositories WHERE id = $1`
	res, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrRepoNotFound
	}
	return nil
}

func (s *sqlRepositoryStore) GetCollaboratorPermission(ctx context.Context, repoID, userID string) (*orgs.Permission, error) {
	query := `SELECT permission FROM repository_members WHERE repository_id = $1 AND user_id = $2`
	var p string
	err := s.db.QueryRowContext(ctx, query, repoID, userID).Scan(&p)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	perm := orgs.Permission(p)
	return &perm, nil
}

func (s *sqlRepositoryStore) AddCollaborator(ctx context.Context, repoID, userID string, perm orgs.Permission) error {
	query := `
		INSERT INTO repository_members (id, repository_id, user_id, permission, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (repository_id, user_id) DO UPDATE SET permission = EXCLUDED.permission
	`
	_, err := s.db.ExecContext(ctx, query, uuid.New().String(), repoID, userID, perm, time.Now().UTC())
	return err
}

func (s *sqlRepositoryStore) RemoveCollaborator(ctx context.Context, repoID, userID string) error {
	query := `DELETE FROM repository_members WHERE repository_id = $1 AND user_id = $2`
	_, err := s.db.ExecContext(ctx, query, repoID, userID)
	return err
}

// ============================================================================
// In-Memory Repository (Standalone Development Mode)
// ============================================================================

type memoryRepositoryStore struct {
	mu             sync.RWMutex
	repos          map[string]*Repository                 // id -> repo
	byOwnerAndSlug map[string]string                      // lower(owner/slug) -> id
	collaborators  map[string]map[string]orgs.Permission  // repoID -> userID -> perm
}

func NewMemoryRepositoryStore() RepositoryStore {
	return &memoryRepositoryStore{
		repos:          make(map[string]*Repository),
		byOwnerAndSlug: make(map[string]string),
		collaborators:  make(map[string]map[string]orgs.Permission),
	}
}

func (m *memoryRepositoryStore) CreateRepo(ctx context.Context, r *Repository) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := strings.ToLower(r.OwnerName + "/" + r.Slug)
	if _, exists := m.byOwnerAndSlug[key]; exists {
		return ErrRepoSlugTaken
	}

	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	r.CreatedAt = now
	r.UpdatedAt = now

	m.repos[r.ID] = r
	m.byOwnerAndSlug[key] = r.ID
	return nil
}

func (m *memoryRepositoryStore) GetRepoByOwnerAndSlug(ctx context.Context, ownerSlug, repoSlug string) (*Repository, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := strings.ToLower(ownerSlug + "/" + repoSlug)
	id, exists := m.byOwnerAndSlug[key]
	if !exists {
		return nil, ErrRepoNotFound
	}
	repo := m.repos[id]
	if repo == nil {
		return nil, ErrRepoNotFound
	}
	copied := *repo
	return &copied, nil
}

func (m *memoryRepositoryStore) GetRepoByID(ctx context.Context, id string) (*Repository, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	repo, exists := m.repos[id]
	if !exists {
		return nil, ErrRepoNotFound
	}
	copied := *repo
	return &copied, nil
}

func (m *memoryRepositoryStore) ListRepositories(ctx context.Context, filter RepoFilter) ([]*Repository, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Repository
	for _, repo := range m.repos {
		if filter.OwnerSlug != "" && !strings.EqualFold(repo.OwnerName, filter.OwnerSlug) {
			continue
		}
		if filter.Query != "" {
			q := strings.ToLower(filter.Query)
			if !strings.Contains(strings.ToLower(repo.Name), q) && !strings.Contains(strings.ToLower(repo.Description), q) {
				continue
			}
		}
		copied := *repo
		result = append(result, &copied)
	}
	return result, nil
}

func (m *memoryRepositoryStore) UpdateRepo(ctx context.Context, r *Repository) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.repos[r.ID]
	if !exists {
		return ErrRepoNotFound
	}

	existing.Description = r.Description
	existing.Visibility = r.Visibility
	existing.DefaultBranch = r.DefaultBranch
	existing.IsArchived = r.IsArchived
	existing.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *memoryRepositoryStore) DeleteRepo(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	repo, exists := m.repos[id]
	if !exists {
		return ErrRepoNotFound
	}

	key := strings.ToLower(repo.OwnerName + "/" + repo.Slug)
	delete(m.byOwnerAndSlug, key)
	delete(m.repos, id)
	delete(m.collaborators, id)
	return nil
}

func (m *memoryRepositoryStore) GetCollaboratorPermission(ctx context.Context, repoID, userID string) (*orgs.Permission, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if userMap, exists := m.collaborators[repoID]; exists {
		if p, ok := userMap[userID]; ok {
			copied := p
			return &copied, nil
		}
	}
	return nil, nil
}

func (m *memoryRepositoryStore) AddCollaborator(ctx context.Context, repoID, userID string, perm orgs.Permission) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.collaborators[repoID]; !exists {
		m.collaborators[repoID] = make(map[string]orgs.Permission)
	}
	m.collaborators[repoID][userID] = perm
	return nil
}

func (m *memoryRepositoryStore) RemoveCollaborator(ctx context.Context, repoID, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if userMap, exists := m.collaborators[repoID]; exists {
		delete(userMap, userID)
	}
	return nil
}
