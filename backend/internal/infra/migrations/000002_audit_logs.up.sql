-- 000002_audit_logs: append-only audit trail for admin write operations.
CREATE TABLE IF NOT EXISTS audit_logs (
    id          bigserial    PRIMARY KEY,
    actor_id    integer      NOT NULL,
    action      varchar(64)  NOT NULL,
    target_type varchar(32)  NOT NULL,
    target_id   varchar(64),
    metadata    jsonb        NOT NULL DEFAULT '{}'::jsonb,
    ip          varchar(45),
    created_at  timestamptz  NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_actor_id    ON audit_logs (actor_id);
CREATE INDEX idx_audit_logs_created_at  ON audit_logs (created_at DESC);
CREATE INDEX idx_audit_logs_target      ON audit_logs (target_type, target_id);
