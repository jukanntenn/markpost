# 配置规范

[English](configuration.md) | 中文

本文档定义 markpost 配置系统的规则与约定，面向开发者和运维人员作为参考。

<a id="1-file-format-and-path"></a>

## 1. 文件格式与路径

<a id="11-format"></a>

### 1.1 格式

配置文件使用 [TOML](https://toml.io/en/)。

<a id="12-default-file-name"></a>

### 1.2 默认文件名

`config.toml`

<a id="13-search-paths"></a>

### 1.3 搜索路径

服务器按以下顺序搜索配置文件：

1. 通过 `--config` / `-c` CLI 标志指定的路径（优先）。
2. `./config.toml` — 与服务器二进制同目录。

若未找到任何文件，应用以内置默认值和仅有的环境变量启动。

<a id="14-cli-flag"></a>

### 1.4 CLI 标志

```
./server -c /etc/markpost/production.toml
./server --config ./my-config.toml
```

若指定文件不存在，服务器以清晰的错误信息启动失败。

<a id="2-loading-mechanism"></a>

## 2. 加载机制

配置在启动时以单例模式（`sync.Once`）加载一次。加载顺序为：

1. **内置默认值** — 硬编码在 `setDefaults()` 中。
2. **TOML 文件** — 覆盖其中出现的所有键的默认值。
3. **环境变量** — 同时覆盖默认值与 TOML 值。

三层合并后若任一字段校验失败，应用启动失败。

<a id="3-value-override-rules"></a>

## 3. 取值覆盖规则

优先级（从高到低）：

```
Environment variable  >  TOML file  >  Built-in default
```

环境变量设置的值总是胜出，即使 TOML 文件中存在同一键。

示例：

```toml
# config.toml
[server]
port = 8080
```

```bash
MARKPOST_SERVER__PORT=9090 ./server
# Effective value: 9090 (env var wins)
```

<a id="4-environment-variable-mapping"></a>

## 4. 环境变量映射

<a id="41-prefix"></a>

### 4.1 前缀

所有环境变量使用前缀 `MARKPOST_`。

<a id="42-nesting-separator"></a>

### 4.2 嵌套分隔符

双下划线 `__` 分隔嵌套键。

| TOML 路径                | 环境变量                            |
| ------------------------ | ----------------------------------- |
| `debug`                  | `MARKPOST_DEBUG`                    |
| `server.host`            | `MARKPOST_SERVER__HOST`             |
| `server.port`            | `MARKPOST_SERVER__PORT`             |
| `oauth.github.client_id` | `MARKPOST_OAUTH__GITHUB__CLIENT_ID` |

<a id="43-key-transformation"></a>

### 4.3 键名转换

TOML 键使用 `snake_case`。环境变量使用 `UPPER_SNAKE_CASE`，并应用前缀与嵌套分隔符。

TOML 键 `post_key_length` 对应环境变量 `MARKPOST_POST_KEY_LENGTH`。

<a id="44-array-values"></a>

### 4.4 数组值

对数组字段（如 `trusted_proxies`、`allow_origins`），环境变量覆盖通常不实用。这类字段通过 TOML 文件配置。

<a id="45-duration-values"></a>

### 4.5 时长值

时长字段接受 Go duration 字符串：

```
"300ms"    "5s"    "1.5h"    "24h"    "720h"
```

有效单位：`ns`、`us` / "µs"、`ms`、`s`、`m`、`h`。

通过环境变量设置时使用相同的字符串格式：

```bash
MARKPOST_JWT__ACCESS_TOKEN_EXPIRE="48h"
MARKPOST_DELIVERY__REQUEST_TIMEOUT="10s"
```

<a id="5-example-file-conventions"></a>

## 5. 示例文件约定

示例配置文件（`config.example.toml`）是面向用户的首要文档，须遵循以下规则：

<a id="51-required-fields"></a>

### 5.1 必填字段

没有安全默认值的必填字段（如 JWT 签名密钥）：

- **不注释** — 保证文件语法有效。
- 设为**占位值**，如 `"CHANGE_ME..."` 或描述性示例。
- 在前导注释中标记 `[REQUIRED]`。

<a id="52-optional-fields"></a>

### 5.2 可选字段

可选字段：

- **注释掉** — 使用 `# ` 前缀。
- 设为其**内置默认值**。
- 在前导注释中标记 `[OPTIONAL]`。
- 附带 `Env:` 标签，展示对应的环境变量名。

<a id="53-section-headers"></a>

### 5.3 分节头

每个配置节以分隔注释和该节用途的简短描述开始。

<a id="54-inline-documentation"></a>

### 5.4 行内文档

每个字段必须包含：

- 一行描述其控制内容。
- `[REQUIRED]` 或 `[OPTIONAL]` 标签。
- 带环境变量名的 `Env:` 标签。
- 带内置默认值的 `Default:` 标签（仅限可选字段）。
- 相关约束（最小值、有效单位等）。
- 安全敏感默认值的 `⚠️` 警告。

<a id="6-validation"></a>

## 6. 校验

字段校验使用 config struct 上的 `go-playground/validator` 标签。

| 标签        | 含义               |
| ----------- | ------------------ |
| `required`  | 必须非空 / 非零    |
| `gte=N`     | 值必须 ≥ N         |
| `oneof=a b` | 值必须是列举值之一 |
| `omitempty` | 为空时跳过后续校验 |
| `url`       | 必须是合法 URL     |

校验在所有覆盖层合并之后运行。校验错误使服务器以描述性信息退出。
