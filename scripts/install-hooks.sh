#!/bin/bash
# Copyright 2025 Scott Friedman. All rights reserved.
# Install Git hooks for maintaining Go Report Card grade A

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
GIT_HOOKS_DIR="$PROJECT_ROOT/.git/hooks"
CUSTOM_HOOKS_DIR="$PROJECT_ROOT/scripts/git-hooks"

echo "🔧 Installing Git hooks for Go Report Card grade A..."

# Check if we're in a git repository
if [ ! -d "$PROJECT_ROOT/.git" ]; then
    echo "❌ Not in a Git repository"
    exit 1
fi

# Create hooks directory if it doesn't exist
mkdir -p "$GIT_HOOKS_DIR"

# Install pre-commit hook
echo "📋 Installing pre-commit hook..."
cp "$CUSTOM_HOOKS_DIR/pre-commit" "$GIT_HOOKS_DIR/pre-commit"
chmod +x "$GIT_HOOKS_DIR/pre-commit"

echo "✅ Git hooks installed successfully!"
echo ""
echo "The pre-commit hook will now:"
echo "  ✅ Check code formatting"
echo "  ✅ Run go vet"
echo "  ✅ Run linter (if available)"
echo "  ✅ Run unit tests"
echo "  ✅ Check test coverage (requires ≥80%)"
echo "  ✅ Run security checks (if available)"
echo "  ✅ Check for TODO/FIXME comments"
echo "  ✅ Verify go.mod is tidy"
echo ""
echo "This ensures your commits maintain Go Report Card grade A! 🎉"