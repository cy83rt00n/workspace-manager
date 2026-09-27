// Command wsm is the temporary Go CLI for the workspace-manager migration
// (AGENTS Stage 5). It parses the user's command line, wires the workspace
// Engine over the injected System factory and dispatches the selected
// use-case. It is a byte-for-byte port of the Python wsm_cli.py semantics with
// the ADR-001 divergences (help exits 0, -f/-l/-h accepted in any position,
// the "(Go)" banner and the temporary prog name wsm-go). The package holds no
// stateful globals and performs no subprocess orchestration: every side effect
// lives behind workspace.Engine.
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/cy83rt00n/workspace-manager/internal/config"
	"github.com/cy83rt00n/workspace-manager/internal/system"
	"github.com/cy83rt00n/workspace-manager/internal/version"
	"github.com/cy83rt00n/workspace-manager/internal/workspace"
)

// usageTemplate is the complete help output. The single %s placeholder is the
// version string; the prog name (wsm-go) is the temporary binary name used
// during the migration (AGENTS Stage 5).
const usageTemplate = `Workspace Manager v%s (Go)

Usage: wsm-go [OPTIONS] <command> [project]

Commands:
  mount, m          Mount remote path via SSHFS
  unmount, u        Safely unmount a project
  run, r            Mount (if needed) and launch editor
  connect, c, ssh   Launch editor native SSH remoting
  delete, del, rm   Delete project config
  help              Show this help

Options:
  -h, --help        Show this help and exit
  -l, --list        List available projects
  -f, --force       Skip confirmation (for delete)
`

// CLI-level diagnostic templates. They are the only messages this package owns;
// every other user-visible string comes from the workspace message catalog.
// The "exit 2" messages mirror the spirit of argparse, while
// errProjectNameRequired already carries the "Error: " prefix (it is sent
// verbatim to stderr by the caller).
const (
	errUnknownOption       = "unknown option: %s"
	errUnknownCommand      = "unknown command: %s"
	errUnexpectedArgument  = "unexpected argument: %s"
	errProjectNameRequired = "Error: project name required"
	errNoProjects          = "No projects configured."
)

// main is the process entry point. It delegates all parsing and dispatch to
// runCLI and translates its returned exit code into os.Exit, keeping defer
// handling trivial and leaving runCLI fully testable.
func main() {
	os.Exit(runCLI(
		os.Args[1:],
		os.Stdin, os.Stdout, os.Stderr,
		config.DefaultDir(),
		version.Version,
		func(ctx context.Context) workspace.System { return system.Default() },
	))
}

// runCLI is the testable entry point: it parses args, wires up an Engine over
// the injected System factory and dispatches the command, RETURNING the process
// exit code (it never calls os.Exit). main() wraps it and calls os.Exit so that
// defer/cleanup stays simple and tests can drive runCLI directly.
//
// Parsing is a single hand-written pass over args (no flag package): ADR-001 D17
// requires -f/-l/-h to be accepted in ANY position, which the stdlib flag
// package cannot express. The processing priority is: --help, then --list, then
// an empty command (usage), then the command word (help / a known alias), then
// the positional-arity check, then the project name.
func runCLI(
	args []string,
	stdin io.Reader,
	stdout, stderr io.Writer,
	configDir string,
	version string,
	newSystem func(context.Context) workspace.System,
) int {
	var (
		listFlag    bool
		helpFlag    bool
		forceFlag   bool
		positionals []string
	)

	for _, a := range args {
		switch a {
		case "-h", "--help":
			helpFlag = true
		case "-l", "--list":
			listFlag = true
		case "-f", "--force":
			forceFlag = true
		default:
			if strings.HasPrefix(a, "-") && a != "-" {
				fmt.Fprintf(stderr, "%s\n", fmt.Sprintf(errUnknownOption, a))
				return 2
			}
			positionals = append(positionals, a)
		}
	}

	if helpFlag {
		printUsage(stdout, version)
		return 0
	}
	if listFlag {
		return listProjects(configDir, stdout, stderr)
	}
	if len(positionals) == 0 {
		printUsage(stdout, version)
		return 0
	}

	command := positionals[0]
	if command == "help" {
		printUsage(stdout, version)
		return 0
	}
	if !isKnownCommand(command) {
		fmt.Fprintf(stderr, "%s\n", fmt.Sprintf(errUnknownCommand, command))
		return 2
	}
	if len(positionals) > 2 {
		fmt.Fprintf(stderr, "%s\n", fmt.Sprintf(errUnexpectedArgument, positionals[2]))
		return 2
	}

	project := ""
	if len(positionals) == 2 {
		project = positionals[1]
	}
	if project == "" {
		fmt.Fprintln(stderr, errProjectNameRequired)
		return 1
	}

	proj := config.FindProject(configDir, project)
	if proj == nil {
		fmt.Fprintf(stderr, "%s\n", fmt.Sprintf(workspace.ErrorLine,
			fmt.Sprintf(workspace.ProjectNotFound, project, configDir)))
		return 1
	}

	timeout := 300 * time.Second // mount, run
	switch command {
	case "connect", "c", "ssh":
		timeout = 15 * time.Second
	case "unmount", "u", "delete", "del", "rm":
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	spinnerRunes := []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")
	spinIndex := 0

	isTTY := false
	if f, ok := stdout.(*os.File); ok {
		if fi, statErr := f.Stat(); statErr == nil {
			isTTY = fi.Mode()&os.ModeCharDevice != 0
		}
	}

	// spin prints a spinner stage; the non-TTY branch is deterministic and is the
	// one exercised by every test (stdout is a *bytes.Buffer there).
	spin := func(stage string) {
		if isTTY {
			fmt.Fprintf(stdout, "\r%s %c", stage, spinnerRunes[spinIndex%len(spinnerRunes)])
			spinIndex++
			// *os.File writes directly (no buffer), so no explicit flush.
		} else {
			fmt.Fprintf(stdout, "%s...\n", stage)
		}
	}

	// spinDone clears the spinner line and prints a completion message. In the
	// TTY branch the line is cleared with a fixed 80-column pad (see
	// terminalWidth); the non-TTY branch prints the bare message.
	spinDone := func(msg string) {
		if isTTY {
			width := 80
			if f, ok := stdout.(*os.File); ok {
				width = terminalWidth(f)
			}
			fmt.Fprintf(stdout, "\r%s\r%s\n", strings.Repeat(" ", width), msg)
		} else {
			fmt.Fprintf(stdout, "%s\n", msg)
		}
	}

	stdinReader := bufio.NewReader(stdin)
	confirm := func(name string) (bool, error) {
		// The prompt is written to stdout without a trailing newline, mirroring
		// Python input(); the follow-up result line is then printed to stdout on
		// its own line.
		fmt.Fprintf(stdout, "Delete config \"%s\"? [y/N]: ", name)
		line, err := stdinReader.ReadString('\n')
		if err != nil && err != io.EOF {
			return false, err
		}
		ans := strings.TrimSpace(strings.ToLower(line))
		return ans == "y" || ans == "yes", nil
	}

	eng := &workspace.Engine{
		ConfigDir: configDir,
		Out: func(format string, args ...any) {
			fmt.Fprintf(stdout, format+"\n", args...)
		},
		ErrOut: func(format string, args ...any) {
			fmt.Fprintf(stderr, format+"\n", args...)
		},
		Spin:     spin,
		SpinDone: spinDone,
		Confirm:  confirm,
		System:   newSystem(ctx),
	}

	var err error
	switch command {
	case "mount", "m":
		err = eng.MountProject(ctx, project)
	case "unmount", "u":
		err = eng.UnmountProject(ctx, project)
	case "run", "r":
		err = eng.RunProject(ctx, project)
	case "connect", "c", "ssh":
		err = eng.ConnectProject(ctx, project, proj.EditorCmd)
	case "delete", "del", "rm":
		err = eng.DeleteProject(ctx, project, forceFlag)
	}
	return reportError(stderr, err)
}

// printUsage writes the full usage text to stdout. It is the target of
// -h/--help, of the "help" command and of an empty command line, all of which
// exit 0 (ADR-001 D5 fixes the argparse exit-2 for help).
func printUsage(stdout io.Writer, version string) {
	fmt.Fprintf(stdout, usageTemplate, version)
}

// isKnownCommand reports whether cmd is one of the 12 accepted command aliases.
// "help" is intentionally NOT included: it is handled before this check (the
// help alias also exits 0).
func isKnownCommand(cmd string) bool {
	switch cmd {
	case "mount", "m",
		"unmount", "u",
		"run", "r",
		"connect", "c", "ssh",
		"delete", "del", "rm":
		return true
	}
	return false
}

// terminalWidth returns the column count used to clear the spinner line. Per
// the primary override (BRIEF-005) the width is NOT read through
// ioctl/TIOCGWINSZ: it is hardcoded to the zero-dependency fallback constant 80
// regardless of f, matching the Python "except OSError: 80" fallback.
func terminalWidth(f *os.File) int {
	return 80
}

// listProjects prints the sorted project names to stdout, or "No projects
// configured." to stderr when the config dir has no projects. It returns 0 in
// both cases, mirroring the --list branch of the reference CLI (an empty list is
// still a successful exit).
func listProjects(configDir string, stdout, stderr io.Writer) int {
	projects, _ := config.Projects(configDir) // the package always returns a nil error
	if len(projects) == 0 {
		fmt.Fprintln(stderr, errNoProjects)
		return 0
	}
	for _, p := range projects {
		fmt.Fprintln(stdout, p.Name) // already sorted inside config.Projects
	}
	return 0
}

// reportError maps an engine error onto the process result: a *workspace.ExitError
// prints "Error: <Msg>" to stderr ONLY when Msg != "" and returns its Code; a
// generic error prints "Error: <err>" and returns 1; nil returns 0. The empty-Msg
// case is important: some failures (e.g. a mount network check) already print
// "FAILED: ..." through the spinner and must not emit a second "Error:" line.
func reportError(stderr io.Writer, err error) int {
	if err == nil {
		return 0
	}
	var ee *workspace.ExitError
	if errors.As(err, &ee) {
		if ee.Msg != "" {
			fmt.Fprintf(stderr, "%s\n", fmt.Sprintf(workspace.ErrorLine, ee.Msg))
		}
		return ee.Code
	}
	fmt.Fprintf(stderr, "%s\n", fmt.Sprintf(workspace.ErrorLine, err.Error()))
	return 1
}
