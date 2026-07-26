# Changelog — python-curses branch

## Added

- **`AGENT.md`** — Agent operational rules and coding standards
- **`TASKS.md`** — Central task tracking file
- **`.gitignore`** — Python cache, swap files, IDE dirs, AGENT.md, TASKS.md
- **`CHANGELOG.md`** — Project changelog
- **`demo.config.toml`** — TOML config template for new projects
- **`.wsm-complete`** — Bash & Zsh autocomplete (commands + project names)
- **Autocomplete hooks** — Injected into `.bashrc` / `.zshrc` during install, removed during uninstall

## Removed

- **`demo.conf`** — Duplicate config, replaced by `demo.config.toml`
- **`AGENT.md`**, **`TASKS.md`** — Removed from git tracking (now in `.gitignore`)

## Changed

- **`README.md`** — Rewritten for python-curses branch: Python-only docs, CLI+TUI commands, hotkeys table, python-curses-specific install/uninstall URLs
- **`install.sh`** — Copies `demo.config.toml` → `~/.config/workspace/demo.conf` during install so TUI has a default project immediately
- **`install.sh`** — Added `mkdir -p` before deps check, clone with `--branch python-curses`, updated messages
- **`install.sh`**, **`uninstall.sh`** — Hooks wrapped in marker comments (`>>> WSM BEGIN >>>` / `<<< WSM END <<<`), uninstall uses sed range for atomic removal
