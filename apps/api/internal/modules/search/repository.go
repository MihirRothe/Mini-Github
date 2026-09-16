package search

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"forgehub/apps/api/internal/database"
	"forgehub/apps/api/internal/modules/auth"
)

type SearchStore interface {
	SearchRepositories(ctx context.Context, currentUserID string, isSiteAdmin bool, q ParsedQuery, page, perPage int) ([]RepoResultItem, int, error)
	SearchIssues(ctx context.Context, currentUserID string, isSiteAdmin bool, q ParsedQuery, page, perPage int) ([]IssueResultItem, int, error)
	SearchPulls(ctx context.Context, currentUserID string, isSiteAdmin bool, q ParsedQuery, page, perPage int) ([]PRResultItem, int, error)
	SearchUsers(ctx context.Context, q ParsedQuery, page, perPage int) ([]UserResultItem, int, error)
	ListAccessibleReposForCodeSearch(ctx context.Context, currentUserID string, isSiteAdmin bool, repoOwner, repoSlug string) ([]RepoAccessible, error)
}

type RepoAccessible struct {
	ID         string
	OwnerName  string
	RepoSlug   string
	DiskPath   string
	Visibility string
}

func NewSearchStore(db *database.DB) SearchStore {
	if db == nil || db.IsStandalone() {
		return NewMemorySearchStore()
	}
	return &sqlSearchStore{db: db}
}

// -------------------------------------------------------------------------
// SQL Implementation (PostgreSQL)
// -------------------------------------------------------------------------

type sqlSearchStore struct {
	db *database.DB
}

func (s *sqlSearchStore) SearchRepositories(ctx context.Context, currentUserID string, isSiteAdmin bool, q ParsedQuery, page, perPage int) ([]RepoResultItem, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	var whereClauses []string
	var args []any
	argIdx := 1

	// Access control clause
	if isSiteAdmin {
		whereClauses = append(whereClauses, "1=1")
	} else if currentUserID != "" {
		clause := fmt.Sprintf(`(
			r.visibility = 'public' 
			OR r.owner_user_id = $%d 
			OR EXISTS (SELECT 1 FROM organization_members om WHERE om.organization_id = r.owner_org_id AND om.user_id = $%d)
			OR EXISTS (SELECT 1 FROM repository_collaborators rc WHERE rc.repository_id = r.id AND rc.user_id = $%d)
		)`, argIdx, argIdx, argIdx)
		whereClauses = append(whereClauses, clause)
		args = append(args, currentUserID)
		argIdx++
	} else {
		whereClauses = append(whereClauses, "r.visibility = 'public'")
	}

	if q.CleanQuery != "" {
		searchTerm := "%" + strings.ToLower(q.CleanQuery) + "%"
		clause := fmt.Sprintf("(LOWER(r.name) LIKE $%d OR LOWER(r.slug) LIKE $%d OR LOWER(COALESCE(r.description, '')) LIKE $%d)", argIdx, argIdx, argIdx)
		whereClauses = append(whereClauses, clause)
		args = append(args, searchTerm)
		argIdx++
	}

	if q.RepoOwner != "" {
		clause := fmt.Sprintf("(LOWER(COALESCE(u.username, o.slug)) = LOWER($%d))", argIdx)
		whereClauses = append(whereClauses, clause)
		args = append(args, q.RepoOwner)
		argIdx++
	}

	if q.RepoSlug != "" {
		clause := fmt.Sprintf("(LOWER(r.slug) = LOWER($%d))", argIdx)
		whereClauses = append(whereClauses, clause)
		args = append(args, q.RepoSlug)
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM repositories r
		LEFT JOIN users u ON r.owner_user_id = u.id
		LEFT JOIN organizations o ON r.owner_org_id = o.id
		WHERE %s
	`, whereSQL)

	var totalCount int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	orderBy := "r.updated_at DESC"
	if q.Sort == "stars" {
		orderBy = "r.star_count DESC, r.updated_at DESC"
	}

	query := fmt.Sprintf(`
		SELECT r.id, r.name, r.slug, COALESCE(r.description, ''), r.visibility, r.default_branch,
		       COALESCE(u.username, o.slug) AS owner_name,
		       CASE WHEN r.owner_user_id IS NOT NULL THEN 'user' ELSE 'org' END AS owner_type,
		       COALESCE(r.star_count, 0), COALESCE(r.fork_count, 0), r.updated_at
		FROM repositories r
		LEFT JOIN users u ON r.owner_user_id = u.id
		LEFT JOIN organizations o ON r.owner_org_id = o.id
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, whereSQL, orderBy, argIdx, argIdx+1)

	queryArgs := append(args, perPage, offset)
	rows, err := s.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []RepoResultItem
	for rows.Next() {
		var item RepoResultItem
		if err := rows.Scan(
			&item.ID, &item.Name, &item.Slug, &item.Description, &item.Visibility, &item.DefaultBranch,
			&item.OwnerName, &item.OwnerType, &item.StarCount, &item.ForkCount, &item.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		results = append(results, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return results, totalCount, nil
}

func (s *sqlSearchStore) SearchIssues(ctx context.Context, currentUserID string, isSiteAdmin bool, q ParsedQuery, page, perPage int) ([]IssueResultItem, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	var whereClauses []string
	var args []any
	argIdx := 1

	// Access control on repo
	if isSiteAdmin {
		whereClauses = append(whereClauses, "1=1")
	} else if currentUserID != "" {
		clause := fmt.Sprintf(`(
			r.visibility = 'public' 
			OR r.owner_user_id = $%d 
			OR EXISTS (SELECT 1 FROM organization_members om WHERE om.organization_id = r.owner_org_id AND om.user_id = $%d)
			OR EXISTS (SELECT 1 FROM repository_collaborators rc WHERE rc.repository_id = r.id AND rc.user_id = $%d)
		)`, argIdx, argIdx, argIdx)
		whereClauses = append(whereClauses, clause)
		args = append(args, currentUserID)
		argIdx++
	} else {
		whereClauses = append(whereClauses, "r.visibility = 'public'")
	}

	if q.CleanQuery != "" {
		searchTerm := "%" + strings.ToLower(q.CleanQuery) + "%"
		clause := fmt.Sprintf("(LOWER(i.title) LIKE $%d OR LOWER(COALESCE(i.body, '')) LIKE $%d)", argIdx, argIdx)
		whereClauses = append(whereClauses, clause)
		args = append(args, searchTerm)
		argIdx++
	}

	if q.State != "" {
		clause := fmt.Sprintf("i.state = $%d", argIdx)
		whereClauses = append(whereClauses, clause)
		args = append(args, q.State)
		argIdx++
	}

	if q.Author != "" {
		clause := fmt.Sprintf("LOWER(u.username) = LOWER($%d)", argIdx)
		whereClauses = append(whereClauses, clause)
		args = append(args, q.Author)
		argIdx++
	}

	if q.RepoOwner != "" {
		clause := fmt.Sprintf("LOWER(COALESCE(ro_u.username, ro_o.slug)) = LOWER($%d)", argIdx)
		whereClauses = append(whereClauses, clause)
		args = append(args, q.RepoOwner)
		argIdx++
	}

	if q.RepoSlug != "" {
		clause := fmt.Sprintf("LOWER(r.slug) = LOWER($%d)", argIdx)
		whereClauses = append(whereClauses, clause)
		args = append(args, q.RepoSlug)
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM issues i
		JOIN repositories r ON i.repository_id = r.id
		LEFT JOIN users ro_u ON r.owner_user_id = ro_u.id
		LEFT JOIN organizations ro_o ON r.owner_org_id = ro_o.id
		LEFT JOIN users u ON i.author_id = u.id
		WHERE %s
	`, whereSQL)

	var totalCount int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT i.id, i.number, i.title, COALESCE(i.body, ''), i.state, i.comments_count, i.created_at, i.updated_at,
		       r.slug AS repo_slug, COALESCE(ro_u.username, ro_o.slug) AS repo_owner,
		       u.id, u.username, u.full_name, COALESCE(u.avatar_url, '')
		FROM issues i
		JOIN repositories r ON i.repository_id = r.id
		LEFT JOIN users ro_u ON r.owner_user_id = ro_u.id
		LEFT JOIN organizations ro_o ON r.owner_org_id = ro_o.id
		LEFT JOIN users u ON i.author_id = u.id
		WHERE %s
		ORDER BY i.updated_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	queryArgs := append(args, perPage, offset)
	rows, err := s.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []IssueResultItem
	for rows.Next() {
		var item IssueResultItem
		var author auth.PublicUser
		var fullBody string
		if err := rows.Scan(
			&item.ID, &item.Number, &item.Title, &fullBody, &item.State, &item.CommentsCount, &item.CreatedAt, &item.UpdatedAt,
			&item.RepoSlug, &item.RepoOwner,
			&author.ID, &author.Username, &author.DisplayName, &author.AvatarURL,
		); err != nil {
			return nil, 0, err
		}
		item.Author = &author
		if len(fullBody) > 160 {
			item.BodySnippet = fullBody[:157] + "..."
		} else {
			item.BodySnippet = fullBody
		}
		results = append(results, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return results, totalCount, nil
}

func (s *sqlSearchStore) SearchPulls(ctx context.Context, currentUserID string, isSiteAdmin bool, q ParsedQuery, page, perPage int) ([]PRResultItem, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	var whereClauses []string
	var args []any
	argIdx := 1

	// Access control on repo
	if isSiteAdmin {
		whereClauses = append(whereClauses, "1=1")
	} else if currentUserID != "" {
		clause := fmt.Sprintf(`(
			r.visibility = 'public' 
			OR r.owner_user_id = $%d 
			OR EXISTS (SELECT 1 FROM organization_members om WHERE om.organization_id = r.owner_org_id AND om.user_id = $%d)
			OR EXISTS (SELECT 1 FROM repository_collaborators rc WHERE rc.repository_id = r.id AND rc.user_id = $%d)
		)`, argIdx, argIdx, argIdx)
		whereClauses = append(whereClauses, clause)
		args = append(args, currentUserID)
		argIdx++
	} else {
		whereClauses = append(whereClauses, "r.visibility = 'public'")
	}

	if q.CleanQuery != "" {
		searchTerm := "%" + strings.ToLower(q.CleanQuery) + "%"
		clause := fmt.Sprintf("(LOWER(pr.title) LIKE $%d OR LOWER(COALESCE(pr.body, '')) LIKE $%d)", argIdx, argIdx)
		whereClauses = append(whereClauses, clause)
		args = append(args, searchTerm)
		argIdx++
	}

	if q.State != "" {
		clause := fmt.Sprintf("pr.state = $%d", argIdx)
		whereClauses = append(whereClauses, clause)
		args = append(args, q.State)
		argIdx++
	}

	if q.Author != "" {
		clause := fmt.Sprintf("LOWER(u.username) = LOWER($%d)", argIdx)
		whereClauses = append(whereClauses, clause)
		args = append(args, q.Author)
		argIdx++
	}

	if q.RepoOwner != "" {
		clause := fmt.Sprintf("LOWER(COALESCE(ro_u.username, ro_o.slug)) = LOWER($%d)", argIdx)
		whereClauses = append(whereClauses, clause)
		args = append(args, q.RepoOwner)
		argIdx++
	}

	if q.RepoSlug != "" {
		clause := fmt.Sprintf("LOWER(r.slug) = LOWER($%d)", argIdx)
		whereClauses = append(whereClauses, clause)
		args = append(args, q.RepoSlug)
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM pull_requests pr
		JOIN repositories r ON pr.repository_id = r.id
		LEFT JOIN users ro_u ON r.owner_user_id = ro_u.id
		LEFT JOIN organizations ro_o ON r.owner_org_id = ro_o.id
		LEFT JOIN users u ON pr.author_id = u.id
		WHERE %s
	`, whereSQL)

	var totalCount int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT pr.id, pr.number, pr.title, COALESCE(pr.body, ''), pr.state, pr.is_draft,
		       pr.source_branch, pr.target_branch, pr.created_at, pr.updated_at,
		       r.slug AS repo_slug, COALESCE(ro_u.username, ro_o.slug) AS repo_owner,
		       u.id, u.username, u.full_name, COALESCE(u.avatar_url, '')
		FROM pull_requests pr
		JOIN repositories r ON pr.repository_id = r.id
		LEFT JOIN users ro_u ON r.owner_user_id = ro_u.id
		LEFT JOIN organizations ro_o ON r.owner_org_id = ro_o.id
		LEFT JOIN users u ON pr.author_id = u.id
		WHERE %s
		ORDER BY pr.updated_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	queryArgs := append(args, perPage, offset)
	rows, err := s.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []PRResultItem
	for rows.Next() {
		var item PRResultItem
		var author auth.PublicUser
		var fullBody string
		if err := rows.Scan(
			&item.ID, &item.Number, &item.Title, &fullBody, &item.State, &item.IsDraft,
			&item.SourceBranch, &item.TargetBranch, &item.CreatedAt, &item.UpdatedAt,
			&item.RepoSlug, &item.RepoOwner,
			&author.ID, &author.Username, &author.DisplayName, &author.AvatarURL,
		); err != nil {
			return nil, 0, err
		}
		item.Author = &author
		if len(fullBody) > 160 {
			item.BodySnippet = fullBody[:157] + "..."
		} else {
			item.BodySnippet = fullBody
		}
		results = append(results, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return results, totalCount, nil
}

func (s *sqlSearchStore) SearchUsers(ctx context.Context, q ParsedQuery, page, perPage int) ([]UserResultItem, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	searchTerm := "%" + strings.ToLower(q.CleanQuery) + "%"

	countQuery := `
		SELECT (
			(SELECT COUNT(*) FROM users WHERE LOWER(username) LIKE $1 OR LOWER(display_name) LIKE $1 OR LOWER(COALESCE(bio, '')) LIKE $1)
			+
			(SELECT COUNT(*) FROM organizations WHERE LOWER(slug) LIKE $1 OR LOWER(name) LIKE $1 OR LOWER(COALESCE(description, '')) LIKE $1)
		)
	`
	var totalCount int
	if err := s.db.QueryRowContext(ctx, countQuery, searchTerm).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, username, display_name, COALESCE(bio, ''), COALESCE(avatar_url, ''), false AS is_org
		FROM users
		WHERE LOWER(username) LIKE $1 OR LOWER(display_name) LIKE $1 OR LOWER(COALESCE(bio, '')) LIKE $1
		UNION ALL
		SELECT id, slug AS username, name AS display_name, COALESCE(description, '') AS bio, COALESCE(avatar_url, ''), true AS is_org
		FROM organizations
		WHERE LOWER(slug) LIKE $1 OR LOWER(name) LIKE $1 OR LOWER(COALESCE(description, '')) LIKE $1
		ORDER BY username ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.QueryContext(ctx, query, searchTerm, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []UserResultItem
	for rows.Next() {
		var item UserResultItem
		if err := rows.Scan(&item.ID, &item.Username, &item.DisplayName, &item.Bio, &item.AvatarURL, &item.IsOrg); err != nil {
			return nil, 0, err
		}
		results = append(results, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return results, totalCount, nil
}

func (s *sqlSearchStore) ListAccessibleReposForCodeSearch(ctx context.Context, currentUserID string, isSiteAdmin bool, repoOwner, repoSlug string) ([]RepoAccessible, error) {
	var whereClauses []string
	var args []any
	argIdx := 1

	if isSiteAdmin {
		whereClauses = append(whereClauses, "1=1")
	} else if currentUserID != "" {
		clause := fmt.Sprintf(`(
			r.visibility = 'public' 
			OR r.owner_user_id = $%d 
			OR EXISTS (SELECT 1 FROM organization_members om WHERE om.organization_id = r.owner_org_id AND om.user_id = $%d)
			OR EXISTS (SELECT 1 FROM repository_collaborators rc WHERE rc.repository_id = r.id AND rc.user_id = $%d)
		)`, argIdx, argIdx, argIdx)
		whereClauses = append(whereClauses, clause)
		args = append(args, currentUserID)
		argIdx++
	} else {
		whereClauses = append(whereClauses, "r.visibility = 'public'")
	}

	if repoOwner != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("LOWER(COALESCE(u.username, o.slug)) = LOWER($%d)", argIdx))
		args = append(args, repoOwner)
		argIdx++
	}

	if repoSlug != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("LOWER(r.slug) = LOWER($%d)", argIdx))
		args = append(args, repoSlug)
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT r.id, COALESCE(u.username, o.slug) AS owner_name, r.slug, r.disk_path, r.visibility
		FROM repositories r
		LEFT JOIN users u ON r.owner_user_id = u.id
		LEFT JOIN organizations o ON r.owner_org_id = o.id
		WHERE %s
		LIMIT 50
	`, strings.Join(whereClauses, " AND "))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []RepoAccessible
	for rows.Next() {
		var item RepoAccessible
		if err := rows.Scan(&item.ID, &item.OwnerName, &item.RepoSlug, &item.DiskPath, &item.Visibility); err != nil {
			return nil, err
		}
		result = append(result, item)
	}

	return result, rows.Err()
}

// -------------------------------------------------------------------------
// In-Memory Implementation (Dual-Mode / Testing)
// -------------------------------------------------------------------------

type MemorySearchStore struct {
	mu           sync.RWMutex
	repos        []RepoResultItem
	repoPaths    map[string]string // id -> diskPath
	issues       []IssueResultItem
	pulls        []PRResultItem
	users        []UserResultItem
	accessibleTo map[string][]string // repoID -> list of allowed userIDs
}

func NewMemorySearchStore() *MemorySearchStore {
	return &MemorySearchStore{
		repoPaths:    make(map[string]string),
		accessibleTo: make(map[string][]string),
	}
}

func (m *MemorySearchStore) AddRepo(item RepoResultItem, diskPath string, allowedUsers ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.repos = append(m.repos, item)
	m.repoPaths[item.ID] = diskPath
	m.accessibleTo[item.ID] = allowedUsers
}

func (m *MemorySearchStore) AddIssue(item IssueResultItem) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.issues = append(m.issues, item)
}

func (m *MemorySearchStore) AddPull(item PRResultItem) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pulls = append(m.pulls, item)
}

func (m *MemorySearchStore) AddUser(item UserResultItem) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users = append(m.users, item)
}

func (m *MemorySearchStore) hasAccess(repoID string, visibility string, userID string, isSiteAdmin bool) bool {
	if isSiteAdmin || visibility == "public" {
		return true
	}
	if userID == "" {
		return false
	}
	allowed := m.accessibleTo[repoID]
	for _, u := range allowed {
		if u == userID {
			return true
		}
	}
	return false
}

func (m *MemorySearchStore) SearchRepositories(ctx context.Context, currentUserID string, isSiteAdmin bool, q ParsedQuery, page, perPage int) ([]RepoResultItem, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var matched []RepoResultItem
	clean := strings.ToLower(q.CleanQuery)

	for _, r := range m.repos {
		if !m.hasAccess(r.ID, r.Visibility, currentUserID, isSiteAdmin) {
			continue
		}
		if q.RepoOwner != "" && !strings.EqualFold(r.OwnerName, q.RepoOwner) {
			continue
		}
		if q.RepoSlug != "" && !strings.EqualFold(r.Slug, q.RepoSlug) {
			continue
		}
		if clean != "" {
			if !strings.Contains(strings.ToLower(r.Name), clean) &&
				!strings.Contains(strings.ToLower(r.Slug), clean) &&
				!strings.Contains(strings.ToLower(r.Description), clean) {
				continue
			}
		}
		matched = append(matched, r)
	}

	sort.Slice(matched, func(i, j int) bool {
		if q.Sort == "stars" {
			return matched[i].StarCount > matched[j].StarCount
		}
		return matched[i].UpdatedAt.After(matched[j].UpdatedAt)
	})

	total := len(matched)
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	start := (page - 1) * perPage
	if start >= total {
		return []RepoResultItem{}, total, nil
	}
	end := start + perPage
	if end > total {
		end = total
	}

	return matched[start:end], total, nil
}

func (m *MemorySearchStore) SearchIssues(ctx context.Context, currentUserID string, isSiteAdmin bool, q ParsedQuery, page, perPage int) ([]IssueResultItem, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var matched []IssueResultItem
	clean := strings.ToLower(q.CleanQuery)

	for _, i := range m.issues {
		// Find repo
		var repoVisibility string
		var repoID string
		for _, r := range m.repos {
			if strings.EqualFold(r.OwnerName, i.RepoOwner) && strings.EqualFold(r.Slug, i.RepoSlug) {
				repoVisibility = r.Visibility
				repoID = r.ID
				break
			}
		}
		if repoID != "" && !m.hasAccess(repoID, repoVisibility, currentUserID, isSiteAdmin) {
			continue
		}

		if q.RepoOwner != "" && !strings.EqualFold(i.RepoOwner, q.RepoOwner) {
			continue
		}
		if q.RepoSlug != "" && !strings.EqualFold(i.RepoSlug, q.RepoSlug) {
			continue
		}
		if q.State != "" && !strings.EqualFold(i.State, q.State) {
			continue
		}
		if q.Author != "" && (i.Author == nil || !strings.EqualFold(i.Author.Username, q.Author)) {
			continue
		}
		if clean != "" {
			if !strings.Contains(strings.ToLower(i.Title), clean) &&
				!strings.Contains(strings.ToLower(i.BodySnippet), clean) {
				continue
			}
		}
		matched = append(matched, i)
	}

	sort.Slice(matched, func(i, j int) bool {
		return matched[i].UpdatedAt.After(matched[j].UpdatedAt)
	})

	total := len(matched)
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	start := (page - 1) * perPage
	if start >= total {
		return []IssueResultItem{}, total, nil
	}
	end := start + perPage
	if end > total {
		end = total
	}

	return matched[start:end], total, nil
}

func (m *MemorySearchStore) SearchPulls(ctx context.Context, currentUserID string, isSiteAdmin bool, q ParsedQuery, page, perPage int) ([]PRResultItem, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var matched []PRResultItem
	clean := strings.ToLower(q.CleanQuery)

	for _, pr := range m.pulls {
		var repoVisibility string
		var repoID string
		for _, r := range m.repos {
			if strings.EqualFold(r.OwnerName, pr.RepoOwner) && strings.EqualFold(r.Slug, pr.RepoSlug) {
				repoVisibility = r.Visibility
				repoID = r.ID
				break
			}
		}
		if repoID != "" && !m.hasAccess(repoID, repoVisibility, currentUserID, isSiteAdmin) {
			continue
		}

		if q.RepoOwner != "" && !strings.EqualFold(pr.RepoOwner, q.RepoOwner) {
			continue
		}
		if q.RepoSlug != "" && !strings.EqualFold(pr.RepoSlug, q.RepoSlug) {
			continue
		}
		if q.State != "" && !strings.EqualFold(pr.State, q.State) {
			continue
		}
		if q.Author != "" && (pr.Author == nil || !strings.EqualFold(pr.Author.Username, q.Author)) {
			continue
		}
		if clean != "" {
			if !strings.Contains(strings.ToLower(pr.Title), clean) &&
				!strings.Contains(strings.ToLower(pr.BodySnippet), clean) {
				continue
			}
		}
		matched = append(matched, pr)
	}

	sort.Slice(matched, func(i, j int) bool {
		return matched[i].UpdatedAt.After(matched[j].UpdatedAt)
	})

	total := len(matched)
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	start := (page - 1) * perPage
	if start >= total {
		return []PRResultItem{}, total, nil
	}
	end := start + perPage
	if end > total {
		end = total
	}

	return matched[start:end], total, nil
}

func (m *MemorySearchStore) SearchUsers(ctx context.Context, q ParsedQuery, page, perPage int) ([]UserResultItem, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var matched []UserResultItem
	clean := strings.ToLower(q.CleanQuery)

	for _, u := range m.users {
		if clean != "" {
			if !strings.Contains(strings.ToLower(u.Username), clean) &&
				!strings.Contains(strings.ToLower(u.DisplayName), clean) &&
				!strings.Contains(strings.ToLower(u.Bio), clean) {
				continue
			}
		}
		matched = append(matched, u)
	}

	sort.Slice(matched, func(i, j int) bool {
		return matched[i].Username < matched[j].Username
	})

	total := len(matched)
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	start := (page - 1) * perPage
	if start >= total {
		return []UserResultItem{}, total, nil
	}
	end := start + perPage
	if end > total {
		end = total
	}

	return matched[start:end], total, nil
}

func (m *MemorySearchStore) ListAccessibleReposForCodeSearch(ctx context.Context, currentUserID string, isSiteAdmin bool, repoOwner, repoSlug string) ([]RepoAccessible, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []RepoAccessible
	for _, r := range m.repos {
		if !m.hasAccess(r.ID, r.Visibility, currentUserID, isSiteAdmin) {
			continue
		}
		if repoOwner != "" && !strings.EqualFold(r.OwnerName, repoOwner) {
			continue
		}
		if repoSlug != "" && !strings.EqualFold(r.Slug, repoSlug) {
			continue
		}
		list = append(list, RepoAccessible{
			ID:         r.ID,
			OwnerName:  r.OwnerName,
			RepoSlug:   r.Slug,
			DiskPath:   m.repoPaths[r.ID],
			Visibility: r.Visibility,
		})
	}

	return list, nil
}
