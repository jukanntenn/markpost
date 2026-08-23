-- VIP 运营策略的用户标志（MRFC 2026-08-23-user-vip-flag）：授予即持久，
-- 独立于策略开关的后续状态；纯荣誉，不承载权限。
ALTER TABLE users ADD COLUMN vip boolean NOT NULL DEFAULT FALSE;
