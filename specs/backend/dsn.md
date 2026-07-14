# Database DSN Specification

本文档定义 markpost 的数据库连接 DSN 格式规范、driver 推断规则，以及 SQLite 目录自动创建机制。

数据库 schema 设计见 [database-schema.md](./database-schema.md)（schema 正文冻结，本文档只管连接）。

## 一、Driver 推断

`db.driver` 配置项是**可选的**。缺省时，后端尽力从 DSN 字符串本身无歧义地推断 driver；无法确定时给出友好报错。显式配置 `driver` 时优先于推断。

### 1.1 推断规则（按优先级，逐条匹配，命中即止）

| # | 匹配特征 | 判定 | 可靠性 | 依据 |
|---|---------|------|--------|------|
| 1 | 以 `postgres://` 或 `postgresql://` 开头 | postgres | 结构化（scheme） | pgx `ParseConfig` 认 scheme |
| 2 | 以 `file:` 开头 | sqlite | 结构化（scheme） | SQLite URI 规范 |
| 3 | 等于 `:memory:` | sqlite | 结构化（字面值） | SQLite 保留名 |
| 4 | 含 `@tcp(` 或 `@unix(` | mysql | 结构化（go-sql-driver 专有格式） | `go-sql-driver/mysql` DSN 格式独此一家 |
| 5 | 含空格 + `=`（key=value 模式） | postgres | 启发式 | PG keyword 格式；sqlite 裸路径不会同时含空格和 `=` |
| 6 | 其它（裸路径，无空格无 scheme） | sqlite | 兜底假设 | sqlite 是默认 driver，裸路径最可能是 sqlite 文件 |

这 6 条覆盖了所有支持的 DSN 形态，每条都有依据。规则 5 和 6 不冲突——PG keyword 一定含空格+`=`，sqlite 裸路径不会。

### 1.2 优先级与报错

```
driver 配置了（非空）→ 用 driver（显式意图优先）
driver 缺省 → 走推断规则 1-6
              → 命中 → 用推断结果
              → 命中 6 兜底 → sqlite（仍算成功）
              → DSN 为空或完全无法归类 → 启动 fatal
```

报错信息（不打印 DSN 本身，避免泄露密码）：

```
无法从 DSN 推断数据库驱动。
请在配置文件中显式指定 [db] driver：
  driver = "sqlite"        # 开发/测试
  driver = "postgresql"    # 生产主选
  driver = "mysql"         # 可选
```

### 1.3 配置改动

| 项 | 现状 | 改为 |
|----|------|------|
| `db.driver` validator tag | `oneof=sqlite mysql postgresql`（隐式 required） | `omitempty,oneof=sqlite mysql postgresql` |
| `config.example.toml` | `driver = "sqlite"`（必填示例） | 注释掉 + 标注 `[OPTIONAL]`，说明缺省时从 DSN 推断 |

---

## 二、三端 DSN 格式

markpost 支持 PostgreSQL（生产主选）、MySQL（可选）、SQLite（开发/测试）。三种 driver 的 GORM dialector 选择需要外部输入（`sqlite.Open` / `postgres.New` / `mysql.New` 是不同构造函数）。

### 2.1 PostgreSQL

底层库：**pgx v5**（gorm-postgres 通过 `pgx.ParseConfig` 解析 DSN，`postgres.go:97`）。**不是** lib/pq。

pgx 同时接受两种 DSN 风格：

**keyword 格式**（可读性好，密码含特殊字符不需转义）：
```
host=localhost port=5432 user=markpost password=CHANGE_ME dbname=markpost sslmode=verify-full TimeZone=Asia/Shanghai
```

**URL 格式**（更像整体 URL，但密码里的 `@:/` 等字符需 percent-encode）：
```
postgres://markpost:CHANGE_ME@localhost:5432/markpost?sslmode=verify-full
```

**Unix domain socket**（同机部署，无 TCP 开销）：
```
host=/var/run/postgresql user=markpost password=CHANGE_ME dbname=markpost sslmode=disable
```

**sslmode 取值**（不在 spec 强制，由部署者按拓扑选择）：

| 取值 | 适用场景 |
|------|---------|
| `disable` | 内网 / Unix socket（无 TLS） |
| `require` | 强制 TLS，但不校验证书 |
| `verify-full` | 强制 TLS + 证书校验（推荐跨网络生产） |

> 配合 [cloudflare.md](./cloudflare.md) 的 CDN↔源站 Full strict，端到端加密闭环。

`TimeZone` 被驱动特殊处理（注册时间戳 codec，`postgres.go:37` 的 `timeZoneMatcher`），建议带上。

### 2.2 MySQL

底层库：`go-sql-driver/mysql`（gorm-mysql 通过 `mysql.ParseDSN` 解析）。

```
markpost:CHANGE_ME@tcp(localhost:3306)/markpost?charset=utf8mb4&parseTime=True&loc=Local
```

| 参数 | 值 | 说明 |
|------|-----|------|
| `charset` | **`utf8mb4`**（强制） | 完整 4 字节 UTF-8，支持 emoji 和 CJK 扩展。不用 `utf8`（实际是 utf8mb3，3 字节） |
| `parseTime` | **`True`**（必带） | 否则 `time.Time` 扫描失败 |
| `loc` | `Local` | 驱动在 `Explain()` 里读 `DSNConfig.Loc` 做时间转换 |

### 2.3 SQLite

底层库：`mattn/go-sqlite3`（gorm-sqlite 透传 DSN，`sqlite.go:53`）。参数用 `_` 前缀。

**生产 / 开发（文件型，固定格式）**：
```
file:./data/markpost.db?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000
```

| 参数 | 值 | 说明 |
|------|-----|------|
| `_foreign_keys` | `on` | 启用外键约束 |
| `_journal_mode` | `WAL` | Write-Ahead Logging，并发读写性能更好 |
| `_busy_timeout` | `5000`（ms） | 锁等待超时，避免立即报 `database is locked` |

**测试（内存型）**：
```
file:testdb?mode=memory&cache=shared
```

---

## 三、SQLite 目录自动创建

### 3.1 设计目标

开发者体验：`./markpost -c config.toml` 丢到 VPS 上直接能跑，无需手动 `mkdir data/`。

### 3.2 触发条件

仅当 `driver == "sqlite"`（推断或显式）时执行。mysql/postgres 是远程连接，目录语义不适用。

### 3.3 执行时机

在 `gorm.Open` **之前**，db 初始化阶段（`internal/infra/db.go`）。

### 3.4 解析算法（dirForSQLite）

`mattn/go-sqlite3` 驱动的 DSN 解析逻辑（源码确认，`sqlite3.go:1182` 的 `Open` 方法）：

- DSN 以 `file:` 开头 → 整串（含 `?query`）透传给 SQLite C 库，由 SQLite 按 [URI 规范](https://www.sqlite.org/uri.html) 解析
- DSN 不以 `file:` 开头 → 驱动自己在第一个 `?` 处截断，把前面的裸路径交给 SQLite

驱动**没有**导出"返回纯文件路径"的 helper，须自行复刻这个规则：

```go
func dirForSQLite(dsn string) (dir string, ok bool) {
    path := dsn
    // 1. 割 query（同驱动逻辑，pos >= 1）
    if i := strings.IndexByte(dsn, '?'); i >= 1 {
        path = dsn[:i]
    }

    // 2. 查 query 里的 mode=memory（强制内存库，不落盘）
    if hasModeMemory(dsn) {
        return "", true
    }

    switch {
    case strings.HasPrefix(path, "file:"):
        body := path[len("file:"):]
        // 特殊：file::memory:
        if body == ":memory:" {
            return "", true
        }
        // authority 形式：file://authority/path
        if strings.HasPrefix(body, "//") {
            rest := body[2:]
            slash := strings.IndexByte(rest, '/')
            if slash < 0 {
                return "", true  // file://localhost 无 path
            }
            authority := rest[:slash]
            filePart := rest[slash:]
            // SQLite 要求 authority 为空或 localhost（否则报错）
            if authority != "" && authority != "localhost" {
                return "", false  // 非法 authority
            }
            d := filepath.Dir(filePart)
            if d == "/" || d == "" {
                return "", true
            }
            return d, true
        }
        // 无 authority：相对路径（file:./data/db 或 file:/abs/path/db）
        d := filepath.Dir(body)
        if d == "." || d == "" {
            return "", true  // cwd，无需建
        }
        return d, true

    default:
        // 非 file: DSN：裸路径字面传给 SQLite
        if path == ":memory:" || path == "" {
            return "", true
        }
        // 误用 sqlite:// scheme（SQLite 不认，会当字面文件名）
        if strings.Contains(path, "://") || strings.HasPrefix(path, "sqlite:") {
            return filepath.Dir(path), false  // 标记为用户错误
        }
        d := filepath.Dir(path)
        if d == "." || d == "" {
            return "", true
        }
        return d, true
    }
}
```

### 3.5 Edge Case 表

| 输入 DSN | 分支 | 需创建目录 | ok |
|---------|------|----------|-----|
| `/abs/path/db.sqlite` | 非 file | `/abs/path` | ✅ |
| `rel/path/db.sqlite` | 非 file | `rel/path` | ✅ |
| `./data/markpost.db` | 非 file | `./data` | ✅ |
| `db.sqlite`（无目录） | 非 file | `""`（cwd，无需建） | ✅ |
| `file:./data/markpost.db?_foreign_keys=on&_journal_mode=WAL` | file | `./data` | ✅ |
| `file:/abs/path/db.sqlite?mode=ro` | file | `/abs/path` | ✅ |
| `file:///abs/path/db.sqlite` | file，authority 空 | `/abs/path` | ✅ |
| `file://localhost/abs/path/db.sqlite` | file，authority=localhost | `/abs/path` | ✅ |
| `file://darkstar/abs/path/db.sqlite` | file，非法 authority | — | ❌ |
| `:memory:` | 非 file 特例 | `""`（内存库） | ✅ |
| `file::memory:` | file，body=`:memory:` | `""`（内存库） | ✅ |
| `file::memory:?cache=shared` | file | `""`（共享内存库） | ✅ |
| `file:data.db?mode=memory` | file，query 含 mode=memory | `""`（内存库） | ✅ |
| `sqlite://host/db` | 非 file，含 `://` | — | ❌ |

### 3.6 失败行为

- `ok=false`（解析失败 / 非法 authority / 误用 scheme）→ 启动 fatal，清晰错误提示
- `os.MkdirAll(dir, 0o755)` 失败 → 启动 fatal

目录权限：`0o755`（标准默认）。

---

## 四、密码处理

DSN 里的 password 是字符串的一部分。密码注入**复用现有 env 覆盖机制**（见 [configuration.md](./configuration.md) §3）：

```
Environment variable  >  TOML file  >  Built-in default
```

生产示例：`config.toml` 里写带占位密码的 DSN，用环境变量覆盖整个 DSN：

```bash
export MARKPOST_DB__DSN="host=db user=markpost password=real_secret dbname=markpost sslmode=verify-full"
```

DSN spec 不专门讲密码注入——它走通用的配置覆盖机制。

---

## 参考

- [database-schema.md](./database-schema.md) — schema 设计（冻结）
- [configuration.md](./configuration.md) — 配置加载、env 覆盖机制
- [cloudflare.md](./cloudflare.md) — 部署模式、CDN↔源站 TLS
