package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"markpost/models"
	"markpost/repositories"
	"markpost/utils"

	"errors"

	"golang.org/x/oauth2"
)

type AuthService struct {
	users repositories.UserRepoInterface
	oauth *oauth2.Config
	jwt   *JWTService
}

func NewAuthService(users repositories.UserRepoInterface, oauth *oauth2.Config, jwt *JWTService) *AuthService {
	return &AuthService{users: users, oauth: oauth, jwt: jwt}
}

func (s *AuthService) GenerateGitHubAuthURL(ctx context.Context) (string, error) {
	state, err := utils.GenerateState()
	if err != nil {
		return "", NewServiceErrorWrap(ErrInternal, "failed to generate state", err)
	}

	return s.oauth.AuthCodeURL(state), nil
}

func (s *AuthService) LoginWithGitHub(ctx context.Context, code string) (*models.User, *JWTTokenPair, error) {
	token, err := s.oauth.Exchange(ctx, code)
	if err != nil {
		return nil, nil, NewServiceErrorWrap(ErrUnauthorized, "oauth exchange failed", err)
	}

	githubUser, err := s.getGitHubUser(ctx, token)
	if err != nil {
		return nil, nil, err
	}

	user, err := s.users.GetOrCreateUserFromGitHub(githubUser)
	if err != nil {
		return nil, nil, NewServiceErrorWrap(ErrInternal, "create user failed", err)
	}

	pair, err := s.jwt.GenerateTokenPair(user.ID)
	if err != nil {
		return nil, nil, NewServiceErrorWrap(ErrInternal, "generate access/refresh token pair failed", err)
	}

	return user, pair, nil
}

func (s *AuthService) getGitHubUser(ctx context.Context, token *oauth2.Token) (*models.GitHubUser, error) {
	client := s.oauth.Client(ctx, token)
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, NewServiceErrorWrap(ErrInternal, "failed to get GitHub user", err)
	}
	defer resp.Body.Close()

	var githubUser models.GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&githubUser); err != nil {
		return nil, NewServiceErrorWrap(ErrInternal, "failed to decode GitHub user data", err)
	}

	if githubUser.ID == 0 || githubUser.Login == "" {
		return nil, NewServiceErrorWrap(ErrInternal, "invalid GitHub user data", fmt.Errorf("ID=%d, Login='%s'", githubUser.ID, githubUser.Login))
	}

	return &githubUser, nil
}

func (s *AuthService) LoginWithPassword(username, password string) (*models.User, *JWTTokenPair, error) {
	user, err := s.users.ValidateUserPassword(username, password)
	if err != nil {
		return nil, nil, NewServiceErrorWrap(ErrInvalidCredentials, "invalid username or password", err)
	}

	pair, err := s.jwt.GenerateTokenPair(user.ID)
	if err != nil {
		return nil, nil, NewServiceErrorWrap(ErrInternal, "generate access/refresh token pair failed", err)
	}

	return user, pair, nil
}

func (s *AuthService) RefreshToken(refreshToken string) (*models.User, *JWTTokenPair, error) {
	claims, err := s.jwt.ValidateRefresh(refreshToken)
	if err != nil {
		return nil, nil, NewServiceErrorWrap(ErrUnauthorized, "invalid refresh token", err)
	}

	userID, err := claims.UserID()
	if err != nil {
		return nil, nil, NewServiceErrorWrap(ErrUnauthorized, "invalid refresh token", err)
	}

	user, err := s.users.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return nil, nil, NewServiceErrorWrap(ErrUnauthorized, "user not found", err)
		}
		return nil, nil, NewServiceErrorWrap(ErrInternal, "query user failed", err)
	}

	pair, err := s.jwt.GenerateTokenPair(user.ID)
	if err != nil {
		return nil, nil, NewServiceErrorWrap(ErrInternal, "generate access/refresh token pair failed", err)
	}

	return user, pair, nil
}

func (s *AuthService) ChangePassword(userID int, current, new string) error {
	user, err := s.users.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return NewServiceErrorWrap(ErrNotFound, fmt.Sprintf("can not find user with ID %d", userID), err)
		}
		return NewServiceErrorWrap(ErrInternal, fmt.Sprintf("query user with ID %d failed", userID), err)
	}

	if user.Password != "" {
		ok, err := utils.CheckPassword(current, user.Password)
		if err != nil {
			return NewServiceErrorWrap(ErrInternal, fmt.Sprintf("validate current password with user ID %d failed", userID), err)
		}
		if !ok {
			return NewServiceErrorWrap(ErrInvalidPassword, "invalid current password", err)
		}
	}

	if err := s.users.SetUserPassword(userID, new); err != nil {
		return NewServiceErrorWrap(ErrInternal, fmt.Sprintf("set password with user ID %d failed", userID), err)
	}

	return nil
}

func (s *AuthService) QueryPostKey(userID int) (string, time.Time, error) {
	user, err := s.users.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return "", time.Time{}, NewServiceErrorWrap(ErrNotFound, fmt.Sprintf("can not find user with ID %d", userID), err)
		}
		return "", time.Time{}, NewServiceErrorWrap(ErrInternal, fmt.Sprintf("query user with ID %d failed", userID), err)
	}
	return user.PostKey, user.CreatedAt, nil
}
