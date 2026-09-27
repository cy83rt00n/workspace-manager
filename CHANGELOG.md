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

## go-lang (Go migration — baseline stage, not released)

- **Go module established** — `go.mod`: module `github.com/cy83rt00n/workspace-manager`, go 1.22, zero dependencies
- **`internal/baseline`** — pinned Python CLI contract as self-documenting Go tables: 12 command forms, exit codes {0,1,2}, message templates, baseline defects D1–D17
- **Characterization tests** — 40 table-driven cases spawning the reference `wsm_cli.py` under an isolated `HOME` (`t.TempDir()`), 15s bounded contexts, python3-absence skip
- **Reference snapshot** — `internal/baseline/testdata/reference/{wsm_cli.py,wsm_core.py}` byte-for-byte from `python-curses` @121cdfd
- **`.opencode/agent/prompt-manager.md`** — hardened: `bash` denied, edits limited to `.ai/` (post-incident fix)
- **Go tooling** — `Makefile` (fmt/vet/test/coverage/build/clean, reproducible build via `-trimpath` + `-ldflags -X …/internal/version.Version`), CI matrix (ubuntu/macos + 4 cross-builds, `CGO_ENABLED=0`), `internal/version` package

- **`internal/config`** — ported config domain (parse/validate/save/discovery/delete guards/desktop snippet), 53 table-driven tests, first-match parse semantics corrected after empirical check

- **`internal/system` + `internal/workspace`** — typed process/mount/net adapters and use-case engine (mount/unmount/run/connect/delete), injected callbacks, ExitError codes; ADR-001 fixes D3/D4/D6/D8 applied
