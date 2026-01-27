package repositories

import (
	"errors"
	"fmt"

	"markpost/conf"
	"markpost/models"
	"markpost/utils"
)

type UserRepoInterface interface {
	GetUserByPostKey(postKey string) (*models.User, error)
	GetUserByID(id int) (*models.User, error)
	GetUserByGitHubID(githubID int64) (*models.User, error)
	GetUserByUsername(username string) (*models.User, error)
	CreateUser(username, password string) (*models.User, error)
	CreateUserFromGitHub(githubUser *models.GitHubUser) (*models.User, error)
	GetOrCreateUserFromGitHub(githubUser *models.GitHubUser) (*models.User, error)
	ValidateUserPassword(username, password string) (*models.User, error)
	SetUserPassword(userID int, password string) error
	SetUserRole(userID int, role models.Role) error
	GetAllUsers(offset, limit int) ([]models.User, error)
	CountUsers() (int64, error)
}

type UserRepo struct {
	database *models.Database
}

func NewUserRepo(database *models.Database) UserRepoInterface {
	return &UserRepo{database: database}
}

func (r *UserRepo) GetUserByPostKey(postKey string) (*models.User, error) {
	return models.GetUser(r.database, map[string]any{"post_key": postKey})
}

func (r *UserRepo) GetUserByID(id int) (*models.User, error) {
	return models.GetUser(r.database, map[string]any{"id": id})
}

func (r *UserRepo) GetUserByGitHubID(githubID int64) (*models.User, error) {
	return models.GetUser(r.database, map[string]any{"github_id": githubID})
}

func (r *UserRepo) GetUserByUsername(username string) (*models.User, error) {
	return models.GetUser(r.database, map[string]any{"username": username})
}

func (r *UserRepo) CreateUser(username, password string) (*models.User, error) {
	exists, err := models.UserExists(r.database, map[string]any{"username": username})
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("username is already taken")
	}

	return r.createUserWithUniquePostKey(username, password, nil)
}

func (r *UserRepo) CreateUserFromGitHub(githubUser *models.GitHubUser) (*models.User, error) {
	return r.createUserWithUniquePostKey(githubUser.Login, "", &githubUser.ID)
}

func (r *UserRepo) GetOrCreateUserFromGitHub(githubUser *models.GitHubUser) (*models.User, error) {
	user, err := r.GetUserByGitHubID(githubUser.ID)
	if err == nil {
		return user, nil
	}

	if !errors.Is(err, models.ErrNotFound) {
		return nil, err
	}

	return r.CreateUserFromGitHub(githubUser)
}

func (r *UserRepo) ValidateUserPassword(username, password string) (*models.User, error) {
	user, err := r.GetUserByUsername(username)
	if err != nil {
		return nil, err
	}

	if user.Password == "" {
		return nil, fmt.Errorf("user has no password set")
	}

	ok, err := utils.CheckPassword(password, user.Password)
	if err != nil {
		return nil, fmt.Errorf("validate user %s password: %w", username, err)
	}
	if !ok {
		return nil, fmt.Errorf("invalid password")
	}

	return user, nil
}

func (r *UserRepo) SetUserPassword(userID int, password string) error {
	user, err := r.GetUserByID(userID)
	if err != nil {
		return err
	}

	hashed, err := utils.HashPassword(password)
	if err != nil {
		return err
	}

	user.Password = hashed
	return user.Update(r.database)
}

func (r *UserRepo) GetAllUsers(offset, limit int) ([]models.User, error) {
	return models.GetUsers(r.database, offset, limit)
}

func (r *UserRepo) CountUsers() (int64, error) {
	return models.CountUsers(r.database)
}

func (r *UserRepo) SetUserRole(userID int, role models.Role) error {
	user, err := r.GetUserByID(userID)
	if err != nil {
		return err
	}

	user.Role = role
	return user.Update(r.database)
}

func (r *UserRepo) createUserWithUniquePostKey(username, password string, githubID *int64) (*models.User, error) {
	for {
		user, err := makeUser(username, password, githubID)
		if err != nil {
			return nil, err
		}

		exists, err := models.UserExists(r.database, map[string]any{"post_key": user.PostKey})
		if err != nil {
			return nil, err
		}
		if exists {
			continue
		}

		if err = user.Create(r.database); err == nil {
			return user, nil
		}

		return nil, err
	}
}

func makeUser(username, password string, githubID *int64) (*models.User, error) {
	postKey, err := utils.GeneratePostKey(conf.Conf().PostKeyLength)
	if err != nil {
		return nil, err
	}

	var hash string
	if password != "" {
		hash, err = utils.HashPassword(password)
		if err != nil {
			return nil, err
		}
	}

	user := models.User{
		Username: username,
		Password: hash,
		PostKey:  postKey,
		GitHubID: githubID,
	}

	return &user, nil
}
