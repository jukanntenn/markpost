# DevOps Specification

This document defines the rules and mechanisms for the project's development
environment and remote deployment pipeline. Any project following this spec
reproduces the same mechanisms with project-specific details while adhering to
the constraints below.

---

## 1. Locality

All devops-related configuration lives under a single top-level directory.
No devops files exist elsewhere in the project.

The local development setup and remote deployment configuration are co-located
within this directory but remain independent — changes to one must not require
changes to the other.

---

## 2. Development Environment Script

### 2.1 Language and Dependencies

Python, standard library only. No third-party packages.

### 2.2 Interface

Two commands: **start** (default) and **stop**. No per-service control — the
script manages the entire stack as a single unit.

The script runs from the **project root**, never from within its own directory.

### 2.3 Execution Model

All services start **in the background**. The script starts everything, waits
for readiness, prints status, and exits. It must never block.

Frontend processes (and any non-containerized services) are tracked via PID
files. Logs are written to files, not stdout.

### 2.4 Constraints

- **No runtime `.env` generation.** The script must not create or modify
  environment files.
- **No project-internal config generation.** The script must not create files
  inside the application source tree (e.g. `frontend/.env.local`). Application
  code should use sensible defaults instead.
- **No branch-aware behavior.** No worktree detection, port offset calculation,
  or any logic that changes behavior based on the current git state.
- **Constants, not configuration files.** Ports and other values used by the
  script are module-level constants matching the application code defaults.
  No external `.env` file is loaded.

### 2.5 Stop Behavior

Stop must be idempotent. It tears down Docker services, kills tracked frontend
processes, and cleans up PID files. If nothing is running, it exits silently.

---

## 3. Configuration

Development environment configuration uses **inline values**, not external
files. The dev script and Docker Compose file contain all necessary values
directly as constants or hardcoded entries.

Rules:

- The dev script uses module-level constants for ports and URLs, matching
  the application code defaults.
- The Docker Compose file hardcodes all environment values inline (see §4).
- No `.env` file is loaded or required by any development tooling.
- Application code provides sensible defaults so that manual startup requires
  minimal configuration (only security-sensitive values via a config file).

---

## 4. Docker Compose

The development compose file uses the **project root** as its build context.
Dockerfile paths and volume mounts are expressed relative to the project root.

All environment values are **hardcoded inline**. No `${VAR:-default}` variable
substitution or external `.env` file is used. Only values that differ from
application code defaults are set (e.g. `SERVER_HOST=0.0.0.0` for Docker,
`DB_DRIVER=postgresql` for the containerized database).

Constraints:

- No dynamic port assignment or offset logic.
- No runtime file generation or templating.
- No variable substitution — all values are literal.
- Resource naming uses a fixed project name.

---

## 5. Ansible — Remote Deployment

### 5.1 Environment-Based Organization

Deployment is organized by **environment**. Each environment owns three things:

1. A **playbook** — defines the deployment steps for that environment.
2. **Templates** — configuration files rendered for that environment.
3. **Secrets** — encrypted variables specific to that environment.

### 5.2 Environment Independence

Environments are fully independent:

- Each environment has its own secrets. No shared vault files.
- Each environment has its own templates. If two environments have identical
  templates, **duplicate the files** rather than sharing them. Divergence will
  happen; duplication is cheaper than abstraction.
- Playbooks load secrets via `vars_files`, never inline vault strings.

### 5.3 Inventory

The inventory uses a **flat structure**. All hosts are listed at the top level
without group nesting. Per-host non-sensitive variables (user, home path) are
kept in per-host variable files.

### 5.4 Lifecycle

Retired hosts, playbooks, and templates are **deleted immediately**. No dead
configuration is kept for potential future use.

---

## 6. Recommended Layout

This project uses the following concrete organization. New projects or
environments should follow this pattern unless there is a specific reason to
deviate.

```
devops/
├── docker-compose.yml          # Development Docker Compose file
├── backend.Dockerfile          # Dev-stage Dockerfiles
├── dev.py                      # Development environment script
└── ansible/
    ├── dev.yml                 # One playbook per environment
    ├── staging.yml
    ├── hosts.yml               # Flat inventory
    ├── host_vars/
    │   ├── fn.yml
    │   └── oect.yml
    ├── templates/
    │   ├── dev/
    │   │   ├── docker-compose.yml.j2
    │   │   └── config.toml.j2
    │   └── staging/
    │       ├── docker-compose.yml.j2
    │       └── config.toml.j2
    └── vars/
        ├── dev/
        │   └── vault.yml
        └── staging/
            └── vault.yml
```

### 6.1 Dev Script — Start Flow

```
python devops/dev.py start
```

1. `docker compose -f devops/docker-compose.yml up -d --build`.
3. Poll `http://localhost:<BACKEND_PORT>/health` until 200.
4. If `frontend/node_modules/` is missing, run `pnpm install`.
5. Launch `pnpm dev` in a new session group. Write PID to
   `devops/frontend.pid`, redirect output to `devops/frontend.log`.
6. Poll `http://localhost:<FRONTEND_PORT>` until reachable.
7. Print service URLs, admin credentials, and stop command. Exit.

### 6.2 Dev Script — Stop Flow

```
python devops/dev.py stop
```

1. `docker compose -f devops/docker-compose.yml down`.
2. Read PID from `devops/frontend.pid`, kill process group (SIGTERM → SIGKILL).
3. Fallback: kill any process on the frontend port via `lsof`.
4. Remove PID file.

### 6.3 Ansible Playbook Pattern

Each playbook follows this structure:

```yaml
- name: Deploy <project> (<env>)
  hosts: <host>
  user: "{{ user }}"
  gather_facts: no
  vars_files:
    - vars/<env>/vault.yml
  vars:
    app_name: <project>
    apps_path: "{{ home }}/docker"
    app_path: "{{ apps_path }}/{{ app_name }}"
    compose_file: "{{ app_path }}/docker-compose.yml"
    config_file: "{{ app_path }}/config.toml"
  tasks:
    - Ensure directories exist
    - Render docker-compose.yml from template
    - Render config.toml from template
    - Stop containers (absent)
    - Pull images
    - Start containers
```

---

## 7. Checklist — Adding a New Environment

1. Create a playbook for the environment.
2. Add the host to the inventory.
3. Add per-host variables (user, paths).
4. Copy templates from an existing environment and modify as needed.
5. Create and encrypt a secrets file for the environment.
