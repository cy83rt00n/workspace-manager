# WSM — Workspace Manager

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)
[![Shell](https://img.shields.io/badge/Shell-Bash%20%7C%20Zsh-1f425f.svg)](#)
[![Python](https://img.shields.io/badge/Python-3.x-3776AB.svg)](#)
[![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20macOS-blue.svg)](#)

Монтируйте удалённые проекты через SSHFS и открывайте в IDE одной командой. Две версии на выбор.

Mount remote projects via SSHFS and open them in your IDE with a single command. Two versions available.

---

## Версии / Versions

| Версия | Ветка | Установка |
|--------|-------|-----------|
| **CLI (Bash)** — минималистичный, без зависимостей | [`cli`](https://github.com/cy83rt00n/workspace-manager/tree/cli) | `curl .../cli/install.sh \| bash` |
| **Python-curses (TUI)** — полнофункциональный, curses-интерфейс | [`python-curses`](https://github.com/cy83rt00n/workspace-manager/tree/python-curses) | `curl .../python-curses/install.sh \| bash` |
| **Универсальный** — выбор при установке | `main` | `curl .../main/install.sh \| bash` |

### CLI (Bash)

- Pure Bash, работает везде где есть bash/zsh
- `wsm` — команды из терминала
- `wsm-manager` — интерактивное меню
- Автодополнение Bash + Zsh
- **Зависимости:** `sshfs`, `nc`

### Python-curses (TUI)

- Python 3 + curses
- `wsm` — CLI с командами
- `wsm-tui` — двухпанельный интерфейс (Norton Commander стиль)
- Спиннеры, box-drawing, цветовые темы
- **Зависимости:** `python3`, `sshfs`, `nc`

---

## Быстрая установка / Quick Install

```bash
# Универсальный установщик — выберите версию интерактивно
curl -fsSL https://raw.githubusercontent.com/cy83rt00n/workspace-manager/main/install.sh | bash

# Или сразу конкретную версию:
curl -fsSL https://raw.githubusercontent.com/cy83rt00n/workspace-manager/cli/install.sh | bash           # CLI (Bash)
curl -fsSL https://raw.githubusercontent.com/cy83rt00n/workspace-manager/python-curses/install.sh | bash   # Python-curses (TUI)

source ~/.bashrc
```

Зависимости устанавливаются автоматически.

---

## Конфигурация / Configuration

Конфиги проектов в `~/.config/workspace/<name>.conf` (TOML):

```toml
remote_path = "ssh-alias:/remote/project/path"
local_mount = "/home/user/.workspace/my-project"
editor_cmd  = "zed"
```

---

## Команды / Commands

```bash
wsm mount    <project>    # Монтировать / Mount
wsm run      <project>    # Монтировать + IDE / Mount + IDE
wsm connect  <project>    # Нативный SSH / Native SSH
wsm unmount  <project>    # Размонтировать / Unmount
wsm delete   <project>    # Удалить конфиг / Delete config
wsm help                  # Справка / Help
```

---

## Development / Структура репозитория

Этот репозиторий использует стратегию **«Ветка как Продукт» (Branch-as-a-Product)**.  
Весь код разрабатывается в параллельных ветках. Ветка `main` — только хаб и установщик.

| Ветка | Назначение | Код |
|-------|-----------|-----|
| [`main`](https://github.com/cy83rt00n/workspace-manager/tree/main) | Мета-репозиторий: установщик, README | — |
| [`cli`](https://github.com/cy83rt00n/workspace-manager/tree/cli) | Bash-версия | `.wsm`, `.wsm-manager` |
| [`python-curses`](https://github.com/cy83rt00n/workspace-manager/tree/python-curses) | Python-версия | `wsm_*.py` |

### Куда отправлять PR

| Если вы изменяете... | Отправляйте PR в ветку |
|---------------------|----------------------|
| Bash-версию | [`cli`](https://github.com/cy83rt00n/workspace-manager/tree/cli) |
| Python-версию | [`python-curses`](https://github.com/cy83rt00n/workspace-manager/tree/python-curses) |

Прямой пуш в `main` запрещён. Установщик и README обновляются мейнтейнерами.

### Процесс релиза / Release process

Каждая ветка версионируется независимо. Релизы оформляются git-тегами с префиксами:

| Ветка | Префикс тега | Пример |
|-------|-------------|--------|
| `cli` | `cli-v*` | `cli-v2.1.0` |
| `python-curses` | `py-v*` | `py-v2.1.0` |

Теги создаются **только на соответствующей ветке**:

```bash
# Релиз Bash-версии
git checkout cli
git tag cli-v2.1.0
git push origin cli-v2.1.0

# Релиз Python-версии
git checkout python-curses
git tag py-v2.1.0
git push origin py-v2.1.0
```

Ветка `main` не тегируется — она всегда отражает актуальное состояние хаба.

---

## Удаление / Uninstall

```bash
curl -fsSL https://raw.githubusercontent.com/cy83rt00n/workspace-manager/main/uninstall.sh | bash
```

---

## License

MIT — see [LICENSE](./LICENSE)
