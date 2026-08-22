# Cloudflare 生产部署

[English](deployment-cloudflare.md) | 中文

面向 SaaS 生产实例（`markpost.cc`）的操作指南：Cloudflare（Free 套餐）之后的 VPS 源站，经 Ansible playbook 部署。设计决策 —— SSL 模式、Origin CA、端口、客户端 IP 接力、缓存与清除契约 —— 见 [Cloudflare 规格](../specs/backend/cloudflare.zh.md)；已冻结的多环境参考（验收、dev、staging、运维任务）是 [`deployment.zh.md`](deployment.zh.md)。

<a id="topology"></a>

## 拓扑

```
Visitor ──HTTPS:443──> Cloudflare edge (proxied, Full strict) ──HTTPS──> VPS :2053 ──> Caddy ──> Go :7330 / static export
```

| 层   | 值                                                                                              |
| ---- | ----------------------------------------------------------------------------------------------- |
| 边缘 | 443 端口，TLS 由 Cloudflare 终结                                                                |
| 源站 | 宿主端口 2053 → 容器 2053，Caddy 持 Origin CA 证书；一条 Origin Rule 把目标端口 443 改写为 2053 |
| 应用 | Go `127.0.0.1:7330`、`/app/frontend` 静态导出、Postgres 经共享 Unix-socket 卷                   |

<a id="one-time-cloudflare-dashboard"></a>

## 一次性：Cloudflare 控制台

1. **DNS** —— DNS → Records：添加 A 记录 `markpost.cc` → 源站 IP，设为 **Proxied**（橙云）。
2. **SSL 模式** —— SSL/TLS → Overview：选择 **Full (strict)**。
3. **Origin CA 证书** —— SSL/TLS → Origin Server → Create Certificate：密钥类型 RSA（2048）、主机名 `markpost.cc, *.markpost.cc`、有效期 15 年。把证书复制为 `origin.pem`、私钥复制为 `origin.key`（只展示一次）。
4. **Origin Rule** —— Rules → Overview → Create rule → Origin Rule：规则名 `origin-port-2053`；When incoming requests match：`Hostname` `equals` `markpost.cc`；Set origin parameters → **Destination port** → `Rewrite to` `2053`；**Deploy**。

<a id="one-time-origin-vps"></a>

## 一次性：源站 VPS

1. **证书** —— 把 `origin.pem` 与 `origin.key` 放到宿主机的 `~/docker/markpost/certs/`（带外传输；私钥仅属主可读）。
2. **防火墙** —— 入站默认拒绝；保留 SSH 供管理；仅放行来自 Cloudflare 公布网段的 TCP 2053，使源站只能经边缘到达。
3. **`cloudflare_cidrs`** —— 把同一批网段（空格分隔）写入 `devops/ansible/group_vars/production/vars.yml`。当前列表的获取命令：

```bash
curl -s https://www.cloudflare.com/ips | tr '\n' ' '
```

<a id="secrets"></a>

## 密钥

`devops/ansible/group_vars/production/vault.yml` 中逐变量的 `!vault` 条目（avpm 身份 `markpost-prod`，来自根 `ansible.cfg`；密钥环经 `avpm unlock` 解锁）。必需键：`jwt_access_signing_key`、`jwt_refresh_signing_key`、`admin_password`、`db_password`、`github_client_id`、`github_client_secret`、`cloudflare_api_token`。逐个生成：

```bash
printf '%s' '<secret>' | ansible-vault encrypt_string \
    --vault-id markpost-prod@~/.local/bin/avpm-client \
    --stdin-name <name> >> devops/ansible/group_vars/production/vault.yml
```

**核对已设置的值**（从仓库根目录运行；`ansible.cfg` 提供 vault 身份）：

```bash
ansible localhost -m debug -a "var=<name>" \
    -e @devops/ansible/group_vars/production/vault.yml
```

**修改某个值** —— 删除 `vault.yml` 中该变量的 `!vault` 块，用新值重跑上面的 `encrypt_string` 命令。

**GitHub OAuth 应用** → `github_client_id` / `github_client_secret`：GitHub → Settings → Developer settings → OAuth Apps → **New OAuth App**。Homepage `https://markpost.cc`、callback URL `https://markpost.cc/auth/callback`。复制 Client ID 并生成 Client Secret。

**Cloudflare API token** → `cloudflare_api_token`：控制台 → My Profile → API Tokens → **Create Token** → 模板 **Zone Cache Purge**、zone 范围 `markpost.cc`（驱动删文时的缓存清除；zone id 位于 `vars.yml`）。

<a id="deploy"></a>

## 部署

生产运行固定版本的 Docker Hub 发布（`docker-publish.yml` 于 `v*` 标签触发）。staging 验证同一版本后晋升：

1. 在 `devops/ansible/group_vars/production/vars.yml` 中，把 `markpost_version` 设为 Docker Hub 标签（无前导 `v`），`expected_version` 设为匹配的 `v` 前缀 git 标签。
2. 从仓库根目录：

```bash
ansible-playbook devops/ansible/deploy.yml -e target=production
```

playbook 渲染 compose + `config.toml` + Caddyfile、拉取固定镜像、按 postgres → migrate → app 启动，并经边缘验证 health + version。

<a id="verify"></a>

## 验证

```bash
curl -f https://markpost.cc/api/v1/health
curl -f https://markpost.cc/api/v1/version        # == expected_version
curl -sI https://markpost.cc/<qid> | grep -i cf-cache-status   # MISS, then HIT on repeat
```

`CF-Cache-Status` 取值见[规格参考](../specs/backend/cloudflare.zh.md#cf-cache-status-reference)。失败时：在宿主机执行 `docker compose logs markpost`；恢复方式是 fix-forward —— 应用起来时迁移通常已经跑过。

<a id="recurring-cloudflare-cidr-resync"></a>

## 例行：Cloudflare CIDR 同步

当 https://www.cloudflare.com/ips/ 变化时，更新列表的两处归宿并重新部署：在 VPS 上刷新宿主防火墙的 Cloudflare 规则，并用上面的获取命令刷新 `devops/ansible/group_vars/production/vars.yml` 的 `cloudflare_cidrs`。
