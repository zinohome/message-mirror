.PHONY: build run test clean docker-build docker-run docker-compose-up docker-compose-down help

# 变量定义
IMAGE_NAME = message-mirror
VERSION = latest
DOCKER_COMPOSE_FILE = docker/docker-compose.yml

# 本地构建
build:
	@echo "构建message-mirror..."
	go mod download
	go build -o message-mirror .

# 运行本地构建的二进制文件
run: build
	@echo "运行message-mirror..."
	./message-mirror -c config.yaml

# 运行测试
test:
	@echo "运行测试..."
	go test -v ./tests/...

# 清理构建产物
clean:
	@echo "清理构建产物..."
	rm -f message-mirror
	rm -rf logs/*.log
	rm -rf logs/*.tar.gz

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
	@echo "  make help                     - 显示帮助信息"

