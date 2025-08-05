# markpost

English | [简体中文](README_zh.md)

A simple Go web project that provides APIs to upload markdown content and query the converted HTML.

## Deployment

### Docker Command

```bash
docker run -d \
  --name markpost \
  -p 8080:8080 \
  -v ./data:/app/data \
  --restart unless-stopped \
  jukanntenn/markpost:latest
```

### Docker Compose

1. Create a `docker-compose.yml` file with the following content:
   ```yaml
   services:
     markpost:
       image: jukanntenn/markpost:latest
       container_name: markpost
       ports:
         - "8080:8080"
       volumes:
         - ./data:/app/data
       restart: unless-stopped
   ```

2. Start the service:
   ```bash
   docker-compose up -d
   ```

### Go Build

1. Prerequisites: Go 1.23.0 or higher

2. Clone the repository and enter the project directory

3. Install dependencies:
   ```bash
   go mod download
   ```

4. Build the project:
   ```bash
   go build -o markpost .
   ```

5. Run the executable:
   ```bash
   ./markpost
   ```

## Configuration

The project reads the `markpost.toml` configuration file, which can configure the following parameters:

```toml
# Maximum number of bytes for the title
TITLE_MAX_SIZE = 200

# Maximum number of bytes for the body
BODY_MAX_SIZE = 1048576

# API rate limit, number of allowed requests per minute
API_RATE_LIMIT = 60
```

When using Docker, you can mount the configuration file:
```bash
docker run -d \
  --name markpost \
  -p 8080:8080 \
  -v ./data:/app/data \
  -v ./markpost.toml:/app/markpost.toml:ro \
  --restart unless-stopped \
  jukanntenn/markpost:latest
```

For Docker Compose, add the configuration file mount in `docker-compose.yml`:
```yaml
volumes:
  - ./data:/app/data
  - ./markpost.toml:/app/markpost.toml:ro
```

### Obtaining post_key

After starting the service, check the startup logs to get the generated post_key. The log will contain content similar to:
```
created admin user with post_key: abc12345
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
  "id": "generated-nanoid",
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

- Success: Returns a complete HTML page containing the converted markdown content
- Failure: Returns an error page
