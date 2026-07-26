# Changelog

Commits since `origin/main`:

```
4b8ad13 fix: resolve bugs found during code audit
8685803 chore: add .gitignore, AGENT.md, TASKS.md
```

## Changes (7 files, +171/−80)

### Bugfixes

- **`.wsm`** — Fixed `connect|c)` block referencing undefined `REMOTE_PATH` / `EDITOR_CMD` variables; now uses already-parsed `remote_path` / `editor_cmd`
- **`install.sh`** — Added `mkdir -p $TARGET_REPO_DIR` before deps check to prevent crash when `.wsm-deps` is written before directory creation

### Project structure

- **`demo.conf` → `demo.config.toml`** — Renamed demo config to `.toml` extension, no key duplication
- **`.gitignore`** — Added (Python cache, swap files, IDE dirs, AGENT.md, TASKS.md)
- **`AGENT.md`** — Added (agent operational rules)
- **`TASKS.md`** — Added (central task file)
- **`__pycache__/`** — Removed orphan Python bytecode

### Documentation

- **`README.md`** — Rewritten from scratch: added `connect` command docs, `connect vs run` comparison table, cleaner table formatting, reference to `demo.config.toml` template
