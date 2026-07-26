# Changelog — main branch

## Changed

- **Repository restructured** — Branch-as-a-Product strategy applied
- **`README.md`** — Rewritten as hub: version table, install options, Development section, release process (cli-v*/py-v* tags)
- **`install.sh`** — Universal installer: interactive choice (CLI / Python-curses), delegates to branch installer, works in pipe mode via /dev/tty
- **`uninstall.sh`** — Universal uninstaller: auto-detection, marker-based hook cleanup (`>>> WSM BEGIN >>>` / `<<< WSM END <<<`), pipe mode preserves configs
- **`.github/PULL_REQUEST_TEMPLATE.md`** — PR template with branch routing, direct push to main forbidden

## Removed

- **`.wsm`**, **`.wsm-manager`** — Bash code moved to `cli` branch

## Fixed

- **`install.sh`** — Pipe mode choice now reads from /dev/tty
- **`uninstall.sh`** — Sed now uses range deletion to prevent orphan if/fi
- **`uninstall.sh`** — Detection includes marker-based fallback
- **`uninstall.sh`** — Pipe mode keeps user configs (defaults to "n" instead of "y")
