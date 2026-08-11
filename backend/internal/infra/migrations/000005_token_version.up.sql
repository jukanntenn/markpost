-- C2.6 token version：改密/强制下线/封禁三类事件的统一即时失效原语。
-- 现有 JWT 无 tv claim（视为 0），与 DEFAULT 0 兼容；迁移后首次改密等事件
-- 即自增，使旧 token 立即失效。
ALTER TABLE users ADD COLUMN token_version bigint NOT NULL DEFAULT 0;
