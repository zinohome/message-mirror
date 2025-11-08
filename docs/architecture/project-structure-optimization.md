# 项目结构优化方案

## 当前问题

1. **根目录文件过多**：39个Go文件混在一起
2. **测试文件混杂**：测试文件和源文件混在一起
3. **职责不清**：核心逻辑、插件、工具类都在根目录
4. **难以维护**：文件太多，难以快速定位

## 优化方案

### 目标结构

```
message-mirror/
├── cmd/
│   └── message-mirror/
│       └── main.go              # 程序入口
├── internal/
│   ├── core/                    # 核心业务逻辑
│   │   ├── mirror.go
│   │   ├── producer.go
│   │   ├── config.go
│   │   ├── config_manager.go
│   │   └── ...
│   ├── plugins/                 # 插件实现
│   │   ├── plugin.go           # 插件接口
│   │   ├── kafka/
│   │   │   ├── kafka.go
│   │   │   └── kafka_test.go
│   │   ├── rabbitmq/
│   │   │   ├── rabbitmq.go
│   │   │   └── rabbitmq_test.go
│   │   └── file/
│   │       ├── file.go
│   │       └── file_test.go
│   └── pkg/                     # 内部工具包
│       ├── metrics/
│       │   ├── metrics.go
│       │   └── metrics_test.go
│       ├── logger/
│       │   ├── logger.go
│       │   └── logger_test.go
│       ├── ratelimiter/
│       │   ├── ratelimiter.go
│       │   └── ratelimiter_test.go
│       ├── retry/
│       │   ├── retry.go
│       │   └── retry_test.go
│       ├── deduplicator/
│       │   ├── deduplicator.go
│       │   └── deduplicator_test.go
│       ├── security/
│       │   ├── security.go
│       │   └── security_test.go
│       └── optimization/
│           ├── optimization.go
│           └── optimization_test.go
├── pkg/                         # 公开API（可选）
│   └── api/
│       └── config.go
├── web/                         # Web UI
│   └── webui.go
├── config/                      # 配置文件
├── docs/                        # 文档
├── docker/                      # Docker配置
└── release/                     # 发布包
```

## 优化原则

### 1. 包组织原则

- **cmd/**: 程序入口，只包含main.go
- **internal/**: 内部代码，外部无法导入
  - **core/**: 核心业务逻辑
  - **plugins/**: 插件实现
  - **pkg/**: 内部工具包
- **pkg/**: 公开API（如果需要）
- **web/**: Web相关代码

### 2. 测试文件位置

- 测试文件与被测试文件在同一目录
- 这是Go的要求，无法改变

### 3. 插件组织

- 每个插件一个子目录
- 插件接口在plugins/根目录
- 具体实现在各自子目录

## 迁移步骤

1. 创建新目录结构
2. 移动文件到对应目录
3. 更新package声明
4. 更新import路径
5. 验证编译和测试

## 优势

1. **清晰的职责分离**：核心逻辑、插件、工具类分开
2. **易于维护**：文件按功能组织，易于查找
3. **符合Go最佳实践**：遵循标准项目布局
4. **更好的可扩展性**：新功能易于添加

## 注意事项

1. **internal包**：外部无法导入，保证封装性
2. **测试文件**：必须与源文件在同一目录
3. **插件接口**：需要保持向后兼容
4. **配置管理**：路径可能需要调整

