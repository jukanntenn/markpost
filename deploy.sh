#!/bin/bash

# markpost 部署脚本

set -e  # 出错时退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 打印彩色信息
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 创建必要的目录
print_info "创建数据目录..."
mkdir -p ./data

# 检查 Docker 是否安装
if ! command -v docker &> /dev/null; then
    print_error "Docker 未安装，请先安装 Docker"
    exit 1
fi

# 检查 docker-compose 是否安装
if ! command -v docker-compose &> /dev/null; then
    print_error "docker-compose 未安装，请先安装 docker-compose"
    exit 1
fi

# 构建并启动服务
print_info "构建和启动 markpost 服务..."
docker-compose up --build -d

# 等待服务启动
print_info "等待服务启动..."
sleep 5

# 检查服务状态
if docker-compose ps | grep -q "Up"; then
    print_info "markpost 服务启动成功！"
    print_info "访问地址: http://localhost:8080"
    print_info "健康检查: http://localhost:8080/health"
    
    # 获取 post_key
    print_info "获取 admin 用户的 post_key..."
    docker-compose logs markpost | grep "post_key:" || print_warning "未找到 post_key，请查看容器日志"
    
    echo ""
    print_info "使用以下命令查看日志:"
    echo "  docker-compose logs -f markpost"
    echo ""
    print_info "使用以下命令停止服务:"
    echo "  docker-compose down"
else
    print_error "服务启动失败，请检查日志:"
    docker-compose logs markpost
    exit 1
fi 