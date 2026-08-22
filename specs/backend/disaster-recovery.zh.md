# 灾难恢复

[English](disaster-recovery.md) | 中文

markpost 的韧性态势：单实例，读路径在源站死亡期间靠 CDN 边缘存活，数据是 7 天保留期的临时内容，备份设计是 WAL 归档到对象存储。已决定但尚未部署的备份架构及其替代方案（实时副本、R2、wal-g）记录在[灾难恢复 MRFC](../../.agents/mrfcs/proposed/2026-07-09-wal-archival-disaster-recovery.zh.md)；单实例决策本身（无 Redis、无副本、无第二台 VPS）是[性能优化 MRFC](../../.agents/mrfcs/implemented/2026-07-09-read-path-performance-pass.zh.md)的一部分。

<a id="current-posture"></a>

## 当前态势

- **单实例。** 一台 VPS 运行 markpost 容器（Caddy + Go + Next.js）与一个同级 Postgres 容器（[`postgres-tuning.zh.md`](./postgres-tuning.zh.md)）。没有副本，没有共享缓存；内存状态（渲染缓存、限流桶）是进程本地的，重启后重建。
- **部署管线不配置自动备份。** `devops/ansible/` 供给实例但不调度任何 `pg_dump`/WAL 上传；备份是运维者按上述 MRFC 中已决定设计执行的动作。
- **读路径在无源站时优雅降级。** 源站宕机期间，已在 CDN 边缘缓存的文章在其最长一小时的 TTL 内保持可读（[`caching.zh.md`](./caching.zh.md)）；只有写路径与未缓存的读取等待恢复。数据自身的价值在同一尺度上衰减 —— 它是 7 天保留期的临时内容。

<a id="recovery-matrix"></a>

## 恢复矩阵

| 故障                    | 影响           | 恢复                                                                        |
| ----------------------- | -------------- | --------------------------------------------------------------------------- |
| VPS 崩溃（可重启）      | 服务停摆       | 重启；RPO = 0                                                               |
| VPS 毁灭 / 宿主数据丢失 | 全部数据丢失   | 新 VPS，从对象存储恢复；RPO 约秒级（WAL）或 ≤1 小时（dump）；RTO 约 30 分钟 |
| Postgres 数据文件损坏   | 部分数据       | PITR 到损坏前的时刻                                                         |
| 误操作 `DELETE`         | 行丢失         | PITR 到删除之前                                                             |
| Cloudflare 故障         | 新文章无法创建 | 既有文章仍可从 CDN 边缘读取；Cloudflare 恢复后写路径回归                    |
| 对象存储故障（罕见）    | 新备份停滞     | 既有备份完好；B2 自身多副本                                                 |

<a id="cost"></a>

## 成本

| 项目                                                         | 成本                              |
| ------------------------------------------------------------ | --------------------------------- |
| Cloudflare 免费版                                            | $0/月（不限带宽、免费 DDoS 防护） |
| Backblaze B2 备份（约 40 GB）                                | $0.20/月                          |
| `ristretto`、`singleflight`、`pgBackRest`、`vegeta`、`pprof` | $0（开源）                        |
| **边际总计**                                                 | **约 $0.20/月**                   |
