// Package cmd provides CLI commands for the application.
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"markpost/internal/config"
	"markpost/internal/domain/post"
	"markpost/internal/infra"
)

// FakePostData represents a fake post data structure for importing.
type FakePostData struct {
	QID       string    `json:"qid"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RunImportFakePosts imports fake posts from a JSON file.
func RunImportFakePosts(configPath, filePath string) error {
	if err := config.Load(configPath); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cfg := config.Get()

	dbInstance, err := infra.New(cfg.DB.DSN)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer dbInstance.Close()

	userRepo := infra.NewUserRepository(dbInstance.DB(), cfg.PostKeyLength)
	u, err := userRepo.GetByUsername(context.Background(), "markpost")
	if err != nil {
		return fmt.Errorf("user 'markpost' not found: %w", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	var fakePosts []FakePostData
	if err := json.Unmarshal(data, &fakePosts); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	posts := make([]post.Post, 0, len(fakePosts))
	for _, fp := range fakePosts {
		p := post.Post{
			QID:       fp.QID,
			Title:     fp.Title,
			Body:      fp.Body,
			CreatedAt: fp.CreatedAt,
			UpdatedAt: fp.UpdatedAt,
			UserID:    u.ID,
		}
		posts = append(posts, p)
	}

	postRepo := infra.NewPostRepository(dbInstance.DB())
	count, err := postRepo.CreateBatch(context.Background(), posts)
	if err != nil {
		return fmt.Errorf("failed to import posts: %w", err)
	}

	fmt.Printf("成功导入 %d 条 Post 数据\n", count)
	return nil
}
