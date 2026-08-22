# 部署与运维指南

[English](deployment.md) | 中文

> **已冻结。** 本多环境指南作为参考快照保留；生产环境的 Cloudflare 部署见 [`deployment-cloudflare.zh.md`](deployment-cloudflare.zh.md)，并在该处维护。

本文档覆盖 markpost 在全部环境下的完整生命周期：本地验收、dev、staging 与 production。所有环境运行同一产物：单镜像内置 s6-overlay 监管的 Caddy + Go 组合，没有独立的 Node 服务器。

<a id="architecture"></a>

## 架构

所有环境共享同一拓扑：

```
           ┌─────────────────────────────────────────────────────┐
           │ markpost container (s6-overlay)                     │
           │                                                     │
  :2053 ─▶ │  Caddy :2053  ──reverse_proxy──▶  Go backend :7330  │
           │                                                     │
           └──────────────────────────┬──────────────────────────┘
                                      │ Unix socket (/var/run/postgresql)
                                      ▼
           ┌─────────────────────────────────────────────────────┐
           │ postgres container (postgres:17-alpine)             │
           └─────────────────────────────────────────────────────┘
```

Caddy 直接服务 Next.js 静态导出（`/app/frontend`），并把 `/api/v1/*`、`/static/*`、`/swagger/*`、`/mpk-*`、`/p-*` 反向代理到 `127.0.0.1:7330` 的 Go 后端。Postgres 经共享的 `postgres-socket` 具名卷（Unix socket）访问。

<a id="internal-ports-identical-everywhere"></a>

### 内部端口（各环境一致）

| 组件     | 端口   | 是否发布                      |
| -------- | ------ | ----------------------------- |
| Go 后端  | `7330` | 否（Caddy 代理到它）          |
| Caddy    | `2053` | 是（宿主端口随环境而异）      |
| Postgres | `5432` | 否（Unix socket，仅容器内部） |

<a id="host-ports-per-environment"></a>

### 宿主端口（随环境而异）

容器始终监听 2053；映射到的宿主端口因环境而异：

| 环境              | 宿主端口 → 容器 | 说明                                                   |
| ----------------- | --------------- | ------------------------------------------------------ |
| e2e / acceptance  | `2053:2053`     | 本地验证                                               |
| dev (fn)          | `8089:2053`     | 为向后兼容固定                                         |
| staging (oect)    | `8089:2053`     | 为向后兼容固定                                         |
| production (ttyo) | `2053:2053`     | 经 Cloudflare（边缘 443 → 经 Origin Rule 到源站 2053） |

生产中，访问者以标准 HTTPS 端口 443 连接 Cloudflare（`https://markpost.cc`）。Cloudflare 再经 2053（源站宿主端口）回源，由 Origin Rule 设置 —— 见 §4.1。443 端口从不出现在源站一侧。

<a id="tls-strategy"></a>

## TLS 策略

| 环境             | TLS profile | 机制                                                 |
| ---------------- | ----------- | ---------------------------------------------------- |
| e2e / acceptance | `internal`  | Caddy `tls internal`（自签 CA，烘焙进镜像）          |
| dev / staging    | `http`      | `auto_https off`，纯 HTTP                            |
| production       | `origin`    | Cloudflare Origin CA 证书（`/app/certs/origin.pem`） |

---

<a id="1-local-acceptance-verification"></a>

## 1. 本地验收

验收 compose（`docker/docker-compose.yml`）镜像生产拓扑（Caddy + Go + Postgres 经 Unix socket），但从本地源码构建，并使用 Caddy 内部 CA 做自签 TLS。它**不持久化任何数据** —— 与 e2e 套件一样，应用数据目录与 Postgres 数据全部活在临时容器里。

```bash
# From the repo root:
docker compose -f docker/docker-compose.yml up -d --build

# Verify (self-signed cert, so -k):
curl -k https://localhost:2053/api/v1/health

# Tear down (no named data volumes to worry about):
docker compose -f docker/docker-compose.yml down
```

配置读取自 `.local/config.acceptance.toml`（gitignored，按开发者各持）。

<a id="2-dev-fn-192-168-5-200"></a>

## 2. Dev（fn @ 192.168.5.200）

远程快速迭代环境。自托管，无 Cloudflare，无 HTTPS。既有数据（489 帖、1 用户、1 渠道）保存在具名 `pgdata` 卷中。

<a id="deploy-update"></a>

### 部署 / 更新

```bash
# Build & push the rolling main tag from the current workspace (default
# tag, default registry), then deploy:
python3 docker/build.py --push
ansible-playbook devops/ansible/deploy.yml          # dev (fn) is the default target
```

playbook 以部署后验证收尾（见 §5）：它经公开 URL 轮询 `/api/v1/health`，并用 `/api/v1/version` 对照部署检出区的 `git describe` —— 不匹配意味着容器仍在运行旧镜像。

<a id="schema-updates"></a>

### Schema 更新

Schema 变更是内嵌于二进制的带版本 SQL 迁移（`backend/internal/infra/migrations/`）。部署 playbook 先启动 Postgres，运行 `migrate` 子命令（`postgres → migrate → app`），然后才启动应用 —— `infra.New` 自身从不迁移。`pgdata` 卷中的数据不受前向迁移影响。

---

<a id="3-staging-oect-192-168-5-50"></a>

## 3. Staging（oect @ 192.168.5.50）

Staging 镜像生产拓扑（Postgres 经 Unix socket），但没有 Cloudflare、没有 HTTPS（`tls_profile: http`；公开 URL `https://markpost.bytehome.fun` 终止于隧道）。它是生产的晋级闸门：运行钉住的 Docker Hub release 而非浮动标签，因此在此验证的产物与生产将运行的逐字节相同。

<a id="deploy-update-1"></a>

### 部署 / 更新

1. 在 `devops/ansible/group_vars/staging/vars.yml` 提升 `markpost_version`（Docker Hub 标签格式，无前导 `v`），并把 `expected_version` 提升为二进制在 `/api/v1/version` 报告的、匹配的带 `v` git 标签：

   ```bash
   # one-off override without editing vars:
   ansible-playbook devops/ansible/deploy.yml -e target=staging -e markpost_version=0.2.1
   ```

2. 部署：

   ```bash
   ansible-playbook devops/ansible/deploy.yml -e target=staging
   ```

playbook 渲染 compose 文件与 `config.toml`（Unix-socket Postgres DSN），拉取钉住的镜像，运行迁移，并经公开 URL 验证健康与版本（见 §5）。staging 没有回滚安全网，也没有停机约束：失败时诊断、修复、重新部署。

---

<a id="4-production-ttyo-43-133-160-29-greenfield"></a>

## 4. Production（ttyo @ 43.133.160.29）—— greenfield

生产是 Cloudflare 之后的一台全新 VPS。本节覆盖一次性设置（VPS 准备、Cloudflare 配置、Origin CA 证书）与日常部署。

<a id="4-1-one-time-cloudflare-setup"></a>

### 4.1 一次性 Cloudflare 配置

在 Cloudflare 控制台对 `markpost.cc` zone 执行：

1. **DNS 记录：** 创建 A 记录 `markpost.cc` → `<VPS 公网 IP>`，**Proxied**（橙云）。这隐藏源站 IP 并启用 CDN/WAF/DDoS。

2. **SSL/TLS 模式：** 设为 **Full (strict)**。这要求源站证书（下一步）并校验之，防止 CF↔源站一腿遭 MITM。

3. **Origin CA 证书：** 进入 SSL/TLS → Origin Server → Create Certificate。
   - 密钥类型：RSA (2048) 或 ECC。
   - 主机名：`markpost.cc, *.markpost.cc`（或仅 `markpost.cc`）。
   - 有效期：15 年（或按需）。
   - 复制**证书**（存为 `origin.pem`）与**私钥**（`origin.key`）。

4. **Origin Rule（目标端口改写）：** 进入 Rules → Origin Rules → Create rule。Cloudflare 默认以 443 连源站，但源站 Caddy 监听 2053（源站宿主端口）。添加规则改写目标端口：
   - **If:** `Hostname` `equals` `markpost.cc`
   - **Then:** `Destination port` `Rewrite to` `2053`
   - Free 计划即可用（10 条规则）。

> **缓存注意：** Cloudflare 依据访客侧边缘端口（443，用户访问的标准 HTTPS 端口）缓存，而非源站端口。`_next/static/*` 资产默认边缘缓存，不会每次请求回源。源站端口（2053）与缓存决策无关 —— 缓存查找发生在边缘、回源连接之前。

<a id="4-2-one-time-vps-preparation"></a>

### 4.2 一次性 VPS 准备

```bash
ssh alice@43.133.160.29 << 'EOF'
# Install Docker if not present (Docker Engine + Compose plugin).
# Create the certs directory and place the Origin CA cert + key:
mkdir -p ~/docker/markpost/certs
# (Transfer origin.pem and origin.key into ~/docker/markpost/certs/ — out of band,
#  e.g. scp from your workstation. The key must stay private.)
chmod 600 ~/docker/markpost/certs/origin.key
chmod 644 ~/docker/markpost/certs/origin.pem
EOF
```

**防火墙 —— 将源站限定为 Cloudflare IP：** 仅允许来自 Cloudflare 公布的 CIDR 段的入站 TCP 2053。这样即使源站 IP 泄露也无法直连。从 `https://www.cloudflare.com/ips/`（IPv4 + IPv6）获取当前段。

ufw 示例（按你的防火墙调整）：

```bash
# Reset and allow SSH + Cloudflare only on 2053.
ufw default deny incoming
ufw allow 22/tcp                          # SSH (consider restricting to your IP)
# IPv4
for ip in $(curl -s https://www.cloudflare.com/ips-v4); do ufw allow from "$ip" to any port 2053 proto tcp; done
# IPv6
for ip in $(curl -s https://www.cloudflare.com/ips-v6); do ufw allow from "$ip" to any port 2053 proto tcp; done
ufw enable
```

同时把 Cloudflare CIDR 段同步进 `group_vars/production/vars.yml` 的 `cloudflare_cidrs` 变量（空格分隔），使 Caddy 的 `trusted_proxies` 只信任 Cloudflare。

<a id="4-3-production-secrets"></a>

### 4.3 生产密钥

密钥是 `group_vars/production/vault.yml` 中逐变量的 `!vault` 块，各自以根 `ansible.cfg` 声明的 avpm keyring 身份 `markpost-prod` 加密。添加或轮换（明文经 stdin）：

```bash
printf '%s' '<the-secret>' | ansible-vault encrypt_string \
    --vault-id markpost-prod@~/.local/bin/avpm-client \
    --stdin-name <name> >> devops/ansible/group_vars/production/vault.yml
```

必需键：`jwt_access_signing_key`、`jwt_refresh_signing_key`、`admin_password`、`db_password`。可选：`github_client_id`、`github_client_secret`、`cloudflare_api_token`。

<a id="4-4-deploy-to-production"></a>

### 4.4 部署到生产

生产运行 `v*` git 标签上由 `docker-publish.yml` 工作流发布的**钉住的 Docker Hub release**（见 §7），而非本地构建的镜像。在 staging 验证同一版本后，提升 `group_vars/production/vars.yml` 的 `markpost_version`，然后：

```bash
ansible-playbook devops/ansible/deploy.yml -e target=production
```

playbook 创建目录结构、渲染三份配置文件、拉取并重建容器（postgres → migrate → app），并经 Cloudflare 验证部署（健康 + 版本）。

<a id="4-5-verify-production"></a>

### 4.5 验证生产

```bash
# Through Cloudflare (what users see):
curl -f https://markpost.cc/api/v1/health

# Direct to origin (should work only from a Cloudflare IP after firewall setup):
curl -f --resolve markpost.cc:2053:43.133.160.29 \
    --cacert <(echo) https://markpost.cc:2053/api/v1/health 2>/dev/null || true
```

---

<a id="5-ansible-playbook-reference"></a>

## 5. Ansible playbook 参考

从仓库根运行 —— 根 `ansible.cfg` 提供 inventory 与 avpm vault 身份，无需 `-i` / `--vault-password-file` 旗标。环境以 `-e target=<env>` 选择；**dev 是默认**，裸命令部署 fn（可崩溃的 dogfood 环境）。

<a id="directory-layout-unified"></a>

### 目录布局（统一）

```
ansible.cfg                     # root: inventory + avpm vault_identity_list
devops/ansible/
  hosts.yml                    # dev/staging/production groups
  deploy.yml                   # single playbook (default target: dev)
  group_vars/
    all.yml                    # shared: app_name, ports, paths
    dev.yml                    # dev: rolling internal-registry main tag
    dev/vault.yml              # dev secrets (per-variable !vault, avpm markpost-dev)
    staging.yml                # staging: pinned Docker Hub release
    staging/vault.yml          # staging secrets (avpm markpost-staging)
    production.yml             # production: pinned release, origin TLS, CF CIDRs
    production/vault.yml       # production secrets (avpm markpost-prod)
  host_vars/
    fn.yml / oect.yml / ttyo.yml
  templates/
    docker-compose.yml.j2      # single, conditional certs mount + host port
    config.toml.j2             # single, full section set
    Caddyfile.dev              # static HTTP Caddyfile (dev)
    Caddyfile.staging          # static HTTP Caddyfile (staging)
    Caddyfile.production.j2    # Origin CA + cloudflare_cidrs (production)
```

<a id="commands"></a>

### 命令

```bash
ansible-playbook devops/ansible/deploy.yml                    # dev (fn)
ansible-playbook devops/ansible/deploy.yml -e target=staging
ansible-playbook devops/ansible/deploy.yml -e target=production
```

<a id="environment-variable-matrix"></a>

### 环境变量矩阵

| 变量               | dev                               | staging                         | production                     |
| ------------------ | --------------------------------- | ------------------------------- | ------------------------------ |
| `image`            | `192.168.5.50:5000/markpost:main` | `jukanntenn/markpost:<pinned>`  | `jukanntenn/markpost:<pinned>` |
| `expected_version` | 部署检出区的 `git describe`       | `<pinned git tag>`              | `<pinned git tag>`             |
| `host_port`        | `8089`                            | `8089`                          | `2053`                         |
| `tls_profile`      | `http`                            | `http`（隧道终结 HTTPS）        | `origin`（CF Full strict）     |
| `public_url`       | `http://192.168.5.200:8089`       | `https://markpost.bytehome.fun` | `https://markpost.cc`          |
| `debug`            | `true`                            | `false`                         | `false`                        |
| `cloudflare_cidrs` | _（未设）_                        | _（未设）_                      | CF CIDR 列表                   |

staging 与 production 钉住同一版本（各自 group_vars 的 `markpost_version`）：staging 是晋级闸门，在那里验证过的产物必须与生产运行的逐字节相同。

<a id="post-deploy-health-check"></a>

### 部署后健康检查

playbook 的最终任务从控制机运行 `scripts/check_deploy.py`：

1. 轮询 `{public_url}/api/v1/health` 直到 `{"status": "ok"}`（5 秒间隔，120 秒超时） —— 走真实访客的路径（dev 走局域网、staging 走隧道、production 走 Cloudflare 边缘），因此端口/Caddy/边缘路径损坏会让部署失败，即便容器本身健康。
2. 将 `{public_url}/api/v1/version` 与期望版本比对：dev 用部署检出区的 `git describe`（`main` 标签部署），staging/production 用钉住的 git 标签。这抓住"容器起来了但还在跑上一个镜像"。

失败时在宿主机查 `docker compose logs markpost`。没有自动回滚 —— 应用起来时迁移通常已运行，恢复路径是向前修复并重新部署。

---

<a id="6-operational-tasks"></a>

## 6. 运维任务

<a id="reset-a-users-password"></a>

### 重置用户密码

```bash
# On the target host, in the app dir:
docker compose exec markpost markpost -c /app/config.toml \
    reset-password -u <username>
```

<a id="prune-expired-posts-cron"></a>

### 清理过期帖子（cron）

```bash
docker compose exec markpost markpost -c /app/config.toml \
    prune-expired-posts --dry-run   # check first, then omit --dry-run
```

<a id="prune-old-delivery-history-cron"></a>

### 清理旧投递历史（cron）

```bash
docker compose exec markpost markpost -c /app/config.toml \
    prune-delivery-history
```

<a id="sync-cloudflare-cidrs"></a>

### 同步 Cloudflare CIDR

Cloudflare 更新 IP 段时，更新两处：

1. VPS 防火墙（重跑上面的 ufw 循环）。
2. `devops/ansible/group_vars/production/vars.yml` → `cloudflare_cidrs`（空格分隔），然后重新部署生产。

<a id="view-logs"></a>

### 查看日志

```bash
docker compose logs -f markpost    # Caddy + Go (s6 merges both to stdout)
```

---

<a id="7-image-tag-semantics"></a>

## 7. 镜像标签语义

| 标签         | registry            | 指向                                       | 由谁移动                              |
| ------------ | ------------------- | ------------------------------------------ | ------------------------------------- |
| `main`       | `192.168.5.50:5000` | `docker/build.py` 最近从某工作区构建的内容 | 本地 `python3 docker/build.py --push` |
| `X.Y.Z`      | Docker Hub          | git 标签 `vX.Y.Z`（稳定 release）          | `docker-publish.yml` 于标签推送       |
| `X.Y.Z-rc.N` | Docker Hub          | git 标签 `vX.Y.Z-rc.N`（预发布）           | `docker-publish.yml` 于标签推送       |
| `latest`     | Docker Hub          | 最新的**稳定** release（绝非预发布）       | `docker-publish.yml`，仅稳定版        |

- Git 标签遵循 SemVer 2.0.0：`v0.1.3` 稳定、`v0.1.3-rc.1` 预发布（连字符必需 —— `v0.1.3rc1` 不是合法 semver，会弄坏每个 semver 感知的工具）。Docker 标签去掉前导 `v`。
- 一个 release 是稳定的，当且仅当 git 标签精确匹配 `^v\d+\.\d+\.\d+$` —— 单一规则，由 `docker-publish.yml`（latest 门控）与 `release.yml`（prerelease / make_latest 旗标）逐字共享。
- 格式切换前发布的 Docker Hub 标签（`v0.1.0` … `v0.2.0-rc.3`，带 `v`）保持原样；新 release 用无 `v` 形式。
- `main` 从不离开内部 registry，CI 也从不构建它 —— 它纯粹是 dev（fn）跟踪的本地构建滚动标签。
- 本地多架构构建（`docker/build.py --push --all-platforms`）经 buildx + QEMU；release 工作流则原生构建每个平台（x86 runner 上 amd64、GitHub arm runner 上 arm64）再合并 manifest —— 无模拟、无 registry 缓存。
