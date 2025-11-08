# Web UI API 文档

## 概述

Message Mirror 提供了Web界面用于配置管理和监控。Web UI通过RESTful API与后端通信。

## API端点

### 1. 获取配置

**GET** `/api/config`

获取当前配置。

**响应示例**:
```json
{
  "source": {
    "type": "kafka",
    "config": {
      "brokers": ["localhost:9092"],
      "topics": ["test-topic"]
    }
  },
  "target": {
    "brokers": ["localhost:9093"],
    "topic": "mirrored-topic"
  },
  "mirror": {
    "worker_count": 4,
    "batch_enabled": true,
    "batch_size": 100
  }
}
```

### 2. 更新配置

**POST** `/api/config`

更新配置并热重载。

**请求体**:
```json
{
  "mirror": {
    "worker_count": 8,
    "batch_size": 200
  }
}
```

**响应**:
```json
{
  "status": "success",
  "message": "配置已更新"
}
```

### 3. 重载配置

**POST** `/api/config/reload`

从配置文件重新加载配置。

**响应**:
```json
{
  "status": "success",
  "message": "配置已重载"
}
```

### 4. 获取统计信息

**GET** `/api/stats`

获取运行统计信息。

**响应示例**:
```json
{
  "messages_consumed": 1000,
  "messages_produced": 998,
  "bytes_consumed": 1048576,
  "bytes_produced": 1047552,
  "errors": 2,
  "last_message_time": "2025-11-05T22:00:00Z",
  "start_time": "2025-11-05T20:00:00Z",
  "uptime_seconds": 7200
}
```

## Web UI

访问 `http://localhost:8080/` 或 `http://localhost:8080/ui` 打开Web界面。

### 功能

1. **配置管理**
   - 查看当前配置
   - 修改配置参数
   - 保存并热重载配置
   - 从文件重载配置

2. **统计监控**
   - 实时查看消息处理统计
   - 自动刷新（每5秒）
   - 手动刷新

### 配置热重载

配置更新后会自动热重载，无需重启服务。支持热重载的配置项：

- Worker数量
- 批处理配置
- 速率限制
- 去重配置
- 重试配置

注意：某些配置（如数据源类型、目标集群）的更改可能需要重启服务才能完全生效。

