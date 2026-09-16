package issues

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"forgehub/apps/api/internal/database"
	"forgehub/apps/api/internal/modules/auth"

	"github.com/google/uuid"
)

var (
	ErrIssueNotFound     = errors.New("issue not found")
	ErrCommentNotFound   = errors.New("comment not found")
	ErrLabelNotFound     = errors.New("label not found")
	ErrLabelNameTaken    = errors.New("label with this name already exists in repository")
	ErrMilestoneNotFound = errors.New("milestone not found")
)

type RepositoryStore interface {
	// Issues
	CreateIssue(ctx context.Context, issue *Issue, labelIDs, assigneeIDs []string) (*Issue, error)
	GetIssueByNumber(ctx context.Context, repoID string, number int) (*IssueDetail, error)
	GetIssueByID(ctx context.Context, issueID string) (*Issue, error)
	ListIssues(ctx context.Context, repoID string, filter IssueFilter) ([]*Issue, int, int, error) // items, openCount, closedCount, err
	UpdateIssue(ctx context.Context, issue *Issue, labelIDs, assigneeIDs *[]string) (*Issue, error)
	DeleteIssue(ctx context.Context, issueID string) error

	// Comments
	CreateComment(ctx context.Context, comment *IssueComment) (*IssueComment, error)
	GetCommentByID(ctx context.Context, commentID string) (*IssueComment, error)
	UpdateComment(ctx context.Context, commentID, body string) (*IssueComment, error)
	DeleteComment(ctx context.Context, commentID string) error
	ListComments(ctx context.Context, issueID string) ([]*IssueComment, error)

	// Labels
	CreateLabel(ctx context.Context, label *Label) (*Label, error)
	GetLabelByID(ctx context.Context, labelID string) (*Label, error)
	GetLabelByName(ctx context.Context, repoID, name string) (*Label, error)
	ListLabels(ctx context.Context, repoID string) ([]*Label, error)
	UpdateLabel(ctx context.Context, label *Label) (*Label, error)
	DeleteLabel(ctx context.Context, labelID string) error

	// Milestones
	CreateMilestone(ctx context.Context, milestone *Milestone) (*Milestone, error)
	GetMilestoneByID(ctx context.Context, milestoneID string) (*Milestone, error)
	ListMilestones(ctx context.Context, repoID string) ([]*Milestone, error)
	UpdateMilestone(ctx context.Context, milestone *Milestone) (*Milestone, error)
	DeleteMilestone(ctx context.Context, milestoneID string) error
}

func NewRepositoryStore(db *database.DB, authRepo auth.Repository) RepositoryStore {
	if db.IsStandalone() {
		return NewMemoryRepositoryStore(authRepo)
	}
	return &sqlRepositoryStore{db: db, authRepo: authRepo}
}

// ============================================================================
// SQL Repository Store (PostgreSQL)
// ============================================================================

type sqlRepositoryStore struct {
	db       *database.DB
	authRepo auth.Repository
}

func (s *sqlRepositoryStore) CreateIssue(ctx context.Context, issue *Issue, labelIDs, assigneeIDs []string) (*Issue, error) {
	if issue.ID == "" {
		issue.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	issue.CreatedAt = now
	issue.UpdatedAt = now
	if issue.State == "" {
		issue.State = StateOpen
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Atomic sequence per repository
	row := tx.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(number), 0) + 1 FROM issues WHERE repository_id = $1
	`, issue.RepositoryID)
	if err := row.Scan(&issue.Number); err != nil {
		return nil, fmt.Errorf("generate issue number: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO issues (id, repository_id, number, title, body, state, author_id, milestone_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, issue.ID, issue.RepositoryID, issue.Number, issue.Title, issue.Body, issue.State, issue.AuthorID, issue.MilestoneID, issue.CreatedAt, issue.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert issue: %w", err)
	}

	// Insert labels
	for _, lid := range labelIDs {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO issue_labels (issue_id, label_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
		`, issue.ID, lid)
		if err != nil {
			return nil, fmt.Errorf("insert issue label: %w", err)
		}
	}

	// Insert assignees
	for _, uid := range assigneeIDs {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO issue_assignees (issue_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
		`, issue.ID, uid)
		if err != nil {
			return nil, fmt.Errorf("insert issue assignee: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.getPopulatedIssue(ctx, issue.ID)
}

func (s *sqlRepositoryStore) getPopulatedIssue(ctx context.Context, issueID string) (*Issue, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT i.id, i.repository_id, i.number, i.title, i.body, i.state, i.author_id, i.milestone_id,
		       i.created_at, i.updated_at, i.closed_at, i.closed_by_id,
		       (SELECT COUNT(*) FROM issue_comments WHERE issue_id = i.id) as comments_count
		FROM issues i WHERE i.id = $1
	`, issueID)

	var iss Issue
	err := row.Scan(&iss.ID, &iss.RepositoryID, &iss.Number, &iss.Title, &iss.Body, &iss.State,
		&iss.AuthorID, &iss.MilestoneID, &iss.CreatedAt, &iss.UpdatedAt, &iss.ClosedAt, &iss.ClosedByID, &iss.CommentsCount)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrIssueNotFound
		}
		return nil, err
	}

	// Author
	if iss.AuthorID != nil {
		if u, err := s.authRepo.GetUserByID(ctx, *iss.AuthorID); err == nil {
			pub := u.ToPublic()
			iss.Author = &pub
		}
	}

	// Closed By
	if iss.ClosedByID != nil {
		if u, err := s.authRepo.GetUserByID(ctx, *iss.ClosedByID); err == nil {
			pub := u.ToPublic()
			iss.ClosedBy = &pub
		}
	}

	// Milestone
	if iss.MilestoneID != nil {
		m, err := s.GetMilestoneByID(ctx, *iss.MilestoneID)
		if err == nil {
			iss.Milestone = m
		}
	}

	// Labels
	iss.Labels = []*Label{}
	labelRows, err := s.db.QueryContext(ctx, `
		SELECT l.id, l.repository_id, l.name, l.color, l.description, l.created_at
		FROM labels l
		INNER JOIN issue_labels il ON il.label_id = l.id
		WHERE il.issue_id = $1
		ORDER BY l.name ASC
	`, iss.ID)
	if err == nil {
		defer labelRows.Close()
		for labelRows.Next() {
			var l Label
			if err := labelRows.Scan(&l.ID, &l.RepositoryID, &l.Name, &l.Color, &l.Description, &l.CreatedAt); err == nil {
				iss.Labels = append(iss.Labels, &l)
			}
		}
	}

	// Assignees
	iss.Assignees = []*auth.PublicUser{}
	userRows, err := s.db.QueryContext(ctx, `
		SELECT user_id FROM issue_assignees WHERE issue_id = $1
	`, iss.ID)
	if err == nil {
		defer userRows.Close()
		for userRows.Next() {
			var uid string
			if err := userRows.Scan(&uid); err == nil {
				if u, err := s.authRepo.GetUserByID(ctx, uid); err == nil {
					pub := u.ToPublic()
					iss.Assignees = append(iss.Assignees, &pub)
				}
			}
		}
	}

	return &iss, nil
}

func (s *sqlRepositoryStore) GetIssueByNumber(ctx context.Context, repoID string, number int) (*IssueDetail, error) {
	var issueID string
	err := s.db.QueryRowContext(ctx, `SELECT id FROM issues WHERE repository_id = $1 AND number = $2`, repoID, number).Scan(&issueID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrIssueNotFound
		}
		return nil, err
	}

	iss, err := s.getPopulatedIssue(ctx, issueID)
	if err != nil {
		return nil, err
	}

	comments, err := s.ListComments(ctx, issueID)
	if err != nil {
		return nil, err
	}

	return &IssueDetail{
		Issue:    *iss,
		Comments: comments,
	}, nil
}

func (s *sqlRepositoryStore) GetIssueByID(ctx context.Context, issueID string) (*Issue, error) {
	return s.getPopulatedIssue(ctx, issueID)
}

func (s *sqlRepositoryStore) ListIssues(ctx context.Context, repoID string, filter IssueFilter) ([]*Issue, int, int, error) {
	// Counts
	var openCount, closedCount int
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM issues WHERE repository_id = $1 AND state = 'open'`, repoID).Scan(&openCount)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM issues WHERE repository_id = $1 AND state = 'closed'`, repoID).Scan(&closedCount)

	query := `
		SELECT DISTINCT i.id
		FROM issues i
		LEFT JOIN issue_labels il ON il.issue_id = i.id
		LEFT JOIN labels l ON l.id = il.label_id
		LEFT JOIN issue_assignees ia ON ia.issue_id = i.id
		LEFT JOIN users au ON au.id = i.author_id
		LEFT JOIN users asu ON asu.id = ia.user_id
		WHERE i.repository_id = $1
	`
	args := []any{repoID}
	idx := 2

	if filter.State != "" && filter.State != "all" {
		query += fmt.Sprintf(" AND i.state = $%d", idx)
		args = append(args, filter.State)
		idx++
	}

	if filter.LabelName != "" {
		query += fmt.Sprintf(" AND l.name = $%d", idx)
		args = append(args, filter.LabelName)
		idx++
	}

	if filter.MilestoneID != "" {
		query += fmt.Sprintf(" AND i.milestone_id = $%d", idx)
		args = append(args, filter.MilestoneID)
		idx++
	}

	if filter.AssigneeUsername != "" {
		query += fmt.Sprintf(" AND asu.username = $%d", idx)
		args = append(args, filter.AssigneeUsername)
		idx++
	}

	if filter.AuthorUsername != "" {
		query += fmt.Sprintf(" AND au.username = $%d", idx)
		args = append(args, filter.AuthorUsername)
		idx++
	}

	if filter.Query != "" {
		query += fmt.Sprintf(" AND (i.title ILIKE $%d OR i.body ILIKE $%d)", idx, idx)
		args = append(args, "%"+filter.Query+"%")
		idx++
	}

	query += " ORDER BY i.created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, 0, err
	}
	defer rows.Close()

	var issueIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			issueIDs = append(issueIDs, id)
		}
	}

	var results []*Issue
	for _, id := range issueIDs {
		if iss, err := s.getPopulatedIssue(ctx, id); err == nil {
			results = append(results, iss)
		}
	}

	if results == nil {
		results = []*Issue{}
	}

	return results, openCount, closedCount, nil
}

func (s *sqlRepositoryStore) UpdateIssue(ctx context.Context, issue *Issue, labelIDs, assigneeIDs *[]string) (*Issue, error) {
	issue.UpdatedAt = time.Now().UTC()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		UPDATE issues
		SET title = $1, body = $2, state = $3, milestone_id = $4, updated_at = $5, closed_at = $6, closed_by_id = $7
		WHERE id = $8
	`, issue.Title, issue.Body, issue.State, issue.MilestoneID, issue.UpdatedAt, issue.ClosedAt, issue.ClosedByID, issue.ID)
	if err != nil {
		return nil, err
	}

	if labelIDs != nil {
		_, _ = tx.ExecContext(ctx, `DELETE FROM issue_labels WHERE issue_id = $1`, issue.ID)
		for _, lid := range *labelIDs {
			_, _ = tx.ExecContext(ctx, `INSERT INTO issue_labels (issue_id, label_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, issue.ID, lid)
		}
	}

	if assigneeIDs != nil {
		_, _ = tx.ExecContext(ctx, `DELETE FROM issue_assignees WHERE issue_id = $1`, issue.ID)
		for _, uid := range *assigneeIDs {
			_, _ = tx.ExecContext(ctx, `INSERT INTO issue_assignees (issue_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, issue.ID, uid)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.getPopulatedIssue(ctx, issue.ID)
}

func (s *sqlRepositoryStore) DeleteIssue(ctx context.Context, issueID string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM issues WHERE id = $1`, issueID)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return ErrIssueNotFound
	}
	return nil
}

func (s *sqlRepositoryStore) CreateComment(ctx context.Context, c *IssueComment) (*IssueComment, error) {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	c.CreatedAt = now
	c.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO issue_comments (id, issue_id, author_id, body, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, c.ID, c.IssueID, c.AuthorID, c.Body, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if c.AuthorID != nil {
		if u, err := s.authRepo.GetUserByID(ctx, *c.AuthorID); err == nil {
			pub := u.ToPublic()
			c.Author = &pub
		}
	}

	return c, nil
}

func (s *sqlRepositoryStore) GetCommentByID(ctx context.Context, commentID string) (*IssueComment, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, issue_id, author_id, body, created_at, updated_at
		FROM issue_comments WHERE id = $1
	`, commentID)

	var c IssueComment
	if err := row.Scan(&c.ID, &c.IssueID, &c.AuthorID, &c.Body, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCommentNotFound
		}
		return nil, err
	}

	if c.AuthorID != nil {
		if u, err := s.authRepo.GetUserByID(ctx, *c.AuthorID); err == nil {
			pub := u.ToPublic()
			c.Author = &pub
		}
	}

	return &c, nil
}

func (s *sqlRepositoryStore) UpdateComment(ctx context.Context, commentID, body string) (*IssueComment, error) {
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx, `
		UPDATE issue_comments SET body = $1, updated_at = $2 WHERE id = $3
	`, body, now, commentID)
	if err != nil {
		return nil, err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return nil, ErrCommentNotFound
	}

	return s.GetCommentByID(ctx, commentID)
}

func (s *sqlRepositoryStore) DeleteComment(ctx context.Context, commentID string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM issue_comments WHERE id = $1`, commentID)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return ErrCommentNotFound
	}
	return nil
}

func (s *sqlRepositoryStore) ListComments(ctx context.Context, issueID string) ([]*IssueComment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, issue_id, author_id, body, created_at, updated_at
		FROM issue_comments WHERE issue_id = $1 ORDER BY created_at ASC
	`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*IssueComment
	for rows.Next() {
		var c IssueComment
		if err := rows.Scan(&c.ID, &c.IssueID, &c.AuthorID, &c.Body, &c.CreatedAt, &c.UpdatedAt); err == nil {
			if c.AuthorID != nil {
				if u, err := s.authRepo.GetUserByID(ctx, *c.AuthorID); err == nil {
					pub := u.ToPublic()
					c.Author = &pub
				}
			}
			comments = append(comments, &c)
		}
	}

	if comments == nil {
		comments = []*IssueComment{}
	}

	return comments, nil
}

func (s *sqlRepositoryStore) CreateLabel(ctx context.Context, label *Label) (*Label, error) {
	if label.ID == "" {
		label.ID = uuid.New().String()
	}
	label.CreatedAt = time.Now().UTC()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO labels (id, repository_id, name, color, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, label.ID, label.RepositoryID, label.Name, label.Color, label.Description, label.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "uq_repo_label") {
			return nil, ErrLabelNameTaken
		}
		return nil, err
	}

	return label, nil
}

func (s *sqlRepositoryStore) GetLabelByID(ctx context.Context, labelID string) (*Label, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, repository_id, name, color, description, created_at
		FROM labels WHERE id = $1
	`, labelID)

	var l Label
	if err := row.Scan(&l.ID, &l.RepositoryID, &l.Name, &l.Color, &l.Description, &l.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrLabelNotFound
		}
		return nil, err
	}
	return &l, nil
}

func (s *sqlRepositoryStore) GetLabelByName(ctx context.Context, repoID, name string) (*Label, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, repository_id, name, color, description, created_at
		FROM labels WHERE repository_id = $1 AND name = $2
	`, repoID, name)

	var l Label
	if err := row.Scan(&l.ID, &l.RepositoryID, &l.Name, &l.Color, &l.Description, &l.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrLabelNotFound
		}
		return nil, err
	}
	return &l, nil
}

func (s *sqlRepositoryStore) ListLabels(ctx context.Context, repoID string) ([]*Label, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, repository_id, name, color, description, created_at
		FROM labels WHERE repository_id = $1 ORDER BY name ASC
	`, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var labels []*Label
	for rows.Next() {
		var l Label
		if err := rows.Scan(&l.ID, &l.RepositoryID, &l.Name, &l.Color, &l.Description, &l.CreatedAt); err == nil {
			labels = append(labels, &l)
		}
	}
	if labels == nil {
		labels = []*Label{}
	}
	return labels, nil
}

func (s *sqlRepositoryStore) UpdateLabel(ctx context.Context, label *Label) (*Label, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE labels SET name = $1, color = $2, description = $3 WHERE id = $4
	`, label.Name, label.Color, label.Description, label.ID)
	if err != nil {
		if strings.Contains(err.Error(), "uq_repo_label") {
			return nil, ErrLabelNameTaken
		}
		return nil, err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return nil, ErrLabelNotFound
	}
	return label, nil
}

func (s *sqlRepositoryStore) DeleteLabel(ctx context.Context, labelID string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM labels WHERE id = $1`, labelID)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return ErrLabelNotFound
	}
	return nil
}

func (s *sqlRepositoryStore) CreateMilestone(ctx context.Context, m *Milestone) (*Milestone, error) {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	m.CreatedAt = now
	m.UpdatedAt = now
	if m.State == "" {
		m.State = StateOpen
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO milestones (id, repository_id, title, description, state, due_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, m.ID, m.RepositoryID, m.Title, m.Description, m.State, m.DueDate, m.CreatedAt, m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (s *sqlRepositoryStore) GetMilestoneByID(ctx context.Context, milestoneID string) (*Milestone, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT m.id, m.repository_id, m.title, m.description, m.state, m.due_date, m.created_at, m.updated_at, m.closed_at,
		       (SELECT COUNT(*) FROM issues WHERE milestone_id = m.id AND state = 'open') as open_count,
		       (SELECT COUNT(*) FROM issues WHERE milestone_id = m.id AND state = 'closed') as closed_count
		FROM milestones m WHERE m.id = $1
	`, milestoneID)

	var m Milestone
	if err := row.Scan(&m.ID, &m.RepositoryID, &m.Title, &m.Description, &m.State, &m.DueDate, &m.CreatedAt, &m.UpdatedAt, &m.ClosedAt,
		&m.OpenIssuesCount, &m.ClosedIssuesCount); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMilestoneNotFound
		}
		return nil, err
	}
	return &m, nil
}

func (s *sqlRepositoryStore) ListMilestones(ctx context.Context, repoID string) ([]*Milestone, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT m.id, m.repository_id, m.title, m.description, m.state, m.due_date, m.created_at, m.updated_at, m.closed_at,
		       (SELECT COUNT(*) FROM issues WHERE milestone_id = m.id AND state = 'open') as open_count,
		       (SELECT COUNT(*) FROM issues WHERE milestone_id = m.id AND state = 'closed') as closed_count
		FROM milestones m WHERE m.repository_id = $1 ORDER BY m.created_at DESC
	`, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var milestones []*Milestone
	for rows.Next() {
		var m Milestone
		if err := rows.Scan(&m.ID, &m.RepositoryID, &m.Title, &m.Description, &m.State, &m.DueDate, &m.CreatedAt, &m.UpdatedAt, &m.ClosedAt,
			&m.OpenIssuesCount, &m.ClosedIssuesCount); err == nil {
			milestones = append(milestones, &m)
		}
	}
	if milestones == nil {
		milestones = []*Milestone{}
	}
	return milestones, nil
}

func (s *sqlRepositoryStore) UpdateMilestone(ctx context.Context, m *Milestone) (*Milestone, error) {
	m.UpdatedAt = time.Now().UTC()
	res, err := s.db.ExecContext(ctx, `
		UPDATE milestones
		SET title = $1, description = $2, state = $3, due_date = $4, updated_at = $5, closed_at = $6
		WHERE id = $7
	`, m.Title, m.Description, m.State, m.DueDate, m.UpdatedAt, m.ClosedAt, m.ID)
	if err != nil {
		return nil, err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return nil, ErrMilestoneNotFound
	}
	return s.GetMilestoneByID(ctx, m.ID)
}

func (s *sqlRepositoryStore) DeleteMilestone(ctx context.Context, milestoneID string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM milestones WHERE id = $1`, milestoneID)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return ErrMilestoneNotFound
	}
	return nil
}

// ============================================================================
// Memory Repository Store (High performance for tests & zero-dep local dev)
// ============================================================================

type memoryRepositoryStore struct {
	mu           sync.RWMutex
	authRepo     auth.Repository
	issues       map[string]*Issue             // ID -> Issue
	issueNumbers map[string]int                // repoID -> maxNumber
	issueLabels  map[string]map[string]bool    // issueID -> labelID -> bool
	issueUsers   map[string]map[string]bool    // issueID -> userID -> bool
	comments     map[string]*IssueComment      // commentID -> Comment
	labels       map[string]*Label             // labelID -> Label
	milestones   map[string]*Milestone         // milestoneID -> Milestone
}

func NewMemoryRepositoryStore(authRepo auth.Repository) RepositoryStore {
	return &memoryRepositoryStore{
		authRepo:     authRepo,
		issues:       make(map[string]*Issue),
		issueNumbers: make(map[string]int),
		issueLabels:  make(map[string]map[string]bool),
		issueUsers:   make(map[string]map[string]bool),
		comments:     make(map[string]*IssueComment),
		labels:       make(map[string]*Label),
		milestones:   make(map[string]*Milestone),
	}
}

func (m *memoryRepositoryStore) CreateIssue(ctx context.Context, issue *Issue, labelIDs, assigneeIDs []string) (*Issue, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if issue.ID == "" {
		issue.ID = uuid.New().String()
	}
	m.issueNumbers[issue.RepositoryID]++
	issue.Number = m.issueNumbers[issue.RepositoryID]
	now := time.Now().UTC()
	issue.CreatedAt = now
	issue.UpdatedAt = now
	if issue.State == "" {
		issue.State = StateOpen
	}

	m.issues[issue.ID] = issue

	m.issueLabels[issue.ID] = make(map[string]bool)
	for _, lid := range labelIDs {
		m.issueLabels[issue.ID][lid] = true
	}

	m.issueUsers[issue.ID] = make(map[string]bool)
	for _, uid := range assigneeIDs {
		m.issueUsers[issue.ID][uid] = true
	}

	return m.populateIssueLocked(ctx, issue), nil
}

func (m *memoryRepositoryStore) populateIssueLocked(ctx context.Context, issue *Issue) *Issue {
	cp := *issue

	// Author
	if cp.AuthorID != nil && m.authRepo != nil {
		if u, err := m.authRepo.GetUserByID(ctx, *cp.AuthorID); err == nil {
			pub := u.ToPublic()
			cp.Author = &pub
		}
	}

	// Closed By
	if cp.ClosedByID != nil && m.authRepo != nil {
		if u, err := m.authRepo.GetUserByID(ctx, *cp.ClosedByID); err == nil {
			pub := u.ToPublic()
			cp.ClosedBy = &pub
		}
	}

	// Milestone
	if cp.MilestoneID != nil {
		if ms, ok := m.milestones[*cp.MilestoneID]; ok {
			cp.Milestone = ms
		}
	}

	// Labels
	cp.Labels = []*Label{}
	if lids, ok := m.issueLabels[cp.ID]; ok {
		for lid := range lids {
			if l, exists := m.labels[lid]; exists {
				cp.Labels = append(cp.Labels, l)
			}
		}
	}
	sort.Slice(cp.Labels, func(i, j int) bool {
		return cp.Labels[i].Name < cp.Labels[j].Name
	})

	// Assignees
	cp.Assignees = []*auth.PublicUser{}
	if uids, ok := m.issueUsers[cp.ID]; ok && m.authRepo != nil {
		for uid := range uids {
			if u, err := m.authRepo.GetUserByID(ctx, uid); err == nil {
				pub := u.ToPublic()
				cp.Assignees = append(cp.Assignees, &pub)
			}
		}
	}

	// Comments count
	count := 0
	for _, c := range m.comments {
		if c.IssueID == cp.ID {
			count++
		}
	}
	cp.CommentsCount = count

	return &cp
}

func (m *memoryRepositoryStore) GetIssueByNumber(ctx context.Context, repoID string, number int) (*IssueDetail, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var target *Issue
	for _, iss := range m.issues {
		if iss.RepositoryID == repoID && iss.Number == number {
			target = iss
			break
		}
	}
	if target == nil {
		return nil, ErrIssueNotFound
	}

	populated := m.populateIssueLocked(ctx, target)

	var comments []*IssueComment
	for _, c := range m.comments {
		if c.IssueID == target.ID {
			cCopy := *c
			if cCopy.AuthorID != nil && m.authRepo != nil {
				if u, err := m.authRepo.GetUserByID(ctx, *cCopy.AuthorID); err == nil {
					pub := u.ToPublic()
					cCopy.Author = &pub
				}
			}
			comments = append(comments, &cCopy)
		}
	}
	sort.Slice(comments, func(i, j int) bool {
		return comments[i].CreatedAt.Before(comments[j].CreatedAt)
	})

	return &IssueDetail{
		Issue:    *populated,
		Comments: comments,
	}, nil
}

func (m *memoryRepositoryStore) GetIssueByID(ctx context.Context, issueID string) (*Issue, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	target, ok := m.issues[issueID]
	if !ok {
		return nil, ErrIssueNotFound
	}
	return m.populateIssueLocked(ctx, target), nil
}

func (m *memoryRepositoryStore) ListIssues(ctx context.Context, repoID string, filter IssueFilter) ([]*Issue, int, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var openCount, closedCount int
	for _, iss := range m.issues {
		if iss.RepositoryID == repoID {
			if iss.State == StateOpen {
				openCount++
			} else if iss.State == StateClosed {
				closedCount++
			}
		}
	}

	var results []*Issue
	for _, iss := range m.issues {
		if iss.RepositoryID != repoID {
			continue
		}

		if filter.State != "" && filter.State != "all" && string(iss.State) != filter.State {
			continue
		}

		if filter.MilestoneID != "" {
			if iss.MilestoneID == nil || *iss.MilestoneID != filter.MilestoneID {
				continue
			}
		}

		if filter.LabelName != "" {
			hasLabel := false
			if lids, ok := m.issueLabels[iss.ID]; ok {
				for lid := range lids {
					if l, exists := m.labels[lid]; exists && strings.EqualFold(l.Name, filter.LabelName) {
						hasLabel = true
						break
					}
				}
			}
			if !hasLabel {
				continue
			}
		}

		if filter.AssigneeUsername != "" && m.authRepo != nil {
			hasUser := false
			if uids, ok := m.issueUsers[iss.ID]; ok {
				for uid := range uids {
					if u, err := m.authRepo.GetUserByID(ctx, uid); err == nil && strings.EqualFold(u.Username, filter.AssigneeUsername) {
						hasUser = true
						break
					}
				}
			}
			if !hasUser {
				continue
			}
		}

		if filter.AuthorUsername != "" && m.authRepo != nil {
			if iss.AuthorID == nil {
				continue
			}
			u, err := m.authRepo.GetUserByID(ctx, *iss.AuthorID)
			if err != nil || !strings.EqualFold(u.Username, filter.AuthorUsername) {
				continue
			}
		}

		if filter.Query != "" {
			q := strings.ToLower(filter.Query)
			if !strings.Contains(strings.ToLower(iss.Title), q) && !strings.Contains(strings.ToLower(iss.Body), q) {
				continue
			}
		}

		results = append(results, m.populateIssueLocked(ctx, iss))
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})

	if results == nil {
		results = []*Issue{}
	}

	return results, openCount, closedCount, nil
}

func (m *memoryRepositoryStore) UpdateIssue(ctx context.Context, issue *Issue, labelIDs, assigneeIDs *[]string) (*Issue, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	curr, ok := m.issues[issue.ID]
	if !ok {
		return nil, ErrIssueNotFound
	}

	curr.Title = issue.Title
	curr.Body = issue.Body
	curr.State = issue.State
	curr.MilestoneID = issue.MilestoneID
	curr.ClosedAt = issue.ClosedAt
	curr.ClosedByID = issue.ClosedByID
	curr.UpdatedAt = time.Now().UTC()

	if labelIDs != nil {
		m.issueLabels[issue.ID] = make(map[string]bool)
		for _, lid := range *labelIDs {
			m.issueLabels[issue.ID][lid] = true
		}
	}

	if assigneeIDs != nil {
		m.issueUsers[issue.ID] = make(map[string]bool)
		for _, uid := range *assigneeIDs {
			m.issueUsers[issue.ID][uid] = true
		}
	}

	return m.populateIssueLocked(ctx, curr), nil
}

func (m *memoryRepositoryStore) DeleteIssue(ctx context.Context, issueID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.issues[issueID]; !ok {
		return ErrIssueNotFound
	}
	delete(m.issues, issueID)
	delete(m.issueLabels, issueID)
	delete(m.issueUsers, issueID)

	for cid, c := range m.comments {
		if c.IssueID == issueID {
			delete(m.comments, cid)
		}
	}
	return nil
}

func (m *memoryRepositoryStore) CreateComment(ctx context.Context, comment *IssueComment) (*IssueComment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if comment.ID == "" {
		comment.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	comment.CreatedAt = now
	comment.UpdatedAt = now

	m.comments[comment.ID] = comment

	cCopy := *comment
	if cCopy.AuthorID != nil && m.authRepo != nil {
		if u, err := m.authRepo.GetUserByID(ctx, *cCopy.AuthorID); err == nil {
			pub := u.ToPublic()
			cCopy.Author = &pub
		}
	}
	return &cCopy, nil
}

func (m *memoryRepositoryStore) GetCommentByID(ctx context.Context, commentID string) (*IssueComment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	c, ok := m.comments[commentID]
	if !ok {
		return nil, ErrCommentNotFound
	}
	cCopy := *c
	if cCopy.AuthorID != nil && m.authRepo != nil {
		if u, err := m.authRepo.GetUserByID(ctx, *cCopy.AuthorID); err == nil {
			pub := u.ToPublic()
			cCopy.Author = &pub
		}
	}
	return &cCopy, nil
}

func (m *memoryRepositoryStore) UpdateComment(ctx context.Context, commentID, body string) (*IssueComment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	c, ok := m.comments[commentID]
	if !ok {
		return nil, ErrCommentNotFound
	}
	c.Body = body
	c.UpdatedAt = time.Now().UTC()

	cCopy := *c
	if cCopy.AuthorID != nil && m.authRepo != nil {
		if u, err := m.authRepo.GetUserByID(ctx, *cCopy.AuthorID); err == nil {
			pub := u.ToPublic()
			cCopy.Author = &pub
		}
	}
	return &cCopy, nil
}

func (m *memoryRepositoryStore) DeleteComment(ctx context.Context, commentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.comments[commentID]; !ok {
		return ErrCommentNotFound
	}
	delete(m.comments, commentID)
	return nil
}

func (m *memoryRepositoryStore) ListComments(ctx context.Context, issueID string) ([]*IssueComment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*IssueComment
	for _, c := range m.comments {
		if c.IssueID == issueID {
			cCopy := *c
			if cCopy.AuthorID != nil && m.authRepo != nil {
				if u, err := m.authRepo.GetUserByID(ctx, *cCopy.AuthorID); err == nil {
					pub := u.ToPublic()
					cCopy.Author = &pub
				}
			}
			list = append(list, &cCopy)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.Before(list[j].CreatedAt)
	})
	if list == nil {
		list = []*IssueComment{}
	}
	return list, nil
}

func (m *memoryRepositoryStore) CreateLabel(ctx context.Context, label *Label) (*Label, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, l := range m.labels {
		if l.RepositoryID == label.RepositoryID && strings.EqualFold(l.Name, label.Name) {
			return nil, ErrLabelNameTaken
		}
	}

	if label.ID == "" {
		label.ID = uuid.New().String()
	}
	label.CreatedAt = time.Now().UTC()
	m.labels[label.ID] = label
	return label, nil
}

func (m *memoryRepositoryStore) GetLabelByID(ctx context.Context, labelID string) (*Label, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	l, ok := m.labels[labelID]
	if !ok {
		return nil, ErrLabelNotFound
	}
	return l, nil
}

func (m *memoryRepositoryStore) GetLabelByName(ctx context.Context, repoID, name string) (*Label, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, l := range m.labels {
		if l.RepositoryID == repoID && strings.EqualFold(l.Name, name) {
			return l, nil
		}
	}
	return nil, ErrLabelNotFound
}

func (m *memoryRepositoryStore) ListLabels(ctx context.Context, repoID string) ([]*Label, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*Label
	for _, l := range m.labels {
		if l.RepositoryID == repoID {
			list = append(list, l)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	if list == nil {
		list = []*Label{}
	}
	return list, nil
}

func (m *memoryRepositoryStore) UpdateLabel(ctx context.Context, label *Label) (*Label, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	curr, ok := m.labels[label.ID]
	if !ok {
		return nil, ErrLabelNotFound
	}

	for _, l := range m.labels {
		if l.ID != label.ID && l.RepositoryID == curr.RepositoryID && strings.EqualFold(l.Name, label.Name) {
			return nil, ErrLabelNameTaken
		}
	}

	curr.Name = label.Name
	curr.Color = label.Color
	curr.Description = label.Description
	return curr, nil
}

func (m *memoryRepositoryStore) DeleteLabel(ctx context.Context, labelID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.labels[labelID]; !ok {
		return ErrLabelNotFound
	}
	delete(m.labels, labelID)

	for _, lids := range m.issueLabels {
		delete(lids, labelID)
	}
	return nil
}

func (m *memoryRepositoryStore) CreateMilestone(ctx context.Context, milestone *Milestone) (*Milestone, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if milestone.ID == "" {
		milestone.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	milestone.CreatedAt = now
	milestone.UpdatedAt = now
	if milestone.State == "" {
		milestone.State = StateOpen
	}

	m.milestones[milestone.ID] = milestone
	return milestone, nil
}

func (m *memoryRepositoryStore) GetMilestoneByID(ctx context.Context, milestoneID string) (*Milestone, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ms, ok := m.milestones[milestoneID]
	if !ok {
		return nil, ErrMilestoneNotFound
	}

	cp := *ms
	openC, closedC := 0, 0
	for _, iss := range m.issues {
		if iss.MilestoneID != nil && *iss.MilestoneID == milestoneID {
			if iss.State == StateOpen {
				openC++
			} else {
				closedC++
			}
		}
	}
	cp.OpenIssuesCount = openC
	cp.ClosedIssuesCount = closedC

	return &cp, nil
}

func (m *memoryRepositoryStore) ListMilestones(ctx context.Context, repoID string) ([]*Milestone, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*Milestone
	for _, ms := range m.milestones {
		if ms.RepositoryID == repoID {
			cp := *ms
			openC, closedC := 0, 0
			for _, iss := range m.issues {
				if iss.MilestoneID != nil && *iss.MilestoneID == ms.ID {
					if iss.State == StateOpen {
						openC++
					} else {
						closedC++
					}
				}
			}
			cp.OpenIssuesCount = openC
			cp.ClosedIssuesCount = closedC
			list = append(list, &cp)
		}
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})
	if list == nil {
		list = []*Milestone{}
	}
	return list, nil
}

func (m *memoryRepositoryStore) UpdateMilestone(ctx context.Context, milestone *Milestone) (*Milestone, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	curr, ok := m.milestones[milestone.ID]
	if !ok {
		return nil, ErrMilestoneNotFound
	}

	curr.Title = milestone.Title
	curr.Description = milestone.Description
	curr.State = milestone.State
	curr.DueDate = milestone.DueDate
	curr.ClosedAt = milestone.ClosedAt
	curr.UpdatedAt = time.Now().UTC()

	return curr, nil
}

func (m *memoryRepositoryStore) DeleteMilestone(ctx context.Context, milestoneID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.milestones[milestoneID]; !ok {
		return ErrMilestoneNotFound
	}
	delete(m.milestones, milestoneID)

	for _, iss := range m.issues {
		if iss.MilestoneID != nil && *iss.MilestoneID == milestoneID {
			iss.MilestoneID = nil
		}
	}
	return nil
}
