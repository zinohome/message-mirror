# Message Mirror 文档中心

欢迎使用 Message Mirror 项目文档。本目录包含项目的所有技术文档和资料。

## 📚 文档导航

### 核心文档
- [README.md](../README.md) - 项目主页和快速入门
- [CHANGELOG.md](../CHANGELOG.md) - 版本变更日志
- [TODO.md](../TODO.md) - 待办事项清单

### 📐 架构设计
**目录**: [architecture/](architecture/)
- [系统架构设计](architecture/system-architecture.md) - 整体架构和组件设计
- [项目结构](architecture/PROJECT_STRUCTURE.md) - 代码组织和目录结构
- [项目概述](architecture/PROJECT_OVERVIEW.md) - 项目全貌和技术栈
- [项目结构优化](architecture/project-structure-optimization.md) - 结构优化方案

### 🚀 部署文档
**目录**: [deployment/](deployment/)
- [部署指南](deployment/DEPLOYMENT.md) - 生产环境部署说明
- [部署就绪](deployment/DEPLOYMENT_READY.md) - 部署前检查清单

### 📦 发布管理
**目录**: [releases/](releases/)
- [发布说明](releases/RELEASE_NOTES.md) - 版本发布记录

### 🔧 开发指南
**目录**: [development/](development/)
- [开发指南](development/development-guide.md) - 开发环境搭建和工作流
- [代码规范](development/coding-standards.md) - Go 代码规范和最佳实践

### 🏗️ 重构文档
**目录**: [refactoring/](refactoring/)
- [重构指南](refactoring/REFACTORING_GUIDE.md) - 重构历史和经验
- [重构计划](refactoring/REFACTORING_PLAN.md) - 重构方案设计
- [重构状态](refactoring/REFACTORING_STATUS.md) - 重构进度跟踪

### 🧪 测试文档
**目录**: [testing/](testing/)
- [E2E 测试指南](testing/E2E_TESTING_GUIDE.md) - 端到端测试指南
- [E2E 测试报告](testing/E2E_TEST_FINAL_REPORT.md) - 最终测试结果
- [WebSocket 测试指南](testing/WEBSOCKET_TESTING_GUIDE.md) - WebSocket 功能测试

### 📊 进度报告
**目录**: [progress/](progress/)
- [项目进度](progress/PROGRESS_REPORT.md) - 整体进度概览
- **阶段报告**:
  - [Phase 3 完成报告](progress/COMPLETION_PHASE3.md)
  - [Phase 3 详细报告](progress/PHASE3_COMPLETION_REPORT.md)
  - [Phase 4 完成报告](progress/PHASE4_COMPLETION_REPORT.md)
  - [Phase 4 总结](progress/PHASE4_SUMMARY.md)
  - [Phase 5 完成报告](progress/PHASE5_COMPLETION_REPORT.md)
  - [Phase 5 最终报告](progress/PHASE5_FINAL_REPORT.md)
  - [Phase 6 完成报告](progress/PHASE6_COMPLETION_REPORT.md)
- **会话总结**:
  - [会话完成总结](progress/SESSION_COMPLETION_SUMMARY.md)
  - [Phase 5 会话总结](progress/SESSION_SUMMARY_PHASE5_FINAL.md)
  - [Phase 6 会话总结](progress/SESSION_SUMMARY_PHASE6.md)
  - [总体会话总结](progress/SESSION_SUMMARY.md)

### 🔌 API 文档
**目录**: [api/](api/)
- [Web UI API](api/web-ui-api.md) - Web 界面接口文档

## 📖 文档规范

### 文档分类原则
- **根目录**: 仅保留最核心的项目级文档（README、CHANGELOG、TODO、VERSION）
- **architecture/**: 架构设计、系统设计、项目结构相关
- **deployment/**: 部署、运维相关文档
- **releases/**: 版本发布、变更记录
- **development/**: 开发规范、开发指南
- **refactoring/**: 重构相关文档
- **testing/**: 测试指南、测试报告
- **progress/**: 项目进度、阶段总结、会话记录
- **api/**: API 接口文档

### 文档命名规范
- 使用大写字母 + 下划线：`PROJECT_OVERVIEW.md`
- 使用小写字母 + 连字符：`system-architecture.md`
- 保持命名清晰、描述性强

### 文档维护
- 新增文档时更新本导航文件
- 废弃文档移至 `docs/archive/` 目录
- 定期审查和更新过期内容

## 💡 快速查找

| 想要... | 请查看... |
|--------|----------|
| 了解项目 | [项目概述](architecture/PROJECT_OVERVIEW.md) |
| 开始开发 | [开发指南](development/development-guide.md) |
| 部署应用 | [部署指南](deployment/DEPLOYMENT.md) |
| 运行测试 | [测试指南](testing/E2E_TESTING_GUIDE.md) |
| 查看进度 | [进度报告](progress/PROGRESS_REPORT.md) |
| API 集成 | [API 文档](api/web-ui-api.md) |

---

**最后更新**: 2025年12月14日

