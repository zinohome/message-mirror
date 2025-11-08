#!/bin/bash
# 项目结构重构脚本

set -e

PROJECT_ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$PROJECT_ROOT"

echo "开始项目结构重构..."
echo ""

# 阶段1: 工具包迁移
echo "阶段1: 迁移工具包..."

# version
echo "  迁移 version..."
mkdir -p internal/pkg/version
mv version.go internal/pkg/version/version.go
sed -i '' 's/^package main$/package version/' internal/pkg/version/version.go

# 更新所有引用version的文件
find . -name "*.go" -type f ! -path "./internal/*" -exec sed -i '' 's/version\.Version/version.Version/g' {} \;
find . -name "*.go" -type f ! -path "./internal/*" -exec sed -i '' 's/version\.BuildTime/version.BuildTime/g' {} \;
find . -name "*.go" -type f ! -path "./internal/*" -exec sed -i '' 's/version\.GitCommit/version.GitCommit/g' {} \;

echo "  阶段1完成"
echo ""

echo "重构脚本已创建，请手动执行或使用自动化工具完成剩余步骤。"

