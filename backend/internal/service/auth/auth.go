package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"markpost/internal/domain/user"
	"markpost/pkg/utils"

	"golang.org/x/oauth2"
)

type AuthService struct {
	users user.Repository
	oauth *oauth2.Config
	jwt   *JWTService
}

func NewAuthService(users user.Repository, oauth *oauth2.Config, jwt *JWTService) *AuthService {
	return &AuthService{users: users, oauth: oauth, jwt: jwt}
}

func (s *AuthService) GenerateGitHubAuthURL(ctx context.Context) (string, error) {
	state, err := utils.GenerateState()
	if err != nil {
		return "", NewServiceErrorWrap(ErrInternal, "failed to generate state", err)
	}

	return s.oauth.AuthCodeURL(state), nil
}

func (s *AuthService) LoginWithGitHub(ctx context.Context, code string) (*user.User, *JWTTokenPair, error) {
	token, err := s.oauth.Exchange(ctx, code)
	if err != nil {
		return nil, nil, NewServiceErrorWrap(ErrUnauthorized, "oauth exchange failed", err)
	}

	githubUser, err := s.getGitHubUser(ctx, token)
	if err != nil {
		return nil, nil, err
	}

	u, err := s.users.GetOrCreateFromGitHub(ctx, githubUser)
	if err != nil {
		return nil, nil, NewServiceErrorWrap(ErrInternal, "create user failed", err)
	}

	pair, err := s.jwt.GenerateTokenPair(u.ID, string(u.Role))
	if err != nil {
		return nil, nil, NewServiceErrorWrap(ErrInternal, "generate access/refresh token pair failed", err)
	}

	return u, pair, nil
}

func (s *AuthService) getGitHubUser(ctx context.Context, token *oauth2.Token) (*user.GitHubUser, error) {
	client := s.oauth.Client(ctx, token)
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, NewServiceErrorWrap(ErrInternal, "failed to get GitHub user", err)
	}
	defer resp.Body.Close()

	var githubUser user.GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&githubUser); err != nil {
		return nil, NewServiceErrorWrap(ErrInternal, "failed to decode GitHub user data", err)
	}

	if githubUser.ID == 0 || githubUser.Login == "" {
		return nil, NewServiceErrorWrap(ErrInternal, "invalid GitHub user data", fmt.Errorf("ID=%d, Login='%s'", githubUser.ID, githubUser.Login))
	}

	return &githubUser, nil
}

func (s *AuthService) LoginWithPassword(ctx context.Context, username, password string) (*user.User, *JWTTokenPair, error) {
	u, err := s.users.ValidatePassword(ctx, username, password)
	if err != nil {
		return nil, nil, NewServiceErrorWrap(ErrInvalidCredentials, "invalid username or password", err)
	}

	pair, err := s.jwt.GenerateTokenPair(u.ID, string(u.Role))
	if err != nil {
		return nil, nil, NewServiceErrorWrap(ErrInternal, "generate access/refresh token pair failed", err)
	}

	return u, pair, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*user.User, *JWTTokenPair, error) {
	claims, err := s.jwt.ValidateRefresh(refreshToken)
	if err != nil {
		return nil, nil, NewServiceErrorWrap(ErrUnauthorized, "invalid refresh token", err)
	}

	u, err := s.users.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, nil, NewServiceErrorWrap(ErrUnauthorized, "user not found", err)
	}

	pair, err := s.jwt.GenerateTokenPair(u.ID, string(u.Role))
	if err != nil {
		return nil, nil, NewServiceErrorWrap(ErrInternal, "generate access/refresh token pair failed", err)
	}

	return u, pair, nil
}

func (s *AuthService) ChangePassword(ctx context.Context, userID int, current, new string) error {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return NewServiceErrorWrap(ErrNotFound, fmt.Sprintf("can not find user with ID %d", userID), err)
	}

	if u.Password != "" {
		ok, err := utils.CheckPassword(current, u.Password)
		if err != nil {
			return NewServiceErrorWrap(ErrInternal, fmt.Sprintf("validate current password with user ID %d failed", userID), err)
		}
		if !ok {
			return NewServiceErrorWrap(ErrInvalidPassword, "invalid current password", err)
		}
	}

	if err := s.users.SetPassword(ctx, userID, new); err != nil {
		return NewServiceErrorWrap(ErrInternal, fmt.Sprintf("set password with user ID %d failed", userID), err)
	}

	return nil
}

func (s *AuthService) QueryPostKey(ctx context.Context, userID int) (string, time.Time, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return "", time.Time{}, NewServiceErrorWrap(ErrNotFound, fmt.Sprintf("can not find user with ID %d", userID), err)
	}
	return u.PostKey, u.CreatedAt, nil
}

func (s *AuthService) GetAllUsers(ctx context.Context, page, limit int) ([]user.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	users, err := s.users.GetAll(ctx, offset, limit)
	if err != nil {
		return nil, 0, NewServiceErrorWrap(ErrInternal, "failed to get users", err)
	}

	total, err := s.users.Count(ctx)
	if err != nil {
		return nil, 0, NewServiceErrorWrap(ErrInternal, "failed to count users", err)
	}

	return users, total, nil
}

func (s *AuthService) InitializeFirstAdmin(ctx context.Context, initialUsername string) error {
	u, err := s.users.GetByUsername(ctx, initialUsername)
	if err != nil {
		return NewServiceError(ErrNotFound, fmt.Sprintf("initial admin user '%s' not found", initialUsername))
	}

	if u.Role == user.RoleAdmin {
		return nil
	}

	if err := s.users.SetRole(ctx, u.ID, user.RoleAdmin); err != nil {
		return NewServiceErrorWrap(ErrInternal, fmt.Sprintf("failed to promote user '%s' to admin", initialUsername), err)
	}

	return nil
}
