# markpost Docker 部署指南

## 概述

markpost 项目已经完全 Docker 化，支持一键部署。项目使用纯 Go 构建（无 CGO 依赖），采用多阶段构建优化镜像大小，并提供完整的数据持久化方案。

## 文件说明

- **Dockerfile**: 多阶段构建的镜像定义文件
- **docker-compose.yml**: 服务编排配置文件
- **deploy.sh**: 自动化部署脚本
- **.dockerignore**: Docker 构建忽略文件

## 快速部署

### 方法一：使用部署脚本（推荐）

```bash
# 给脚本执行权限（如果还没有）
chmod +x deploy.sh

# 执行部署
./deploy.sh
```

### 方法二：手动部署

```bash
# 1. 创建数据目录
mkdir -p ./data

# 2. 构建并启动服务
docker-compose up --build -d

# 3. 查看日志
docker-compose logs -f markpost

# 4. 检查服务状态
docker-compose ps
```

## 常用命令

### 查看服务状态

```bash
docker-compose ps
```

### 查看实时日志

```bash
docker-compose logs -f markpost
```

### 重启服务

```bash
docker-compose restart markpost
```

### 停止服务

```bash
docker-compose stop
```

### 完全停止并删除容器

```bash
docker-compose down
```

### 重新构建镜像

```bash
docker-compose build --no-cache
docker-compose up -d
```

## 配置管理

### 修改配置文件

1. 直接编辑 `markpost.toml` 文件
2. 重启服务使配置生效：
   ```bash
   docker-compose restart markpost
   ```

### 查看当前配置

```bash
cat markpost.toml
```

## 数据管理

### 数据持久化

- 数据库文件保存在 `./data/db.sqlite3`
- 即使删除容器，数据也会保留
- 可以随时备份 `./data` 目录

### 备份数据

```bash
# 创建备份
tar -czf markpost-backup-$(date +%Y%m%d).tar.gz data/

# 恢复备份（停止服务后）
docker-compose down
tar -xzf markpost-backup-YYYYMMDD.tar.gz
docker-compose up -d
```

### 重置数据

```bash
# 停止服务
docker-compose down

# 删除数据文件
rm -rf ./data

# 重新启动（会创建新的数据库）
docker-compose up -d
```

## 健康检查

项目内置健康检查端点：

```bash
# 检查服务健康状态
curl http://localhost:8080/health

# 预期响应
{
  "status": "ok",
  "message": "markpost is running"
}
```

## 获取 post_key

服务启动时会自动创建 admin 用户并生成 post_key：

```bash
# 查看启动日志获取 post_key
docker-compose logs markpost | grep "post_key:"

# 或查看完整日志
docker-compose logs markpost
```

## 测试 API

获取 post_key 后，可以测试 API：

```bash
# 替换 YOUR_POST_KEY 为实际的 post_key
curl -X POST http://localhost:8080/YOUR_POST_KEY \
  -H "Content-Type: application/json" \
  -d '{
    "title": "测试标题",
    "body": "# 这是一个测试\n\n这是一些 **markdown** 内容。"
  }'

# 使用返回的 ID 获取内容（会转换为 HTML）
curl http://localhost:8080/RETURNED_ID
```

## 故障排除

### 服务无法启动

1. 检查日志：

   ```bash
   docker-compose logs markpost
   ```

2. 检查端口占用：

   ```bash
   netstat -tulpn | grep 8080
   ```

3. 重新构建镜像：
   ```bash
   docker-compose build --no-cache
   ```

### 数据库问题

1. 检查数据目录权限：

   ```bash
   ls -la ./data
   ```

2. 重新创建数据目录：
   ```bash
   docker-compose down
   sudo rm -rf ./data
   mkdir -p ./data
   docker-compose up -d
   ```

### 配置问题

1. 验证配置文件格式：

   ```bash
   cat markpost.toml
   ```

2. 恢复默认配置：
   ```bash
   git checkout markpost.toml
   docker-compose restart markpost
   ```

## 性能优化

### 镜像优化

- 使用多阶段构建减小镜像大小
- 基于 Alpine Linux 最小化基础镜像
- 纯 Go 构建，无 CGO 依赖，简化构建过程
- 只复制必要的文件

### 运行时优化

- 使用非 root 用户运行
- 启用 SQLite WAL 模式提高并发性能
- 配置合适的限流参数

## 安全建议

1. **网络安全**: 在生产环境中使用反向代理（如 Nginx）
2. **访问控制**: 保护好 post_key，定期更换
3. **数据备份**: 定期备份数据目录
4. **监控**: 启用日志监控和健康检查
5. **更新**: 定期更新基础镜像和依赖包

## 扩展部署

### 使用外部数据库

如需要更高性能，可以考虑使用外部 PostgreSQL 或 MySQL 数据库。

### 负载均衡

可以运行多个实例并配置负载均衡器。

### 监控集成

可以集成 Prometheus、Grafana 等监控工具。
