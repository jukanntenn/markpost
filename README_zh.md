# markpost

简体中文 | [English](README.md)

一个简单的 Go Web 项目，提供上传 Markdown 内容和查询转换后 HTML 的 API 接口。

## 部署

### Docker 命令

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

1. 创建 `docker-compose.yml` 文件，内容如下：

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

2. 启动服务：

   ```bash
   docker-compose up -d
   ```

### Go 编译

1. 环境要求：Go 1.24.0 或更高版本

2. 克隆仓库并进入项目目录

3. 构建前端静态资源（Web UI `/ui` 需要）：

   ```bash
   cd frontend
   pnpm install
   pnpm build
   ```

4. 编译后端：

   ```bash
   cd ../backend
   go mod download
   go build -o markpost .
   ```

5. 启动服务：

   ```bash
   ./markpost serve -c ./config.toml
   ```

## 配置

项目会读取 `config.toml` 配置文件，详情请参考 [config.example.toml](backend/config.example.toml)。

默认会在进程工作目录下查找 `./config.toml`，也可以通过 `-c/--config` 指定路径。配置支持通过环境变量覆盖，前缀为 `MARKPOST__`（例如：`MARKPOST__JWT__ACCESS_SIGNING_KEY`）。

使用 Docker 时，可挂载配置文件：

```bash
docker run -d \
  --name markpost \
  -p 7330:7330 \
  -v ./data:/app/data \
  -v ./config.toml:/app/config.toml:ro \
  --restart unless-stopped \
  jukanntenn/markpost:latest
```

使用 Docker Compose 时，在 `docker-compose.yml` 中添加配置文件挂载：

```yaml
volumes:
  - ./data:/app/data
  - ./config.toml:/app/config.toml:ro
```

### 获取 post_key

您可以通过登录 Web 管理控制台获取 post_key：

1. 访问 Web 界面：`http://127.0.0.1:7330`
2. 使用初始凭据：
   - **用户名**：可通过 `admin.initial_username` 或环境变量 `MARKPOST__ADMIN__INITIAL_USERNAME` 指定（默认 `markpost`）
   - **密码**：可通过 `admin.initial_password` 或环境变量 `MARKPOST__ADMIN__INITIAL_PASSWORD` 指定（默认 `markpost`）
3. 成功登录后，您的 post_key 将显示在仪表板上

## API 接口

### 上传内容

**POST** `/:post_key`

使用有效的 post_key 上传 markdown 内容。系统会在首次启动时为初始管理员用户生成 post_key，可在登录 Web 管理控制台后获取。

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
  "id": "生成的-nanoid"
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
