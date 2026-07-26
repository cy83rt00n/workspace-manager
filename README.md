# WSM — Workspace Manager

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)
[![Shell](https://img.shields.io/badge/Shell-Bash%20%7C%20Zsh-1f425f.svg)](#)
[![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20macOS-blue.svg)](#)

Shell-инструмент для монтирования удалённых проектов через SSHFS и запуска в IDE одной командой.

A shell tool for mounting remote projects via SSHFS and opening them in your IDE with a single command.

---

- [English](#english)
- [Русский](#русский)

---

## English

### Overview

Mount any remote directory via SSHFS, work with it as if local, and launch your IDE — all with a single `wsm run` command.

### Features

- 🔗 **One command** — `wsm run <project>` mounts and opens IDE
- 🛡️ **Safe configs** — TOML format, regex parser, no injection risks
- 🌐 **Network check** — resolves SSH alias and verifies host before mount
- 🧹 **Clean unmount** — kills file locks, lazy-unmount fallback
- ⌨️ **Autocomplete** — Bash & Zsh: commands and project names
- 🖥️ **Interactive manager** — `wsm-manager` for configs, desktop entries, SSH keys
- 📋 **XDG Desktop Actions** — launch projects from IDE right-click menu

### Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/cy83rt00n/workspace-manager/main/install.sh | bash
source ~/.bashrc   # or: ~/.zshrc
```

Requirements (`sshfs`, `nc`) are installed automatically. See [manual install](#manual-install) for alternatives.

### What Gets Deployed

| Path | Purpose |
|---|---|
| `~/.wsm` | Core: `wsm` function, parser, autocomplete |
| `~/.wsm-manager` | Interactive `wsm-manager` function |
| `~/.workspace-manager` | Installer, uninstaller, repo copy |

Hooks are injected into `~/.bashrc` and `~/.zshrc`.

### Uninstall

```bash
curl -fsSL https://raw.githubusercontent.com/cy83rt00n/workspace-manager/main/uninstall.sh | bash
```

Prompts to keep or delete configs and repo. Shows removal commands for auto-installed packages.

### Configuration

Project configs live in `~/.config/workspace/<name>.conf` in TOML format:

```toml
remote_path = "ssh-alias:/remote/project/path"
local_mount = "/home/user/.workspace/my-project"
editor_cmd = "zed"
```

| Field | Description |
|---|---|
| `remote_path` | SSH alias + remote path (`alias:/path`) |
| `local_mount` | Local mount point |
| `editor_cmd` | Editor / IDE launch command |

### Commands

```bash
wsm mount   <project>   # Check network, then mount
wsm run     <project>   # Mount + open in IDE
wsm unmount <project>   # Safely unmount
wsm help                # Show help
```

Shortcuts: `m`, `r`, `u`.

### Interactive Manager

```bash
wsm-manager
```

Four options:

1. **Create / Update** a project config
2. **Generate** an XDG Desktop Action snippet
3. **Generate** an ED25519 SSH keypair
4. **Exit**

### Desktop Actions

Paste the generated snippet into your IDE's `.desktop` file at `~/.local/share/applications/`:

```ini
Actions=my-project;

[Desktop Action my-project]
Name=Open Remote: my-project
Exec=bash -c 'source $HOME/.wsm && wsm run my-project'
Identifier=my-project
```

### Manual Install

```bash
git clone https://github.com/cy83rt00n/workspace-manager.git
cd workspace-manager
chmod +x install.sh
./install.sh
source ~/.bashrc
```

---

## Русский

### Обзор

Монтируйте любую удалённую директорию через SSHFS, работайте как с локальной и запускайте IDE — всё одной командой `wsm run`.

### Возможности

- 🔗 **Одна команда** — `wsm run <проект>` монтирует и открывает IDE
- 🛡️ **Безопасные конфиги** — TOML, regex-парсер, без инъекций
- 🌐 **Проверка сети** — резолв SSH-алиаса и проверка хоста перед mount
- 🧹 **Чистый unmount** — снятие блокировок, fallback на lazy-unmount
- ⌨️ **Автодополнение** — Bash и Zsh: команды и имена проектов
- 🖥️ **Интерактивный менеджер** — `wsm-manager`: конфиги, desktop, ключи
- 📋 **XDG Desktop Actions** — запуск проектов из контекстного меню редактора

### Быстрая установка

```bash
curl -fsSL https://raw.githubusercontent.com/cy83rt00n/workspace-manager/main/install.sh | bash
source ~/.bashrc   # или: ~/.zshrc
```

Зависимости (`sshfs`, `nc`) устанавливаются автоматически. См. [ручную установку](#ручная-установка).

### Что разворачивается

| Путь | Назначение |
|---|---|
| `~/.wsm` | Ядро: функция `wsm`, парсер, автодополнение |
| `~/.wsm-manager` | Интерактивная функция `wsm-manager` |
| `~/.workspace-manager` | Установщик, деинсталлер, копия репо |

Хуки добавляются в `~/.bashrc` и `~/.zshrc`.

### Удаление

```bash
curl -fsSL https://raw.githubusercontent.com/cy83rt00n/workspace-manager/main/uninstall.sh | bash
```

Запрашивает подтверждение на удаление конфигов и репо. Показывает команды для удаления автовстановленных пакетов.

### Конфигурация

Конфиги проектов в `~/.config/workspace/<имя>.conf`, формат TOML:

```toml
remote_path = "ssh-alias:/remote/project/path"
local_mount = "/home/user/.workspace/my-project"
editor_cmd = "zed"
```

| Поле | Описание |
|---|---|
| `remote_path` | SSH-алиас + удалённый путь (`алиас:/путь`) |
| `local_mount` | Локальная точка монтирования |
| `editor_cmd` | Команда запуска редактора / IDE |

### Команды

```bash
wsm mount   <проект>   # Проверить сеть и смонтировать
wsm run     <проект>   # Смонтировать и открыть в IDE
wsm unmount <проект>   # Безопасно размонтировать
wsm help               # Справка
```

Сокращения: `m`, `r`, `u`.

### Интерактивный менеджер

```bash
wsm-manager
```

Четыре пункта:

1. **Создать / Обновить** конфиг проекта
2. **Сгенерировать** фрагмент XDG Desktop Action
3. **Сгенерировать** ключевую пару ED25519
4. **Выход**

### Desktop Actions

Вставьте сгенерированный фрагмент в `.desktop`-файл редактора в `~/.local/share/applications/`:

```ini
Actions=my-project;

[Desktop Action my-project]
Name=Open Remote: my-project
Exec=bash -c 'source $HOME/.wsm && wsm run my-project'
Identifier=my-project
```

### Ручная установка

```bash
git clone https://github.com/cy83rt00n/workspace-manager.git
cd workspace-manager
chmod +x install.sh
./install.sh
source ~/.bashrc
```

---

## License

MIT — see [LICENSE](./LICENSE)
