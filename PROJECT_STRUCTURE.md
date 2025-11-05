# 项目结构说明

本项目遵循Go语言项目结构最佳实践。

## 目录结构

```
message-mirror/
├── cmd/                    # 可执行文件（可选，当前main.go在根目录）
│   └── message-mirror/
│       └── main.go
├── internal/               # 内部包（可选，当前包都在根目录）
│   └── ...
├── pkg/                     # 可被外部导入的包（可选）
│   └── ...
├── api/                     # API定义（可选）
│   └── ...
├── web/                     # Web资源（可选）
│   └── ...
├── configs/                 # 配置文件（可选）
│   └── ...
├── scripts/                 # 脚本文件（可选）
│   └── ...
├── docker/                  # Docker相关文件
│   ├── Dockerfile
│   ├── docker-compose.yml
│   ├── docker-compose.full.yml
│   ├── .dockerignore
│   └── monitoring/
│       ├── prometheus.yml
│       └── grafana/
├── tests/                   # 测试文件
│   ├── *_test.go
│   ├── benchmark_test.go
│   ├── integration_test.go
│   └── README.md
├── docs/                    # 文档（可选）
│   └── ...
├── config.yaml              # 配置文件示例
├── config.*.yaml            # 其他配置示例
├── go.mod                   # Go模块定义
├── go.sum                   # Go模块校验和
├── Makefile                 # Makefile工具
├── README.md                # 项目说明
├── DEPLOYMENT.md            # 部署文档
├── PROJECT_STRUCTURE.md     # 项目结构说明（本文件）
├── .gitignore              # Git忽略文件
├── .dockerignore            # Docker忽略文件（已移至docker/）
└── *.go                     # Go源代码文件
```

## Go项目结构最佳实践

### 1. 标准Go项目布局（Standard Go Project Layout）

参考: https://github.com/golang-standards/project-layout

#### 核心目录
- `/cmd`: 应用程序的main文件
- `/internal`: 私有应用程序代码
- `/pkg`: 可被外部应用程序使用的库代码
- `/api`: API定义（OpenAPI/Swagger等）
- `/web`: Web应用程序资源

#### 服务应用目录
- `/configs`: 配置文件模板或默认配置
- `/scripts`: 用于执行各种构建、安装、分析等的脚本
- `/build`: 打包和持续集成
- `/deployments`: 部署配置（Kubernetes、Docker Compose等）

#### 其他目录
- `/docs`: 设计和用户文档
- `/test`: 额外的外部测试应用和测试数据
- `/examples`: 应用程序或公共库的示例
- `/third_party`: 外部辅助工具、forked代码和其他第三方工具
- `/tools`: 项目的支持工具
- `/website`: 项目的网站数据（如果使用）

### 2. 本项目结构说明

#### 当前结构（简化版）
本项目采用简化的Go项目结构，适合中小型项目：

- **根目录**: 包含主要的Go源代码文件（符合Go标准）
- **docker/**: 所有Docker相关文件集中管理
- **tests/**: 所有测试文件集中管理
- **configs/**: 配置文件（可选，当前在根目录）

#### 为什么这样组织？

1. **根目录的Go文件**: Go语言约定，可执行程序通常将main.go放在根目录
2. **docker/目录**: 将Docker相关文件集中管理，便于维护
3. **tests/目录**: 将测试文件集中管理，保持根目录整洁
4. **配置文件**: 示例配置文件放在根目录，便于查找

### 3. 目录说明

#### `/docker`
包含所有Docker相关文件：
- `Dockerfile`: Docker镜像构建文件
- `docker-compose.yml`: Docker Compose配置（基础版）
- `docker-compose.full.yml`: Docker Compose配置（完整环境）
- `.dockerignore`: Docker构建忽略文件
- `monitoring/`: 监控配置（Prometheus、Grafana）

#### `/tests`
包含所有测试文件：
- `*_test.go`: 单元测试文件
- `benchmark_test.go`: 性能基准测试
- `integration_test.go`: 集成测试
- `README.md`: 测试说明文档

#### `/plugins`
**注意**: 本目录仅用于说明插件系统，实际的插件实现文件位于根目录。

由于Go语言的限制（同一个包的所有文件必须在同一目录），插件实现文件（`kafka_plugin.go`、`rabbitmq_plugin.go`、`file_plugin.go`）都在根目录的 `package main` 中。

- `README.md`: 插件系统说明文档

#### `/configs`（可选）
如果配置文件较多，可以创建此目录：
- `config.yaml`: 默认配置
- `config.prod.yaml`: 生产环境配置
- `config.dev.yaml`: 开发环境配置

### 4. 文件组织原则

#### Go源代码
- 每个包一个目录
- 相关功能放在同一个包中
- 避免循环依赖
- 使用`internal/`包保护私有代码

#### 测试文件
- 测试文件以`_test.go`结尾
- 测试文件放在`tests/`目录中
- 单元测试、集成测试、性能测试分开组织

#### 配置文件
- 示例配置文件放在根目录
- 实际配置文件通过环境变量或挂载提供
- 敏感信息使用环境变量或密钥管理

#### Docker文件
- 所有Docker相关文件放在`docker/`目录
- Dockerfile使用相对路径引用项目根目录
- docker-compose.yml使用相对路径挂载卷

### 5. 命令和工具

#### 构建
```bash
go build -o message-mirror
```

#### 测试
```bash
go test -v ./tests/...
```

#### Docker构建
```bash
docker build -f docker/Dockerfile -t message-mirror:latest .
```

#### Docker Compose
```bash
cd docker && docker-compose up -d
```

### 6. 扩展建议

如果项目继续增长，可以考虑：

1. **创建`cmd/`目录**: 如果有多个可执行文件
2. **创建`internal/`目录**: 如果有需要保护的内部包
3. **创建`pkg/`目录**: 如果有可被外部使用的库代码
4. **创建`api/`目录**: 如果有API定义
5. **创建`configs/`目录**: 如果配置文件较多
6. **创建`scripts/`目录**: 如果有构建脚本
7. **创建`docs/`目录**: 如果有详细文档

### 7. 参考资源

- [Standard Go Project Layout](https://github.com/golang-standards/project-layout)
- [Go Project Structure](https://github.com/golang/go/wiki/Projects)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

