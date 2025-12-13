# 多阶段构建：构建阶段
FROM golang:1.21-alpine AS builder

# 设置构建参数
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT

# 设置工作目录
WORKDIR /build

# 安装构建依赖
RUN apk add --no-cache git make ca-certificates

# 复制go mod文件（利用Docker缓存层）
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download && go mod verify

# 复制源代码
COPY . .

# 构建应用（注入版本信息）
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -extldflags '-static' \
    -X message-mirror/internal/pkg/version.Version=${VERSION} \
    -X message-mirror/internal/pkg/version.BuildTime=${BUILD_TIME} \
    -X message-mirror/internal/pkg/version.GitCommit=${GIT_COMMIT}" \
    -a -installsuffix cgo \
    -o message-mirror ./cmd/message-mirror

# 运行阶段 - 使用distroless或alpine
FROM alpine:3.19

# 安装必要的运行时依赖
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    wget \
    curl && \
    update-ca-certificates

# 设置时区（可通过环境变量覆盖）
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 标签
LABEL maintainer="message-mirror" \
      description="Message Mirror - Pluggable Message Mirroring Tool" \
      version="${VERSION}"

# 创建非root用户
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# 设置必要的目录
RUN mkdir -p /app/logs /app/data /app/config && \
    chown -R appuser:appuser /app

# 设置环境变量
ENV CONFIG_PATH=/app/config/config.yaml \
    LOG_PATH=/app/logs/message-mirror.log \
    DATA_PATH=/app/data

# 切换到非root用户
USER appuser

# 暴露HTTP服务器端口（默认8080）
EXPOSE 8080

# 健康检查（liveness探针）
HEALTHCHECK --interval=30s --timeout=3s --start-period=30s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 设置默认命令
ENTRYPOINT ["./message-mirror"]

# 默认参数（可在运行时覆盖）
CMD ["-c", "/app/config
# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 设置默认命令
ENTRYPOINT ["./message-mirror"]

# 默认参数
CMD ["-c", "/app/config.yaml"]

