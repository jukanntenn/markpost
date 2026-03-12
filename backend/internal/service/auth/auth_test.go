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

func (m *mockUserRepository) Create(ctx context.Context, username, password string) (*user.User, error) {
	for _, u := range m.users {
		if u.Username == username {
			return nil, nil
		}
	}
	u := &user.User{
		ID:       m.nextID,
		Username: username,
		Password: password,
		PostKey:  "test-post-key",
	}
	m.users[m.nextID] = u
	m.postKeyMap[u.PostKey] = u
	m.nextID++
	return u, nil
}

func (m *mockUserRepository) ValidatePassword(ctx context.Context, username, password string) (*user.User, error) {
	u, err := m.GetByUsername(ctx, username)
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
		Username: githubUser.Login,
		PostKey:  "test-post-key",
		GitHubID: &githubUser.ID,
	}
	m.users[m.nextID] = u
	m.postKeyMap[u.PostKey] = u
	m.nextID++
	return u, nil
}

func TestAuthService_LoginWithPassword(t *testing.T) {
	t.Run("returns tokens for valid credentials", func(t *testing.T) {
		mockRepo := newMockUserRepository()
		ctx := context.Background()

		testUser, _ := mockRepo.Create(ctx, "testuser", "correctpassword")
		mockRepo.SetRole(ctx, testUser.ID, user.RoleUser)

		jwtSvc := NewJWTService("test-access-secret", "test-refresh-secret", time.Hour, time.Hour*24)
		authSvc := NewAuthService(mockRepo, nil, jwtSvc)

		u, tokens, err := authSvc.LoginWithPassword(ctx, "testuser", "correctpassword")
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

	t.Run("returns error for invalid username", func(t *testing.T) {
		mockRepo := newMockUserRepository()
		ctx := context.Background()

		jwtSvc := NewJWTService("test-access-secret", "test-refresh-secret", time.Hour, time.Hour*24)
		authSvc := NewAuthService(mockRepo, nil, jwtSvc)

		_, _, err := authSvc.LoginWithPassword(ctx, "nonexistent", "password")
		if err == nil {
			t.Fatal("expected error for invalid username")
		}
	})

	t.Run("returns error for invalid password", func(t *testing.T) {
		mockRepo := newMockUserRepository()
		ctx := context.Background()

		testUser, _ := mockRepo.Create(ctx, "testuser", "correctpassword")
		mockRepo.SetRole(ctx, testUser.ID, user.RoleUser)

		jwtSvc := NewJWTService("test-access-secret", "test-refresh-secret", time.Hour, time.Hour*24)
		authSvc := NewAuthService(mockRepo, nil, jwtSvc)

		_, _, err := authSvc.LoginWithPassword(ctx, "testuser", "wrongpassword")
		if err == nil {
			t.Fatal("expected error for invalid password")
		}
	})
}

func TestAuthService_QueryPostKey(t *testing.T) {
	t.Run("returns post key for valid user", func(t *testing.T) {
		mockRepo := newMockUserRepository()
		ctx := context.Background()

		testUser, _ := mockRepo.Create(ctx, "testuser", "password")

		jwtSvc := NewJWTService("test-access-secret", "test-refresh-secret", time.Hour, time.Hour*24)
		authSvc := NewAuthService(mockRepo, nil, jwtSvc)

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
		ctx := context.Background()
		jwtSvc := NewJWTService("test-access-secret", "test-refresh-secret", time.Hour, time.Hour*24)
		authSvc := NewAuthService(mockRepo, nil, jwtSvc)

		_, _, err := authSvc.QueryPostKey(ctx, 99999)
		if err == nil {
			t.Fatal("expected error for non-existent user")
		}
	})
}
