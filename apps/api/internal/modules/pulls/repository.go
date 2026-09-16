package pulls

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
	ErrPRNotFound      = errors.New("pull request not found")
	ErrReviewNotFound  = errors.New("pull request review not found")
	ErrCommentNotFound = errors.New("pull request comment not found")
)

type RepositoryStore interface {
	CreatePullRequest(ctx context.Context, pr *PullRequest) (*PullRequest, error)
	GetPRByNumber(ctx context.Context, repoID string, number int) (*PullRequest, error)
	GetPRByID(ctx context.Context, prID string) (*PullRequest, error)
	ListPullRequests(ctx context.Context, repoID string, filter PRFilter) ([]*PullRequest, int, int, int, error) // items, openC, closedC, mergedC, err
	UpdatePullRequest(ctx context.Context, pr *PullRequest) (*PullRequest, error)

	CreateReview(ctx context.Context, review *PullRequestReview) (*PullRequestReview, error)
	ListReviews(ctx context.Context, prID string) ([]*PullRequestReview, error)

	CreateComment(ctx context.Context, comment *PullRequestComment) (*PullRequestComment, error)
	ListComments(ctx context.Context, prID string) ([]*PullRequestComment, error)
	DeleteComment(ctx context.Context, commentID string) error
}

func NewRepositoryStore(db *database.DB, authRepo auth.Repository) RepositoryStore {
	if db.IsStandalone() {
		return NewMemoryRepositoryStore(authRepo)
	}
	return &sqlRepositoryStore{db: db, authRepo: authRepo}
}

// ============================================================================
// PostgreSQL Repository Store
// ============================================================================

type sqlRepositoryStore struct {
	db       *database.DB
	authRepo auth.Repository
}

func (s *sqlRepositoryStore) CreatePullRequest(ctx context.Context, pr *PullRequest) (*PullRequest, error) {
	if pr.ID == "" {
		pr.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	pr.CreatedAt = now
	pr.UpdatedAt = now
	if pr.State == "" {
		pr.State = StateOpen
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Atomic sequence per repository
	row := tx.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(number), 0) + 1 FROM pull_requests WHERE repository_id = $1
	`, pr.RepositoryID)
	if err := row.Scan(&pr.Number); err != nil {
		return nil, fmt.Errorf("generate PR number: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO pull_requests (
			id, repository_id, number, title, body, state, source_branch, target_branch,
			author_id, is_draft, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, pr.ID, pr.RepositoryID, pr.Number, pr.Title, pr.Body, pr.State, pr.SourceBranch, pr.TargetBranch,
		pr.AuthorID, pr.IsDraft, pr.CreatedAt, pr.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert PR: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.getPopulatedPR(ctx, pr.ID)
}

func (s *sqlRepositoryStore) getPopulatedPR(ctx context.Context, prID string) (*PullRequest, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT p.id, p.repository_id, p.number, p.title, p.body, p.state, p.source_branch, p.target_branch,
		       p.author_id, p.is_draft, p.merge_commit_sha, p.merged_by_id, p.merged_at,
		       p.closed_by_id, p.closed_at, p.created_at, p.updated_at,
		       (SELECT COUNT(*) FROM pull_request_comments WHERE pull_request_id = p.id) as comments_count
		FROM pull_requests p WHERE p.id = $1
	`, prID)

	var pr PullRequest
	err := row.Scan(&pr.ID, &pr.RepositoryID, &pr.Number, &pr.Title, &pr.Body, &pr.State, &pr.SourceBranch, &pr.TargetBranch,
		&pr.AuthorID, &pr.IsDraft, &pr.MergeCommitSHA, &pr.MergedByID, &pr.MergedAt,
		&pr.ClosedByID, &pr.ClosedAt, &pr.CreatedAt, &pr.UpdatedAt, &pr.CommentsCount)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPRNotFound
		}
		return nil, err
	}

	if pr.AuthorID != nil && s.authRepo != nil {
		if u, err := s.authRepo.GetUserByID(ctx, *pr.AuthorID); err == nil {
			pub := u.ToPublic()
			pr.Author = &pub
		}
	}
	if pr.MergedByID != nil && s.authRepo != nil {
		if u, err := s.authRepo.GetUserByID(ctx, *pr.MergedByID); err == nil {
			pub := u.ToPublic()
			pr.MergedBy = &pub
		}
	}
	if pr.ClosedByID != nil && s.authRepo != nil {
		if u, err := s.authRepo.GetUserByID(ctx, *pr.ClosedByID); err == nil {
			pub := u.ToPublic()
			pr.ClosedBy = &pub
		}
	}

	return &pr, nil
}

func (s *sqlRepositoryStore) GetPRByNumber(ctx context.Context, repoID string, number int) (*PullRequest, error) {
	var prID string
	err := s.db.QueryRowContext(ctx, `
		SELECT id FROM pull_requests WHERE repository_id = $1 AND number = $2
	`, repoID, number).Scan(&prID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPRNotFound
		}
		return nil, err
	}
	return s.getPopulatedPR(ctx, prID)
}

func (s *sqlRepositoryStore) GetPRByID(ctx context.Context, prID string) (*PullRequest, error) {
	return s.getPopulatedPR(ctx, prID)
}

func (s *sqlRepositoryStore) ListPullRequests(ctx context.Context, repoID string, filter PRFilter) ([]*PullRequest, int, int, int, error) {
	var openC, closedC, mergedC int
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pull_requests WHERE repository_id = $1 AND state = 'open'`, repoID).Scan(&openC)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pull_requests WHERE repository_id = $1 AND state = 'closed'`, repoID).Scan(&closedC)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pull_requests WHERE repository_id = $1 AND state = 'merged'`, repoID).Scan(&mergedC)

	query := `
		SELECT p.id
		FROM pull_requests p
		LEFT JOIN users u ON u.id = p.author_id
		WHERE p.repository_id = $1
	`
	args := []any{repoID}
	idx := 2

	if filter.State != "" && filter.State != "all" {
		query += fmt.Sprintf(" AND p.state = $%d", idx)
		args = append(args, filter.State)
		idx++
	}

	if filter.AuthorUsername != "" {
		query += fmt.Sprintf(" AND u.username = $%d", idx)
		args = append(args, filter.AuthorUsername)
		idx++
	}

	if filter.SourceBranch != "" {
		query += fmt.Sprintf(" AND p.source_branch = $%d", idx)
		args = append(args, filter.SourceBranch)
		idx++
	}

	if filter.TargetBranch != "" {
		query += fmt.Sprintf(" AND p.target_branch = $%d", idx)
		args = append(args, filter.TargetBranch)
		idx++
	}

	if filter.Query != "" {
		query += fmt.Sprintf(" AND (p.title ILIKE $%d OR p.body ILIKE $%d)", idx, idx)
		args = append(args, "%"+filter.Query+"%")
		idx++
	}

	query += " ORDER BY p.created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	defer rows.Close()

	var prIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			prIDs = append(prIDs, id)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, 0, 0, 0, err
	}

	var results []*PullRequest
	for _, id := range prIDs {
		if pr, err := s.getPopulatedPR(ctx, id); err == nil {
			results = append(results, pr)
		}
	}
	if results == nil {
		results = []*PullRequest{}
	}

	return results, openC, closedC, mergedC, nil
}

func (s *sqlRepositoryStore) UpdatePullRequest(ctx context.Context, pr *PullRequest) (*PullRequest, error) {
	pr.UpdatedAt = time.Now().UTC()

	res, err := s.db.ExecContext(ctx, `
		UPDATE pull_requests
		SET title = $1, body = $2, state = $3, target_branch = $4, is_draft = $5,
		    merge_commit_sha = $6, merged_by_id = $7, merged_at = $8,
		    closed_by_id = $9, closed_at = $10, updated_at = $11
		WHERE id = $12
	`, pr.Title, pr.Body, pr.State, pr.TargetBranch, pr.IsDraft,
		pr.MergeCommitSHA, pr.MergedByID, pr.MergedAt,
		pr.ClosedByID, pr.ClosedAt, pr.UpdatedAt, pr.ID)
	if err != nil {
		return nil, err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return nil, ErrPRNotFound
	}

	return s.getPopulatedPR(ctx, pr.ID)
}

func (s *sqlRepositoryStore) CreateReview(ctx context.Context, r *PullRequestReview) (*PullRequestReview, error) {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	r.CreatedAt = now
	r.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pull_request_reviews (id, pull_request_id, reviewer_id, state, body, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, r.ID, r.PullRequestID, r.ReviewerID, r.State, r.Body, r.CreatedAt, r.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if s.authRepo != nil {
		if u, err := s.authRepo.GetUserByID(ctx, r.ReviewerID); err == nil {
			pub := u.ToPublic()
			r.Reviewer = &pub
		}
	}
	return r, nil
}

func (s *sqlRepositoryStore) ListReviews(ctx context.Context, prID string) ([]*PullRequestReview, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, pull_request_id, reviewer_id, state, body, created_at, updated_at
		FROM pull_request_reviews WHERE pull_request_id = $1 ORDER BY created_at ASC
	`, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []*PullRequestReview
	for rows.Next() {
		var rev PullRequestReview
		if err := rows.Scan(&rev.ID, &rev.PullRequestID, &rev.ReviewerID, &rev.State, &rev.Body, &rev.CreatedAt, &rev.UpdatedAt); err == nil {
			if s.authRepo != nil {
				if u, err := s.authRepo.GetUserByID(ctx, rev.ReviewerID); err == nil {
					pub := u.ToPublic()
					rev.Reviewer = &pub
				}
			}
			reviews = append(reviews, &rev)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if reviews == nil {
		reviews = []*PullRequestReview{}
	}
	return reviews, nil
}

func (s *sqlRepositoryStore) CreateComment(ctx context.Context, c *PullRequestComment) (*PullRequestComment, error) {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	c.CreatedAt = now
	c.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pull_request_comments (id, pull_request_id, review_id, author_id, file_path, line_number, body, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, c.ID, c.PullRequestID, c.ReviewID, c.AuthorID, c.FilePath, c.LineNumber, c.Body, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if c.AuthorID != nil && s.authRepo != nil {
		if u, err := s.authRepo.GetUserByID(ctx, *c.AuthorID); err == nil {
			pub := u.ToPublic()
			c.Author = &pub
		}
	}
	return c, nil
}

func (s *sqlRepositoryStore) ListComments(ctx context.Context, prID string) ([]*PullRequestComment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, pull_request_id, review_id, author_id, file_path, line_number, body, created_at, updated_at
		FROM pull_request_comments WHERE pull_request_id = $1 ORDER BY created_at ASC
	`, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*PullRequestComment
	for rows.Next() {
		var c PullRequestComment
		if err := rows.Scan(&c.ID, &c.PullRequestID, &c.ReviewID, &c.AuthorID, &c.FilePath, &c.LineNumber, &c.Body, &c.CreatedAt, &c.UpdatedAt); err == nil {
			if c.AuthorID != nil && s.authRepo != nil {
				if u, err := s.authRepo.GetUserByID(ctx, *c.AuthorID); err == nil {
					pub := u.ToPublic()
					c.Author = &pub
				}
			}
			comments = append(comments, &c)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if comments == nil {
		comments = []*PullRequestComment{}
	}
	return comments, nil
}

func (s *sqlRepositoryStore) DeleteComment(ctx context.Context, commentID string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM pull_request_comments WHERE id = $1`, commentID)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return ErrCommentNotFound
	}
	return nil
}

// ============================================================================
// In-Memory Repository Store
// ============================================================================

type memoryRepositoryStore struct {
	mu        sync.RWMutex
	authRepo  auth.Repository
	prs       map[string]*PullRequest        // ID -> PR
	prNumbers map[string]int                 // repoID -> maxNumber
	reviews   map[string]*PullRequestReview  // ID -> Review
	comments  map[string]*PullRequestComment // ID -> Comment
}

func NewMemoryRepositoryStore(authRepo auth.Repository) RepositoryStore {
	return &memoryRepositoryStore{
		authRepo:  authRepo,
		prs:       make(map[string]*PullRequest),
		prNumbers: make(map[string]int),
		reviews:   make(map[string]*PullRequestReview),
		comments:  make(map[string]*PullRequestComment),
	}
}

func (m *memoryRepositoryStore) CreatePullRequest(ctx context.Context, pr *PullRequest) (*PullRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if pr.ID == "" {
		pr.ID = uuid.New().String()
	}
	m.prNumbers[pr.RepositoryID]++
	pr.Number = m.prNumbers[pr.RepositoryID]
	now := time.Now().UTC()
	pr.CreatedAt = now
	pr.UpdatedAt = now
	if pr.State == "" {
		pr.State = StateOpen
	}

	m.prs[pr.ID] = pr
	return m.populatePRLocked(ctx, pr), nil
}

func (m *memoryRepositoryStore) populatePRLocked(ctx context.Context, pr *PullRequest) *PullRequest {
	cp := *pr
	if cp.AuthorID != nil && m.authRepo != nil {
		if u, err := m.authRepo.GetUserByID(ctx, *cp.AuthorID); err == nil {
			pub := u.ToPublic()
			cp.Author = &pub
		}
	}
	if cp.MergedByID != nil && m.authRepo != nil {
		if u, err := m.authRepo.GetUserByID(ctx, *cp.MergedByID); err == nil {
			pub := u.ToPublic()
			cp.MergedBy = &pub
		}
	}
	if cp.ClosedByID != nil && m.authRepo != nil {
		if u, err := m.authRepo.GetUserByID(ctx, *cp.ClosedByID); err == nil {
			pub := u.ToPublic()
			cp.ClosedBy = &pub
		}
	}

	count := 0
	for _, c := range m.comments {
		if c.PullRequestID == cp.ID {
			count++
		}
	}
	cp.CommentsCount = count
	return &cp
}

func (m *memoryRepositoryStore) GetPRByNumber(ctx context.Context, repoID string, number int) (*PullRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, pr := range m.prs {
		if pr.RepositoryID == repoID && pr.Number == number {
			return m.populatePRLocked(ctx, pr), nil
		}
	}
	return nil, ErrPRNotFound
}

func (m *memoryRepositoryStore) GetPRByID(ctx context.Context, prID string) (*PullRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pr, ok := m.prs[prID]
	if !ok {
		return nil, ErrPRNotFound
	}
	return m.populatePRLocked(ctx, pr), nil
}

func (m *memoryRepositoryStore) ListPullRequests(ctx context.Context, repoID string, filter PRFilter) ([]*PullRequest, int, int, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var openC, closedC, mergedC int
	for _, pr := range m.prs {
		if pr.RepositoryID == repoID {
			switch pr.State {
			case StateOpen:
				openC++
			case StateClosed:
				closedC++
			case StateMerged:
				mergedC++
			}
		}
	}

	var results []*PullRequest
	for _, pr := range m.prs {
		if pr.RepositoryID != repoID {
			continue
		}

		if filter.State != "" && filter.State != "all" && string(pr.State) != filter.State {
			continue
		}

		if filter.SourceBranch != "" && pr.SourceBranch != filter.SourceBranch {
			continue
		}

		if filter.TargetBranch != "" && pr.TargetBranch != filter.TargetBranch {
			continue
		}

		if filter.AuthorUsername != "" && m.authRepo != nil {
			if pr.AuthorID == nil {
				continue
			}
			u, err := m.authRepo.GetUserByID(ctx, *pr.AuthorID)
			if err != nil || !strings.EqualFold(u.Username, filter.AuthorUsername) {
				continue
			}
		}

		if filter.Query != "" {
			q := strings.ToLower(filter.Query)
			if !strings.Contains(strings.ToLower(pr.Title), q) && !strings.Contains(strings.ToLower(pr.Body), q) {
				continue
			}
		}

		results = append(results, m.populatePRLocked(ctx, pr))
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})

	if results == nil {
		results = []*PullRequest{}
	}

	return results, openC, closedC, mergedC, nil
}

func (m *memoryRepositoryStore) UpdatePullRequest(ctx context.Context, pr *PullRequest) (*PullRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	curr, ok := m.prs[pr.ID]
	if !ok {
		return nil, ErrPRNotFound
	}

	curr.Title = pr.Title
	curr.Body = pr.Body
	curr.State = pr.State
	curr.TargetBranch = pr.TargetBranch
	curr.IsDraft = pr.IsDraft
	curr.MergeCommitSHA = pr.MergeCommitSHA
	curr.MergedByID = pr.MergedByID
	curr.MergedAt = pr.MergedAt
	curr.ClosedByID = pr.ClosedByID
	curr.ClosedAt = pr.ClosedAt
	curr.UpdatedAt = time.Now().UTC()

	return m.populatePRLocked(ctx, curr), nil
}

func (m *memoryRepositoryStore) CreateReview(ctx context.Context, review *PullRequestReview) (*PullRequestReview, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if review.ID == "" {
		review.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	review.CreatedAt = now
	review.UpdatedAt = now
	m.reviews[review.ID] = review

	cp := *review
	if m.authRepo != nil {
		if u, err := m.authRepo.GetUserByID(ctx, cp.ReviewerID); err == nil {
			pub := u.ToPublic()
			cp.Reviewer = &pub
		}
	}
	return &cp, nil
}

func (m *memoryRepositoryStore) ListReviews(ctx context.Context, prID string) ([]*PullRequestReview, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*PullRequestReview
	for _, r := range m.reviews {
		if r.PullRequestID == prID {
			cp := *r
			if m.authRepo != nil {
				if u, err := m.authRepo.GetUserByID(ctx, cp.ReviewerID); err == nil {
					pub := u.ToPublic()
					cp.Reviewer = &pub
				}
			}
			list = append(list, &cp)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.Before(list[j].CreatedAt)
	})
	if list == nil {
		list = []*PullRequestReview{}
	}
	return list, nil
}

func (m *memoryRepositoryStore) CreateComment(ctx context.Context, comment *PullRequestComment) (*PullRequestComment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if comment.ID == "" {
		comment.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	comment.CreatedAt = now
	comment.UpdatedAt = now
	m.comments[comment.ID] = comment

	cp := *comment
	if cp.AuthorID != nil && m.authRepo != nil {
		if u, err := m.authRepo.GetUserByID(ctx, *cp.AuthorID); err == nil {
			pub := u.ToPublic()
			cp.Author = &pub
		}
	}
	return &cp, nil
}

func (m *memoryRepositoryStore) ListComments(ctx context.Context, prID string) ([]*PullRequestComment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*PullRequestComment
	for _, c := range m.comments {
		if c.PullRequestID == prID {
			cp := *c
			if cp.AuthorID != nil && m.authRepo != nil {
				if u, err := m.authRepo.GetUserByID(ctx, *cp.AuthorID); err == nil {
					pub := u.ToPublic()
					cp.Author = &pub
				}
			}
			list = append(list, &cp)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.Before(list[j].CreatedAt)
	})
	if list == nil {
		list = []*PullRequestComment{}
	}
	return list, nil
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
