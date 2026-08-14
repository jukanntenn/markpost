-- Pt1 可观测性：给 delivery_history 加规范化的 error_category，让管理后台
-- 投递历史能按失败类别筛选/分组（card_rejected / upstream_* / network / internal）。
-- 旧行 category 留空（admin “all” 仍可见）。
ALTER TABLE delivery_history ADD COLUMN error_category TEXT NOT NULL DEFAULT '';

CREATE INDEX idx_dh_error_category ON delivery_history (error_category);
