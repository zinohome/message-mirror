# 开发规范

## 代码风格

### Go 代码规范

#### 1. 命名规范

- **包名**: 小写，简短，有意义
  ```go
  package config
  package metrics
  ```

- **函数名**: 驼峰命名，公开函数首字母大写
  ```go
  func LoadConfig(path string) (*Config, error)
  func validateConfig(c *Config) error
  ```

- **变量名**: 驼峰命名，简短有意义
  ```go
  var configPath string
  var messageCount int64
  ```

- **常量**: 全大写，下划线分隔
  ```go
  const DefaultWorkerCount = 4
  const MAX_RETRY_COUNT = 3
  ```

- **接口名**: 通常以 `-er` 结尾
  ```go
  type SourcePlugin interface {
      Initialize(config map[string]interface{}) error
      Start() error
      Stop() error
      Messages() <-chan *Message
  }
  ```

#### 2. 代码组织

- **文件结构**:
  ```go
  package main
  
  import (
      // 标准库
      "fmt"
      "time"
      
      // 第三方库
      "github.com/IBM/sarama"
      
      // 本地包
  )
  
  // 类型定义
  type Config struct {
      // ...
  }
  
  // 常量
  const (
      // ...
  )
  
  // 变量
  var (
      // ...
  )
  
  // 函数
  func main() {
      // ...
  }
  ```

#### 3. 错误处理

- **始终检查错误**:
  ```go
  config, err := LoadConfig(path)
  if err != nil {
      return fmt.Errorf("加载配置失败: %w", err)
  }
  ```

- **使用错误包装**:
  ```go
  if err != nil {
      return fmt.Errorf("处理消息失败: %w", err)
  }
  ```

- **提供有意义的错误信息**:
  ```go
  // 好
  return fmt.Errorf("无法连接到Kafka集群 %v: %w", brokers, err)
  
  // 不好
  return err
  ```

#### 4. 注释规范

- **包注释**: 每个包都应该有包注释
  ```go
  // Package config 提供配置加载和管理功能
  package config
  ```

- **公开函数注释**: 所有公开函数都应该有注释
  ```go
  // LoadConfig 从指定路径加载配置文件
  // 支持 YAML、JSON、TOML 格式
  // 如果文件不存在，返回默认配置
  func LoadConfig(path string) (*Config, error) {
      // ...
  }
  ```

- **复杂逻辑注释**: 对复杂逻辑添加注释
  ```go
  // 使用指数退避策略计算重试延迟
  // delay = baseDelay * (2 ^ retryCount) + jitter
  delay := baseDelay * time.Duration(1<<retryCount)
  delay += time.Duration(rand.Intn(int(jitter)))
  ```

#### 5. 并发安全

- **使用互斥锁保护共享状态**:
  ```go
  type Stats struct {
      mu   sync.RWMutex
      count int64
  }
  
  func (s *Stats) Increment() {
      s.mu.Lock()
      defer s.mu.Unlock()
      s.count++
  }
  ```

- **使用 channel 进行通信**:
  ```go
  msgChan := make(chan *Message, 100)
  ```

- **使用 context 控制生命周期**:
  ```go
  ctx, cancel := context.WithCancel(context.Background())
  defer cancel()
  ```

#### 6. 资源管理

- **及时关闭资源**:
  ```go
  file, err := os.Open(path)
  if err != nil {
      return err
  }
  defer file.Close()
  ```

- **使用 defer 确保清理**:
  ```go
  func (p *Plugin) Start() error {
      p.conn, err = amqp.Dial(url)
      if err != nil {
          return err
      }
      defer func() {
          if err != nil {
              p.conn.Close()
          }
      }()
      // ...
  }
  ```

### 测试规范

#### 1. 测试文件命名

- 测试文件以 `_test.go` 结尾
- 测试文件与被测试文件在同一包中

#### 2. 测试函数命名

```go
func TestFunctionName(t *testing.T) {
    // ...
}

func TestFunctionName_Scenario(t *testing.T) {
    // ...
}

func BenchmarkFunctionName(b *testing.B) {
    // ...
}
```

#### 3. 测试结构

```go
func TestLoadConfig(t *testing.T) {
    // Arrange (准备)
    testCases := []struct {
        name    string
        path    string
        wantErr bool
    }{
        {"valid config", "testdata/config.yaml", false},
        {"invalid path", "nonexistent.yaml", true},
    }
    
    // Act & Assert (执行和断言)
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            config, err := LoadConfig(tc.path)
            if (err != nil) != tc.wantErr {
                t.Errorf("LoadConfig() error = %v, wantErr %v", err, tc.wantErr)
                return
            }
            if !tc.wantErr && config == nil {
                t.Error("LoadConfig() returned nil config")
            }
        })
    }
}
```

#### 4. 测试覆盖率

- 目标覆盖率: 核心组件 > 80%
- 使用 `go test -cover` 查看覆盖率
- 使用 `go tool cover -html=coverage.out` 查看详细报告

### Git 提交规范

#### 提交信息格式

```
<type>(<scope>): <subject>

<body>

<footer>
```

#### Type 类型

- `feat`: 新功能
- `fix`: 修复bug
- `docs`: 文档更新
- `style`: 代码格式调整（不影响功能）
- `refactor`: 重构
- `test`: 测试相关
- `chore`: 构建/工具相关

#### 示例

```
feat(plugin): 添加RabbitMQ插件支持

实现了RabbitMQ数据源插件，支持从RabbitMQ队列消费消息并写入Kafka。

Closes #123
```

### 代码审查清单

- [ ] 代码符合Go代码规范
- [ ] 所有公开函数都有注释
- [ ] 错误处理完整
- [ ] 并发安全（如需要）
- [ ] 资源正确释放
- [ ] 有相应的测试
- [ ] 测试通过
- [ ] 没有引入新的警告
- [ ] 提交信息符合规范

### 性能优化原则

1. **避免过早优化**: 先确保功能正确
2. **性能测试**: 使用基准测试验证性能
3. **资源使用**: 注意内存和CPU使用
4. **并发优化**: 合理使用goroutine和channel

### 安全规范

1. **敏感信息**: 不要在代码中硬编码密码、密钥
2. **输入验证**: 验证所有外部输入
3. **错误信息**: 不要泄露敏感信息
4. **依赖管理**: 定期更新依赖，修复安全漏洞

