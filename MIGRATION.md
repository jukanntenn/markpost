# SQLite 驱动迁移说明

## 概述

项目已从使用 CGO 依赖的 `github.com/mattn/go-sqlite3` 迁移到纯 Go 实现的 `modernc.org/sqlite` 驱动。

## 迁移原因

### 使用 CGO 依赖驱动的问题：

1. **构建复杂**：需要 C 编译器和相关开发库
2. **跨平台困难**：不同平台需要不同的编译环境
3. **Docker 镜像体积大**：需要安装额外的系统依赖
4. **部署复杂**：目标环境需要相应的 C 库

### 纯 Go 驱动的优势：

1. **构建简单**：只需要 Go 环境，无需 C 编译器
2. **跨平台编译**：支持轻松的交叉编译
3. **Docker 镜像更小**：无需额外系统依赖
4. **部署简便**：单一二进制文件，无外部依赖

## 技术变更

### 依赖变更

**之前：**

```go
import _ "github.com/mattn/go-sqlite3"
// 使用驱动名称: "sqlite3"
db, err := sql.Open("sqlite3", "./data/db.sqlite3")
```

**现在：**

```go
import _ "modernc.org/sqlite"
// 使用驱动名称: "sqlite"
db, err := sql.Open("sqlite", "./data/db.sqlite3")
```

⚠️ **重要提醒**：驱动名称必须从 `"sqlite3"` 改为 `"sqlite"`

### 构建变更

**之前：**

```bash
CGO_ENABLED=1 go build -o markpost .
```

**现在：**

```bash
CGO_ENABLED=0 go build -o markpost .
```

### Dockerfile 变更

**构建阶段简化：**

```dockerfile
# 之前需要：
RUN apk add --no-cache git gcc musl-dev sqlite-dev

# 现在只需要：
RUN apk add --no-cache git
```

**运行时镜像简化：**

```dockerfile
# 之前需要：
RUN apk --no-cache add ca-certificates sqlite wget

# 现在只需要：
RUN apk --no-cache add ca-certificates wget
```

## 兼容性

### API 兼容性

- ✅ 完全兼容 `database/sql` 标准接口
- ✅ SQL 语法完全兼容 SQLite
- ✅ 现有数据库文件无需迁移
- ✅ 所有功能保持不变

### 性能对比

- 🟡 启动速度：略有提升（无需加载 C 库）
- 🟡 运行性能：基本相当
- ✅ 内存使用：略有减少
- ✅ 并发性能：保持相同（WAL 模式）

## 迁移步骤

### 1. 更新依赖

```bash
# 删除旧的 go.sum
rm go.sum

# 更新 go.mod（已完成）
# 重新下载依赖
go mod tidy
```

### 2. 重新构建

```bash
# 本地构建
CGO_ENABLED=0 go build -o markpost .

# Docker 构建
docker-compose build --no-cache
```

### 3. 测试验证

```bash
# 启动服务
./markpost
# 或
docker-compose up -d

# 测试 API
curl http://localhost:8080/health
```

## 验证清单

- [ ] 项目能够正常编译
- [ ] 数据库连接正常
- [ ] API 接口功能正常
- [ ] 数据读写正常
- [ ] Docker 构建成功
- [ ] 容器运行正常

## 回滚方案

如果遇到问题，可以快速回滚到 CGO 版本：

```bash
# 1. 恢复 go.mod
git checkout go.mod

# 2. 恢复 db.go 导入
sed -i 's|modernc.org/sqlite|github.com/mattn/go-sqlite3|g' db.go

# 3. 恢复 Dockerfile
git checkout Dockerfile

# 4. 恢复 Makefile
git checkout Makefile

# 5. 重新构建
go mod tidy
```

## 常见问题

### Q: 遇到 "sql: unknown driver \"sqlite3\"" 错误怎么办？

A: 这是因为驱动名称没有更新。请确保：

1. 导入了 `_ "modernc.org/sqlite"`
2. 使用 `sql.Open("sqlite", "...")` 而不是 `sql.Open("sqlite3", "...")`

### Q: 数据库文件是否需要迁移？

A: 不需要。`modernc.org/sqlite` 完全兼容 SQLite 文件格式，现有数据库文件可以直接使用。

### Q: 性能是否会下降？

A: 基本不会。纯 Go 实现的性能与 CGO 版本基本相当，在某些场景下甚至更好。

### Q: 是否支持所有 SQLite 功能？

A: 是的。`modernc.org/sqlite` 是 SQLite 的完整 Go 移植，支持所有标准功能。

### Q: 交叉编译是否更容易？

A: 是的。无 CGO 依赖使得交叉编译变得非常简单，可以轻松编译到不同平台。

## 总结

这次迁移带来的主要好处：

1. **简化构建**：不再需要 C 编译器和系统库
2. **减小镜像**：Docker 镜像体积显著减小
3. **提高可移植性**：单一二进制文件，无外部依赖
4. **简化部署**：部署过程更加简单可靠
5. **改善开发体验**：本地开发环境搭建更容易

迁移过程保持了完全的向后兼容性，现有的数据和功能都不受影响。
