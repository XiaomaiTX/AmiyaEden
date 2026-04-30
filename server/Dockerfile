# ---- Build Stage ----
FROM golang:1.26.2-alpine AS builder

WORKDIR /app

# 安装必要的构建工具
RUN apk add --no-cache git

# 缓存依赖层
COPY go.mod go.sum ./
RUN go mod download

# 复制源码并构建
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./main.go

# ---- Runtime Stage ----
FROM alpine:3.20

WORKDIR /app

# 安装时区数据和 ca 证书（用于 HTTPS 请求 EVE SSO 等）
RUN apk add --no-cache tzdata ca-certificates \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

# 从构建阶段复制二进制
COPY --from=builder /app/server .

# 创建日志目录
RUN mkdir -p logs

EXPOSE 8080

CMD ["./server"]
