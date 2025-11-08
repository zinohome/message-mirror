# 项目结构重构指南

## 重构原因

当前根目录下有39个Go文件，结构混乱，难以维护。需要重构为清晰的分层结构。

## 重构方案

### 新目录结构

```
message-mirror/
├── cmd/
│   └── message-mirror/
│       └── main.go              # 程序入口（package main）
├── internal/
│   ├── core/                    # 核心业务逻辑（package core）
│   │   ├── mirror.go
│   │   ├── producer.go
│   │   ├── config.go
│   │   ├── config_manager.go
│   │   └── *_test.go
│   ├── plugins/                 # 插件系统
│   │   ├── plugin.go            # 插件接口（package plugins）
│   │   ├── kafka/
│   │   │   ├── kafka.go         # package kafka
│   │   │   └── kafka_test.go
│   │   ├── rabbitmq/
│   │   │   ├── rabbitmq.go      # package rabbitmq
│   │   │   └── rabbitmq_test.go
│   │   └── file/
│   │       ├── file.go          # package file
│   │       └── file_test.go
│   └── pkg/                     # 内部工具包
│       ├── metrics/             # package metrics
│       ├── logger/              # package logger
│       ├── ratelimiter/         # package ratelimiter
│       ├── retry/               # package retry
│       ├── deduplicator/        # package deduplicator
│       ├── security/            # package security
│       ├── optimization/        # package optimization
│       └── version/             # package version
├── web/                         # Web UI（package web）
│   └── webui.go
└── ...
```

## 重构步骤

### 阶段1：工具包迁移（最简单）

这些包相对独立，依赖最少，先迁移它们。

1. **version** → `internal/pkg/version/`
2. **metrics** → `internal/pkg/metrics/`
3. **logger** → `internal/pkg/logger/`
4. **ratelimiter** → `internal/pkg/ratelimiter/`
5. **retry** → `internal/pkg/retry/`
6. **deduplicator** → `internal/pkg/deduplicator/`
7. **security** → `internal/pkg/security/`
8. **optimization** → `internal/pkg/optimization/`

### 阶段2：插件迁移

1. **plugin.go** → `internal/plugins/plugin.go` (package plugins)
2. **kafka_plugin.go** → `internal/plugins/kafka/kafka.go` (package kafka)
3. **rabbitmq_plugin.go** → `internal/plugins/rabbitmq/rabbitmq.go` (package rabbitmq)
4. **file_plugin.go** → `internal/plugins/file/file.go` (package file)

### 阶段3：核心业务逻辑迁移

1. **config.go** → `internal/core/config.go` (package core)
2. **config_manager.go** → `internal/core/config_manager.go` (package core)
3. **producer.go** → `internal/core/producer.go` (package core)
4. **mirror.go** → `internal/core/mirror.go` (package core)

### 阶段4：Web UI迁移

1. **webui.go** → `web/webui.go` (package web)

### 阶段5：主程序迁移

1. **main.go** → `cmd/message-mirror/main.go` (package main)

## 注意事项

1. **导出函数**：需要导出的函数首字母大写
2. **import路径**：更新所有import路径为 `message-mirror/internal/pkg/...`
3. **测试文件**：跟随源文件移动
4. **编译验证**：每阶段完成后验证编译

## 执行顺序

按阶段顺序执行，每完成一个阶段验证编译通过后再进行下一阶段。

