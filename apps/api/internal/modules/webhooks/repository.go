package webhooks

import (
	"context"
	"database/sql"
	"encoding/json"
	"sort"
	"sync"
	"time"

	"forgehub/apps/api/internal/database"

	"github.com/google/uuid"
)

type RepositoryStore interface {
	CreateWebhook(ctx context.Context, hook *Webhook) error
	GetWebhook(ctx context.Context, hookID string) (*Webhook, error)
	ListWebhooks(ctx context.Context, repoID string) ([]*Webhook, error)
	ListActiveWebhooksForEvent(ctx context.Context, repoID string, event string) ([]*Webhook, error)
	UpdateWebhook(ctx context.Context, hook *Webhook) error
	DeleteWebhook(ctx context.Context, hookID string) error

	CreateDelivery(ctx context.Context, delivery *WebhookDelivery) error
	GetDelivery(ctx context.Context, deliveryID string) (*WebhookDelivery, error)
	ListDeliveries(ctx context.Context, hookID string, limit int) ([]*WebhookDelivery, error)
}

func NewRepositoryStore(db *database.DB) RepositoryStore {
	if db == nil || db.IsStandalone() {
		return NewMemoryRepositoryStore()
	}
	return &sqlRepositoryStore{db: db}
}

// -------------------------------------------------------------------------
// SQL Implementation (PostgreSQL)
// -------------------------------------------------------------------------

type sqlRepositoryStore struct {
	db *database.DB
}

func (s *sqlRepositoryStore) CreateWebhook(ctx context.Context, hook *Webhook) error {
	if hook.ID == "" {
		hook.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	hook.CreatedAt = now
	hook.UpdatedAt = now

	query := `
		INSERT INTO webhooks (
			id, repository_id, url, content_type, secret, events, is_active, ssl_verification, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := s.db.ExecContext(ctx, query,
		hook.ID, hook.RepositoryID, hook.URL, hook.ContentType, hook.Secret,
		hook.Events, hook.IsActive, hook.SSLVerification, hook.CreatedAt, hook.UpdatedAt,
	)
	return err
}

func (s *sqlRepositoryStore) GetWebhook(ctx context.Context, hookID string) (*Webhook, error) {
	query := `
		SELECT id, repository_id, url, content_type, secret, events, is_active, ssl_verification, created_at, updated_at
		FROM webhooks
		WHERE id = $1
	`
	var hook Webhook
	var secret sql.NullString
	var events []string

	err := s.db.QueryRowContext(ctx, query, hookID).Scan(
		&hook.ID, &hook.RepositoryID, &hook.URL, &hook.ContentType, &secret,
		&events, &hook.IsActive, &hook.SSLVerification, &hook.CreatedAt, &hook.UpdatedAt,
	)
	if err != nil {
		return nil, ErrWebhookNotFound
	}
	if secret.Valid {
		hook.Secret = secret.String
		hook.HasSecret = hook.Secret != ""
	}
	hook.Events = events
	return &hook, nil
}

func (s *sqlRepositoryStore) ListWebhooks(ctx context.Context, repoID string) ([]*Webhook, error) {
	query := `
		SELECT w.id, w.repository_id, w.url, w.content_type, w.secret, w.events, w.is_active, w.ssl_verification, w.created_at, w.updated_at,
		       (SELECT d.response_status_code FROM webhook_deliveries d WHERE d.webhook_id = w.id ORDER BY d.delivered_at DESC LIMIT 1) AS last_status
		FROM webhooks w
		WHERE w.repository_id = $1
		ORDER BY w.created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hooks []*Webhook
	for rows.Next() {
		var hook Webhook
		var secret sql.NullString
		var events []string
		var lastStatus sql.NullInt32

		if err := rows.Scan(
			&hook.ID, &hook.RepositoryID, &hook.URL, &hook.ContentType, &secret,
			&events, &hook.IsActive, &hook.SSLVerification, &hook.CreatedAt, &hook.UpdatedAt,
			&lastStatus,
		); err != nil {
			return nil, err
		}

		if secret.Valid {
			hook.Secret = secret.String
			hook.HasSecret = hook.Secret != ""
		}
		hook.Events = events
		if lastStatus.Valid {
			statusVal := int(lastStatus.Int32)
			hook.LastStatus = &statusVal
		}
		hooks = append(hooks, &hook)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return hooks, nil
}

func (s *sqlRepositoryStore) ListActiveWebhooksForEvent(ctx context.Context, repoID string, event string) ([]*Webhook, error) {
	query := `
		SELECT id, repository_id, url, content_type, secret, events, is_active, ssl_verification, created_at, updated_at
		FROM webhooks
		WHERE repository_id = $1 AND is_active = TRUE AND ($2 = ANY(events) OR '*' = ANY(events))
	`
	rows, err := s.db.QueryContext(ctx, query, repoID, event)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hooks []*Webhook
	for rows.Next() {
		var hook Webhook
		var secret sql.NullString
		var events []string

		if err := rows.Scan(
			&hook.ID, &hook.RepositoryID, &hook.URL, &hook.ContentType, &secret,
			&events, &hook.IsActive, &hook.SSLVerification, &hook.CreatedAt, &hook.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if secret.Valid {
			hook.Secret = secret.String
			hook.HasSecret = hook.Secret != ""
		}
		hook.Events = events
		hooks = append(hooks, &hook)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return hooks, nil
}

func (s *sqlRepositoryStore) UpdateWebhook(ctx context.Context, hook *Webhook) error {
	hook.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE webhooks
		SET url = $1, content_type = $2, secret = $3, events = $4, is_active = $5, ssl_verification = $6, updated_at = $7
		WHERE id = $8
	`
	_, err := s.db.ExecContext(ctx, query,
		hook.URL, hook.ContentType, hook.Secret, hook.Events, hook.IsActive, hook.SSLVerification, hook.UpdatedAt, hook.ID,
	)
	return err
}

func (s *sqlRepositoryStore) DeleteWebhook(ctx context.Context, hookID string) error {
	query := `DELETE FROM webhooks WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, hookID)
	return err
}

func (s *sqlRepositoryStore) CreateDelivery(ctx context.Context, delivery *WebhookDelivery) error {
	if delivery.ID == "" {
		delivery.ID = uuid.NewString()
	}
	delivery.DeliveredAt = time.Now().UTC()

	reqHeadersJSON, _ := json.Marshal(delivery.RequestHeaders)
	reqPayloadJSON, _ := json.Marshal(delivery.RequestPayload)
	resHeadersJSON, _ := json.Marshal(delivery.ResponseHeaders)

	query := `
		INSERT INTO webhook_deliveries (
			id, webhook_id, event, action, request_headers, request_payload,
			response_status_code, response_headers, response_body, duration_ms,
			is_success, error_message, delivered_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := s.db.ExecContext(ctx, query,
		delivery.ID, delivery.WebhookID, delivery.Event, delivery.Action,
		string(reqHeadersJSON), string(reqPayloadJSON), delivery.ResponseStatusCode,
		string(resHeadersJSON), delivery.ResponseBody, delivery.DurationMs,
		delivery.IsSuccess, delivery.ErrorMessage, delivery.DeliveredAt,
	)
	return err
}

func (s *sqlRepositoryStore) GetDelivery(ctx context.Context, deliveryID string) (*WebhookDelivery, error) {
	query := `
		SELECT id, webhook_id, event, action, request_headers, request_payload,
		       response_status_code, response_headers, response_body, duration_ms,
		       is_success, error_message, delivered_at
		FROM webhook_deliveries
		WHERE id = $1
	`
	var d WebhookDelivery
	var action sql.NullString
	var reqHeadersRaw, reqPayloadRaw, resHeadersRaw []byte
	var statusCode sql.NullInt32
	var resBody sql.NullString
	var errMsg sql.NullString

	err := s.db.QueryRowContext(ctx, query, deliveryID).Scan(
		&d.ID, &d.WebhookID, &d.Event, &action, &reqHeadersRaw, &reqPayloadRaw,
		&statusCode, &resHeadersRaw, &resBody, &d.DurationMs,
		&d.IsSuccess, &errMsg, &d.DeliveredAt,
	)
	if err != nil {
		return nil, ErrDeliveryNotFound
	}

	if action.Valid {
		d.Action = action.String
	}
	if statusCode.Valid {
		code := int(statusCode.Int32)
		d.ResponseStatusCode = &code
	}
	if resBody.Valid {
		d.ResponseBody = &resBody.String
	}
	if errMsg.Valid {
		d.ErrorMessage = &errMsg.String
	}

	_ = json.Unmarshal(reqHeadersRaw, &d.RequestHeaders)
	_ = json.Unmarshal(reqPayloadRaw, &d.RequestPayload)
	_ = json.Unmarshal(resHeadersRaw, &d.ResponseHeaders)

	return &d, nil
}

func (s *sqlRepositoryStore) ListDeliveries(ctx context.Context, hookID string, limit int) ([]*WebhookDelivery, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}

	query := `
		SELECT id, webhook_id, event, action, request_headers, request_payload,
		       response_status_code, response_headers, response_body, duration_ms,
		       is_success, error_message, delivered_at
		FROM webhook_deliveries
		WHERE webhook_id = $1
		ORDER BY delivered_at DESC
		LIMIT $2
	`
	rows, err := s.db.QueryContext(ctx, query, hookID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliveries []*WebhookDelivery
	for rows.Next() {
		var d WebhookDelivery
		var action sql.NullString
		var reqHeadersRaw, reqPayloadRaw, resHeadersRaw []byte
		var statusCode sql.NullInt32
		var resBody sql.NullString
		var errMsg sql.NullString

		if err := rows.Scan(
			&d.ID, &d.WebhookID, &d.Event, &action, &reqHeadersRaw, &reqPayloadRaw,
			&statusCode, &resHeadersRaw, &resBody, &d.DurationMs,
			&d.IsSuccess, &errMsg, &d.DeliveredAt,
		); err != nil {
			return nil, err
		}

		if action.Valid {
			d.Action = action.String
		}
		if statusCode.Valid {
			code := int(statusCode.Int32)
			d.ResponseStatusCode = &code
		}
		if resBody.Valid {
			d.ResponseBody = &resBody.String
		}
		if errMsg.Valid {
			d.ErrorMessage = &errMsg.String
		}

		_ = json.Unmarshal(reqHeadersRaw, &d.RequestHeaders)
		_ = json.Unmarshal(reqPayloadRaw, &d.RequestPayload)
		_ = json.Unmarshal(resHeadersRaw, &d.ResponseHeaders)

		deliveries = append(deliveries, &d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return deliveries, nil
}

// -------------------------------------------------------------------------
// In-Memory Implementation (for Tests & Zero-Dependency Mode)
// -------------------------------------------------------------------------

type memoryRepositoryStore struct {
	mu         sync.RWMutex
	hooks      map[string]*Webhook
	deliveries map[string][]*WebhookDelivery // keyed by webhook_id
}

func NewMemoryRepositoryStore() RepositoryStore {
	return &memoryRepositoryStore{
		hooks:      make(map[string]*Webhook),
		deliveries: make(map[string][]*WebhookDelivery),
	}
}

func (m *memoryRepositoryStore) CreateWebhook(ctx context.Context, hook *Webhook) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if hook.ID == "" {
		hook.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	hook.CreatedAt = now
	hook.UpdatedAt = now
	hook.HasSecret = hook.Secret != ""

	copied := *hook
	m.hooks[hook.ID] = &copied
	return nil
}

func (m *memoryRepositoryStore) GetWebhook(ctx context.Context, hookID string) (*Webhook, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	hook, ok := m.hooks[hookID]
	if !ok {
		return nil, ErrWebhookNotFound
	}
	copied := *hook
	return &copied, nil
}

func (m *memoryRepositoryStore) ListWebhooks(ctx context.Context, repoID string) ([]*Webhook, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var out []*Webhook
	for _, h := range m.hooks {
		if h.RepositoryID == repoID {
			copied := *h
			// Attach last status if any
			if dels, ok := m.deliveries[h.ID]; ok && len(dels) > 0 {
				copied.LastStatus = dels[len(dels)-1].ResponseStatusCode
			}
			out = append(out, &copied)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func (m *memoryRepositoryStore) ListActiveWebhooksForEvent(ctx context.Context, repoID string, event string) ([]*Webhook, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var out []*Webhook
	for _, h := range m.hooks {
		if h.RepositoryID == repoID && h.IsActive {
			matches := false
			for _, e := range h.Events {
				if e == event || e == "*" {
					matches = true
					break
				}
			}
			if matches {
				copied := *h
				out = append(out, &copied)
			}
		}
	}
	return out, nil
}

func (m *memoryRepositoryStore) UpdateWebhook(ctx context.Context, hook *Webhook) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.hooks[hook.ID]
	if !ok {
		return ErrWebhookNotFound
	}
	hook.UpdatedAt = time.Now().UTC()
	hook.CreatedAt = existing.CreatedAt
	hook.HasSecret = hook.Secret != ""

	copied := *hook
	m.hooks[hook.ID] = &copied
	return nil
}

func (m *memoryRepositoryStore) DeleteWebhook(ctx context.Context, hookID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.hooks, hookID)
	delete(m.deliveries, hookID)
	return nil
}

func (m *memoryRepositoryStore) CreateDelivery(ctx context.Context, delivery *WebhookDelivery) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if delivery.ID == "" {
		delivery.ID = uuid.NewString()
	}
	delivery.DeliveredAt = time.Now().UTC()

	copied := *delivery
	m.deliveries[delivery.WebhookID] = append(m.deliveries[delivery.WebhookID], &copied)
	return nil
}

func (m *memoryRepositoryStore) GetDelivery(ctx context.Context, deliveryID string) (*WebhookDelivery, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, list := range m.deliveries {
		for _, d := range list {
			if d.ID == deliveryID {
				copied := *d
				return &copied, nil
			}
		}
	}
	return nil, ErrDeliveryNotFound
}

func (m *memoryRepositoryStore) ListDeliveries(ctx context.Context, hookID string, limit int) ([]*WebhookDelivery, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := m.deliveries[hookID]
	if len(list) == 0 {
		return []*WebhookDelivery{}, nil
	}

	copied := make([]*WebhookDelivery, len(list))
	copy(copied, list)
	// Sort newest first
	sort.Slice(copied, func(i, j int) bool {
		return copied[i].DeliveredAt.After(copied[j].DeliveredAt)
	})

	if limit > 0 && len(copied) > limit {
		copied = copied[:limit]
	}
	return copied, nil
}
