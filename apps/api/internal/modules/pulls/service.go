package pulls

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"forgehub/apps/api/internal/git"
	"forgehub/apps/api/internal/modules/auth"
	"forgehub/apps/api/internal/modules/orgs"
	"forgehub/apps/api/internal/modules/repos"
)

var (
	ErrAccessDenied      = errors.New("access denied: insufficient permissions")
	ErrEmptyTitle        = errors.New("pull request title cannot be empty")
	ErrSameBranches      = errors.New("source branch and target branch must be different")
	ErrBranchNotFound    = errors.New("branch not found in repository")
	ErrPRAlreadyMerged   = errors.New("pull request is already merged")
	ErrPRClosed          = errors.New("pull request is closed and cannot be merged")
	ErrHasConflicts      = errors.New("cannot merge: pull request has merge conflicts")
	ErrEmptyCommentBody  = errors.New("comment body cannot be empty")
	ErrInvalidMergeMethod = errors.New("invalid merge method: must be 'merge' or 'squash'")
)

type Service interface {
	CreatePullRequest(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, req CreatePRRequest) (*PullRequest, error)
	GetPullRequest(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, number int) (*PullRequestDetail, error)
	ListPullRequests(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, filter PRFilter) ([]*PullRequest, int, int, int, error)
	UpdatePullRequest(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, number int, req UpdatePRRequest) (*PullRequest, error)
	MergePullRequest(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, number int, req MergePRRequest) (*PullRequest, error)

	CreateReview(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, number int, req CreateReviewRequest) (*PullRequestReview, error)
	ListReviews(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, number int) ([]*PullRequestReview, error)

	CreateComment(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, number int, req CreatePRCommentRequest) (*PullRequestComment, error)
	ListComments(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, number int) ([]*PullRequestComment, error)
	DeleteComment(ctx context.Context, currentUserID string, isSiteAdmin bool, commentID string) error

	CompareBranches(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug, baseRef, headRef string) (*git.DiffResult, []git.CommitInfo, bool, error)
	GetPRDiff(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, number int) (*git.DiffResult, error)
	GetPRCommits(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, number int) ([]git.CommitInfo, error)
}

type service struct {
	store      RepositoryStore
	reposSvc   repos.Service
	gitStorage git.Storage
	gitReader  git.Reader
	authRepo   auth.Repository
}

func NewService(
	store RepositoryStore,
	reposSvc repos.Service,
	gitStorage git.Storage,
	gitReader git.Reader,
	authRepo auth.Repository,
) Service {
	return &service{
		store:      store,
		reposSvc:   reposSvc,
		gitStorage: gitStorage,
		gitReader:  gitReader,
		authRepo:   authRepo,
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

func (s *service) CreatePullRequest(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, req CreatePRRequest) (*PullRequest, error) {
	if currentUserID == "" {
		return nil, ErrAccessDenied
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, ErrEmptyTitle
	}
	sourceBranch := strings.TrimSpace(req.SourceBranch)
	targetBranch := strings.TrimSpace(req.TargetBranch)
	if sourceBranch == "" || targetBranch == "" {
		return nil, errors.New("source_branch and target_branch are required")
	}
	if sourceBranch == targetBranch {
		return nil, ErrSameBranches
	}

	repo, perm, err := s.getRepoAndPermission(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, err
	}
	if !perm.Includes(orgs.PermRead) {
		return nil, ErrAccessDenied
	}

	// Verify branches exist in git repository
	branches, err := s.gitReader.ListBranches(repo.DiskPath)
	if err != nil {
		return nil, fmt.Errorf("read branches: %w", err)
	}
	hasSource, hasTarget := false, false
	for _, b := range branches {
		if b.Name == sourceBranch {
			hasSource = true
		}
		if b.Name == targetBranch {
			hasTarget = true
		}
	}
	if !hasSource || !hasTarget {
		return nil, ErrBranchNotFound
	}

	pr := &PullRequest{
		RepositoryID: repo.ID,
		Title:        title,
		Body:         strings.TrimSpace(req.Body),
		State:        StateOpen,
		SourceBranch: sourceBranch,
		TargetBranch: targetBranch,
		AuthorID:     &currentUserID,
		IsDraft:      req.IsDraft,
	}

	return s.store.CreatePullRequest(ctx, pr)
}

func (s *service) GetPullRequest(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, number int) (*PullRequestDetail, error) {
	repo, perm, err := s.getRepoAndPermission(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, err
	}
	if !perm.Includes(orgs.PermRead) {
		return nil, ErrAccessDenied
	}

	pr, err := s.store.GetPRByNumber(ctx, repo.ID, number)
	if err != nil {
		return nil, err
	}

	reviews, err := s.store.ListReviews(ctx, pr.ID)
	if err != nil {
		return nil, err
	}

	comments, err := s.store.ListComments(ctx, pr.ID)
	if err != nil {
		return nil, err
	}

	// Fetch commits and diff between branches
	commits, _ := s.gitReader.GetCommitsBetween(repo.DiskPath, pr.TargetBranch, pr.SourceBranch)
	diff, _ := s.gitReader.DiffBranches(repo.DiskPath, pr.TargetBranch, pr.SourceBranch)

	canMerge := false
	hasConflicts := false
	if pr.State == StateOpen {
		mergeable, _ := s.gitStorage.CheckMergeable(repo.DiskPath, pr.TargetBranch, pr.SourceBranch)
		canMerge = mergeable
		hasConflicts = !mergeable
	}

	return &PullRequestDetail{
		PullRequest:  *pr,
		Reviews:      reviews,
		Comments:     comments,
		Commits:      commits,
		Diff:         diff,
		CanMerge:     canMerge,
		HasConflicts: hasConflicts,
	}, nil
}

func (s *service) ListPullRequests(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, filter PRFilter) ([]*PullRequest, int, int, int, error) {
	repo, perm, err := s.getRepoAndPermission(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	if !perm.Includes(orgs.PermRead) {
		return nil, 0, 0, 0, ErrAccessDenied
	}

	return s.store.ListPullRequests(ctx, repo.ID, filter)
}

func (s *service) UpdatePullRequest(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, number int, req UpdatePRRequest) (*PullRequest, error) {
	if currentUserID == "" {
		return nil, ErrAccessDenied
	}

	repo, perm, err := s.getRepoAndPermission(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, err
	}

	pr, err := s.store.GetPRByNumber(ctx, repo.ID, number)
	if err != nil {
		return nil, err
	}

	isAuthor := pr.AuthorID != nil && *pr.AuthorID == currentUserID
	isTriage := perm.Includes(orgs.PermTriage)

	if !isAuthor && !isTriage {
		return nil, ErrAccessDenied
	}

	if req.Title != nil {
		t := strings.TrimSpace(*req.Title)
		if t == "" {
			return nil, ErrEmptyTitle
		}
		pr.Title = t
	}

	if req.Body != nil {
		pr.Body = strings.TrimSpace(*req.Body)
	}

	if req.IsDraft != nil {
		pr.IsDraft = *req.IsDraft
	}

	if req.TargetBranch != nil {
		tb := strings.TrimSpace(*req.TargetBranch)
		if tb != "" && tb != pr.SourceBranch {
			pr.TargetBranch = tb
		}
	}

	if req.State != nil {
		if pr.State == StateMerged {
			return nil, ErrPRAlreadyMerged
		}
		if *req.State == StateClosed && pr.State != StateClosed {
			pr.State = StateClosed
			now := time.Now().UTC()
			pr.ClosedAt = &now
			pr.ClosedByID = &currentUserID
		} else if *req.State == StateOpen && pr.State != StateOpen {
			pr.State = StateOpen
			pr.ClosedAt = nil
			pr.ClosedByID = nil
		}
	}

	return s.store.UpdatePullRequest(ctx, pr)
}

func (s *service) MergePullRequest(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, number int, req MergePRRequest) (*PullRequest, error) {
	if currentUserID == "" {
		return nil, ErrAccessDenied
	}

	repo, perm, err := s.getRepoAndPermission(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, err
	}
	// Merging requires write permissions on the repository
	if !perm.Includes(orgs.PermWrite) {
		return nil, ErrAccessDenied
	}

	pr, err := s.store.GetPRByNumber(ctx, repo.ID, number)
	if err != nil {
		return nil, err
	}

	if pr.State == StateMerged {
		return nil, ErrPRAlreadyMerged
	}
	if pr.State == StateClosed {
		return nil, ErrPRClosed
	}

	method := strings.ToLower(strings.TrimSpace(req.Method))
	if method == "" {
		method = "merge"
	}
	if method != "merge" && method != "squash" {
		return nil, ErrInvalidMergeMethod
	}

	// 1. Verify mergeable
	mergeable, err := s.gitStorage.CheckMergeable(repo.DiskPath, pr.TargetBranch, pr.SourceBranch)
	if err != nil || !mergeable {
		return nil, ErrHasConflicts
	}

	// Fetch user details for committer name/email
	u, err := s.authRepo.GetUserByID(ctx, currentUserID)
	if err != nil {
		return nil, err
	}

	commitMsg := strings.TrimSpace(req.CommitMessage)
	if commitMsg == "" {
		commitMsg = fmt.Sprintf("Merge pull request #%d from %s\n\n%s", pr.Number, pr.SourceBranch, pr.Title)
	}

	// 2. Perform merge in bare repository
	mergeCommitSHA, err := s.gitStorage.MergeBranches(repo.DiskPath, pr.TargetBranch, pr.SourceBranch, method, u.DisplayName, u.Email, commitMsg)
	if err != nil {
		return nil, fmt.Errorf("git merge execution: %w", err)
	}

	// 3. Update PR status
	now := time.Now().UTC()
	pr.State = StateMerged
	pr.MergeCommitSHA = &mergeCommitSHA
	pr.MergedByID = &currentUserID
	pr.MergedAt = &now
	pr.ClosedAt = &now
	pr.ClosedByID = &currentUserID

	return s.store.UpdatePullRequest(ctx, pr)
}

func (s *service) CreateReview(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, number int, req CreateReviewRequest) (*PullRequestReview, error) {
	if currentUserID == "" {
		return nil, ErrAccessDenied
	}

	repo, perm, err := s.getRepoAndPermission(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, err
	}
	if !perm.Includes(orgs.PermRead) {
		return nil, ErrAccessDenied
	}

	pr, err := s.store.GetPRByNumber(ctx, repo.ID, number)
	if err != nil {
		return nil, err
	}

	revState := req.State
	if revState != ReviewApproved && revState != ReviewChangesRequested && revState != ReviewCommented {
		revState = ReviewCommented
	}

	review := &PullRequestReview{
		PullRequestID: pr.ID,
		ReviewerID:    currentUserID,
		State:         revState,
		Body:          strings.TrimSpace(req.Body),
	}
	return s.store.CreateReview(ctx, review)
}

func (s *service) ListReviews(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, number int) ([]*PullRequestReview, error) {
	repo, perm, err := s.getRepoAndPermission(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, err
	}
	if !perm.Includes(orgs.PermRead) {
		return nil, ErrAccessDenied
	}

	pr, err := s.store.GetPRByNumber(ctx, repo.ID, number)
	if err != nil {
		return nil, err
	}

	return s.store.ListReviews(ctx, pr.ID)
}

func (s *service) CreateComment(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, number int, req CreatePRCommentRequest) (*PullRequestComment, error) {
	if currentUserID == "" {
		return nil, ErrAccessDenied
	}
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return nil, ErrEmptyCommentBody
	}

	repo, perm, err := s.getRepoAndPermission(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, err
	}
	if !perm.Includes(orgs.PermRead) {
		return nil, ErrAccessDenied
	}

	pr, err := s.store.GetPRByNumber(ctx, repo.ID, number)
	if err != nil {
		return nil, err
	}

	comment := &PullRequestComment{
		PullRequestID: pr.ID,
		AuthorID:      &currentUserID,
		ReviewID:      req.ReviewID,
		FilePath:      req.FilePath,
		LineNumber:    req.LineNumber,
		Body:          body,
	}
	return s.store.CreateComment(ctx, comment)
}

func (s *service) ListComments(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, number int) ([]*PullRequestComment, error) {
	repo, perm, err := s.getRepoAndPermission(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, err
	}
	if !perm.Includes(orgs.PermRead) {
		return nil, ErrAccessDenied
	}

	pr, err := s.store.GetPRByNumber(ctx, repo.ID, number)
	if err != nil {
		return nil, err
	}

	return s.store.ListComments(ctx, pr.ID)
}

func (s *service) DeleteComment(ctx context.Context, currentUserID string, isSiteAdmin bool, commentID string) error {
	if currentUserID == "" {
		return ErrAccessDenied
	}
	return s.store.DeleteComment(ctx, commentID)
}

func (s *service) CompareBranches(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug, baseRef, headRef string) (*git.DiffResult, []git.CommitInfo, bool, error) {
	repo, perm, err := s.getRepoAndPermission(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, nil, false, err
	}
	if !perm.Includes(orgs.PermRead) {
		return nil, nil, false, ErrAccessDenied
	}

	diff, err := s.gitReader.DiffBranches(repo.DiskPath, baseRef, headRef)
	if err != nil {
		return nil, nil, false, err
	}

	commits, err := s.gitReader.GetCommitsBetween(repo.DiskPath, baseRef, headRef)
	if err != nil {
		return nil, nil, false, err
	}

	canMerge, _ := s.gitStorage.CheckMergeable(repo.DiskPath, baseRef, headRef)
	return diff, commits, canMerge, nil
}

func (s *service) GetPRDiff(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, number int) (*git.DiffResult, error) {
	repo, perm, err := s.getRepoAndPermission(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, err
	}
	if !perm.Includes(orgs.PermRead) {
		return nil, ErrAccessDenied
	}

	pr, err := s.store.GetPRByNumber(ctx, repo.ID, number)
	if err != nil {
		return nil, err
	}

	return s.gitReader.DiffBranches(repo.DiskPath, pr.TargetBranch, pr.SourceBranch)
}

func (s *service) GetPRCommits(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, number int) ([]git.CommitInfo, error) {
	repo, perm, err := s.getRepoAndPermission(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, err
	}
	if !perm.Includes(orgs.PermRead) {
		return nil, ErrAccessDenied
	}

	pr, err := s.store.GetPRByNumber(ctx, repo.ID, number)
	if err != nil {
		return nil, err
	}

	return s.gitReader.GetCommitsBetween(repo.DiskPath, pr.TargetBranch, pr.SourceBranch)
}
