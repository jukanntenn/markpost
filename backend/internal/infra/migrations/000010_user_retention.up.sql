-- 按用户历史保留策略（MRFC 2026-08-31-per-user-history-retention-policy）：
-- NULL = 继承各表全局配置；0 = 永久保存（沿用 [post] retention_days 的
-- 0 = 永不过期编码）；1–3650 = 保留 N 天。一个值同时驱动 posts 与
-- delivery_history 的清理窗口。
ALTER TABLE users ADD COLUMN retention_days integer;
