package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cy83rt00n/workspace-manager/internal/config"
	"github.com/cy83rt00n/workspace-manager/internal/system"
	"github.com/cy83rt00n/workspace-manager/internal/workspace"
)

// commandAliases is the canonical 12-form command list used to drive the
// "without project" and "nonexistent project" matrix rows without copying them.
var commandAliases = []string{
	"mount", "m",
	"unmount", "u",
	"run", "r",
	"connect", "c", "ssh",
	"delete", "del", "rm",
}

// fakeSystem is a scripted workspace.System. The zero value is benign, so the
// exit-code and list tables can simply pass a zero fakeSystem. It never touches
// the network, a real terminal or a real mount: IsMounted/Mount/Unmount/Start
// are pure stubs and CheckNet returns the scripted message.
type fakeSystem struct {
	mounted     bool
	checkNetOK  bool
	checkNetMsg string
	mountErr    error
	startErr    error
}

func (f fakeSystem) CheckNet(ctx context.Context, alias string) (bool, string) {
	return f.checkNetOK, f.checkNetMsg
}

func (f fakeSystem) IsMounted(path string) bool { return f.mounted }

func (f fakeSystem) Mount(ctx context.Context, remotePath, localMount string, poll func(int)) error {
	return f.mountErr
}

func (f fakeSystem) Unmount(ctx context.Context, localMount string, out func(string, ...any)) (bool, error) {
	return true, nil
}

func (f fakeSystem) Start(ctx context.Context, name string, args ...string) (*system.Proc, error) {
	return &system.Proc{}, f.startErr
}

// writeConfig writes a single project config file in dir with the three
// recognised keys. It returns the absolute config path so tests can check
// existence afterwards.
func writeConfig(t *testing.T, dir, name, remote, local, editor string) string {
	t.Helper()
	content := fmt.Sprintf("remote_path = %q\nlocal_mount = %q\neditor_cmd = %q\n", remote, local, editor)
	path := filepath.Join(dir, name+config.ConfigSuffix)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// run invokes runCLI with the test version string and captures stdout/stderr
// into buffers. It always uses the non-TTY branch (bytes.Buffer is never a
// char device), so the spinner output is deterministic.
func run(t *testing.T, args []string, stdin string, dir string, fs fakeSystem) (int, string, string) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	code := runCLI(
		args,
		strings.NewReader(stdin),
		&outBuf, &errBuf,
		dir,
		"0.0.0-test",
		func(ctx context.Context) workspace.System { return fs },
	)
	return code, outBuf.String(), errBuf.String()
}

// wantUsage is the expected usage output used by the help/usage assertions.
func wantUsage() string {
	return fmt.Sprintf(usageTemplate, "0.0.0-test")
}

// TestCLIUsageAndErrors drives the §10.1 exit-code matrix: usage cases, unknown
// command/option, extra positional, the 12 missing-project cases and the 12
// nonexistent-project cases.
func TestCLIUsageAndErrors(t *testing.T) {
	dir := t.TempDir() // empty config dir for the matrix rows that need none

	t.Run("usage cases", func(t *testing.T) {
		cases := []struct {
			name string
			args []string
		}{
			{name: "no-args", args: []string{}},
			{name: "-h", args: []string{"-h"}},
			{name: "--help", args: []string{"--help"}},
			{name: "help", args: []string{"help"}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				code, out, errOut := run(t, tc.args, "", dir, fakeSystem{})
				if code != 0 {
					t.Errorf("code = %d, want 0", code)
				}
				if out != wantUsage() {
					t.Errorf("stdout = %q, want usage %q", out, wantUsage())
				}
				if errOut != "" {
					t.Errorf("stderr = %q, want empty", errOut)
				}
				if !strings.Contains(out, "Workspace Manager v0.0.0-test (Go)") {
					t.Errorf("usage missing banner, got %q", out)
				}
				if !strings.Contains(out, "wsm-go") {
					t.Errorf("usage missing prog name wsm-go, got %q", out)
				}
			})
		}
	})

	t.Run("unknown command", func(t *testing.T) {
		code, _, errOut := run(t, []string{"bogus"}, "", dir, fakeSystem{})
		if code != 2 {
			t.Errorf("code = %d, want 2", code)
		}
		if errOut != "unknown command: bogus\n" {
			t.Errorf("stderr = %q, want %q", errOut, "unknown command: bogus\n")
		}
	})

	t.Run("unknown option", func(t *testing.T) {
		code, _, errOut := run(t, []string{"--bogus"}, "", dir, fakeSystem{})
		if code != 2 {
			t.Errorf("code = %d, want 2", code)
		}
		if errOut != "unknown option: --bogus\n" {
			t.Errorf("stderr = %q, want %q", errOut, "unknown option: --bogus\n")
		}
	})

	t.Run("extra positional", func(t *testing.T) {
		code, _, errOut := run(t, []string{"mount", "a", "b"}, "", dir, fakeSystem{})
		if code != 2 {
			t.Errorf("code = %d, want 2", code)
		}
		if errOut != "unexpected argument: b\n" {
			t.Errorf("stderr = %q, want %q", errOut, "unexpected argument: b\n")
		}
	})

	t.Run("without project x12", func(t *testing.T) {
		for _, cmd := range commandAliases {
			t.Run(cmd, func(t *testing.T) {
				code, _, errOut := run(t, []string{cmd}, "", dir, fakeSystem{})
				if code != 1 {
					t.Errorf("code = %d, want 1", code)
				}
				if errOut != "Error: project name required\n" {
					t.Errorf("stderr = %q, want %q", errOut, "Error: project name required\n")
				}
			})
		}
	})

	t.Run("nonexistent project x12", func(t *testing.T) {
		for _, cmd := range commandAliases {
			t.Run(cmd, func(t *testing.T) {
				code, _, errOut := run(t, []string{cmd, "nope"}, "", dir, fakeSystem{})
				if code != 1 {
					t.Errorf("code = %d, want 1", code)
				}
				want := fmt.Sprintf("Error: Project config 'nope' not found in %s\n", dir)
				if errOut != want {
					t.Errorf("stderr = %q, want %q", errOut, want)
				}
			})
		}
	})
}

// TestCLIList drives the --list/-l behaviour: empty directory, fixture output
// and the "list wins over command" precedence (ADR-001 D17).
func TestCLIList(t *testing.T) {
	emptyDir := t.TempDir()

	t.Run("list empty", func(t *testing.T) {
		code, out, errOut := run(t, []string{"--list"}, "", emptyDir, fakeSystem{})
		if code != 0 {
			t.Errorf("code = %d, want 0", code)
		}
		if out != "" {
			t.Errorf("stdout = %q, want empty", out)
		}
		if errOut != "No projects configured.\n" {
			t.Errorf("stderr = %q, want %q", errOut, "No projects configured.\n")
		}
	})

	fixtureDir := t.TempDir()
	writeConfig(t, fixtureDir, "beta", "beta:/data", "/mnt/beta", "zed")
	writeConfig(t, fixtureDir, "alpha", "alpha:/data", "/mnt/alpha", "zed")

	t.Run("list fixtures --list", func(t *testing.T) {
		code, out, errOut := run(t, []string{"--list"}, "", fixtureDir, fakeSystem{})
		if code != 0 {
			t.Errorf("code = %d, want 0", code)
		}
		if out != "alpha\nbeta\n" {
			t.Errorf("stdout = %q, want %q", out, "alpha\nbeta\n")
		}
		if errOut != "" {
			t.Errorf("stderr = %q, want empty", errOut)
		}
	})

	t.Run("list fixtures -l", func(t *testing.T) {
		code, out, errOut := run(t, []string{"-l"}, "", fixtureDir, fakeSystem{})
		if code != 0 {
			t.Errorf("code = %d, want 0", code)
		}
		if out != "alpha\nbeta\n" {
			t.Errorf("stdout = %q, want %q", out, "alpha\nbeta\n")
		}
		if errOut != "" {
			t.Errorf("stderr = %q, want empty", errOut)
		}
	})

	t.Run("list wins over command", func(t *testing.T) {
		code, out, errOut := run(t, []string{"-l", "mount", "demo"}, "", fixtureDir, fakeSystem{})
		if code != 0 {
			t.Errorf("code = %d, want 0", code)
		}
		if out != "alpha\nbeta\n" {
			t.Errorf("stdout = %q, want %q", out, "alpha\nbeta\n")
		}
		if errOut != "" {
			t.Errorf("stderr = %q, want empty", errOut)
		}
	})
}

// TestCLIMountNetFail drives the §10.2 non-TTY output matrix: a mount whose
// network check fails emits "Resolving alias..." then "FAILED: <msg>" and exits
// 1 WITHOUT a second "Error:" line (the ExitError.Msg is empty).
func TestCLIMountNetFail(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "demo", "alias:/data", "/mnt/x", "zed")
	fs := fakeSystem{
		checkNetOK:  false,
		checkNetMsg: "Host alias:22 is unreachable. Check network or VPN.",
	}
	code, out, errOut := run(t, []string{"mount", "demo"}, "", dir, fs)

	if code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
	wantOut := "Resolving alias...\nFAILED: Host alias:22 is unreachable. Check network or VPN.\n"
	if out != wantOut {
		t.Errorf("stdout = %q, want %q", out, wantOut)
	}
	if errOut != "" {
		t.Errorf("stderr = %q, want empty", errOut)
	}
}

// TestCLIDelete drives the §10.3 integration-delete matrix against the real
// config package (read/write on t.TempDir) with the fake System. Each case gets
// its own TempDir; file existence is verified after runCLI.
func TestCLIDelete(t *testing.T) {
	cases := []struct {
		name        string
		args        []string
		stdin       string
		mounted     bool
		wantCode    int
		wantStdout  string
		wantStderr  string
		wantDeleted bool
	}{
		{name: "delete n", args: []string{"delete", "demo"}, stdin: "n\n", wantCode: 0, wantStdout: "Delete config \"demo\"? [y/N]: Cancelled.\n", wantDeleted: false},
		{name: "delete y", args: []string{"delete", "demo"}, stdin: "y\n", wantCode: 0, wantStdout: "Delete config \"demo\"? [y/N]: Config \"demo\" deleted.\n", wantDeleted: true},
		{name: "delete yes", args: []string{"delete", "demo"}, stdin: "yes\n", wantCode: 0, wantStdout: "Delete config \"demo\"? [y/N]: Config \"demo\" deleted.\n", wantDeleted: true},
		{name: "delete EOF", args: []string{"delete", "demo"}, stdin: "", wantCode: 0, wantStdout: "Delete config \"demo\"? [y/N]: Cancelled.\n", wantDeleted: false},
		{name: "delete -f before", args: []string{"-f", "delete", "demo"}, wantCode: 0, wantStdout: "Config \"demo\" deleted.\n", wantDeleted: true},
		{name: "delete -f after", args: []string{"delete", "demo", "-f"}, wantCode: 0, wantStdout: "Config \"demo\" deleted.\n", wantDeleted: true},
		{name: "delete blocked mount", args: []string{"delete", "demo"}, mounted: true, wantCode: 1, wantStdout: "", wantStderr: "Error: Project is mounted. Unmount first.\n", wantDeleted: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			confPath := writeConfig(t, dir, "demo", "alias:/data", "/mnt/x", "zed")
			fs := fakeSystem{mounted: tc.mounted}

			code, out, errOut := run(t, tc.args, tc.stdin, dir, fs)

			if code != tc.wantCode {
				t.Errorf("code = %d, want %d", code, tc.wantCode)
			}
			if out != tc.wantStdout {
				t.Errorf("stdout = %q, want %q", out, tc.wantStdout)
			}
			if tc.wantStderr != "" && errOut != tc.wantStderr {
				t.Errorf("stderr = %q, want %q", errOut, tc.wantStderr)
			}
			if tc.wantStderr == "" && errOut != "" {
				t.Errorf("stderr = %q, want empty", errOut)
			}

			_, statErr := os.Stat(confPath)
			if tc.wantDeleted {
				if statErr == nil {
					t.Errorf("config file should have been deleted: %s", confPath)
				}
			} else {
				if statErr != nil {
					t.Errorf("config file should still exist after cancellation: %v", statErr)
				}
			}
		})
	}
}
