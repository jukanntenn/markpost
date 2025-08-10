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

项目会读取 `config.toml` 配置文件，详情请参考 [config.example.toml](config.example.toml)。

使用 Docker 时，可挂载配置文件：

```bash
docker run -d \
  --name markpost \
  -p 8080:8080 \
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

有两种方式可以获取 post_key：

#### 方式 1：从启动日志获取

启动服务后，查看启动日志以获取生成的 post_key。日志中会包含类似以下内容：

```text
created markpost user with post_key: abc12345
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

#### 方式 2：通过 Web 管理控制台

您也可以通过登录 Web 管理控制台获取 post_key：

1. 访问 Web 界面：`http://127.0.0.1:7330`
2. 使用默认凭据：
   - **用户名**：`markpost`
   - **密码**：`markpost`
3. 成功登录后，您的 post_key 将显示在仪表板上

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
