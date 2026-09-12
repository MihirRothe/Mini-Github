package orgs

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"forgehub/apps/api/internal/modules/auth"
)

var (
	orgSlugRegex  = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{1,38}[a-zA-Z0-9]$`)
	teamSlugRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{1,38}[a-zA-Z0-9]$`)

	reservedSlugs = map[string]bool{
		"api": true, "admin": true, "new": true, "settings": true, "explore": true,
		"login": true, "register": true, "logout": true, "organizations": true,
		"orgs": true, "teams": true, "users": true, "help": true, "assets": true,
	}

	ErrInvalidOrgSlug       = errors.New("organization slug must be 3-40 characters (alphanumeric, hyphens, underscores)")
	ErrReservedOrgSlug      = errors.New("this organization slug is reserved")
	ErrInvalidTeamSlug      = errors.New("team slug must be 3-40 characters (alphanumeric, hyphens, underscores)")
	ErrInsufficientPrivilege = errors.New("you do not have permission to perform this action")
	ErrUserNotOrgMember     = errors.New("user must first be an organization member before joining a team")
)

type Service interface {
	CreateOrganization(ctx context.Context, userID string, req CreateOrgRequest) (*Organization, error)
	GetOrganization(ctx context.Context, currentUserID, slug string) (*Organization, error)
	ListUserOrganizations(ctx context.Context, userID string) ([]*Organization, error)
	UpdateOrganization(ctx context.Context, currentUserID string, isSiteAdmin bool, slug string, req UpdateOrgRequest) (*Organization, error)
	DeleteOrganization(ctx context.Context, currentUserID string, isSiteAdmin bool, slug string) error

	ListMembers(ctx context.Context, currentUserID, slug string) ([]*OrgMember, error)
	AddMember(ctx context.Context, currentUserID string, isSiteAdmin bool, slug string, req AddOrgMemberRequest) (*OrgMember, error)
	UpdateMemberRole(ctx context.Context, currentUserID string, isSiteAdmin bool, slug, targetUsername string, newRole OrgRole) error
	RemoveMember(ctx context.Context, currentUserID string, isSiteAdmin bool, slug, targetUsername string) error

	CreateTeam(ctx context.Context, currentUserID string, isSiteAdmin bool, slug string, req CreateTeamRequest) (*Team, error)
	ListTeams(ctx context.Context, currentUserID, slug string) ([]*Team, error)
	GetTeam(ctx context.Context, currentUserID, slug, teamSlug string) (*Team, error)
	UpdateTeam(ctx context.Context, currentUserID string, isSiteAdmin bool, slug, teamSlug string, req UpdateTeamRequest) (*Team, error)
	DeleteTeam(ctx context.Context, currentUserID string, isSiteAdmin bool, slug, teamSlug string) error

	ListTeamMembers(ctx context.Context, currentUserID, slug, teamSlug string) ([]*TeamMember, error)
	AddTeamMember(ctx context.Context, currentUserID string, isSiteAdmin bool, slug, teamSlug string, req AddTeamMemberRequest) (*TeamMember, error)
	UpdateTeamMemberRole(ctx context.Context, currentUserID string, isSiteAdmin bool, slug, teamSlug, targetUsername string, newRole TeamRole) error
	RemoveTeamMember(ctx context.Context, currentUserID string, isSiteAdmin bool, slug, teamSlug, targetUsername string) error
}

type service struct {
	repo     Repository
	authRepo auth.Repository
}

func NewService(repo Repository, authRepo auth.Repository) Service {
	return &service{
		repo:     repo,
		authRepo: authRepo,
	}
}

// ----------------------------------------------------------------------------
// Organizations
// ----------------------------------------------------------------------------

func (s *service) CreateOrganization(ctx context.Context, userID string, req CreateOrgRequest) (*Organization, error) {
	slug := strings.ToLower(strings.TrimSpace(req.Slug))
	if !orgSlugRegex.MatchString(slug) {
		return nil, ErrInvalidOrgSlug
	}
	if reservedSlugs[slug] {
		return nil, ErrReservedOrgSlug
	}

	name := strings.TrimSpace(req.Name)
	if len(name) < 2 || len(name) > 128 {
		return nil, errors.New("organization name must be between 2 and 128 characters")
	}

	org := &Organization{
		Name:        name,
		Slug:        slug,
		Description: strings.TrimSpace(req.Description),
		Website:     strings.TrimSpace(req.Website),
		Location:    strings.TrimSpace(req.Location),
		AvatarURL:   strings.TrimSpace(req.AvatarURL),
	}

	if err := s.repo.CreateOrganization(ctx, org, userID); err != nil {
		return nil, err
	}

	org.Role = OrgRoleOwner
	org.MemberCount = 1
	return org, nil
}

func (s *service) GetOrganization(ctx context.Context, currentUserID, slug string) (*Organization, error) {
	org, err := s.repo.GetOrganizationBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	if currentUserID != "" {
		if member, err := s.repo.GetMember(ctx, org.ID, currentUserID); err == nil {
			org.Role = member.Role
		}
	}

	return org, nil
}

func (s *service) ListUserOrganizations(ctx context.Context, userID string) ([]*Organization, error) {
	return s.repo.ListUserOrganizations(ctx, userID)
}

func (s *service) UpdateOrganization(ctx context.Context, currentUserID string, isSiteAdmin bool, slug string, req UpdateOrgRequest) (*Organization, error) {
	org, err := s.repo.GetOrganizationBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	if !isSiteAdmin {
		member, err := s.repo.GetMember(ctx, org.ID, currentUserID)
		if err != nil || (member.Role != OrgRoleOwner && member.Role != OrgRoleAdmin) {
			return nil, ErrInsufficientPrivilege
		}
	}

	name := strings.TrimSpace(req.Name)
	if name != "" {
		if len(name) < 2 || len(name) > 128 {
			return nil, errors.New("organization name must be between 2 and 128 characters")
		}
		org.Name = name
	}
	org.Description = strings.TrimSpace(req.Description)
	org.Website = strings.TrimSpace(req.Website)
	org.Location = strings.TrimSpace(req.Location)
	if req.AvatarURL != "" {
		org.AvatarURL = strings.TrimSpace(req.AvatarURL)
	}

	if err := s.repo.UpdateOrganization(ctx, org); err != nil {
		return nil, err
	}

	return org, nil
}

func (s *service) DeleteOrganization(ctx context.Context, currentUserID string, isSiteAdmin bool, slug string) error {
	org, err := s.repo.GetOrganizationBySlug(ctx, slug)
	if err != nil {
		return err
	}

	if !isSiteAdmin {
		member, err := s.repo.GetMember(ctx, org.ID, currentUserID)
		if err != nil || member.Role != OrgRoleOwner {
			return ErrInsufficientPrivilege
		}
	}

	return s.repo.DeleteOrganization(ctx, org.ID)
}

// ----------------------------------------------------------------------------
// Organization Members
// ----------------------------------------------------------------------------

func (s *service) ListMembers(ctx context.Context, currentUserID, slug string) ([]*OrgMember, error) {
	org, err := s.repo.GetOrganizationBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return s.repo.ListMembers(ctx, org.ID)
}

func (s *service) AddMember(ctx context.Context, currentUserID string, isSiteAdmin bool, slug string, req AddOrgMemberRequest) (*OrgMember, error) {
	org, err := s.repo.GetOrganizationBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	var callerRole OrgRole
	if !isSiteAdmin {
		caller, err := s.repo.GetMember(ctx, org.ID, currentUserID)
		if err != nil || (caller.Role != OrgRoleOwner && caller.Role != OrgRoleAdmin) {
			return nil, ErrInsufficientPrivilege
		}
		callerRole = caller.Role
	}

	// Validate target role
	role := req.Role
	if role == "" {
		role = OrgRoleMember
	}
	if role == OrgRoleOwner && callerRole != OrgRoleOwner && !isSiteAdmin {
		return nil, errors.New("only owners can add or promote other owners")
	}

	// Lookup user by username
	targetUser, err := s.authRepo.GetUserByUsername(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	member := &OrgMember{
		OrganizationID: org.ID,
		UserID:         targetUser.ID,
		Role:           role,
	}

	if err := s.repo.AddMember(ctx, member); err != nil {
		return nil, err
	}

	member.Username = targetUser.Username
	member.DisplayName = targetUser.DisplayName
	member.AvatarURL = targetUser.AvatarURL
	member.Email = targetUser.Email

	return member, nil
}

func (s *service) UpdateMemberRole(ctx context.Context, currentUserID string, isSiteAdmin bool, slug, targetUsername string, newRole OrgRole) error {
	org, err := s.repo.GetOrganizationBySlug(ctx, slug)
	if err != nil {
		return err
	}

	targetUser, err := s.authRepo.GetUserByUsername(ctx, targetUsername)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	targetMember, err := s.repo.GetMember(ctx, org.ID, targetUser.ID)
	if err != nil {
		return err
	}

	if !isSiteAdmin {
		caller, err := s.repo.GetMember(ctx, org.ID, currentUserID)
		if err != nil || (caller.Role != OrgRoleOwner && caller.Role != OrgRoleAdmin) {
			return ErrInsufficientPrivilege
		}

		// Only owners can modify other owners' roles or promote to owner
		if (targetMember.Role == OrgRoleOwner || newRole == OrgRoleOwner) && caller.Role != OrgRoleOwner {
			return errors.New("only owners can promote to owner or modify owner roles")
		}
	}

	// If demoting an owner, guard against demoting the last owner
	if targetMember.Role == OrgRoleOwner && newRole != OrgRoleOwner {
		ownerCount, err := s.repo.CountOwners(ctx, org.ID)
		if err != nil {
			return err
		}
		if ownerCount <= 1 {
			return ErrCannotRemoveLastOwner
		}
	}

	return s.repo.UpdateMemberRole(ctx, org.ID, targetUser.ID, newRole)
}

func (s *service) RemoveMember(ctx context.Context, currentUserID string, isSiteAdmin bool, slug, targetUsername string) error {
	org, err := s.repo.GetOrganizationBySlug(ctx, slug)
	if err != nil {
		return err
	}

	targetUser, err := s.authRepo.GetUserByUsername(ctx, targetUsername)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	targetMember, err := s.repo.GetMember(ctx, org.ID, targetUser.ID)
	if err != nil {
		return err
	}

	isSelf := currentUserID == targetUser.ID

	if !isSiteAdmin && !isSelf {
		caller, err := s.repo.GetMember(ctx, org.ID, currentUserID)
		if err != nil || (caller.Role != OrgRoleOwner && caller.Role != OrgRoleAdmin) {
			return ErrInsufficientPrivilege
		}
		// Admin cannot remove owner
		if caller.Role == OrgRoleAdmin && targetMember.Role == OrgRoleOwner {
			return ErrInsufficientPrivilege
		}
	}

	// If removing an owner, ensure not the last owner
	if targetMember.Role == OrgRoleOwner {
		ownerCount, err := s.repo.CountOwners(ctx, org.ID)
		if err != nil {
			return err
		}
		if ownerCount <= 1 {
			return ErrCannotRemoveLastOwner
		}
	}

	return s.repo.RemoveMember(ctx, org.ID, targetUser.ID)
}

// ----------------------------------------------------------------------------
// Teams
// ----------------------------------------------------------------------------

func (s *service) CreateTeam(ctx context.Context, currentUserID string, isSiteAdmin bool, slug string, req CreateTeamRequest) (*Team, error) {
	org, err := s.repo.GetOrganizationBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	if !isSiteAdmin {
		caller, err := s.repo.GetMember(ctx, org.ID, currentUserID)
		if err != nil || (caller.Role != OrgRoleOwner && caller.Role != OrgRoleAdmin) {
			return nil, ErrInsufficientPrivilege
		}
	}

	teamSlug := strings.ToLower(strings.TrimSpace(req.Slug))
	if !teamSlugRegex.MatchString(teamSlug) {
		return nil, ErrInvalidTeamSlug
	}

	name := strings.TrimSpace(req.Name)
	if len(name) < 2 || len(name) > 128 {
		return nil, errors.New("team name must be between 2 and 128 characters")
	}

	team := &Team{
		OrganizationID: org.ID,
		Name:           name,
		Slug:           teamSlug,
		Description:    strings.TrimSpace(req.Description),
		Privacy:        req.Privacy,
	}

	if err := s.repo.CreateTeam(ctx, team); err != nil {
		return nil, err
	}

	// Add creator as team maintainer
	_ = s.repo.AddTeamMember(ctx, &TeamMember{
		TeamID: team.ID,
		UserID: currentUserID,
		Role:   TeamRoleMaintainer,
	})

	team.MemberCount = 1
	team.CurrentUserRole = TeamRoleMaintainer
	return team, nil
}

func (s *service) ListTeams(ctx context.Context, currentUserID, slug string) ([]*Team, error) {
	org, err := s.repo.GetOrganizationBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	teams, err := s.repo.ListTeams(ctx, org.ID)
	if err != nil {
		return nil, err
	}

	if currentUserID != "" {
		for _, team := range teams {
			if tm, err := s.repo.GetTeamMember(ctx, team.ID, currentUserID); err == nil {
				team.CurrentUserRole = tm.Role
			}
		}
	}

	return teams, nil
}

func (s *service) GetTeam(ctx context.Context, currentUserID, slug, teamSlug string) (*Team, error) {
	org, err := s.repo.GetOrganizationBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	team, err := s.repo.GetTeamBySlug(ctx, org.ID, teamSlug)
	if err != nil {
		return nil, err
	}

	if currentUserID != "" {
		if tm, err := s.repo.GetTeamMember(ctx, team.ID, currentUserID); err == nil {
			team.CurrentUserRole = tm.Role
		}
	}

	return team, nil
}

func (s *service) UpdateTeam(ctx context.Context, currentUserID string, isSiteAdmin bool, slug, teamSlug string, req UpdateTeamRequest) (*Team, error) {
	org, err := s.repo.GetOrganizationBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	team, err := s.repo.GetTeamBySlug(ctx, org.ID, teamSlug)
	if err != nil {
		return nil, err
	}

	if !isSiteAdmin {
		canEdit := false
		if member, err := s.repo.GetMember(ctx, org.ID, currentUserID); err == nil && (member.Role == OrgRoleOwner || member.Role == OrgRoleAdmin) {
			canEdit = true
		}
		if !canEdit {
			if tm, err := s.repo.GetTeamMember(ctx, team.ID, currentUserID); err == nil && tm.Role == TeamRoleMaintainer {
				canEdit = true
			}
		}
		if !canEdit {
			return nil, ErrInsufficientPrivilege
		}
	}

	name := strings.TrimSpace(req.Name)
	if name != "" {
		if len(name) < 2 || len(name) > 128 {
			return nil, errors.New("team name must be between 2 and 128 characters")
		}
		team.Name = name
	}
	team.Description = strings.TrimSpace(req.Description)
	if req.Privacy != "" {
		team.Privacy = req.Privacy
	}

	if err := s.repo.UpdateTeam(ctx, team); err != nil {
		return nil, err
	}

	return team, nil
}

func (s *service) DeleteTeam(ctx context.Context, currentUserID string, isSiteAdmin bool, slug, teamSlug string) error {
	org, err := s.repo.GetOrganizationBySlug(ctx, slug)
	if err != nil {
		return err
	}

	team, err := s.repo.GetTeamBySlug(ctx, org.ID, teamSlug)
	if err != nil {
		return err
	}

	if !isSiteAdmin {
		canDelete := false
		if member, err := s.repo.GetMember(ctx, org.ID, currentUserID); err == nil && (member.Role == OrgRoleOwner || member.Role == OrgRoleAdmin) {
			canDelete = true
		}
		if !canDelete {
			if tm, err := s.repo.GetTeamMember(ctx, team.ID, currentUserID); err == nil && tm.Role == TeamRoleMaintainer {
				canDelete = true
			}
		}
		if !canDelete {
			return ErrInsufficientPrivilege
		}
	}

	return s.repo.DeleteTeam(ctx, team.ID)
}

// ----------------------------------------------------------------------------
// Team Members
// ----------------------------------------------------------------------------

func (s *service) ListTeamMembers(ctx context.Context, currentUserID, slug, teamSlug string) ([]*TeamMember, error) {
	org, err := s.repo.GetOrganizationBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	team, err := s.repo.GetTeamBySlug(ctx, org.ID, teamSlug)
	if err != nil {
		return nil, err
	}

	return s.repo.ListTeamMembers(ctx, team.ID)
}

func (s *service) AddTeamMember(ctx context.Context, currentUserID string, isSiteAdmin bool, slug, teamSlug string, req AddTeamMemberRequest) (*TeamMember, error) {
	org, err := s.repo.GetOrganizationBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	team, err := s.repo.GetTeamBySlug(ctx, org.ID, teamSlug)
	if err != nil {
		return nil, err
	}

	if !isSiteAdmin {
		canManage := false
		if member, err := s.repo.GetMember(ctx, org.ID, currentUserID); err == nil && (member.Role == OrgRoleOwner || member.Role == OrgRoleAdmin) {
			canManage = true
		}
		if !canManage {
			if tm, err := s.repo.GetTeamMember(ctx, team.ID, currentUserID); err == nil && tm.Role == TeamRoleMaintainer {
				canManage = true
			}
		}
		if !canManage {
			return nil, ErrInsufficientPrivilege
		}
	}

	targetUser, err := s.authRepo.GetUserByUsername(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Must be an org member first!
	if _, err := s.repo.GetMember(ctx, org.ID, targetUser.ID); err != nil {
		return nil, ErrUserNotOrgMember
	}

	role := req.Role
	if role == "" {
		role = TeamRoleMember
	}

	member := &TeamMember{
		TeamID: team.ID,
		UserID: targetUser.ID,
		Role:   role,
	}

	if err := s.repo.AddTeamMember(ctx, member); err != nil {
		return nil, err
	}

	member.Username = targetUser.Username
	member.DisplayName = targetUser.DisplayName
	member.AvatarURL = targetUser.AvatarURL
	return member, nil
}

func (s *service) UpdateTeamMemberRole(ctx context.Context, currentUserID string, isSiteAdmin bool, slug, teamSlug, targetUsername string, newRole TeamRole) error {
	org, err := s.repo.GetOrganizationBySlug(ctx, slug)
	if err != nil {
		return err
	}

	team, err := s.repo.GetTeamBySlug(ctx, org.ID, teamSlug)
	if err != nil {
		return err
	}

	if !isSiteAdmin {
		canManage := false
		if member, err := s.repo.GetMember(ctx, org.ID, currentUserID); err == nil && (member.Role == OrgRoleOwner || member.Role == OrgRoleAdmin) {
			canManage = true
		}
		if !canManage {
			if tm, err := s.repo.GetTeamMember(ctx, team.ID, currentUserID); err == nil && tm.Role == TeamRoleMaintainer {
				canManage = true
			}
		}
		if !canManage {
			return ErrInsufficientPrivilege
		}
	}

	targetUser, err := s.authRepo.GetUserByUsername(ctx, targetUsername)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	return s.repo.UpdateTeamMemberRole(ctx, team.ID, targetUser.ID, newRole)
}

func (s *service) RemoveTeamMember(ctx context.Context, currentUserID string, isSiteAdmin bool, slug, teamSlug, targetUsername string) error {
	org, err := s.repo.GetOrganizationBySlug(ctx, slug)
	if err != nil {
		return err
	}

	team, err := s.repo.GetTeamBySlug(ctx, org.ID, teamSlug)
	if err != nil {
		return err
	}

	targetUser, err := s.authRepo.GetUserByUsername(ctx, targetUsername)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	isSelf := currentUserID == targetUser.ID

	if !isSiteAdmin && !isSelf {
		canManage := false
		if member, err := s.repo.GetMember(ctx, org.ID, currentUserID); err == nil && (member.Role == OrgRoleOwner || member.Role == OrgRoleAdmin) {
			canManage = true
		}
		if !canManage {
			if tm, err := s.repo.GetTeamMember(ctx, team.ID, currentUserID); err == nil && tm.Role == TeamRoleMaintainer {
				canManage = true
			}
		}
		if !canManage {
			return ErrInsufficientPrivilege
		}
	}

	return s.repo.RemoveTeamMember(ctx, team.ID, targetUser.ID)
}
