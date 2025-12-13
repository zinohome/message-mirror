# Message Mirror Web Frontend

现代化React+Vite前端，用于实时配置管理和监控。

## 功能

- ✅ **Overview**: 实时统计（消费/生产消息数、字节数、错误数）
- ✅ **Configuration**: JSON配置编辑与重载
- ✅ **Monitoring**: 实时监控表格（WebSocket）
- ✅ **响应式设计**: 支持移动设备
- ✅ **现代UI**: 使用CSS Grid、Flexbox、渐变背景

## 快速开始

### 1. 安装依赖
```bash
cd web/frontend
npm install
```

### 2. 开发模式
```bash
npm run dev
```
访问 http://localhost:3000

### 3. 生产构建
```bash
npm run build
```
输出到 `web/dist`

## 架构

### React Components
- **App.jsx**: 主应用组件，包含Tab导航和内容区域
- **API Integration**: axios实现HTTP请求，WebSocket实时数据

### CSS模块
- **App.css**: 全局样式、响应式断点、色彩主题
- CSS变量便于后续定制

### Vite配置
- **开发服务器**: 代理后端API到localhost:8080
- **生产构建**: 输出到web/dist（可供Go embed）
- **插件**: React Fast Refresh

## 后端集成

### HTTP API端点
```
GET    /api/config              # 获取配置
POST   /api/config              # 保存配置
POST   /api/config/reload       # 重载配置
GET    /health                   # 健康检查
GET    /ready                    # 就绪检查
GET    /metrics                  # Prometheus指标
```

### WebSocket端点
```
WS     /ws/stats                 # 实时统计（JSON流）
```

### 统计数据格式
```json
{
  "messages_consumed": 1000,
  "messages_produced": 1000,
  "bytes_consumed": 102400,
  "bytes_produced": 102400,
  "errors": 0,
  "uptime": "1h 30m"
}
```

## 后端代码修改（Go）

### 1. 添加Web UI服务（http_server.go）

```go
// 静态文件服务（Web UI）
http.FileServer(http.Dir("web/dist"))

// WebSocket统计端点
upgrader := websocket.Upgrader{}
http.HandleFunc("/ws/stats", func(w http.ResponseWriter, r *http.Request) {
    ws, _ := upgrader.Upgrade(w, r, nil)
    defer ws.Close()
    
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        stats := mm.GetStats()
        json.NewEncoder(ws).Encode(stats)
    }
})
```

### 2. 配置API端点（http_server.go）

```go
// GET /api/config
http.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
    if r.Method != "GET" {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    configBytes, _ := configManager.GetConfigJSON()
    w.Write(configBytes)
})

// POST /api/config (update)
// POST /api/config/reload
```

## 样式定制

编辑 `src/App.css` 中的CSS变量：

```css
:root {
  --primary-color: #1890ff;      /* 主色调 */
  --success-color: #52c41a;      /* 成功色 */
  --error-color: #ff4d4f;        /* 错误色 */
  --border-radius: 8px;          /* 圆角 */
}
```

## 响应式断点

- **Desktop**: 1200px+
- **Tablet**: 768px - 1199px
- **Mobile**: < 768px

## 性能优化

- ✅ 生产构建压缩（Terser）
- ✅ 代码分割（动态导入）
- ✅ WebSocket实时数据（无轮询）
- ✅ CSS变量减少冗余

## 浏览器兼容性

- Chrome 90+
- Firefox 88+
- Safari 15+
- Edge 90+

## 常见问题

### Q: 开发时访问出现CORS错误？
A: 检查vite.config.js的proxy配置，确保后端运行在localhost:8080

### Q: WebSocket连接失败？
A: 确认后端已实现WebSocket端点，检查ws://地址是否正确

### Q: 生产构建后无法加载？
A: 确保运行`npm run build`生成dist目录，Go服务器配置正确的静态文件路径

## 未来计划

- [ ] 深色主题切换
- [ ] 实时日志查看
- [ ] 插件管理界面
- [ ] 性能指标图表（Chart.js）
- [ ] 多语言支持（i18n）
