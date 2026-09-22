# Changelog — python-curses branch

## Added

- **`.wsm-complete`** — Bash & Zsh autocomplete
- **Autocomplete hooks** — Injected during install, removed during uninstall

## Fixed

- **`install.sh`** — Copies `demo.config.toml` → `~/.config/workspace/demo.config.toml` during install
- **`install.sh`** — CLI/TUI choice now works in pipe mode via /dev/tty
- **`install.sh`**, **`uninstall.sh`** — Hooks wrapped in markers, uninstall uses sed range; pipe mode preserves configs
- **`wsm_render.py`** — Alias always included in form result, enabling config rename
- **`wsm_core.py`** — `validate_config` expands `~` to home directory
- **`wsm_tui.py`** — `config_form` loops back to form on validation error instead of closing
- **`wsm_core.py`**, **`wsm_cli.py`** — Project name derives from the full `.config.toml` suffix (`.stem` left a stray `.config`), so `mount`/`run` no longer look for `<name>.config.config.toml` and fail with "not found"

## Changed

- **`README.md`** — Rewritten for python-curses branch
- **`install.sh`** — `mkdir -p` before deps, clone with `--branch python-curses`
- **`wsm_core.py`, `wsm_cli.py`, `wsm_tui.py`, `install.sh`** — Config extension: `.conf` → `.config.toml`
- **`wsm_render.py`** — `render_form` dialog widened to 110 columns, with horizontal scroll and cursor keys (←→/Home/End) so long paths are fully visible and editable
- **`install.sh`** — Default `wsm` interface is now TUI (choice 2)

## Removed

- **`demo.conf`** — Replaced by `demo.config.toml`
- **`AGENT.md`**, **`TASKS.md`** — Untracked (now in `.gitignore`)
