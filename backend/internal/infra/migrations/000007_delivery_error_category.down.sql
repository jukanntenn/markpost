DROP INDEX IF EXISTS idx_dh_error_category;

ALTER TABLE delivery_history DROP COLUMN IF EXISTS error_category;
