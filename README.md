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

1. Prerequisites: Go 1.23.0 or higher

2. Clone repository and enter project directory

3. Install dependencies:

   ```bash
   go mod download
   ```

4. Build project:

   ```bash
   go build -o markpost .
   ```

5. Run executable:

   ```bash
   ./markpost
   ```

## Configuration

The project reads `config.toml` configuration file, see [config.example.toml](config.example.toml) for details.

When using Docker, you can mount configuration file:

```bash
docker run -d \
  --name markpost \
  -p 8080:8080 \
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

There are two ways to obtain the post_key:

#### Method 1: From Startup Logs

After starting the service, check the startup logs to get the generated post_key. The log will contain content similar to:

```text
created markpost user with post_key: abc12345
```

For Docker containers, view logs with:

```bash
docker logs markpost
```

For Docker Compose, view logs with:

```bash
docker-compose logs markpost
```

For direct execution, the log will be displayed in the console.

#### Method 2: Via Web Management Console

You can also obtain the post_key from the web UI:

1. Access the web UI at: `http://127.0.0.1:7330`
2. Use the default credentials:
   - **Username**: `markpost`
   - **Password**: `markpost`
3. After successful login, your post_key will be displayed on the dashboard

## APIs

### Upload Content

**POST** `/:post_key`

Upload markdown content using a valid post_key. The system automatically generates a post_key for the admin user at startup; check the startup log to get it.

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
