# WSM Python-curses — Python Version

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)
[![Python](https://img.shields.io/badge/Python-3.x-3776AB.svg)](#)
[![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20macOS-blue.svg)](#)

Python-инструмент для монтирования удалённых проектов через SSHFS. CLI + curses-TUI с навигацией в стиле Norton Commander.

A Python tool for mounting remote projects via SSHFS. CLI + curses TUI with Norton Commander style navigation.

---

- [English](#english)
- [Русский](#русский)

---

## English

### Overview

`wsm` mounts remote directories over SSHFS and opens them in your IDE. Two interfaces: fast CLI and interactive TUI.

### Features

| Feature             | Description                                           |
| ------------------- | ----------------------------------------------------- |
| 🔗 One command       | `wsm run <project>` — mount + open IDE                |
| 🖥️ Curses TUI        | `wsm-tui` — project list, mount status, hotkeys       |
| 🖥️ CLI               | `wsm` — all commands from terminal                    |
| 🔌 Native SSH        | `wsm connect <project>` — Zed / VS Code without FUSE  |
| 🛡️ Safe configs      | TOML format, Python regex parser                      |
| 🌐 Network check     | Resolves SSH alias, verifies host:port via `nc`       |
| 🧹 Clean unmount     | `fuser` lock cleanup, lazy-unmount fallback           |
| 🗑️ Delete configs     | `wsm delete <project>` with safety checks             |
| ⌨️ Autocomplete       | Bash & Zsh: commands + project names                  |
| 🔑 Built-in tools    | Config wizard, ED25519 key generator, Desktop Actions |
| 📋 XDG Desktop       | Launch projects from IDE right-click menu             |

### Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/cy83rt00n/workspace-manager/python-curses/install.sh | bash
source ~/.bashrc
```

Dependencies (`python3`, `sshfs`, `nc`) are auto-installed. During install you choose the default interface: CLI or TUI.

### What Gets Deployed

| Path                       | Purpose                                   |
| -------------------------- | ----------------------------------------- |
| `~/.local/bin/wsm`         | CLI or TUI (chosen at install)            |
| `~/.local/bin/wsm-tui`     | Interactive curses TUI (always available) |
| `~/.wsm-complete`          | Bash & Zsh autocomplete                   |
| `~/.workspace-manager/`    | Repo copy, installer, uninstaller         |

A PATH hook is added to `~/.bashrc` and `~/.zshrc`.

### Uninstall

```bash
curl -fsSL https://raw.githubusercontent.com/cy83rt00n/workspace-manager/python-curses/uninstall.sh | bash
```

### Configuration

Project configs in `~/.config/workspace/<name>.conf` — TOML format.  
Template: [`demo.config.toml`](./demo.config.toml).

```toml
remote_path = "ssh-alias:/remote/project/path"
local_mount = "/home/user/.workspace/my-project"
editor_cmd  = "zed"
```

| Field          | Description                             |
| -------------- | --------------------------------------- |
| `remote_path`  | SSH alias + remote path (`alias:/path`) |
| `local_mount`  | Local mount point                       |
| `editor_cmd`   | IDE launch command (`zed`, `code`, ...) |

### CLI Commands

```bash
wsm mount    <project>    # Check network, then SSHFS-mount
wsm run      <project>    # Mount + open in IDE
wsm connect  <project>    # Native SSH remoting (no FUSE)
wsm unmount  <project>    # Safely unmount
wsm delete   <project>    # Delete project config
wsm --list                # List configured projects
```

Shortcuts: `m`, `r`, `c`, `u`, `del`, `rm`.

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

| Key        | Action                 |
| ---------- | ---------------------- |
| `Enter`/`r`| Mount + launch editor  |
| `m`        | Mount only             |
| `u`        | Unmount                |
| `s`        | Native SSH connect     |
| `c`        | Create new config      |
| `e`        | Edit config            |
| `x`        | Delete config          |
| `d`        | Desktop Action snippet |
| `k`        | Generate ED25519 key   |
| `h`/`F1`   | Help                   |
| `↑`/`↓`/`jk`| Navigate              |
| `Tab`/`←→` | Switch panels          |
| `q`/`F10`  | Quit                   |

### Desktop Actions

Paste the generated snippet into `~/.local/share/applications/<editor>.desktop`:

```ini
Actions=my-project;

[Desktop Action my-project]
Name=Open Remote: my-project
Exec=bash -c 'source $HOME/.wsm && wsm run my-project'
Identifier=my-project
```

### Manual Install

```bash
git clone --branch python-curses https://github.com/cy83rt00n/workspace-manager.git
cd workspace-manager
chmod +x install.sh
./install.sh
source ~/.bashrc
```

---

## Русский

### Обзор

`wsm` монтирует удалённые директории через SSHFS и запускает IDE. Два интерфейса: быстрый CLI и интерактивный TUI.

### Возможности

| Возможность          | Описание                                              |
| -------------------- | ----------------------------------------------------- |
| 🔗 Одна команда       | `wsm run <проект>` — mount + IDE                      |
| 🖥️ Curses TUI         | `wsm-tui` — список проектов, статус, горячие клавиши  |
| 🖥️ CLI                | `wsm` — все команды из терминала                      |
| 🔌 Нативный SSH       | `wsm connect <проект>` — Zed / VS Code без FUSE       |
| 🛡️ Безопасные конфиги | TOML, Python regex-парсер                             |
| 🌐 Проверка сети      | Резолв SSH-алиаса, проверка host:port через `nc`      |
| 🧹 Чистый unmount     | `fuser`, fallback на lazy-unmount                     |
| 🗑️ Удаление конфигов   | `wsm delete <проект>` с проверками безопасности        |
| ⌨️ Автодополнение      | Bash и Zsh: команды + имена проектов                  |
| 🔑 Инструменты        | Мастер конфигов, генератор ED25519, Desktop Actions   |
| 📋 XDG Desktop        | Запуск проектов из контекстного меню редактора         |

### Быстрая установка

```bash
curl -fsSL https://raw.githubusercontent.com/cy83rt00n/workspace-manager/python-curses/install.sh | bash
source ~/.bashrc
```

Зависимости (`python3`, `sshfs`, `nc`) устанавливаются автоматически. При установке выбирается интерфейс по умолчанию: CLI или TUI.

### Что разворачивается

| Путь                       | Назначение                                     |
| -------------------------- | ---------------------------------------------- |
| `~/.local/bin/wsm`         | CLI или TUI (выбирается при установке)         |
| `~/.local/bin/wsm-tui`     | Интерактивный curses TUI (всегда доступен)     |
| `~/.wsm-complete`          | Автодополнение Bash и Zsh                      |
| `~/.workspace-manager/`    | Копия репо, установщик, деинсталлер            |

Хук PATH добавляется в `~/.bashrc` и `~/.zshrc`.

### Удаление

```bash
curl -fsSL https://raw.githubusercontent.com/cy83rt00n/workspace-manager/python-curses/uninstall.sh | bash
```

### Конфигурация

Конфиги в `~/.config/workspace/<имя>.conf` — формат TOML.  
Шаблон: [`demo.config.toml`](./demo.config.toml).

```toml
remote_path = "ssh-alias:/remote/project/path"
local_mount = "/home/user/.workspace/my-project"
editor_cmd  = "zed"
```

| Поле           | Описание                                   |
| -------------- | ------------------------------------------ |
| `remote_path`  | SSH-алиас + удалённый путь (`алиас:/путь`) |
| `local_mount`  | Локальная точка монтирования               |
| `editor_cmd`   | Команда запуска IDE (`zed`, `code`, ...)   |

### Команды CLI

```bash
wsm mount    <проект>    # Проверить сеть и смонтировать
wsm run      <проект>    # Смонтировать + открыть в IDE
wsm connect  <проект>    # Нативный SSH (без FUSE)
wsm unmount  <проект>    # Безопасно размонтировать
wsm delete   <проект>    # Удалить конфиг проекта
wsm --list               # Список проектов
```

Сокращения: `m`, `r`, `c`, `u`, `del`, `rm`.

### TUI

```bash
wsm-tui
```

| Клавиша     | Действие                   |
| ----------- | -------------------------- |
| `Enter`/`r` | Смонтировать + редактор    |
| `m`         | Только смонтировать        |
| `u`         | Размонтировать             |
| `s`         | Нативный SSH connect       |
| `c`         | Создать новый конфиг       |
| `e`         | Редактировать конфиг       |
| `x`         | Удалить конфиг             |
| `d`         | Desktop Action             |
| `k`         | Сгенерировать ключ ED25519 |
| `h`/`F1`    | Справка                    |
| `↑`/`↓`/`jk`| Навигация                  |
| `Tab`/`←→`  | Переключение панелей       |
| `q`/`F10`   | Выход                      |

### Desktop Actions

Вставьте фрагмент в `~/.local/share/applications/<editor>.desktop`:

```ini
Actions=my-project;

[Desktop Action my-project]
Name=Open Remote: my-project
Exec=bash -c 'source $HOME/.wsm && wsm run my-project'
Identifier=my-project
```

### Ручная установка

```bash
git clone --branch python-curses https://github.com/cy83rt00n/workspace-manager.git
cd workspace-manager
chmod +x install.sh
./install.sh
source ~/.bashrc
```

---

## License

MIT — see [LICENSE](./LICENSE)
