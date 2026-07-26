# Changelog — cli branch

## Added

- **`.gitignore`** — Python cache, swap files, IDE dirs, AGENT.md, TASKS.md
- **`CHANGELOG.md`** — Project changelog
- **`demo.config.toml`** — TOML config template for new projects
- **`.wsm`** — Ported `delete` command (aliases: `del`, `rm`): safety check (mounted = refuse), confirmation prompt

## Removed

- **`demo.conf`** — Duplicate config, replaced by `demo.config.toml`
- **`AGENT.md`**, **`TASKS.md`** — Removed from git tracking (now in `.gitignore`)

## Fixed

- **`.wsm`** — Fixed `connect|c)` referencing undefined `REMOTE_PATH`/`EDITOR_CMD`; now uses `remote_path`/`editor_cmd`
- **`uninstall.sh`** — Sed now uses range deletion to prevent orphan if/fi; pipe mode preserves configs

## Changed

- **`README.md`** — Rewritten for cli branch: Bash-only docs, cli-specific install/uninstall URLs
- **`install.sh`** — Added `mkdir -p` before deps check, clone with `--branch cli`
- **`install.sh`**, **`uninstall.sh`** — Hooks wrapped in marker comments (`>>> WSM BEGIN >>>` / `<<< WSM END <<<`)
- **`.wsm`**, **`.wsm-manager`** — Config extension: `.conf` → `.config.toml`
