# WebSocket + API 快速测试指南

## 本地测试步骤

### 1. 启动 Backend 服务

```bash
# 编译
go build -o message-mirror ./cmd/message-mirror

# 启动服务（监听 8080 端口）
./message-mirror --config config.yaml --validate-config
# 或使用默认配置
./message-mirror
```

服务启动后，HTTP服务器监听 `http://localhost:8080`

### 2. 测试 REST API（使用 curl）

#### 获取配置
```bash
curl -X GET http://localhost:8080/api/config | jq .
```

**预期响应**:
```json
{
  "source": {
    "type": "kafka",
    "brokers": ["localhost:9092"]
  },
  "target": {
    "brokers": ["localhost:9093"],
    "topic": "mirrored-messages"
  },
  ...
}
```

#### 更新配置
```bash
curl -X POST http://localhost:8080/api/config \
  -H "Content-Type: application/json" \
  -d '{
    "source": {"type": "rabbitmq"},
    "target": {"brokers": ["localhost:9093"], "topic": "new-topic"},
    "mirror": {"worker_count": 8}
  }' | jq .
```

**预期响应**:
```json
{
  "status": "success",
  "message": "配置已更新"
}
```

#### 重新加载配置
```bash
curl -X POST http://localhost:8080/api/config/reload | jq .
```

**预期响应**:
```json
{
  "status": "success",
  "message": "配置已重载"
}
```

#### 获取统计信息
```bash
curl -X GET http://localhost:8080/api/stats | jq .
```

**预期响应**:
```json
{
  "messages_consumed": 1000,
  "messages_produced": 950,
  "bytes_consumed": 102400,
  "bytes_produced": 97280,
  "errors": 2,
  "last_message_time": "2024-01-01T12:00:00Z",
  "start_time": "2024-01-01T11:00:00Z",
  "uptime_seconds": 3600
}
```

### 3. 测试 WebSocket（使用 wscat）

#### 安装 wscat
```bash
npm install -g wscat
```

#### 连接 WebSocket 端点
```bash
wscat -c ws://localhost:8080/ws/stats
```

#### 接收实时数据

连接成功后，服务器会每秒推送统计数据：

```json
{
  "messages_consumed": 0,
  "messages_produced": 0,
  "bytes_consumed": 0,
  "bytes_produced": 0,
  "errors": 0,
  "last_message_time": 0,
  "start_time": 1704067000,
  "uptime_seconds": 0.005,
  "consume_rate": 0,
  "produce_rate": 0
}
```

每秒钟会收到一条新消息。

#### WebSocket 测试脚本（JavaScript）

创建文件 `test-ws.js`:

```javascript
const ws = new WebSocket('ws://localhost:8080/ws/stats');

ws.onopen = () => {
  console.log('✅ WebSocket 连接成功');
  console.log('开始接收实时统计数据...\n');
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log(`[${new Date().toLocaleTimeString()}] 统计数据:`);
  console.log(`  消费: ${data.messages_consumed} 消息, ${data.consume_rate.toFixed(2)} KB/s`);
  console.log(`  生产: ${data.messages_produced} 消息, ${data.produce_rate.toFixed(2)} KB/s`);
  console.log(`  错误: ${data.errors}`);
  console.log(`  运行时间: ${data.uptime_seconds.toFixed(1)}s\n`);
};

ws.onclose = () => {
  console.log('❌ WebSocket 连接已关闭');
};

ws.onerror = (error) => {
  console.error('❌ WebSocket 错误:', error);
};

// 10秒后关闭连接
setTimeout(() => {
  ws.close();
}, 10000);
```

运行:
```bash
node test-ws.js
```

**预期输出**:
```
✅ WebSocket 连接成功
开始接收实时统计数据...

[12:34:56] 统计数据:
  消费: 100 消息, 10.24 KB/s
  生产: 95 消息, 9.73 KB/s
  错误: 0
  运行时间: 10.5s

[12:34:57] 统计数据:
  消费: 101 消息, 10.24 KB/s
  生产: 96 消息, 9.73 KB/s
  错误: 0
  运行时间: 11.5s
```

### 4. 运行单元测试

```bash
# 运行 WebSocket 测试
go test -v ./internal/core -run "WebSocket" -timeout 30s

# 运行所有 HTTP 服务器测试
go test -v ./internal/core -run "HTTP" -timeout 30s

# 运行所有测试
go test -v ./...
```

### 5. 测试 CORS

```bash
# 发送 OPTIONS 预检请求
curl -X OPTIONS http://localhost:8080/api/config \
  -H "Origin: http://localhost:3000" \
  -H "Access-Control-Request-Method: POST" \
  -v
```

**预期响应头**:
```
< Access-Control-Allow-Origin: *
< Access-Control-Allow-Methods: GET, POST, PUT, OPTIONS
< Access-Control-Allow-Headers: Content-Type
< HTTP/1.1 200 OK
```

## 与 React 前端集成

前端 (`web/frontend`) 已配置为通过 Vite dev server 代理 API 请求：

```javascript
// vite.config.js 中的代理配置
proxy: {
  '/api': {
    target: 'http://localhost:8080',
    changeOrigin: true,
  },
  '/ws': {
    target: 'ws://localhost:8080',
    ws: true,
  }
}
```

### 本地开发步骤

1. **终端1 - 启动 Backend**:
```bash
./message-mirror
```

2. **终端2 - 启动 Frontend Dev Server**:
```bash
cd web/frontend
npm install  # 首次运行
npm run dev
```

3. **在浏览器中打开**:
```
http://localhost:3000
```

### 浏览器中的实时效果

- **Overview** 标签: 实时显示消费/生产速率（每秒更新）
- **Configuration** 标签: 可以查看和更新配置
- **Monitoring** 标签: 实时表格显示消费/生产统计

## 故障排除

### WebSocket 连接失败

**症状**: `WebSocket is undefined` 或连接超时

**解决方案**:
1. 确保服务器正在运行: `netstat -an | grep 8080`
2. 检查防火墙: `sudo ufw allow 8080/tcp`
3. 检查日志: `tail -f message-mirror.log`

### API 返回 500 错误

**症状**: `{"error":"MirrorMaker未初始化"}`

**解决方案**:
1. 确保配置文件有效: `./message-mirror --validate-config`
2. 检查配置文件路径: `--config /path/to/config.yaml`
3. 查看启动日志

### CORS 错误

**症状**: `Access to XMLHttpRequest has been blocked by CORS policy`

**解决方案**:
1. 使用 Vite 代理（推荐）
2. 或使用浏览器扩展禁用CORS检查（开发用）
3. 检查 `Access-Control-Allow-Origin` 头

## 性能测试

### WebSocket 连接数限制测试

```bash
# 模拟多个客户端连接
for i in {1..100}; do
  wscat -c ws://localhost:8080/ws/stats &
done

# 查看连接数
netstat -an | grep 8080 | wc -l
```

### API 响应时间测试

```bash
# 使用 Apache Bench
ab -n 1000 -c 10 http://localhost:8080/api/config

# 或使用 wrk
wrk -t4 -c100 -d10s http://localhost:8080/api/config
```

## 清理资源

```bash
# 关闭 Backend 服务
pkill -f "message-mirror"

# 关闭 Frontend dev server
# (Ctrl+C 或 pkill -f "vite")
```

---

**最后更新**: Phase 4 完成  
**维护者**: Message Mirror Team
