-- Markpost Development Seed Data
-- Run manually if needed: sqlite3 data/markpost.db < seed.sql
--
-- Admin Account:
--   Username: markpost
--   Password: markpost

INSERT OR IGNORE INTO posts (id, user_id, title, body, qid, created_at, updated_at)
VALUES
    (1, 1, 'Welcome to Markpost', 'This is your first post. Markpost is a simple pastebin service for sharing text snippets.', 'welcome', datetime('now'), datetime('now')),
    (2, 1, 'Markdown Example', '## Heading\n\n- item one\n- item two\n\n```go\nfmt.Println("hello")\n```', 'markdown-demo', datetime('now'), datetime('now'));
