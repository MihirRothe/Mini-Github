package issues

import (
	"context"
	"errors"
	"strings"
	"time"

	"forgehub/apps/api/internal/modules/auth"
	"forgehub/apps/api/internal/modules/orgs"
	"forgehub/apps/api/internal/modules/repos"
)

var (
	ErrAccessDenied       = errors.New("access denied: insufficient permissions")
	ErrEmptyTitle         = errors.New("issue title cannot be empty")
	ErrEmptyComment       = errors.New("comment body cannot be empty")
	ErrEmptyLabelName     = errors.New("label name cannot be empty")
	ErrEmptyMilestoneTitle = errors.New("milestone title cannot be empty")
)

type Service interface {
	// Issues
	CreateIssue(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, req CreateIssueRequest) (*Issue, error)
	GetIssue(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, number int) (*IssueDetail, error)
	ListIssues(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, filter IssueFilter) ([]*Issue, int, int, error)
	UpdateIssue(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, number int, req UpdateIssueRequest) (*Issue, error)
	DeleteIssue(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, number int) error

	// Comments
	CreateComment(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, number int, req CreateCommentRequest) (*IssueComment, error)
	UpdateComment(ctx context.Context, currentUserID string, isSiteAdmin bool, commentID string, req UpdateCommentRequest) (*IssueComment, error)
	DeleteComment(ctx context.Context, currentUserID string, isSiteAdmin bool, commentID string) error

	// Labels
	CreateLabel(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, req CreateLabelRequest) (*Label, error)
	ListLabels(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string) ([]*Label, error)
	UpdateLabel(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, labelID string, req UpdateLabelRequest) (*Label, error)
	DeleteLabel(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, labelID string) error

	// Milestones
	CreateMilestone(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, req CreateMilestoneRequest) (*Milestone, error)
	ListMilestones(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string) ([]*Milestone, error)
	UpdateMilestone(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, milestoneID string, req UpdateMilestoneRequest) (*Milestone, error)
	DeleteMilestone(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, milestoneID string) error

	// Helpers
	EnsureDefaultLabels(ctx context.Context, repoID string) error
}

type service struct {
	store    RepositoryStore
	reposSvc repos.Service
	authRepo auth.Repository
}

func NewService(store RepositoryStore, reposSvc repos.Service, authRepo auth.Repository) Service {
	return &service{
		store:    store,
		reposSvc: reposSvc,
		authRepo: authRepo,
	}
}

func (s *service) getRepoAndPermission(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string) (*repos.Repository, orgs.Permission, error) {
	repo, err := s.reposSvc.GetRepository(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, orgs.PermNone, err
	}
	perm, err := s.reposSvc.ResolveUserPermission(ctx, currentUserID, isSiteAdmin, repo)
	if err != nil {
		return nil, orgs.PermNone, err
	}
	return repo, perm, nil
}

func (s *service) CreateIssue(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, req CreateIssueRequest) (*Issue, error) {
	if currentUserID == "" {
		return nil, ErrAccessDenied
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, ErrEmptyTitle
	}

	repo, perm, err := s.getRepoAndPermission(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, err
	}
	// Any user with read permission on the repo can create an issue
	if !perm.Includes(orgs.PermRead) {
		return nil, ErrAccessDenied
	}

	// Auto-seed default labels if none exist yet
	_ = s.EnsureDefaultLabels(ctx, repo.ID)

	issue := &Issue{
		RepositoryID: repo.ID,
		Title:        title,
		Body:         strings.TrimSpace(req.Body),
		State:        StateOpen,
		AuthorID:     &currentUserID,
		MilestoneID:  req.MilestoneID,
	}

	// Only triage+ can assign or add labels on creation
	var labels []string
	var assignees []string
	if perm.Includes(orgs.PermTriage) {
		labels = req.LabelIDs
		assignees = req.AssigneeIDs
	}

	return s.store.CreateIssue(ctx, issue, labels, assignees)
}

func (s *service) GetIssue(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, number int) (*IssueDetail, error) {
	repo, perm, err := s.getRepoAndPermission(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, err
	}
	if !perm.Includes(orgs.PermRead) {
		return nil, ErrAccessDenied
	}

	return s.store.GetIssueByNumber(ctx, repo.ID, number)
}

func (s *service) ListIssues(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, filter IssueFilter) ([]*Issue, int, int, error) {
	repo, perm, err := s.getRepoAndPermission(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, 0, 0, err
	}
	if !perm.Includes(orgs.PermRead) {
		return nil, 0, 0, ErrAccessDenied
	}

	return s.store.ListIssues(ctx, repo.ID, filter)
}

func (s *service) UpdateIssue(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, number int, req UpdateIssueRequest) (*Issue, error) {
	if currentUserID == "" {
		return nil, ErrAccessDenied
	}

	repo, perm, err := s.getRepoAndPermission(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, err
	}

	issueDetail, err := s.store.GetIssueByNumber(ctx, repo.ID, number)
	if err != nil {
		return nil, err
	}

	isAuthor := issueDetail.AuthorID != nil && *issueDetail.AuthorID == currentUserID
	isTriage := perm.Includes(orgs.PermTriage)

	// Editing title/body: author or triage+
	if req.Title != nil || req.Body != nil {
		if !isAuthor && !isTriage {
			return nil, ErrAccessDenied
		}
		if req.Title != nil {
			trimmed := strings.TrimSpace(*req.Title)
			if trimmed == "" {
				return nil, ErrEmptyTitle
			}
			issueDetail.Title = trimmed
		}
		if req.Body != nil {
			issueDetail.Body = strings.TrimSpace(*req.Body)
		}
	}

	// Changing state (close/reopen): author or triage+
	if req.State != nil {
		if !isAuthor && !isTriage {
			return nil, ErrAccessDenied
		}
		if *req.State == StateClosed && issueDetail.State != StateClosed {
			issueDetail.State = StateClosed
			now := time.Now().UTC()
			issueDetail.ClosedAt = &now
			issueDetail.ClosedByID = &currentUserID
		} else if *req.State == StateOpen && issueDetail.State != StateOpen {
			issueDetail.State = StateOpen
			issueDetail.ClosedAt = nil
			issueDetail.ClosedByID = nil
		}
	}

	// Milestone, Labels, Assignees: triage+ only
	if req.MilestoneID != nil || req.LabelIDs != nil || req.AssigneeIDs != nil {
		if !isTriage {
			return nil, ErrAccessDenied
		}
		if req.MilestoneID != nil {
			if *req.MilestoneID == "" {
				issueDetail.MilestoneID = nil
			} else {
				issueDetail.MilestoneID = req.MilestoneID
			}
		}
	}

	return s.store.UpdateIssue(ctx, &issueDetail.Issue, req.LabelIDs, req.AssigneeIDs)
}

func (s *service) DeleteIssue(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, number int) error {
	if currentUserID == "" {
		return ErrAccessDenied
	}
	repo, perm, err := s.getRepoAndPermission(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return err
	}
	// Deleting issues requires maintain or admin
	if !perm.Includes(orgs.PermMaintain) {
		return ErrAccessDenied
	}

	issueDetail, err := s.store.GetIssueByNumber(ctx, repo.ID, number)
	if err != nil {
		return err
	}
	return s.store.DeleteIssue(ctx, issueDetail.ID)
}

func (s *service) CreateComment(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, number int, req CreateCommentRequest) (*IssueComment, error) {
	if currentUserID == "" {
		return nil, ErrAccessDenied
	}
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return nil, ErrEmptyComment
	}

	repo, perm, err := s.getRepoAndPermission(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, err
	}
	if !perm.Includes(orgs.PermRead) {
		return nil, ErrAccessDenied
	}

	issueDetail, err := s.store.GetIssueByNumber(ctx, repo.ID, number)
	if err != nil {
		return nil, err
	}

	comment := &IssueComment{
		IssueID:  issueDetail.ID,
		AuthorID: &currentUserID,
		Body:     body,
	}
	return s.store.CreateComment(ctx, comment)
}

func (s *service) UpdateComment(ctx context.Context, currentUserID string, isSiteAdmin bool, commentID string, req UpdateCommentRequest) (*IssueComment, error) {
	if currentUserID == "" {
		return nil, ErrAccessDenied
	}
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return nil, ErrEmptyComment
	}

	comment, err := s.store.GetCommentByID(ctx, commentID)
	if err != nil {
		return nil, err
	}

	isAuthor := comment.AuthorID != nil && *comment.AuthorID == currentUserID
	if !isAuthor && !isSiteAdmin {
		return nil, ErrAccessDenied
	}

	return s.store.UpdateComment(ctx, commentID, body)
}

func (s *service) DeleteComment(ctx context.Context, currentUserID string, isSiteAdmin bool, commentID string) error {
	if currentUserID == "" {
		return ErrAccessDenied
	}

	comment, err := s.store.GetCommentByID(ctx, commentID)
	if err != nil {
		return err
	}

	isAuthor := comment.AuthorID != nil && *comment.AuthorID == currentUserID
	if !isAuthor && !isSiteAdmin {
		return ErrAccessDenied
	}

	return s.store.DeleteComment(ctx, commentID)
}

func (s *service) CreateLabel(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, req CreateLabelRequest) (*Label, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, ErrEmptyLabelName
	}
	color := strings.TrimSpace(req.Color)
	if color == "" {
		color = "#0366d6"
	}
	if !strings.HasPrefix(color, "#") {
		color = "#" + color
	}

	repo, perm, err := s.getRepoAndPermission(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, err
	}
	if !perm.Includes(orgs.PermTriage) {
		return nil, ErrAccessDenied
	}

	label := &Label{
		RepositoryID: repo.ID,
		Name:         name,
		Color:        color,
		Description:  strings.TrimSpace(req.Description),
	}
	return s.store.CreateLabel(ctx, label)
}

func (s *service) ListLabels(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string) ([]*Label, error) {
	repo, perm, err := s.getRepoAndPermission(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, err
	}
	if !perm.Includes(orgs.PermRead) {
		return nil, ErrAccessDenied
	}

	_ = s.EnsureDefaultLabels(ctx, repo.ID)
	return s.store.ListLabels(ctx, repo.ID)
}

func (s *service) UpdateLabel(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, labelID string, req UpdateLabelRequest) (*Label, error) {
	repo, perm, err := s.getRepoAndPermission(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, err
	}
	if !perm.Includes(orgs.PermTriage) {
		return nil, ErrAccessDenied
	}

	label, err := s.store.GetLabelByID(ctx, labelID)
	if err != nil {
		return nil, err
	}
	if label.RepositoryID != repo.ID {
		return nil, ErrLabelNotFound
	}

	if req.Name != nil {
		n := strings.TrimSpace(*req.Name)
		if n == "" {
			return nil, ErrEmptyLabelName
		}
		label.Name = n
	}
	if req.Color != nil {
		c := strings.TrimSpace(*req.Color)
		if !strings.HasPrefix(c, "#") {
			c = "#" + c
		}
		label.Color = c
	}
	if req.Description != nil {
		label.Description = strings.TrimSpace(*req.Description)
	}

	return s.store.UpdateLabel(ctx, label)
}

func (s *service) DeleteLabel(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, labelID string) error {
	repo, perm, err := s.getRepoAndPermission(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return err
	}
	if !perm.Includes(orgs.PermTriage) {
		return ErrAccessDenied
	}

	label, err := s.store.GetLabelByID(ctx, labelID)
	if err != nil {
		return err
	}
	if label.RepositoryID != repo.ID {
		return ErrLabelNotFound
	}

	return s.store.DeleteLabel(ctx, labelID)
}

func (s *service) CreateMilestone(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, req CreateMilestoneRequest) (*Milestone, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, ErrEmptyMilestoneTitle
	}

	repo, perm, err := s.getRepoAndPermission(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, err
	}
	if !perm.Includes(orgs.PermTriage) {
		return nil, ErrAccessDenied
	}

	m := &Milestone{
		RepositoryID: repo.ID,
		Title:        title,
		Description:  strings.TrimSpace(req.Description),
		DueDate:      req.DueDate,
		State:        StateOpen,
	}
	return s.store.CreateMilestone(ctx, m)
}

func (s *service) ListMilestones(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string) ([]*Milestone, error) {
	repo, perm, err := s.getRepoAndPermission(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, err
	}
	if !perm.Includes(orgs.PermRead) {
		return nil, ErrAccessDenied
	}

	return s.store.ListMilestones(ctx, repo.ID)
}

func (s *service) UpdateMilestone(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, milestoneID string, req UpdateMilestoneRequest) (*Milestone, error) {
	repo, perm, err := s.getRepoAndPermission(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, err
	}
	if !perm.Includes(orgs.PermTriage) {
		return nil, ErrAccessDenied
	}

	m, err := s.store.GetMilestoneByID(ctx, milestoneID)
	if err != nil {
		return nil, err
	}
	if m.RepositoryID != repo.ID {
		return nil, ErrMilestoneNotFound
	}

	if req.Title != nil {
		t := strings.TrimSpace(*req.Title)
		if t == "" {
			return nil, ErrEmptyMilestoneTitle
		}
		m.Title = t
	}
	if req.Description != nil {
		m.Description = strings.TrimSpace(*req.Description)
	}
	if req.DueDate != nil {
		m.DueDate = req.DueDate
	}
	if req.State != nil {
		if *req.State == StateClosed && m.State != StateClosed {
			m.State = StateClosed
			now := time.Now().UTC()
			m.ClosedAt = &now
		} else if *req.State == StateOpen && m.State != StateOpen {
			m.State = StateOpen
			m.ClosedAt = nil
		}
	}

	return s.store.UpdateMilestone(ctx, m)
}

func (s *service) DeleteMilestone(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, milestoneID string) error {
	repo, perm, err := s.getRepoAndPermission(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return err
	}
	if !perm.Includes(orgs.PermTriage) {
		return ErrAccessDenied
	}

	m, err := s.store.GetMilestoneByID(ctx, milestoneID)
	if err != nil {
		return err
	}
	if m.RepositoryID != repo.ID {
		return ErrMilestoneNotFound
	}

	return s.store.DeleteMilestone(ctx, milestoneID)
}

func (s *service) EnsureDefaultLabels(ctx context.Context, repoID string) error {
	existing, err := s.store.ListLabels(ctx, repoID)
	if err != nil || len(existing) > 0 {
		return nil
	}

	defaults := []struct {
		name, color, desc string
	}{
		{"bug", "#d73a4a", "Something isn't working"},
		{"documentation", "#0075ca", "Improvements or additions to documentation"},
		{"duplicate", "#cfd3d7", "This issue or pull request already exists"},
		{"enhancement", "#a2eeef", "New feature or request"},
		{"good first issue", "#7057ff", "Good for newcomers"},
		{"help wanted", "#008672", "Extra attention is needed"},
		{"invalid", "#e4e669", "This doesn't seem right"},
		{"question", "#d876e3", "Further information is requested"},
		{"wontfix", "#ffffff", "This will not be worked on"},
	}

	for _, d := range defaults {
		_, _ = s.store.CreateLabel(ctx, &Label{
			RepositoryID: repoID,
			Name:         d.name,
			Color:        d.color,
			Description:  d.desc,
		})
	}
	return nil
}
