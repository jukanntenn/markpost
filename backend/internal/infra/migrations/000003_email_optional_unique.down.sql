-- Revert to the original table-level UNIQUE constraint on email.
DROP INDEX IF EXISTS idx_users_email_unique;

ALTER TABLE users ADD CONSTRAINT uni_users_email UNIQUE (email);
