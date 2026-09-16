package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"forgehub/apps/api/internal/database"

	"github.com/google/uuid"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrSessionNotFound = errors.New("session not found")
	ErrTokenNotFound   = errors.New("token not found")
	ErrDuplicateUser   = errors.New("user with this username or email already exists")
)

type Repository interface {
	CreateUser(ctx context.Context, u *User) error
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByLogin(ctx context.Context, login string) (*User, error)
	UpdateUser(ctx context.Context, u *User) error
	UpdatePassword(ctx context.Context, userID, newHash string) error

	CreateSession(ctx context.Context, s *Session) error
	GetSessionByTokenHash(ctx context.Context, tokenHash string) (*Session, *User, error)
	DeleteSession(ctx context.Context, tokenHash string) error
	DeleteUserSessions(ctx context.Context, userID string) error

	CreateAPIToken(ctx context.Context, t *APIToken) error
	ListAPITokens(ctx context.Context, userID string) ([]*APIToken, error)
	GetAPITokenByHash(ctx context.Context, tokenHash string) (*APIToken, *User, error)
	DeleteAPIToken(ctx context.Context, userID, tokenID string) error

	CreateAuditLog(ctx context.Context, actorID, action, targetType, targetID, ip, ua string, metadata map[string]any) error
}

func NewRepository(db *database.DB) Repository {
	if db.IsStandalone() {
		return NewMemoryRepository()
	}
	return &sqlRepository{db: db}
}

// ============================================================================
// SQL Repository (PostgreSQL)
// ============================================================================

type sqlRepository struct {
	db *database.DB
}

func (r *sqlRepository) CreateUser(ctx context.Context, u *User) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	u.CreatedAt = time.Now().UTC()
	u.UpdatedAt = u.CreatedAt

	query := `
		INSERT INTO users (id, username, email, password_hash, display_name, avatar_url, bio, location, website, is_admin, is_suspended, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := r.db.ExecContext(ctx, query,
		u.ID, u.Username, u.Email, u.PasswordHash, u.DisplayName, u.AvatarURL,
		u.Bio, u.Location, u.Website, u.IsAdmin, u.IsSuspended, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return ErrDuplicateUser
		}
		return err
	}
	return nil
}

func (r *sqlRepository) GetUserByID(ctx context.Context, id string) (*User, error) {
	query := `SELECT id, username, email, password_hash, display_name, avatar_url, bio, location, website, is_admin, is_suspended, created_at, updated_at FROM users WHERE id = $1`
	var u User
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.DisplayName, &u.AvatarURL,
		&u.Bio, &u.Location, &u.Website, &u.IsAdmin, &u.IsSuspended, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	return &u, err
}

func (r *sqlRepository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	query := `SELECT id, username, email, password_hash, display_name, avatar_url, bio, location, website, is_admin, is_suspended, created_at, updated_at FROM users WHERE LOWER(username) = LOWER($1)`
	var u User
	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.DisplayName, &u.AvatarURL,
		&u.Bio, &u.Location, &u.Website, &u.IsAdmin, &u.IsSuspended, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	return &u, err
}

func (r *sqlRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `SELECT id, username, email, password_hash, display_name, avatar_url, bio, location, website, is_admin, is_suspended, created_at, updated_at FROM users WHERE LOWER(email) = LOWER($1)`
	var u User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.DisplayName, &u.AvatarURL,
		&u.Bio, &u.Location, &u.Website, &u.IsAdmin, &u.IsSuspended, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	return &u, err
}

func (r *sqlRepository) GetUserByLogin(ctx context.Context, login string) (*User, error) {
	if strings.Contains(login, "@") {
		return r.GetUserByEmail(ctx, login)
	}
	return r.GetUserByUsername(ctx, login)
}

func (r *sqlRepository) UpdateUser(ctx context.Context, u *User) error {
	u.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE users
		SET display_name = $1, bio = $2, location = $3, website = $4, avatar_url = $5, updated_at = $6
		WHERE id = $7
	`
	_, err := r.db.ExecContext(ctx, query, u.DisplayName, u.Bio, u.Location, u.Website, u.AvatarURL, u.UpdatedAt, u.ID)
	return err
}

func (r *sqlRepository) UpdatePassword(ctx context.Context, userID, newHash string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, newHash, userID)
	return err
}

func (r *sqlRepository) CreateSession(ctx context.Context, s *Session) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	s.CreatedAt = time.Now().UTC()
	query := `
		INSERT INTO sessions (id, user_id, token_hash, ip_address, user_agent, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query, s.ID, s.UserID, s.TokenHash, s.IPAddress, s.UserAgent, s.ExpiresAt, s.CreatedAt)
	return err
}

func (r *sqlRepository) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*Session, *User, error) {
	query := `
		SELECT s.id, s.user_id, s.token_hash, s.ip_address, s.user_agent, s.expires_at, s.created_at,
		       u.id, u.username, u.email, u.password_hash, u.display_name, u.avatar_url, u.bio, u.location, u.website, u.is_admin, u.is_suspended, u.created_at, u.updated_at
		FROM sessions s
		JOIN users u ON s.user_id = u.id
		WHERE s.token_hash = $1 AND s.expires_at > NOW()
	`
	var s Session
	var u User
	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&s.ID, &s.UserID, &s.TokenHash, &s.IPAddress, &s.UserAgent, &s.ExpiresAt, &s.CreatedAt,
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.DisplayName, &u.AvatarURL,
		&u.Bio, &u.Location, &u.Website, &u.IsAdmin, &u.IsSuspended, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, ErrSessionNotFound
	}
	return &s, &u, err
}

func (r *sqlRepository) DeleteSession(ctx context.Context, tokenHash string) error {
	query := `DELETE FROM sessions WHERE token_hash = $1`
	_, err := r.db.ExecContext(ctx, query, tokenHash)
	return err
}

func (r *sqlRepository) DeleteUserSessions(ctx context.Context, userID string) error {
	query := `DELETE FROM sessions WHERE user_id = $1`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

func (r *sqlRepository) CreateAPIToken(ctx context.Context, t *APIToken) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	t.CreatedAt = time.Now().UTC()
	query := `
		INSERT INTO api_tokens (id, user_id, name, token_hash, token_prefix, scopes, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query, t.ID, t.UserID, t.Name, t.TokenHash, t.TokenPrefix, t.Scopes, t.ExpiresAt, t.CreatedAt)
	return err
}

func (r *sqlRepository) ListAPITokens(ctx context.Context, userID string) ([]*APIToken, error) {
	query := `
		SELECT id, user_id, name, token_prefix, scopes, last_used_at, expires_at, created_at
		FROM api_tokens
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*APIToken
	for rows.Next() {
		var t APIToken
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.TokenPrefix, &t.Scopes, &t.LastUsedAt, &t.ExpiresAt, &t.CreatedAt); err != nil {
			return nil, err
		}
		tokens = append(tokens, &t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tokens, nil
}

func (r *sqlRepository) GetAPITokenByHash(ctx context.Context, tokenHash string) (*APIToken, *User, error) {
	query := `
		SELECT t.id, t.user_id, t.name, t.token_prefix, t.scopes, t.last_used_at, t.expires_at, t.created_at,
		       u.id, u.username, u.email, u.password_hash, u.display_name, u.avatar_url, u.bio, u.location, u.website, u.is_admin, u.is_suspended, u.created_at, u.updated_at
		FROM api_tokens t
		JOIN users u ON t.user_id = u.id
		WHERE t.token_hash = $1 AND (t.expires_at IS NULL OR t.expires_at > NOW())
	`
	var t APIToken
	var u User
	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&t.ID, &t.UserID, &t.Name, &t.TokenPrefix, &t.Scopes, &t.LastUsedAt, &t.ExpiresAt, &t.CreatedAt,
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.DisplayName, &u.AvatarURL,
		&u.Bio, &u.Location, &u.Website, &u.IsAdmin, &u.IsSuspended, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, ErrTokenNotFound
	}

	// Update last_used_at asynchronously
	go func(tokenID string) {
		_, _ = r.db.ExecContext(context.Background(), "UPDATE api_tokens SET last_used_at = NOW() WHERE id = $1", tokenID)
	}(t.ID)

	return &t, &u, nil
}

func (r *sqlRepository) DeleteAPIToken(ctx context.Context, userID, tokenID string) error {
	query := `DELETE FROM api_tokens WHERE user_id = $1 AND id = $2`
	_, err := r.db.ExecContext(ctx, query, userID, tokenID)
	return err
}

func (r *sqlRepository) CreateAuditLog(ctx context.Context, actorID, action, targetType, targetID, ip, ua string, metadata map[string]any) error {
	metaJSON, _ := json.Marshal(metadata)
	query := `
		INSERT INTO audit_logs (id, actor_id, action, target_type, target_id, ip_address, user_agent, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
	`
	var actorParam any = actorID
	if actorID == "" {
		actorParam = nil
	}
	_, err := r.db.ExecContext(ctx, query, uuid.New().String(), actorParam, action, targetType, targetID, ip, ua, metaJSON)
	return err
}

// ============================================================================
// In-Memory Repository (Standalone fallback for zero-dependency local dev)
// ============================================================================

type MemoryRepository struct {
	mu       sync.RWMutex
	users    map[string]*User      // key: ID
	sessions map[string]*Session   // key: TokenHash
	tokens   map[string]*APIToken  // key: TokenHash
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		users:    make(map[string]*User),
		sessions: make(map[string]*Session),
		tokens:   make(map[string]*APIToken),
	}
}

func (m *MemoryRepository) CreateUser(ctx context.Context, u *User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, existing := range m.users {
		if strings.EqualFold(existing.Username, u.Username) || strings.EqualFold(existing.Email, u.Email) {
			return ErrDuplicateUser
		}
	}

	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	u.CreatedAt = time.Now().UTC()
	u.UpdatedAt = u.CreatedAt

	m.users[u.ID] = u
	return nil
}

func (m *MemoryRepository) GetUserByID(ctx context.Context, id string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	u, ok := m.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (m *MemoryRepository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, u := range m.users {
		if strings.EqualFold(u.Username, username) {
			return u, nil
		}
	}
	return nil, ErrUserNotFound
}

func (m *MemoryRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, u := range m.users {
		if strings.EqualFold(u.Email, email) {
			return u, nil
		}
	}
	return nil, ErrUserNotFound
}

func (m *MemoryRepository) GetUserByLogin(ctx context.Context, login string) (*User, error) {
	if strings.Contains(login, "@") {
		return m.GetUserByEmail(ctx, login)
	}
	return m.GetUserByUsername(ctx, login)
}

func (m *MemoryRepository) UpdateUser(ctx context.Context, u *User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.users[u.ID]
	if !ok {
		return ErrUserNotFound
	}

	existing.DisplayName = u.DisplayName
	existing.Bio = u.Bio
	existing.Location = u.Location
	existing.Website = u.Website
	existing.AvatarURL = u.AvatarURL
	existing.UpdatedAt = time.Now().UTC()

	return nil
}

func (m *MemoryRepository) UpdatePassword(ctx context.Context, userID, newHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.users[userID]
	if !ok {
		return ErrUserNotFound
	}
	existing.PasswordHash = newHash
	existing.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *MemoryRepository) CreateSession(ctx context.Context, s *Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	s.CreatedAt = time.Now().UTC()
	m.sessions[s.TokenHash] = s
	return nil
}

func (m *MemoryRepository) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*Session, *User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, ok := m.sessions[tokenHash]
	if !ok || time.Now().After(s.ExpiresAt) {
		return nil, nil, ErrSessionNotFound
	}

	u, ok := m.users[s.UserID]
	if !ok {
		return nil, nil, ErrUserNotFound
	}

	return s, u, nil
}

func (m *MemoryRepository) DeleteSession(ctx context.Context, tokenHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, tokenHash)
	return nil
}

func (m *MemoryRepository) DeleteUserSessions(ctx context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for k, s := range m.sessions {
		if s.UserID == userID {
			delete(m.sessions, k)
		}
	}
	return nil
}

func (m *MemoryRepository) CreateAPIToken(ctx context.Context, t *APIToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	t.CreatedAt = time.Now().UTC()
	m.tokens[t.TokenHash] = t
	return nil
}

func (m *MemoryRepository) ListAPITokens(ctx context.Context, userID string) ([]*APIToken, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*APIToken
	for _, t := range m.tokens {
		if t.UserID == userID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (m *MemoryRepository) GetAPITokenByHash(ctx context.Context, tokenHash string) (*APIToken, *User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	t, ok := m.tokens[tokenHash]
	if !ok || (t.ExpiresAt != nil && time.Now().After(*t.ExpiresAt)) {
		return nil, nil, ErrTokenNotFound
	}

	u, ok := m.users[t.UserID]
	if !ok {
		return nil, nil, ErrUserNotFound
	}

	now := time.Now().UTC()
	t.LastUsedAt = &now

	return t, u, nil
}

func (m *MemoryRepository) DeleteAPIToken(ctx context.Context, userID, tokenID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for k, t := range m.tokens {
		if t.UserID == userID && t.ID == tokenID {
			delete(m.tokens, k)
			return nil
		}
	}
	return nil
}

func (m *MemoryRepository) CreateAuditLog(ctx context.Context, actorID, action, targetType, targetID, ip, ua string, metadata map[string]any) error {
	return nil
}
