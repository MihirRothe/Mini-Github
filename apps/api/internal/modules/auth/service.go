package auth

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode"

	internalAuth "forgehub/apps/api/internal/auth"
	"forgehub/apps/api/internal/config"
	"forgehub/apps/api/internal/logger"
)

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{1,37}[a-zA-Z0-9]$`)

	ErrInvalidUsername     = errors.New("username must be 3-39 characters, contain only alphanumeric, dashes, or underscores, and cannot start or end with a hyphen/underscore")
	ErrInvalidEmail        = errors.New("a valid email address is required")
	ErrWeakPassword        = errors.New("password must be at least 8 characters long and contain at least one uppercase letter, one lowercase letter, and one number")
	ErrInvalidCredentials  = errors.New("invalid username/email or password")
	ErrAccountSuspended    = errors.New("this account has been suspended")
	ErrSessionExpired      = errors.New("session has expired")
)

type Service struct {
	repo   Repository
	cfg    *config.Config
	params *internalAuth.HashParams
}

func NewService(repo Repository, cfg *config.Config) *Service {
	s := &Service{
		repo:   repo,
		cfg:    cfg,
		params: internalAuth.DefaultParams,
	}

	// Seed development admin user if not present
	if !cfg.IsProduction() {
		s.seedDevAdmin()
	}

	return s
}

func (s *Service) seedDevAdmin() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.repo.GetUserByUsername(ctx, "admin")
	if errors.Is(err, ErrUserNotFound) {
		log := logger.Get()
		log.Info("Seeding development admin user: admin / admin@forgehub.local / ForgeHubAdmin123!")

		hash, err := internalAuth.HashPassword("ForgeHubAdmin123!")
		if err != nil {
			log.Error("Failed to hash dev admin password", "err", err)
			return
		}

		admin := &User{
			Username:     "admin",
			Email:        "admin@forgehub.local",
			PasswordHash: hash,
			DisplayName:  "ForgeHub Administrator",
			Bio:          "Default system administrator account for local development.",
			IsAdmin:      true,
			IsSuspended:  false,
		}

		if err := s.repo.CreateUser(ctx, admin); err != nil {
			log.Error("Failed to seed dev admin user", "err", err)
		}
	}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest, ip, ua string) (*User, string, error) {
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if err := ValidateUsername(req.Username); err != nil {
		return nil, "", err
	}
	if err := ValidateEmail(req.Email); err != nil {
		return nil, "", err
	}
	if err := ValidatePasswordStrength(req.Password); err != nil {
		return nil, "", err
	}

	hash, err := internalAuth.HashPasswordWithParams(req.Password, s.params)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash password: %w", err)
	}

	user := &User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hash,
		DisplayName:  req.Username,
		IsAdmin:      false,
		IsSuspended:  false,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, "", err
	}

	// Create session
	rawToken, err := internalAuth.GenerateRandomToken(32)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate session token: %w", err)
	}

	session := &Session{
		UserID:    user.ID,
		TokenHash: internalAuth.HashToken(rawToken),
		IPAddress: ip,
		UserAgent: ua,
		ExpiresAt: time.Now().UTC().Add(s.cfg.SessionTTL),
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, "", fmt.Errorf("failed to persist session: %w", err)
	}

	_ = s.repo.CreateAuditLog(ctx, user.ID, "user.registered", "user", user.ID, ip, ua, map[string]any{
		"username": user.Username,
		"email":    user.Email,
	})

	return user, rawToken, nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest, ip, ua string) (*User, string, error) {
	req.Login = strings.TrimSpace(req.Login)
	if req.Login == "" || req.Password == "" {
		return nil, "", ErrInvalidCredentials
	}

	user, err := s.repo.GetUserByLogin(ctx, req.Login)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, "", ErrInvalidCredentials
		}
		return nil, "", err
	}

	if user.IsSuspended {
		return nil, "", ErrAccountSuspended
	}

	match, err := internalAuth.VerifyPassword(req.Password, user.PasswordHash)
	if err != nil || !match {
		return nil, "", ErrInvalidCredentials
	}

	// Generate session token
	rawToken, err := internalAuth.GenerateRandomToken(32)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate session token: %w", err)
	}

	session := &Session{
		UserID:    user.ID,
		TokenHash: internalAuth.HashToken(rawToken),
		IPAddress: ip,
		UserAgent: ua,
		ExpiresAt: time.Now().UTC().Add(s.cfg.SessionTTL),
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, "", fmt.Errorf("failed to create session: %w", err)
	}

	_ = s.repo.CreateAuditLog(ctx, user.ID, "user.login", "user", user.ID, ip, ua, map[string]any{
		"method": "password",
	})

	return user, rawToken, nil
}

func (s *Service) Logout(ctx context.Context, rawToken, ip, ua string) error {
	tokenHash := internalAuth.HashToken(rawToken)
	sess, user, err := s.repo.GetSessionByTokenHash(ctx, tokenHash)
	if err == nil && sess != nil {
		_ = s.repo.CreateAuditLog(ctx, user.ID, "user.logout", "session", sess.ID, ip, ua, nil)
	}
	return s.repo.DeleteSession(ctx, tokenHash)
}

func (s *Service) ValidateSession(ctx context.Context, rawToken string) (*User, error) {
	tokenHash := internalAuth.HashToken(rawToken)
	sess, user, err := s.repo.GetSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}
	if time.Now().UTC().After(sess.ExpiresAt) {
		_ = s.repo.DeleteSession(ctx, tokenHash)
		return nil, ErrSessionExpired
	}
	if user.IsSuspended {
		return nil, ErrAccountSuspended
	}
	return user, nil
}

func (s *Service) ValidateAPIToken(ctx context.Context, rawToken string) (*User, *APIToken, error) {
	tokenHash := internalAuth.HashToken(rawToken)
	token, user, err := s.repo.GetAPITokenByHash(ctx, tokenHash)
	if err != nil {
		return nil, nil, err
	}
	if user.IsSuspended {
		return nil, nil, ErrAccountSuspended
	}
	return user, token, nil
}

func (s *Service) AuthenticateCredentials(ctx context.Context, login, password string) (*User, error) {
	login = strings.TrimSpace(login)
	if login == "" || password == "" {
		return nil, ErrInvalidCredentials
	}
	user, err := s.repo.GetUserByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if user.IsSuspended {
		return nil, ErrAccountSuspended
	}
	match, err := internalAuth.VerifyPassword(password, user.PasswordHash)
	if err != nil || !match {
		return nil, ErrInvalidCredentials
	}
	return user, nil
}

func (s *Service) GetUserProfile(ctx context.Context, username string) (*PublicUser, error) {
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	pub := user.ToPublic()
	return &pub, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (*User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if req.DisplayName != nil {
		user.DisplayName = strings.TrimSpace(*req.DisplayName)
	}
	if req.Bio != nil {
		user.Bio = strings.TrimSpace(*req.Bio)
	}
	if req.Location != nil {
		user.Location = strings.TrimSpace(*req.Location)
	}
	if req.Website != nil {
		user.Website = strings.TrimSpace(*req.Website)
	}
	if req.AvatarURL != nil {
		user.AvatarURL = strings.TrimSpace(*req.AvatarURL)
	}

	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Service) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	match, err := internalAuth.VerifyPassword(currentPassword, user.PasswordHash)
	if err != nil || !match {
		return errors.New("current password does not match")
	}

	if err := ValidatePasswordStrength(newPassword); err != nil {
		return err
	}

	newHash, err := internalAuth.HashPasswordWithParams(newPassword, s.params)
	if err != nil {
		return err
	}

	if err := s.repo.UpdatePassword(ctx, userID, newHash); err != nil {
		return err
	}

	// Revoke all existing sessions for security
	return s.repo.DeleteUserSessions(ctx, userID)
}

func (s *Service) CreateAPIToken(ctx context.Context, userID string, req CreateTokenRequest) (*CreateTokenResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("token name is required")
	}

	rawToken, prefix, digest, err := internalAuth.GeneratePAT()
	if err != nil {
		return nil, err
	}

	var expiresAt *time.Time
	if req.ExpiresInDays > 0 {
		exp := time.Now().UTC().AddDate(0, 0, req.ExpiresInDays)
		expiresAt = &exp
	}

	token := &APIToken{
		UserID:      userID,
		Name:        name,
		TokenHash:   digest,
		TokenPrefix: prefix,
		Scopes:      req.Scopes,
		ExpiresAt:   expiresAt,
	}

	if err := s.repo.CreateAPIToken(ctx, token); err != nil {
		return nil, err
	}

	return &CreateTokenResponse{
		Token:       rawToken,
		TokenPrefix: prefix,
		Name:        name,
		Scopes:      req.Scopes,
		ExpiresAt:   expiresAt,
	}, nil
}

func (s *Service) ListAPITokens(ctx context.Context, userID string) ([]*APIToken, error) {
	return s.repo.ListAPITokens(ctx, userID)
}

func (s *Service) DeleteAPIToken(ctx context.Context, userID, tokenID string) error {
	return s.repo.DeleteAPIToken(ctx, userID, tokenID)
}

// ============================================================================
// Validators
// ============================================================================

func ValidateUsername(username string) error {
	if len(username) < 3 || len(username) > 39 {
		return ErrInvalidUsername
	}
	if !usernameRegex.MatchString(username) {
		return ErrInvalidUsername
	}
	return nil
}

func ValidateEmail(email string) error {
	if email == "" {
		return ErrInvalidEmail
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return ErrInvalidEmail
	}
	return nil
}

func ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return ErrWeakPassword
	}

	var hasUpper, hasLower, hasDigit bool
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch) || unicode.IsPunct(ch) || unicode.IsSymbol(ch):
			hasDigit = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit {
		return ErrWeakPassword
	}

	return nil
}
