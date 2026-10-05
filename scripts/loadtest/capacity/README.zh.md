# 容量 / 甜点位测试环境

[English](README.md) | 中文

回答针对 2c/2g/3Mbps-无-Cloudflare 生产目标的两个问题：**甜点位在哪**（SLO 保持且有余量的可持续速率），以及**硬上限在哪**（每种机制撞上哪堵资源墙）。完整方法论与结果：仓库根 load-test README 引用的 `docs` 报告；设计讨论位于 `specs/backend/caching.md`。

## 布局

| 文件                 | 角色                                                                                                                                              |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| `docker-compose.yml` | 被测栈：app+postgres 钉在 `cpuset 0,1` 并带内存上限，PG 走 Unix socket 加生产 GUC，e2e mock；速率限制放开（L1 10000/s）使阶梯测试测的是源站天花板 |
| `config.toml`        | 该栈的 app 配置（socket DSN、放宽的限流器、256 KiB 正文上限）                                                                                     |
| `shape.sh`           | 对 app 出口施加 3 mbit tbf + 30 ms netem（经 app 网络命名空间内的 NET_ADMIN sidecar）                                                             |
| `monitor.sh`         | 每容器 cgroup v2 采样器（CPU/内存/PSI）→ CSV                                                                                                      |
| `capacity.sh`        | 运行驱动：把 k6 钉在核 2-11，用 monitor + manifest 包裹运行                                                                                       |
| `preflight.sh`       | 长跑之前的轻量验证（栈、压缩、整形、限流器、30-60s 迷你运行）                                                                                     |
| `analyze.py`         | 每阶段拐点表 / 重启风暴分桶 / 浸泡指标提取                                                                                                        |
| `restart-storm.sh`   | 发布窗口测试：再验证负载 + 运行中途 app 重启                                                                                                      |
| `slow-client.sh`     | 慢上传/慢下载探针（超时姿态文档化）                                                                                                               |
| `../verify-cf.sh`    | 部署后 Cloudflare 边缘验证清单（仅 curl 层）                                                                                                      |

k6 场景位于 `../k6/`：`capacity.js`（阶梯，MECH=cold/re304/warm）、`mixed.js`（业务画像阶梯/保持）、`spike.js`（病毒式单 QID）。

## 快速开始

```bash
# 1. Stack up (first build takes a few minutes)
docker compose -f scripts/loadtest/capacity/docker-compose.yml up -d --build

# 2. Shape egress to the VPS envelope
bash scripts/loadtest/capacity/shape.sh apply

# 3. Seeds (16k×32KB cold pool + 60×256KB worst-case + 100 write users)
COUNT=16000 SEED=1 SERVICE=app COMPOSE_FILE=scripts/loadtest/capacity/docker-compose.yml \
  bash scripts/loadtest/seed.sh
COUNT=60 BODY_BYTES=262144 SEED=2 SERVICE=app COMPOSE_FILE=scripts/loadtest/capacity/docker-compose.yml \
  bash scripts/loadtest/seed.sh && mv scripts/loadtest/out/qids.json scripts/loadtest/out/qids_256k.json
WEBHOOK_URL="http://webhook-mock:3002/webhook" USERS=100 CHANNELS=1 CHANNEL_KEYWORDS="Load" \
  SERVICE=app COMPOSE_FILE=scripts/loadtest/capacity/docker-compose.yml \
  bash scripts/loadtest/seed_write.sh

# 4. Light validation BEFORE anything long
bash scripts/loadtest/capacity/preflight.sh all

# 5. Staircases → sweet-spot hold → soak (see capacity.sh usage)
bash scripts/loadtest/capacity/capacity.sh scan cold
bash scripts/loadtest/capacity/capacity.sh scan re304
bash scripts/loadtest/capacity/capacity.sh scan warm
bash scripts/loadtest/capacity/capacity.sh scan warmcpu     # unshaped CPU-ceiling control
bash scripts/loadtest/capacity/capacity.sh scan mixed
bash scripts/loadtest/capacity/capacity.sh hold <RATE> 1800  # sweet-spot confirmation

python3 scripts/loadtest/capacity/analyze.py run scan-cold-<ts>
```

## 判定标准（出自评审过的计划）

- **甜点位**：可持续速率满足 p95 TTFB 冷 ≤ 300 ms / 304 ≤ 30 ms / 写 ≤ 200 ms，出口 ≤ 3 Mbps 的 70%，CPU PSI some avg10 < 10%，错误率 < 0.1%，保持窗口内内存走平。
- **上限**：p95 时长 > 甜点位值的 2 倍、错误 > 1%，或出口 ≥ 95%——连同绑定的资源（带宽 / CPU / 内存 / 限流器）按机制记录。

## 与真实 VPS 的偏差（已文档化）

1. 内存拆成两个容器上限（app 1280m / postgres 768m）而不是一个内核预算；容器 OOM-kill ≠ 宿主 OOM-killer。
2. 负载发生器与被测系统同机（核 2-11），共享 Docker 守护与内核——最终数字应在真实 2c/2g VPS 上用第二台机器的 k6 复核一次。
3. netem 延迟建模的是到某个 Cloudflare PoP 的单一 RTT 档（30 ms ± 5 ms）。
