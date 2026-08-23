-- 运行时设置表（MRFC 2026-08-23-github-login-vip-grant-strategy）：运营策略
-- 开关的共用归宿，管理员经 API 翻转、即时生效、走审计；值用 JSONB，
-- 未来策略扩展结构而不扩展 schema。播种 vip 策略为开启。
CREATE TABLE settings (
    key text PRIMARY KEY,
    value jsonb NOT NULL,
    updated_by bigint,
    updated_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO settings (key, value) VALUES ('vip', '{"enabled": true}'::jsonb);
