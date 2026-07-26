#!/usr/bin/env bash
set -euo pipefail

REPO_URL="https://github.com/cy83rt00n/workspace-manager"
TARGET_REPO_DIR="$HOME/.workspace-manager"

echo "========================================="
echo "  WSM Workspace Manager — Installer"
echo "========================================="
echo ""

# ── Choose version ─────────────────────────────────────────────

echo "Which version do you want to install?"
echo "  1) CLI (Bash)        — minimal, pure Bash, no Python"
echo "  2) Python-curses     — full TUI, Python-based"
echo ""
echo -n "Choice [1]: "
if [ -t 0 ]; then
    read -r choice
elif [ -e /dev/tty ]; then
    read -r choice < /dev/tty
else
    echo ""
    echo "[Headless mode] Defaulting to CLI (Bash)."
    echo "Use --branch flag or specific install URL to pick version."
    choice="1"
fi
choice="${choice:-1}"

case "$choice" in
    2)
        BRANCH="python-curses"
        VERSION_NAME="Python-curses"
        ;;
    *)
        BRANCH="cli"
        VERSION_NAME="CLI (Bash)"
        ;;
esac

echo ""
echo "Installing WSM $VERSION_NAME..."
echo ""

# ── Clone / pull ───────────────────────────────────────────────

mkdir -p "$TARGET_REPO_DIR"

if [ ! -t 0 ]; then
    echo "[Network Mode] Cloning repository ($BRANCH)..."
    if [ -d "$TARGET_REPO_DIR/.git" ]; then
        cd "$TARGET_REPO_DIR" && git pull
    else
        git clone --branch "$BRANCH" "$REPO_URL" "$TARGET_REPO_DIR"
    fi
else
    echo "[Local Mode] Deploying from local source tree..."
    CURRENT_SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    # In local mode, we're already on the right branch
    rsync -a "$CURRENT_SOURCE_DIR"/ "$TARGET_REPO_DIR"/ \
        --exclude='.git' --exclude='.gitignore' --exclude='.github'
fi

# ── Delegate to version-specific installer ──────────────────────

cd "$TARGET_REPO_DIR"
chmod +x install.sh
exec bash install.sh
