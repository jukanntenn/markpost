package main

import (
	"fmt"
	"log"
	"time"
)

// CleanupExpiredPosts 清理过期的 post 记录
func CleanupExpiredPosts(retentionDays int, batchSize int) error {
	if retentionDays <= 0 {
		return fmt.Errorf("retention days must be positive, got: %d", retentionDays)
	}

	if batchSize <= 0 {
		batchSize = 100 // 默认批量大小
	}

	// 计算过期时间点
	expiredBefore := time.Now().AddDate(0, 0, -retentionDays)

	log.Printf("[CLEANUP] 开始清理创建时间早于 %s 的 post 记录", expiredBefore.Format("2006-01-02 15:04:05"))
	log.Printf("[CLEANUP] 批量大小: %d", batchSize)

	totalDeleted := 0
	batchNumber := 1

	for {
		// 查询一批过期的 post ID
		var postIDs []string
		result := db.Model(&Post{}).
			Select("id").
			Where("created_at < ?", expiredBefore).
			Limit(batchSize).
			Pluck("id", &postIDs)

		if result.Error != nil {
			return fmt.Errorf("查询过期 post 失败: %v", result.Error)
		}

		// 如果没有更多记录，退出循环
		if len(postIDs) == 0 {
			log.Printf("[CLEANUP] 没有更多过期的 post 需要清理")
			break
		}

		// 删除这批记录
		result = db.Where("id IN ?", postIDs).Delete(&Post{})
		if result.Error != nil {
			return fmt.Errorf("删除 post 批次 %d 失败: %v", batchNumber, result.Error)
		}

		deletedCount := result.RowsAffected
		totalDeleted += int(deletedCount)

		log.Printf("[CLEANUP] 批次 %d: 删除了 %d 条记录 (Post IDs: %v)",
			batchNumber, deletedCount, postIDs)

		// 如果删除的记录数少于批量大小，说明已经处理完了
		if deletedCount < int64(batchSize) {
			log.Printf("[CLEANUP] 已删除所有过期记录")
			break
		}

		batchNumber++

		// 为了避免对数据库造成过大压力，在批次之间稍作暂停
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("[CLEANUP] 清理完成，共删除 %d 条过期 post 记录", totalDeleted)
	return nil
}

// GetExpiredPostsCount 获取过期 post 的数量（用于预览）
func GetExpiredPostsCount(retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		return 0, fmt.Errorf("retention days must be positive, got: %d", retentionDays)
	}

	expiredBefore := time.Now().AddDate(0, 0, -retentionDays)

	var count int64
	result := db.Model(&Post{}).Where("created_at < ?", expiredBefore).Count(&count)
	if result.Error != nil {
		return 0, fmt.Errorf("统计过期 post 数量失败: %v", result.Error)
	}

	return count, nil
}

// PreviewExpiredPosts 预览即将删除的过期 post（显示前几条）
func PreviewExpiredPosts(retentionDays int, limit int) ([]Post, error) {
	if retentionDays <= 0 {
		return nil, fmt.Errorf("retention days must be positive, got: %d", retentionDays)
	}

	if limit <= 0 {
		limit = 10 // 默认显示 10 条
	}

	expiredBefore := time.Now().AddDate(0, 0, -retentionDays)

	var posts []Post
	result := db.Where("created_at < ?", expiredBefore).
		Order("created_at ASC").
		Limit(limit).
		Find(&posts)

	if result.Error != nil {
		return nil, fmt.Errorf("查询过期 post 预览失败: %v", result.Error)
	}

	return posts, nil
}
