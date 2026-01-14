#!/usr/bin/env bash
# Install claude-conductor to ~/.local/bin
# Usage: curl -fsSL https://raw.githubusercontent.com/brittonhayes/claude-conductor/main/install.sh | bash

set -euo pipefail

INSTALL_DIR="${HOME}/.local/bin"
SCRIPT_NAME="claude-conductor"
REPO_URL="https://raw.githubusercontent.com/brittonhayes/claude-conductor/main/bin/launch"

echo "Installing claude-conductor..."

# Create install directory if it doesn't exist
mkdir -p "$INSTALL_DIR"

# Download the script
echo "Downloading from GitHub..."
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$REPO_URL" -o "$INSTALL_DIR/$SCRIPT_NAME"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$INSTALL_DIR/$SCRIPT_NAME" "$REPO_URL"
else
  echo "Error: Neither curl nor wget found. Please install one of them." >&2
  exit 1
fi

# Make it executable
chmod +x "$INSTALL_DIR/$SCRIPT_NAME"

echo "✓ Installed to $INSTALL_DIR/$SCRIPT_NAME"
echo ""

# Check if directory is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
  echo "⚠ $INSTALL_DIR is not in your PATH"
  echo ""
  echo "Add it by running:"
  echo ""

  # Detect shell and provide appropriate command
  SHELL_NAME=$(basename "$SHELL")
  case "$SHELL_NAME" in
    bash)
      echo "  echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.bashrc"
      echo "  source ~/.bashrc"
      ;;
    zsh)
      echo "  echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.zshrc"
      echo "  source ~/.zshrc"
      ;;
    fish)
      echo "  fish_add_path ~/.local/bin"
      ;;
    *)
      echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
      ;;
  esac
  echo ""
else
  echo "✓ $INSTALL_DIR is in your PATH"
  echo ""
fi

# Check dependencies
echo "Checking dependencies..."
MISSING_DEPS=()

if ! command -v tmux >/dev/null 2>&1; then
  MISSING_DEPS+=("tmux")
fi

if ! command -v claude-code >/dev/null 2>&1; then
  MISSING_DEPS+=("claude-code")
fi

if [ ${#MISSING_DEPS[@]} -gt 0 ]; then
  echo "⚠ Missing required dependencies: ${MISSING_DEPS[*]}"
  echo ""
  echo "Install them:"
  for dep in "${MISSING_DEPS[@]}"; do
    case "$dep" in
      tmux)
        echo "  - tmux: https://github.com/tmux/tmux/wiki/Installing"
        ;;
      claude-code)
        echo "  - claude-code: npm install -g @anthropic-ai/claude-code"
        ;;
    esac
  done
  echo ""
else
  echo "✓ All dependencies found"
  echo ""
fi

echo "Installation complete!"
echo ""
echo "Usage:"
echo "  $SCRIPT_NAME \"task1\" \"task2\" \"task3\""
echo "  $SCRIPT_NAME -f tasks.txt"
echo "  $SCRIPT_NAME -h"
echo ""
echo "Run '$SCRIPT_NAME -h' for more options"
