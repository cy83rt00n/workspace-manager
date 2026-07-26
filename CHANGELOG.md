# Changelog

Commits since `origin/main`:

```
46a688b chore: add CHANGELOG.md, update AGENT.md and TASKS.md
4b8ad13 fix: resolve bugs found during code audit
8685803 chore: add .gitignore, AGENT.md, TASKS.md
```

## Changes (8 files, +213/−84)

### Bugfixes

- **`.wsm`** — Fixed `connect|c)` block referencing undefined `REMOTE_PATH` / `EDITOR_CMD` variables; now uses already-parsed `remote_path` / `editor_cmd`
- **`install.sh`** — Added `mkdir -p $TARGET_REPO_DIR` before deps check to prevent crash when `.wsm-deps` is written before directory creation

### Project structure

- **`demo.conf` → `demo.config.toml`** — Renamed demo config to `.toml` extension
- **`.gitignore`** — Added (Python cache, swap files, IDE dirs, AGENT.md, TASKS.md)
- **`AGENT.md`** — Added and updated agent operational rules
- **`TASKS.md`** — Added task tracking file, all 6 tasks completed
- **`CHANGELOG.md`** — Added project changelog
- **`__pycache__/`** — Removed orphan Python bytecode

### Documentation

- **`README.md`** — Rewritten: added `connect` command docs, `connect vs run` table, cleaner formatting, reference to `demo.config.toml`
