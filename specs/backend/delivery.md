# Delivery (Persistent Best-Effort Retry Queue)

This document specifies the message-delivery subsystem: how posts are pushed to configured channels (Feishu today, extensible tomorrow) with persistence, bounded retry, and crash recovery. It is the authoritative reference for the data model, retry strategy, concurrency model, and the rejected alternatives recorded with their rationale.

> **Sources.** Technical claims are verified against the source repos at `~/Workspace/contexts/`, checked out at the versions markpost uses: GORM `v1.31.1` (`schema/field.go`, `migrator/migrator.go`), SQLite `3.50.4` (trunk `3.54.0`; grammar/semantics identical — `src/parse.y`, `doc/`), Postgres `REL_17_STABLE` (`doc/src/sgml/`), MySQL `mysql-8.0.46` (`mysql-server` 8.0 — the production line; 8.0 is LTS, 9.x is innovation-track and not used in production — `sql/sql_yacc.yy`, `storage/innobase/`, `sql/field.cc`), `pond v2` (`github.com/alitto/pond/v2`), `ants v2` (`github.com/panjf2000/ants/v2`). The existing markpost code is cited as `backend/...` with line numbers.

The delivery feature was previously an in-process fire-and-forget mechanism (a buffered channel drained by a single goroutine). This spec closes that gap by upgrading it to a **persistent best-effort** delivery model: a delivery attempt that survives process restarts, retries with backoff, and reaches a terminal state within a bounded time wall.

## Scope and Product Semantics

markpost's delivery sends a notification (an interactive Feishu card) to each of a post author's configured delivery channels when a post is created. The keyword filter expression ([`keyword-filter.md`](./keyword-filter.md)) decides, per channel, whether a given post should be pushed.

- **Trigger:** post creation only (`CreatePost`, `internal/service/post/post.go:140`). There is no update-triggered or deletion-triggered delivery; posts are immutable ([`performance-optimization.md`](./performance-optimization.md) Decision 10).
- **Target:** Feishu webhook URLs today (`delivery.ChannelKindFeishu`). The `switch channel.Kind` in the delivery path leaves room for more kinds without a schema change.
- **Delivery is best-effort, not guaranteed.** The product contract is "try hard within a bounded window," not "deliver exactly once." A notification that cannot reach Feishu after the retry sequence is exhausted is recorded as failed and shown to the user; it is not retried forever.
- **Message latency is acceptable.** Seconds-to-minutes delay between post creation and notification is fine. This permits persistence + scheduled retry rather than synchronous send.
- **Delivery is explicitly out of scope for the performance-optimization pass** for the _read_ path (`performance-optimization.md:14`). This spec concerns only the write-path delivery subsystem.

## Capacity Envelope (the SaaS reference instance)

The hardware envelope is a 2-core / 2 GB / 3 Mbps VPS (`performance-optimization.md:20-28`). Under the worst-case load model — 10 000 users all saturating the L2 write limit (1000 posts/day each) — the sustained delivery ceiling is ~116 posts/s (10 000 × 1000 ÷ 86 400).

Two conclusions drove the design:

1. **CPU is not the bottleneck.** Delivery is I/O-bound (goroutines blocked on Feishu HTTP responses consume near-zero CPU). At 116 jobs/s the entire delivery subsystem (filter + DB writes + HTTP fan-out) costs under 7% of two cores. The existing filter benchmarks (`internal/service/delivery/filter/filter_bench_test.go`) show compile+match at ~1.6 µs/op for a 256-byte title; even with 10 channels per post this is sub-percent CPU.
2. **The single-goroutine fire-and-forget model cannot carry the load.** The previous dispatcher drained its queue with **one** goroutine issuing synchronous HTTP calls (default 5 s timeout). At a 300 ms Feishu response its throughput was ~3 jobs/s — two orders of magnitude below the 116 jobs/s ceiling. A 256-deep buffer fills in under a second, after which 97% of deliveries are silently dropped. This is why concurrency, not persistence, was the first problem to solve.

## Architecture: Three Layers, Three Responsibilities

```
CreatePost (synchronous, in the post-create transaction)
   │  keyword-filter each enabled channel against the post title
   └─ INSERT delivery_attempts (status=pending) ──────────────── Layer 1: Persistence (DB)
                                                                  survives crash

Scheduler (one goroutine, 1 s ticker)
   ├─ expire wall: pending past the wall → expired + archive (batched per tick)
   └─ claim due:   atomic UPDATE...WHERE id IN (SELECT...) RETURNING ─► pond ── Layer 2: Scheduling (Go ticker)
                   (PG/MySQL: + FOR UPDATE SKIP LOCKED; SQLite: none, MaxOpenConns(1) serializes)

pond worker pool (32 workers) ────────────────────────────────── Layer 3: Concurrency (pond v2)
   └─ GET post by id (PK lookup, post always alive)
      └─ send Feishu card
         ├─ success   → archive to history + delete attempt
         └─ failure   → attempts++, next_at = now + backoff[n]
                        attempts > len(seq) → failed → archive + delete
```

Each layer uses the right tool:

| Layer          | Responsibility                                                       | Mechanism                                                                                                                 | Why                                                                                                                              |
| -------------- | -------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| 1. Persistence | survive process restart; do not lose pending work                    | `delivery_attempts` table (Postgres / MySQL / SQLite)                                                                     | A worker-pool library is an in-memory concurrency primitive; persistence is the database's job, not the pool's.                  |
| 2. Scheduling  | scan due tasks; claim without double-delivery; apply the expiry wall | Go `time.Ticker` (1 s) + atomic claim (`FOR UPDATE SKIP LOCKED` on Postgres/MySQL; single-writer serialization on SQLite) | A simple polling scheduler is easy to reason about and needs no external broker.                                                 |
| 3. Concurrency | issue up to N Feishu HTTP calls in parallel, bounded                 | pond v2 worker pool                                                                                                       | Bounded concurrency controls outbound rate (Feishu QPS limits) and bandwidth (3 Mbps cap); unbounded goroutines would risk both. |

### Why pond v2 for the concurrency layer

`github.com/alitto/pond/v2` (zero external dependencies — `go.sum` is empty) provides a bounded pool with first-class `context` support, a built-in task queue (`WithQueueSize`), non-blocking submit (`WithNonBlocking` → `ErrQueueFull` on full queue), graceful drain (`StopAndWait`), default panic recovery, and built-in metrics (`RunningWorkers`, `DroppedTasks`). These map one-to-one onto the delivery requirements.

The alternative, `panjf2000/ants` v2, has **no built-in task queue** (it is a goroutine-recycling pool, not a queue+worker model), so adopting it would require keeping the existing buffered channel as a separate queuing layer — more moving parts for no gain. ants also carries a `golang.org/x/sync` runtime dependency. pond's built-in queue replaces markpost's hand-rolled channel+goroutine dispatcher directly.

## Retry Strategy: Hardcoded Fixed Sequence + Auto-Computed Wall

### The backoff sequence

Retry intervals are a **hardcoded fixed sequence** in source, not configurable:

```go
var backoffSequence = [...]time.Duration{
    1 * time.Minute,
    5 * time.Minute,
    10 * time.Minute,
    20 * time.Minute,
}
```

The sequence is the list of delays applied **after each failed attempt**. The first delivery attempt happens immediately on claim (no leading wait); each subsequent attempt waits `backoff[attempts-1]` before retrying.

Timeline (worst case, all attempts fail):

```
t=0     create attempt. scheduler claims, first delivery attempt.
        fail → attempts=1, wait backoff[0]=1m
t=1m    retry 1. fail → attempts=2, wait backoff[1]=5m
t=6m    retry 2. fail → attempts=3, wait backoff[2]=10m
t=16m   retry 3. fail → attempts=4, wait backoff[3]=20m
t=36m   retry 4. fail → attempts=5 > len(seq)=4 → FAILED (sequence exhausted)
```

So the sequence yields **1 immediate attempt + 4 retries = up to 5 delivery attempts**, exhausting at t=36m.

**Why hardcoded, not configurable.** markpost ships exactly one delivery channel kind today (Feishu). There is no per-channel retry requirement to satisfy, and exposing `backoff_sequence` to operators now would create a tuning surface with no consumer — the very definition of premature configurability. The sequence lives in `internal/service/delivery/backoff.go` and changes via a code change + release (which already rotates the process and clears in-flight state). If a second channel kind with different retry semantics is added later, configurability can be introduced at that point; until then it is a constant (see Decision 4).

### The expiry wall (auto-computed from the sequence)

The hard time wall is **derived from the hardcoded sequence** at process init, not configured:

```
wall = round_up_to_10min( sum(backoffSequence) )
```

For the sequence above: sum(1+5+10+20) = 36 min → round up to **40 min**. The 4-minute margin (40−36) ensures the last retry (t=36m) does not collide with the wall. `computeExpiryWall` is a package function exercised by a unit test; it is not operator-tunable.

- If a delivery is still `pending` when `created_at + wall` passes, the scheduler transitions it to **expired**.
- An empty sequence would disable retry (first failure → `failed`, wall = 0 and does not participate) — the fire-and-forget degenerate mode. This is retained as a code-level switch, not a config flag.

### Terminal states

| Status      | int8 | Meaning                                                | When                                    |
| ----------- | ---- | ------------------------------------------------------ | --------------------------------------- |
| `pending`   | 0    | awaiting or mid-delivery                               | initial; in-flight                      |
| `delivered` | 1    | a Feishu send succeeded                                | any attempt succeeds                    |
| `failed`    | 2    | the retry sequence was exhausted without success       | `attempts > len(backoffSequence)`       |
| `expired`   | 3    | the time wall passed before the sequence was exhausted | `pending` and `created_at + wall < now` |

`failed` and `expired` are distinct so the user/history can tell "we tried everything" apart from "we ran out of time." With the sequence above, `failed` fires at t=36m and `expired` almost never fires (the sequence exhausts before the 40m wall). `expired` becomes relevant only when the scheduler is stalled. (The numeric mapping is fixed by the `Status` enum — see Data Model; ordering is append-only forever.)

## Data Model

Two tables, hot and cold, separated by access pattern and lifecycle. The design follows database normalization strictly: **no redundant columns unless required as a query key for performance.** Foreign keys enforce referential integrity. The tables are declared as **GORM models** so they migrate identically across Postgres, MySQL, and SQLite via `AutoMigrate` (verified against GORM `v1.31.1`); the only dialect-specific SQL is the indexes and the Postgres storage options, applied in a gated post-migrate step (see _Per-dialect tuning_).

### The `Status` enum (shared by both tables)

```go
type Status int8

const (
    StatusPending   Status = 0 // default; "due" / "in-flight"
    StatusDelivered Status = 1 // terminal — a send succeeded
    StatusFailed    Status = 2 // terminal — sequence exhausted
    StatusExpired   Status = 3 // terminal — wall passed
)
```

**Integer enum, not a string enum.** A `type Status int8` (rather than the `type Role string` pattern used by `user.User`) is chosen deliberately to satisfy the "most compact, no strings, portable across 3 dialects" requirement:

- **Most compact.** GORM resolves the column type from the Go `reflect.Kind` and, for integer kinds, from the auto-computed `Size` (`schema/field.go:233-341`). `int8` → `Size=8`. Each driver then maps by size (`schema/field.go:330-341` is where the size is set; the driver's `DataTypeOf` consumes it):
  - **MySQL** → `tinyint` (1 byte) — the only dialect where `int8` actually saves space vs wider ints.
  - **Postgres** → `smallint` (2 bytes) — `int8` and `int16` map to the _same_ `smallint` because Postgres has no 1-byte integer type; 2 bytes is its floor.
  - **SQLite** → `integer` (value-width: a small value occupies 1–2 bytes regardless of the declared width).
- **Portable with no `type:` tag.** This is the critical point. GORM emits a `type:` tag value **verbatim on every dialect** (`schema/field.go:320-328`, the `default` branch stores the value as-is, not lowercased) and each driver's `DataTypeOf` `default` arm returns it literally. So `type:tinyint` would be sent to Postgres (which has no such type → DDL error) and `type:int2` to MySQL (no such type → DDL error). The bare `int8` form avoids this entirely: it relies on the size-based driver logic that all three drivers implement.
- **`StatusPending = 0`** so the database default (`default:0`) lands on the pending state; no literal is needed in the column default.
- **Tradeoffs (accepted):** values are stored as `0/1/2/3`, not human-readable strings — DB inspection shows numbers, and the mapping lives in code. The `iota` order must be **append-only forever**: inserting a state in the middle would renumber every later state and silently corrupt existing rows. Terminal states are never re-added, so appending (e.g. a future `StatusCanceled = 4`) is always safe.

### Table: `delivery_attempts` (hot queue — short-lived)

```go
// TableName returns the database table name for Attempt.
func (Attempt) TableName() string { return "delivery_attempts" }

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

    User    user.User            `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
    Post    post.Post            `json:"-" gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
    Channel delivery.Channel     `json:"-" gorm:"foreignKey:ChannelID;constraint:OnDelete:CASCADE"`
}
```

The struct follows the existing codebase conventions exactly: value-receiver `TableName()` (as `delivery.Channel` does at `delivery.go:26`), `int64` PK with `autoIncrement` (the `BIGSERIAL`/`BIGINT AUTO_INCREMENT` intent), `gorm:"not null;column:..."` FK columns, and explicit `foreignKey` + `constraint:OnDelete:CASCADE` associations (the same shape `post.Post` and `delivery.Channel` use). No `type:` tag appears on any column except `last_error` (which is `text` on all three dialects — verified portable).

**Lifecycle.** A row lives only while delivery is in progress — at most `wall` (40 min default). On any terminal state the row is **archived to `delivery_history` and deleted in the same transaction**. Steady-state row count is therefore bounded by the wall window (~28 万 rows at 116/s × 2400s ≈ 22 MB).

**ON DELETE CASCADE on posts/channels/users.** A delivery attempt for a deleted post or channel is meaningless (the Feishu card links to a dead post), so cascading the delete to its attempts is the correct semantics. There is no need to keep attempting delivery of a gone post. CASCADE is safe here _because_ an attempt row lives ≤40 min — the cascade always deletes a small, bounded set (see _history_ for the contrasting decision).

**`user_id` denormalization (the only one).** `user_id` is technically derivable via `post_id → posts.user_id`, but it is retained here as a query key: the scheduler and history queries filter by user, and avoiding a join on the hot path is a deliberate performance trade-off. Every other column is non-redundant.

**Why not store the post body / title here.** The previous design snapshot-stored a 200-char body preview in each attempt row. That is unnecessary: a delivery attempt lives at most 40 minutes, while posts are retained 7 days (`config: post.retention_days = 7`). At delivery time the post is guaranteed to exist, so the worker does a primary-key `GetByID` and reads the body then. This keeps attempt rows narrow (~80 bytes) and avoids a snapshot-consistency problem that does not exist at this timescale.

### Table: `delivery_history` (cold archive — 7-day user-facing record)

```go
// TableName returns the database table name for History.
func (History) TableName() string { return "delivery_history" }

type History struct {
    ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
    UserID    *int      `json:"user_id" gorm:"column:user_id;index"`         // nullable; ON DELETE SET NULL
    PostID    *int      `json:"post_id" gorm:"column:post_id;index"`         // nullable; ON DELETE SET NULL
    ChannelID *int      `json:"channel_id" gorm:"column:channel_id;index"`   // nullable; ON DELETE SET NULL
    Status    Status    `json:"status" gorm:"not null"`                       // delivered | failed | expired
    LastError string    `json:"last_error" gorm:"not null;type:text;default:''"`
    CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`

    User    *user.User        `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL"`
    Post    *post.Post        `json:"-" gorm:"foreignKey:PostID;constraint:OnDelete:SET NULL"`
    Channel *delivery.Channel `json:"-" gorm:"foreignKey:ChannelID;constraint:OnDelete:SET NULL"`
}
```

**`user_id` is `ON DELETE SET NULL`, not CASCADE (Review #5).** A user's 7-day history can be large (millions of rows under the worst-case load). `ON DELETE CASCADE` would delete all of them inside the `DELETE FROM users` transaction — holding locks on a huge row set (MySQL/InnoDB) or producing a massive dead-tuple burst that stalls the table (Postgres). `SET NULL` instead preserves each history row with a null `user_id`; the UI renders "用户已注销" when `user_id` is null, exactly as it already renders "原 post 已删除" / "投递渠道已删除" for the null `post_id`/`channel_id`. This makes `user_id` nullable (`*int`), consistent with the already-nullable `post_id`/`channel_id`.

**CASCADE on attempts vs SET NULL on history — why the difference.** An attempt row lives ≤40 min, so the cascade from deleting a user (or post/channel) touches at most the user's currently-in-flight attempts — a small, bounded set. A history row lives 7 days, so the same cascade would touch the user's entire delivery record — unbounded and large. The two tables therefore take different FK actions for the _same logical column_: bounded lifetime → CASCADE (cheap, semantically clean); unbounded lifetime → SET NULL (no lock storm, preserves audit). See Decision 6.

**Strictly normalized — no snapshot columns.** The earlier design stored `post_title` and `channel_name` snapshots here. That violated the project's normalization requirement. Instead, history is queried with `LEFT JOIN` to `posts` and `delivery_channels` to fetch title/name at read time. A user's history page is paginated (20 rows), and two PK joins are sub-millisecond — there is no performance justification for denormalization.

```sql
SELECT h.id, h.status, h.last_error, h.created_at,
       p.title AS post_title, p.qid AS post_qid,
       c.name AS channel_name
FROM delivery_history h
LEFT JOIN posts p ON p.id = h.post_id
LEFT JOIN delivery_channels c ON c.id = h.channel_id
WHERE h.user_id = $1            -- NULL user_id rows are excluded from a user's own page;
                                -- an admin "all history" view omits the user filter
  AND h.channel_id = $2         -- optional; omitted when the user views all channels
ORDER BY h.created_at DESC
LIMIT 20 OFFSET $3;
```

**Dropped columns (compared to an earlier draft) and why:**

- `channel_id` retained as FK and as a query key (the per-channel history filter `WHERE channel_id = ?`), but **no `channel_name` snapshot** — JOIN at read time (normalization).
- `attempts` (retry count) — dropped; users care about the outcome, not how many tries it took.
- `delivered_at` — dropped; `created_at` already records when delivery was initiated, and `status` records the outcome.

**Storage.** ~7000 万 rows over 7 days at ~60 bytes/row ≈ **4.2 GB**, well within the 40 GB disk alongside posts' ~11 GB.

### Per-dialect index design

The claim query is `WHERE status = <pending> AND next_at <= ? ORDER BY next_at`. The _optimal_ index for it differs by dialect, and GORM cannot express a partial index or a per-dialect index in a struct tag — so these are created in a gated post-migrate step in `internal/infra/db.go` (mirroring the existing `migratePostBodyCompressionLZ4` pattern at `db.go:126-130`), branched on `cfg.DB.Driver`. Status is the `int8` enum, so `pending = 0`.

**Postgres** — partial index (protects HOT updates):

```sql
CREATE INDEX IF NOT EXISTS idx_da_pending
    ON delivery_attempts (next_at)
    WHERE status = 0;
```

_Rationale._ The hot table's workload is `status`-transition UPDATEs (`pending` → terminal). A partial index on only the `pending` rows keeps the index tiny and, crucially, cooperates with Postgres HOT (Heap-Only Tuple) updates. Per the Postgres docs: "in heavily updated tables smaller fillfactors are appropriate" (`doc/src/sgml/ref/create_table.sgml`), and the "unbilled orders" example notes that a partial index "does not need to be updated in all cases" (`doc/src/sgml/indices.sgml`) — once a row leaves `pending`, its index entry is dropped and later UPDATEs never touch this index. HOT itself requires that the update "does not modify any columns referenced by the table's indexes" (`doc/src/sgml/storage.sgml`); the index key is only `next_at`, and while a status-transition UPDATE does modify `next_at` (the reservation bump on claim), the row _leaves the partial index's predicate_ (`status=0`) at the same transition, so it is removed from the index rather than re-indexed in place. This is the same design principle the prior Postgres-only spec relied on; the change here is only that the literal is `0` (the enum value), not the string `'pending'`.

**SQLite** — same partial index (size optimization):

```sql
CREATE INDEX IF NOT EXISTS idx_da_pending
    ON delivery_attempts (next_at)
    WHERE status = 0;
```

_Rationale._ SQLite supports partial indexes since 3.8.0 (verified: the grammar production at `~/Workspace/contexts/sqlite/src/parse.y:1622-1630` includes the trailing `where_opt`, and `doc/partialindex.html` documents it). The driver in use is `mattn/go-sqlite3 v1.14.32` (SQLite 3.50.4 ≫ 3.8.0). SQLite has no HOT analog, so the only benefit here is keeping the index small (only `pending` rows indexed) — which still matters, since under `MaxOpenConns(1)` every claim scans this index serially.

**MySQL** — composite index (partial indexes are unsupported):

```sql
CREATE INDEX idx_da_status_next
    ON delivery_attempts (status, next_at);
```

_Rationale._ MySQL/InnoDB **does not support partial indexes** — `CREATE INDEX` has no `WHERE` grammar (verified against `mysql-8.0.46`: the `create_index_stmt` production at `mysql-server/sql/sql_yacc.yy:3637-3663` and the `index_option`/`common_index_option` enumeration at `:7957-7981` admit only `KEY_BLOCK_SIZE`/`COMMENT`/`visibility`/engine-attribute options, never a predicate). The portable substitute is a composite index with `status` as the leading column, which covers the claim query's `WHERE status=0 AND next_at <= ? ORDER BY next_at` exactly. This index is **load-bearing** on MySQL in a way the partial index is not on Postgres: a locking read sets a next-key lock on _every index record the cursor scans_ (`storage/innobase/row/row0sel.cc:5240-5243`, the `sel_set_rec_lock` call site inside `row_search_mvcc`), and without a usable index the optimizer falls back to a full clustered-index scan (`ha_innodb.cc:10815-10831` `rnd_init` forces the primary key via `change_active_index`), so the claim would take an exclusive next-key lock on every row + gap in the table. The composite index confines the scan — and thus the lock footprint — to the `status=0` range actually being claimed.

**History indexes (all dialects, identical):**

The history read path has two shapes that one index cannot serve together, so the history table carries two indexes, both created in the post-migrate step:

```sql
CREATE INDEX idx_dh_user_channel_created ON delivery_history (user_id, channel_id, created_at DESC);
CREATE INDEX idx_dh_created              ON delivery_history (created_at DESC);
```

- `idx_dh_user_channel_created` — the three-column composite covers both the user-scoped history query (`WHERE user_id = ?`) and the per-channel query (`WHERE user_id = ? AND channel_id = ?`) via the leftmost-prefix rule. `user_id` leads because it is the always-present equality predicate; `channel_id` follows as the optional equality predicate; `created_at DESC` trails to support the `ORDER BY`. A bare `(user_id, created_at)` predecessor (`idx_dh_user_created`) is fully covered by the leftmost prefix of this composite and is dropped as redundant.
- `idx_dh_created` — serves the admin "all history" view (`ORDER BY created_at DESC` with **no** `user_id` predicate). The `user_id`-leading composite cannot serve this query (no leftmost equality), so without this index the admin page would degrade to a full-table scan + sort on the ~7000 万-row cold table. A plain single-column index on the sort key is the fix.

### Per-dialect table tuning

The Postgres storage options cannot go in the GORM struct (they are Postgres-only `WITH (...)` reloptions) and have **no direct equivalent on MySQL or SQLite**. They are applied dialect-specifically:

**Postgres** (post-migrate `ALTER TABLE ... SET (...)`, gated on `driver == "postgresql"`):

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

- `fillfactor = 90` on the hot table reserves page space so status-transition UPDATEs land updated tuples on the same page → HOT likely (`doc/src/sgml/ref/create_table.sgml`).
- `fillfactor = 100` on the append-only history table (no updates) → full packing, minimal storage.
- Aggressive autovacuum on the high-UPDATE attempts table reclaims dead tuples promptly.

**MySQL/InnoDB** — _no per-table fillfactor, no autovacuum analog_ (verified against `mysql-8.0.46`):

- `innodb_fill_factor` is a **global** system variable (declared `MYSQL_SYSVAR_LONG(fill_factor, ddl::fill_factor, PLUGIN_VAR_RQCMDARG, ...)` at `storage/innobase/handler/ha_innodb.cc:22469-22471`; scope is GLOBAL-only because the flag lacks `PLUGIN_VAR_THDLOCAL`; backing var `ddl::fill_factor` defined `btr0load.cc:44`) and applies only at sorted/bulk index-build time (it is read solely in `storage/innobase/btr/btr0load.cc:398-402,579`, never in the row-insert/page-split path). It is not a per-table reloption and not a `fillfactor` substitute.
- `KEY_BLOCK_SIZE` controls **compressed page size** and forces `ROW_FORMAT=COMPRESSED` (`ha_innodb.cc:6268-6280` `get_zip_shift_size`, plus `:6302` in `get_real_row_type`), not fill — not a `fillfactor` substitute either.
- There is no autovacuum: a grep for `autovacuum` across `storage/innobase/` returns zero hits. The closest analog is the **purge coordinator thread** (`storage/innobase/srv/srv0srv.cc:3077`), which reclaims **undo-log** history for committed transactions — not per-table dead-tuple vacuum the way Postgres autovacuum works. InnoDB's B-trees are maintained online, so there is nothing table-level to "vacuum."
- The MySQL "tuning" for the hot table is therefore the **composite index** (above) plus the **bounded 40-min row lifetime** (the wall keeps the table small) — not a storage option. Default `ROW_FORMAT=DYNAMIC` is used for both tables.

**SQLite** — no per-table tuning knobs. Under the existing production setup (`PRAGMA journal_mode=WAL` + `SetMaxOpenConns(1)`, `db.go:86,227`) all writes serialize through one connection, so there is no lock contention to tune away on this table. An optional operator step documented for self-hosters: `PRAGMA synchronous=NORMAL`, which is safe under WAL and speeds commits (the WAL guarantees integrity across crashes; `synchronous=FULL` is only needed to protect the WAL itself, and `NORMAL` is the documented WAL-appropriate setting per the SQLite docs `doc/pragma.html`).

## Delivery Flow

### 1. Enqueue (in `CreatePost`, synchronous)

```
CreatePost succeeds (post.ID known)
  channels := channelRepo.GetByUserID(userID)
  for each enabled channel:
      if keywordFilter(channel.Keywords).Match(post.Title):
          INSERT delivery_attempts
              (user_id, post_id, channel_id, status=StatusPending,
               attempts=0, next_at=now, created_at=now, updated_at=now)
```

The keyword filter runs **before** persistence, so only channels that actually match produce attempt rows. This keeps the queue free of no-op rows. `DeliveryJob.Title` (already part of the `Enqueue` contract, `post.go:150`) supplies the title for filtering.

> **Note on transactional coupling.** Enqueue is best-effort relative to the post-create response: if the INSERT fails, the post is still created (delivery is not the post's concern). The existing `DeliveryEnqueuer.Enqueue` contract returns no error and must remain non-fatal to `CreatePost`.

### 2. Scheduler (one goroutine, 1 s `time.Ticker`)

Two duties per tick. Both the expire sweep and the claim are **batched** (bounded batch size per tick) and use the **dialect-safe subquery-`LIMIT` form**, not bare `DELETE/UPDATE ... LIMIT`, because Postgres does not support `LIMIT` on UPDATE/DELETE while the `WHERE id IN (SELECT ... LIMIT N)` form is valid on all three dialects (verified: Postgres subquery semantics; MySQL subquery support; SQLite `parse.y` LIMIT-in-subselect). On MySQL/SQLite bare `... LIMIT N` would also work, but the subquery form is the one portable across the trio.

**Expire wall sweep** — transition pending rows past the wall to `expired`, batched, looped until a tick drains them:

```sql
-- repeated within one tick until RowsAffected == 0:
UPDATE delivery_attempts
   SET status = <expired>, updated_at = $now
 WHERE id IN (
     SELECT id FROM delivery_attempts
      WHERE status = <pending> AND created_at < $now - $wall_ms
      ORDER BY created_at
      LIMIT 64
 )
RETURNING *;
-- each returned row → archiveAndDelete(status=<expired>)
```

**Why batched, not one unbounded sweep.** A single unbounded `UPDATE ... RETURNING *` would, under a large `pending` backlog (e.g. a Feishu outage accumulating tens of thousands of stalled rows), lock that whole row range in one statement — a long-held lock on MySQL/InnoDB (a locking read takes a next-key lock on every scanned record at `storage/innobase/row/row0sel.cc:5240-5243`, and at the default REPEATABLE READ a DELETE/UPDATE does not release locks on non-matching scanned rows — `ha_innodb.cc` documents `unlock_row` as a no-op at higher isolation) and a large dead-tuple burst under the 1 s scheduler tick on Postgres. Batching to N=64 per statement bounds both the lock scope and the dead-tuple volume per statement, and the within-tick loop still drains the backlog promptly (the wall is minutes-scale; sub-second drain is not required). This is the direct fix for Review #7 — "if the scheduler archives every expired record every tick and there are many, does it destroy performance?" Answer: no, because the sweep is bounded per statement and only the expire predicate (`status=<pending> AND created_at < wall`) is matched, not the entire table.

**Claim due tasks** — atomically claim and reserve, dialect-conditional:

_Postgres / MySQL (row-locking dialects):_

```sql
UPDATE delivery_attempts SET next_at = $now + 35000   -- reserve past the request timeout
 WHERE id IN (
     SELECT id FROM delivery_attempts
      WHERE status = <pending> AND next_at <= $now
      ORDER BY next_at
      LIMIT 64
      FOR UPDATE SKIP LOCKED
 )
RETURNING *;
-- each returned row → pool.Go(execute)
```

_SQLite (no row-level locking):_

```sql
UPDATE delivery_attempts SET next_at = $now + 35000
 WHERE id IN (
     SELECT id FROM delivery_attempts
      WHERE status = <pending> AND next_at <= $now
      ORDER BY next_at
      LIMIT 64
 )
RETURNING *;
```

**Why the claim differs by dialect.** `SELECT ... FOR UPDATE SKIP LOCKED` is supported on Postgres (the docs describe it as the queue-like use case — "multiple consumers accessing a queue-like table," `doc/src/sgml/ref/select.sgml`) and on MySQL (verified against `mysql-8.0.46`: the `locked_row_action` production `SKIP_SYM LOCKED_SYM → Locked_row_action::SKIP` at `mysql-server/sql/sql_yacc.yy:10158-10161`, mapping to the `select_mode` `SELECT_SKIP_LOCKED` at `storage/innobase/include/lock0types.h`; the skip-instead-of-wait branch is `lock_rec_lock_slow` returning `DB_SKIP_LOCKED` at `storage/innobase/lock/lock0lock.cc:1826-1827`, acted on by `row_search_mvcc` with `goto next_rec` at `storage/innobase/row/row0sel.cc:5261-5264`). It is **not supported on SQLite at all**: the grammar at `~/Workspace/contexts/sqlite/src/parse.y:651-665` terminates a SELECT after `limit_opt` with no `for_update` production, and a grep for `FOR UPDATE` / `SKIP LOCKED` / `TK_FOR` across the SQLite `src/` tree returns zero matches — issuing it is a parse-time _syntax error_, not a runtime no-op. (The current spec's claim that "on SQLite the clause is a no-op" was therefore incorrect and is corrected here.)

The SQLite path drops the clause. This is safe _only because_ the production SQLite pool runs with `SetMaxOpenConns(1)` (`db.go:86`), so every write — including the scheduler's claim — serializes through one connection: two claim operations cannot run concurrently in-process, so there is no double-claim to prevent. See _SQLite-mode concurrency_ below for the full argument. The claim body itself (atomic `UPDATE ... WHERE id IN (SELECT ... LIMIT) RETURNING *`) is identical across all three dialects; only the locking clause is conditional.

The claim advances `next_at` to `now + request_timeout + buffer` so the next 1-second tick does not re-select these rows while they are being delivered. This is the concurrency-correctness fix for the double-claim hazard: without it, a row whose delivery takes longer than 1 s would be re-claimed and re-delivered.

### 3. Worker execution (pond pool, 32 workers)

```
execute(attempt):
  post := posts.GetByID(attempt.post_id)          -- PK lookup; post guaranteed alive (< 40m < 7d retention)
  channel := channels.GetByIDAndUserID(attempt.channel_id, attempt.user_id)
  bodyPreview := truncate(post.Body, bodyPreviewChars)
  err := feishu.SendCard(channel.WebhookURL, post.Title, bodyPreview, ...)

  if err == nil:
      archiveAndDelete(attempt, status=StatusDelivered, last_error='')
  else:
      attempts := attempt.attempts + 1
      if attempts > len(backoffSequence):           -- sequence exhausted
          archiveAndDelete(attempt, status=StatusFailed, last_error=truncate(err, 200))
      else:
          backoff := backoffSequence[attempts-1]    -- [1m,5m,10m,20m][attempts-1]
          UPDATE delivery_attempts
          SET attempts=attempts, next_at=now+backoff,
              last_error=truncate(err, 200), updated_at=now
          WHERE id=attempt.id
```

**`archiveAndDelete` (single transaction):**

```
BEGIN
  INSERT delivery_history
    (user_id, post_id, channel_id, status, last_error, created_at)
    VALUES (attempt.user_id, attempt.post_id, attempt.channel_id, status, last_error, attempt.created_at)
  DELETE FROM delivery_attempts WHERE id = attempt.id
COMMIT
```

The terminal-state archive + delete is one transaction so the history record and the attempt removal are atomic.

### Distributed cleanup of `delivery_attempts`

There is **no centralized batch DELETE** that sweeps terminal attempts. Each attempt row is deleted by its own delivery operation at the moment it reaches a terminal state (`archiveAndDelete`, called from the worker on `delivered`/`failed`, and from the scheduler on `expired`). Cleanup is therefore naturally distributed across all in-flight delivery operations and never produces a large dead-tuple burst. This is why the `delivery_attempts` table stays small and why its autovacuum load stays light.

> `delivery_history` retention (7 days) is swept by a periodic lightweight batched DELETE, run from a cron-invoked command in the style of `cmd/prune_expired_posts.go`. It uses the **dialect-safe subquery-`LIMIT` form**, not bare `DELETE ... LIMIT`:
>
> ```sql
> DELETE FROM delivery_history
>  WHERE id IN (
>      SELECT id FROM delivery_history
>       WHERE created_at < $now - $retention
>       ORDER BY created_at
>       LIMIT $batchSize
>  );
> -- looped until RowsAffected == 0
> ```
>
> Bare `DELETE ... LIMIT N` is valid on MySQL (the single-table `delete_stmt` production accepts `opt_simple_limit` at `mysql-server/sql/sql_yacc.yy:13490` within the `:13480-13493` single-table alternative; multi-table DELETE at `:13494-13514` deliberately omits it) and on SQLite (the `mattn/go-sqlite3 v1.14.32` driver compiles with `-DSQLITE_ENABLE_UPDATE_DELETE_LIMIT`, verified at `sqlite3.go:22`), but is a **syntax error on Postgres** (no `LIMIT` on DELETE). The subquery form is portable across all three. History rows are append-only, so this DELETE does not contend with updates.

## Concurrency and Crash Recovery

**At-least-once delivery.** A worker may crash between a successful Feishu send and the `archiveAndDelete` commit. On restart, the row is still `pending` (the archive never committed), the scheduler re-claims it, and Feishu receives a duplicate card. This is the inherent cost of at-least-once; Feishu webhooks should tolerate occasional duplicates (the card content is idempotent — re-sending the same post is annoying, not harmful). Exactly-once would require an outbox/dedup table and is explicitly rejected (Decision 5).

**Crash recovery.** All pending state lives in `delivery_attempts`, not in process memory. A process restart re-runs the scheduler, which re-claims every `pending` row whose `next_at <= now`. The only delivery work lost on crash is an in-flight HTTP call that had not yet returned — and that row is re-claimed on the next tick.

**Double-claim prevention.** The claim query advances `next_at` past the request timeout, so a row being delivered is invisible to the next scheduler tick. If the worker dies mid-delivery, the reserved `next_at` elapses and the row becomes re-claimable — the retry resumes naturally.

**SQLite-mode concurrency.** SQLite deployments (homelab mode, and the dev/test path) need a separate concurrency argument, because SQLite's concurrency model is nothing like Postgres/MySQL's:

- WAL mode allows _many concurrent readers plus exactly one writer_ (verified: SQLite docs `doc/wal.html`, "There can only be a single writer"; the exclusive WRITER lock is acquired in `src/wal.c`). A second writer that cannot acquire the WRITER lock gets `SQLITE_BUSY` (not silent serialization) — unless a busy handler / `PRAGMA busy_timeout` is set, in which case it polls until the timeout then still returns `SQLITE_BUSY`.
- **But markpost's production SQLite pool pins `SetMaxOpenConns(1)`** (`db.go:86`). That routes every read _and_ write through a single `*sql` connection, so Go's `database/sql` pool hands the connection to one goroutine at a time and the rest block on the free-conn channel. In-process DB access is therefore **fully serialized**, regardless of WAL's reader/writer concurrency. This is stronger than WAL alone: even reads serialize. (The in-memory test DB at `db.go:145-148` sets WAL + FK but does _not_ pin `MaxOpenConns(1)`, so the test path is not the serialized configuration — tests rely on the pond pool's bounded workers plus the claim reservation, not on connection-level serialization.)
- **Implication for double-claim: there is none in-process on SQLite.** Two scheduler ticks or two workers cannot run a claim concurrently because they share the one connection; the second blocks until the first's transaction returns. This is exactly why the SQLite claim query (above) can omit `FOR UPDATE SKIP LOCKED`: there is no concurrent claimer to exclude.
- **What the pond pool still buys on SQLite.** The 32 workers do not parallelize DB writes (those serialize at the connection), but they _do_ parallelize the **outbound Feishu HTTP calls** — which is the actual I/O bottleneck the pool exists to address. A worker blocked on a 300 ms Feishu response holds no DB lock (it has already committed its claim UPDATE and returned the connection to the pool), so 32 workers can have 32 Feishu calls in flight while DB writes trickle through the single connection one at a time. The pool's value is preserved; only the DB-write fan-out is serial.
- WAL's reader/writer non-blocking property is effectively nullified by the single-connection pool, but this is acceptable: delivery write volume is tiny relative to the read path (which SQLite handles fine under WAL with `MaxOpenConns(1)` because reads still don't block the writer at the SQLite level, even if they queue at the connection level).

The design consequence: the claim/expire/archive SQL is written to be **dialect-correct everywhere**, but the _correctness argument_ for SQLite rests on `MaxOpenConns(1)` serialization, while for Postgres/MySQL it rests on `FOR UPDATE SKIP LOCKED` + the partial/composite index.

## Configuration

```toml
[delivery]
body_preview_chars = 200                 # Feishu card body preview length
request_timeout = "5s"                   # per Feishu HTTP call
workers = 32                             # pond pool size (concurrent Feishu sends)
queue_size = 1024                        # pond task queue depth
scan_interval = "1s"                     # scheduler tick
history_retention = "168h"               # 7 days; delivery_history prune threshold
```

The retry sequence and the expiry wall are **not** in this section: they are hardcoded in `internal/service/delivery/backoff.go` (`backoffSequence = [1m,5m,10m,20m]`, `expiryWall = computeExpiryWall(backoffSequence) = 40m`). There is exactly one delivery channel kind today, so per-channel retry tuning has no consumer; see Decision 4.

**Defaults rationale.**

- `workers = 32` covers the 116 jobs/s sustained ceiling at a 300 ms Feishu response (32/0.3 ≈ 106/s) with headroom from the queue for bursts.
- `queue_size = 1024` gives the pond pool burst headroom beyond the worker count (verified: pond v2's `WithQueueSize` decouples "workers busy" from "queue full," `pool.go:151`/`pooloptions.go:17-21`).
- `scan_interval = 1s` balances delivery latency (sub-second-to-second) against DB load (one indexed query per second is negligible).
- `history_retention = 168h` (7 days) matches `post.retention_days`; history outlives no post by more than the post's own lifetime, so the JOIN at read time always finds a live post within the retention window.

## Decision Record

Each decision is recorded with the rationale and the alternatives that were considered and rejected.

### 1. Persistent best-effort over fire-and-forget

The previous design was an in-process buffered channel drained by a single goroutine: fire-and-forget, no persistence, no retry. _Adopted:_ a persisted attempt table (Postgres / MySQL / SQLite) with bounded retry. _Rejected:_ keep fire-and-forget — it drops ~97% of deliveries at the 116 jobs/s ceiling (single-goroutine throughput ~3/s) and loses all pending work on restart. The product now requires "try hard within a bounded window," which fire-and-forget cannot meet.

### 2. Three-layer separation (DB / scheduler / pond)

Persistence is the database's job; concurrency is the pool's job; scheduling is a Go ticker's job. _Rejected:_ asking a worker-pool library to also persist — pond and ants are in-memory concurrency primitives with no persistence layer; conflating the two is a category error. _Rejected:_ an external message broker (Redis, RabbitMQ, Cloudflare Queues) — see Decision 7.

### 3. pond v2 over ants v2 for the concurrency layer

pond has a built-in task queue (`WithQueueSize`), which maps directly onto the existing dispatcher's buffered-channel model; ants has no queue, so adopting it would require retaining the hand-rolled channel as a separate layer. pond is also zero-dependency (`go.sum` empty) versus ants's `golang.org/x/sync`, and offers first-class `context`, default panic recovery (verified: pond v2 defaults `panicRecovery: true` at `pool.go:573`; a panicking task is converted to `ErrPanic` and does not crash the worker, `task.go:58-70`; disable via `WithoutPanicRecovery`), and a metrics surface that includes `RunningWorkers()`/`WaitingTasks()`/`DroppedTasks()` (verified: `pool.go:38-93` exposes a richer set). _Rejected:_ ants — more moving parts for no gain on this workload (verified: ants v2 hands tasks directly to a per-worker channel via `retrieveWorker`, `ants.go:511-548`, with no central queue; non-blocking submit yields `ErrPoolOverload`, not pond's `ErrQueueFull`). _Rejected:_ unbounded goroutines (one per delivery) — would violate Feishu's per-bot QPS limits and saturate the 3 Mbps egress; bounded concurrency is required.

### 4. Hardcoded fixed backoff sequence + auto-computed wall (not configurable)

The retry intervals are a **hardcoded** fixed sequence `[1m, 5m, 10m, 20m]` in `internal/service/delivery/backoff.go`, and the expiry wall is `round_up_to_10min(sum(sequence))` = 40 min, computed once at init. _Why hardcoded:_ there is one delivery channel kind (Feishu) and no per-channel retry requirement, so a config knob would be a tuning surface with no consumer — premature configurability. Changing the sequence is a code change + release, which already restarts the process (clearing in-flight state) and is the natural rollback boundary. _Rejected:_ exposing `backoff_sequence` / `expiry_wall` as config now — YAGNI; revisit if a second channel kind needs different retry semantics. _Rejected:_ pure exponential backoff (30s→1m→2m→4m→8m→16m) — late intervals become too sparse to catch medium-duration outages (a Feishu recovery at minute 20 falls in a 16-minute gap), and the last interval (16m) collides with a 30-minute wall, wasting the final wait. _Rejected:_ capped exponential (30s base, 5m cap, 8 retries) — workable, but the fixed sequence is simpler to reason about, and its total is deterministic, making the auto-wall exact. _Retained (code-level, not config):_ an empty sequence disables retry (first failure → `failed`, wall = 0) — the fire-and-forget degenerate switch.

### 5. At-least-once, not exactly-once

Delivery is at-least-once: a crash between send-success and archive-commit can produce a duplicate Feishu card. _Rejected:_ exactly-once — would require a transactional outbox or a dedup table keyed by an idempotency key, adding a write to the hot path for every delivery. Feishu cards are idempotent in content (re-sending the same post is benign), so the cost of occasional duplicates is lower than the cost of dedup machinery.

### 6. Strict normalization; history uses FK + ON DELETE SET NULL on _all_ FKs including `user_id`

`delivery_history` carries no snapshot columns; titles and channel names are JOINed at read time. All three foreign keys (`user_id`, `post_id`, `channel_id`) are nullable with `ON DELETE SET NULL`. _Rejected:_ snapshot columns (`post_title`, `channel_name`) — violates the project's normalization rule, and the read-time JOIN on a 20-row paginated history page is sub-millisecond, so there is no performance justification. _Rejected:_ `user_id ... ON DELETE CASCADE` on history — deleting a user would cascade-delete their entire 7-day history (potentially millions of rows) inside the `DELETE FROM users` transaction, holding a large lock set on MySQL/InnoDB and producing a massive dead-tuple burst on Postgres; `SET NULL` preserves each history row as an anonymous record instead (Review #5). _Note:_ `delivery_attempts.user_id` (and `post_id`/`channel_id`) keep `ON DELETE CASCADE` — an attempt row lives ≤40 min, so the cascade from a user/post/channel delete touches at most the user's currently-in-flight attempts, a small bounded set. The two tables take different FK actions for the _same logical column_ deliberately: bounded lifetime (attempts) → CASCADE (cheap, clean); unbounded lifetime (history) → SET NULL (no lock storm, preserves audit).

### 7. No external message broker (Cloudflare Queues / Redis / RabbitMQ)

The delivery queue is the Postgres `delivery_attempts` table, scheduled by a Go ticker, executed by a pond pool. _Rejected:_ Cloudflare Queues — evaluated in depth. Queues' **consumer side is Worker-only for push** (the docs: "a push-based consumer runs on Workers"); pulling from the VPS over HTTP would be strictly worse than the in-process channel it replaces. Using Queues for push-to-Feishu would require deploying and maintaining a Cloudflare Worker that reimplements the Feishu card logic — a second codebase, second deploy pipeline, and a cloud dependency, for no gain over a Postgres table + ticker. The free-tier 10 000 ops/day is ample, but the architecture mismatch is decisive. _Rejected:_ Redis/RabbitMQ — adds an always-on dependency and operational burden for a single-instance deployment whose write rate is ~0.12 posts/s mean; a Postgres table + ticker meets the need at zero additional infrastructure (`performance-optimization.md` Decision 28 rejects a second VPS / shared cache for the same reason).

### 8. Distributed cleanup of `delivery_attempts`; batched cleanup of `delivery_history` and of the expire sweep

Terminal attempts are archived-and-deleted by their own delivery operation (in `archiveAndDelete`), not by a centralized sweep. _Rejected:_ a periodic batch DELETE over terminal attempts — produces a large dead-tuple burst and vacuum pressure; distributing the delete across in-flight operations keeps each delete small. The scheduler's **expire-wall sweep is batched per tick** (`MarkExpired` runs `UPDATE ... WHERE id IN (SELECT ... LIMIT 64) RETURNING *`, looped within the tick until no rows match), not one unbounded `UPDATE ... RETURNING *` — a large `pending` backlog (a Feishu outage accumulating tens of thousands of stalled rows) would otherwise lock the whole matched range in one statement (MySQL/InnoDB record locks on every scanned index record) or burst dead tuples under the 1 s tick (Postgres); batching bounds both (Review #7). `delivery_history` retention (7 days) uses a lightweight batched DELETE from a cron command, in the portable subquery-`LIMIT` form (bare `DELETE ... LIMIT` is a Postgres syntax error), because history rows are append-only and never contend with updates.

### 9. No `status='running'` intermediate state

The state machine is `pending → delivered | failed | expired` — there is no `running`/`processing` state for "currently being delivered." In-flight deduplication is handled by the claim query advancing `next_at` past the request timeout, which makes in-flight rows invisible to the next tick. _Rejected:_ a `running` state — would require an extra transition on both claim (pending→running) and completion (running→terminal), doubling the UPDATE count and breaking the partial-index design (a `running` row would need its own index treatment). The `next_at` reservation achieves the same correctness with one state fewer.

## Implementation Plan

Organized by layer. Each item lists the verification approach inline.

### Source file structure

The delivery subsystem touches four layers, matching the existing codebase conventions (domain → infra → service → cmd). Files marked **(new)** are created; files marked **(extend)** already exist and are appended to.

```
backend/
  go.mod                                            # (extend) +gorm.io/driver/mysql (P0-item 0)
  internal/
    domain/delivery/
      delivery.go            (extend)  # +Status int8 enum; +Attempt struct; +History struct; +TableName()
      repository.go          (extend)  # +AttemptRepository interface (Create/ClaimDue/MarkFailed/MarkExpired/ArchiveAndDelete)
    infra/
      db.go                  (extend)  # +mysql driver branch (New switch); +Attempt/History in allModels;
                                        # +per-dialect index migration + Postgres storage-options migration
                                        #   (mirrors migratePostBodyCompressionLZ4 gating at db.go:126-130)
      delivery_attempt_repo.go (new)   # AttemptRepository impl; dialect-safe claim/expire/archive SQL
      testdb.go              (extend)  # +Attempt/History in testModels (mirror allModels)
    service/delivery/
      dispatcher.go          (rewrite) # pond v2 pool + 1s ticker scheduler; replaces the channel dispatcher
      backoff.go             (new)     # hardcoded backoffSequence + computeExpiryWall (not config)
    config/
      config.go              (extend)  # DeliveryConfig: +Workers/QueueSize/ScanInterval/HistoryRetention;
                                        #   remove dead RetryCount; backoff NOT in config
  cmd/
    prune_delivery_history.go (new)     # batched, dialect-safe history DELETE; cron-invoked;
                                        #   style of cmd/prune_expired_posts.go
    server/main.go          (extend)  # construct attemptRepo; NewDispatcher(...); Start(ctx)
```

The existing files referenced above (verified current state): `internal/domain/delivery/delivery.go:105-116` (`Channel`), `internal/domain/delivery/repository.go:6-14` (`Repository`), `internal/infra/db.go:24-30` (`allModels`), `internal/infra/db.go:65-81` (driver switch), `internal/service/delivery/dispatcher.go:1-56` (current channel dispatcher to be replaced), `internal/config/config.go:107-112` (`DeliveryConfig`), `cmd/prune_expired_posts.go` (style template).

### P0 — Data model

0. **Add the MySQL driver dependency** (`go.mod`, `internal/infra/db.go`): add `gorm.io/driver/mysql v1.6.x` and a `case "mysql":` branch in the `New(dsn)` driver switch (`db.go:65-81`), modeled on the existing `postgresql` branch (open + pool tuning). Register `delivery.Attempt`/`History` (item 1/2) in `allModels` so `AutoMigrate` covers all three dialects. _Note:_ adding a dependency crosses the `AGENTS.md` "Ask first: adding new dependencies" boundary — confirmed with the operator; MySQL is positioned as a self-hosted optional deployment target. _Verification:_ `go build ./...`; `db_test.go` `TestNew_UnsupportedDriver` is updated (it currently asserts `driver="mysql"` is rejected — `db_test.go:121`). Connection-pool tuning for MySQL mirrors Postgres (`SetMaxOpenConns(25)`, idle/lifetime), since MySQL is also a multi-connection server DB.

1. **Add `delivery.Status` enum + `delivery.Attempt` domain model** (`internal/domain/delivery/delivery.go`): the `Status int8` enum and `Attempt` struct exactly as specified in _Data Model_. `TableName()` returns `delivery_attempts`. Register `&delivery.Attempt{}` in `allModels` (`db.go:24`) **and** `testModels` (`testdb.go`). _Verification:_ unit — `NewTestDatabase` migrates without error on SQLite; the `delivery_attempts` table exists; a round-trip insert/read of an `Attempt` preserves the `Status` value; on Postgres/MySQL integration runs the column type is `smallint`/`tinyint` respectively (verified mapping).

2. **Add `delivery.History` domain model** (same file): the `History` struct with nullable `*int` FKs and `ON DELETE SET NULL` constraints, as specified. Register in `allModels` + `testModels`. _Verification:_ as above, plus a FK-cascade test — deleting a `user.User` with history rows nulls `history.user_id` (does _not_ delete the row); deleting a `post.Post` nulls `history.post_id`.

3. **Add per-dialect index + Postgres storage-options migration** (`internal/infra/db.go`, new method e.g. `migrateDeliveryIndexes`): gated on `cfg.DB.Driver`, creates the indexes specified in _Per-dialect index design_ (partial on postgres/sqlite, composite on mysql) and, on postgres only, the `ALTER TABLE ... SET (fillfactor=..., autovacuum=...)` reloptions. Use `CREATE INDEX IF NOT EXISTS` / an idempotency guard so re-runs are cheap. Call it from `New` after `AutoMigrate`, alongside `migratePostBodyCompressionLZ4` (`db.go:126-130`). _Verification:_ integration — on each dialect, `\d delivery_attempts` / `SHOW INDEX` confirms the expected index shape; a second startup issues no new DDL.

4. **Add `AttemptRepository`** (`internal/domain/delivery/repository.go` interface + `internal/infra/delivery_attempt_repo.go` impl): `Create(ctx, []*Attempt)`, `ClaimDue(ctx, now, limit) []*Attempt` (dialect-conditional — `FOR UPDATE SKIP LOCKED` on postgres/mysql, omitted on sqlite; subquery-`LIMIT` form on all), `MarkExpired(ctx, wall, batchSize) []*Attempt` (batched, looped by the caller), `MarkFailed(ctx, id, attempts, err, nextAt)`, `ArchiveAndDelete(ctx, attempt, status, lastError)` (single-tx INSERT history + DELETE attempt). _Verification:_ unit + integration — create attempts, claim them (assert no double-claim across two concurrent claim calls on postgres/mysql; on sqlite assert the single-connection serialization holds), mark failed with backoff (assert `next_at` advances by `backoff[attempts-1]`), mark expired (assert rows past the wall transition, in batches), archive (assert history row written and attempt row gone in the same tx).

### P0 — Scheduler + pool

5. **Add `backoff.go`** (`internal/service/delivery/backoff.go`): the hardcoded `backoffSequence` and `computeExpiryWall(seq) = round_up_10min(sum(seq))`. _Verification:_ unit — `computeExpiryWall([1m,5m,10m,20m]) == 40m`; `computeExpiryWall([]) == 0`.

6. **Rewrite `Dispatcher`** (`internal/service/delivery/dispatcher.go`) to the three-layer model: pond v2 pool + 1 s ticker scheduler + batched expire sweep + `archiveAndDelete` on terminal. Preserve the `post.DeliveryEnqueuer` interface (`Enqueue`). Add `Start(ctx)` (starts the ticker) and `Shutdown(ctx)` (drains via `pool.StopAndWait()`, verified pond v2 API). _Verification:_ unit — enqueue inserts pending rows; a fake clock advances due rows to claimed; success → delivered + archived; failure → backoff advances; sequence exhausted → failed; wall passed → expired (batched). _Race:_ `-race` with N concurrent enqueues. _Fuzz:_ fuzz QID/title to confirm no panic in key construction.

7. **Wire `main.go`** (`cmd/server/main.go`): construct `attemptRepo`, pass the new config to `NewDispatcher`, call `Start`. Add graceful shutdown (`Shutdown` on SIGTERM) — the current `Start(context.Background())` wiring (`main.go:171`) never cancels. _Verification:_ `go build ./...`.

### P0 — Config

8. **Extend `DeliveryConfig`** (`internal/config/config.go`): add `Workers`, `QueueSize`, `ScanInterval`, `HistoryRetention`. **Do not** add `BackoffSequence`/`ExpiryWall` — those are hardcoded (Decision 4). Remove the dead `RetryCount` field (`config.go:111`, never consumed). Set defaults (`workers=32`, `queue_size=1024`, `scan_interval=1s`, `history_retention=168h`). _Verification:_ unit — config loads with defaults; validation passes.

### P1 — History retention

9. **Add history prune command** (`cmd/prune_delivery_history.go`) in the style of `cmd/prune_expired_posts.go`: batched `DELETE FROM delivery_history WHERE id IN (SELECT id ... WHERE created_at < now - retention ORDER BY created_at LIMIT N)` (the portable subquery-`LIMIT` form, not bare `DELETE ... LIMIT`), looped until none remain. Invoked by external cron. _Verification:_ integration — insert old history rows, run the command, assert they are gone; recent rows untouched; the statement parses on all three dialects (a Postgres run confirms bare-`LIMIT` was correctly avoided).

### P2 — Observability (optional, not blocking)

10. **Expose pool metrics** via a `/api/v1/...` debug endpoint or structured logs: `pool.RunningWorkers()`, `pool.WaitingTasks()`, `pool.DroppedTasks()` (verified pond v2 methods), and a count of attempts in each status. _Verification:_ manual.

### Testing strategy

| Category    | Coverage                                                                                                                                                               | Approach                                                     |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------ |
| Unit        | enqueue→pending, claim→no-double-claim, backoff advance, batched wall expiry, terminal archive+delete, `computeExpiryWall`, FK SET NULL on user/post/channel delete    | `go test`; in-memory SQLite for repo; fake Feishu client     |
| Integration | `FOR UPDATE SKIP LOCKED` concurrency (Postgres/MySQL), dialect-specific index shape, history JOIN query, `SKIP LOCKED` _absence_ + single-conn serialization on SQLite | `go test` against real Postgres + MySQL + SQLite (CI matrix) |
| Race        | concurrent enqueue + scheduler + workers                                                                                                                               | `go test -race`                                              |
| Fuzz        | QID/title/key strings in enqueue path                                                                                                                                  | `go test -fuzz`                                              |
| Manual      | end-to-end Feishu delivery, prune command, per-dialect `\d`/`SHOW INDEX`                                                                                               | operator-run                                                 |

## What This Spec Explicitly Does Not Do

A consolidated list of rejections, each cross-referencing the Decision Record:

- **No fire-and-forget** — Decision 1. Persistence + retry is now required.
- **No external message broker** (Cloudflare Queues / Redis / RabbitMQ) — Decision 7. DB table + ticker + pond pool.
- **No exactly-once delivery** — Decision 5. At-least-once; duplicates tolerated.
- **No snapshot/denormalized columns in history** — Decision 6. Strict normalization; JOIN at read time.
- **No `running` intermediate state** — Decision 9. `next_at` reservation handles in-flight dedup.
- **No centralized batch cleanup of `delivery_attempts`** — Decision 8. Distributed across delivery operations; the expire sweep is batched per tick.
- **No per-delivery goroutine (unbounded)** — Decision 3. Bounded pond pool.
- **No configurable backoff sequence / expiry wall** — Decision 4. Hardcoded constants; one channel kind today, so no per-channel tuning consumer.
- **No MySQL native `ENUM` column** — the `Status` column is `int8` (→ `tinyint`/`smallint`/`integer`), not MySQL `ENUM(...)`. A native ENUM would be equally compact for ≤255 members (`sql/field.h:285-287` in `mysql-8.0.46`: `get_enum_pack_length` returns 1 byte `< 256`, 2 otherwise), but it is rejected for two source-verified reasons: GORM has no first-class ENUM tag (a plain Go `string` field becomes `varchar`/`longtext`, not `ENUM`), and even a hand-written `type:enum(...)` column is operationally fragile — `Field_enum::is_equal` (`mysql-server/sql/field.cc:8487-8531`) returns `IS_EQUAL_YES` (a pure-DD, INSTANT change) **only** for a value appended at the end that does not cross the 255→256 byte boundary (the `pack_length` check at `:8529`); any insert-in-middle, reorder, rename, or removal maps to `IS_EQUAL_NO` (`:8504-8518`), forcing a full table rebuild. The int8 form is portable across all three dialects and a new status is just a new constant — no schema change, no rewrite risk.
- **No bare `DELETE/UPDATE ... LIMIT N`** — the portable subquery-`LIMIT` form (`WHERE id IN (SELECT ... LIMIT N)`) is used everywhere batching appears, because bare `... LIMIT` on UPDATE/DELETE is a Postgres syntax error (and only conditionally compiled into SQLite, though the mattn driver enables it).
