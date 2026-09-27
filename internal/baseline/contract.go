// Package baseline is a self-documenting specification of the public contract
// of the WS Manager Python CLI/WIP implementation. It carries no business
// logic: it only fixes the command aliases, the exit-code semantics, the
// config-store schema and the diagnostic message templates that the
// characterization tests of this package spawn against the reference Python
// implementation. It is the single source of truth for those contract points
// during the migration to Go.
//
// Дефекты baseline Python/WIP (зафиксированы как есть, НЕ исправляются):
//
//	D1  .wsm-complete ищет *.conf вместо *.config.toml (bash-файл, spawn-не-наблюдаем).
//	D2  parse/save без экранирования: regex ^<key>\s*=\s*"(.+)" жадный, '"' в значении ломается.
//	D3  mount: печатает FAILED, но exit 0.
//	D4  unmount: lazy-fallback игнорирует ошибку, всегда "detached", exit 0.
//	D5  help → exit 2 (help только в README/completion).
//	D6  generate_keypair не проверяет rc ssh-keygen/mv.
//	D7  desktop_snippet_text: имя проекта в single-quoted shell (инъекция ').
//	D8  check_net: rc ssh -G не проверяется, дубли hostname/port — берётся последний.
//	D9  installer перезаписывает demo.config.toml.
//	D11 --list пусто → сообщение в stderr при exit 0.
//	D12 TUI→CLI subprocess: timeout=60 хардкод, нет cancellation.
//	D13 два пути обнаружения (list_projects в cli vs load_projects в core).
//	D14 match_key (Cyrillic) — только TUI.
//	D15 save не атомарен.
//	D17 -f/--force принимается только ДО команды.
//
// Номера D10 и D16 пропущены намеренно — их нет в утверждённом brief.
package baseline

// CommandAliases lists the 12 accepted forms of the user-facing commands, in
// the canonical order: full word first, then its short alias.
var CommandAliases = []string{
	"mount",
	"m",
	"unmount",
	"u",
	"run",
	"r",
	"connect",
	"c",
	"ssh",
	"delete",
	"del",
	"rm",
}

// ExitCode is the process exit status used by the baseline CLI.
//
// Коды выхода (канон):
//
//	0  help/usage (без команды, -h/--help), --list/-l (в т.ч. пусто), успешные операции.
//	1  project не задан; конфиг не найден; (по коду: net check fail, delete заблокирован, VFS-таймаут).
//	2  argparse: неизвестная команда, help, -f/--force после команды с последующим project.
type ExitCode int

const (
	// ExitOK means help/usage was shown, --list/-l ran (including an empty list)
	// or an operation completed successfully.
	ExitOK ExitCode = 0

	// ExitOperational means an operational error occurred: a missing project
	// name, a missing project config, a network check failure, a blocked
	// delete, or a VFS timeout.
	ExitOperational ExitCode = 1

	// ExitArgparse means the command line could not be parsed: an unknown
	// command, "help" given as a command, or -f/--force placed after the
	// command together with a project.
	ExitArgparse ExitCode = 2
)

// UsageFragment is the substring that must appear in the usage/help output.
const UsageFragment = "usage:"

// VersionBannerFragment is the substring that identifies the baseline Python
// version banner printed by the CLI.
const VersionBannerFragment = "Workspace Manager v2.0.0 (Python)"

// InvalidChoiceFragment is the argparse fragment it emits for an unknown
// command name (it includes the offending token right after it).
const InvalidChoiceFragment = "invalid choice"

// ProjectNameRequiredMessage is the exact stderr line emitted when a command is
// invoked without a project name.
const ProjectNameRequiredMessage = "Error: project name required\n"

// ProjectConfigNotFoundFormat is the stderr format emitted when a project
// config file is missing. The first %s is the project name and the second %s
// is the absolute config directory path.
const ProjectConfigNotFoundFormat = "Error: Project config '%s' not found in %s\n"

// ProjectNotFoundSuffix is the invariant tail of the project-not-found
// diagnostic. It is used to recognise the message regardless of the absolute
// config directory path in the emitted output.
const ProjectNotFoundSuffix = "not found in"

// NoProjectsConfiguredMessage is the exact stderr line emitted by an empty
// --list/-l run (baseline defect D11: the message goes to stderr while the
// process still exits 0).
const NoProjectsConfiguredMessage = "No projects configured.\n"

// DeleteConfigPromptFragment is the confirmation prompt emitted by the CLI
// before deleting a project config.
const DeleteConfigPromptFragment = `Delete config "partial"? [y/N]: `

// DeleteCancelledFragment is the stdout line emitted when a delete is
// cancelled by the user.
const DeleteCancelledFragment = "Cancelled."

// DeleteConfirmedFragment is the stdout line emitted when a delete succeeds.
const DeleteConfirmedFragment = `Config "partial" deleted.`

// UnrecognizedArgumentsFragment is the argparse fragment describing arguments
// that were not accepted (used for the misplaced -f/--force after the command).
const UnrecognizedArgumentsFragment = "unrecognized arguments"

// ConfigDirName is the per-user config directory name under $HOME.
const ConfigDirName = ".config/workspace"

// ConfigFileSuffix is the filename suffix of a single project config file.
const ConfigFileSuffix = ".config.toml"

// ConfigFilePattern describes the per-project filename layout. The config
// store lives in ~/.config/workspace and holds one file per project named
// <name>.config.toml; the project name is the filename without the suffix.
// Each file is parsed line-by-line with the regex ^<key>\s*=\s*"(.+)" which is
// greedy: a '"' inside a value breaks the parse (baseline defect D2). The
// recognised keys are remote_path, local_mount and editor_cmd. A missing key
// yields an empty string and the first occurrence of a key wins (the Python
// loop returns on its first match — verified empirically). A malformed file
// never breaks the whole invocation.
const ConfigFilePattern = "<name>.config.toml"
