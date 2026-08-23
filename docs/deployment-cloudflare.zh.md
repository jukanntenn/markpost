# Cloudflare 生产部署

[English](deployment-cloudflare.md) | 中文

面向 SaaS 生产实例（`markpost.cc`）的操作指南：Cloudflare（Free 套餐）之后的 VPS 源站，经 Ansible playbook 部署。设计决策 —— SSL 模式、Origin CA、端口、客户端 IP 接力、缓存与清除契约 —— 见 [Cloudflare 规格](../specs/backend/cloudflare.zh.md)；已冻结的多环境参考（验收、dev、staging、运维任务）是 [`deployment.zh.md`](deployment.zh.md)。

<a id="topology"></a>

## 拓扑

```
Visitor ──HTTPS:443──> Cloudflare edge (proxied, Full strict) ──HTTPS:443──> host Caddy gateway (Origin CA) ──HTTP──> 127.0.0.1:8080 → container Caddy :2053 ──> Go :7330 / static export
```

| 层   | 值                                                                                                                        |
| ---- | ------------------------------------------------------------------------------------------------------------------------- |
| 边缘 | 443 端口，TLS 由 Cloudflare 终结                                                                                          |
| 源站 | 宿主端口 443 → 宿主机 systemd Caddy 网关（VPS 上多服务共享），持 Origin CA 证书；把 `markpost.cc` 代理到 `127.0.0.1:8080` |
| 应用 | 容器 Caddy 明文 HTTP `:2053`，仅回环发布；Go `127.0.0.1:7330`、`/app/frontend` 静态导出、Postgres 经共享 Unix-socket 卷   |

443 是唯一启用边缘缓存的代理 HTTPS 端口 —— 其余代理 HTTPS 端口（2053、8443……）一律缓存禁用，重新启用仅限 Enterprise。不需要边缘缓存的服务可以留在那些端口上。

<a id="one-time-cloudflare-dashboard"></a>

## 一次性：Cloudflare 控制台

1. **DNS** —— DNS → Records：添加 A 记录 `markpost.cc` → 源站 IP，设为 **Proxied**（橙云）。
2. **SSL 模式** —— SSL/TLS → Overview：选择 **Full (strict)**。
3. **Origin CA 证书** —— SSL/TLS → Origin Server → Create Certificate：密钥类型 RSA（2048）、主机名 `markpost.cc, *.markpost.cc`、有效期 15 年。把证书复制为 `origin.pem`、私钥复制为 `origin.key`（只展示一次）。
4. **帖子页 Cache Rule** —— Rules → Overview → Create rule → Cache Rule：规则名 `cache-post-pages`；When incoming requests match：`URI Path` `starts with` `/p-`；Then → Cache eligibility → **Eligible for cache**；规则的 **Edge TTL** 保持未设置，让源站的 `s-maxage` 生效（显式设置会覆盖它，且免费版下限为 2 小时）；**Deploy**。缺了这条规则，帖子页恒为 `DYNAMIC` —— HTML 默认不入缓存。
5. **Browser Cache TTL** —— Caching → Configuration：选择 **Respect Existing Headers**，让源站的 `max-age` 原样到达浏览器。

<a id="one-time-origin-vps"></a>

## 一次性：源站 VPS

1. **证书** —— 把 `origin.pem` 与 `origin.key` 放到宿主机的 `~/docker/markpost/certs/`（带外传输；私钥仅属主可读）。部署 playbook 会把它们复制到 `/etc/caddy/certs/markpost/` 供网关使用。
2. **宿主 Caddy 网关** —— 宿主机上由 systemd 运行的 Caddy 绑定 443，配置为 `import /etc/caddy/conf.d/*.caddy`；部署 playbook 会模板化 markpost 的站点块（Origin CA TLS → 反向代理到回环端口）并在变更时重载。其他服务向同一目录丢各自的站点块即可接入。
3. **防火墙** —— 入站默认拒绝；保留 SSH 供管理；仅放行来自 Cloudflare 公布网段的 TCP 443，使源站只能经边缘到达。
4. **`cloudflare_cidrs`** —— 把同一批网段（空格分隔）写入 `devops/ansible/group_vars/production/vars.yml`。当前列表的获取命令：

```bash
{ curl -s https://www.cloudflare.com/ips-v4; echo; curl -s https://www.cloudflare.com/ips-v6; } | tr '\n' ' '
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
2. 从仓库根目录（VPS 上的网关任务需要 root —— 提供 sudo 密码）：

```bash
ansible-playbook devops/ansible/deploy.yml -e target=production --ask-become-pass
```

playbook 渲染 compose + `config.toml` + Caddyfile、落位宿主 Caddy 网关的站点块（变更时重载）、拉取固定镜像、按 postgres → migrate → app 启动，并经边缘验证 health + version。

<a id="verify"></a>

## 验证

```bash
curl -f https://markpost.cc/api/v1/health
curl -f https://markpost.cc/api/v1/version        # == expected_version
curl -sD- -o /dev/null https://markpost.cc/<qid> | grep -i cf-cache-status   # MISS, then HIT on repeat (GET — HEAD is never cached)
```

`CF-Cache-Status` 取值见[规格参考](../specs/backend/cloudflare.zh.md#cf-cache-status-reference)。若一切均为 `DYNAMIC` —— 包括无需任何规则就默认缓存的 `/_next/static/*` 资源 —— 说明整个 zone 的缓存被挂起：检查 **Development Mode**（Caching → Configuration；3 小时后自动结束），以及 Cache Rule 是否为 **Deployed** 而非草稿。失败时：在宿主机执行 `docker compose logs markpost`；恢复方式是 fix-forward —— 应用起来时迁移通常已经跑过。

<a id="recurring-cloudflare-cidr-resync"></a>

## 例行：Cloudflare CIDR 同步

当 https://www.cloudflare.com/ips/ 变化时，在 VPS 上刷新宿主防火墙的 Cloudflare 规则（唯一的执行点），并用上面的获取命令刷新 `devops/ansible/group_vars/production/vars.yml` 的 `cloudflare_cidrs` —— 该列表在文档中的归宿。
