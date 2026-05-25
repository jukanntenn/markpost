# Deployment Guide

## Docker Quick Start

### Single Container

The simplest way to run Markpost is with a single Docker container using SQLite:

```bash
docker run -d \
  --name markpost \
  -p 7330:7330 \
  -e MARKPOST_JWT__ACCESS_SIGNING_KEY="your-secret-access-key-min-32-chars" \
  -e MARKPOST_JWT__REFRESH_SIGNING_KEY="your-secret-refresh-key-min-32-chars" \
  -v markpost-data:/app/data \
  jukanntenn/markpost:latest
```

SQLite data is stored in `/app/data/` inside the container. Mount a volume to persist data across restarts.

### Docker Compose

```yaml
services:
  markpost:
    image: jukanntenn/markpost:latest
    ports:
      - "7330:7330"
    environment:
      MARKPOST_SERVER__HOST: "0.0.0.0"
      MARKPOST_JWT__ACCESS_SIGNING_KEY: "your-secret-access-key"
      MARKPOST_JWT__REFRESH_SIGNING_KEY: "your-secret-refresh-key"
    volumes:
      - markpost-data:/app/data

volumes:
  markpost-data:
```

## Configuration

Markpost is configured via TOML file, environment variables, or both. See `backend/config.example.toml` for the full reference.

### Environment Variable Overrides

All config values can be set via environment variables with the `MARKPOST_` prefix and `__` for nested keys:

| Variable | Description | Default |
|----------|-------------|---------|
| `MARKPOST_SERVER__HOST` | Bind address | `127.0.0.1` |
| `MARKPOST_SERVER__PORT` | Listen port | `7330` |
| `MARKPOST_SERVER__PUBLIC_URL` | Public-facing URL | `http://{host}:{port}` |
| `MARKPOST_DB__DRIVER` | Database driver (`sqlite`, `postgresql`) | `sqlite` |
| `MARKPOST_DB__DSN` | Database connection string | `file:./data/markpost.db?_foreign_keys=on&_journal_mode=WAL` |
| `MARKPOST_ADMIN__INITIAL_USERNAME` | Initial admin username | `markpost` |
| `MARKPOST_ADMIN__INITIAL_PASSWORD` | Initial admin password | `markpost` |
| `MARKPOST_JWT__ACCESS_SIGNING_KEY` | Access token signing key (required in production) | (empty) |
| `MARKPOST_JWT__REFRESH_SIGNING_KEY` | Refresh token signing key (required in production) | (empty) |
| `MARKPOST_JWT__ACCESS_TOKEN_EXPIRE` | Access token lifetime | `24h` |
| `MARKPOST_JWT__REFRESH_TOKEN_EXPIRE` | Refresh token lifetime | `720h` |
| `MARKPOST_CORS__ALLOW_ORIGINS` | CORS allowed origins | `["*"]` |
| `MARKPOST_RATELIMIT__PER_SECOND` | Requests per second per IP | Unlimited |
| `MARKPOST_RATELIMIT__BURST` | Burst size | Unlimited |
| `MARKPOST_SERVER__TRUSTED_PROXIES` | Trusted reverse proxy IPs | `["127.0.0.1", "::1"]` |
| `MARKPOST_OAUTH__GITHUB__CLIENT_ID` | GitHub OAuth App Client ID | `""` (disabled) |
| `MARKPOST_OAUTH__GITHUB__CLIENT_SECRET` | GitHub OAuth App Client Secret | `""` (disabled) |
| `MARKPOST_OAUTH__GITHUB__REDIRECT_URL` | GitHub OAuth redirect URL | `""` |
| `MARKPOST_DEBUG` | Enable debug mode (Swagger UI) | `false` |

### TOML Configuration File

Create a `markpost.toml` file and mount it or specify via `-c` flag:

```bash
docker run -d \
  -v ./markpost.toml:/app/markpost.toml \
  jukanntenn/markpost:latest -c /app/markpost.toml
```

## Database Options

### SQLite (Default)

Zero configuration. Data is stored in a single file at `./data/markpost.db`. Suitable for single-instance deployments with moderate traffic.

- WAL mode enabled by default for concurrent reads
- Foreign keys enforced
- No separate database process needed

### PostgreSQL

For production deployments with higher traffic or multiple instances:

```bash
MARKPOST_DB__DRIVER=postgresql
MARKPOST_DB__DSN="host=db.example.com user=markpost password=secret dbname=markpost sslmode=require"
```

Docker Compose example with PostgreSQL:

```yaml
services:
  db:
    image: postgres:16
    environment:
      POSTGRES_DB: markpost
      POSTGRES_USER: markpost
      POSTGRES_PASSWORD: secret
    volumes:
      - pg-data:/var/lib/postgresql/data

  markpost:
    image: jukanntenn/markpost:latest
    depends_on:
      - db
    environment:
      MARKPOST_SERVER__HOST: "0.0.0.0"
      MARKPOST_DB__DRIVER: "postgresql"
      MARKPOST_DB__DSN: "host=db user=markpost password=secret dbname=markpost sslmode=disable"
      MARKPOST_JWT__ACCESS_SIGNING_KEY: "your-secret-access-key"
      MARKPOST_JWT__REFRESH_SIGNING_KEY: "your-secret-refresh-key"
    ports:
      - "7330:7330"

volumes:
  pg-data:
```

## Reverse Proxy

### Nginx

```nginx
server {
    listen 443 ssl http2;
    server_name markpost.example.com;

    ssl_certificate /etc/letsencrypt/live/markpost.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/markpost.example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:7330;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Caddy

```
markpost.example.com {
    reverse_proxy localhost:7330
}
```

When deploying behind a reverse proxy, set `MARKPOST_SERVER__TRUSTED_PROXIES` and `MARKPOST_SERVER__PUBLIC_URL`:

```bash
MARKPOST_SERVER__TRUSTED_PROXIES='["127.0.0.1", "::1"]'
MARKPOST_SERVER__PUBLIC_URL="https://markpost.example.com"
```

## Ansible Deployment

Automated deployment playbooks are available in `devops/ansible/`:

- `dev.yml` — Development/staging environment
- `staging.yml` — Staging environment

Run with:

```bash
ansible-playbook devops/ansible/dev.yml -i devops/ansible/hosts.yml
```

## Security Checklist

Before deploying to production:

- [ ] Change `MARKPOST_ADMIN__INITIAL_USERNAME` and `MARKPOST_ADMIN__INITIAL_PASSWORD`
- [ ] Set strong JWT signing keys (≥ 32 characters, different for access and refresh)
- [ ] Restrict `MARKPOST_CORS__ALLOW_ORIGINS` to your frontend domain
- [ ] Configure rate limiting (`MARKPOST_RATELIMIT__PER_SECOND`, `MARKPOST_RATELIMIT__BURST`)
- [ ] Set `MARKPOST_SERVER__PUBLIC_URL` to your actual URL
- [ ] Use HTTPS via a reverse proxy
- [ ] Set `MARKPOST_DEBUG=false`
