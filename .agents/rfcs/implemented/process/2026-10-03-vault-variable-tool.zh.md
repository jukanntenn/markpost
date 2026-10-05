# RFC: scripts/vault.py for ergonomic per-variable vault management

Status: implemented

[English](../2026-10-03-vault-variable-tool.md) | 中文

## Problem

文档教的手工配方（`ansible-vault encrypt_string --vault-id ... >> vault.yml`）在仓库自身的 ansible.cfg 下必然失败：`vault_identity_list` 预载全部三个身份，`--vault-id` 只是再追加第四个，ansible 要求改用 `--encrypt-vault-id` —— 首次实际使用就踩中。三个进一步的人体工学缺口叠加其中：`echo` 管道喂值会带尾随换行、静默毁掉 B2 密钥；追加式写入留下重复变量块、靠"后者覆盖"的隐式语义收敛；验证解密值需要一条 ad-hoc `ansible localhost -m debug` 命令，其输出解析脆弱。删除变量则完全靠手工。

## Decision

**`scripts/vault.py` —— 纯标准库 CLI（`set`/`get`/`list`/`check`/`remove`），围绕逐变量块形态编排 ansible-vault CLI。** 两个原生原语使无临时文件成为可能：实测发现 `ansible-vault view -` 可将 stdin 解密到 stdout（`VaultEditor.read_data('-')`），而 `decrypt --output=-` 输出字节精确的明文（`view` 的显示层会给缺换行的值补一个换行，产生歧义）。`set` 把值管道喂给 `encrypt_string --encrypt-vault-id`，按文件自身缩进 upsert 块，写盘前做密码学自验。所有子进程以清空的 `ANSIBLE_VAULT_IDENTITY_LIST` 加单个显式 `--vault-id` 运行 —— 用其他环境密码加密的密文会失败而非静默解密（实测：不加此项，仓库配置会跨环境解密）。`env` 为白名单，映射到文件路径与身份标签，隐藏 `production`→`markpost-prod` 的坑。除既有工作流已要求的 ansible-vault 与 avpm 二进制外零 Python 依赖；预留的环境变量缝（`MARKPOST_VAULT_CLIENT`、`MARKPOST_VAULT_DIR`）保证测试套件封闭。

## Alternatives considered

**从 ansible 导入 `VaultLib`。** 落选：其 cipher 委托给 `cryptography` 包（编译 wheel），把一个 40 行的胶水工具绑上 ansible-core 的依赖链与导入机制。

**内嵌纯 Python AES-256-CTR 核心（标准库密码学）。** 在发现 `read_data('-')` 后落选：为省一次子进程而实现 FIPS-197 AES（约 200 行自有代码）并不值得，CLI 本身已提供字节精确的 stdin/stdout 原语。

**临时文件中转 `view`/`decrypt`。** 双重落选：优雅性与必要性——stdin 路径（`read_data('-')`）使中间文件毫无必要，且 0600 文件仍把密文留在磁盘上。

## Consequences

`ansible.cfg`、`docs/backup.md`、`docs/monitoring.md`（双语）中的手写配方改为教授本工具；`check` 可入 CI（任一解密失败即退出码 1）并按环境报告通过数。对 ansible-core 未来升级的兼容性由 `scripts/vault_test.py` 钉住：以真实二进制为 oracle 的 roundtrip 用例（经 `MARKPOST_VAULT_CLIENT`/`MARKPOST_VAULT_DIR` 封闭）、带/不带尾换行值的字节保真用例、重复块与损坏块的结构完整性用例。`get` 仍是唯一输出明文的命令；不加 `--quiet` 时先在 stderr 警告，其 stdout 字节精确（`decrypt --output=-`），因此只要不把值回贴进后续命令行，值流入消费者就不会落进 shell history。
