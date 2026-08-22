# PostgreSQL 调优

[English](postgres-tuning.md) | 中文

markpost 所连接的单个 Postgres 实例的连接、存储与拓扑调优：Go 进程中的连接池边界、部署模板与迁移施加的服务器 GUC 与 TOAST 压缩，以及同级容器 + Unix 套接字拓扑。决策记录（GUC 取值、lz4 而非应用层压缩、Docker 而非裸金属）见[性能优化 MRFC](../../.agents/mrfcs/implemented/2026-07-09-read-path-performance-pass.zh.md)；DSN 格式见 [`dsn.zh.md`](./dsn.zh.md)。

<a id="connection-pool"></a>

## 连接池

`internal/infra/db.go`（`New`）为 GORM 支撑的连接池设界：

```go
sqlDB.SetMaxOpenConns(25)
sqlDB.SetMaxIdleConns(10)
sqlDB.SetConnMaxLifetime(30 * time.Minute)
```

并发读取下无界的连接池会耗尽 Postgres 连接；25 打开 / 10 空闲加 30 分钟回收按 2 核包络与约 0.12 次写入/秒的平均负载定尺。

<a id="server-gucs"></a>

## 服务器 GUC

五个 GUC 以 postgres 服务命令上的 `-c` 标志施加（在镜像 initdb 生成的 `postgresql.conf` 之上分层覆盖），落在生产 Ansible 模板（`devops/ansible/templates/docker-compose.yml.j2`），并经 `command: postgres -c ...` 落在开发 compose（`devops/docker-compose.yml`）：

| GUC                    | 取值   | 缘由                                                                                             |
| ---------------------- | ------ | ------------------------------------------------------------------------------------------------ |
| `shared_buffers`       | 256 MB | 这台机器不是专用 DB 服务器（2 GB 与 Caddy + Go + Next.js 共享）；256 MB 给 OS 缓存留出空间。     |
| `effective_cache_size` | 1 GB   | 向规划器提示总可用缓存（shared_buffers + OS 缓存）。                                             |
| `maintenance_work_mem` | 128 MB | Vacuum/索引维护的工作内存。                                                                      |
| `max_connections`      | 50     | 在连接池的 25 个打开连接之上留余量。                                                             |
| `synchronous_commit`   | off    | 写入速率约 0.12/秒，均为 7 天保留期的临时内容；崩溃窗口内的丢失可接受，且 `off` 不会造成不一致。 |

`shared_buffers` 与 `max_connections` 是 postmaster 上下文（需要重启）；其余可重载。它们不是 Go 代码、无法单元测试 —— 重启后用 `SHOW` 确认取值，`pg_settings.source` 对覆盖项报告 `command line`。

<a id="toast-and-lz4"></a>

## TOAST 与 lz4

Postgres TOAST 自动压缩并外置存储任何超过约 2 KB 的 `text` 值（默认开启，对 SQL 透明）—— 32 KB 的文章正文压缩后存储约 10–12 KB。`posts.body` 列使用 **lz4** TOAST 压缩器而非默认的 pglz：

```sql
-- internal/infra/migrations/000001_init.up.sql
ALTER TABLE posts ALTER COLUMN body SET COMPRESSION lz4;
```

lz4 以相近的压缩比提供约 3 倍于 pglz 的解压速度；32 KB 正文的解压成本是每次读取几十微秒，再由渲染缓存摊销。`ALTER` 位于版本化迁移中 —— golang-migrate 对每个数据库恰好运行每个版本一次（`schema_migrations` 门控），因此 `SET COMPRESSION` 获取的 `AccessExclusiveLock` 恰好被获取一次。`SET COMPRESSION` 只改元数据（效果幂等；不重写行）：现有行保持旧压缩直至被重写，因此维护窗口内的一次性 `VACUUM FULL posts` 可为它们翻新 —— 刻意不自动化，因为它在重写期间全程持有 `AccessExclusiveLock`。

应用层的 gzip 进 `BYTEA`（约 4 倍压缩，对比 TOAST 的约 3 倍）是磁盘压力确实需要时的文档化升级步骤；它把解压移进每次读取的 Go 热路径并改变列类型，因此 TOAST-lz4 是默认（见 MRFC）。

<a id="topology-sibling-container-unix-socket-docker"></a>

## 拓扑：同级容器、Unix 套接字、Docker

Postgres 运行在**同级容器**中（不在 markpost 容器之内），数据放在绕过容器 overlay2 可写层的命名卷（`pgdata`）上，Go 进程经共享的 `/var/run/postgresql` 卷通过 **Unix 域套接字**连接 —— 消除跨容器 TCP 连接会引入的 TCP/NAT 开销。套接字路径与 postgres 镜像默认的 `unix_socket_directories` 一致；DSN 使用 `host=/var/run/postgresql ... sslmode=disable`（见 [`dsn.zh.md`](./dsn.zh.md)）。

Docker 的开销是否损害这个工作负载 —— 没有，原因在拓扑本身：

| 开销类别   | 此处影响              | 缘由                                                                                                                                               |
| ---------- | --------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| CPU 虚拟化 | **无**                | Linux 容器使用 cgroups + 命名空间；没有指令翻译。容器里的 Go 二进制就是裸金属上同一个二进制，由同一个内核调度。                                    |
| 存储 I/O   | **可忽略**            | `./data`、`pgdata` 与 Postgres 套接字目录是绑定挂载 / 命名卷 —— 它们绕过 overlay2 直达宿主文件系统。只有只读的镜像内容在 overlay2 上，不在热路径。 |
| 网络       | **一跳 NAT,约微秒级** | 宿主→容器端口跳是唯一的跨命名空间穿越；容器内三个服务经回环通信，Go↔Postgres 用 Unix 套接字而非 docker0 网桥。                                     |

该拓扑正落在 Docker 的甜点区 —— 它避开了两个著名的容器性能陷阱（跨容器组网与 overlay2 数据写入）。裸金属每请求能省个位数微秒，却让约束性瓶颈（3 Mbps 链路的出站字节，见 [`caching.zh.md`](./caching.zh.md)）原封不动，还会放弃声明式的可复现环境、一条命令的自托管安装（`docker compose up`）以及在 amd64/arm64 间一致构建的 Ansible/CI 管线。

<a id="storage-estimation"></a>

## 存储估算

"每用户每天至多 1000 篇"是硬上限，不是预期均值；一个通知/临时分享工具的真实量级是每用户每天个位数篇。取保守均值 μ = 每用户每天 10 篇：

```
10 000 users × 10 posts/day × 32 KB × 7 days = 22.4 GB (raw)
× 1.3 (Postgres row/index overhead) = 29 GB
× TOAST compression (32 KB → ~12 KB) ≈ 11 GB on disk
```

11 GB 舒适地落在 40 GB 磁盘内；即使均值翻倍也约 22 GB，仍在预算内。若实际增长超出估算，已决定的升级阶梯（免费的 lz4 切换 → 应用层 gzip 进 BYTEA → 冷数据下沉对象存储）记录在 MRFC 中。

投递队列表额外增加约 4.2 GB（见 [`delivery-queue.zh.md`](./delivery-queue.zh.md) _存储_）。
