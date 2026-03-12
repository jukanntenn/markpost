package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"markpost/internal/config"
	"markpost/internal/domain/post"
	"markpost/internal/infra/database"
)

type FakePostData struct {
	QID       string    `json:"qid"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func RunImportFakePosts(configPath, filePath string) error {
	if err := config.Load(configPath); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cfg := config.Get()

	dbInstance, err := database.New(cfg.DB.DSN)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() {
		sqlDB, err := dbInstance.DB().DB()
		if err == nil && sqlDB != nil {
			sqlDB.Close()
		}
	}()

	userRepo := database.NewUserRepository(dbInstance.DB())
	u, err := userRepo.GetByUsername(nil, "markpost")
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

	postRepo := database.NewPostRepository(dbInstance.DB())
	count, err := postRepo.CreateBatch(nil, posts)
	if err != nil {
		return fmt.Errorf("failed to import posts: %w", err)
	}

	fmt.Printf("成功导入 %d 条 Post 数据\n", count)
	return nil
}
