package webhooks

import (
	"context"
	"strings"

	"forgehub/apps/api/internal/modules/auth"
	"forgehub/apps/api/internal/modules/orgs"
	"forgehub/apps/api/internal/modules/repos"
)

type Service interface {
	ListWebhooks(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string) ([]*Webhook, error)
	CreateWebhook(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, req CreateWebhookRequest) (*Webhook, error)
	GetWebhook(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug, hookID string) (*Webhook, error)
	UpdateWebhook(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug, hookID string, req UpdateWebhookRequest) (*Webhook, error)
	DeleteWebhook(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug, hookID string) error
	TestPing(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug, hookID string) (*WebhookDelivery, error)
	ListDeliveries(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug, hookID string, limit int) ([]*WebhookDelivery, error)
	GetDelivery(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug, hookID, deliveryID string) (*WebhookDelivery, error)
	Redeliver(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug, hookID, deliveryID string) (*WebhookDelivery, error)
	BroadcastEvent(ctx context.Context, repoID, event, action string, payload any)
}

type service struct {
	store      RepositoryStore
	dispatcher Dispatcher
	reposSvc   repos.Service
	authRepo   auth.Repository
}

func NewService(
	store RepositoryStore,
	dispatcher Dispatcher,
	reposSvc repos.Service,
	authRepo auth.Repository,
) Service {
	return &service{
		store:      store,
		dispatcher: dispatcher,
		reposSvc:   reposSvc,
		authRepo:   authRepo,
	}
}

func (s *service) checkAdminAccess(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string) (*repos.Repository, error) {
	repo, err := s.reposSvc.GetRepository(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, err
	}

	if !isSiteAdmin {
		perm := repo.CurrentUserPermission
		if perm != orgs.PermAdmin && perm != orgs.PermMaintain {
			return nil, ErrAccessDenied
		}
	}
	return repo, nil
}

func (s *service) ListWebhooks(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string) ([]*Webhook, error) {
	repo, err := s.checkAdminAccess(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, err
	}
	hooks, err := s.store.ListWebhooks(ctx, repo.ID)
	if err != nil {
		return nil, err
	}
	// Redact secret values
	for _, h := range hooks {
		h.Secret = ""
	}
	return hooks, nil
}

func (s *service) CreateWebhook(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug string, req CreateWebhookRequest) (*Webhook, error) {
	repo, err := s.checkAdminAccess(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, err
	}

	url := strings.TrimSpace(req.URL)
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return nil, ErrInvalidWebhookURL
	}

	if len(req.Events) == 0 {
		req.Events = []string{EventPush}
	}

	contentType := req.ContentType
	if contentType == "" {
		contentType = "application/json"
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	sslVerification := true
	if req.SSLVerification != nil {
		sslVerification = *req.SSLVerification
	}

	hook := &Webhook{
		RepositoryID:    repo.ID,
		URL:             url,
		ContentType:     contentType,
		Secret:          strings.TrimSpace(req.Secret),
		Events:          req.Events,
		IsActive:        isActive,
		SSLVerification: sslVerification,
	}

	if err := s.store.CreateWebhook(ctx, hook); err != nil {
		return nil, err
	}

	hookForPing := *hook
	// Trigger initial ping verification
	go func() {
		_, _ = s.sendPing(context.Background(), &hookForPing, repo, currentUserID)
	}()

	hook.Secret = ""
	return hook, nil
}

func (s *service) GetWebhook(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug, hookID string) (*Webhook, error) {
	repo, err := s.checkAdminAccess(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, err
	}

	hook, err := s.store.GetWebhook(ctx, hookID)
	if err != nil {
		return nil, err
	}
	if hook.RepositoryID != repo.ID {
		return nil, ErrWebhookNotFound
	}
	hook.Secret = ""
	return hook, nil
}

func (s *service) UpdateWebhook(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug, hookID string, req UpdateWebhookRequest) (*Webhook, error) {
	repo, err := s.checkAdminAccess(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, err
	}

	hook, err := s.store.GetWebhook(ctx, hookID)
	if err != nil {
		return nil, err
	}
	if hook.RepositoryID != repo.ID {
		return nil, ErrWebhookNotFound
	}

	if req.URL != nil {
		u := strings.TrimSpace(*req.URL)
		if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
			return nil, ErrInvalidWebhookURL
		}
		hook.URL = u
	}
	if req.ContentType != nil && *req.ContentType != "" {
		hook.ContentType = *req.ContentType
	}
	if req.Secret != nil {
		hook.Secret = strings.TrimSpace(*req.Secret)
	}
	if len(req.Events) > 0 {
		hook.Events = req.Events
	}
	if req.IsActive != nil {
		hook.IsActive = *req.IsActive
	}
	if req.SSLVerification != nil {
		hook.SSLVerification = *req.SSLVerification
	}

	if err := s.store.UpdateWebhook(ctx, hook); err != nil {
		return nil, err
	}

	hook.Secret = ""
	return hook, nil
}

func (s *service) DeleteWebhook(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug, hookID string) error {
	repo, err := s.checkAdminAccess(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return err
	}

	hook, err := s.store.GetWebhook(ctx, hookID)
	if err != nil {
		return err
	}
	if hook.RepositoryID != repo.ID {
		return ErrWebhookNotFound
	}

	return s.store.DeleteWebhook(ctx, hookID)
}

func (s *service) TestPing(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug, hookID string) (*WebhookDelivery, error) {
	repo, err := s.checkAdminAccess(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, err
	}

	hook, err := s.store.GetWebhook(ctx, hookID)
	if err != nil {
		return nil, err
	}
	if hook.RepositoryID != repo.ID {
		return nil, ErrWebhookNotFound
	}

	return s.sendPing(ctx, hook, repo, currentUserID)
}

func (s *service) sendPing(ctx context.Context, hook *Webhook, repo *repos.Repository, currentUserID string) (*WebhookDelivery, error) {
	var username string
	if u, _ := s.authRepo.GetUserByID(ctx, currentUserID); u != nil {
		username = u.Username
	} else {
		username = "system"
	}

	ping := PingPayload{
		Zen:    "Approachable, fast, and secure developer collaboration on ForgeHub.",
		HookID: hook.ID,
	}
	ping.Hook.Type = "Repository"
	ping.Hook.ID = hook.ID
	ping.Hook.Active = hook.IsActive
	ping.Hook.Events = hook.Events

	ping.Repository.ID = repo.ID
	ping.Repository.Name = repo.Name
	ping.Repository.Slug = repo.Slug

	ping.Sender.Username = username

	return s.dispatcher.Dispatch(ctx, hook, EventPing, "", ping)
}

func (s *service) ListDeliveries(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug, hookID string, limit int) ([]*WebhookDelivery, error) {
	repo, err := s.checkAdminAccess(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, err
	}

	hook, err := s.store.GetWebhook(ctx, hookID)
	if err != nil {
		return nil, err
	}
	if hook.RepositoryID != repo.ID {
		return nil, ErrWebhookNotFound
	}

	return s.store.ListDeliveries(ctx, hookID, limit)
}

func (s *service) GetDelivery(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug, hookID, deliveryID string) (*WebhookDelivery, error) {
	repo, err := s.checkAdminAccess(ctx, currentUserID, isSiteAdmin, owner, repoSlug)
	if err != nil {
		return nil, err
	}

	hook, err := s.store.GetWebhook(ctx, hookID)
	if err != nil {
		return nil, err
	}
	if hook.RepositoryID != repo.ID {
		return nil, ErrWebhookNotFound
	}

	del, err := s.store.GetDelivery(ctx, deliveryID)
	if err != nil {
		return nil, err
	}
	if del.WebhookID != hook.ID {
		return nil, ErrDeliveryNotFound
	}
	return del, nil
}

func (s *service) Redeliver(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repoSlug, hookID, deliveryID string) (*WebhookDelivery, error) {
	del, err := s.GetDelivery(ctx, currentUserID, isSiteAdmin, owner, repoSlug, hookID, deliveryID)
	if err != nil {
		return nil, err
	}

	hook, err := s.store.GetWebhook(ctx, hookID)
	if err != nil {
		return nil, err
	}

	return s.dispatcher.Dispatch(ctx, hook, del.Event, del.Action, del.RequestPayload)
}

func (s *service) BroadcastEvent(ctx context.Context, repoID, event, action string, payload any) {
	hooks, err := s.store.ListActiveWebhooksForEvent(ctx, repoID, event)
	if err != nil || len(hooks) == 0 {
		return
	}

	for _, h := range hooks {
		s.dispatcher.DispatchAsync(h, event, action, payload)
	}
}
