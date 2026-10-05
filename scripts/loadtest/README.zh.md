# 负载测试（k6）

[English](README.md) | 中文

markpost 的端到端 HTTP 负载测试。针对 2c/2g/3Mbps 的容量研究（甜点位 / 硬上限）见 [CAPACITY_REPORT.md](CAPACITY_REPORT.md) 与 [`capacity/`](capacity/README.zh.md)；下文场景是按机制划分的回归套件。设计针对 [`specs/backend/caching.md`](../../specs/backend/caching.zh.md) 中的真实生产架构：源站位于 **Cloudflare CDN** 之后，几乎不会收到朴素的重复 GET（边缘节点已吸收这类请求）。到达源站的是 (a) **冷未命中**——某个边缘节点第一次渲染某 QID——以及 (b) **CDN 再验证**——`s-maxage`（1h）到期后携带上次 ETag 的条件 GET，源站从渲染缓存回答无响应体的 `304`。

这些场景通过切换 `If-None-Match` 头在**没有真实 CDN** 的情况下复现该请求形态，因此测得的延迟反映 2 核 / 3 Mbps 源站必须实际承载的负载。

## 前置条件

1. **一个运行中的服务。** 使用 **e2e compose**——它最接近生产：单容器镜像（Caddy + Go 经 s6）、自签 HTTPS 于 `https://localhost:2053`，以及写/投递路径所需的 mock 服务（一个 OAuth mock 和一个飞书 webhook mock）。它还放开了速率限制，写负载测试不会被 L2 限流。

   ```bash
   docker compose -f e2e/docker-compose.yml up -d --build
   curl -k https://localhost:2053/api/v1/health
   ```

   k6 脚本默认以 `insecureSkipTLSVerify` 指向 `https://localhost:2053`。可用 `SCHEME`/`HOST`/`PORT` 覆盖（例如指向明文 HTTP 的开发服务：`SCHEME=http PORT=7330`）。

2. **jq + curl** —— `run.sh` 首次运行时拉取固定版本的 k6 二进制。

3. **种子数据** —— 读场景需要 `out/qids.json`，写/浸泡需要 `out/write_keys.txt`（见下文 Seeding）。种子 CLI 跑在 e2e app 容器内：`SERVICE=app COMPOSE_FILE=e2e/docker-compose.yml`。

## 快速开始

```bash
# 0. Start the e2e stack (production-shaped, self-signed HTTPS, mocks)
docker compose -f e2e/docker-compose.yml up -d --build

# 1. Seed posts + write targets (runs the seed CLIs inside the e2e app container)
SERVICE=app COMPOSE_FILE=e2e/docker-compose.yml bash scripts/loadtest/seed.sh
SERVICE=app COMPOSE_FILE=e2e/docker-compose.yml bash scripts/loadtest/seed_write.sh

# 2. Run all short scenarios (cold-miss, revalidate-304, warm-hit, write)
bash scripts/loadtest/run.sh

# 3. Soak (1h) — run explicitly
SCENARIO=soak bash scripts/loadtest/run.sh
```

k6 二进制在首次运行时拉取到 `scripts/loadtest/k6-bin/`（已 gitignore）。结果落在 `scripts/loadtest/out/results/`（`*.json` 原始、`*-summary.json` 导出）。

## 场景

| 场景                  | 模拟                                                                                    | 速率         | 默认时长                     |
| --------------------- | --------------------------------------------------------------------------------------- | ------------ | ---------------------------- |
| `read-cold-miss`      | 边缘节点首次见到某 QID——完整 DB 读 + 渲染，singleflight 合并并发的同 QID 未命中。       | 20 req/s     | 60s                          |
| `read-revalidate-304` | `s-maxage` 之后的 CDN 再验证：GET 预热缓存，随后 `If-None-Match` 触发无响应体的 `304`。 | 50 req/s     | 60s                          |
| `read-warm-hit`       | 病毒式帖子：固定的小 QID 集轮转命中；除每个 QID 的首次外全部是渲染缓存命中。            | 100 req/s    | 60s                          |
| `write`               | `POST /:post_key`（异步投递 `Enqueue`）+ 播种用户间的 L2 限流分布。                     | 10 req/s     | 60s                          |
| `soak`                | 混合读（15/s）+ 写（2/s）保持 60m 以暴露内存/连接/goroutine 泄漏。                      | 15 + 2 req/s | 爬坡 2m + 保持 60m + 爬坡 2m |

速率按源站约 25 resp/s 的物理包络校准（`caching.md`：375 KB/s ÷ 约 15 KB/页 ≈ 25 源站响应/s）。它们建模的是 CDN 吸收大部分用户流量之后的**回源**负载，而不是总用户并发（后者由边缘处理）。

```bash
SCENARIO=read-revalidate-304 RATE=50 bash scripts/loadtest/run.sh
SCENARIO=write RATE=10 DURATION=60s bash scripts/loadtest/run.sh
SCENARIO=soak HOLD=60m READ_RATE=15 WRITE_RATE=2 bash scripts/loadtest/run.sh
```

## 每个场景测什么

- **延迟** p50/p95/p99（`http_req_duration`）。
- **源站工作拆分**——自定义计数器 `origin_revalidate_304` 与 `origin_cold_miss_200` 把廉价的再验证路径与昂贵的冷渲染分开（CDN 背后决定性的区分）。
- **带宽**——`data_received` 对 3 Mbps 源站包络；摘要报告平均 Mbps 与利用率 %，会饱和链路的场景一眼可见。
- **失败率**（`http_req_failed`）；超阈值即判失败。

### 写场景：验证投递扇出

写场景的 `POST /:post_key` 触发**异步**投递扇出：`CreatePost` 入队一个 `DeliveryJob`，调度器按 ticker 认领待处理尝试并发往作者 的飞书 webhook。HTTP 响应在发送落地之前返回，因此验证投递需要几步额外操作（e2e 栈的 `webhook-mock` 是汇聚点）：

1. 播种**通道指向 mock** 的用户，并设置匹配生成标题（`Load`）的关键词：
   ```bash
   WEBHOOK_URL="http://webhook-mock:3002/webhook" USERS=100 CHANNELS=1 \
     CHANNEL_KEYWORDS="Load" SERVICE=app COMPOSE_FILE=e2e/docker-compose.yml \
     bash scripts/loadtest/seed_write.sh
   ```
2. 运行写场景。
3. 检查 mock 是否对每个创建的帖子收到一个 webhook：
   ```bash
   docker exec e2e-app-1 wget -qO- http://webhook-mock:3002/webhooks | jq length
   ```
4. 在 `metrics-*.jsonl` 中，`markpost.delivery.dispatched_total` 应跟随创建数，`markpost.delivery.pending` 应保持在零附近（调度器的排空快于写入到达）。

注意：成功的尝试归档进 `delivery_history` 并从 `delivery_attempts` 移除，因此运行后 `delivery_attempts` 为空是**正常的**（它只持有在途 / 重试中的行）。

### 浸泡：运行后检查什么

浸泡摘要打印 k6 侧数字，但慢故障信号位于后端的 `metrics-*.jsonl`（确保 dev/prod 挂载了它——见 `devops/dev.py` / 各 compose 文件）：

- `process.runtime.go.mem.heap_alloc` —— 应在渲染缓存 `MaxCost`（128 MiB）附近走平，而不是单调爬升（ristretto TinyLFU 稳态）。
- `process.runtime.go.goroutines` —— 应保持稳定（投递 worker 池 + http handler），而不是无限增长。
- `markpost.delivery.pending` —— 应跟随写速率，而不是累积。

60 分钟保持特意超过 Postgres 的 `ConnMaxLifetime`（30m）和 CDN 的 `s-maxage`（1h），使测试期间发生一次完整的连接回收和缓存再验证周期。

## 播种

`seed.sh` 生成假帖子并经 `import-fake-posts` CLI 导入（生产 user-repo 路径，不暴露 DB 端口）：

| 变量         | 默认值  | 说明                                           |
| ------------ | ------- | ---------------------------------------------- |
| `COUNT`      | `1000`  | 帖子数。要真正全冷，设为 ≥ `RATE × DURATION`。 |
| `BODY_BYTES` | `32768` | 正文大小；匹配规格的 32 KB 平均值。            |
| `SEED`       | `1`     | 固定 RNG 种子 → 可复现的 QID/正文。            |
| `HOT_COUNT`  | `10`    | 为热命中池保留的 QID。                         |

`seed_write.sh` 播种用户（带 `mpk-` 帖子键）和可选投递通道，把键捕获到 `out/write_keys.txt`。L2 限制为 10/min/user；在 `RATE=10 × 60s = 600` 请求下需要 ≥100 个用户（`USERS=100`）。

```bash
bash scripts/loadtest/seed.sh
USERS=100 CHANNELS=3 bash scripts/loadtest/seed_write.sh
```

## 微基准

Go 级渲染/投递基准独立于服务运行，精确定位哪个阶段占主导（goldmark vs bluemonday vs minify vs 投递过滤器）：

```bash
cd backend
go test -bench=. -benchmem -run=^$ ./internal/service/post/
go test -bench=. -benchmem -run=^$ ./internal/service/delivery/filter/
```
