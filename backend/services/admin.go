package services

import (
	"errors"
	"fmt"

	"markpost/models"
	"markpost/repositories"
)

type AdminService struct {
	users    repositories.UserRepoInterface
	posts    repositories.PostRepoInterface
	channels repositories.DeliveryChannelRepoInterface
}

func NewAdminService(users repositories.UserRepoInterface, posts repositories.PostRepoInterface, channels repositories.DeliveryChannelRepoInterface) *AdminService {
	return &AdminService{users: users, posts: posts, channels: channels}
}

func (s *AdminService) UpdateUserRole(id int, role models.Role) (*models.User, error) {
	_, err := s.users.GetUserByID(id)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return nil, NewServiceErrorWrap(ErrNotFound, fmt.Sprintf("user with ID %d not found", id), err)
		}
		return nil, NewServiceErrorWrap(ErrInternal, fmt.Sprintf("get user with ID %d failed", id), err)
	}

	if err := s.users.SetUserRole(id, role); err != nil {
		return nil, NewServiceErrorWrap(ErrInternal, fmt.Sprintf("set role for user with ID %d failed", id), err)
	}

	user, err := s.users.GetUserByID(id)
	if err != nil {
		return nil, NewServiceErrorWrap(ErrInternal, fmt.Sprintf("re-query user with ID %d failed", id), err)
	}

	return user, nil
}

func (s *AdminService) DeleteUser(id int) error {
	rows, err := s.users.DeleteUserByID(id)
	if err != nil {
		return NewServiceErrorWrap(ErrInternal, fmt.Sprintf("delete user with ID %d failed", id), err)
	}
	if rows == 0 {
		return NewServiceError(ErrNotFound, fmt.Sprintf("user with ID %d not found", id))
	}
	return nil
}

func (s *AdminService) ListAllPosts(search string, offset int, limit int) ([]models.Post, int64, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	posts, err := s.posts.ListAllPosts(search, offset, limit)
	if err != nil {
		return nil, 0, NewServiceErrorWrap(ErrInternal, "list posts failed", err)
	}

	total, err := s.posts.CountAllPosts(search)
	if err != nil {
		return nil, 0, NewServiceErrorWrap(ErrInternal, "count posts failed", err)
	}

	return posts, total, nil
}

func (s *AdminService) UpdatePost(id int, title string, body string) (*models.Post, error) {
	post, err := s.posts.UpdatePostByID(id, title, body)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return nil, NewServiceErrorWrap(ErrNotFound, fmt.Sprintf("post with ID %d not found", id), err)
		}
		return nil, NewServiceErrorWrap(ErrInternal, fmt.Sprintf("update post with ID %d failed", id), err)
	}
	return post, nil
}

func (s *AdminService) DeletePost(id int) error {
	rows, err := s.posts.DeletePostByID(id)
	if err != nil {
		return NewServiceErrorWrap(ErrInternal, fmt.Sprintf("delete post with ID %d failed", id), err)
	}
	if rows == 0 {
		return NewServiceError(ErrNotFound, fmt.Sprintf("post with ID %d not found", id))
	}
	return nil
}

func (s *AdminService) ListAllDeliveryChannels() ([]models.DeliveryChannel, error) {
	channels, err := s.channels.ListAll()
	if err != nil {
		return nil, NewServiceErrorWrap(ErrInternal, "list delivery channels failed", err)
	}
	return channels, nil
}

func (s *AdminService) UpdateDeliveryChannel(id int, name *string, webhookURL *string, keywords *string, enabled *bool) (*models.DeliveryChannel, error) {
	channel, err := s.channels.GetByID(id)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return nil, NewServiceErrorWrap(ErrNotFound, fmt.Sprintf("delivery channel with ID %d not found", id), err)
		}
		return nil, NewServiceErrorWrap(ErrInternal, fmt.Sprintf("get delivery channel with ID %d failed", id), err)
	}

	if name != nil {
		channel.Name = *name
	}
	if enabled != nil {
		channel.Enabled = *enabled
	}
	if webhookURL != nil {
		if err := validateChannel(channel.Kind, *webhookURL); err != nil {
			return nil, err
		}
		channel.WebhookURL = *webhookURL
	}
	if keywords != nil {
		channel.Keywords = *keywords
	}

	if err := s.channels.Update(channel); err != nil {
		return nil, NewServiceErrorWrap(ErrInternal, fmt.Sprintf("update delivery channel with ID %d failed", id), err)
	}

	updated, err := s.channels.GetByID(id)
	if err != nil {
		return nil, NewServiceErrorWrap(ErrInternal, fmt.Sprintf("re-query delivery channel with ID %d failed", id), err)
	}

	return updated, nil
}

func (s *AdminService) DeleteDeliveryChannel(id int) error {
	rows, err := s.channels.DeleteByID(id)
	if err != nil {
		return NewServiceErrorWrap(ErrInternal, fmt.Sprintf("delete delivery channel with ID %d failed", id), err)
	}
	if rows == 0 {
		return NewServiceError(ErrNotFound, fmt.Sprintf("delivery channel with ID %d not found", id))
	}
	return nil
}
