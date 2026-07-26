#!/usr/bin/env bash
set -euo pipefail

TARGET_REPO_DIR="$HOME/.workspace-manager"
BIN_DIR="$HOME/.local/bin"
CONFIG_DIR="$HOME/.config/workspace"

INTERACTIVE=true
if [ ! -t 0 ]; then
    INTERACTIVE=false
    echo "[Pipe Mode] Auto-removing all WSM components..."
fi

echo "========================================="
echo "  WSM Workspace Manager — Uninstaller"
echo "========================================="
echo ""

# 1. Remove shell hooks
remove_hook() {
    local rc_file="$1"
    if [ -f "$rc_file" ]; then
        if grep -q ">>> WSM BEGIN >>>" "$rc_file" 2>/dev/null; then
            echo "Removing WSM hook from $rc_file..."
            sed -i '/>>> WSM BEGIN >>>/,/<<< WSM END <<</d' "$rc_file"
        fi
    fi
}

remove_hook "$HOME/.bashrc"
remove_hook "$HOME/.zshrc"

# 2. Remove symlinks and autocomplete
for bin in wsm wsm-tui; do
    if [ -L "${BIN_DIR}/${bin}" ]; then
        echo "Removing ${BIN_DIR}/${bin} symlink..."
        rm -f "${BIN_DIR}/${bin}"
    fi
done

if [ -f "$HOME/.wsm-complete" ]; then
    echo "Removing $HOME/.wsm-complete..."
    rm -f "$HOME/.wsm-complete"
fi

# 3. Show dependency removal hint
DEPS_FILE="$TARGET_REPO_DIR/.wsm-deps"
if [ -f "$DEPS_FILE" ]; then
    echo ""
    echo "Packages installed by WSM:"
    while read -r pkg; do
        echo "  - $pkg"
    done < "$DEPS_FILE"
    echo ""
    echo "Remove with your package manager, e.g.:"
    echo "  sudo apt-get remove <package>"
    echo "  sudo dnf remove <package>"
    echo "  sudo pacman -R <package>"
    echo "  brew uninstall <package>"
fi

echo ""

# 4. Remove repo directory
if [ -d "$TARGET_REPO_DIR" ]; then
    DEL_REPO="n"
    if $INTERACTIVE; then
        read -p "Delete $TARGET_REPO_DIR? [y/N]: " DEL_REPO
    else
        DEL_REPO="y"
    fi
    if [[ "$DEL_REPO" =~ ^[Yy]$ ]]; then
        echo "Removing $TARGET_REPO_DIR..."
        rm -rf "$TARGET_REPO_DIR"
    else
        echo "Skipped."
    fi
else
    echo "$TARGET_REPO_DIR not found — skipped."
fi

# 5. Optionally remove configs
if [ -d "$CONFIG_DIR" ]; then
    DEL_CONF="n"
    if $INTERACTIVE; then
        read -p "Delete project configs in $CONFIG_DIR? [y/N]: " DEL_CONF
    else
        DEL_CONF="y"
    fi
    if [[ "$DEL_CONF" =~ ^[Yy]$ ]]; then
        echo "Removing $CONFIG_DIR..."
        rm -rf "$CONFIG_DIR"
    else
        echo "Skipped."
    fi
else
    echo "$CONFIG_DIR not found — skipped."
fi

echo ""
echo "Uninstall complete."
echo "Run: source ~/.bashrc"
