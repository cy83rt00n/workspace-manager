# WSM CLI — Bash Version

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)
[![Shell](https://img.shields.io/badge/Shell-Bash%20%7C%20Zsh-1f425f.svg)](#)
[![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20macOS-blue.svg)](#)

Минималистичный CLI для монтирования удалённых проектов через SSHFS. Только Bash, без внешних зависимостей.

A minimal CLI for mounting remote projects via SSHFS. Pure Bash, zero external dependencies.

---

- [English](#english)
- [Русский](#русский)

---

## English

### Overview

`wsm` is a pure Bash function that mounts remote directories over SSHFS and opens them in your IDE — all in one command.

No Python, no pip, no npm. Just `curl | bash` and you're ready.

### Features

| Feature              | Description                                          |
| -------------------- | ---------------------------------------------------- |
| 🔗 One command        | `wsm run <project>` — mount + open IDE               |
| 🔌 Native SSH         | `wsm connect <project>` — Zed / VS Code native SSH   |
| 🛡️ Safe configs       | TOML format, regex parser, no injection              |
| 🌐 Network check      | Resolves SSH alias, verifies host:port via `nc`      |
| 🧹 Clean unmount      | `fuser` lock cleanup, lazy-unmount fallback          |
| ⌨️ Autocomplete        | Bash & Zsh: commands + project names                 |
| 🖥️ Interactive manager | `wsm-manager` — configs, desktop entries, SSH keys   |
| 🗑️ Delete configs      | `wsm delete <project>` with safety checks            |
| 📋 XDG Desktop Actions | Right-click project launch from IDE dock             |

### Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/cy83rt00n/workspace-manager/cli/install.sh | bash
source ~/.bashrc   # or: source ~/.zshrc
```

Dependencies (`sshfs`, `nc`) are installed automatically.

### What Gets Deployed

| Path                    | Purpose                                |
| ----------------------- | -------------------------------------- |
| `~/.wsm`                | Core: `wsm` function, parser, autocomplete |
| `~/.wsm-manager`        | Interactive `wsm-manager` menu         |
| `~/.workspace-manager/` | Repo copy, installer, uninstaller      |

Shell hooks are injected into `~/.bashrc` and `~/.zshrc`.

### Uninstall

```bash
curl -fsSL https://raw.githubusercontent.com/cy83rt00n/workspace-manager/cli/uninstall.sh | bash
```

### Configuration

Project configs live in `~/.config/workspace/<name>.conf` (TOML format).  
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

### Commands

```bash
wsm mount    <project>    # Check network, then SSHFS-mount
wsm run      <project>    # Mount + open in IDE
wsm connect  <project>    # Native SSH remoting (no FUSE)
wsm unmount  <project>    # Safely unmount
wsm delete   <project>    # Delete project config
wsm help                  # Show help
```

Shortcuts: `m`, `r`, `c`, `u`, `del`, `rm`.

### Interactive Manager

```bash
wsm-manager
```

1. **Create / Update** a project config
2. **Generate** an XDG Desktop Action snippet
3. **Generate** an ED25519 SSH keypair
4. **Exit**

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
git clone --branch cli https://github.com/cy83rt00n/workspace-manager.git
cd workspace-manager
chmod +x install.sh
./install.sh
source ~/.bashrc
```

---

## Русский

### Обзор

`wsm` — функция на чистом Bash для монтирования удалённых директорий через SSHFS и запуска IDE одной командой.

Никакого Python, pip или npm. Только `curl | bash` — и готово.

### Возможности

| Возможность            | Описание                                              |
| ---------------------- | ----------------------------------------------------- |
| 🔗 Одна команда         | `wsm run <проект>` — mount + IDE                      |
| 🔌 Нативный SSH         | `wsm connect <проект>` — Zed / VS Code без FUSE       |
| 🛡️ Безопасные конфиги   | TOML, regex-парсер, без инъекций                      |
| 🌐 Проверка сети        | Резолв SSH-алиаса, проверка host:port через `nc`      |
| 🧹 Чистый unmount       | `fuser`, fallback на lazy-unmount                     |
| ⌨️ Автодополнение        | Bash и Zsh: команды + имена проектов                  |
| 🖥️ Интерактивный менеджер| `wsm-manager` — конфиги, desktop, SSH-ключи           |
| 🗑️ Удаление конфигов     | `wsm delete <проект>` с проверками безопасности        |
| 📋 XDG Desktop Actions   | Запуск проектов из контекстного меню редактора         |

### Быстрая установка

```bash
curl -fsSL https://raw.githubusercontent.com/cy83rt00n/workspace-manager/cli/install.sh | bash
source ~/.bashrc   # или: source ~/.zshrc
```

Зависимости (`sshfs`, `nc`) устанавливаются автоматически.

### Что разворачивается

| Путь                    | Назначение                                  |
| ----------------------- | ------------------------------------------- |
| `~/.wsm`                | Ядро: функция `wsm`, парсер, автодополнение |
| `~/.wsm-manager`        | Интерактивное меню `wsm-manager`            |
| `~/.workspace-manager/` | Копия репо, установщик, деинсталлер         |

Хуки добавляются в `~/.bashrc` и `~/.zshrc`.

### Удаление

```bash
curl -fsSL https://raw.githubusercontent.com/cy83rt00n/workspace-manager/cli/uninstall.sh | bash
```

### Конфигурация

Конфиги проектов в `~/.config/workspace/<имя>.conf` (формат TOML).  
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

### Команды

```bash
wsm mount    <проект>    # Проверить сеть и смонтировать
wsm run      <проект>    # Смонтировать + открыть в IDE
wsm connect  <проект>    # Нативный SSH (без FUSE)
wsm unmount  <проект>    # Безопасно размонтировать
wsm delete   <проект>    # Удалить конфиг проекта
wsm help                 # Справка
```

Сокращения: `m`, `r`, `c`, `u`, `del`, `rm`.

### Интерактивный менеджер

```bash
wsm-manager
```

1. **Создать / Обновить** конфиг проекта
2. **Сгенерировать** фрагмент XDG Desktop Action
3. **Сгенерировать** ключевую пару ED25519
4. **Выход**

### Desktop Actions

Вставьте сгенерированный фрагмент в `~/.local/share/applications/<editor>.desktop`:

```ini
Actions=my-project;

[Desktop Action my-project]
Name=Open Remote: my-project
Exec=bash -c 'source $HOME/.wsm && wsm run my-project'
Identifier=my-project
```

### Ручная установка

```bash
git clone --branch cli https://github.com/cy83rt00n/workspace-manager.git
cd workspace-manager
chmod +x install.sh
./install.sh
source ~/.bashrc
```

---

## License

MIT — see [LICENSE](./LICENSE)
