#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="$HOME/.workspace-manager"
CONFIG_DIR="$HOME/.config/workspace"
BIN_DIR="$HOME/.local/bin"

INTERACTIVE=true
if [ ! -t 0 ]; then
    INTERACTIVE=false
    echo "[Pipe Mode] Auto-removing all WSM components..."
fi

echo "========================================="
echo "  WSM Workspace Manager — Uninstaller"
echo "========================================="
echo ""

# ── Detect installed version ──────────────────────────────────

DETECTED=""

if [ -f "$HOME/.wsm" ] || [ -f "$HOME/.wsm-manager" ]; then
    DETECTED="cli"
elif [ -L "${BIN_DIR}/wsm" ] || [ -f "$HOME/.wsm-complete" ]; then
    DETECTED="python-curses"
fi

echo "Detected: ${DETECTED:-unknown}"
echo ""

# ── Remove shell hooks ──────────────────────────────────────────

remove_hook() {
    local rc_file="$1"
    if [ -f "$rc_file" ]; then
        sed -i '/# Workspace Manager/d' "$rc_file" 2>/dev/null || true
        sed -i '/export PATH="$HOME\/.local\/bin:$PATH"/d' "$rc_file" 2>/dev/null || true
        sed -i '/source.*\.wsm-complete/d' "$rc_file" 2>/dev/null || true
        sed -i '/source.*\.wsm/d' "$rc_file" 2>/dev/null || true
        echo "  Cleaned $rc_file"
    fi
}

remove_hook "$HOME/.bashrc"
remove_hook "$HOME/.zshrc"

# ── Remove CLI files ────────────────────────────────────────────

for f in "$HOME/.wsm" "$HOME/.wsm-manager" "$HOME/.wsm-complete"; do
    if [ -f "$f" ]; then
        echo "Removing $f..."
        rm -f "$f"
    fi
done

# ── Remove Python symlinks ──────────────────────────────────────

for bin in wsm wsm-tui; do
    if [ -L "${BIN_DIR}/${bin}" ]; then
        echo "Removing ${BIN_DIR}/${bin} symlink..."
        rm -f "${BIN_DIR}/${bin}"
    fi
done

# ── Remove repo directory ──────────────────────────────────────

if [ -d "$REPO_DIR" ]; then
    DEL_REPO="n"
    if $INTERACTIVE; then
        read -r -p "Delete $REPO_DIR? [y/N]: " DEL_REPO
    else
        DEL_REPO="y"
    fi
    if [[ "$DEL_REPO" =~ ^[Yy]$ ]]; then
        echo "Removing $REPO_DIR..."
        rm -rf "$REPO_DIR"
    else
        echo "Skipped."
    fi
else
    echo "$REPO_DIR not found — skipped."
fi

# ── Remove configs ─────────────────────────────────────────────

if [ -d "$CONFIG_DIR" ]; then
    DEL_CONF="n"
    if $INTERACTIVE; then
        read -r -p "Delete project configs in $CONFIG_DIR? [y/N]: " DEL_CONF
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
