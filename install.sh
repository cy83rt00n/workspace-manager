#!/usr/bin/env bash
set -euo pipefail

TARGET_REPO_DIR="$HOME/.workspace-manager"
DEPS_FILE="$TARGET_REPO_DIR/.wsm-deps"

echo "Initializing Workspace Manager installation pipeline..."

# ── Проверка и установка зависимостей ──────────────────────────

REQUIRED_PKGS=(
    "sshfs:FUSE-based SSH filesystem"
    "nc:netcat / TCP connectivity check"
)

# Базовые утилиты (должны быть в системе, предупреждаем если нет)
BASE_UTILS=("ssh" "ssh-keygen" "mountpoint" "fuser" "umount")

detect_pkg_manager() {
    if command -v apt-get &>/dev/null; then
        echo "apt"
    elif command -v dnf &>/dev/null; then
        echo "dnf"
    elif command -v yum &>/dev/null; then
        echo "yum"
    elif command -v pacman &>/dev/null; then
        echo "pacman"
    elif command -v brew &>/dev/null; then
        echo "brew"
    else
        echo ""
    fi
}

pkg_install_cmd() {
    case "$1" in
        apt)    echo "sudo apt-get install -y" ;;
        dnf)    echo "sudo dnf install -y" ;;
        yum)    echo "sudo yum install -y" ;;
        pacman) echo "sudo pacman -S --noconfirm" ;;
        brew)   echo "brew install" ;;
    esac
}

pkg_name() {
    local util="$1"
    case "$util" in
        nc)
            case "$PKG_MANAGER" in
                apt)    echo "netcat-openbsd" ;;
                dnf|yum) echo "nmap-ncat" ;;
                pacman) echo "openbsd-netcat" ;;
                brew)   echo "netcat" ;;
            esac ;;
        sshfs)
            case "$PKG_MANAGER" in
                apt)    echo "sshfs" ;;
                dnf|yum) echo "fuse-sshfs" ;;
                pacman) echo "sshfs" ;;
                brew)   echo "sshfs" ;;
            esac ;;
    esac
}

mkdir -p "$TARGET_REPO_DIR"

echo ""
echo "── Checking dependencies ──"
PKG_MANAGER=$(detect_pkg_manager)
INSTALLED_DEPS=()

for entry in "${REQUIRED_PKGS[@]}"; do
    util="${entry%%:*}"
    desc="${entry##*:}"

    if command -v "$util" &>/dev/null; then
        echo "  [✓] $util — $desc"
    else
        echo "  [✗] $util — $desc"
        if [ -n "$PKG_MANAGER" ]; then
            pkg=$(pkg_name "$util")
            cmd="$(pkg_install_cmd "$PKG_MANAGER") $pkg"
            echo "       Installing via $PKG_MANAGER: $cmd"
            eval "$cmd"
            INSTALLED_DEPS+=("$pkg")
        else
            echo "       WARNING: No supported package manager found. Install '$util' manually."
        fi
    fi
done

for util in "${BASE_UTILS[@]}"; do
    if command -v "$util" &>/dev/null; then
        echo "  [✓] $util"
    else
        echo "  [✗] $util — MISSING! May cause runtime errors."
    fi
done

# Сохраняем список установленных нами пакетов для деинсталлера
if [ ${#INSTALLED_DEPS[@]} -gt 0 ]; then
    printf '%s\n' "${INSTALLED_DEPS[@]}" > "$DEPS_FILE"
fi

echo ""
# ── Клонирование / копирование ──────────────────────────────────

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

# ── Развёртывание ───────────────────────────────────────────────

cp "$TARGET_REPO_DIR/.wsm" "$HOME/.wsm"
cp "$TARGET_REPO_DIR/.wsm-manager" "$HOME/.wsm-manager"

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
