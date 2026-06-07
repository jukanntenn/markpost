# Deployment Guide

## Docker Quick Start

### Single Container

For quick evaluation only — use Docker Compose for long-term deployments.

```bash
docker run -d \
  --name markpost \
  -p 7157:7157 \
  -e MARKPOST_JWT__ACCESS_SIGNING_KEY="your-secret-access-key-min-32-chars" \
  -e MARKPOST_JWT__REFRESH_SIGNING_KEY="your-secret-refresh-key-min-32-chars" \
  -v markpost-data:/app/data \
  jukanntenn/markpost:latest
```

### Docker Compose (Recommended)

1. Create a project directory and download the example config:

   ```bash
   mkdir -p ~/docker/markpost && cd ~/docker/markpost
   curl -fsSL https://raw.githubusercontent.com/jukanntenn/markpost/refs/heads/main/backend/config.example.toml -o config.toml
   ```

2. Edit `config.toml` — the following values **must** be changed:

   ```toml
   [server]
   host = "127.0.0.1"

   [db]
   driver = "postgresql"
   dsn = "host=db user=postgres password=postgres dbname=markpost sslmode=disable"

   [jwt]
   access_signing_key = "your-secret-access-key-min-32-chars"
   refresh_signing_key = "your-secret-refresh-key-min-32-chars"

   [admin]
   initial_password = "your-secret-admin-password"
   ```

3. Create `docker-compose.yml`:

   ```yaml
   services:
     markpost:
       image: jukanntenn/markpost:latest
       ports:
         - "7157:7157"
       volumes:
         - ./config.toml:/app/markpost.toml:ro
       depends_on:
         db:
           condition: service_healthy
       healthcheck:
         test:
           [
             "CMD",
             "wget",
             "--no-verbose",
             "--tries=1",
             "--spider",
             "http://127.0.0.1:7157/api/v1/health",
           ]
         interval: 10s
         timeout: 5s
         retries: 3
         start_period: 30s
       restart: always

     db:
       image: postgres:17-alpine
       volumes:
         - markpost-db:/var/lib/postgresql/data
         - ./config.toml:/app/markpost.toml:ro
       environment:
         - POSTGRES_DB=markpost
         - POSTGRES_USER=postgres
         - POSTGRES_PASSWORD=postgres
       healthcheck:
         test: ["CMD-SHELL", "pg_isready -U postgres"]
         interval: 10s
         timeout: 5s
         retries: 5
       restart: always

   volumes:
     markpost-db:
   ```

## Container Architecture

Markpost runs as a single container with three internal services managed by [s6-overlay](https://github.com/just-containers/s6-overlay):

- **Caddy** — reverse proxy on port 7157 (external entry point)
- **Go backend** — API server on `127.0.0.1:7330` (internal)
- **Next.js** — frontend server on `127.0.0.1:3000` (internal)

```
                    ┌──────────────────────────────────────────┐
                    │       markpost container (:7157)         │
                    │                                          │
  External ────────►│  Caddy (0.0.0.0:7157)                   │
  :7157             │    ├ /api/v1/*   ──► Go (127.0.0.1:7330)│
                    │    ├ /swagger/*  ──► Go                  │
                    │    ├ /mpk-* POST ──► Go                  │
                    │    ├ /p-* GET    ──► Go                  │
                    │    └ rest         ──► Next.js (:3000)    │
                    │                                          │
                    │  s6-overlay manages: go, nextjs, caddy   │
                    └──────────────────────────────────────────┘
```

Caddy handles TLS termination, logging, and request routing. s6-overlay ensures services start in the correct order (Go and Next.js before Caddy) and restarts crashed processes.

## Database Options

### PostgreSQL (Recommended)

Recommended for all deployments. PostgreSQL uses minimal resources even in a container — no need to worry about overhead.

### SQLite

Suitable only for resource-constrained environments (homelab, NAS). Zero configuration — data is stored at `./data/markpost.db`. Single-instance only; not recommended for moderate-to-high traffic.

`docker-compose.yml` for SQLite:

```yaml
services:
  markpost:
    image: jukanntenn/markpost:latest
    restart: always
    volumes:
      - ./data:/app/data
      - ./config.toml:/app/markpost.toml:ro
    ports:
      - "7157:7157"
    healthcheck:
      test:
        [
          "CMD",
          "wget",
          "--no-verbose",
          "--tries=1",
          "--spider",
          "http://127.0.0.1:7157/api/v1/health",
        ]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 30s
```

## Reverse Proxy

To deploy behind an external reverse proxy, forward traffic to port 7157:

```caddyfile
markpost.cc {
    reverse_proxy localhost:7157
}
```

Set these environment variables when running behind a reverse proxy:

```bash
MARKPOST_SERVER__TRUSTED_PROXIES='["127.0.0.1", "::1"]'
MARKPOST_SERVER__PUBLIC_URL="https://markpost.cc"
```

## Building the Image

```bash
python3 docker/build.py                    # Build for local platform
python3 docker/build.py --push             # Build and push multi-platform
python3 docker/build.py --platform amd64   # Build specific platform
python3 docker/build.py --tags v1.0.0      # Additional tags
```

Run `python3 docker/build.py --help` for more options.

## Ansible Automation

Automated deployment playbooks are in `devops/ansible/`. Variables are encrypted with `ansible-vault` for internal use — external users should replace `vault.yml` with their own values.

Available playbooks:

- `dev.yml` — development (with PostgreSQL)
- `staging.yml` — staging
- `production.yml` — production

Each playbook contains the exact run command in a header comment:

```bash
ansible-playbook -i devops/ansible/hosts.yml devops/ansible/dev.yml --vault-password-file ~/.ansible-vault/markpost-dev.pwd
```
