package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"markpost/conf"
	"markpost/models"
	"markpost/repositories"
)

type FakePostData struct {
	QID       string    `json:"qid"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func RunImportFakePosts(configPath, filePath string) error {
	if err := conf.LoadConfig(configPath); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	database, err := models.NewDatabase(conf.Conf().DB.DSN)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() {
		sqlDB, err := database.DB().DB()
		if err == nil && sqlDB != nil {
			sqlDB.Close()
		}
	}()

	userRepo := repositories.NewUserRepo(database)
	user, err := userRepo.GetUserByUsername("markpost")
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

	posts := make([]models.Post, 0, len(fakePosts))
	for _, fp := range fakePosts {
		post := models.Post{
			QID:       fp.QID,
			Title:     fp.Title,
			Body:      fp.Body,
			CreatedAt: fp.CreatedAt,
			UpdatedAt: fp.UpdatedAt,
			UserID:    user.ID,
		}
		posts = append(posts, post)
	}

	postRepo := repositories.NewPostRepo(database)
	count, err := postRepo.CreatePosts(posts)
	if err != nil {
		return fmt.Errorf("failed to import posts: %w", err)
	}

	fmt.Printf("成功导入 %d 条 Post 数据\n", count)
	return nil
}
