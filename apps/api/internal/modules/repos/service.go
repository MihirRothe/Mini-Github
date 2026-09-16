package repos

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"forgehub/apps/api/internal/config"
	"forgehub/apps/api/internal/git"
	"forgehub/apps/api/internal/modules/auth"
	"forgehub/apps/api/internal/modules/orgs"
)

var (
	repoSlugRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{1,99}$`)

	ErrInvalidRepoSlug      = errors.New("repository name must be 2-100 characters (alphanumeric, dots, dashes, underscores)")
	ErrInsufficientRepoPerm = errors.New("you do not have permission to perform this action on this repository")
)

type Service interface {
	CreateRepository(ctx context.Context, currentUserID string, isSiteAdmin bool, req CreateRepoRequest) (*Repository, error)
	GetRepository(ctx context.Context, currentUserID string, isSiteAdmin bool, ownerSlug, repoSlug string) (*Repository, error)
	ListRepositories(ctx context.Context, currentUserID string, isSiteAdmin bool, filter RepoFilter) ([]*Repository, error)
	UpdateRepository(ctx context.Context, currentUserID string, isSiteAdmin bool, ownerSlug, repoSlug string, req UpdateRepoRequest) (*Repository, error)
	DeleteRepository(ctx context.Context, currentUserID string, isSiteAdmin bool, ownerSlug, repoSlug string) error

	GetBranches(ctx context.Context, currentUserID string, isSiteAdmin bool, ownerSlug, repoSlug string) ([]git.BranchInfo, error)
	GetCommits(ctx context.Context, currentUserID string, isSiteAdmin bool, ownerSlug, repoSlug, ref string, limit int) ([]git.CommitInfo, error)
	GetTree(ctx context.Context, currentUserID string, isSiteAdmin bool, ownerSlug, repoSlug, ref, subPath string) ([]git.TreeEntry, error)
	GetBlob(ctx context.Context, currentUserID string, isSiteAdmin bool, ownerSlug, repoSlug, ref, path string) (*git.BlobInfo, error)
	GetReadme(ctx context.Context, currentUserID string, isSiteAdmin bool, ownerSlug, repoSlug, ref string) (*git.BlobInfo, error)

	ResolveUserPermission(ctx context.Context, currentUserID string, isSiteAdmin bool, repo *Repository) (orgs.Permission, error)
}

type service struct {
	repoStore  RepositoryStore
	gitStorage git.Storage
	gitReader  git.Reader
	authRepo   auth.Repository
	orgsRepo   orgs.Repository
	cfg        *config.Config
}

func NewService(
	repoStore RepositoryStore,
	gitStorage git.Storage,
	gitReader git.Reader,
	authRepo auth.Repository,
	orgsRepo orgs.Repository,
	cfg *config.Config,
) Service {
	return &service{
		repoStore:  repoStore,
		gitStorage: gitStorage,
		gitReader:  gitReader,
		authRepo:   authRepo,
		orgsRepo:   orgsRepo,
		cfg:        cfg,
	}
}

func (s *service) CreateRepository(ctx context.Context, currentUserID string, isSiteAdmin bool, req CreateRepoRequest) (*Repository, error) {
	name := strings.TrimSpace(req.Name)
	if len(name) < 2 || len(name) > 100 {
		return nil, errors.New("repository name must be between 2 and 100 characters")
	}

	slug := strings.ToLower(strings.TrimSpace(req.Slug))
	if slug == "" {
		slug = strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	}
	if !repoSlugRegex.MatchString(slug) {
		return nil, ErrInvalidRepoSlug
	}

	visibility := strings.ToLower(strings.TrimSpace(req.Visibility))
	if visibility == "" {
		visibility = "public"
	}
	if visibility != "public" && visibility != "private" && visibility != "internal" {
		return nil, errors.New("visibility must be 'public', 'private', or 'internal'")
	}

	defaultBranch := strings.TrimSpace(req.DefaultBranch)
	if defaultBranch == "" {
		defaultBranch = s.cfg.GitDefaultBranch
	}

	// Determine owner and check permissions
	var ownerType string
	var ownerName string
	var ownerUserID *string
	var ownerOrgID *string

	if req.OwnerType == "org" && req.OwnerSlug != "" {
		org, err := s.orgsRepo.GetOrganizationBySlug(ctx, req.OwnerSlug)
		if err != nil {
			return nil, fmt.Errorf("lookup organization: %w", err)
		}
		if !isSiteAdmin {
			member, err := s.orgsRepo.GetMember(ctx, org.ID, currentUserID)
			if err != nil || (member.Role != orgs.OrgRoleOwner && member.Role != orgs.OrgRoleAdmin) {
				return nil, errors.New("only organization owners and admins can create repositories in this organization")
			}
		}
		ownerType = "orgs"
		ownerName = org.Slug
		ownerOrgID = &org.ID
	} else {
		// Personal user repository
		user, err := s.authRepo.GetUserByID(ctx, currentUserID)
		if err != nil {
			return nil, fmt.Errorf("lookup user: %w", err)
		}
		ownerType = "users"
		ownerName = user.Username
		ownerUserID = &user.ID
	}

	// 1. Initialize bare repository on disk
	diskPath, err := s.gitStorage.InitRepository(ownerType, ownerName, slug, defaultBranch)
	if err != nil {
		if errors.Is(err, git.ErrRepoAlreadyExists) {
			return nil, ErrRepoSlugTaken
		}
		return nil, fmt.Errorf("init git repo on disk: %w", err)
	}

	// 2. If requested, write initial README commit
	if req.InitWithReadme {
		user, _ := s.authRepo.GetUserByID(ctx, currentUserID)
		authorName := "ForgeHub User"
		authorEmail := "noreply@forgehub.local"
		if user != nil {
			authorName = user.DisplayName
			if authorName == "" {
				authorName = user.Username
			}
			authorEmail = user.Email
		}
		readmeContent := fmt.Sprintf("# %s\n\n%s\n", name, strings.TrimSpace(req.Description))
		_ = s.gitStorage.CreateInitialCommit(diskPath, defaultBranch, readmeContent, authorName, authorEmail)
	}

	cloneURL := fmt.Sprintf("%s/%s/%s.git", s.cfg.BaseURL, ownerName, slug)

	repo := &Repository{
		Name:                  name,
		Slug:                  slug,
		Description:           strings.TrimSpace(req.Description),
		Visibility:            visibility,
		DefaultBranch:         defaultBranch,
		OwnerType:             strings.TrimSuffix(ownerType, "s"), // "user" or "org"
		OwnerUserID:           ownerUserID,
		OwnerOrgID:            ownerOrgID,
		OwnerName:             ownerName,
		DiskPath:              diskPath,
		CloneURL:              cloneURL,
		HTTPCloneURL:          cloneURL,
		CurrentUserPermission: orgs.PermAdmin,
	}

	if err := s.repoStore.CreateRepo(ctx, repo); err != nil {
		_ = s.gitStorage.DeleteRepository(ownerType, ownerName, slug)
		return nil, err
	}

	return repo, nil
}

func (s *service) GetRepository(ctx context.Context, currentUserID string, isSiteAdmin bool, ownerSlug, repoSlug string) (*Repository, error) {
	repo, err := s.repoStore.GetRepoByOwnerAndSlug(ctx, ownerSlug, repoSlug)
	if err != nil {
		return nil, err
	}

	perm, err := s.ResolveUserPermission(ctx, currentUserID, isSiteAdmin, repo)
	if err != nil {
		return nil, err
	}

	// If repository is private and user has no access, hide existence
	if repo.Visibility != "public" && perm == orgs.PermNone {
		return nil, ErrRepoNotFound
	}

	repo.CurrentUserPermission = perm
	cloneURL := fmt.Sprintf("%s/%s/%s.git", s.cfg.BaseURL, repo.OwnerName, repo.Slug)
	repo.CloneURL = cloneURL
	repo.HTTPCloneURL = cloneURL

	return repo, nil
}

func (s *service) ListRepositories(ctx context.Context, currentUserID string, isSiteAdmin bool, filter RepoFilter) ([]*Repository, error) {
	list, err := s.repoStore.ListRepositories(ctx, filter)
	if err != nil {
		return nil, err
	}

	var visible []*Repository
	for _, repo := range list {
		perm, err := s.ResolveUserPermission(ctx, currentUserID, isSiteAdmin, repo)
		if err == nil {
			if repo.Visibility == "public" || perm != orgs.PermNone {
				repo.CurrentUserPermission = perm
				cloneURL := fmt.Sprintf("%s/%s/%s.git", s.cfg.BaseURL, repo.OwnerName, repo.Slug)
				repo.CloneURL = cloneURL
				repo.HTTPCloneURL = cloneURL
				visible = append(visible, repo)
			}
		}
	}

	return visible, nil
}

func (s *service) UpdateRepository(ctx context.Context, currentUserID string, isSiteAdmin bool, ownerSlug, repoSlug string, req UpdateRepoRequest) (*Repository, error) {
	repo, err := s.GetRepository(ctx, currentUserID, isSiteAdmin, ownerSlug, repoSlug)
	if err != nil {
		return nil, err
	}

	if !repo.CurrentUserPermission.Includes(orgs.PermMaintain) {
		return nil, ErrInsufficientRepoPerm
	}

	if req.Description != nil {
		repo.Description = strings.TrimSpace(*req.Description)
	}
	if req.Visibility != nil {
		v := strings.ToLower(strings.TrimSpace(*req.Visibility))
		if v == "public" || v == "private" || v == "internal" {
			repo.Visibility = v
		}
	}
	if req.DefaultBranch != nil && *req.DefaultBranch != "" {
		repo.DefaultBranch = strings.TrimSpace(*req.DefaultBranch)
	}
	if req.IsArchived != nil {
		repo.IsArchived = *req.IsArchived
	}

	if err := s.repoStore.UpdateRepo(ctx, repo); err != nil {
		return nil, err
	}

	return repo, nil
}

func (s *service) DeleteRepository(ctx context.Context, currentUserID string, isSiteAdmin bool, ownerSlug, repoSlug string) error {
	repo, err := s.GetRepository(ctx, currentUserID, isSiteAdmin, ownerSlug, repoSlug)
	if err != nil {
		return err
	}

	if !repo.CurrentUserPermission.Includes(orgs.PermAdmin) {
		return ErrInsufficientRepoPerm
	}

	// Remove bare repo on disk
	ownerType := "users"
	if repo.OwnerType == "org" {
		ownerType = "orgs"
	}
	_ = s.gitStorage.DeleteRepository(ownerType, repo.OwnerName, repo.Slug)

	return s.repoStore.DeleteRepo(ctx, repo.ID)
}

func (s *service) GetBranches(ctx context.Context, currentUserID string, isSiteAdmin bool, ownerSlug, repoSlug string) ([]git.BranchInfo, error) {
	repo, err := s.GetRepository(ctx, currentUserID, isSiteAdmin, ownerSlug, repoSlug)
	if err != nil {
		return nil, err
	}
	return s.gitReader.ListBranches(repo.DiskPath)
}

func (s *service) GetCommits(ctx context.Context, currentUserID string, isSiteAdmin bool, ownerSlug, repoSlug, ref string, limit int) ([]git.CommitInfo, error) {
	repo, err := s.GetRepository(ctx, currentUserID, isSiteAdmin, ownerSlug, repoSlug)
	if err != nil {
		return nil, err
	}
	return s.gitReader.ListCommits(repo.DiskPath, ref, limit)
}

func (s *service) GetTree(ctx context.Context, currentUserID string, isSiteAdmin bool, ownerSlug, repoSlug, ref, subPath string) ([]git.TreeEntry, error) {
	repo, err := s.GetRepository(ctx, currentUserID, isSiteAdmin, ownerSlug, repoSlug)
	if err != nil {
		return nil, err
	}
	return s.gitReader.ListTree(repo.DiskPath, ref, subPath)
}

func (s *service) GetBlob(ctx context.Context, currentUserID string, isSiteAdmin bool, ownerSlug, repoSlug, ref, path string) (*git.BlobInfo, error) {
	repo, err := s.GetRepository(ctx, currentUserID, isSiteAdmin, ownerSlug, repoSlug)
	if err != nil {
		return nil, err
	}
	return s.gitReader.GetBlob(repo.DiskPath, ref, path)
}

func (s *service) GetReadme(ctx context.Context, currentUserID string, isSiteAdmin bool, ownerSlug, repoSlug, ref string) (*git.BlobInfo, error) {
	repo, err := s.GetRepository(ctx, currentUserID, isSiteAdmin, ownerSlug, repoSlug)
	if err != nil {
		return nil, err
	}
	return s.gitReader.GetReadme(repo.DiskPath, ref)
}

func (s *service) ResolveUserPermission(ctx context.Context, currentUserID string, isSiteAdmin bool, repo *Repository) (orgs.Permission, error) {
	if isSiteAdmin {
		return orgs.PermAdmin, nil
	}
	if currentUserID == "" {
		if repo.Visibility == "public" {
			return orgs.PermRead, nil
		}
		return orgs.PermNone, nil
	}

	// Check if user is repository direct owner
	if repo.OwnerUserID != nil && *repo.OwnerUserID == currentUserID {
		return orgs.PermAdmin, nil
	}

	var orgRole *orgs.OrgRole
	var teamPerms []orgs.Permission

	// If owned by an organization, lookup org membership and team permissions
	if repo.OwnerOrgID != nil {
		if member, err := s.orgsRepo.GetMember(ctx, *repo.OwnerOrgID, currentUserID); err == nil {
			orgRole = &member.Role
		}
		if tPerms, err := s.orgsRepo.GetUserTeamPermissions(ctx, *repo.OwnerOrgID, currentUserID, repo.ID); err == nil {
			teamPerms = tPerms
		}
	}

	// Direct collaborator permissions
	directPerm, _ := s.repoStore.GetCollaboratorPermission(ctx, repo.ID, currentUserID)

	effective := orgs.ResolveEffectivePermission(false, orgRole, directPerm, teamPerms)

	// Public repos allow at least read
	if repo.Visibility == "public" && effective == orgs.PermNone {
		effective = orgs.PermRead
	}

	return effective, nil
}
