.PHONY: build run test clean docker-build docker-run docker-compose-up docker-compose-down help release version

# 变量定义
IMAGE_NAME = message-mirror
VERSION ?= $(shell cat VERSION 2>/dev/null || echo "0.1.1")
BUILD_TIME = $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DOCKER_COMPOSE_FILE = docker/docker-compose.yml

# 支持的平台
PLATFORMS = linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

# 本地构建
build:
	@echo "构建message-mirror v$(VERSION)..."
	go mod download
	go build -ldflags "-X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)" -o message-mirror .

# 运行本地构建的二进制文件
run: build
	@echo "运行message-mirror..."
	./message-mirror -c config.yaml

# 运行测试
test:
	@echo "运行测试..."
	go test -v .

# 显示版本信息
version:
	@echo "版本: $(VERSION)"
	@echo "构建时间: $(BUILD_TIME)"
	@echo "Git提交: $(GIT_COMMIT)"

# 多平台编译
build-all:
	@echo "构建所有平台..."
	@mkdir -p release
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} \
		GOARCH=$${platform#*/} \
		BINARY_NAME=message-mirror-$${platform%/*}-$${platform#*/} \
		OUTPUT=release/$${BINARY_NAME}; \
		if [ "$${platform%/*}" = "windows" ]; then \
			OUTPUT=$${OUTPUT}.exe; \
		fi; \
		echo "构建 $$platform -> $$OUTPUT"; \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} \
		go build -ldflags "-X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)" \
			-o $$OUTPUT . || exit 1; \
	done
	@echo "构建完成！二进制文件在 release/ 目录"

# 发布版本：构建、打包、创建标签
release: clean test build-all
	@echo "准备发布 v$(VERSION)..."
	@mkdir -p release
	@echo "创建发布包..."
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} \
		GOARCH=$${platform#*/} \
		BINARY_NAME=message-mirror-$${platform%/*}-$${platform#*/} \
		ARCHIVE_NAME=message-mirror-v$(VERSION)-$${platform%/*}-$${platform#*/}; \
		if [ "$${platform%/*}" = "windows" ]; then \
			BINARY_NAME=$${BINARY_NAME}.exe; \
			ARCHIVE_NAME=$${ARCHIVE_NAME}.zip; \
			cd release && zip -q $$ARCHIVE_NAME $${BINARY_NAME} README.md CHANGELOG.md LICENSE 2>/dev/null || zip -q $$ARCHIVE_NAME $${BINARY_NAME} 2>/dev/null || true; \
		else \
			ARCHIVE_NAME=$${ARCHIVE_NAME}.tar.gz; \
			cd release && tar -czf $$ARCHIVE_NAME $${BINARY_NAME} README.md CHANGELOG.md LICENSE 2>/dev/null || tar -czf $$ARCHIVE_NAME $${BINARY_NAME} 2>/dev/null || true; \
		fi; \
		cd ..; \
	done
	@echo "发布包创建完成！"
	@echo "文件列表:"
	@ls -lh release/*.tar.gz release/*.zip 2>/dev/null || ls -lh release/
	@echo ""
	@echo "下一步:"
	@echo "  1. 检查 release/ 目录中的文件"
	@echo "  2. 创建Git标签: git tag -a v$(VERSION) -m 'Release v$(VERSION)'"
	@echo "  3. 推送标签: git push origin v$(VERSION)"

# 清理构建产物
clean:
	@echo "清理构建产物..."
	rm -f message-mirror
	rm -rf logs/*.log
	rm -rf logs/*.tar.gz
	rm -rf release/*

# Docker构建
docker-build:
	@echo "构建Docker镜像..."
	docker build -t $(IMAGE_NAME):$(VERSION) -f docker/Dockerfile .

# Docker运行
docker-run: docker-build
	@echo "运行Docker容器..."
	docker run -d \
		--name $(IMAGE_NAME) \
		-p 8080:8080 \
		-v $(PWD)/config:/app/config:ro \
		-v $(PWD)/logs:/app/logs \
		-v $(PWD)/data:/app/data \
		$(IMAGE_NAME):$(VERSION)

# Docker Compose启动
docker-compose-up:
	@echo "启动Docker Compose服务..."
	cd docker && docker-compose -f docker-compose.yml up -d

# Docker Compose启动（完整环境）
docker-compose-up-full:
	@echo "启动完整Docker Compose环境（包括Kafka）..."
	cd docker && docker-compose -f docker-compose.full.yml up -d

# Docker Compose停止
docker-compose-down:
	@echo "停止Docker Compose服务..."
	cd docker && docker-compose -f docker-compose.yml down

# Docker Compose停止（完整环境）
docker-compose-down-full:
	@echo "停止完整Docker Compose环境..."
	cd docker && docker-compose -f docker-compose.full.yml down

# 查看日志
docker-logs:
	cd docker && docker-compose -f docker-compose.yml logs -f message-mirror

# 查看完整环境日志
docker-logs-full:
	cd docker && docker-compose -f docker-compose.full.yml logs -f

# 启动监控服务
docker-compose-monitoring:
	@echo "启动监控服务（Prometheus + Grafana）..."
	cd docker && docker-compose --profile monitoring up -d

# 健康检查
health-check:
	@echo "检查服务健康状态..."
	@curl -s http://localhost:8080/health || echo "服务未运行"

# 查看指标
metrics:
	@echo "查看Prometheus指标..."
	@curl -s http://localhost:8080/metrics | head -20

# 帮助信息
help:
	@echo "Message Mirror Makefile"
	@echo ""
	@echo "可用命令:"
	@echo "  make build                    - 本地构建"
	@echo "  make run                      - 运行本地构建的二进制文件"
	@echo "  make test                     - 运行测试"
	@echo "  make clean                    - 清理构建产物"
	@echo "  make docker-build             - 构建Docker镜像"
	@echo "  make docker-run               - 运行Docker容器"
	@echo "  make docker-compose-up       - 启动Docker Compose服务"
	@echo "  make docker-compose-up-full  - 启动完整环境（包括Kafka）"
	@echo "  make docker-compose-down     - 停止Docker Compose服务"
	@echo "  make docker-compose-down-full - 停止完整环境"
	@echo "  make docker-logs             - 查看日志"
	@echo "  make docker-compose-monitoring - 启动监控服务"
	@echo "  make health-check             - 健康检查"
	@echo "  make metrics                  - 查看指标"
	@echo "  make version                  - 显示版本信息"
	@echo "  make build-all                - 构建所有平台"
	@echo "  make release                  - 构建、测试并打包发布版本"
	@echo "  make help                     - 显示帮助信息"

