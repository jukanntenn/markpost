package auth

import (
	"context"
	"fmt"
	"testing"
	"time"

	"markpost/internal/domain/user"
)

type mockUserRepository struct {
	users      map[int]*user.User
	postKeyMap map[string]*user.User
	nextID     int
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users:      make(map[int]*user.User),
		postKeyMap: make(map[string]*user.User),
		nextID:     1,
	}
}

func (m *mockUserRepository) GetByID(ctx context.Context, id int) (*user.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, user.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepository) GetByPostKey(ctx context.Context, postKey string) (*user.User, error) {
	u, ok := m.postKeyMap[postKey]
	if !ok {
		return nil, user.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepository) GetByUsername(ctx context.Context, username string) (*user.User, error) {
	for _, u := range m.users {
		if u.Username == username {
			return u, nil
		}
	}
	return nil, user.ErrNotFound
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, user.ErrNotFound
}

func (m *mockUserRepository) Create(ctx context.Context, email, username, password string) (*user.User, error) {
	for _, u := range m.users {
		if u.Email == email || u.Username == username {
			return nil, nil
		}
	}
	u := &user.User{
		ID:       m.nextID,
		Email:    email,
		Username: username,
		Password: password,
		PostKey:  "test-post-key",
		IsActive: true,
	}
	m.users[m.nextID] = u
	m.postKeyMap[u.PostKey] = u
	m.nextID++
	return u, nil
}

func (m *mockUserRepository) ValidatePassword(ctx context.Context, email, password string) (*user.User, error) {
	u, err := m.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if u.Password != password {
		return nil, fmt.Errorf("invalid password")
	}
	return u, nil
}

func (m *mockUserRepository) SetPassword(ctx context.Context, userID int, password string) error {
	u, ok := m.users[userID]
	if !ok {
		return user.ErrNotFound
	}
	u.Password = password
	return nil
}

func (m *mockUserRepository) SetRole(ctx context.Context, userID int, role user.Role) error {
	u, ok := m.users[userID]
	if !ok {
		return user.ErrNotFound
	}
	u.Role = role
	return nil
}

func (m *mockUserRepository) GetByGitHubID(ctx context.Context, githubID int64) (*user.User, error) {
	return nil, user.ErrNotFound
}

func (m *mockUserRepository) GetOrCreateFromGitHub(ctx context.Context, githubUser *user.GitHubUser) (*user.User, error) {
	return nil, nil
}

func (m *mockUserRepository) DeleteByID(ctx context.Context, userID int) (int64, error) {
	delete(m.users, userID)
	return 1, nil
}

func (m *mockUserRepository) GetAll(ctx context.Context, offset, limit int) ([]user.User, error) {
	return nil, nil
}

func (m *mockUserRepository) Count(ctx context.Context) (int64, error) {
	return int64(len(m.users)), nil
}

func (m *mockUserRepository) CreateFromGitHub(ctx context.Context, githubUser *user.GitHubUser) (*user.User, error) {
	u := &user.User{
		ID:       m.nextID,
		Email:    githubUser.Email,
		Username: githubUser.Login,
		PostKey:  "test-post-key",
		GitHubID: &githubUser.ID,
		IsActive: true,
	}
	m.users[m.nextID] = u
	m.postKeyMap[u.PostKey] = u
	m.nextID++
	return u, nil
}

func (m *mockUserRepository) UpdateLastLoginAt(ctx context.Context, userID int, lastLoginAt time.Time) error {
	return nil
}

type mockTokenRepository struct {
	refreshTokens map[string]*user.RefreshToken
	blacklist     map[string]bool
}

func newMockTokenRepository() *mockTokenRepository {
	return &mockTokenRepository{
		refreshTokens: make(map[string]*user.RefreshToken),
		blacklist:     make(map[string]bool),
	}
}

func (m *mockTokenRepository) StoreRefreshToken(ctx context.Context, userID int, tokenHash string, expiresAt time.Time) error {
	m.refreshTokens[tokenHash] = &user.RefreshToken{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}
	return nil
}

func (m *mockTokenRepository) GetRefreshToken(ctx context.Context, tokenHash string) (*user.RefreshToken, error) {
	token, ok := m.refreshTokens[tokenHash]
	if !ok {
		return nil, user.ErrNotFound
	}
	return token, nil
}

func (m *mockTokenRepository) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	delete(m.refreshTokens, tokenHash)
	return nil
}

func (m *mockTokenRepository) DeleteRefreshTokensByUserID(ctx context.Context, userID int) error {
	for hash, token := range m.refreshTokens {
		if token.UserID == userID {
			delete(m.refreshTokens, hash)
		}
	}
	return nil
}

func (m *mockTokenRepository) StoreBlacklistedToken(ctx context.Context, tokenHash string, expiresAt time.Time) error {
	m.blacklist[tokenHash] = true
	return nil
}

func (m *mockTokenRepository) IsTokenBlacklisted(ctx context.Context, tokenHash string) (bool, error) {
	return m.blacklist[tokenHash], nil
}

func (m *mockTokenRepository) CleanupExpiredTokens(ctx context.Context) error {
	return nil
}

func TestAuthService_LoginWithEmail(t *testing.T) {
	t.Run("returns tokens for valid credentials", func(t *testing.T) {
		mockRepo := newMockUserRepository()
		mockTokenRepo := newMockTokenRepository()
		ctx := context.Background()

		testUser, _ := mockRepo.Create(ctx, "test@example.com", "testuser", "correctpassword")
		mockRepo.SetRole(ctx, testUser.ID, user.RoleUser)

		jwtSvc := NewJWTService("test-access-secret", "test-refresh-secret", time.Hour, time.Hour*24)
		authSvc := NewAuthService(mockRepo, mockTokenRepo, nil, jwtSvc, "markpost")

		u, tokens, err := authSvc.LoginWithEmail(ctx, "test@example.com", "correctpassword")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if u == nil {
			t.Fatal("expected user, got nil")
		}
		if tokens == nil {
			t.Fatal("expected tokens, got nil")
		}
		if tokens.AccessToken == "" {
			t.Error("expected access token, got empty")
		}
		if tokens.RefreshToken == "" {
			t.Error("expected refresh token, got empty")
		}
	})

	t.Run("returns error for invalid email", func(t *testing.T) {
		mockRepo := newMockUserRepository()
		mockTokenRepo := newMockTokenRepository()
		ctx := context.Background()

		jwtSvc := NewJWTService("test-access-secret", "test-refresh-secret", time.Hour, time.Hour*24)
		authSvc := NewAuthService(mockRepo, mockTokenRepo, nil, jwtSvc, "markpost")

		_, _, err := authSvc.LoginWithEmail(ctx, "nonexistent@example.com", "password")
		if err == nil {
			t.Fatal("expected error for invalid email")
		}
	})

	t.Run("returns error for invalid password", func(t *testing.T) {
		mockRepo := newMockUserRepository()
		mockTokenRepo := newMockTokenRepository()
		ctx := context.Background()

		testUser, _ := mockRepo.Create(ctx, "test@example.com", "testuser", "correctpassword")
		mockRepo.SetRole(ctx, testUser.ID, user.RoleUser)

		jwtSvc := NewJWTService("test-access-secret", "test-refresh-secret", time.Hour, time.Hour*24)
		authSvc := NewAuthService(mockRepo, mockTokenRepo, nil, jwtSvc, "markpost")

		_, _, err := authSvc.LoginWithEmail(ctx, "test@example.com", "wrongpassword")
		if err == nil {
			t.Fatal("expected error for invalid password")
		}
	})
}

func TestAuthService_QueryPostKey(t *testing.T) {
	t.Run("returns post key for valid user", func(t *testing.T) {
		mockRepo := newMockUserRepository()
		mockTokenRepo := newMockTokenRepository()
		ctx := context.Background()

		testUser, _ := mockRepo.Create(ctx, "test@example.com", "testuser", "password")

		jwtSvc := NewJWTService("test-access-secret", "test-refresh-secret", time.Hour, time.Hour*24)
		authSvc := NewAuthService(mockRepo, mockTokenRepo, nil, jwtSvc, "markpost")

		postKey, _, err := authSvc.QueryPostKey(ctx, testUser.ID)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if postKey == "" {
			t.Error("expected post key, got empty")
		}
	})

	t.Run("returns error for non-existent user", func(t *testing.T) {
		mockRepo := newMockUserRepository()
		mockTokenRepo := newMockTokenRepository()
		ctx := context.Background()
		jwtSvc := NewJWTService("test-access-secret", "test-refresh-secret", time.Hour, time.Hour*24)
		authSvc := NewAuthService(mockRepo, mockTokenRepo, nil, jwtSvc, "markpost")

		_, _, err := authSvc.QueryPostKey(ctx, 999)
		if err == nil {
			t.Fatal("expected error for non-existent user")
		}
	})
}
