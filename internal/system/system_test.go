package system

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// fakeCommand is one expected invocation: a name/args match yields the preset
// stdout/stderr/rc/err. An empty args slice acts as a name-only wildcard.
type fakeCommand struct {
	name           string
	args           []string
	stdout, stderr string
	rc             int
	err            error
}

// fakeRunner records every call and resolves it against a script of expected
// commands. It never launches a real process.
type fakeRunner struct {
	script []fakeCommand
	calls  [][]string // recorded invocations: [name, arg1, arg2, ...]
}

// find returns the first script entry matching name and, when its args are
// non-empty, the exact args; an empty-args entry is a name-only wildcard.
func (f *fakeRunner) find(name string, args []string) *fakeCommand {
	for i := range f.script {
		c := &f.script[i]
		if c.name != name {
			continue
		}
		if len(c.args) == 0 {
			return c
		}
		if reflect.DeepEqual(c.args, args) {
			return c
		}
	}
	return nil
}

// Run records the call and returns the matched preset, translating a non-zero
// rc into *RunError.
func (f *fakeRunner) Run(ctx context.Context, name string, args ...string) (string, string, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	c := f.find(name, args)
	if c == nil {
		return "", "", nil
	}
	if c.rc != 0 {
		return c.stdout, c.stderr, &RunError{ExitCode: c.rc}
	}
	return c.stdout, c.stderr, c.err
}

// Start records the call and returns a Proc whose Wait yields the matched
// preset result.
func (f *fakeRunner) Start(ctx context.Context, name string, args ...string) (*Proc, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	c := f.find(name, args)
	if c == nil {
		return &Proc{wait: func() error { return nil }}, nil
	}
	return &Proc{wait: func() error {
		if c.rc != 0 {
			return &RunError{ExitCode: c.rc}
		}
		return c.err
	}}, nil
}

// fakeMountChecker reports is-mounted per path from a plain map.
type fakeMountChecker struct {
	mounted map[string]bool
}

func (f *fakeMountChecker) IsMounted(path string) bool {
	return f.mounted[path]
}

// fakeNetworkChecker returns a fixed (ok, msg) pair.
type fakeNetworkChecker struct {
	ok  bool
	msg string
}

func (f *fakeNetworkChecker) CheckNet(ctx context.Context, alias string) (bool, string) {
	return f.ok, f.msg
}

// sshfsArgs is the exact, byte-for-byte sshfs argument vector (9 elements).
func sshfsArgs(remotePath, localMount string) []string {
	return []string{
		remotePath, localMount,
		"-o", "cache=yes,cache_stat_timeout=1200,cache_dir_timeout=1200,cache_link_timeout=1200",
		"-o", "compression=yes,reconnect,ServerAliveInterval=15",
		"-o", "kernel_cache,noauto_cache",
	}
}

func TestMountSSHFSVector(t *testing.T) {
	remotePath := "demo-alias:/opt/app"
	localMount := filepath.Join(t.TempDir(), "mnt")
	runner := &fakeRunner{script: []fakeCommand{{name: "sshfs", rc: 0}}}
	sys := &System{Runner: runner}

	var polls []int
	err := sys.Mount(context.Background(), remotePath, localMount, func(i int) {
		polls = append(polls, i)
	})
	if err != nil {
		t.Fatalf("Mount returned error: %v", err)
	}

	if len(runner.calls) < 1 {
		t.Fatalf("expected at least one recorded call, got %d", len(runner.calls))
	}
	got := runner.calls[0]
	want := append([]string{"sshfs"}, sshfsArgs(remotePath, localMount)...)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sshfs vector = %v, want %v", got, want)
	}

	// poll must be invoked with an incrementing counter (0, 1, ...), at least once.
	if len(polls) < 1 {
		t.Fatalf("poll was not called; got %d calls", len(polls))
	}
	for i, p := range polls {
		if p != i {
			t.Errorf("poll[%d] = %d, want %d", i, p, i)
		}
	}
}

func TestMountNonZeroExit(t *testing.T) {
	remotePath := "demo-alias:/opt/app"
	localMount := filepath.Join(t.TempDir(), "mnt")
	runner := &fakeRunner{script: []fakeCommand{{name: "sshfs", rc: 42}}}
	sys := &System{Runner: runner}

	err := sys.Mount(context.Background(), remotePath, localMount, func(int) {})
	if err == nil {
		t.Fatal("Mount returned nil, want error for non-zero sshfs exit")
	}
	runErr, ok := err.(*RunError)
	if !ok {
		t.Fatalf("Mount error = %T (%v), want *RunError", err, err)
	}
	if runErr.ExitCode != 42 {
		t.Errorf("Mount exit code = %d, want 42", runErr.ExitCode)
	}
}

func TestUnmountBranches(t *testing.T) {
	ctx := context.Background()
	localMount := "/mnt/x"

	collectOut := func() (func(string, ...any), *[]string) {
		var lines []string
		return func(format string, args ...any) {
			lines = append(lines, fmt.Sprintf(format, args...))
		}, &lines
	}

	t.Run("not mounted -> alreadyUnmounted true, no fuser lines", func(t *testing.T) {
		runner := &fakeRunner{}
		sys := &System{Runner: runner, MountChecker: &fakeMountChecker{mounted: map[string]bool{}}}
		out, lines := collectOut()

		already, err := sys.Unmount(ctx, localMount, out)
		if err != nil {
			t.Fatalf("Unmount returned error: %v", err)
		}
		if !already {
			t.Errorf("alreadyUnmounted = false, want true")
		}
		if len(*lines) != 0 {
			t.Errorf("out lines = %v, want none", *lines)
		}
	})

	t.Run("mounted, umount ok", func(t *testing.T) {
		runner := &fakeRunner{script: []fakeCommand{
			{name: "fuser", args: []string{"-k", "-M", localMount}, rc: 0},
			{name: "umount", args: []string{localMount}, rc: 0},
		}}
		sys := &System{Runner: runner, MountChecker: &fakeMountChecker{mounted: map[string]bool{localMount: true}}}
		out, lines := collectOut()

		already, err := sys.Unmount(ctx, localMount, out)
		if err != nil {
			t.Fatalf("Unmount returned error: %v", err)
		}
		if already {
			t.Errorf("alreadyUnmounted = true, want false")
		}
		wantLines := []string{FuserKill, fmt.Sprintf(Unmounting, localMount)}
		if !reflect.DeepEqual(*lines, wantLines) {
			t.Errorf("out lines = %v, want %v", *lines, wantLines)
		}
	})

	t.Run("mounted, umount err, lazy ok", func(t *testing.T) {
		runner := &fakeRunner{script: []fakeCommand{
			{name: "fuser", args: []string{"-k", "-M", localMount}, rc: 1},
			{name: "umount", args: []string{localMount}, rc: 1},
			{name: "umount", args: []string{"-l", localMount}, rc: 0},
		}}
		sys := &System{Runner: runner, MountChecker: &fakeMountChecker{mounted: map[string]bool{localMount: true}}}
		out, lines := collectOut()

		already, err := sys.Unmount(ctx, localMount, out)
		if err != nil {
			t.Fatalf("Unmount returned error: %v", err)
		}
		if already {
			t.Errorf("alreadyUnmounted = true, want false")
		}
		wantLines := []string{FuserKill, fmt.Sprintf(Unmounting, localMount), LazyUnmount}
		if !reflect.DeepEqual(*lines, wantLines) {
			t.Errorf("out lines = %v, want %v", *lines, wantLines)
		}
	})

	t.Run("mounted, umount err, lazy err -> error (D4)", func(t *testing.T) {
		runner := &fakeRunner{script: []fakeCommand{
			{name: "fuser", args: []string{"-k", "-M", localMount}, rc: 0},
			{name: "umount", args: []string{localMount}, rc: 1},
			{name: "umount", args: []string{"-l", localMount}, rc: 7},
		}}
		sys := &System{Runner: runner, MountChecker: &fakeMountChecker{mounted: map[string]bool{localMount: true}}}
		out, lines := collectOut()

		already, err := sys.Unmount(ctx, localMount, out)
		if err == nil {
			t.Fatal("Unmount returned nil, want error for lazy umount failure")
		}
		if already {
			t.Errorf("alreadyUnmounted = true, want false")
		}
		wantLines := []string{FuserKill, fmt.Sprintf(Unmounting, localMount), LazyUnmount}
		if !reflect.DeepEqual(*lines, wantLines) {
			t.Errorf("out lines = %v, want %v", *lines, wantLines)
		}
	})
}

func TestMountpointIsMounted(t *testing.T) {
	t.Run("mounted path yields true", func(t *testing.T) {
		runner := &fakeRunner{script: []fakeCommand{{name: "mountpoint", args: []string{"-q", "/mnt/x"}, rc: 0}}}
		mc := &mountChecker{runner: runner}
		if !mc.IsMounted("/mnt/x") {
			t.Errorf("IsMounted(/mnt/x) = false, want true")
		}
		got := runner.calls[0]
		want := []string{"mountpoint", "-q", "/mnt/x"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("mountpoint vector = %v, want %v", got, want)
		}
	})

	t.Run("non-zero exit yields false", func(t *testing.T) {
		runner := &fakeRunner{script: []fakeCommand{{name: "mountpoint", args: []string{"-q", "/mnt/x"}, rc: 1}}}
		mc := &mountChecker{runner: runner}
		if mc.IsMounted("/mnt/x") {
			t.Errorf("IsMounted(/mnt/x) = true, want false")
		}
	})

	t.Run("empty path yields false", func(t *testing.T) {
		runner := &fakeRunner{}
		mc := &mountChecker{runner: runner}
		if mc.IsMounted("") {
			t.Errorf("IsMounted(\"\") = true, want false")
		}
		if len(runner.calls) != 0 {
			t.Errorf("empty path should not invoke mountpoint; got %v", runner.calls)
		}
	})
}

func TestCheckNetBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("last wins across case variants", func(t *testing.T) {
		runner := &fakeRunner{script: []fakeCommand{
			{name: "ssh", args: []string{"-G", "demo"}, rc: 0, stdout: "Hostname host1\nPort 2222\nhostname host2\nport 22\n"},
			{name: "nc", args: []string{"-z", "-w", "2", "host2", "22"}, rc: 0},
		}}
		nc := &networkChecker{runner: runner}

		ok, msg := nc.CheckNet(ctx, "demo")
		if !ok {
			t.Fatalf("CheckNet ok = false, msg = %q", msg)
		}
		if len(runner.calls) < 2 {
			t.Fatalf("expected ssh and nc calls, got %v", runner.calls)
		}
		wantNC := []string{"nc", "-z", "-w", "2", "host2", "22"}
		if !reflect.DeepEqual(runner.calls[1], wantNC) {
			t.Errorf("nc vector = %v, want %v", runner.calls[1], wantNC)
		}
	})

	t.Run("empty host -> cannot resolve", func(t *testing.T) {
		runner := &fakeRunner{script: []fakeCommand{{name: "ssh", args: []string{"-G", "demo"}, rc: 0, stdout: "User root\n"}}}
		nc := &networkChecker{runner: runner}

		ok, msg := nc.CheckNet(ctx, "demo")
		if ok {
			t.Fatalf("CheckNet ok = true, want false")
		}
		want := fmt.Sprintf(CannotResolveHostname, "demo")
		if msg != want {
			t.Errorf("msg = %q, want %q", msg, want)
		}
	})

	t.Run("nc non-zero -> unreachable", func(t *testing.T) {
		runner := &fakeRunner{script: []fakeCommand{
			{name: "ssh", args: []string{"-G", "demo"}, rc: 0, stdout: "Hostname host\n"},
			{name: "nc", args: []string{"-z", "-w", "2", "host", "22"}, rc: 1},
		}}
		nc := &networkChecker{runner: runner}

		ok, msg := nc.CheckNet(ctx, "demo")
		if ok {
			t.Fatalf("CheckNet ok = true, want false")
		}
		want := fmt.Sprintf(HostUnreachable, "host", "22")
		if msg != want {
			t.Errorf("msg = %q, want %q", msg, want)
		}
	})

	t.Run("ssh error/rc -> alias not found (D8)", func(t *testing.T) {
		for _, rc := range []int{1, 42} {
			runner := &fakeRunner{script: []fakeCommand{{name: "ssh", args: []string{"-G", "demo"}, rc: rc}}}
			nc := &networkChecker{runner: runner}

			ok, msg := nc.CheckNet(ctx, "demo")
			if ok {
				t.Fatalf("CheckNet ok = true for rc=%d, want false", rc)
			}
			want := fmt.Sprintf(SSHAliasNotFound, "demo")
			if msg != want {
				t.Errorf("rc=%d msg = %q, want %q", rc, msg, want)
			}
		}
	})
}

func TestGenerateKeypairCleaning(t *testing.T) {
	cases := []struct {
		name        string
		in          string
		wantPrivate string // cleaned base name, or "" for invalid
		wantInvalid bool
	}{
		{name: "simple", in: "foo", wantPrivate: "foo"},
		{name: "directory stripped", in: "dir/foo.key", wantPrivate: "foo"},
		{name: "pub stripped", in: "bar.pub", wantPrivate: "bar"},
		{name: "double extension stripped once", in: "x.key.pub", wantPrivate: "x"},
		{name: "empty name invalid", in: "", wantInvalid: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			runner := &fakeRunner{script: []fakeCommand{
				{name: "ssh-keygen", rc: 0},
				{name: "mv", rc: 0},
			}}
			sys := &System{Runner: runner}

			private, public, err := sys.GenerateKeypair(context.Background(), tc.in, dir)
			if tc.wantInvalid {
				if err == nil {
					t.Fatalf("GenerateKeypair returned nil error, want %q", InvalidKeyName)
				}
				if err.Error() != InvalidKeyName {
					t.Errorf("err = %q, want %q", err.Error(), InvalidKeyName)
				}
				return
			}
			if err != nil {
				t.Fatalf("GenerateKeypair returned error: %v", err)
			}
			wantPrivate := filepath.Join(dir, tc.wantPrivate+".key")
			wantPublic := filepath.Join(dir, tc.wantPrivate+".pub")
			if private != wantPrivate {
				t.Errorf("private = %q, want %q", private, wantPrivate)
			}
			if public != wantPublic {
				t.Errorf("public = %q, want %q", public, wantPublic)
			}

			if len(runner.calls) < 2 {
				t.Fatalf("expected 2 calls, got %v", runner.calls)
			}
			wantKeygen := []string{"ssh-keygen", "-t", "ed25519", "-f", wantPrivate, "-C", "wsm-" + tc.wantPrivate, "-N", ""}
			if !reflect.DeepEqual(runner.calls[0], wantKeygen) {
				t.Errorf("ssh-keygen vector = %v, want %v", runner.calls[0], wantKeygen)
			}
			wantMV := []string{"mv", wantPrivate + ".pub", wantPublic}
			if !reflect.DeepEqual(runner.calls[1], wantMV) {
				t.Errorf("mv vector = %v, want %v", runner.calls[1], wantMV)
			}
		})
	}
}

func TestGenerateKeypairExistingPrivate(t *testing.T) {
	dir := t.TempDir()
	private := filepath.Join(dir, "foo.key")
	if err := os.WriteFile(private, []byte("existing"), 0o600); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
	runner := &fakeRunner{}
	sys := &System{Runner: runner}

	_, _, err := sys.GenerateKeypair(context.Background(), "foo", dir)
	if err == nil {
		t.Fatal("GenerateKeypair returned nil, want already-exists error")
	}
	want := private + " already exists"
	if err.Error() != want {
		t.Errorf("err = %q, want %q", err.Error(), want)
	}
	if len(runner.calls) != 0 {
		t.Errorf("expected no calls, got %v", runner.calls)
	}
}

func TestGenerateKeypairSSHKeygenError(t *testing.T) {
	dir := t.TempDir()
	runner := &fakeRunner{script: []fakeCommand{{name: "ssh-keygen", rc: 42}}}
	sys := &System{Runner: runner}

	_, _, err := sys.GenerateKeypair(context.Background(), "foo", dir)
	if err == nil {
		t.Fatal("GenerateKeypair returned nil, want error from ssh-keygen")
	}
	if _, ok := err.(*RunError); !ok {
		t.Errorf("err = %T (%v), want *RunError", err, err)
	}
}

func TestGenerateKeypairMVError(t *testing.T) {
	dir := t.TempDir()
	runner := &fakeRunner{script: []fakeCommand{
		{name: "ssh-keygen", rc: 0},
		{name: "mv", rc: 7},
	}}
	sys := &System{Runner: runner}

	_, _, err := sys.GenerateKeypair(context.Background(), "foo", dir)
	if err == nil {
		t.Fatal("GenerateKeypair returned nil, want error from mv")
	}
	if _, ok := err.(*RunError); !ok {
		t.Errorf("err = %T (%v), want *RunError", err, err)
	}
}

// TestNetworkCheckerFake verifies the injected NetworkChecker is consumed by the
// embedded System CheckNet method (used by the workspace layer).
func TestNetworkCheckerFake(t *testing.T) {
	sys := &System{NetworkChecker: &fakeNetworkChecker{ok: true}}
	ok, msg := sys.CheckNet(context.Background(), "demo")
	if !ok || msg != "" {
		t.Errorf("CheckNet = (%v, %q), want (true, \"\")", ok, msg)
	}
}
