-- 000003_email_optional_unique: allow multiple users with empty email.
-- The original UNIQUE(email) constraint blocks a second empty-string email;
-- replace it with a partial unique index that only enforces uniqueness for
-- non-empty addresses.
ALTER TABLE users DROP CONSTRAINT IF EXISTS uni_users_email;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_unique
    ON users (email)
    WHERE email <> '';
