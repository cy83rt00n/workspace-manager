# Changelog — cli branch

## Added

- **`AGENT.md`** — Agent operational rules and coding standards
- **`TASKS.md`** — Central task tracking file
- **`.gitignore`** — Python cache, swap files, IDE dirs, AGENT.md, TASKS.md
- **`CHANGELOG.md`** — Project changelog
- **`demo.config.toml`** — TOML config template for new projects

## Removed

- **`demo.conf`** — Duplicate config, replaced by `demo.config.toml`
- **`AGENT.md`**, **`TASKS.md`** — Removed from git tracking (now in `.gitignore`)

## Fixed

- **`.wsm`** — Fixed `connect|c)` referencing undefined `REMOTE_PATH`/`EDITOR_CMD`; now uses `remote_path`/`editor_cmd`

## Added

- **`.wsm`** — Ported `delete` command (aliases: `del`, `rm`) from python-curses: safety check (mounted = refuse), confirmation prompt
