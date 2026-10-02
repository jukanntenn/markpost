# MRFC: scripts/vault.py for ergonomic per-variable vault management

Status: implemented

English | [中文](2026-10-03-vault-variable-tool.zh.md)

## Problem

The documented recipe for adding a vault variable (`ansible-vault encrypt_string --vault-id ... >> vault.yml`) fails outright under the repo's own ansible.cfg: `vault_identity_list` pre-loads all three identities, `--vault-id` merely appends a fourth, and ansible demands `--encrypt-vault-id` - a trap hit on the first real use. Three further ergonomics gaps compound it: `echo`-piped values gain a trailing newline that silently breaks B2 keys; appending leaves duplicate variable blocks whose resolution depends on last-wins semantics; and verifying a decrypted value requires an ad-hoc `ansible localhost -m debug` command whose output parsing is fragile. Removing a variable is entirely manual.

## Decision

**`scripts/vault.py` - a stdlib-only CLI (`set`/`get`/`list`/`check`/`remove`) that orchestrates the ansible-vault CLI around the per-variable block shape.** Two native primitives make it temp-file-free: `ansible-vault view -` was found to decrypt stdin to stdout (`VaultEditor.read_data('-')`), and `decrypt --output=-` writes byte-exact plaintext (the `view` display layer appends a newline to values lacking one, which is ambiguous). `set` pipes values into `encrypt_string --encrypt-vault-id`, upserts the block with the file's own indentation, and cryptographically self-verifies before writing. All subprocesses run with `ANSIBLE_VAULT_IDENTITY_LIST` emptied plus one explicit `--vault-id`, so a secret encrypted under another environment's password fails instead of silently decrypting (measured: without this, the repo cfg decrypts across environments). `env` is a whitelist mapping to file path and identity label, hiding the `production`->`markpost-prod` trap. Zero Python dependencies beyond the ansible-vault and avpm binaries the existing workflow already requires; a documented env-var seam (`MARKPOST_VAULT_CLIENT`, `MARKPOST_VAULT_DIR`) keeps the test suite hermetic.

## Alternatives considered

**Import `VaultLib` from ansible.** It lost: the cipher delegates to the `cryptography` package (compiled wheels), coupling a 40-line glue tool to ansible-core's dependency chain and import machinery.

**Embed a pure-Python AES-256-CTR core (stdlib-only crypto).** It lost once `read_data('-')` was found: implementing FIPS-197 AES (~200 owned lines) to avoid one subprocess is not worth it when the CLI itself provides byte-exact stdin/stdout primitives.

**Temp-file relay into `view`/`decrypt`.** It lost on both elegance and necessity: the stdin path (`read_data('-')`) makes intermediate files unnecessary, and files at 0600 still persist the ciphertext on disk.

## Consequences

The handwritten recipes in `ansible.cfg`, `docs/backup.md`, and `docs/monitoring.md` (both languages) now teach the tool; `check` is CI-able (exit 1 on any decrypt failure) and reports per-environment pass counts. Correctness against future ansible-core upgrades is pinned by `scripts/vault_test.py`: NIST-independent roundtrip oracles against the real binary (hermetic via `MARKPOST_VAULT_CLIENT`/`MARKPOST_VAULT_DIR`), byte-fidelity cases for values with and without trailing newlines, and structural-integrity cases for duplicate and corrupted blocks. `get` remains the single plaintext-emitting command; it prints a stderr warning unless `--quiet`, and its stdout is byte-exact (`decrypt --output=-`), so values flow into consumers without landing in shell history as long as they are not pasted back into later command lines.
