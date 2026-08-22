# 数据库 DSN 规范

[English](dsn.md) | 中文

本文档定义 markpost 的 PostgreSQL 连接 DSN 格式、sslmode 选项、时区处理与密码注入。数据库 schema 由 [database-schema.zh.md](./database-schema.zh.md) 拥有；配置加载与环境变量覆盖机制由 [configuration.zh.md](./configuration.zh.md) 拥有。

PostgreSQL 是唯一支持的数据库。`db.driver` 配置键只接受 `"postgresql"`（由 `oneof=postgresql` 校验、由 Viper 提供默认值），`internal/infra/db.go` 用 GORM 基于 pgx v5 的 postgres 驱动打开连接——绝不用 lib/pq。

<a id="dsn-formats"></a>

## DSN 格式

pgx 接受两种 DSN 风格；两者在一切接受 DSN 的地方（`config.toml` 的 `[db] dsn`、`MARKPOST_DB__DSN`、开发 compose 环境）都可用。

**关键词格式**（可读性好；含特殊字符的密码无需转义）：

```
host=localhost port=5432 user=markpost password=CHANGE_ME dbname=markpost sslmode=verify-full
```

**URL 格式**（密码含 `@:/` 及类似字符时必须百分号编码）：

```
postgres://markpost:CHANGE_ME@localhost:5432/markpost?sslmode=verify-full
```

**Unix domain socket**（同机部署；无 TCP 开销——调优笔记见 [postgres-tuning.md](./postgres-tuning.zh.md)）：

```
host=/var/run/postgresql user=markpost password=CHANGE_ME dbname=markpost sslmode=disable
```

<a id="sslmode"></a>

## sslmode

该值是部署者的拓扑选择，本规范不强制：

| 值            | 适用场景                                  |
| ------------- | ----------------------------------------- |
| `disable`     | 私有网络 / Unix socket（无 TLS）          |
| `require`     | 强制 TLS，不校验证书                      |
| `verify-full` | 强制 TLS + 证书校验（跨网络生产环境推荐） |

配合 [cloudflare.md](./cloudflare.zh.md) 的 Cloudflare Full（strict）设置，`verify-full` 合上 CDN 与源站之间的端到端加密闭环。

<a id="timezone-handling"></a>

## 时区处理

`db.timezone`（IANA 名称，默认 `UTC`）把三件事钉在同一个时区上，使写入、读取与 `time.Now()` 无论进程 `TZ` 或服务器默认值如何都保持一致（`internal/infra/db.go`）：

- `time.Local` 被设为配置的时区，pgx 的 timestamptz 解码与每个 `time.Now()` 调用方都落在其中
- 一个 `timezone=<zone>` 参数被注入 DSN（除非已存在），pgx 驱动把它作为会话时区应用到每条池化连接
- GORM 的 `NowFunc` 以同一时区为 `autoCreateTime` / `autoUpdateTime` 列盖章

时区名在启动时经 `time.LoadLocation` 校验，非法名称快速失败。

<a id="connection-pool"></a>

## 连接池

`infra.New` 配置连接池：`MaxOpenConns(25)`、`MaxIdleConns(10)`、`ConnMaxLifetime(30m)`。开发 compose 除服务器侧的 `max_connections=50` 外不调高任何单连接 Postgres 限制。

<a id="password-handling"></a>

## 密码处理

密码是 DSN 字符串的一部分。注入遵循标准配置优先级（见 [configuration.zh.md](./configuration.zh.md)）：环境变量 > TOML 文件 > 内置默认值。生产模式是 `config.toml` 中带占位符的 DSN，真实值由环境提供：

```bash
export MARKPOST_DB__DSN="host=db user=markpost password=real_secret dbname=markpost sslmode=verify-full"
```

<a id="references"></a>

## 参考

- [database-schema.zh.md](./database-schema.zh.md) — schema 设计
- [configuration.zh.md](./configuration.zh.md) — 配置加载、环境变量覆盖机制
- [cloudflare.md](./cloudflare.zh.md) — 部署模式、CDN ↔ 源站 TLS
