-- 000004_timestamp_defaults: backfill NULLs and give every created_at/
-- updated_at column a DB-level NOT NULL DEFAULT now(). Previously these columns
-- relied solely on GORM's autoCreateTime/autoUpdateTime hooks, so any write
-- that bypassed GORM (manual SQL, future importers) left them NULL and they
-- deserialized to Go's zero time.Time (0001-01-01). expires_at / last_login_at
-- are intentionally untouched: they carry independent semantics.

-- Backfill any existing NULL rows with the current instant so the NOT NULL
-- constraint below can be applied without data loss.
UPDATE posts             SET created_at = now(), updated_at = now() WHERE created_at IS NULL;
UPDATE users             SET created_at = now()                    WHERE created_at IS NULL;
UPDATE users             SET updated_at = now()                    WHERE updated_at IS NULL;
UPDATE delivery_attempts SET created_at = now(), updated_at = now() WHERE created_at IS NULL;
UPDATE delivery_channels SET created_at = now(), updated_at = now() WHERE created_at IS NULL;
UPDATE delivery_history  SET created_at = now()                    WHERE created_at IS NULL;
UPDATE refresh_tokens    SET created_at = now()                    WHERE created_at IS NULL;
UPDATE token_blacklist   SET created_at = now()                    WHERE created_at IS NULL;

-- posts
ALTER TABLE posts ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE posts ALTER COLUMN created_at SET NOT NULL;
ALTER TABLE posts ALTER COLUMN updated_at SET DEFAULT now();
ALTER TABLE posts ALTER COLUMN updated_at SET NOT NULL;

-- users
ALTER TABLE users ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE users ALTER COLUMN created_at SET NOT NULL;
ALTER TABLE users ALTER COLUMN updated_at SET DEFAULT now();
ALTER TABLE users ALTER COLUMN updated_at SET NOT NULL;

-- delivery_attempts
ALTER TABLE delivery_attempts ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE delivery_attempts ALTER COLUMN created_at SET NOT NULL;
ALTER TABLE delivery_attempts ALTER COLUMN updated_at SET DEFAULT now();
ALTER TABLE delivery_attempts ALTER COLUMN updated_at SET NOT NULL;

-- delivery_channels
ALTER TABLE delivery_channels ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE delivery_channels ALTER COLUMN created_at SET NOT NULL;
ALTER TABLE delivery_channels ALTER COLUMN updated_at SET DEFAULT now();
ALTER TABLE delivery_channels ALTER COLUMN updated_at SET NOT NULL;

-- delivery_history
ALTER TABLE delivery_history ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE delivery_history ALTER COLUMN created_at SET NOT NULL;

-- refresh_tokens
ALTER TABLE refresh_tokens ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE refresh_tokens ALTER COLUMN created_at SET NOT NULL;

-- token_blacklist
ALTER TABLE token_blacklist ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE token_blacklist ALTER COLUMN created_at SET NOT NULL;
