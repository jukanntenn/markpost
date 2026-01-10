package repositories

import (
	"testing"

	"markpost/models"
)

func setupUserTestDatabase(t *testing.T) *models.Database {
	t.Helper()

	database, err := models.NewTestDatabase()
	if err != nil {
		t.Fatalf("NewTestDatabase error: %v", err)
	}

	return database
}
func TestUserRepository_GetUserByPostKey(t *testing.T) {
	t.Run("returns expected record for valid postKey", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepo(database)

		u, err := repo.CreateUser("alice", "password")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		got, err := repo.GetUserByPostKey(u.PostKey)
		if err != nil {
			t.Fatalf("GetUserByPostKey error: %v", err)
		}

		if got == nil || got.Username != "alice" || got.PostKey != u.PostKey {
			t.Fatalf("unexpected user: %+v", got)
		}
	})

	t.Run("returns models.ErrNotFound for wrong postKey", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepo(database)

		_, err := repo.CreateUser("alice", "password")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		_, err = repo.GetUserByPostKey("not-exist")
		if err == nil || err != models.ErrNotFound {
			t.Fatalf("expected models.ErrNotFound, got %v", err)
		}
	})
}

func TestUserRepository_GetUserByID(t *testing.T) {
	t.Run("returns expected record for valid ID", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepo(database)

		u, err := repo.CreateUser("alice", "password")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		got, err := repo.GetUserByID(u.ID)
		if err != nil {
			t.Fatalf("GetUserByID error: %v", err)
		}

		if got == nil || got.ID != u.ID || got.Username != "alice" {
			t.Fatalf("unexpected user: %+v", got)
		}
	})

	t.Run("returns models.ErrNotFound for wrong ID", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepo(database)

		_, err := repo.CreateUser("alice", "password")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		_, err = repo.GetUserByID(123456)
		if err == nil || err != models.ErrNotFound {
			t.Fatalf("expected models.ErrNotFound, got %v", err)
		}
	})
}

func TestUserRepository_GetUserByGitHubID(t *testing.T) {
	t.Run("returns expected record for valid GitHubID", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepo(database)

		githubID := int64(12345)
		_, err := repo.CreateUserFromGitHub(&models.GitHubUser{ID: githubID, Login: "alice"})
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		got, err := repo.GetUserByGitHubID(githubID)
		if err != nil {
			t.Fatalf("GetUserByGitHubID error: %v", err)
		}

		if got == nil || got.ID == 0 || got.Username != "alice" || got.GitHubID == nil || *got.GitHubID != githubID {
			t.Fatalf("unexpected user: %+v", got)
		}
	})

	t.Run("returns models.ErrNotFound for wrong GitHubID", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepo(database)

		_, err := repo.CreateUserFromGitHub(&models.GitHubUser{ID: 12345, Login: "alice"})
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		_, err = repo.GetUserByGitHubID(99999)
		if err == nil || err != models.ErrNotFound {
			t.Fatalf("expected models.ErrNotFound, got %v", err)
		}
	})
}

func TestUserRepository_GetUserByUsername(t *testing.T) {
	t.Run("returns expected record for valid username", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepo(database)

		_, err := repo.CreateUser("alice", "password")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		got, err := repo.GetUserByUsername("alice")
		if err != nil {
			t.Fatalf("GetUserByUsername error: %v", err)
		}

		if got == nil || got.Username != "alice" {
			t.Fatalf("unexpected user: %+v", got)
		}
	})

	t.Run("returns models.ErrNotFound for wrong username", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepo(database)

		_, err := repo.CreateUser("alice", "password")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		_, err = repo.GetUserByUsername("not-exist")
		if err == nil || err != models.ErrNotFound {
			t.Fatalf("expected models.ErrNotFound, got %v", err)
		}
	})
}

func TestUserRepository_CreateUser(t *testing.T) {
	t.Run("creates user with valid data", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepo(database)

		user, err := repo.CreateUser("alice", "password")
		if err != nil {
			t.Fatalf("CreateUser error: %v", err)
		}

		if user == nil || user.Username != "alice" || user.Password == "" || user.PostKey == "" {
			t.Fatalf("unexpected user: %+v", user)
		}
	})

	t.Run("returns error for duplicate username", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepo(database)

		_, err := repo.CreateUser("alice", "password")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		_, err = repo.CreateUser("alice", "password2")
		if err == nil || err.Error() != "username is already taken" {
			t.Fatalf("expected 'username is already taken' error, got %v", err)
		}
	})

	t.Run("creates user with unique postKey", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepo(database)

		user1, err := repo.CreateUser("alice", "password")
		if err != nil {
			t.Fatalf("CreateUser error: %v", err)
		}

		user2, err := repo.CreateUser("bob", "password")
		if err != nil {
			t.Fatalf("CreateUser error: %v", err)
		}

		if user1.PostKey == user2.PostKey {
			t.Fatalf("postKeys should be unique: %s", user1.PostKey)
		}
	})
}

func TestUserRepository_CreateUserFromGitHub(t *testing.T) {
	t.Run("creates user with valid GitHub data", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepo(database)

		githubID := int64(12345)
		githubUser := &models.GitHubUser{ID: githubID, Login: "alice"}

		user, err := repo.CreateUserFromGitHub(githubUser)
		if err != nil {
			t.Fatalf("CreateUserFromGitHub error: %v", err)
		}

		if user == nil || user.Username != "alice" || user.GitHubID == nil || *user.GitHubID != githubID {
			t.Fatalf("unexpected user: %+v", user)
		}

		if user.Password != "" {
			t.Fatalf("GitHub user should not have password: %+v", user)
		}
	})

	t.Run("creates user with unique postKey", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepo(database)

		githubUser1 := &models.GitHubUser{ID: 12345, Login: "alice"}
		githubUser2 := &models.GitHubUser{ID: 67890, Login: "bob"}

		user1, err := repo.CreateUserFromGitHub(githubUser1)
		if err != nil {
			t.Fatalf("CreateUserFromGitHub error: %v", err)
		}

		user2, err := repo.CreateUserFromGitHub(githubUser2)
		if err != nil {
			t.Fatalf("CreateUserFromGitHub error: %v", err)
		}

		if user1.PostKey == user2.PostKey {
			t.Fatalf("postKeys should be unique: %s", user1.PostKey)
		}
	})
}

func TestUserRepository_GetOrCreateUserFromGitHub(t *testing.T) {
	t.Run("returns existing user for known GitHub ID", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepo(database)

		githubID := int64(12345)
		githubUser := &models.GitHubUser{ID: githubID, Login: "alice"}

		originalUser, err := repo.CreateUserFromGitHub(githubUser)
		if err != nil {
			t.Fatalf("CreateUserFromGitHub error: %v", err)
		}

		retrievedUser, err := repo.GetOrCreateUserFromGitHub(githubUser)
		if err != nil {
			t.Fatalf("GetOrCreateUserFromGitHub error: %v", err)
		}

		if retrievedUser.ID != originalUser.ID {
			t.Fatalf("expected same user ID %d, got %d", originalUser.ID, retrievedUser.ID)
		}
	})

	t.Run("creates new user for unknown GitHub ID", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepo(database)

		githubID := int64(54321)
		githubUser := &models.GitHubUser{ID: githubID, Login: "charlie"}

		user, err := repo.GetOrCreateUserFromGitHub(githubUser)
		if err != nil {
			t.Fatalf("GetOrCreateUserFromGitHub error: %v", err)
		}

		if user == nil || user.Username != "charlie" || user.GitHubID == nil || *user.GitHubID != githubID {
			t.Fatalf("unexpected user: %+v", user)
		}
	})
}

func TestUserRepository_ValidateUserPassword(t *testing.T) {
	t.Run("validates correct password", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepo(database)

		password := "password123"
		_, err := repo.CreateUser("alice", password)
		if err != nil {
			t.Fatalf("CreateUser error: %v", err)
		}

		user, err := repo.ValidateUserPassword("alice", password)
		if err != nil {
			t.Fatalf("ValidateUserPassword error: %v", err)
		}

		if user == nil || user.Username != "alice" {
			t.Fatalf("unexpected user: %+v", user)
		}
	})

	t.Run("returns error for wrong password", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepo(database)

		_, err := repo.CreateUser("alice", "password123")
		if err != nil {
			t.Fatalf("CreateUser error: %v", err)
		}

		_, err = repo.ValidateUserPassword("alice", "wrongpassword")
		if err == nil || err.Error() != "invalid password" {
			t.Fatalf("expected 'invalid password' error, got %v", err)
		}
	})

	t.Run("returns error for non-existent user", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepo(database)

		_, err := repo.ValidateUserPassword("nonexistent", "password")
		if err == nil || err != models.ErrNotFound {
			t.Fatalf("expected models.ErrNotFound, got %v", err)
		}
	})

	t.Run("returns error for user without password", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepo(database)

		githubUser := &models.GitHubUser{ID: 12345, Login: "alice"}
		_, err := repo.CreateUserFromGitHub(githubUser)
		if err != nil {
			t.Fatalf("CreateUserFromGitHub error: %v", err)
		}

		_, err = repo.ValidateUserPassword("alice", "password")
		if err == nil || err.Error() != "user has no password set" {
			t.Fatalf("expected 'user has no password set' error, got %v", err)
		}
	})
}

func TestUserRepository_SetUserPassword(t *testing.T) {
	t.Run("sets password for existing user", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepo(database)

		user, err := repo.CreateUser("alice", "oldpassword")
		if err != nil {
			t.Fatalf("CreateUser error: %v", err)
		}

		newPassword := "newpassword123"
		err = repo.SetUserPassword(user.ID, newPassword)
		if err != nil {
			t.Fatalf("SetUserPassword error: %v", err)
		}

		validateUser, err := repo.ValidateUserPassword("alice", newPassword)
		if err != nil {
			t.Fatalf("ValidateUserPassword error: %v", err)
		}

		if validateUser.ID != user.ID {
			t.Fatalf("expected user ID %d, got %d", user.ID, validateUser.ID)
		}
	})

	t.Run("sets password for GitHub user without password", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepo(database)

		githubUser := &models.GitHubUser{ID: 12345, Login: "alice"}
		user, err := repo.CreateUserFromGitHub(githubUser)
		if err != nil {
			t.Fatalf("CreateUserFromGitHub error: %v", err)
		}

		newPassword := "newpassword123"
		err = repo.SetUserPassword(user.ID, newPassword)
		if err != nil {
			t.Fatalf("SetUserPassword error: %v", err)
		}

		validateUser, err := repo.ValidateUserPassword("alice", newPassword)
		if err != nil {
			t.Fatalf("ValidateUserPassword error: %v", err)
		}

		if validateUser.ID != user.ID {
			t.Fatalf("expected user ID %d, got %d", user.ID, validateUser.ID)
		}
	})

	t.Run("returns error for non-existent user ID", func(t *testing.T) {
		database := setupUserTestDatabase(t)
		repo := NewUserRepo(database)

		err := repo.SetUserPassword(999999, "password")
		if err == nil {
			t.Fatalf("expected error for non-existent user ID")
		}
	})
}
