#!/usr/bin/env bash
set -euo pipefail

TARGET_REPO_DIR="$HOME/.workspace-manager"

echo "Initializing Workspace Manager installation pipeline..."

# Определение типа запуска (сетевой стрим или локальная папка)
if [ ! -t 0 ]; then
    echo "[Network Mode] Cloning repository from upstream remote..."
    if [ -d "$TARGET_REPO_DIR" ]; then
        cd "$TARGET_REPO_DIR" && git pull
    else
        git clone "https://github.com/cy83rt00n/workspace-manager" "$TARGET_REPO_DIR"
    fi
else
    echo "[Local Mode] Deploying from local source tree..."
    CURRENT_SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    mkdir -p "$TARGET_REPO_DIR"
    cp -r "$CURRENT_SOURCE_DIR"/* "$TARGET_REPO_DIR/"
fi

# Копирование ядра в домашнюю директорию
cp "$TARGET_REPO_DIR/.wsm" "$HOME/.wsm"
cp "$TARGET_REPO_DIR/.wsm-manager" "$HOME/.wsm-manager"

# Инъекция хука инициализации в командные оболочки
inject_hook() {
    local rc_file="$1"
    if [ -f "$rc_file" ]; then
        if ! grep -q "source \$HOME/.wsm" "$rc_file"; then
            echo -e "\n# Workspace Manager Hook\nif [ -f \"\$HOME/.wsm\" ]; then source \"\$HOME/.wsm\"; fi" >> "$rc_file"
            echo "Hook injected successfully into $rc_file"
        fi
    fi
}

inject_hook "$HOME/.bashrc"
inject_hook "$HOME/.zshrc"

echo "=========================================================="
echo "Installation complete!"
echo "Please reload your profile: 'source ~/.bashrc' or 'source ~/.zshrc'"
echo "Commands now available:"
echo "  -> wsm [command] [project]  (CLI Utility)"
echo "  -> wsm-manager              (Interactive Configuration Menu)"
echo "=========================================================="
