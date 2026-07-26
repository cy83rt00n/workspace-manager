# WSM — Workspace Manager

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)
[![Python](https://img.shields.io/badge/Python-3.x-3776AB.svg)](#)
[![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20macOS-blue.svg)](#)

Python-инструмент для монтирования удалённых проектов через SSHFS и запуска в IDE одной командой.
Встроенный curses-TUI с навигацией и статусом монтирования.

A Python tool for mounting remote projects via SSHFS and opening them in your IDE with a single command.
Built-in curses TUI with project navigation and mount status indicators.

---

- [English](#english)
- [Русский](#русский)

---

## English

### Overview

Mount any remote directory via SSHFS, work with it as if local, and launch your IDE — all with a single command. The interactive TUI shows project status at a glance.

### Features

- 🔗 **One command** — `wsm run <project>` mounts and opens IDE
- 🖥️ **Curses TUI** — `wsm-tui` with project list, mount status, keyboard navigation
- 🛡️ **Safe configs** — TOML format, Python regex parser
- 🌐 **Network check** — resolves SSH alias, verifies host before mount
- 🧹 **Clean unmount** — kills file locks, lazy-unmount fallback
- 🔑 **Built-in tools** — config wizard, ED25519 key generator, Desktop Action snippets
- 📋 **XDG Desktop Actions** — launch projects from IDE right-click menu

### Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/cy83rt00n/workspace-manager/main/install.sh | bash
source ~/.bashrc
```

Dependencies (`python3`, `sshfs`, `nc`) are auto-installed if missing.

### What Gets Deployed

| Path | Purpose |
|---|---|
| `~/.local/bin/wsm` | CLI: mount, run, unmount |
| `~/.local/bin/wsm-tui` | Interactive curses TUI |
| `~/.workspace-manager` | Installer, uninstaller, repo |

A PATH hook is added to `~/.bashrc` and `~/.zshrc`.

### Uninstall

```bash
curl -fsSL https://raw.githubusercontent.com/cy83rt00n/workspace-manager/main/uninstall.sh | bash
```

### Configuration

Project configs in `~/.config/workspace/<name>.conf` — TOML format:

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

### CLI Commands

```bash
wsm mount   <project>   # Check network, then mount
wsm run     <project>   # Mount + open in IDE
wsm unmount <project>   # Safely unmount
wsm --list              # List configured projects
```

Shortcuts: `m`, `r`, `u`.

### TUI

```bash
wsm-tui
```

```
┌──────────────────────────────────────────────┐
│  WSM Workspace Manager v2.0.0                │
├──────────────────────────────────────────────┤
│ PROJECT              STATUS    REMOTE        │
│ > lavoyage           MOUNTED   dev.lavoyage  │
│   parallax           idle      cy83r.wr.c... │
│   lavoyage-source    idle      dev.lavoyage  │
│ [mounted] /home/.../workspace/dev.lavoyage   │
├──────────────────────────────────────────────┤
│ Enter/r:Run  m:Mount  u:Unmount  c:New  ...  │
└──────────────────────────────────────────────┘
```

| Key | Action |
|---|---|
| `Enter` / `r` | Mount + launch editor |
| `m` | Mount only |
| `u` | Unmount |
| `c` | Create new project config |
| `d` | Show Desktop Action snippet |
| `k` | Generate ED25519 keypair |
| `h` | Help screen |
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `q` | Quit |

### Desktop Actions

Paste the generated snippet into your IDE's `.desktop` file at `~/.local/share/applications/`:

```ini
Actions=my-project;

[Desktop Action my-project]
Name=Open Remote: my-project
Exec=bash -c 'source $HOME/.wsm && wsm run my-project'
Identifier=my-project
```

---

## Русский

### Обзор

Монтируйте удалённую директорию через SSHFS и запускайте IDE одной командой. Интерактивный TUI показывает статус проектов с первого взгляда.

### Возможности

- 🔗 **Одна команда** — `wsm run <проект>` монтирует и открывает IDE
- 🖥️ **Curses TUI** — `wsm-tui`: список проектов, статус, клавиатурная навигация
- 🛡️ **Безопасные конфиги** — TOML, Python regex-парсер
- 🌐 **Проверка сети** — резолв SSH-алиаса, проверка хоста перед mount
- 🧹 **Чистый unmount** — снятие блокировок, fallback на lazy-unmount
- 🔑 **Встроенные инструменты** — мастер конфигов, генератор ED25519, Desktop Action
- 📋 **XDG Desktop Actions** — запуск проектов из контекстного меню редактора

### Быстрая установка

```bash
curl -fsSL https://raw.githubusercontent.com/cy83rt00n/workspace-manager/main/install.sh | bash
source ~/.bashrc
```

Зависимости (`python3`, `sshfs`, `nc`) устанавливаются автоматически.

### Что разворачивается

| Путь | Назначение |
|---|---|
| `~/.local/bin/wsm` | CLI: mount, run, unmount |
| `~/.local/bin/wsm-tui` | Интерактивный curses TUI |
| `~/.workspace-manager` | Установщик, деинсталлер, репо |

Хук PATH добавляется в `~/.bashrc` и `~/.zshrc`.

### Удаление

```bash
curl -fsSL https://raw.githubusercontent.com/cy83rt00n/workspace-manager/main/uninstall.sh | bash
```

### Конфигурация

Конфиги в `~/.config/workspace/<имя>.conf` — формат TOML:

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

### Команды CLI

```bash
wsm mount   <проект>   # Проверить сеть и смонтировать
wsm run     <проект>   # Смонтировать и открыть в IDE
wsm unmount <проект>   # Безопасно размонтировать
wsm --list             # Список проектов
```

Сокращения: `m`, `r`, `u`.

### TUI

```bash
wsm-tui
```

| Клавиша | Действие |
|---|---|
| `Enter` / `r` | Смонтировать + запустить редактор |
| `m` | Только смонтировать |
| `u` | Размонтировать |
| `c` | Создать новый конфиг |
| `d` | Показать Desktop Action |
| `k` | Сгенерировать ключ ED25519 |
| `h` | Справка |
| `j` / `↓` | Вниз |
| `k` / `↑` | Вверх |
| `q` | Выход |

### Desktop Actions

Вставьте фрагмент в `.desktop`-файл редактора в `~/.local/share/applications/`:

```ini
Actions=my-project;

[Desktop Action my-project]
Name=Open Remote: my-project
Exec=bash -c 'source $HOME/.wsm && wsm run my-project'
Identifier=my-project
```

---

## License

MIT — see [LICENSE](./LICENSE)
