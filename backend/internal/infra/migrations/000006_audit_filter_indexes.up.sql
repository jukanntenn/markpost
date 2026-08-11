-- K.7 D4-3：审计筛选复合索引（D4 筛选 = actor/action/target/时间范围）。
-- 现有 (actor_id)、(created_at DESC)、(target_type, target_id) 单列/双列索引
-- 无法支撑带时间范围的复合筛选。
CREATE INDEX idx_audit_action_created ON audit_logs (action, created_at);
CREATE INDEX idx_audit_target_created ON audit_logs (target_type, target_id, created_at);
CREATE INDEX idx_audit_actor_created ON audit_logs (actor_id, created_at);
