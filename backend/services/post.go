package services

import (
	"bytes"
	"fmt"

	"markpost/models"
	"markpost/repositories"

	"github.com/yuin/goldmark"
)

type PostService struct {
	postRepo repositories.PostRepoInterface
}

func NewPostService(postRepo repositories.PostRepoInterface) *PostService {
	return &PostService{postRepo: postRepo}
}

func (s *PostService) CreatePost(title, body string, userID int) (string, error) {
	post, err := s.postRepo.CreatePost(title, body, userID)
	if err != nil {
		return "", NewServiceErrorWrap(ErrInternal, "create post failed", err)
	}
	return post.QID, nil
}

func (s *PostService) RenderPostHTML(qid string) (string, string, error) {
	post, err := s.postRepo.GetPostByQID(qid)
	if err != nil {
		if err == models.ErrNotFound {
			return "", "", NewServiceErrorWrap(ErrNotFound, fmt.Sprintf("post with qid %s not found", qid), err)
		}

		return "", "", NewServiceErrorWrap(ErrInternal, fmt.Sprintf("get post with qid %s failed", qid), err)
	}

	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(post.Body), &buf); err != nil {
		return "", "", NewServiceErrorWrap(ErrInternal, fmt.Sprintf("convert post with qid %s failed", qid), err)
	}

	return post.Title, buf.String(), nil
}

func (s *PostService) GetUserPosts(userID int, page int, limit int) ([]models.Post, int64, error) {
	if page <= 0 {
		page = 1
	}

	if limit <= 0 {
		limit = 20
	}

	total, err := s.postRepo.CountPostsByUserID(userID)
	if err != nil {
		return nil, 0, NewServiceErrorWrap(ErrInternal, fmt.Sprintf("query posts count failed with userID %d", userID), err)
	}

	offset := (page - 1) * limit
	posts, err := s.postRepo.GetPostsByUserID(userID, offset, limit)
	if err != nil {
		return nil, 0, NewServiceErrorWrap(ErrInternal, fmt.Sprintf("query posts failed with userID %d", userID), err)
	}

	return posts, total, nil
}
