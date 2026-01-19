# markpost

English | [简体中文](README_zh.md)

A simple Go web project that provides APIs to upload markdown content and query converted HTML.

## Deployment

### Docker Command

```bash
docker run -d \
  --name markpost \
  -p 7330:7330 \
  -v ./data:/app/data \
  # -v ./config.toml:/app/config.toml:ro \
  --restart unless-stopped \
  jukanntenn/markpost:latest
```

### Docker Compose

1. Create a `docker-compose.yml` file with following content:

   ```yaml
   services:
     markpost:
       image: jukanntenn/markpost:latest
       container_name: markpost
       ports:
         - "7330:7330"
       volumes:
         - ./data:/app/data
         # - ./config.toml:/app/config.toml:ro
       restart: unless-stopped
   ```

2. Start service:

   ```bash
   docker-compose up -d
   ```

### Go Build

1. Prerequisites: Go 1.24.0 or higher

2. Clone repository and enter project directory

3. Build frontend assets (required for Web UI under `/ui`):

   ```bash
   cd frontend
   pnpm install
   pnpm build
   ```

4. Build backend:

   ```bash
   cd ../backend
   go mod download
   go build -o markpost .
   ```

5. Run server:

   ```bash
   ./markpost serve -c ./config.toml
   ```

## Configuration

The project reads `config.toml` configuration file. See [config.example.toml](backend/config.example.toml) for details.

By default, it looks for `./config.toml` in the process working directory, or you can pass `-c/--config` to specify a path. Configuration can also be overridden via environment variables with prefix `MARKPOST__` (for example: `MARKPOST__JWT__ACCESS_SIGNING_KEY`).

When using Docker, you can mount configuration file:

```bash
docker run -d \
  --name markpost \
  -p 7330:7330 \
  -v ./data:/app/data \
  -v ./config.toml:/app/config.toml:ro \
  --restart unless-stopped \
  jukanntenn/markpost:latest
```

For Docker Compose, add configuration file mount in `docker-compose.yml`:

```yaml
volumes:
  - ./data:/app/data
  - ./config.toml:/app/config.toml:ro
```

### Obtaining post_key

You can obtain the post_key from the web UI:

1. Access the web UI at: `http://127.0.0.1:7330`
2. Use the initial credentials:
   - **Username**: configurable via `admin.initial_username` or env `MARKPOST__ADMIN__INITIAL_USERNAME` (defaults to `markpost`)
   - **Password**: configurable via `admin.initial_password` or env `MARKPOST__ADMIN__INITIAL_PASSWORD` (defaults to `markpost`)
3. After successful login, your post_key will be displayed on the dashboard

## APIs

### Upload Content

**POST** `/:post_key`

Upload markdown content using a valid post_key. The system generates a post_key for the initial admin user on first startup; retrieve it from the web UI after login.

Request body:

```json
{
  "title": "Article Title", // optional
  "body": "markdown content"
}
```

Response:

```json
{
  "id": "generated-nanoid"
}
```

Error response:

```json
{
  "error": "Error message"
}
```

### Get Content

**GET** `/:id`

Retrieve uploaded content by ID, returning a rendered HTML page (not JSON format).

- Success: Returns a complete HTML page containing converted markdown content
- Failure: Returns an error page
