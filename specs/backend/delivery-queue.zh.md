# 投递队列数据模型

[English](delivery-queue.md) | 中文

投递队列将其状态持久化在两张按访问模式与生命周期划分的 PostgreSQL 表中：`delivery_attempts`（热表，短生命周期行）与 `delivery_history`（冷表，7 天归档）。GORM 模型位于 `internal/domain/delivery/delivery.go`；schema —— 表、索引与 Postgres 存储选项 —— 声明在版本化 SQL 迁移中（表与索引在 `000001_init.up.sql`，历史表的错误分类列在 `000007_delivery_error_category.up.sql`）。设计严格遵循数据库范式：除作为性能所需查询键之外没有冗余列，外键强制引用完整性。清空队列的调度器见 [`delivery-scheduler.zh.md`](./delivery-scheduler.zh.md)；重试时序与终态见 [`delivery-retry.zh.md`](./delivery-retry.zh.md)；决策理由见[投递 MRFC](../../.agents/mrfcs/implemented/2026-07-10-persistent-best-effort-delivery-queue.zh.md)。

<a id="the-status-enum-shared-by-both-tables"></a>

## `Status` 枚举（两张表共享）

```go
type Status int8

const (
    StatusPending   Status = 0 // default; "due" / "in-flight"
    StatusDelivered Status = 1 // terminal — a send succeeded
    StatusFailed    Status = 2 // terminal — sequence exhausted
    StatusExpired   Status = 3 // terminal — wall passed
)
```

用 `type Status int8`（而非 `user.User` 使用的 `type Role string` 模式）让 status 列保持紧凑，且无需列类型 tag：

- **现有最紧凑的形式。** GORM 从 Go 的 `reflect.Kind` 解析列类型，整数类型再从自动计算的 `Size` 解析（`schema/field.go`）。`int8` → `Size=8`；Postgres 驱动把每个 ≤16 位的整数映射为 `smallint`（2 字节）—— Postgres 没有 1 字节整数类型，2 字节即其下限。
- **无需 `type:` tag。** GORM 会原样输出 `type:` tag 的值，因此手写列类型将完全接管 DDL；裸 `int8` 形式依赖基于 size 的驱动映射，不会与驱动自身的选择发生漂移。
- **`StatusPending = 0`** 使数据库默认值（`default:0`）落在 pending 状态上；列默认值无需字面量。
- **不使用原生 ENUM 列。** 新状态即新常量 —— 无 schema 变更、无重写风险。代价是：值以 `0/1/2/3` 存储，数据库查看时显示数字，且 `iota` 顺序**永远只允许追加** —— 在中间插入状态会给之后所有状态重新编号，并静默损坏现有行。终态不会被重新加入，因此追加（例如未来的 `StatusCanceled = 4`）总是安全的。

<a id="delivery_attempts-hot-queue--short-lived"></a>

## `delivery_attempts`（热队列 —— 短生命周期）

```go
type Attempt struct {
    ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
    UserID    int       `json:"user_id" gorm:"not null;column:user_id;index"`
    PostID    int       `json:"post_id" gorm:"not null;column:post_id;index"`
    ChannelID int       `json:"channel_id" gorm:"not null;column:channel_id;index"`
    Status    Status    `json:"status" gorm:"not null;default:0"`
    Attempts  int       `json:"attempts" gorm:"not null;default:0"`
    NextAt    int64     `json:"next_at" gorm:"not null"`          // epoch ms; when the next attempt may run
    LastError string    `json:"last_error" gorm:"not null;type:text;default:''"`
    CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"` // drives the expiry wall
    UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

    User    user.User `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
    Post    post.Post `json:"-" gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
    Channel Channel   `json:"-" gorm:"foreignKey:ChannelID;constraint:OnDelete:CASCADE"`
}
```

**生命周期。** 一行仅在投递进行期间存在 —— 至多一个过期墙（40 分钟）。到达任一终态时，行在同一事务内归档进 `delivery_history` 并删除（`ArchiveAndDelete`）。稳态行数因此以墙窗口为界（116 任务/秒上限下约 280,000 行 × 2400 秒 ≈ 22 MB）。

**posts/channels/users 上的 ON DELETE CASCADE。** 为已删除文章或渠道所做的投递尝试没有意义（飞书卡片链接的是一篇已不存在的文章），因此把删除级联到其尝试行是正确的语义。这里 CASCADE 是安全的，_因为_尝试行存活 ≤40 分钟 —— 级联删除的永远是一个小而有界的集合。

**`user_id` 反范式化（唯一一处）。** `user_id` 技术上可经 `post_id → posts.user_id` 推导，但保留为查询键：调度器与历史查询按用户过滤，在热路径上避免一次 join 是有意的性能权衡。其余每一列都非冗余。

**不快照文章正文或标题。** 一次投递尝试至多存活 40 分钟，而文章保留 7 天（`post.retention_days = 7`），因此投递时文章必然存在；worker 做主键 `GetByID` 并在那时读取正文。这让尝试行保持窄行（约 80 字节），并避免了一个在这个时间尺度上并不存在的快照一致性问题。

<a id="delivery_history-cold-archive--7-day-user-facing-record"></a>

## `delivery_history`（冷归档 —— 7 天用户可见记录）

```go
type History struct {
    ID            int64     `json:"id" gorm:"primaryKey;autoIncrement"`
    UserID        *int      `json:"user_id" gorm:"column:user_id;index"`       // nullable; ON DELETE SET NULL
    PostID        *int      `json:"post_id" gorm:"column:post_id;index"`       // nullable; ON DELETE SET NULL
    ChannelID     *int      `json:"channel_id" gorm:"column:channel_id;index"` // nullable; ON DELETE SET NULL
    Status        Status    `json:"status" gorm:"not null"`                    // delivered | failed | expired
    LastError     string    `json:"last_error" gorm:"not null;type:text;default:''"`
    ErrorCategory string    `json:"error_category" gorm:"not null;type:text;default:''"` // classified send-failure category (empty for delivered/expired/legacy rows)
    CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`

    User    *user.User `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL"`
    Post    *post.Post `json:"-" gorm:"foreignKey:PostID;constraint:OnDelete:SET NULL"`
    Channel *Channel   `json:"-" gorm:"foreignKey:ChannelID;constraint:OnDelete:SET NULL"`
}
```

**`user_id` 是 `ON DELETE SET NULL` 而非 CASCADE。** 一个用户 7 天的历史可能很大（最坏负载下数百万行）。`ON DELETE CASCADE` 会在 `DELETE FROM users` 事务内全部删除它们，产生使表停滞的大规模死元组爆发。`SET NULL` 则保留每一行历史并把 `user_id` 置空；界面在 `user_id` 为 null 时渲染本地化的"用户已删除"占位符，与 `post_id`/`channel_id` 为 null 时渲染"文章已删除"/"渠道已删除"占位符完全一致。这使 `user_id` 可空（`*int`），与本来就可空的 `post_id`/`channel_id` 一致。

**attempts 上 CASCADE 与 history 上 SET NULL —— 差异缘由。** 尝试行存活 ≤40 分钟，删除用户（或文章/渠道）触发的级联至多触及该用户当前在途的尝试 —— 一个小而有界的集合。历史行存活 7 天，同样的级联会触及该用户的全部投递记录 —— 无界且庞大。两张表因此对_同一个逻辑列_采取不同的外键动作：有界生命周期 → CASCADE（廉价、语义干净）；无界生命周期 → SET NULL（无锁风暴、保留审计）。

**严格范式化 —— 无快照列、无尝试计数、无 delivered_at。** 标题、渠道名与用户名在读取时 JOIN；`delivery_history` 记录结果（`status`）、失败细节（`last_error` + `error_category`）与投递发起时间（`created_at`）。用户的历史页是分页的（20 行），主键 join 是亚毫秒级 —— 反范式化没有性能理由。规范读取：

```sql
SELECT h.id, h.status, h.last_error, h.error_category, h.created_at,
       p.title AS post_title, p.qid AS post_qid,
       c.name AS channel_name,
       u.username AS username
FROM delivery_history h
LEFT JOIN posts p ON p.id = h.post_id
LEFT JOIN delivery_channels c ON c.id = h.channel_id
LEFT JOIN users u ON u.id = h.user_id
WHERE h.user_id = $1            -- NULL user_id rows are excluded from a user's own page;
                                -- the admin "all history" view omits the user filter
  AND h.channel_id = $2         -- optional; omitted when the user views all channels
ORDER BY h.created_at DESC
LIMIT 20 OFFSET $3;
```

`HistoryRow`（`delivery.go`）是此 join 的 GORM 读取投影；其可空指针字段对应 `ON DELETE SET NULL` —— nil 字段表示被引用行已被删除。

**`error_category`。** 终态失败携带分类后的发送失败类别（见 [`delivery-retry.zh.md`](./delivery-retry.zh.md)）；`delivered` 与 `expired` 行存储空字符串，该列存在之前写入的行同样如此。类别驱动管理员历史过滤器（`HistoryFilter.ErrorCategory`）与 `delivery_failed` 指标标签。

**历史读取面。** `AttemptRepository`（internal/domain/delivery/repository.go）暴露 UI 所需的查询集，全部落在这张表上：`ListHistory`/`CountHistory`（带用户/渠道/状态/错误类别过滤的分页用户或管理员历史）、`LatestPerChannel`（渠道列表上每渠道的投递健康概览）、`DailyStats`/`DailyStatsAll`（趋势图用的按天结果计数）、`TodayCounts`（今日已投递/失败加上在途 pending）、`CountSince`（管理员周环比增量），以及 `LockedChannels`（24 小时窗口内全部失败或失败率 >50% 的渠道，供管理员"需要关注"卡片使用）。

**存储。** 7 天约 7000 万行，每行约 60 字节 ≈ **4.2 GB**，连同文章的约 11 GB 一起远在 40 GB 磁盘之内。

<a id="index-design"></a>

## 索引设计

认领查询是 `WHERE status = 0 AND next_at <= ? ORDER BY next_at`。GORM 无法在 struct tag 中表达部分索引，因此索引声明在迁移 SQL 中。

**`delivery_attempts`** —— 部分索引（保护 HOT 更新）：

```sql
CREATE INDEX idx_da_pending
    ON delivery_attempts (next_at)
    WHERE status = 0;
```

热表的工作负载是 `status` 迁移 UPDATE（`pending` → 终态）。只覆盖 `pending` 行的部分索引让索引保持极小，并与 Postgres HOT（Heap-Only Tuple）更新协作：行一旦离开 `pending`，其索引条目即被丢弃，后续 UPDATE 永远不会触及该索引。HOT 要求更新不得修改该表索引引用的任何列；这个索引的键只有 `next_at`，而状态迁移 UPDATE 虽然确实会修改 `next_at`（认领时的预留推进），但该行在同一个迁移中_离开了部分索引的谓词_（`status=0`），因此是从索引中移除，而非就地重建。（依据 Postgres 关于 fillfactor、部分索引与 HOT 的 `CREATE TABLE`/索引/存储文档。）

**`delivery_history`** —— 单个索引无法同时服务的两种工作负载形态：

```sql
CREATE INDEX idx_dh_user_channel_created ON delivery_history (user_id, channel_id, created_at DESC);
CREATE INDEX idx_dh_created              ON delivery_history (created_at DESC);
```

- `idx_dh_user_channel_created` —— `user_id` 打头，作为始终存在的等值谓词；`channel_id` 跟随，作为可选等值谓词；`created_at DESC` 收尾，在两个等值都在场时支撑 `ORDER BY`。
- `idx_dh_created` —— 服务管理员"全部历史"视图（`ORDER BY created_at DESC`，**无** `user_id` 谓词）。`user_id` 打头的组合索引无法服务该查询（没有最左等值），因此没有这个索引，管理页会退化为约 7000 万行冷表上的全表扫描 + 排序。
- `idx_dh_error_category`（`000007`）—— 支撑管理员按错误类别过滤 `delivery_history`。

<a id="table-tuning"></a>

## 表级调优

仅 Postgres 支持的 `WITH (...)` reloptions 无法放进 GORM struct，因此声明在同一份迁移 SQL 中：

```sql
ALTER TABLE delivery_attempts SET (
    fillfactor = 90,
    autovacuum_vacuum_scale_factor = 0.05,
    autovacuum_vacuum_threshold = 1000,
    autovacuum_analyze_scale_factor = 0.02,
    autovacuum_analyze_threshold = 1000
);
ALTER TABLE delivery_history SET (fillfactor = 100);
```

- 热表 `fillfactor = 90` 预留页内空间，让状态迁移 UPDATE 把更新后的元组落在同一页上 → HOT 大概率发生。
- 只追加的历史表 `fillfactor = 100`（无更新）→ 完全装满，存储最小。
- 高 UPDATE 的 attempts 表上激进的 autovacuum 及时回收死元组。

<a id="retention"></a>

## 保留期

`delivery_history` 行超过 7 天保留窗口（`[delivery] history_retention`，默认 `168h`）即被清扫：以子查询 `LIMIT` 形式的分批 DELETE（`DELETE ... WHERE id IN (SELECT id ... WHERE created_at < $cutoff ORDER BY created_at LIMIT $batch)`）循环执行，直到没有剩余行为止。它由 cron 调用的 `prune-delivery-history` CLI 子命令（`cmd/prune_delivery_history.go`）运行，与 `prune-expired-posts` 是同一模式。历史行只追加，因此该 DELETE 永不与更新竞争。`delivery_attempts` 没有集中清扫 —— 分布式清理见 [`delivery-scheduler.zh.md`](./delivery-scheduler.zh.md)。
