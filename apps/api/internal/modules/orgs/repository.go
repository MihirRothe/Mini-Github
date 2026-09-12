package orgs

import (
	"context"
	"database/sql"
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
	ErrOrgNotFound          = errors.New("organization not found")
	ErrOrgSlugTaken         = errors.New("organization slug is already in use")
	ErrTeamNotFound         = errors.New("team not found")
	ErrTeamSlugTaken        = errors.New("team slug already exists in this organization")
	ErrMemberAlreadyExists   = errors.New("user is already a member")
	ErrMemberNotFound       = errors.New("member not found")
	ErrCannotRemoveLastOwner = errors.New("cannot remove or demote the last owner of an organization")
)

// Repository defines data access methods for organizations, teams, and memberships.
type Repository interface {
	CreateOrganization(ctx context.Context, org *Organization, ownerUserID string) error
	GetOrganizationBySlug(ctx context.Context, slug string) (*Organization, error)
	GetOrganizationByID(ctx context.Context, id string) (*Organization, error)
	ListUserOrganizations(ctx context.Context, userID string) ([]*Organization, error)
	UpdateOrganization(ctx context.Context, org *Organization) error
	DeleteOrganization(ctx context.Context, orgID string) error

	GetMember(ctx context.Context, orgID, userID string) (*OrgMember, error)
	ListMembers(ctx context.Context, orgID string) ([]*OrgMember, error)
	AddMember(ctx context.Context, member *OrgMember) error
	UpdateMemberRole(ctx context.Context, orgID, userID string, role OrgRole) error
	RemoveMember(ctx context.Context, orgID, userID string) error
	CountOwners(ctx context.Context, orgID string) (int, error)

	CreateTeam(ctx context.Context, team *Team) error
	GetTeamBySlug(ctx context.Context, orgID, slug string) (*Team, error)
	GetTeamByID(ctx context.Context, teamID string) (*Team, error)
	ListTeams(ctx context.Context, orgID string) ([]*Team, error)
	UpdateTeam(ctx context.Context, team *Team) error
	DeleteTeam(ctx context.Context, teamID string) error

	GetTeamMember(ctx context.Context, teamID, userID string) (*TeamMember, error)
	ListTeamMembers(ctx context.Context, teamID string) ([]*TeamMember, error)
	AddTeamMember(ctx context.Context, member *TeamMember) error
	UpdateTeamMemberRole(ctx context.Context, teamID, userID string, role TeamRole) error
	RemoveTeamMember(ctx context.Context, teamID, userID string) error

	GetUserTeamPermissions(ctx context.Context, orgID, userID, repoID string) ([]Permission, error)
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

func (r *sqlRepository) CreateOrganization(ctx context.Context, org *Organization, ownerUserID string) error {
	if org.ID == "" {
		org.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	org.CreatedAt = now
	org.UpdatedAt = now

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	queryOrg := `
		INSERT INTO organizations (id, name, slug, description, avatar_url, website, location, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err = tx.ExecContext(ctx, queryOrg, org.ID, org.Name, org.Slug, org.Description, org.AvatarURL, org.Website, org.Location, org.CreatedAt, org.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "idx_organizations_slug") {
			return ErrOrgSlugTaken
		}
		return err
	}

	memberID := uuid.New().String()
	queryMember := `
		INSERT INTO organization_members (id, org_id, user_id, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = tx.ExecContext(ctx, queryMember, memberID, org.ID, ownerUserID, OrgRoleOwner, now, now)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *sqlRepository) GetOrganizationBySlug(ctx context.Context, slug string) (*Organization, error) {
	query := `
		SELECT o.id, o.name, o.slug, o.description, o.avatar_url, COALESCE(o.website, ''), COALESCE(o.location, ''), o.created_at, o.updated_at,
		       (SELECT COUNT(*) FROM organization_members WHERE org_id = o.id) AS member_count,
		       (SELECT COUNT(*) FROM teams WHERE org_id = o.id) AS team_count,
		       (SELECT COUNT(*) FROM repositories WHERE owner_org_id = o.id) AS repo_count
		FROM organizations o
		WHERE LOWER(o.slug) = LOWER($1)
	`
	org := &Organization{}
	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&org.ID, &org.Name, &org.Slug, &org.Description, &org.AvatarURL, &org.Website, &org.Location,
		&org.CreatedAt, &org.UpdatedAt, &org.MemberCount, &org.TeamCount, &org.RepoCount,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrgNotFound
		}
		return nil, err
	}
	return org, nil
}

func (r *sqlRepository) GetOrganizationByID(ctx context.Context, id string) (*Organization, error) {
	query := `
		SELECT o.id, o.name, o.slug, o.description, o.avatar_url, COALESCE(o.website, ''), COALESCE(o.location, ''), o.created_at, o.updated_at,
		       (SELECT COUNT(*) FROM organization_members WHERE org_id = o.id) AS member_count,
		       (SELECT COUNT(*) FROM teams WHERE org_id = o.id) AS team_count,
		       (SELECT COUNT(*) FROM repositories WHERE owner_org_id = o.id) AS repo_count
		FROM organizations o
		WHERE o.id = $1
	`
	org := &Organization{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&org.ID, &org.Name, &org.Slug, &org.Description, &org.AvatarURL, &org.Website, &org.Location,
		&org.CreatedAt, &org.UpdatedAt, &org.MemberCount, &org.TeamCount, &org.RepoCount,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrgNotFound
		}
		return nil, err
	}
	return org, nil
}

func (r *sqlRepository) ListUserOrganizations(ctx context.Context, userID string) ([]*Organization, error) {
	query := `
		SELECT o.id, o.name, o.slug, o.description, o.avatar_url, COALESCE(o.website, ''), COALESCE(o.location, ''), o.created_at, o.updated_at,
		       m.role,
		       (SELECT COUNT(*) FROM organization_members WHERE org_id = o.id) AS member_count,
		       (SELECT COUNT(*) FROM teams WHERE org_id = o.id) AS team_count,
		       (SELECT COUNT(*) FROM repositories WHERE owner_org_id = o.id) AS repo_count
		FROM organizations o
		JOIN organization_members m ON o.id = m.org_id
		WHERE m.user_id = $1
		ORDER BY o.name ASC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*Organization
	for rows.Next() {
		org := &Organization{}
		var role string
		err := rows.Scan(
			&org.ID, &org.Name, &org.Slug, &org.Description, &org.AvatarURL, &org.Website, &org.Location,
			&org.CreatedAt, &org.UpdatedAt, &role, &org.MemberCount, &org.TeamCount, &org.RepoCount,
		)
		if err != nil {
			return nil, err
		}
		org.Role = OrgRole(role)
		result = append(result, org)
	}
	return result, rows.Err()
}

func (r *sqlRepository) UpdateOrganization(ctx context.Context, org *Organization) error {
	org.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE organizations
		SET name = $1, description = $2, avatar_url = $3, website = $4, location = $5, updated_at = $6
		WHERE id = $7
	`
	res, err := r.db.ExecContext(ctx, query, org.Name, org.Description, org.AvatarURL, org.Website, org.Location, org.UpdatedAt, org.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrOrgNotFound
	}
	return nil
}

func (r *sqlRepository) DeleteOrganization(ctx context.Context, orgID string) error {
	query := `DELETE FROM organizations WHERE id = $1`
	res, err := r.db.ExecContext(ctx, query, orgID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrOrgNotFound
	}
	return nil
}

func (r *sqlRepository) GetMember(ctx context.Context, orgID, userID string) (*OrgMember, error) {
	query := `
		SELECT m.id, m.org_id, m.user_id, m.role, m.created_at, m.updated_at,
		       u.username, u.display_name, u.avatar_url, u.email
		FROM organization_members m
		JOIN users u ON m.user_id = u.id
		WHERE m.org_id = $1 AND m.user_id = $2
	`
	member := &OrgMember{}
	var role string
	err := r.db.QueryRowContext(ctx, query, orgID, userID).Scan(
		&member.ID, &member.OrganizationID, &member.UserID, &role, &member.CreatedAt, &member.UpdatedAt,
		&member.Username, &member.DisplayName, &member.AvatarURL, &member.Email,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMemberNotFound
		}
		return nil, err
	}
	member.Role = OrgRole(role)
	return member, nil
}

func (r *sqlRepository) ListMembers(ctx context.Context, orgID string) ([]*OrgMember, error) {
	query := `
		SELECT m.id, m.org_id, m.user_id, m.role, m.created_at, m.updated_at,
		       u.username, u.display_name, u.avatar_url, u.email
		FROM organization_members m
		JOIN users u ON m.user_id = u.id
		WHERE m.org_id = $1
		ORDER BY m.created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*OrgMember
	for rows.Next() {
		m := &OrgMember{}
		var role string
		err := rows.Scan(
			&m.ID, &m.OrganizationID, &m.UserID, &role, &m.CreatedAt, &m.UpdatedAt,
			&m.Username, &m.DisplayName, &m.AvatarURL, &m.Email,
		)
		if err != nil {
			return nil, err
		}
		m.Role = OrgRole(role)
		list = append(list, m)
	}
	return list, rows.Err()
}

func (r *sqlRepository) AddMember(ctx context.Context, member *OrgMember) error {
	if member.ID == "" {
		member.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	member.CreatedAt = now
	member.UpdatedAt = now

	query := `
		INSERT INTO organization_members (id, org_id, user_id, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, query, member.ID, member.OrganizationID, member.UserID, member.Role, member.CreatedAt, member.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique") {
			return ErrMemberAlreadyExists
		}
		return err
	}
	return nil
}

func (r *sqlRepository) UpdateMemberRole(ctx context.Context, orgID, userID string, role OrgRole) error {
	query := `
		UPDATE organization_members
		SET role = $1, updated_at = $2
		WHERE org_id = $3 AND user_id = $4
	`
	res, err := r.db.ExecContext(ctx, query, role, time.Now().UTC(), orgID, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrMemberNotFound
	}
	return nil
}

func (r *sqlRepository) RemoveMember(ctx context.Context, orgID, userID string) error {
	query := `DELETE FROM organization_members WHERE org_id = $1 AND user_id = $2`
	res, err := r.db.ExecContext(ctx, query, orgID, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrMemberNotFound
	}
	return nil
}

func (r *sqlRepository) CountOwners(ctx context.Context, orgID string) (int, error) {
	query := `SELECT COUNT(*) FROM organization_members WHERE org_id = $1 AND role = 'owner'`
	var count int
	err := r.db.QueryRowContext(ctx, query, orgID).Scan(&count)
	return count, err
}

func (r *sqlRepository) CreateTeam(ctx context.Context, team *Team) error {
	if team.ID == "" {
		team.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	team.CreatedAt = now
	team.UpdatedAt = now
	if team.Privacy == "" {
		team.Privacy = "visible"
	}

	query := `
		INSERT INTO teams (id, org_id, name, slug, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query, team.ID, team.OrganizationID, team.Name, team.Slug, team.Description, team.CreatedAt, team.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "slug") {
			return ErrTeamSlugTaken
		}
		return err
	}
	return nil
}

func (r *sqlRepository) GetTeamBySlug(ctx context.Context, orgID, slug string) (*Team, error) {
	query := `
		SELECT t.id, t.org_id, t.name, t.slug, t.description, t.created_at, t.updated_at,
		       (SELECT COUNT(*) FROM team_members WHERE team_id = t.id) AS member_count
		FROM teams t
		WHERE t.org_id = $1 AND LOWER(t.slug) = LOWER($2)
	`
	team := &Team{}
	err := r.db.QueryRowContext(ctx, query, orgID, slug).Scan(
		&team.ID, &team.OrganizationID, &team.Name, &team.Slug, &team.Description,
		&team.CreatedAt, &team.UpdatedAt, &team.MemberCount,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTeamNotFound
		}
		return nil, err
	}
	team.Privacy = "visible"
	return team, nil
}

func (r *sqlRepository) GetTeamByID(ctx context.Context, teamID string) (*Team, error) {
	query := `
		SELECT t.id, t.org_id, t.name, t.slug, t.description, t.created_at, t.updated_at,
		       (SELECT COUNT(*) FROM team_members WHERE team_id = t.id) AS member_count
		FROM teams t
		WHERE t.id = $1
	`
	team := &Team{}
	err := r.db.QueryRowContext(ctx, query, teamID).Scan(
		&team.ID, &team.OrganizationID, &team.Name, &team.Slug, &team.Description,
		&team.CreatedAt, &team.UpdatedAt, &team.MemberCount,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTeamNotFound
		}
		return nil, err
	}
	team.Privacy = "visible"
	return team, nil
}

func (r *sqlRepository) ListTeams(ctx context.Context, orgID string) ([]*Team, error) {
	query := `
		SELECT t.id, t.org_id, t.name, t.slug, t.description, t.created_at, t.updated_at,
		       (SELECT COUNT(*) FROM team_members WHERE team_id = t.id) AS member_count
		FROM teams t
		WHERE t.org_id = $1
		ORDER BY t.name ASC
	`
	rows, err := r.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []*Team
	for rows.Next() {
		t := &Team{}
		err := rows.Scan(
			&t.ID, &t.OrganizationID, &t.Name, &t.Slug, &t.Description,
			&t.CreatedAt, &t.UpdatedAt, &t.MemberCount,
		)
		if err != nil {
			return nil, err
		}
		t.Privacy = "visible"
		teams = append(teams, t)
	}
	return teams, rows.Err()
}

func (r *sqlRepository) UpdateTeam(ctx context.Context, team *Team) error {
	team.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE teams
		SET name = $1, description = $2, updated_at = $3
		WHERE id = $4
	`
	res, err := r.db.ExecContext(ctx, query, team.Name, team.Description, team.UpdatedAt, team.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrTeamNotFound
	}
	return nil
}

func (r *sqlRepository) DeleteTeam(ctx context.Context, teamID string) error {
	query := `DELETE FROM teams WHERE id = $1`
	res, err := r.db.ExecContext(ctx, query, teamID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrTeamNotFound
	}
	return nil
}

func (r *sqlRepository) GetTeamMember(ctx context.Context, teamID, userID string) (*TeamMember, error) {
	query := `
		SELECT tm.id, tm.team_id, tm.user_id, tm.role, tm.created_at,
		       u.username, u.display_name, u.avatar_url
		FROM team_members tm
		JOIN users u ON tm.user_id = u.id
		WHERE tm.team_id = $1 AND tm.user_id = $2
	`
	tm := &TeamMember{}
	var role string
	err := r.db.QueryRowContext(ctx, query, teamID, userID).Scan(
		&tm.ID, &tm.TeamID, &tm.UserID, &role, &tm.CreatedAt,
		&tm.Username, &tm.DisplayName, &tm.AvatarURL,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMemberNotFound
		}
		return nil, err
	}
	tm.Role = TeamRole(role)
	return tm, nil
}

func (r *sqlRepository) ListTeamMembers(ctx context.Context, teamID string) ([]*TeamMember, error) {
	query := `
		SELECT tm.id, tm.team_id, tm.user_id, tm.role, tm.created_at,
		       u.username, u.display_name, u.avatar_url
		FROM team_members tm
		JOIN users u ON tm.user_id = u.id
		WHERE tm.team_id = $1
		ORDER BY tm.created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*TeamMember
	for rows.Next() {
		tm := &TeamMember{}
		var role string
		err := rows.Scan(
			&tm.ID, &tm.TeamID, &tm.UserID, &role, &tm.CreatedAt,
			&tm.Username, &tm.DisplayName, &tm.AvatarURL,
		)
		if err != nil {
			return nil, err
		}
		tm.Role = TeamRole(role)
		list = append(list, tm)
	}
	return list, rows.Err()
}

func (r *sqlRepository) AddTeamMember(ctx context.Context, member *TeamMember) error {
	if member.ID == "" {
		member.ID = uuid.New().String()
	}
	member.CreatedAt = time.Now().UTC()
	query := `
		INSERT INTO team_members (id, team_id, user_id, role, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, query, member.ID, member.TeamID, member.UserID, member.Role, member.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique") {
			return ErrMemberAlreadyExists
		}
		return err
	}
	return nil
}

func (r *sqlRepository) UpdateTeamMemberRole(ctx context.Context, teamID, userID string, role TeamRole) error {
	query := `
		UPDATE team_members
		SET role = $1
		WHERE team_id = $2 AND user_id = $3
	`
	res, err := r.db.ExecContext(ctx, query, role, teamID, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrMemberNotFound
	}
	return nil
}

func (r *sqlRepository) RemoveTeamMember(ctx context.Context, teamID, userID string) error {
	query := `DELETE FROM team_members WHERE team_id = $1 AND user_id = $2`
	res, err := r.db.ExecContext(ctx, query, teamID, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrMemberNotFound
	}
	return nil
}

func (r *sqlRepository) GetUserTeamPermissions(ctx context.Context, orgID, userID, repoID string) ([]Permission, error) {
	query := `
		SELECT tr.permission
		FROM team_repositories tr
		JOIN team_members tm ON tr.team_id = tm.team_id
		WHERE tm.user_id = $1 AND tr.repository_id = $2
	`
	rows, err := r.db.QueryContext(ctx, query, userID, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []Permission
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		perms = append(perms, Permission(p))
	}
	return perms, rows.Err()
}

// ============================================================================
// In-Memory Repository (Standalone Development Mode)
// ============================================================================

type memoryRepository struct {
	mu           sync.RWMutex
	authRepo     auth.Repository
	orgs         map[string]*Organization // orgID -> Org
	orgBySlug    map[string]string        // lower(slug) -> orgID
	orgMembers   map[string][]*OrgMember  // orgID -> []OrgMember
	teams        map[string]*Team         // teamID -> Team
	teamMembers  map[string][]*TeamMember // teamID -> []TeamMember
	teamRepos    map[string][]TeamRepoPermission // teamID -> []permissions
}

func NewMemoryRepository(authRepo auth.Repository) Repository {
	return &memoryRepository{
		authRepo:    authRepo,
		orgs:        make(map[string]*Organization),
		orgBySlug:   make(map[string]string),
		orgMembers:  make(map[string][]*OrgMember),
		teams:       make(map[string]*Team),
		teamMembers: make(map[string][]*TeamMember),
		teamRepos:   make(map[string][]TeamRepoPermission),
	}
}

func (m *memoryRepository) CreateOrganization(ctx context.Context, org *Organization, ownerUserID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	slugKey := strings.ToLower(org.Slug)
	if _, exists := m.orgBySlug[slugKey]; exists {
		return ErrOrgSlugTaken
	}

	if org.ID == "" {
		org.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	org.CreatedAt = now
	org.UpdatedAt = now

	// Fetch owner user details
	ownerUser, err := m.authRepo.GetUserByID(ctx, ownerUserID)
	if err != nil {
		return fmt.Errorf("lookup owner: %w", err)
	}

	m.orgs[org.ID] = org
	m.orgBySlug[slugKey] = org.ID

	// Seed owner member
	member := &OrgMember{
		ID:             uuid.New().String(),
		OrganizationID: org.ID,
		UserID:         ownerUserID,
		Username:       ownerUser.Username,
		DisplayName:    ownerUser.DisplayName,
		AvatarURL:      ownerUser.AvatarURL,
		Email:          ownerUser.Email,
		Role:           OrgRoleOwner,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	m.orgMembers[org.ID] = append(m.orgMembers[org.ID], member)

	return nil
}

func (m *memoryRepository) GetOrganizationBySlug(ctx context.Context, slug string) (*Organization, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	orgID, exists := m.orgBySlug[strings.ToLower(slug)]
	if !exists {
		return nil, ErrOrgNotFound
	}
	org := m.orgs[orgID]
	if org == nil {
		return nil, ErrOrgNotFound
	}

	copied := *org
	copied.MemberCount = len(m.orgMembers[orgID])
	teamCount := 0
	for _, t := range m.teams {
		if t.OrganizationID == orgID {
			teamCount++
		}
	}
	copied.TeamCount = teamCount
	return &copied, nil
}

func (m *memoryRepository) GetOrganizationByID(ctx context.Context, id string) (*Organization, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	org, exists := m.orgs[id]
	if !exists {
		return nil, ErrOrgNotFound
	}
	copied := *org
	copied.MemberCount = len(m.orgMembers[id])
	teamCount := 0
	for _, t := range m.teams {
		if t.OrganizationID == id {
			teamCount++
		}
	}
	copied.TeamCount = teamCount
	return &copied, nil
}

func (m *memoryRepository) ListUserOrganizations(ctx context.Context, userID string) ([]*Organization, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Organization
	for orgID, members := range m.orgMembers {
		for _, member := range members {
			if member.UserID == userID {
				org := m.orgs[orgID]
				if org != nil {
					copied := *org
					copied.Role = member.Role
					copied.MemberCount = len(members)
					teamCount := 0
					for _, t := range m.teams {
						if t.OrganizationID == orgID {
							teamCount++
						}
					}
					copied.TeamCount = teamCount
					result = append(result, &copied)
				}
				break
			}
		}
	}
	return result, nil
}

func (m *memoryRepository) UpdateOrganization(ctx context.Context, org *Organization) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.orgs[org.ID]
	if !exists {
		return ErrOrgNotFound
	}

	existing.Name = org.Name
	existing.Description = org.Description
	existing.AvatarURL = org.AvatarURL
	existing.Website = org.Website
	existing.Location = org.Location
	existing.UpdatedAt = time.Now().UTC()

	return nil
}

func (m *memoryRepository) DeleteOrganization(ctx context.Context, orgID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	org, exists := m.orgs[orgID]
	if !exists {
		return ErrOrgNotFound
	}

	delete(m.orgBySlug, strings.ToLower(org.Slug))
	delete(m.orgs, orgID)
	delete(m.orgMembers, orgID)

	// Clean up teams
	for teamID, team := range m.teams {
		if team.OrganizationID == orgID {
			delete(m.teams, teamID)
			delete(m.teamMembers, teamID)
			delete(m.teamRepos, teamID)
		}
	}

	return nil
}

func (m *memoryRepository) GetMember(ctx context.Context, orgID, userID string) (*OrgMember, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	members := m.orgMembers[orgID]
	for _, m := range members {
		if m.UserID == userID {
			copied := *m
			return &copied, nil
		}
	}
	return nil, ErrMemberNotFound
}

func (m *memoryRepository) ListMembers(ctx context.Context, orgID string) ([]*OrgMember, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	members := m.orgMembers[orgID]
	var list []*OrgMember
	for _, item := range members {
		copied := *item
		list = append(list, &copied)
	}
	return list, nil
}

func (m *memoryRepository) AddMember(ctx context.Context, member *OrgMember) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	members := m.orgMembers[member.OrganizationID]
	for _, existing := range members {
		if existing.UserID == member.UserID {
			return ErrMemberAlreadyExists
		}
	}

	user, err := m.authRepo.GetUserByID(ctx, member.UserID)
	if err != nil {
		return fmt.Errorf("user lookup: %w", err)
	}

	if member.ID == "" {
		member.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	member.CreatedAt = now
	member.UpdatedAt = now
	member.Username = user.Username
	member.DisplayName = user.DisplayName
	member.AvatarURL = user.AvatarURL
	member.Email = user.Email

	m.orgMembers[member.OrganizationID] = append(m.orgMembers[member.OrganizationID], member)
	return nil
}

func (m *memoryRepository) UpdateMemberRole(ctx context.Context, orgID, userID string, role OrgRole) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	members := m.orgMembers[orgID]
	for _, member := range members {
		if member.UserID == userID {
			member.Role = role
			member.UpdatedAt = time.Now().UTC()
			return nil
		}
	}
	return ErrMemberNotFound
}

func (m *memoryRepository) RemoveMember(ctx context.Context, orgID, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	members := m.orgMembers[orgID]
	var updated []*OrgMember
	found := false
	for _, member := range members {
		if member.UserID == userID {
			found = true
			continue
		}
		updated = append(updated, member)
	}
	if !found {
		return ErrMemberNotFound
	}
	m.orgMembers[orgID] = updated

	// Remove from all teams in this org as well
	for teamID, team := range m.teams {
		if team.OrganizationID == orgID {
			tMembers := m.teamMembers[teamID]
			var updatedT []*TeamMember
			for _, tm := range tMembers {
				if tm.UserID != userID {
					updatedT = append(updatedT, tm)
				}
			}
			m.teamMembers[teamID] = updatedT
		}
	}

	return nil
}

func (m *memoryRepository) CountOwners(ctx context.Context, orgID string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, member := range m.orgMembers[orgID] {
		if member.Role == OrgRoleOwner {
			count++
		}
	}
	return count, nil
}

func (m *memoryRepository) CreateTeam(ctx context.Context, team *Team) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, existing := range m.teams {
		if existing.OrganizationID == team.OrganizationID && strings.EqualFold(existing.Slug, team.Slug) {
			return ErrTeamSlugTaken
		}
	}

	if team.ID == "" {
		team.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	team.CreatedAt = now
	team.UpdatedAt = now
	if team.Privacy == "" {
		team.Privacy = "visible"
	}

	m.teams[team.ID] = team
	return nil
}

func (m *memoryRepository) GetTeamBySlug(ctx context.Context, orgID, slug string) (*Team, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, team := range m.teams {
		if team.OrganizationID == orgID && strings.EqualFold(team.Slug, slug) {
			copied := *team
			copied.MemberCount = len(m.teamMembers[team.ID])
			return &copied, nil
		}
	}
	return nil, ErrTeamNotFound
}

func (m *memoryRepository) GetTeamByID(ctx context.Context, teamID string) (*Team, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	team, exists := m.teams[teamID]
	if !exists {
		return nil, ErrTeamNotFound
	}
	copied := *team
	copied.MemberCount = len(m.teamMembers[teamID])
	return &copied, nil
}

func (m *memoryRepository) ListTeams(ctx context.Context, orgID string) ([]*Team, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*Team
	for _, team := range m.teams {
		if team.OrganizationID == orgID {
			copied := *team
			copied.MemberCount = len(m.teamMembers[team.ID])
			list = append(list, &copied)
		}
	}
	return list, nil
}

func (m *memoryRepository) UpdateTeam(ctx context.Context, team *Team) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.teams[team.ID]
	if !exists {
		return ErrTeamNotFound
	}

	existing.Name = team.Name
	existing.Description = team.Description
	if team.Privacy != "" {
		existing.Privacy = team.Privacy
	}
	existing.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *memoryRepository) DeleteTeam(ctx context.Context, teamID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.teams[teamID]; !exists {
		return ErrTeamNotFound
	}

	delete(m.teams, teamID)
	delete(m.teamMembers, teamID)
	delete(m.teamRepos, teamID)
	return nil
}

func (m *memoryRepository) GetTeamMember(ctx context.Context, teamID, userID string) (*TeamMember, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	members := m.teamMembers[teamID]
	for _, tm := range members {
		if tm.UserID == userID {
			copied := *tm
			return &copied, nil
		}
	}
	return nil, ErrMemberNotFound
}

func (m *memoryRepository) ListTeamMembers(ctx context.Context, teamID string) ([]*TeamMember, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	members := m.teamMembers[teamID]
	var list []*TeamMember
	for _, item := range members {
		copied := *item
		list = append(list, &copied)
	}
	return list, nil
}

func (m *memoryRepository) AddTeamMember(ctx context.Context, member *TeamMember) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	members := m.teamMembers[member.TeamID]
	for _, existing := range members {
		if existing.UserID == member.UserID {
			return ErrMemberAlreadyExists
		}
	}

	user, err := m.authRepo.GetUserByID(ctx, member.UserID)
	if err != nil {
		return fmt.Errorf("user lookup: %w", err)
	}

	if member.ID == "" {
		member.ID = uuid.New().String()
	}
	member.CreatedAt = time.Now().UTC()
	member.Username = user.Username
	member.DisplayName = user.DisplayName
	member.AvatarURL = user.AvatarURL

	m.teamMembers[member.TeamID] = append(m.teamMembers[member.TeamID], member)
	return nil
}

func (m *memoryRepository) UpdateTeamMemberRole(ctx context.Context, teamID, userID string, role TeamRole) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	members := m.teamMembers[teamID]
	for _, tm := range members {
		if tm.UserID == userID {
			tm.Role = role
			return nil
		}
	}
	return ErrMemberNotFound
}

func (m *memoryRepository) RemoveTeamMember(ctx context.Context, teamID, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	members := m.teamMembers[teamID]
	var updated []*TeamMember
	found := false
	for _, tm := range members {
		if tm.UserID == userID {
			found = true
			continue
		}
		updated = append(updated, tm)
	}
	if !found {
		return ErrMemberNotFound
	}
	m.teamMembers[teamID] = updated
	return nil
}

func (m *memoryRepository) GetUserTeamPermissions(ctx context.Context, orgID, userID, repoID string) ([]Permission, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var perms []Permission
	for teamID, members := range m.teamMembers {
		userInTeam := false
		for _, tm := range members {
			if tm.UserID == userID {
				userInTeam = true
				break
			}
		}
		if userInTeam {
			for _, tr := range m.teamRepos[teamID] {
				if tr.RepositoryID == repoID {
					perms = append(perms, tr.Permission)
				}
			}
		}
	}
	return perms, nil
}
