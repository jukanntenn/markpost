# markpost

简体中文 | [English](README.md)

一个简单的 Go Web 项目，提供上传 Markdown 内容和查询转换后 HTML 的 API 接口。

## 部署

### Docker 命令

```bash
docker run -d \
  --name markpost \
  -p 8080:8080 \
  -v ./data:/app/data \
  --restart unless-stopped \
  jukanntenn/markpost:latest
```

### Docker Compose

1. 创建 `docker-compose.yml` 文件，内容如下：
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

2. 启动服务：
   ```bash
   docker-compose up -d
   ```

### Go 编译

1. 环境要求：Go 1.23.0 或更高版本

2. 克隆仓库并进入项目目录

3. 安装依赖：
   ```bash
   go mod download
   ```

4. 编译项目：
   ```bash
   go build -o markpost .
   ```

5. 运行可执行文件：
   ```bash
   ./markpost
   ```

## 配置

项目会读取 `markpost.toml` 配置文件，可配置以下参数：

```toml
# 标题最大字节数
TITLE_MAX_SIZE = 200

# 正文最大字节数
BODY_MAX_SIZE = 1048576

# API 速率限制，每分钟允许的请求数
API_RATE_LIMIT = 60
```

使用 Docker 时，可挂载配置文件：
```bash
docker run -d \
  --name markpost \
  -p 8080:8080 \
  -v ./data:/app/data \
  -v ./markpost.toml:/app/markpost.toml:ro \
  --restart unless-stopped \
  jukanntenn/markpost:latest
```

使用 Docker Compose 时，在 `docker-compose.yml` 中添加配置文件挂载：
```yaml
volumes:
  - ./data:/app/data
  - ./markpost.toml:/app/markpost.toml:ro
```

### 获取 post_key

启动服务后，查看启动日志获取生成的 post_key。日志中会包含类似以下内容：
```
created admin user with post_key: abc12345
```

Docker 容器查看日志：
```bash
docker logs markpost
```

Docker Compose 查看日志：
```bash
docker-compose logs markpost
```

直接执行时，日志会显示在控制台。

## API 接口

### 上传内容

**POST** `/:post_key`

使用有效的 post_key 上传 markdown 内容。系统会在启动时为管理员用户自动生成 post_key，请查看启动日志获取。

请求体：
```json
{
  "title": "文章标题", // 可选
  "body": "markdown内容"
}
```

响应：
```json
{
  "id": "生成的-nanoid",
}
```

错误响应：
```json
{
  "error": "错误信息"
}
```

### 获取内容

**GET** `/:id`

通过 ID 获取已上传的内容，返回渲染后的 HTML 页面（非 JSON 格式）。

- 成功：返回完整的 HTML 页面，包含转换后的 markdown 内容
- 失败：返回错误页面
