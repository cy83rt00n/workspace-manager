#!/usr/bin/env bash
set -euo pipefail

TARGET_REPO_DIR="$HOME/.workspace-manager"
CONFIG_DIR="$HOME/.config/workspace"

# Определение режима: терминал или curl|bash
INTERACTIVE=true
if [ ! -t 0 ]; then
    INTERACTIVE=false
    echo "[Pipe Mode] Auto-removing all WSM components without prompts..."
fi

echo "========================================="
echo "  WSM Workspace Manager — Uninstaller"
echo "========================================="
echo ""

# 1. Remove init hooks from rc files
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

# 2. Remove core files
if [ -f "$HOME/.wsm" ]; then
    echo "Removing $HOME/.wsm..."
    rm -f "$HOME/.wsm"
fi

if [ -f "$HOME/.wsm-manager" ]; then
    echo "Removing $HOME/.wsm-manager..."
    rm -f "$HOME/.wsm-manager"
fi

# 3. Show how to remove installed system dependencies
DEPS_FILE="$TARGET_REPO_DIR/.wsm-deps"
if [ -f "$DEPS_FILE" ]; then
    echo ""
    echo "The following packages were installed by WSM installer:"
    while read -r pkg; do
        echo "  - $pkg"
    done < "$DEPS_FILE"
    echo ""
    echo "To remove them you can use your package manager, e.g.:"
    echo "  sudo apt-get remove <package>     (Debian/Ubuntu)"
    echo "  sudo dnf remove <package>         (Fedora/RHEL)"
    echo "  sudo pacman -R <package>          (Arch)"
    echo "  brew uninstall <package>          (macOS)"
fi

# 4. Remove deployed repo directory
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
        read -p "Delete all project configs in $CONFIG_DIR? [y/N]: " DEL_CONF
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
echo "Run: source ~/.bashrc   (or: source ~/.zshrc)"
