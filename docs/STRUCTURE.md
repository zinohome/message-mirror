# 文档结构说明

## 📂 目录组织

```
docs/
├── README.md                           # 文档导航中心
├── STRUCTURE.md                        # 本文件：文档结构说明
│
├── architecture/                       # 架构设计文档
│   ├── system-architecture.md         # 系统架构设计
│   ├── PROJECT_STRUCTURE.md           # 项目代码结构
│   ├── PROJECT_OVERVIEW.md            # 项目概述
│   └── project-structure-optimization.md  # 结构优化方案
│
├── deployment/                         # 部署文档
│   ├── DEPLOYMENT.md                  # 部署指南
│   └── DEPLOYMENT_READY.md            # 部署就绪检查清单
│
├── releases/                           # 发布管理
│   └── RELEASE_NOTES.md               # 版本发布说明
│
├── development/                        # 开发文档
│   ├── coding-standards.md            # 代码规范
│   └── development-guide.md           # 开发指南
│
├── refactoring/                        # 重构文档
│   ├── REFACTORING_GUIDE.md           # 重构指南
│   ├── REFACTORING_PLAN.md            # 重构计划
│   └── REFACTORING_STATUS.md          # 重构状态
│
├── testing/                            # 测试文档
│   ├── E2E_TESTING_GUIDE.md           # E2E 测试指南
│   ├── E2E_TEST_FINAL_REPORT.md       # E2E 测试报告
│   └── WEBSOCKET_TESTING_GUIDE.md     # WebSocket 测试指南
│
├── progress/                           # 进度报告
│   ├── PROGRESS_REPORT.md             # 整体进度
│   ├── COMPLETION_PHASE3.md           # Phase 3 完成报告
│   ├── PHASE3_COMPLETION_REPORT.md    # Phase 3 详细报告
│   ├── PHASE4_COMPLETION_REPORT.md    # Phase 4 完成报告
│   ├── PHASE4_SUMMARY.md              # Phase 4 总结
│   ├── PHASE5_COMPLETION_REPORT.md    # Phase 5 完成报告
│   ├── PHASE5_FINAL_REPORT.md         # Phase 5 最终报告
│   ├── PHASE6_COMPLETION_REPORT.md    # Phase 6 完成报告
│   ├── SESSION_COMPLETION_SUMMARY.md  # 会话完成总结
│   ├── SESSION_SUMMARY.md             # 总体会话总结
│   ├── SESSION_SUMMARY_PHASE5_FINAL.md # Phase 5 会话总结
│   └── SESSION_SUMMARY_PHASE6.md      # Phase 6 会话总结
│
└── api/                                # API 文档
    └── web-ui-api.md                  # Web UI API 文档
```

## 🏠 根目录文档（保留）

```
/
├── README.md                           # 项目主页（必须保留）
├── CHANGELOG.md                        # 变更日志（标准文件）
├── TODO.md                             # 待办事项（项目管理）
└── VERSION                             # 版本号文件（非 markdown）
```

## 📋 分类说明

### 1. **architecture/** - 架构设计
存放系统架构、设计方案、项目结构等技术设计文档。

### 2. **deployment/** - 部署文档
包含生产环境部署指南、部署检查清单等运维相关文档。

### 3. **releases/** - 发布管理
版本发布说明、版本历史、发布计划等。

### 4. **development/** - 开发文档
开发规范、开发指南、环境搭建等开发者文档。

### 5. **refactoring/** - 重构文档
重构计划、重构记录、重构经验总结。

### 6. **testing/** - 测试文档
测试指南、测试报告、测试用例等质量保证文档。

### 7. **progress/** - 进度报告
项目进度跟踪、阶段总结、会话记录等项目管理文档。

### 8. **api/** - API 文档
接口文档、API 参考、集成指南等。

## 🎯 命名规范

### 推荐命名风格
- **kebab-case**: `system-architecture.md` - 推荐用于新文档
- **UPPER_SNAKE_CASE**: `PROJECT_OVERVIEW.md` - 历史遗留，保持一致性

### 文件命名原则
1. 使用英文，清晰描述内容
2. 避免使用空格和特殊字符
3. 保持一致的命名风格（同一目录下）
4. 文件名应具有自解释性

## 📝 文档维护

### 添加新文档
1. 确定文档分类，放入对应目录
2. 更新 `docs/README.md` 导航
3. 如需要，更新本文件的目录树

### 删除过期文档
1. 创建 `docs/archive/` 目录（如不存在）
2. 移动过期文档到归档目录
3. 更新 `docs/README.md` 导航
4. 在归档目录添加 `README.md` 说明归档原因

### 重命名文档
1. 更新所有引用该文档的链接
2. 更新 `docs/README.md` 导航
3. 提交时说明重命名原因

## 🔍 查找文档

### 按主题查找
参考 [docs/README.md](README.md) 的文档导航部分

### 按类型查找
直接进入对应的分类目录

### 全文搜索
```bash
# 在所有文档中搜索关键字
grep -r "关键字" docs/

# 只搜索 markdown 文件
find docs/ -name "*.md" -exec grep -H "关键字" {} \;
```

## 📊 文档统计

- **总目录数**: 8 个分类目录
- **总文档数**: 30+ 个 markdown 文档
- **覆盖范围**: 架构、开发、测试、部署、进度管理等全生命周期

---

**维护者**: Message Mirror Team  
**最后更新**: 2025年12月14日
