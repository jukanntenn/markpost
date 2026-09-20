# 灾难恢复

[English](disaster-recovery.md) | 中文

markpost 的韧性态势：单实例，读路径在源站死亡期间靠 CDN 边缘存活，数据是保留策略管理的临时内容，备份档位是 pgBackRest WAL 归档到对象存储、辅以每日逻辑转储做格式多样性。架构及其替代方案（每小时 dump 起步、实时副本、R2、wal-g）记录在[WAL 归档 MRFC](../../.agents/mrfcs/implemented/2026-07-09-wal-archival-disaster-recovery.zh.md)；单实例决策本身（无 Redis、无副本、无第二台 VPS）是[性能优化 MRFC](../../.agents/mrfcs/implemented/2026-07-09-read-path-performance-pass.zh.md)的一部分。操作规程 —— 供给、激活、恢复、演练 —— 见 [`docs/backup.md`](../../docs/backup.zh.md)。

<a id="current-posture"></a>

## 当前态势

- **单实例。** 一台 VPS 运行 markpost 容器（Caddy + Go + Next.js）与一个同级 Postgres 容器（[`postgres-tuning.zh.md`](./postgres-tuning.zh.md)）。没有副本，没有共享缓存；内存状态（渲染缓存、限流桶）是进程本地的，重启后重建。
- **部署管线供给归档档位，按环境经 vault 激活。** vault 定义 `b2_repo_key_id` 后（staging 与生产各一对、各对着自己的桶），`devops/ansible/` 把 postgres 换成本地构建、带 pgBackRest 客户端的镜像，施加三个归档 GUC，模板化 B2 仓库配置，并排上备份/观测/演练的 cron；没有该 vault 变量时部署与备份前管线逐字节一致（与心跳、Beszel 代理共享的设置顺序契约）。
- **备份不可变且被观测。** B2 桶版本化，上传密钥无删除能力；每日检查把归档往返、备份新鲜度与 `pg_wal` 增长的判定推送到 uptime-kuma；每月 cron 演练在临时容器内恢复并断言行数落在 RPO 界内。
- **读路径在无源站时优雅降级。** 源站宕机期间，已在 CDN 边缘缓存的文章在其最长一小时的 TTL 内保持可读（[`caching.zh.md`](./caching.zh.md)）；只有写路径与未缓存的读取等待恢复。数据自身的价值在同一尺度上衰减 —— 保留策略管理的临时内容（默认 7 天，按用户覆盖，VIP 可无限）。

<a id="recovery-matrix"></a>

## 恢复矩阵

| 故障                    | 影响           | 恢复                                                                            |
| ----------------------- | -------------- | ------------------------------------------------------------------------------- |
| VPS 崩溃（可重启）      | 服务停摆       | 重启；RPO = 0                                                                   |
| VPS 毁灭 / 宿主数据丢失 | 全部数据丢失   | 新 VPS，从对象存储恢复；RPO ≤ 5 分钟（WAL 尾巴）；RTO 在 3 Mbps 下约 30–45 分钟 |
| Postgres 数据文件损坏   | 部分数据       | PITR 到损坏前的时刻；每日逻辑转储兜底基础镜像损坏                               |
| 误操作 `DELETE`         | 行丢失         | PITR 到删除之前；被保留清扫清掉的行会回来，交由下一次清扫再清                   |
| Cloudflare 故障         | 新文章无法创建 | 既有文章仍可从 CDN 边缘读取；Cloudflare 恢复后写路径回归                        |
| 对象存储故障（罕见）    | 新备份停滞     | 既有备份完好；B2 自身多副本；`pg_wal` 增长绊线告警                              |

<a id="cost"></a>

## 成本

| 项目                                                         | 成本                              |
| ------------------------------------------------------------ | --------------------------------- |
| Cloudflare 免费版                                            | $0/月（不限带宽、免费 DDoS 防护） |
| Backblaze B2 备份（现实量级约 2–3 GB）                       | ~$0–0.10/月，10 GB 免费额度内     |
| `ristretto`、`singleflight`、`pgBackRest`、`vegeta`、`pprof` | $0（开源）                        |
| **边际总计**                                                 | **≤ $0.10/月**                    |
