#!/bin/bash
# install-hooks.sh: 安装 git hooks

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HOOKS_DIR="$SCRIPT_DIR/git-hooks"
GIT_HOOKS_DIR="$(git rev-parse --git-dir)/hooks"

echo "📦 正在安装 git hooks..."
echo ""

# 检查 git hooks 目录是否存在
if [ ! -d "$GIT_HOOKS_DIR" ]; then
    echo "❌ 错误：找不到 .git/hooks 目录"
    echo "请确保在 git 仓库根目录运行此脚本"
    exit 1
fi

# 安装 commit-msg hook
if [ -f "$HOOKS_DIR/commit-msg" ]; then
    cp "$HOOKS_DIR/commit-msg" "$GIT_HOOKS_DIR/commit-msg"
    chmod +x "$GIT_HOOKS_DIR/commit-msg"
    echo "✓ 已安装 commit-msg hook"
else
    echo "⚠️  警告：找不到 commit-msg hook"
fi

# 安装 pre-commit hook
if [ -f "$HOOKS_DIR/pre-commit" ]; then
    cp "$HOOKS_DIR/pre-commit" "$GIT_HOOKS_DIR/pre-commit"
    chmod +x "$GIT_HOOKS_DIR/pre-commit"
    echo "✓ 已安装 pre-commit hook"
else
    echo "⚠️  警告：找不到 pre-commit hook"
fi

echo ""
echo "✅ Git hooks 安装完成！"
echo ""
echo "这些 hooks 将会："
echo "  - 验证提交消息格式 (type(scope): description)"
echo "  - 防止在 main 分支提交服务代码"
echo ""
echo "提交消息格式示例："
echo "  feat(pkg): add new middleware"
echo "  fix(cmd): correct flag parsing"
echo "  docs(example): update guide"
echo ""
