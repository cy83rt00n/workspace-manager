# WSM — Workspace Manager

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)
[![Shell](https://img.shields.io/badge/Shell-Bash%20%7C%20Zsh-1f425f.svg)](#)
[![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20macOS-blue.svg)](#)

Shell-инструмент для монтирования удалённых проектов через SSHFS с последующим открытием в IDE одной командой.

A shell tool for mounting remote projects via SSHFS and opening them in your IDE with a single command.

---

- [English](#english)
- [Русский](#русский)

---

## English

### Overview

**WSM** is a lightweight CLI tool that bridges your local environment with remote servers. Mount any remote directory via SSHFS, work with it as if it's local, and launch your IDE — all with one command. No more manual `sshfs` incantations.

### Features

- 🔗 **One-command workflow** — `wsm run <project>` mounts and opens in IDE
- 🛡️ **Safe config parsing** — regex-based parser, no `source`-based injection risks
- 🌐 **Network pre-check** — resolves SSH alias and verifies host reachability before mounting
- 🧹 **Clean unmount** — kills locking file descriptors, falls back to lazy-unmount
- ⌨️ **Shell autocomplete** — Bash & Zsh support for commands and project names
- 🖥️ **Interactive TUI** — `wsm-manager` for config, desktop entries, and SSH key generation
- 📋 **XDG Desktop Actions** — launch remote projects from your editor's right-click menu

### Requirements

| Utility | Purpose |
|---|---|
| `bash` | Shell runtime |
| `ssh` | Remote connection |
| `sshfs` | Filesystem mount |
| `ssh-keygen` | Key pair generation |
| `nc` (netcat) | Host/port availability check |
| `mountpoint` | Mount state detection |
| `fuser` | File descriptor release |
| `umount` | Unmounting |

### Installation

#### Quick install (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/cy83rt00n/workspace-manager/main/install.sh | bash
source ~/.bashrc   # or: source ~/.zshrc
```

#### Manual install

```bash
git clone https://github.com/cy83rt00n/workspace-manager.git
cd workspace-manager
chmod +x install.sh
./install.sh
source ~/.bashrc   # or: source ~/.zshrc
```

### Uninstall
The installer:

- Deploys files to `~/.workspace-manager`
- Copies `.wsm` and `.wsm-manager` to `~/.wsm` and `~/.wsm-manager`
- Injects a source hook into `.bashrc` and `.zshrc`

### Uninstall

```bash
# Quick uninstall (recommended)
curl -fsSL https://raw.githubusercontent.com/cy83rt00n/workspace-manager/main/uninstall.sh | bash

# Or run locally
~/.workspace-manager/uninstall.sh
```

### Configuration

Project configs are stored as `.conf` files in `~/.config/workspace/`.

```toml
# ~/.config/workspace/my-project.conf
remote_path = "ssh-alias:/remote/project/path"
local_mount = "/home/user/.workspace/my-project"
editor_cmd = "zed"
```

| Field | Description |
|---|---|
| `remote_path` | SSH alias and remote path (`alias:/path`) |
| `local_mount` | Local mount point directory |
| `editor_cmd` | Editor/IDE launch command |

### Commands

```bash
wsm mount   <project>   # Check network and mount
wsm run     <project>   # Mount + open in IDE
wsm unmount <project>   # Safely unmount
wsm help                # Show help
```

Short aliases: `m`, `r`, `u`.

### Interactive Manager

```bash
wsm-manager
```

Interactive menu with four options:

1. **Create / Update Project Config** — guided wizard for new `.conf` files
2. **Generate XDG Desktop Action** — copy-paste snippet for your editor's `.desktop` file
3. **Generate ED25519 Keypair** — modern, secure SSH key generation
4. **Exit**

### Desktop Action Integration

Add the generated snippet into your IDE's desktop file at `~/.local/share/applications/`:

```ini
Actions=my-project;

[Desktop Action my-project]
Name=Open Remote: my-project
Exec=bash -c 'source $HOME/.wsm && wsm run my-project'
Identifier=my-project
```

### Autocomplete

Press `Tab` after `wsm` to get command suggestions (`mount`, `run`, `unmount`, `help`) and project names from `~/.config/workspace`. Works in both Bash and Zsh.

---

## Русский

### Обзор

**WSM** — легковесный CLI-инструмент, связывающий локальное окружение с удалёнными серверами. Монтируйте любую удалённую директорию через SSHFS, работайте с ней как с локальной и запускайте IDE — всё одной командой. Больше никаких ручных заклинаний с `sshfs`.

### Возможности

- 🔗 **Одна команда для всего** — `wsm run <проект>` монтирует и открывает в IDE
- 🛡️ **Безопасный парсинг конфигов** — на основе регулярных выражений, без риска инъекций через `source`
- 🌐 **Предварительная проверка сети** — резолвит SSH-алиас и проверяет доступность хоста перед монтированием
- 🧹 **Чистый unmount** — убивает блокирующие файловые дескрипторы, fallback на lazy-unmount
- ⌨️ **Автодополнение** — поддержка Bash и Zsh для команд и названий проектов
- 🖥️ **Интерактивное меню** — `wsm-manager` для конфигов, desktop-записей и генерации SSH-ключей
- 📋 **XDG Desktop Actions** — запуск удалённых проектов из контекстного меню редактора

### Требования

| Утилита | Назначение |
|---|---|
| `bash` | Среда выполнения |
| `ssh` | Удалённое подключение |
| `sshfs` | Монтирование файловой системы |
| `ssh-keygen` | Генерация ключевых пар |
| `nc` (netcat) | Проверка доступности хоста/порта |
| `mountpoint` | Определение состояния монтирования |
| `fuser` | Освобождение файловых дескрипторов |
| `umount` | Размонтирование |

### Установка

#### Быстрая установка (рекомендуется)

```bash
curl -fsSL https://raw.githubusercontent.com/cy83rt00n/workspace-manager/main/install.sh | bash
source ~/.bashrc   # или: source ~/.zshrc
```

#### Ручная установка

```bash
git clone https://github.com/cy83rt00n/workspace-manager.git
cd workspace-manager
chmod +x install.sh
./install.sh
source ~/.bashrc   # или: source ~/.zshrc
```

### Удаление
Установщик:

- Разворачивает файлы в `~/.workspace-manager`
- Копирует `.wsm` и `.wsm-manager` в `~/.wsm` и `~/.wsm-manager`
- Добавляет хук-инициализацию в `.bashrc` и `.zshrc`

### Удаление

```bash
# Быстрое удаление (рекомендуется)
curl -fsSL https://raw.githubusercontent.com/cy83rt00n/workspace-manager/main/uninstall.sh | bash

# Или запустить локально
~/.workspace-manager/uninstall.sh
```

### Конфигурация

Конфиги проектов хранятся в виде `.conf`-файлов в `~/.config/workspace/`.

```toml
# ~/.config/workspace/my-project.conf
remote_path = "ssh-alias:/remote/project/path"
local_mount = "/home/user/.workspace/my-project"
editor_cmd = "zed"
```

| Поле | Описание |
|---|---|
| `remote_path` | SSH-алиас и удалённый путь (`алиас:/путь`) |
| `local_mount` | Локальная точка монтирования |
| `editor_cmd` | Команда запуска редактора/IDE |

### Команды

```bash
wsm mount   <проект>   # Проверить сеть и смонтировать
wsm run     <проект>   # Смонтировать и открыть в IDE
wsm unmount <проект>   # Безопасно размонтировать
wsm help               # Показать справку
```

Краткие алиасы: `m`, `r`, `u`.

### Интерактивный менеджер

```bash
wsm-manager
```

Интерактивное меню с четырьмя пунктами:

1. **Создать / Обновить конфиг проекта** — пошаговый мастер для новых `.conf`-файлов
2. **Сгенерировать XDG Desktop Action** — фрагмент для вставки в `.desktop`-файл редактора
3. **Сгенерировать ключевую пару ED25519** — современная генерация SSH-ключей
4. **Выход**

### Интеграция с Desktop Action

Добавьте сгенерированный фрагмент в desktop-файл вашего IDE в `~/.local/share/applications/`:

```ini
Actions=my-project;

[Desktop Action my-project]
Name=Open Remote: my-project
Exec=bash -c 'source $HOME/.wsm && wsm run my-project'
Identifier=my-project
```

### Автодополнение

Нажмите `Tab` после `wsm` для подсказок команд (`mount`, `run`, `unmount`, `help`) и списка проектов из `~/.config/workspace`. Работает в Bash и Zsh.

---

## License

MIT — see [LICENSE](./LICENSE)
