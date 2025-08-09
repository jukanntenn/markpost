package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

// CleanupCommand 数据清理命令
func CleanupCommand() {
	// 定义命令行参数
	var (
		batchSize = flag.Int("batch-size", 100, "单次删除的记录数")
		dryRun    = flag.Bool("dry-run", false, "预览模式，只显示将要删除的记录数量")
		preview   = flag.Int("preview", 0, "预览即将删除的记录数（指定显示条数）")
		help      = flag.Bool("help", false, "显示帮助信息")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "markpost 数据清理工具\n\n")
		fmt.Fprintf(os.Stderr, "用法: %s cleanup [选项]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "选项:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n示例:\n")
		fmt.Fprintf(os.Stderr, "  %s cleanup --dry-run                    # 预览将要删除的记录数量\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s cleanup --preview 10                # 预览前10条即将删除的记录\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s cleanup --batch-size 50             # 每次删除50条记录\n", os.Args[0])
	}

	flag.Parse()

	if *help {
		flag.Usage()
		return
	}

	// 加载配置
	if err := LoadConfig(); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化数据库
	InitDB()
	defer CloseDB()

	retentionDays := config.DataCleanup.PostRetentionDays

	if retentionDays <= 0 {
		log.Fatalf("配置错误: post_retention_days 必须大于 0，当前值: %d", retentionDays)
	}

	log.Printf("配置: post 保留天数 = %d 天", retentionDays)

	// 预览模式
	if *preview > 0 {
		posts, err := PreviewExpiredPosts(retentionDays, *preview)
		if err != nil {
			log.Fatalf("预览过期 post 失败: %v", err)
		}

		if len(posts) == 0 {
			fmt.Println("没有找到过期的 post 记录")
			return
		}

		fmt.Printf("前 %d 条即将删除的过期 post:\n", len(posts))
		fmt.Println("=====================================")
		for i, post := range posts {
			fmt.Printf("%d. ID: %s\n", i+1, post.ID)
			fmt.Printf("   标题: %s\n", truncateString(post.Title, 50))
			fmt.Printf("   创建时间: %s\n", post.CreatedAt.Format("2006-01-02 15:04:05"))
			if post.UserID != nil {
				fmt.Printf("   用户ID: %d\n", *post.UserID)
			} else {
				fmt.Printf("   用户ID: (匿名)\n")
			}
			fmt.Println("   ---")
		}
		return
	}

	// Dry run 模式
	if *dryRun {
		count, err := GetExpiredPostsCount(retentionDays)
		if err != nil {
			log.Fatalf("统计过期 post 数量失败: %v", err)
		}

		fmt.Printf("预览: 将要删除 %d 条创建时间超过 %d 天的 post 记录\n", count, retentionDays)

		if count > 0 {
			fmt.Printf("提示: 使用 --preview 10 可以查看前10条记录的详细信息\n")
			fmt.Printf("提示: 移除 --dry-run 参数可以执行实际删除操作\n")
		}
		return
	}

	// 执行实际清理
	if err := CleanupExpiredPosts(retentionDays, *batchSize); err != nil {
		log.Fatalf("清理过期 post 失败: %v", err)
	}

	fmt.Println("数据清理完成")
}

// truncateString 截断字符串到指定长度
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
