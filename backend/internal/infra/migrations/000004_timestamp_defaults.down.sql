-- 000004_timestamp_defaults.down: revert to nullable, no-default columns
-- (restoring the original GORM-hook-only behavior).

-- posts
ALTER TABLE posts ALTER COLUMN updated_at DROP NOT NULL;
ALTER TABLE posts ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE posts ALTER COLUMN created_at DROP NOT NULL;
ALTER TABLE posts ALTER COLUMN created_at DROP DEFAULT;

-- users
ALTER TABLE users ALTER COLUMN updated_at DROP NOT NULL;
ALTER TABLE users ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE users ALTER COLUMN created_at DROP NOT NULL;
ALTER TABLE users ALTER COLUMN created_at DROP DEFAULT;

-- delivery_attempts
ALTER TABLE delivery_attempts ALTER COLUMN updated_at DROP NOT NULL;
ALTER TABLE delivery_attempts ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE delivery_attempts ALTER COLUMN created_at DROP NOT NULL;
ALTER TABLE delivery_attempts ALTER COLUMN created_at DROP DEFAULT;

-- delivery_channels
ALTER TABLE delivery_channels ALTER COLUMN updated_at DROP NOT NULL;
ALTER TABLE delivery_channels ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE delivery_channels ALTER COLUMN created_at DROP NOT NULL;
ALTER TABLE delivery_channels ALTER COLUMN created_at DROP DEFAULT;

-- delivery_history
ALTER TABLE delivery_history ALTER COLUMN created_at DROP NOT NULL;
ALTER TABLE delivery_history ALTER COLUMN created_at DROP DEFAULT;

-- refresh_tokens
ALTER TABLE refresh_tokens ALTER COLUMN created_at DROP NOT NULL;
ALTER TABLE refresh_tokens ALTER COLUMN created_at DROP DEFAULT;

-- token_blacklist
ALTER TABLE token_blacklist ALTER COLUMN created_at DROP NOT NULL;
ALTER TABLE token_blacklist ALTER COLUMN created_at DROP DEFAULT;
