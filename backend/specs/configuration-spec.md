# Configuration Specification

This document defines the rules and conventions for the MarkPost configuration
system. It is intended as a reference for developers and operators.

## 1. File Format and Path

### 1.1 Format

Configuration files use [TOML](https://toml.io/en/).

### 1.2 Default File Name

`markpost.toml`

### 1.3 Search Paths

The server searches for the configuration file in the following order:

1. Path specified via `--config` / `-c` CLI flag (takes precedence).
2. `./markpost.toml` — same directory as the server binary.

If no file is found, the application starts using built-in defaults and
environment variables only.

### 1.4 CLI Flag

```
./server -c /etc/markpost/production.toml
./server --config ./my-config.toml
```

If the specified file does not exist, the server fails to start with a clear
error message.

## 2. Loading Mechanism

Configuration is loaded once at startup using a singleton pattern (`sync.Once`).
The loading order is:

1. **Built-in defaults** — hardcoded in `setDefaults()`.
2. **TOML file** — overrides defaults for any keys present.
3. **Environment variables** — override both defaults and TOML values.

The application fails to start if any field fails validation after all three
layers are merged.

## 3. Value Override Rules

Priority (highest to lowest):

```
Environment variable  >  TOML file  >  Built-in default
```

A value set via environment variable always wins, even if the same key is
present in the TOML file.

Example:

```toml
# markpost.toml
[server]
port = 8080
```

```bash
MARKPOST_SERVER__PORT=9090 ./server
# Effective value: 9090 (env var wins)
```

## 4. Environment Variable Mapping

### 4.1 Prefix

All environment variables use the prefix `MARKPOST_`.

### 4.2 Nesting Separator

Double underscore `__` separates nested keys.

| TOML path                | Environment variable              |
|--------------------------|-----------------------------------|
| `debug`                  | `MARKPOST_DEBUG`                  |
| `server.host`            | `MARKPOST_SERVER__HOST`           |
| `server.port`            | `MARKPOST_SERVER__PORT`           |
| `oauth.github.client_id` | `MARKPOST_OAUTH__GITHUB__CLIENT_ID` |

### 4.3 Key Transformation

TOML keys use `snake_case`. Environment variables use `UPPER_SNAKE_CASE` with
the prefix and nesting separators applied.

TOML key `post_key_length` becomes environment variable
`MARKPOST_POST_KEY_LENGTH`.

### 4.4 Array Values

For array fields (e.g. `trusted_proxies`, `allow_origins`), environment variable
override is generally impractical. Configure these via TOML file.

### 4.5 Duration Values

Duration fields accept Go duration strings:

```
"300ms"    "5s"    "1.5h"    "24h"    "720h"
```

Valid units: `ns`, `us` / "µs", `ms`, `s`, `m`, `h`.

When set via environment variable, the same string format applies:

```bash
MARKPOST_JWT__ACCESS_TOKEN_EXPIRE="48h"
MARKPOST_DELIVERY__REQUEST_TIMEOUT="10s"
```

## 5. Example File Conventions

The example configuration file (`config.example.toml`) serves as the primary
user-facing documentation. It must follow these rules:

### 5.1 Required Fields

Required fields that have no safe default (e.g. JWT signing keys) are:

- **Uncommented** — so the file is syntactically valid.
- Set to a **placeholder value** like `"CHANGE_ME..."` or a descriptive example.
- Marked with `[REQUIRED]` in the preceding comment.

### 5.2 Optional Fields

Optional fields are:

- **Commented out** — using `# ` prefix.
- Set to their **built-in default value**.
- Marked with `[OPTIONAL]` in the preceding comment.
- Include the `Env:` tag showing the corresponding environment variable name.

### 5.3 Section Headers

Each configuration section begins with a separator comment and a brief
description of the section's purpose.

### 5.4 Inline Documentation

Every field must include:

- A one-line description of what it controls.
- The `[REQUIRED]` or `[OPTIONAL]` tag.
- The `Env:` tag with the environment variable name.
- The `Default:` tag with the built-in default value (only for optional fields).
- Relevant constraints (minimum values, valid units, etc.).
- A `⚠️` warning for security-sensitive defaults.

## 6. Validation

Field validation uses `go-playground/validator` tags on the config structs.

| Tag          | Meaning                                |
|--------------|----------------------------------------|
| `required`   | Must be non-empty / non-zero           |
| `gte=N`      | Value must be ≥ N                      |
| `oneof=a b`  | Value must be one of the listed values |
| `omitempty`  | Skip further validation if empty       |
| `url`        | Must be a valid URL                    |

Validation runs after all override layers are merged. A validation error causes
the server to exit with a descriptive message.
