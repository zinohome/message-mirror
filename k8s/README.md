# Kubernetes 部署指南

本目录包含Message Mirror在Kubernetes上的部署清单。

## 文件说明

```
k8s/
├── namespace.yaml          # 命名空间
├── configmap.yaml          # 配置文件
├── secret.yaml             # 敏感信息（密码、证书）
├── rbac.yaml               # ServiceAccount和权限
├── deployment.yaml         # Deployment部署配置
├── service.yaml            # Service服务
├── hpa.yaml                # 水平自动伸缩
├── pdb.yaml                # Pod中断预算
├── servicemonitor.yaml     # Prometheus监控
└── README.md               # 本文件
```

## 前置要求

- Kubernetes 1.19+
- kubectl已配置并连接到集群
- Kafka集群已部署（源集群和目标集群）
- （可选）Prometheus Operator（用于ServiceMonitor）
- （可选）Metrics Server（用于HPA）

## 快速开始

### 1. 创建命名空间

```bash
kubectl apply -f namespace.yaml
```

### 2. 配置Secret

编辑 `secret.yaml` 添加Kafka认证信息：

```bash
# 使用base64编码敏感信息
echo -n "your-username" | base64
echo -n "your-password" | base64

# 编辑secret.yaml填入base64编码后的值
kubectl apply -f secret.yaml
```

或者使用kubectl直接创建：

```bash
kubectl create secret generic message-mirror-secret \
  --from-literal=source-sasl-username='your-username' \
  --from-literal=source-sasl-password='your-password' \
  --from-literal=target-sasl-username='your-username' \
  --from-literal=target-sasl-password='your-password' \
  -n message-mirror
```

### 3. 修改ConfigMap

编辑 `configmap.yaml` 根据实际环境配置：

- Kafka brokers地址
- Topic名称
- Consumer Group ID
- 安全协议
- Worker数量
- 速率限制

```bash
kubectl apply -f configmap.yaml
```

### 4. 部署应用

```bash
# 创建RBAC
kubectl apply -f rbac.yaml

# 部署应用
kubectl apply -f deployment.yaml

# 创建Service
kubectl apply -f service.yaml

# （可选）启用自动伸缩
kubectl apply -f hpa.yaml

# （可选）配置Pod中断预算
kubectl apply -f pdb.yaml

# （可选）配置Prometheus监控
kubectl apply -f servicemonitor.yaml
```

### 5. 验证部署

```bash
# 查看Pod状态
kubectl get pods -n message-mirror

# 查看日志
kubectl logs -f deployment/message-mirror -n message-mirror

# 查看服务状态
kubectl get svc -n message-mirror

# 检查健康状态
kubectl exec -it deployment/message-mirror -n message-mirror -- wget -O- http://localhost:8080/health
```

## 一键部署

```bash
# 按顺序部署所有资源
kubectl apply -f namespace.yaml
kubectl apply -f secret.yaml
kubectl apply -f configmap.yaml
kubectl apply -f rbac.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
kubectl apply -f hpa.yaml
kubectl apply -f pdb.yaml
kubectl apply -f servicemonitor.yaml
```

或使用单个命令：

```bash
kubectl apply -f .
```

## 配置说明

### 环境变量

Deployment中的环境变量可通过以下方式覆盖：

```yaml
env:
- name: WORKER_COUNT
  value: "8"  # 修改worker数量
- name: BYTES_RATE_LIMIT
  value: "20971520"  # 20MB/s
```

### 资源配置

默认资源配置：

```yaml
resources:
  requests:
    cpu: 500m
    memory: 512Mi
  limits:
    cpu: 2000m
    memory: 2Gi
```

根据实际负载调整：

- **低负载**（<1000 msg/s）: 500m CPU, 512Mi Memory
- **中负载**（1000-5000 msg/s）: 1000m CPU, 1Gi Memory
- **高负载**（>5000 msg/s）: 2000m+ CPU, 2Gi+ Memory

### 副本数配置

```yaml
spec:
  replicas: 2  # 默认2个副本
```

建议配置：
- **开发环境**: 1个副本
- **测试环境**: 2个副本
- **生产环境**: 3+个副本

### HPA自动伸缩

HPA配置范围：2-10个副本

触发条件：
- CPU使用率 > 70%
- 内存使用率 > 80%

禁用HPA：

```bash
kubectl delete hpa message-mirror -n message-mirror
```

## 健康检查

### Liveness Probe（存活探针）

检查应用是否运行：

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10
```

### Readiness Probe（就绪探针）

检查应用是否准备好接收流量：

```yaml
readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 5
```

### Startup Probe（启动探针）

给应用更长的启动时间：

```yaml
startupProbe:
  httpGet:
    path: /health
    port: 8080
  failureThreshold: 30  # 最多等待150秒
  periodSeconds: 5
```

## 监控

### Prometheus集成

ServiceMonitor会自动被Prometheus Operator发现：

```bash
# 查看ServiceMonitor
kubectl get servicemonitor -n message-mirror

# 查看指标
kubectl port-forward svc/message-mirror 8080:8080 -n message-mirror
curl http://localhost:8080/metrics
```

### 可用指标

- `mirror_messages_consumed_total`: 消费的消息总数
- `mirror_messages_produced_total`: 生产的消息总数
- `mirror_messages_failed_total`: 失败的消息总数
- `mirror_bytes_consumed_total`: 消费的字节总数
- `mirror_bytes_produced_total`: 生产的字节总数
- `mirror_latency_seconds`: 消息处理延迟

## 日志

### 查看日志

```bash
# 实时查看日志
kubectl logs -f deployment/message-mirror -n message-mirror

# 查看所有Pod日志
kubectl logs -l app=message-mirror -n message-mirror --tail=100

# 查看特定Pod日志
kubectl logs message-mirror-xxx-yyy -n message-mirror
```

### 日志级别

通过ConfigMap调整：

```yaml
log:
  level: info  # debug, info, warn, error
```

## 更新部署

### 滚动更新

```bash
# 更新镜像
kubectl set image deployment/message-mirror \
  message-mirror=message-mirror:v0.2.0 \
  -n message-mirror

# 查看更新状态
kubectl rollout status deployment/message-mirror -n message-mirror

# 查看更新历史
kubectl rollout history deployment/message-mirror -n message-mirror
```

### 回滚

```bash
# 回滚到上一版本
kubectl rollout undo deployment/message-mirror -n message-mirror

# 回滚到指定版本
kubectl rollout undo deployment/message-mirror --to-revision=2 -n message-mirror
```

### 配置热重载

修改ConfigMap后，需要重启Pod：

```bash
# 方法1：强制重启
kubectl rollout restart deployment/message-mirror -n message-mirror

# 方法2：删除Pod让其自动重建
kubectl delete pod -l app=message-mirror -n message-mirror
```

## 故障排查

### Pod无法启动

```bash
# 查看Pod事件
kubectl describe pod <pod-name> -n message-mirror

# 查看容器日志
kubectl logs <pod-name> -n message-mirror

# 查看上一次容器日志
kubectl logs <pod-name> -n message-mirror --previous
```

### 健康检查失败

```bash
# 进入容器检查
kubectl exec -it <pod-name> -n message-mirror -- sh

# 手动测试健康检查
wget -O- http://localhost:8080/health
wget -O- http://localhost:8080/ready
```

### 连接Kafka失败

```bash
# 测试网络连通性
kubectl exec -it <pod-name> -n message-mirror -- nc -zv kafka.kafka.svc.cluster.local 9092

# 查看环境变量
kubectl exec <pod-name> -n message-mirror -- env | grep KAFKA
```

## 安全加固

### 使用非root用户

Deployment已配置：

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  fsGroup: 1000
```

### TLS证书

如果使用TLS，在Secret中添加证书：

```bash
kubectl create secret generic message-mirror-tls \
  --from-file=ca.crt=path/to/ca.crt \
  --from-file=client.crt=path/to/client.crt \
  --from-file=client.key=path/to/client.key \
  -n message-mirror
```

更新Deployment挂载证书：

```yaml
volumeMounts:
- name: tls
  mountPath: /app/certs
  readOnly: true
volumes:
- name: tls
  secret:
    secretName: message-mirror-tls
```

## 清理

```bash
# 删除所有资源
kubectl delete -f .

# 或者删除整个命名空间
kubectl delete namespace message-mirror
```

## 生产环境建议

1. **资源限制**: 根据实际负载设置合理的resources
2. **副本数**: 至少3个副本确保高可用
3. **HPA**: 启用自动伸缩应对流量波动
4. **PDB**: 配置Pod中断预算保证更新时的可用性
5. **监控**: 集成Prometheus监控和告警
6. **日志**: 集成ELK或Loki日志聚合系统
7. **备份**: 定期备份ConfigMap和Secret
8. **TLS**: 生产环境启用TLS加密
9. **RBAC**: 使用最小权限原则
10. **Network Policy**: 限制Pod间网络访问

## 参考

- [Kubernetes官方文档](https://kubernetes.io/docs/)
- [Prometheus Operator](https://github.com/prometheus-operator/prometheus-operator)
- [Message Mirror文档](../docs/)
