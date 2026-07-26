# WSM — Workspace Manager

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)
[![Shell](https://img.shields.io/badge/Shell-Bash%20%7C%20Zsh-1f425f.svg)](#)
[![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20macOS-blue.svg)](#)

Монтируйте удалённые проекты через SSHFS и открывайте в IDE одной командой.  
Mount remote projects via SSHFS and open them in your IDE with a single command.

---

- [English](#english)
- [Русский](#русский)

---

## English

### Overview

`wsm` mounts a remote directory over SSHFS, works with it as if local, and launches your IDE — all in one command. No manual `sshfs`, no repetitive mount checks, no forgotten unmounts.

### Features

| Feature                  | Description                                          |
| ------------------------ | ---------------------------------------------------- |
| 🔗 One command            | `wsm run <project>` — mount + open IDE               |
| 🔌 Native SSH Remoting    | `wsm connect <project>` — bypass FUSE, use Zed/VS Code native |
| 🛡️ Safe configs           | TOML format, regex parser, no `source` injection     |
| 🌐 Pre-flight network check | Resolves SSH alias, verifies host:port via `nc`    |
| 🧹 Clean unmount          | Kills file locks via `fuser`, lazy-unmount fallback  |
| ⌨️ Autocomplete            | Bash & Zsh: commands and project names               |
| 🖥️ Interactive manager     | `wsm-manager` — configs, desktop entries, SSH keys   |
| 📋 XDG Desktop Actions     | Right-click project launch from IDE dock menu        |

### Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/cy83rt00n/workspace-manager/main/install.sh | bash
source ~/.bashrc   # or: source ~/.zshrc
```

Dependencies (`sshfs`, `nc`) are installed automatically. See [Manual Install](#manual-install).

### What Gets Deployed

| Path                       | Purpose                                |
| -------------------------- | -------------------------------------- |
| `~/.wsm`                   | Core: `wsm` function, parser, autocomplete |
| `~/.wsm-manager`           | Interactive `wsm-manager` menu         |
| `~/.workspace-manager/`    | Repo copy, installer, uninstaller      |

Shell hooks are injected into `~/.bashrc` and `~/.zshrc`.

### Uninstall

```bash
curl -fsSL https://raw.githubusercontent.com/cy83rt00n/workspace-manager/main/uninstall.sh | bash
```

Prompts to keep or delete configs and repo. Shows removal commands for auto-installed packages.

### Configuration

Project configs live in `~/.config/workspace/<name>.conf` (TOML format).  
A template is provided in the repo: [`demo.config.toml`](./demo.config.toml).

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
wsm help                  # Show help
```

Shortcuts: `m`, `r`, `c`, `u`.

#### `connect` vs `run`

| Command   | Transport      | Use case                              |
| --------- | -------------- | ------------------------------------- |
| `run`     | SSHFS (FUSE)   | Full local filesystem, any editor     |
| `connect` | Native SSH     | Zed remote / VS Code Remote-SSH       |

### Interactive Manager

```bash
wsm-manager
```

1. **Create / Update** a project config
2. **Generate** an XDG Desktop Action snippet
3. **Generate** an ED25519 SSH keypair
4. **Exit**

### Desktop Actions

Paste the generated snippet into your editor's `.desktop` file at `~/.local/share/applications/`:

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

`wsm` монтирует удалённую директорию через SSHFS, работает как с локальной и запускает IDE — всё одной командой. Без ручного `sshfs`, без повторных проверок монтирования, без забытых unmount.

### Возможности

| Возможность                | Описание                                              |
| -------------------------- | ----------------------------------------------------- |
| 🔗 Одна команда             | `wsm run <проект>` — mount + IDE                      |
| 🔌 Нативный SSH Remoting    | `wsm connect <проект>` — без FUSE, Zed/VS Code напрямую |
| 🛡️ Безопасные конфиги       | TOML, regex-парсер, без `source`-инъекций             |
| 🌐 Проверка сети перед mount | Резолв SSH-алиаса, проверка host:port через `nc`     |
| 🧹 Чистый unmount           | Снятие блокировок `fuser`, fallback на lazy-unmount   |
| ⌨️ Автодополнение            | Bash и Zsh: команды и имена проектов                  |
| 🖥️ Интерактивный менеджер    | `wsm-manager` — конфиги, desktop, SSH-ключи           |
| 📋 XDG Desktop Actions       | Запуск проектов из контекстного меню редактора        |

### Быстрая установка

```bash
curl -fsSL https://raw.githubusercontent.com/cy83rt00n/workspace-manager/main/install.sh | bash
source ~/.bashrc   # или: source ~/.zshrc
```

Зависимости (`sshfs`, `nc`) устанавливаются автоматически. См. [Ручная установка](#ручная-установка).

### Что разворачивается

| Путь                       | Назначение                                  |
| -------------------------- | ------------------------------------------- |
| `~/.wsm`                   | Ядро: функция `wsm`, парсер, автодополнение |
| `~/.wsm-manager`           | Интерактивное меню `wsm-manager`            |
| `~/.workspace-manager/`    | Копия репо, установщик, деинсталлер         |

Хуки добавляются в `~/.bashrc` и `~/.zshrc`.

### Удаление

```bash
curl -fsSL https://raw.githubusercontent.com/cy83rt00n/workspace-manager/main/uninstall.sh | bash
```

Запрашивает подтверждение на удаление конфигов и репо. Показывает команды для удаления автовстановленных пакетов.

### Конфигурация

Конфиги проектов хранятся в `~/.config/workspace/<имя>.conf` (формат TOML).  
Шаблон — в репозитории: [`demo.config.toml`](./demo.config.toml).

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
wsm connect  <проект>    # Нативный SSH remoting (без FUSE)
wsm unmount  <проект>    # Безопасно размонтировать
wsm help                 # Справка
```

Сокращения: `m`, `r`, `c`, `u`.

#### `connect` против `run`

| Команда   | Транспорт      | Сценарий                              |
| --------- | -------------- | ------------------------------------- |
| `run`     | SSHFS (FUSE)   | Полная локальная ФС, любой редактор   |
| `connect` | Нативный SSH   | Zed remote / VS Code Remote-SSH       |

### Интерактивный менеджер

```bash
wsm-manager
```

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
