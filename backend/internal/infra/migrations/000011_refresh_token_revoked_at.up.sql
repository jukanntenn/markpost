-- Refresh token rotation grace window (MRFC 2026-09-03-refresh-token-rotation-grace-window):
-- a nullable revocation timestamp lets the reuse-detection path tell a replay
-- of a freshly rotated token (rotation race) from theft. NULL = the row was
-- revoked before this column existed and takes the strict path.
ALTER TABLE refresh_tokens ADD COLUMN revoked_at timestamptz;
