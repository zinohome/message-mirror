# Docker部署指南

## 快速开始

### 1. 准备环境

确保已安装：
- Docker 20.10+
- Docker Compose 2.0+

### 2. 配置准备

```bash
# 创建必要的目录
mkdir -p config logs data

# 复制配置文件示例
cp config.yaml.example config/config.yaml

# 编辑配置文件
vi config/config.yaml
```

### 3. 启动服务

#### 方式一：仅启动Message Mirror（推荐）

适用于已有Kafka集群的场景：

```bash
cd docker && docker-compose up -d
```

#### 方式二：启动完整环境

适用于开发测试环境，包含完整的Kafka集群：

```bash
cd docker && docker-compose -f docker-compose.full.yml up -d
```

### 4. 验证部署

```bash
# 检查服务状态
docker-compose ps

# 查看日志
docker-compose logs -f message-mirror

# 健康检查
curl http://localhost:8080/health

# 查看指标
curl http://localhost:8080/metrics
```

## 配置文件说明

配置文件需要放在 `config/` 目录下，容器启动时会读取 `/app/config/config.yaml`。

### Docker环境配置要点

1. **Kafka地址**: 使用Docker服务名或外部Kafka地址
   ```yaml
   source:
     brokers:
       - "kafka-source:9092"  # Docker服务名
   ```

2. **日志路径**: 使用容器内路径
   ```yaml
   log:
     file_path: "/app/logs/message-mirror.log"
   ```

3. **HTTP服务器**: 确保监听所有接口
   ```yaml
   server:
     address: ":8080"  # 监听所有接口
   ```

## 监控部署

### 启动监控服务

```bash
docker-compose --profile monitoring up -d
```

### 访问监控界面

- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000
  - 用户名: `admin`
  - 密码: `admin`

### 配置Prometheus

Prometheus会自动从 `monitoring/prometheus.yml` 读取配置，监控Message Mirror的指标。

## 生产环境部署

### 1. 使用外部Kafka集群

修改 `docker-compose.yml`，移除Kafka依赖，使用外部Kafka地址：

```yaml
services:
  message-mirror:
    # ... 其他配置
    # 移除depends_on中的kafka服务
    environment:
      - KAFKA_SOURCE_BROKERS=kafka1:9092,kafka2:9092
      - KAFKA_TARGET_BROKERS=kafka3:9092,kafka4:9092
```

### 2. 资源配置

根据实际负载调整资源配置：

```yaml
deploy:
  resources:
    limits:
      cpus: '4.0'
      memory: 4G
    reservations:
      cpus: '1.0'
      memory: 1G
```

### 3. 日志管理

日志文件会保存到 `logs/` 目录，建议配置日志轮转和归档策略。

### 4. 数据持久化

重要数据（如去重记录）建议使用Docker卷持久化：

```yaml
volumes:
  - ./data:/app/data
  - message_mirror_data:/app/data
```

## 故障排查

### 查看容器日志

```bash
# 查看所有日志
cd docker && docker-compose logs message-mirror

# 实时查看日志
cd docker && docker-compose logs -f message-mirror

# 查看最近100行日志
cd docker && docker-compose logs --tail=100 message-mirror
```

### 进入容器调试

```bash
docker exec -it message-mirror sh
```

### 检查健康状态

```bash
# 健康检查
curl http://localhost:8080/health

# 就绪检查
curl http://localhost:8080/ready
```

### 常见问题

1. **容器无法启动**: 检查配置文件路径和权限
2. **无法连接Kafka**: 检查网络配置和Kafka地址
3. **内存不足**: 调整资源配置或限制去重记录数
4. **端口冲突**: 修改docker-compose.yml中的端口映射

## 更新部署

### 更新镜像

```bash
# 重新构建镜像
cd docker && docker-compose build

# 重启服务
cd docker && docker-compose up -d
```

### 滚动更新

```bash
# 停止旧容器
cd docker && docker-compose down

# 启动新容器
cd docker && docker-compose up -d
```

## 安全建议

1. **配置文件**: 使用只读挂载，避免容器内修改
2. **用户权限**: 容器以非root用户运行
3. **网络隔离**: 使用Docker网络隔离服务
4. **资源限制**: 设置合理的CPU和内存限制
5. **敏感信息**: 使用环境变量或密钥管理服务

