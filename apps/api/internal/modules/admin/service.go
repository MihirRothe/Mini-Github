package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"time"

	"forgehub/apps/api/internal/modules/auth"
)

var (
	startTime = time.Now()

	ErrUnauthorizedAdmin = errors.New("access denied: requires site administrator privileges")
	ErrCannotDemoteSelf  = errors.New("cannot remove your own administrator status")
)

type Service interface {
	GetStats(ctx context.Context, actor *auth.User) (*AdminStats, error)
	ListAuditLogs(ctx context.Context, actor *auth.User, filter AuditLogFilter) ([]*AuditLogEntry, int, error)
	LogAction(ctx context.Context, actorID *string, actorUsername, action, targetType, targetID, ip, userAgent string, metadata any) error
	ListUsers(ctx context.Context, actor *auth.User, query string, limit, offset int) ([]*AdminUserItem, int, error)
	UpdateUserStatus(ctx context.Context, actor *auth.User, targetUserID string, req UpdateUserAdminRequest, ip, userAgent string) (*AdminUserItem, error)
	ListOrganizations(ctx context.Context, actor *auth.User, limit, offset int) ([]*AdminOrgItem, int, error)
	ListRepositories(ctx context.Context, actor *auth.User, limit, offset int) ([]*AdminRepoItem, int, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) verifyAdmin(actor *auth.User) error {
	if actor == nil || !actor.IsAdmin {
		return ErrUnauthorizedAdmin
	}
	return nil
}

func (s *service) GetStats(ctx context.Context, actor *auth.User) (*AdminStats, error) {
	if err := s.verifyAdmin(actor); err != nil {
		return nil, err
	}

	stats, err := s.repo.GetSystemCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve platform statistics: %w", err)
	}

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	stats.GoroutinesCount = runtime.NumGoroutine()
	stats.AllocatedMemBytes = mem.Alloc
	stats.SysMemBytes = mem.Sys
	stats.GCCycles = mem.NumGC
	stats.UptimeSeconds = int64(time.Since(startTime).Seconds())
	stats.Timestamp = time.Now().UTC()

	return stats, nil
}

func (s *service) ListAuditLogs(ctx context.Context, actor *auth.User, filter AuditLogFilter) ([]*AuditLogEntry, int, error) {
	if err := s.verifyAdmin(actor); err != nil {
		return nil, 0, err
	}
	return s.repo.ListAuditLogs(ctx, filter)
}

func (s *service) LogAction(ctx context.Context, actorID *string, actorUsername, action, targetType, targetID, ip, userAgent string, metadata any) error {
	var metaBytes []byte
	if metadata != nil {
		metaBytes, _ = json.Marshal(metadata)
	} else {
		metaBytes = []byte("{}")
	}

	entry := &AuditLogEntry{
		ActorID:       actorID,
		ActorUsername: actorUsername,
		Action:        action,
		TargetType:    targetType,
		TargetID:      targetID,
		IPAddress:     ip,
		UserAgent:     userAgent,
		Metadata:      json.RawMessage(metaBytes),
		CreatedAt:     time.Now().UTC(),
	}

	return s.repo.CreateAuditLog(ctx, entry)
}

func (s *service) ListUsers(ctx context.Context, actor *auth.User, query string, limit, offset int) ([]*AdminUserItem, int, error) {
	if err := s.verifyAdmin(actor); err != nil {
		return nil, 0, err
	}
	return s.repo.ListUsers(ctx, query, limit, offset)
}

func (s *service) UpdateUserStatus(ctx context.Context, actor *auth.User, targetUserID string, req UpdateUserAdminRequest, ip, userAgent string) (*AdminUserItem, error) {
	if err := s.verifyAdmin(actor); err != nil {
		return nil, err
	}

	// Prevent admin from removing their own admin rights
	if req.IsAdmin != nil && !*req.IsAdmin && actor.ID == targetUserID {
		return nil, ErrCannotDemoteSelf
	}

	updated, err := s.repo.UpdateUserStatus(ctx, targetUserID, req.IsAdmin, req.IsSuspended)
	if err != nil {
		return nil, err
	}

	// Audit logging for state transitions
	actorID := actor.ID
	if req.IsAdmin != nil {
		action := "admin.user_promoted"
		if !*req.IsAdmin {
			action = "admin.user_demoted"
		}
		_ = s.LogAction(ctx, &actorID, actor.Username, action, "user", targetUserID, ip, userAgent, map[string]any{
			"username": updated.Username,
			"is_admin": *req.IsAdmin,
		})
	}

	if req.IsSuspended != nil {
		action := "admin.user_suspended"
		if !*req.IsSuspended {
			action = "admin.user_unsuspended"
		}
		_ = s.LogAction(ctx, &actorID, actor.Username, action, "user", targetUserID, ip, userAgent, map[string]any{
			"username":     updated.Username,
			"is_suspended": *req.IsSuspended,
		})
	}

	return updated, nil
}

func (s *service) ListOrganizations(ctx context.Context, actor *auth.User, limit, offset int) ([]*AdminOrgItem, int, error) {
	if err := s.verifyAdmin(actor); err != nil {
		return nil, 0, err
	}
	return s.repo.ListOrganizations(ctx, limit, offset)
}

func (s *service) ListRepositories(ctx context.Context, actor *auth.User, limit, offset int) ([]*AdminRepoItem, int, error) {
	if err := s.verifyAdmin(actor); err != nil {
		return nil, 0, err
	}
	return s.repo.ListRepositories(ctx, limit, offset)
}
