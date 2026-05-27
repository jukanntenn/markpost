package infra

import (
	"context"
	"errors"
	"testing"
	"time"

	"markpost/internal/domain"
	"markpost/internal/domain/user"
)

func TestUserRepository_Create(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewUserRepository(db, 16)
	ctx := context.Background()

	t.Run("creates user successfully", func(t *testing.T) {
		u, err := repo.Create(ctx, "alice@example.com", "alice", "password123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.Email != "alice@example.com" {
			t.Errorf("email = %q, want %q", u.Email, "alice@example.com")
		}
		if u.Username != "alice" {
			t.Errorf("username = %q, want %q", u.Username, "alice")
		}
		if u.PostKey == "" {
			t.Error("expected non-empty post key")
		}
		if !u.IsActive {
			t.Error("expected user to be active")
		}
		if u.Password == "" {
			t.Error("expected hashed password")
		}
	})

	t.Run("rejects duplicate email", func(t *testing.T) {
		_, _ = repo.Create(ctx, "dup@example.com", "user1", "pass")
		_, err := repo.Create(ctx, "dup@example.com", "user2", "pass")
		if err == nil {
			t.Fatal("expected error for duplicate email")
		}
		if !errors.Is(err, domain.ErrEmailTaken) {
			t.Errorf("expected ErrEmailTaken, got: %v", err)
		}
	})

	t.Run("rejects duplicate username", func(t *testing.T) {
		_, _ = repo.Create(ctx, "unique@example.com", "dupuser", "pass")
		_, err := repo.Create(ctx, "other@example.com", "dupuser", "pass")
		if err == nil {
			t.Fatal("expected error for duplicate username")
		}
		if !errors.Is(err, domain.ErrUsernameTaken) {
			t.Errorf("expected ErrUsernameTaken, got: %v", err)
		}
	})
}

func TestUserRepository_GetByID(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewUserRepository(db, 16)
	ctx := context.Background()

	created, _ := repo.Create(ctx, "test@example.com", "testuser", "pass")

	t.Run("finds existing user", func(t *testing.T) {
		u, err := repo.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.Username != "testuser" {
			t.Errorf("username = %q, want %q", u.Username, "testuser")
		}
	})

	t.Run("returns ErrNotFound for missing user", func(t *testing.T) {
		_, err := repo.GetByID(ctx, 9999)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})
}

func TestUserRepository_GetByUsername(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewUserRepository(db, 16)
	ctx := context.Background()

	_, _ = repo.Create(ctx, "test@example.com", "testuser", "pass")

	t.Run("finds existing user", func(t *testing.T) {
		u, err := repo.GetByUsername(ctx, "testuser")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.Email != "test@example.com" {
			t.Errorf("email = %q, want %q", u.Email, "test@example.com")
		}
	})

	t.Run("returns ErrNotFound for missing user", func(t *testing.T) {
		_, err := repo.GetByUsername(ctx, "nonexistent")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})
}

func TestUserRepository_GetByEmail(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewUserRepository(db, 16)
	ctx := context.Background()

	_, _ = repo.Create(ctx, "test@example.com", "testuser", "pass")

	u, err := repo.GetByEmail(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Username != "testuser" {
		t.Errorf("username = %q, want %q", u.Username, "testuser")
	}
}

func TestUserRepository_GetByPostKey(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewUserRepository(db, 16)
	ctx := context.Background()

	created, _ := repo.Create(ctx, "test@example.com", "testuser", "pass")

	u, err := repo.GetByPostKey(ctx, created.PostKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != created.ID {
		t.Errorf("id = %d, want %d", u.ID, created.ID)
	}
}

func TestUserRepository_ValidatePassword(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewUserRepository(db, 16)
	ctx := context.Background()

	_, _ = repo.Create(ctx, "test@example.com", "testuser", "correctpass")

	t.Run("valid password", func(t *testing.T) {
		u, err := repo.ValidatePassword(ctx, "testuser", "correctpass")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.Username != "testuser" {
			t.Errorf("username = %q, want %q", u.Username, "testuser")
		}
	})

	t.Run("invalid password", func(t *testing.T) {
		_, err := repo.ValidatePassword(ctx, "testuser", "wrongpass")
		if err == nil {
			t.Fatal("expected error for wrong password")
		}
		if !errors.Is(err, domain.ErrBadPassword) {
			t.Errorf("expected ErrBadPassword, got: %v", err)
		}
	})

	t.Run("nonexistent user", func(t *testing.T) {
		_, err := repo.ValidatePassword(ctx, "nonexistent", "pass")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})
}

func TestUserRepository_SetPassword(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewUserRepository(db, 16)
	ctx := context.Background()

	created, _ := repo.Create(ctx, "test@example.com", "testuser", "oldpass")

	if err := repo.SetPassword(ctx, created.ID, "newpass"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	u, err := repo.ValidatePassword(ctx, "testuser", "newpass")
	if err != nil {
		t.Fatalf("expected new password to work, got: %v", err)
	}
	if u.ID != created.ID {
		t.Errorf("id = %d, want %d", u.ID, created.ID)
	}
}

func TestUserRepository_SetRole(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewUserRepository(db, 16)
	ctx := context.Background()

	created, _ := repo.Create(ctx, "test@example.com", "testuser", "pass")

	if err := repo.SetRole(ctx, created.ID, user.RoleAdmin); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	u, _ := repo.GetByID(ctx, created.ID)
	if u.Role != user.RoleAdmin {
		t.Errorf("role = %q, want %q", u.Role, user.RoleAdmin)
	}
}

func TestUserRepository_DeleteByID(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewUserRepository(db, 16)
	ctx := context.Background()

	created, _ := repo.Create(ctx, "test@example.com", "testuser", "pass")

	affected, err := repo.DeleteByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if affected != 1 {
		t.Errorf("affected = %d, want 1", affected)
	}

	_, err = repo.GetByID(ctx, created.ID)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got: %v", err)
	}
}

func TestUserRepository_GetAll(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewUserRepository(db, 16)
	ctx := context.Background()

	_, _ = repo.Create(ctx, "a@example.com", "alice", "pass")
	_, _ = repo.Create(ctx, "b@example.com", "bob", "pass")
	_, _ = repo.Create(ctx, "c@example.com", "charlie", "pass")

	t.Run("returns all users", func(t *testing.T) {
		users, err := repo.GetAll(ctx, 0, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(users) != 3 {
			t.Errorf("got %d users, want 3", len(users))
		}
	})

	t.Run("respects pagination", func(t *testing.T) {
		users, err := repo.GetAll(ctx, 1, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(users) != 1 {
			t.Errorf("got %d users, want 1", len(users))
		}
	})
}

func TestUserRepository_Count(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewUserRepository(db, 16)
	ctx := context.Background()

	_, _ = repo.Create(ctx, "a@example.com", "alice", "pass")
	_, _ = repo.Create(ctx, "b@example.com", "bob", "pass")

	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestUserRepository_CreateFromGitHub(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewUserRepository(db, 16)
	ctx := context.Background()

	ghUser := &user.GitHubUser{
		ID:        12345,
		Login:     "ghuser",
		Email:     "gh@example.com",
		AvatarURL: "https://avatar.url",
	}

	u, err := repo.CreateFromGitHub(ctx, ghUser)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Username != "ghuser" {
		t.Errorf("username = %q, want %q", u.Username, "ghuser")
	}
	if u.GitHubID == nil || *u.GitHubID != 12345 {
		t.Errorf("github_id = %v, want 12345", u.GitHubID)
	}
	if u.Password != "" {
		t.Error("expected empty password for github user")
	}
}

func TestUserRepository_GetOrCreateFromGitHub(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewUserRepository(db, 16)
	ctx := context.Background()

	ghUser := &user.GitHubUser{ID: 12345, Login: "ghuser", Email: "gh@example.com"}

	t.Run("creates new user", func(t *testing.T) {
		u, err := repo.GetOrCreateFromGitHub(ctx, ghUser)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.Username != "ghuser" {
			t.Errorf("username = %q, want %q", u.Username, "ghuser")
		}
	})

	t.Run("returns existing user", func(t *testing.T) {
		u, err := repo.GetOrCreateFromGitHub(ctx, ghUser)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.Username != "ghuser" {
			t.Errorf("username = %q, want %q", u.Username, "ghuser")
		}
	})
}

func TestUserRepository_UpdateLastLoginAt(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewUserRepository(db, 16)
	ctx := context.Background()

	created, _ := repo.Create(ctx, "test@example.com", "testuser", "pass")

	now := timeNow()
	if err := repo.UpdateLastLoginAt(ctx, created.ID, now); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	u, _ := repo.GetByID(ctx, created.ID)
	if u.LastLoginAt == nil {
		t.Fatal("expected LastLoginAt to be set")
	}
}

func timeNow() time.Time {
	return time.Now().Truncate(time.Second)
}
