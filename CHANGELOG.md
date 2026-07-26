# Changelog — main branch

## Changed

- **Repository restructured** — Branch-as-a-Product strategy applied
- **`README.md`** — Rewritten as hub: version table, install options, Development section with PR guidelines
- **`install.sh`** — New universal installer: interactive version choice (CLI / Python-curses), delegates to branch-specific installer
- **`uninstall.sh`** — New universal uninstaller: auto-detects installed version, cleans up all components
- **`.github/PULL_REQUEST_TEMPLATE.md`** — PR template with branch routing instructions

## Removed

- **`.wsm`**, **`.wsm-manager`** — Bash code moved to `cli` branch
- **`uninstall.sh`** — Hooks now cleaned via single sed range (`>>> WSM BEGIN >>>` / `<<< WSM END <<<`) instead of fragile per-line regex
