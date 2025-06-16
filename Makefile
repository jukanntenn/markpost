.PHONY: help build up down logs restart clean test

# 默认目标
help:
	@echo "markpost 项目 - 可用命令:"
	@echo "  make build    - 构建 Docker 镜像"
	@echo "  make up       - 启动服务"
	@echo "  make down     - 停止服务"
	@echo "  make logs     - 查看日志"
	@echo "  make restart  - 重启服务"
	@echo "  make clean    - 清理容器和镜像"
	@echo "  make test     - 测试 API"
	@echo "  make deploy   - 一键部署"

# 构建镜像
build:
	docker-compose build

# 启动服务
up:
	mkdir -p ./data
	docker-compose up -d

# 停止服务
down:
	docker-compose down

# 查看日志
logs:
	docker-compose logs -f markpost

# 重启服务
restart:
	docker-compose restart markpost

# 清理
clean:
	docker-compose down -v
	docker rmi markpost_markpost 2>/dev/null || true

# 测试健康检查
test:
	@echo "测试健康检查端点..."
	curl -s http://localhost:8080/health || echo "服务未启动或不可访问"

# 一键部署
deploy:
	./deploy.sh

# 开发模式运行
dev:
	go run .

# 安装依赖
deps:
	go mod download

# 构建二进制文件
build-binary:
	CGO_ENABLED=0 go build -o markpost .

# 显示服务状态
status:
	docker-compose ps 