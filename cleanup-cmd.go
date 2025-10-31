package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func CleanupCommand() {
	var (
		batchSize = flag.Int("batch-size", 100, "Number of records to delete per batch")
		dryRun    = flag.Bool("dry-run", false, "Preview mode, only show the number of records to be deleted")
		preview   = flag.Int("preview", 0, "Preview number of records to be deleted (specify count)")
		help      = flag.Bool("help", false, "Show help information")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "markpost data cleanup tool\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s cleanup [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s cleanup --dry-run                    # Preview the number of records to be deleted\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s cleanup --preview 10                # Preview the first 10 records to be deleted\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s cleanup --batch-size 50             # Delete 50 records per batch\n", os.Args[0])
	}

	flag.Parse()

	if *help {
		flag.Usage()
		return
	}

	if err := LoadConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	dbInstance, err := NewDatabase(config.Database.URL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		sqlDB, err := dbInstance.GetDB().DB()
		if err == nil && sqlDB != nil {
			sqlDB.Close()
		}
	}()
	database = dbInstance

	retentionDays := config.DataCleanup.PostRetentionDays

	if retentionDays <= 0 {
		log.Fatalf("Configuration error: post_retention_days must be greater than 0, current value: %d", retentionDays)
	}

	log.Printf("Configuration: post retention days = %d days", retentionDays)

	if *preview > 0 {
		posts, err := database.GetPostRepository().PreviewExpiredPosts(retentionDays, *preview)
		if err != nil {
			log.Fatalf("Failed to preview expired posts: %v", err)
		}

		if len(posts) == 0 {
			fmt.Println("No expired post records found")
			return
		}

		fmt.Printf("First %d expired posts to be deleted:\n", len(posts))
		fmt.Println("=====================================")
		for i, post := range posts {
			fmt.Printf("%d. QID: %s\n", i+1, post.QID)
			fmt.Printf("   Title: %s\n", truncateString(post.Title, 50))
			fmt.Printf("   Created at: %s\n", post.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("   User ID: %d\n", post.UserID)
			fmt.Println("   ---")
		}
		return
	}

	if *dryRun {
		count, err := database.GetPostRepository().GetExpiredPostsCount(retentionDays)
		if err != nil {
			log.Fatalf("Failed to count expired posts: %v", err)
		}

		fmt.Printf("Preview: Will delete %d post records older than %d days\n", count, retentionDays)

		if count > 0 {
			fmt.Printf("Tip: Use --preview 10 to view details of the first 10 records\n")
			fmt.Printf("Tip: Remove --dry-run parameter to perform actual deletion\n")
		}
		return
	}

	if err := database.GetPostRepository().CleanupExpiredPosts(retentionDays, *batchSize); err != nil {
		log.Fatalf("Failed to cleanup expired posts: %v", err)
	}

	fmt.Println("Data cleanup completed")
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
