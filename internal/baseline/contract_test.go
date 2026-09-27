package baseline

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Fixture contents are written verbatim into the temporary HOME before each
// fixture-dependent spawn. They are the "as is" data of the golden matrix.
const (
	// fixtureDemoContent is a fully populated project config.
	fixtureDemoContent = `remote_path = "demo-alias:/opt/app"
local_mount = "/home/u/.workspace/app"
editor_cmd = "zed"`

	// fixturePartialContent is a config that carries only the remote_path key
	// (no local_mount), so a delete on it never touches a mount and is safe.
	fixturePartialContent = `remote_path = "alias:/p"`

	// fixtureQuotedContent documents baseline defect D2: a '"' inside the
	// value breaks the greedy line-by-line parse.
	fixtureQuotedContent = `remote_path = "alias:/pa"th"`
)

// fixtureEmptyContent is an empty config file (0 bytes).
const fixtureEmptyContent = ""

var (
	python3Path     string
	python3LookupOK bool
)

// TestMain resolves the python3 interpreter once and shares the result with
// every test. When python3 is absent each test skips instead of failing.
func TestMain(m *testing.M) {
	path, err := exec.LookPath("python3")
	if err == nil {
		python3Path = path
		python3LookupOK = true
	}
	os.Exit(m.Run())
}

// referenceCLIPath returns the absolute path to the reference wsm_cli.py that
// the tests spawn. wsm_core.py lives next to it and is imported via sys.path.
func referenceCLIPath() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	return filepath.Join(filepath.Dir(thisFile), "testdata", "reference", "wsm_cli.py")
}

// withHomeEnv derives the child environment from os.Environ() replacing the
// existing HOME value with the given absolute temporary home directory.
func withHomeEnv(home string) []string {
	env := make([]string, 0, len(os.Environ())+1)
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "HOME=") {
			continue
		}
		env = append(env, kv)
	}
	return append(env, "HOME="+home)
}

// writeFixture creates <home>/.config/workspace/<name>.config.toml with the
// given literal content, creating the config directory if needed.
func writeFixture(t *testing.T, home, content, name string) {
	t.Helper()
	dir := filepath.Join(home, ConfigDirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("failed to create config dir %s: %v", dir, err)
	}
	path := filepath.Join(dir, name+ConfigFileSuffix)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write fixture %s: %v", path, err)
	}
}

// cliResult is the captured outcome of a single reference-CLI spawn.
type cliResult struct {
	stdout   string
	stderr   string
	exitCode int
}

// runReferenceCLI spawns python3 wsm_cli.py with the given arguments under an
// isolated absolute HOME, capturing stdout and stderr separately. Each spawn is
// bounded by a 15-second context timeout. A non-empty stdin argument is only
// wired for delete-confirmation cases; otherwise stdin stays closed.
func runReferenceCLI(t *testing.T, home string, args []string, stdin string) cliResult {
	t.Helper()
	if !python3LookupOK {
		t.Skip("python3 not found in PATH")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmdArgs := append([]string{referenceCLIPath()}, args...)
	cmd := exec.CommandContext(ctx, python3Path, cmdArgs...)
	cmd.Env = withHomeEnv(home)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	err := cmd.Run()
	code := int(ExitOK)
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("failed to spawn python3: %v", err)
		}
		code = exitErr.ExitCode()
	}

	return cliResult{
		stdout:   stdoutBuf.String(),
		stderr:   stderrBuf.String(),
		exitCode: code,
	}
}

// TestUsageAndInvalidCommandTable covers the golden cases of section 4.1:
// command-less usage, help flags, the "help" command, an unknown command, and
// a bare -f/--force flag.
func TestUsageAndInvalidCommandTable(t *testing.T) {
	home := t.TempDir()

	cases := []struct {
		name             string
		args             []string
		wantExit         ExitCode
		wantStdoutDegree []string
		wantStderrDegree []string
		wantEmptyStderr  bool
	}{
		{name: "no args", args: nil, wantExit: ExitOK, wantStdoutDegree: []string{UsageFragment, VersionBannerFragment}, wantEmptyStderr: true},
		{name: "help short flag", args: []string{"-h"}, wantExit: ExitOK, wantStdoutDegree: []string{UsageFragment}},
		{name: "help long flag", args: []string{"--help"}, wantExit: ExitOK, wantStdoutDegree: []string{UsageFragment}},
		{name: "help as command", args: []string{"help"}, wantExit: ExitArgparse, wantStderrDegree: []string{InvalidChoiceFragment, "'help'"}},
		{name: "unknown command", args: []string{"frobnicate"}, wantExit: ExitArgparse, wantStderrDegree: []string{InvalidChoiceFragment, "frobnicate"}},
		{name: "force short alone", args: []string{"-f"}, wantExit: ExitOK, wantStdoutDegree: []string{UsageFragment}},
		{name: "force long alone", args: []string{"--force"}, wantExit: ExitOK, wantStdoutDegree: []string{UsageFragment}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			res := runReferenceCLI(t, home, tc.args, "")
			if res.exitCode != int(tc.wantExit) {
				t.Fatalf("exit = %d, want %d; stdout=%q stderr=%q", res.exitCode, tc.wantExit, res.stdout, res.stderr)
			}
			for _, frag := range tc.wantStdoutDegree {
				if !strings.Contains(res.stdout, frag) {
					t.Errorf("stdout missing %q; got %q", frag, res.stdout)
				}
			}
			for _, frag := range tc.wantStderrDegree {
				if !strings.Contains(res.stderr, frag) {
					t.Errorf("stderr missing %q; got %q", frag, res.stderr)
				}
			}
			if tc.wantEmptyStderr && res.stderr != "" {
				t.Errorf("stderr = %q, want empty", res.stderr)
			}
		})
	}
}

// TestCommandWithoutProject covers section 4.2: every one of the 12 accepted
// command forms invoked with no project name must exit 1 with the exact
// project-required message.
func TestCommandWithoutProject(t *testing.T) {
	home := t.TempDir()

	for _, cmd := range CommandAliases {
		cmd := cmd
		t.Run(cmd, func(t *testing.T) {
			res := runReferenceCLI(t, home, []string{cmd}, "")
			if res.exitCode != int(ExitOperational) {
				t.Fatalf("exit = %d, want %d; stderr=%q", res.exitCode, ExitOperational, res.stderr)
			}
			if res.stderr != ProjectNameRequiredMessage {
				t.Errorf("stderr = %q, want %q", res.stderr, ProjectNameRequiredMessage)
			}
		})
	}
}

// TestCommandWithMissingProject covers section 4.3: every one of the 12
// accepted command forms invoked with a non-existent project must exit 1 with
// the exact project-not-found message. The absolute tmp config directory in
// the emitted output is normalised to the <confdir> placeholder.
func TestCommandWithMissingProject(t *testing.T) {
	home := t.TempDir()
	const projectName = "nosuchproj"

	for _, cmd := range CommandAliases {
		cmd := cmd
		t.Run(cmd, func(t *testing.T) {
			res := runReferenceCLI(t, home, []string{cmd, projectName}, "")
			if res.exitCode != int(ExitOperational) {
				t.Fatalf("exit = %d, want %d; stderr=%q", res.exitCode, ExitOperational, res.stderr)
			}

			if !strings.Contains(res.stderr, ProjectNotFoundSuffix) {
				t.Errorf("stderr missing %q; got %q", ProjectNotFoundSuffix, res.stderr)
			}

			confdir := filepath.Join(home, ConfigDirName)
			normalized := strings.ReplaceAll(res.stderr, confdir, "<confdir>")
			want := fmt.Sprintf(ProjectConfigNotFoundFormat, projectName, "<confdir>")
			if normalized != want {
				t.Errorf("stderr normalized = %q, want %q", normalized, want)
			}
		})
	}
}

// TestListProjects covers section 4.4: --list/-l on an empty config directory
// exits 0 with an empty stdout and a stderr message (defect D11), while the
// same flags on the four fixtures print the sorted project names to stdout.
func TestListProjects(t *testing.T) {
	t.Run("empty config directory", func(t *testing.T) {
		home := t.TempDir()
		for _, flag := range []string{"--list", "-l"} {
			flag := flag
			t.Run(flag, func(t *testing.T) {
				res := runReferenceCLI(t, home, []string{flag}, "")
				if res.exitCode != int(ExitOK) {
					t.Fatalf("exit = %d, want %d; stdout=%q stderr=%q", res.exitCode, ExitOK, res.stdout, res.stderr)
				}
				if res.stdout != "" {
					t.Errorf("stdout = %q, want empty", res.stdout)
				}
				if res.stderr != NoProjectsConfiguredMessage {
					t.Errorf("stderr = %q, want %q", res.stderr, NoProjectsConfiguredMessage)
				}
			})
		}
	})

	t.Run("with fixtures", func(t *testing.T) {
		home := t.TempDir()
		writeFixture(t, home, fixtureDemoContent, "demo")
		writeFixture(t, home, fixtureEmptyContent, "empty")
		writeFixture(t, home, fixturePartialContent, "partial")
		writeFixture(t, home, fixtureQuotedContent, "quoted")

		want := "demo\nempty\npartial\nquoted\n"
		for _, flag := range []string{"--list", "-l"} {
			flag := flag
			t.Run(flag, func(t *testing.T) {
				res := runReferenceCLI(t, home, []string{flag}, "")
				if res.exitCode != int(ExitOK) {
					t.Fatalf("exit = %d, want %d; stdout=%q stderr=%q", res.exitCode, ExitOK, res.stdout, res.stderr)
				}
				if res.stdout != want {
					t.Errorf("stdout = %q, want %q", res.stdout, want)
				}
				if res.stderr != "" {
					t.Errorf("stderr = %q, want empty", res.stderr)
				}
			})
		}
	})
}

// TestDeleteWithConfirmation covers section 4.5: deleting the "partial"
// project (which has no local_mount) with stdin "n" cancels and keeps the
// file, while "y" deletes the config file. Both exit 0.
func TestDeleteWithConfirmation(t *testing.T) {
	cases := []struct {
		name       string
		answer     string
		contain    []string
		fileExists bool
	}{
		{name: "decline", answer: "n\n", contain: []string{DeleteConfigPromptFragment, DeleteCancelledFragment}, fileExists: true},
		{name: "confirm", answer: "y\n", contain: []string{DeleteConfigPromptFragment, DeleteConfirmedFragment}, fileExists: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			writeFixture(t, home, fixturePartialContent, "partial")

			res := runReferenceCLI(t, home, []string{"delete", "partial"}, tc.answer)
			if res.exitCode != int(ExitOK) {
				t.Fatalf("exit = %d, want %d; stdout=%q stderr=%q", res.exitCode, ExitOK, res.stdout, res.stderr)
			}
			for _, frag := range tc.contain {
				if !strings.Contains(res.stdout, frag) {
					t.Errorf("stdout missing %q; got %q", frag, res.stdout)
				}
			}

			path := filepath.Join(home, ConfigDirName, "partial"+ConfigFileSuffix)
			_, statErr := os.Stat(path)
			if tc.fileExists {
				if statErr != nil {
					t.Errorf("config file %s should still exist after decline: %v", path, statErr)
				}
			} else {
				if statErr == nil {
					t.Errorf("config file %s should be deleted after confirm", path)
				}
			}
		})
	}
}

// TestDeleteForceAfterCommandRejected covers section 4.6: a -f/--force placed
// after the delete command together with a project name is an argparse error
// (exit 2) carrying the unrecognized-arguments diagnostic.
func TestDeleteForceAfterCommandRejected(t *testing.T) {
	home := t.TempDir()

	cases := []struct {
		name string
		args []string
	}{
		{name: "short force after command", args: []string{"delete", "-f", "demo"}},
		{name: "long force after command", args: []string{"delete", "--force", "demo"}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			res := runReferenceCLI(t, home, tc.args, "")
			if res.exitCode != int(ExitArgparse) {
				t.Fatalf("exit = %d, want %d; stdout=%q stderr=%q", res.exitCode, ExitArgparse, res.stdout, res.stderr)
			}
			if !strings.Contains(res.stderr, UnrecognizedArgumentsFragment) {
				t.Errorf("stderr missing %q; got %q", UnrecognizedArgumentsFragment, res.stderr)
			}
			if !strings.Contains(res.stderr, "demo") {
				t.Errorf("stderr missing 'demo'; got %q", res.stderr)
			}
		})
	}
}

// TestDeleteForceBeforeCommand covers the valid force position of section 4.6:
// "-f delete partial" deletes the config file (exit 0) because --force before
// the command is accepted.
func TestDeleteForceBeforeCommand(t *testing.T) {
	home := t.TempDir()
	writeFixture(t, home, fixturePartialContent, "partial")

	res := runReferenceCLI(t, home, []string{"-f", "delete", "partial"}, "")
	if res.exitCode != int(ExitOK) {
		t.Fatalf("exit = %d, want %d; stdout=%q stderr=%q", res.exitCode, ExitOK, res.stdout, res.stderr)
	}

	path := filepath.Join(home, ConfigDirName, "partial"+ConfigFileSuffix)
	if _, statErr := os.Stat(path); statErr == nil {
		t.Errorf("config file %s should be deleted by -f delete partial", path)
	}
}
