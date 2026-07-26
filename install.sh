#!/usr/bin/env bash
set -euo pipefail

TARGET_REPO_DIR="$HOME/.workspace-manager"
BIN_DIR="$HOME/.local/bin"
DEPS_FILE="$TARGET_REPO_DIR/.wsm-deps"

echo "Initializing WSM Python-curses installation..."

mkdir -p "$TARGET_REPO_DIR"

# ── Dependency check & install ──────────────────────────────────

REQUIRED_PKGS=(
    "sshfs:FUSE-based SSH filesystem"
    "nc:netcat / TCP connectivity check"
    "python3:Python 3 interpreter"
)

BASE_UTILS=("ssh" "ssh-keygen" "mountpoint" "fuser" "umount")

detect_pkg_manager() {
    if command -v apt-get &>/dev/null; then echo "apt"
    elif command -v dnf &>/dev/null; then echo "dnf"
    elif command -v yum &>/dev/null; then echo "yum"
    elif command -v pacman &>/dev/null; then echo "pacman"
    elif command -v brew &>/dev/null; then echo "brew"
    else echo ""; fi
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
        python3)
            case "$PKG_MANAGER" in
                apt)    echo "python3" ;;
                dnf|yum) echo "python3" ;;
                pacman) echo "python" ;;
                brew)   echo "python3" ;;
            esac ;;
    esac
}

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
            echo "       WARNING: No supported package manager. Install '$util' manually."
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

if [ ${#INSTALLED_DEPS[@]} -gt 0 ]; then
    printf '%s\n' "${INSTALLED_DEPS[@]}" > "$DEPS_FILE"
fi

echo ""

# ── Deploy ──────────────────────────────────────────────────────

if [ ! -t 0 ]; then
    echo "[Network Mode] Cloning repository..."
    if [ -d "$TARGET_REPO_DIR" ]; then
        cd "$TARGET_REPO_DIR" && git pull
    else
        git clone --branch python-curses "https://github.com/cy83rt00n/workspace-manager" "$TARGET_REPO_DIR"
    fi
else
    echo "[Local Mode] Deploying from local source tree..."
    CURRENT_SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    mkdir -p "$TARGET_REPO_DIR"
    rsync -a "$CURRENT_SOURCE_DIR"/ "$TARGET_REPO_DIR"/ \
        --exclude='.git' --exclude='.gitignore'
fi

# ── Install executables ─────────────────────────────────────────

mkdir -p "$BIN_DIR"
for mod in wsm_core.py wsm_render.py wsm_cli.py wsm_tui.py; do
    if [ -f "$TARGET_REPO_DIR/$mod" ]; then
        chmod +x "$TARGET_REPO_DIR/$mod"
    fi
done

# Always symlink TUI
ln -sf "$TARGET_REPO_DIR/wsm_tui.py" "${BIN_DIR}/wsm-tui"

# Ask which interface to use as default 'wsm'
echo ""
echo "Select default interface for 'wsm' command:"
echo "  1) CLI  — fast terminal commands"
echo "  2) TUI  — interactive Midnight Commander style"
echo -n "Choice [1]: "
read -r choice
choice="${choice:-1}"

case "$choice" in
    2)
        ln -sf "$TARGET_REPO_DIR/wsm_tui.py" "${BIN_DIR}/wsm"
        echo "Default: wsm → TUI"
        ;;
    *)
        ln -sf "$TARGET_REPO_DIR/wsm_cli.py" "${BIN_DIR}/wsm"
        echo "Default: wsm → CLI  (wsm-tui available separately)"
        ;;
esac

# ── Shell PATH hook ─────────────────────────────────────────────

cp "$TARGET_REPO_DIR/.wsm-complete" "$HOME/.wsm-complete"

inject_hook() {
    local rc_file="$1"
    if [ -f "$rc_file" ]; then
        if ! grep -q ">>> WSM BEGIN >>>" "$rc_file"; then
            cat << 'EOF' >> "$rc_file"

# >>> WSM BEGIN >>>
export PATH="$HOME/.local/bin:$PATH"
if [ -f "$HOME/.wsm-complete" ]; then source "$HOME/.wsm-complete"; fi
# <<< WSM END <<<
EOF
            echo "Hook injected into $rc_file"
        fi
    fi
}

inject_hook "$HOME/.bashrc"
inject_hook "$HOME/.zshrc"

echo "=========================================================="
echo "WSM Python-curses installation complete!"
echo "Reload: source ~/.bashrc  (or: source ~/.zshrc)"
echo "Commands:"
echo "  wsm       [command] [project]   (default interface)"
echo "  wsm-tui                         (interactive TUI)"
echo "=========================================================="
