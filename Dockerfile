# 多阶段构建，第一阶段：构建应用程序
FROM golang:1.21-alpine AS builder

# 安装必要的工具
RUN apk add --no-cache git

# 设置工作目录
WORKDIR /app

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用程序
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o markpost .

# 第二阶段：创建最终运行镜像
FROM alpine:latest

# 安装运行时依赖
RUN apk --no-cache add ca-certificates wget

# 创建非root用户
RUN addgroup -S markpost && adduser -S markpost -G markpost

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/markpost .

# 复制配置文件
COPY markpost.toml .

# 创建数据目录并设置权限
RUN mkdir -p /app/data

# 切换到非root用户
# USER markpost

# 暴露端口
EXPOSE 8080

# 启动命令
CMD ["./markpost"] 