# Delivery Queue Data Model

The delivery queue persists its state in two PostgreSQL tables split by access pattern and lifecycle: `delivery_attempts` (hot, short-lived rows) and `delivery_history` (cold, 7-day archive). The GORM models live in `internal/domain/delivery/delivery.go`; the schema — tables, indexes, and Postgres storage options — is declared in versioned SQL migrations (`000001_init.up.sql` for the tables and indexes, `000007_delivery_error_category.up.sql` for the history error-classification column). The design follows database normalization strictly: no redundant columns unless required as a query key for performance, and foreign keys enforce referential integrity. For the scheduler that drains the queue see [`delivery-scheduler.md`](./delivery-scheduler.md); for retry timing and terminal states see [`delivery-retry.md`](./delivery-retry.md); the decision rationale lives in [the delivery MRFC](../../mrfc/implemented/2026-07-10-persistent-best-effort-delivery-queue.md).

## The `Status` enum (shared by both tables)

```go
type Status int8

const (
    StatusPending   Status = 0 // default; "due" / "in-flight"
    StatusDelivered Status = 1 // terminal — a send succeeded
    StatusFailed    Status = 2 // terminal — sequence exhausted
    StatusExpired   Status = 3 // terminal — wall passed
)
```

A `type Status int8` (rather than the `type Role string` pattern used by `user.User`) keeps the status column compact with no column-type tag:

- **Most compact available form.** GORM resolves the column type from the Go `reflect.Kind` and, for integer kinds, from the auto-computed `Size` (`schema/field.go`). `int8` → `Size=8`; the Postgres driver maps every integer ≤16 bits to `smallint` (2 bytes) — Postgres has no 1-byte integer type, so 2 bytes is its floor.
- **No `type:` tag needed.** GORM emits a `type:` tag value verbatim, so a hand-written column type would own the DDL outright; the bare `int8` form relies on the size-based driver mapping and cannot drift from the driver's own choice.
- **`StatusPending = 0`** so the database default (`default:0`) lands on the pending state; no literal is needed in the column default.
- **No native ENUM column.** A new status is a new constant — no schema change, no rewrite risk. The trade-off: values are stored as `0/1/2/3`, DB inspection shows numbers, and the `iota` order is **append-only forever** — inserting a state in the middle would renumber every later state and silently corrupt existing rows. Terminal states are never re-added, so appending (e.g. a future `StatusCanceled = 4`) is always safe.

## `delivery_attempts` (hot queue — short-lived)

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

**Lifecycle.** A row exists only while delivery is in progress — at most the expiry wall (40 minutes). On any terminal state the row is archived to `delivery_history` and deleted in the same transaction (`ArchiveAndDelete`). Steady-state row count is therefore bounded by the wall window (~280,000 rows at the 116 jobs/s ceiling × 2400 s ≈ 22 MB).

**ON DELETE CASCADE on posts/channels/users.** A delivery attempt for a deleted post or channel is meaningless (the Feishu card links to a dead post), so cascading the delete to its attempts is the correct semantics. CASCADE is safe here _because_ an attempt row lives ≤40 min — the cascade always deletes a small, bounded set.

**`user_id` denormalization (the only one).** `user_id` is technically derivable via `post_id → posts.user_id`, but it is retained as a query key: the scheduler and history queries filter by user, and avoiding a join on the hot path is a deliberate performance trade-off. Every other column is non-redundant.

**No post body or title snapshot.** A delivery attempt lives at most 40 minutes, while posts are retained 7 days (`post.retention_days = 7`), so the post is guaranteed to exist at delivery time; the worker does a primary-key `GetByID` and reads the body then. This keeps attempt rows narrow (~80 bytes) and avoids a snapshot-consistency problem that does not exist at this timescale.

## `delivery_history` (cold archive — 7-day user-facing record)

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

**`user_id` is `ON DELETE SET NULL`, not CASCADE.** A user's 7-day history can be large (millions of rows under the worst-case load). `ON DELETE CASCADE` would delete all of them inside the `DELETE FROM users` transaction, producing a massive dead-tuple burst that stalls the table. `SET NULL` instead preserves each history row with a null `user_id`; the UI renders "用户已注销" when `user_id` is null, exactly as it renders "原 post 已删除" / "投递渠道已删除" for the null `post_id`/`channel_id`. This makes `user_id` nullable (`*int`), consistent with the already-nullable `post_id`/`channel_id`.

**CASCADE on attempts vs SET NULL on history — why the difference.** An attempt row lives ≤40 min, so the cascade from deleting a user (or post/channel) touches at most the user's currently-in-flight attempts — a small, bounded set. A history row lives 7 days, so the same cascade would touch the user's entire delivery record — unbounded and large. The two tables therefore take different FK actions for the _same logical column_: bounded lifetime → CASCADE (cheap, semantically clean); unbounded lifetime → SET NULL (no lock storm, preserves audit).

**Strictly normalized — no snapshot columns, no attempt count, no delivered_at.** Titles, channel names, and usernames are JOINed at read time; `delivery_history` records the outcome (`status`), the failure detail (`last_error` + `error_category`), and when delivery was initiated (`created_at`). A user's history page is paginated (20 rows), and the PK joins are sub-millisecond — there is no performance justification for denormalization. The canonical read:

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

`HistoryRow` (`delivery.go`) is the GORM read projection of this join; its nullable pointer fields reflect `ON DELETE SET NULL` — a nil field means the referenced row was deleted.

**`error_category`.** Terminal failures carry the classified send-failure category (see [`delivery-retry.md`](./delivery-retry.md)); `delivered` and `expired` rows store the empty string, as do rows written before the column existed. The category drives the admin history filter (`HistoryFilter.ErrorCategory`) and the `delivery_failed` metric label.

**The history read surface.** `AttemptRepository` (internal/domain/delivery/repository.go) exposes the query set the UI needs, all over this table: `ListHistory`/`CountHistory` (paginated user or admin history with user/channel/status/error-category filters), `LatestPerChannel` (per-channel delivery-health overview on the channel list), `DailyStats`/`DailyStatsAll` (per-day outcome counts for the trend charts), `TodayCounts` (today's delivered/failed plus in-flight pending), `CountSince` (admin week-over-week delta), and `LockedChannels` (channels whose 24 h window shows all failures or a >50% failure rate, for the admin "需要关注" card).

**Storage.** ~70 million rows over 7 days at ~60 bytes/row ≈ **4.2 GB**, well within the 40 GB disk alongside posts' ~11 GB.

## Index design

The claim query is `WHERE status = 0 AND next_at <= ? ORDER BY next_at`. GORM cannot express a partial index in a struct tag, so the indexes are declared in the migration SQL.

**`delivery_attempts`** — partial index (protects HOT updates):

```sql
CREATE INDEX idx_da_pending
    ON delivery_attempts (next_at)
    WHERE status = 0;
```

The hot table's workload is `status`-transition UPDATEs (`pending` → terminal). A partial index on only the `pending` rows keeps the index tiny and cooperates with Postgres HOT (Heap-Only Tuple) updates: once a row leaves `pending`, its index entry is dropped and later UPDATEs never touch this index. HOT requires that an update not modify any column referenced by the table's indexes; the index key is only `next_at`, and while a status-transition UPDATE does modify `next_at` (the reservation bump on claim), the row _leaves the partial index's predicate_ (`status=0`) at the same transition, so it is removed from the index rather than re-indexed in place. (Grounded in the Postgres `CREATE TABLE` / indices / storage docs on fillfactor, partial indexes, and HOT.)

**`delivery_history`** — two workload shapes that one index cannot serve together:

```sql
CREATE INDEX idx_dh_user_channel_created ON delivery_history (user_id, channel_id, created_at DESC);
CREATE INDEX idx_dh_created              ON delivery_history (created_at DESC);
```

- `idx_dh_user_channel_created` — `user_id` leads as the always-present equality predicate, `channel_id` follows as the optional equality predicate, and `created_at DESC` trails to support the `ORDER BY` when both equalities are present.
- `idx_dh_created` — serves the admin "all history" view (`ORDER BY created_at DESC` with **no** `user_id` predicate). The `user_id`-leading composite cannot serve this query (no leftmost equality), so without this index the admin page would degrade to a full-table scan + sort on the ~70-million-row cold table.
- `idx_dh_error_category` (`000007`) — supports the admin error-category filter over `delivery_history`.

## Table tuning

Postgres-only `WITH (...)` reloptions cannot go in the GORM struct, so they are declared in the same migration SQL:

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

- `fillfactor = 90` on the hot table reserves page space so status-transition UPDATEs land updated tuples on the same page → HOT likely.
- `fillfactor = 100` on the append-only history table (no updates) → full packing, minimal storage.
- Aggressive autovacuum on the high-UPDATE attempts table reclaims dead tuples promptly.

## Retention

`delivery_history` rows are swept past the 7-day retention window (`[delivery] history_retention`, default `168h`) by a batched DELETE in the subquery-`LIMIT` form (`DELETE ... WHERE id IN (SELECT id ... WHERE created_at < $cutoff ORDER BY created_at LIMIT $batch)`), looped until no rows remain. It runs from the cron-invoked `prune-delivery-history` CLI subcommand (`cmd/prune_delivery_history.go`), the same pattern as `prune-expired-posts`. History rows are append-only, so this DELETE never contends with updates. There is no centralized sweep of `delivery_attempts` — see [`delivery-scheduler.md`](./delivery-scheduler.md) for the distributed cleanup.
