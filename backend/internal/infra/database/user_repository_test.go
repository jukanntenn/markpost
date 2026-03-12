package database

import (
	"testing"

	"markpost/internal/domain/user"
)

func setupUserTestDatabase(t *testing.T) *Database {
	t.Helper()

	database, err := NewTestDatabase()
	if err != nil {
		t.Fatalf("NewTestDatabase error: %v", err)
	}

	return database
}

func TestUserRepository_GetByPostKey(t *testing.T) {
	t.Run("returns expected record for valid postKey", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepository(database.DB())

		u, err := repo.Create(nil, "alice", "password")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		got, err := repo.GetByPostKey(nil, u.PostKey)
		if err != nil {
			t.Fatalf("GetByPostKey error: %v", err)
		}

		if got == nil || got.Username != "alice" || got.PostKey != u.PostKey {
			t.Fatalf("unexpected user: %+v", got)
		}
	})

	t.Run("returns ErrNotFound for wrong postKey", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepository(database.DB())

		_, err := repo.Create(nil, "alice", "password")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		_, err = repo.GetByPostKey(nil, "not-exist")
		if err == nil || err != user.ErrNotFound {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestUserRepository_GetByID(t *testing.T) {
	t.Run("returns expected record for valid ID", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepository(database.DB())

		u, err := repo.Create(nil, "alice", "password")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		got, err := repo.GetByID(nil, u.ID)
		if err != nil {
			t.Fatalf("GetByID error: %v", err)
		}

		if got == nil || got.ID != u.ID || got.Username != "alice" {
			t.Fatalf("unexpected user: %+v", got)
		}
	})

	t.Run("returns ErrNotFound for wrong ID", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepository(database.DB())

		_, err := repo.Create(nil, "alice", "password")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		_, err = repo.GetByID(nil, 123456)
		if err == nil || err != user.ErrNotFound {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestUserRepository_Create(t *testing.T) {
	t.Run("creates user with valid data", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepository(database.DB())

		u, err := repo.Create(nil, "alice", "password")
		if err != nil {
			t.Fatalf("Create error: %v", err)
		}

		if u == nil || u.Username != "alice" || u.Password == "" || u.PostKey == "" {
			t.Fatalf("unexpected user: %+v", u)
		}
	})

	t.Run("returns error for duplicate username", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepository(database.DB())

		_, err := repo.Create(nil, "alice", "password")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		_, err = repo.Create(nil, "alice", "password2")
		if err == nil || err.Error() != "username is already taken" {
			t.Fatalf("expected 'username is already taken' error, got %v", err)
		}
	})
}

func TestUserRepository_ValidatePassword(t *testing.T) {
	t.Run("validates correct password", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepository(database.DB())

		password := "password123"
		_, err := repo.Create(nil, "alice", password)
		if err != nil {
			t.Fatalf("Create error: %v", err)
		}

		u, err := repo.ValidatePassword(nil, "alice", password)
		if err != nil {
			t.Fatalf("ValidatePassword error: %v", err)
		}

		if u == nil || u.Username != "alice" {
			t.Fatalf("unexpected user: %+v", u)
		}
	})

	t.Run("returns error for wrong password", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepository(database.DB())

		_, err := repo.Create(nil, "alice", "password123")
		if err != nil {
			t.Fatalf("Create error: %v", err)
		}

		_, err = repo.ValidatePassword(nil, "alice", "wrongpassword")
		if err == nil || err.Error() != "invalid password" {
			t.Fatalf("expected 'invalid password' error, got %v", err)
		}
	})
}
