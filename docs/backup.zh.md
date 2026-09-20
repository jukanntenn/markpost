# 备份与恢复

[English](backup.md) | 中文

markpost 灾难恢复备份的运维指南：pgBackRest WAL 归档加每日逻辑转储到 Backblaze B2（设计与取舍见[WAL 归档 MRFC](../.agents/mrfcs/implemented/2026-07-09-wal-archival-disaster-recovery.zh.md)；当前态势见 [`specs/backend/disaster-recovery.md`](../specs/backend/disaster-recovery.zh.md)）。档位覆盖 **staging 与生产** —— staging 是晋升门，必须演练生产形态 —— 各自对着自己的桶，在该环境的 vault 定义 `b2_repo_key_id` 时激活；此前部署与备份前的管线逐字节一致。dev 不跑备份。

<a id="b2-provisioning"></a>

## B2 供给（每环境一次，首次激活前）

1. 建该环境的桶 —— `markpost-backups`（生产）或 `markpost-backups-staging`（staging）—— 并**开启版本化**：失陷的主机此后只能新增对象，改写不了历史。
2. 建限定该桶的应用密钥，能力只给 `writeFiles`、`readFiles`、`listFiles`，**不给 `deleteFiles`** —— 过期由 B2 服务端生命周期执行，不经这把键。
3. 加生命周期规则：`dumps/*` 的非当前版本 14 天后过期；`markpost/`（pgBackRest 仓库）前缀交给仓库自身的 `repo1-retention-full=2` 管理，非当前版本 30 天过期兜底。
4. 把密钥对写入 vault（见 [vault 小节](#vault-variables)）并部署。

<a id="vault-variables"></a>

## Vault 变量

每环境一对 —— 下面是生产；staging 同法，写入 `group_vars/staging/vault.yml`、用该桶的键：

```sh
ansible-vault encrypt_string --vault-id markpost-prod@avpm-client --stdin-name b2_repo_key_id \
    >> devops/ansible/group_vars/production/vault.yml
ansible-vault encrypt_string --vault-id markpost-prod@avpm-client --stdin-name b2_repo_app_key \
    >> devops/ansible/group_vars/production/vault.yml
```

`kuma_backup_url`（可选，与[监控](monitoring.zh.md)里可用性心跳同一推送模式）把每日检查变成显式的 up/down 推送；没有它，检查仍会经 cron 日志响亮地失败。

<a id="activation"></a>

## 首次激活（安全顺序）

部署会施加归档 GUC（`archive_mode=on` 需要的重启正是一次容器重建）。部署后立即初始化 stanza 并做首次全量 —— 没有基础备份的 WAL 累积不可用：

```sh
docker compose exec postgres pgbackrest --stanza=markpost stanza-create
docker compose exec postgres pgbackrest --stanza=markpost backup --type=full
docker compose exec postgres pgbackrest --stanza=markpost check
```

<a id="schedule"></a>

## 常态计划（cron，谷段）

| 时间 (UTC)      | 条目                   | 内容                                                  |
| --------------- | ---------------------- | ----------------------------------------------------- |
| 每日 03:30      | `pg-dump-backup.py`    | 逻辑转储 → zstd → rclone 上 B2 `dumps/`，限速 2 MB/s  |
| 每日 04:45      | `pgbackrest-backup.py` | 增量；每月 1 日全量                                   |
| 每日 05:30      | `pgbackrest-check.py`  | 归档往返检查 + 26 小时新鲜度 + pg_wal 增长绊线 → kuma |
| 每月 2 日 05:00 | `pgbackrest-drill.py`  | 临时容器内恢复演练，带行数断言                        |

RPO 由 WAL 尾巴界定（`archive_timeout=300` → ≤ 5 分钟写入）；RTO 约 30–45 分钟，前提是 3 Mbps 的恢复下载。

<a id="disaster-recovery"></a>

## 灾难恢复手册

1. 开通替代 VPS 并照常部署（vault 带 B2 凭据，归档档位随之就位）。
2. 动数据目录前先停 postgres：`docker compose stop postgres`。
3. 恢复最新备份集：`docker compose run --rm --entrypoint pgbackrest postgres --stanza=markpost restore`（`pgdata` 卷需为空；PITR 到更早时刻加 `--type=time --target="…"` —— 被清理过的行会回来，交由下一次保留清扫再清掉）。
4. 启动 postgres：`docker compose up -d postgres` —— WAL 自动重放到归档末端。
5. 验证（`/api/v1/ready`、抽查行数），IP 变了则改 DNS/CDN 指向。
6. 立即做一次新的全量 —— 这为后续事故重置 RPO 时钟。

<a id="failure-modes"></a>

## 故障模式

- **`pgbackrest-check.py` 往返检查失败** —— B2 不可达或凭据损坏；盯 `pg_wal` 增长（检查在 128 段 / 2 GB 绊线触发），在 40 GB 磁盘填满前修复：未归档的段无法回收，磁盘满会停掉写路径。
- **新鲜度探测失败（26 小时无备份）** —— 备份 cron 静默死亡；查 `pgbackrest-backup.log`。
- **演练行数断言失败** —— 恢复链断了；当事故处理，修复后重跑演练，在此之前不要信任仓库。
