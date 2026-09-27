package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/cy83rt00n/workspace-manager/internal/config"
	"github.com/cy83rt00n/workspace-manager/internal/system"
)

// recorder captures the injected output callbacks.
type recorder struct {
	spin     []string
	spinDone []string
	out      []string
	errOut   []string
}

func (r *recorder) recSpin(stage string)   { r.spin = append(r.spin, stage) }
func (r *recorder) recSpinDone(msg string) { r.spinDone = append(r.spinDone, msg) }
func (r *recorder) recOut(format string, args ...any) {
	r.out = append(r.out, fmt.Sprintf(format, args...))
}
func (r *recorder) recErrOut(format string, args ...any) {
	r.errOut = append(r.errOut, fmt.Sprintf(format, args...))
}

// fakeSystem implements workspace.System with scriptable behaviours and a record
// of the invocations. A nil function field yields a benign default.
type fakeSystem struct {
	checkNet  func(ctx context.Context, alias string) (bool, string)
	isMounted func(path string) bool
	mount     func(ctx context.Context, remotePath, localMount string, poll func(int)) error
	unmount   func(ctx context.Context, localMount string, out func(string, ...any)) (bool, error)
	start     func(ctx context.Context, name string, args ...string) (*system.Proc, error)

	checkNetAliases []string
	mountCalls      [][]string
	unmountLocals   []string
	startCalls      [][]string
}

func (f *fakeSystem) CheckNet(ctx context.Context, alias string) (bool, string) {
	f.checkNetAliases = append(f.checkNetAliases, alias)
	if f.checkNet != nil {
		return f.checkNet(ctx, alias)
	}
	return true, ""
}

func (f *fakeSystem) IsMounted(path string) bool {
	if f.isMounted != nil {
		return f.isMounted(path)
	}
	return false
}

func (f *fakeSystem) Mount(ctx context.Context, remotePath, localMount string, poll func(int)) error {
	f.mountCalls = append(f.mountCalls, []string{remotePath, localMount})
	if f.mount != nil {
		return f.mount(ctx, remotePath, localMount, poll)
	}
	return nil
}

func (f *fakeSystem) Unmount(ctx context.Context, localMount string, out func(string, ...any)) (bool, error) {
	f.unmountLocals = append(f.unmountLocals, localMount)
	if f.unmount != nil {
		return f.unmount(ctx, localMount, out)
	}
	return false, nil
}

func (f *fakeSystem) Start(ctx context.Context, name string, args ...string) (*system.Proc, error) {
	f.startCalls = append(f.startCalls, append([]string{name}, args...))
	if f.start != nil {
		return f.start(ctx, name, args...)
	}
	return &system.Proc{}, nil
}

// writeProject writes a single project config file in dir (real TOML content).
func writeProject(t *testing.T, dir, name, remotePath, localMount, editorCmd string) string {
	t.Helper()
	path := filepath.Join(dir, name+config.ConfigSuffix)
	content := fmt.Sprintf("remote_path = %q\nlocal_mount = %q\neditor_cmd = %q\n", remotePath, localMount, editorCmd)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write fixture %s: %v", path, err)
	}
	return path
}

// wantExitError asserts err is *ExitError with the given code and exact message.
func wantExitError(t *testing.T, err error, code int, msg string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected *ExitError, got nil")
	}
	ee, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("err = %T (%v), want *ExitError", err, err)
	}
	if ee.Code != code {
		t.Errorf("Code = %d, want %d", ee.Code, code)
	}
	if ee.Msg != msg {
		t.Errorf("Msg = %q, want %q", ee.Msg, msg)
	}
	if ee.ExitCode() != code {
		t.Errorf("ExitCode() = %d, want %d", ee.ExitCode(), code)
	}
}

func TestMountProject(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		dir := t.TempDir()
		eng := &Engine{ConfigDir: dir, System: &fakeSystem{}}
		err := eng.MountProject(context.Background(), "x")
		wantExitError(t, err, 1, fmt.Sprintf(ProjectNotFound, "x", dir))
	})

	t.Run("check net fail", func(t *testing.T) {
		dir := t.TempDir()
		writeProject(t, dir, "demo", "demo-alias:/opt/app", "/mnt/x", "zed")
		rec := &recorder{}
		sys := &fakeSystem{checkNet: func(ctx context.Context, alias string) (bool, string) {
			return false, "boom"
		}}
		eng := &Engine{ConfigDir: dir, Spin: rec.recSpin, SpinDone: rec.recSpinDone, System: sys}

		err := eng.MountProject(context.Background(), "demo")
		wantExitError(t, err, 1, "")
		wantSpinDone := []string{fmt.Sprintf(SpinFailed, "boom")}
		if !reflect.DeepEqual(rec.spinDone, wantSpinDone) {
			t.Errorf("SpinDone = %v, want %v", rec.spinDone, wantSpinDone)
		}
	})

	t.Run("success", func(t *testing.T) {
		dir := t.TempDir()
		writeProject(t, dir, "demo", "demo-alias:/opt/app", "/mnt/x", "zed")
		rec := &recorder{}
		sys := &fakeSystem{
			checkNet: func(ctx context.Context, alias string) (bool, string) { return true, "" },
			mount: func(ctx context.Context, remotePath, localMount string, poll func(int)) error {
				poll(0)
				return nil
			},
		}
		eng := &Engine{ConfigDir: dir, Spin: rec.recSpin, SpinDone: rec.recSpinDone, System: sys}

		if err := eng.MountProject(context.Background(), "demo"); err != nil {
			t.Fatalf("MountProject returned error: %v", err)
		}
		wantSpin := []string{
			fmt.Sprintf(MountResolving, "demo-alias"),
			fmt.Sprintf(MountSpinning, "demo-alias", "/mnt/x"),
		}
		if !reflect.DeepEqual(rec.spin, wantSpin) {
			t.Errorf("Spin = %v, want %v", rec.spin, wantSpin)
		}
		wantSpinDone := []string{
			fmt.Sprintf(MountResolved, "demo-alias"),
			fmt.Sprintf(MountMounted, "demo-alias"),
		}
		if !reflect.DeepEqual(rec.spinDone, wantSpinDone) {
			t.Errorf("SpinDone = %v, want %v", rec.spinDone, wantSpinDone)
		}
	})

	t.Run("mount error carries exit code (D3)", func(t *testing.T) {
		dir := t.TempDir()
		writeProject(t, dir, "demo", "demo-alias:/opt/app", "/mnt/x", "zed")
		rec := &recorder{}
		sys := &fakeSystem{
			checkNet: func(ctx context.Context, alias string) (bool, string) { return true, "" },
			mount: func(ctx context.Context, remotePath, localMount string, poll func(int)) error {
				return &system.RunError{ExitCode: 42}
			},
		}
		eng := &Engine{ConfigDir: dir, Spin: rec.recSpin, SpinDone: rec.recSpinDone, System: sys}

		err := eng.MountProject(context.Background(), "demo")
		wantExitError(t, err, 1, "")
		wantSpinDone := []string{
			fmt.Sprintf(MountResolved, "demo-alias"),
			fmt.Sprintf(MountFailed, 42),
		}
		if !reflect.DeepEqual(rec.spinDone, wantSpinDone) {
			t.Errorf("SpinDone = %v, want %v", rec.spinDone, wantSpinDone)
		}
		wantSpin := []string{fmt.Sprintf(MountResolving, "demo-alias")}
		if !reflect.DeepEqual(rec.spin, wantSpin) {
			t.Errorf("Spin = %v, want %v", rec.spin, wantSpin)
		}
	})
}

func TestUnmountProject(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		dir := t.TempDir()
		eng := &Engine{ConfigDir: dir, System: &fakeSystem{}}
		err := eng.UnmountProject(context.Background(), "x")
		wantExitError(t, err, 1, fmt.Sprintf(ProjectNotFound, "x", dir))
	})

	t.Run("not mounted", func(t *testing.T) {
		dir := t.TempDir()
		writeProject(t, dir, "demo", "demo-alias:/opt/app", "/mnt/x", "zed")
		rec := &recorder{}
		sys := &fakeSystem{unmount: func(ctx context.Context, localMount string, out func(string, ...any)) (bool, error) {
			return true, nil
		}}
		eng := &Engine{ConfigDir: dir, Out: rec.recOut, System: sys}

		if err := eng.UnmountProject(context.Background(), "demo"); err != nil {
			t.Fatalf("UnmountProject returned error: %v", err)
		}
		wantOut := []string{MountNotMounted}
		if !reflect.DeepEqual(rec.out, wantOut) {
			t.Errorf("Out = %v, want %v", rec.out, wantOut)
		}
	})

	t.Run("success", func(t *testing.T) {
		dir := t.TempDir()
		writeProject(t, dir, "demo", "demo-alias:/opt/app", "/mnt/x", "zed")
		rec := &recorder{}
		sys := &fakeSystem{unmount: func(ctx context.Context, localMount string, out func(string, ...any)) (bool, error) {
			return false, nil
		}}
		eng := &Engine{ConfigDir: dir, Out: rec.recOut, System: sys}

		if err := eng.UnmountProject(context.Background(), "demo"); err != nil {
			t.Fatalf("UnmountProject returned error: %v", err)
		}
		wantOut := []string{MountDetached}
		if !reflect.DeepEqual(rec.out, wantOut) {
			t.Errorf("Out = %v, want %v", rec.out, wantOut)
		}
	})

	t.Run("error (D4)", func(t *testing.T) {
		dir := t.TempDir()
		writeProject(t, dir, "demo", "demo-alias:/opt/app", "/mnt/x", "zed")
		sys := &fakeSystem{unmount: func(ctx context.Context, localMount string, out func(string, ...any)) (bool, error) {
			return false, errors.New("lazy umount failed")
		}}
		eng := &Engine{ConfigDir: dir, System: sys}

		err := eng.UnmountProject(context.Background(), "demo")
		wantExitError(t, err, 1, "lazy umount failed")
	})
}

func TestRunProject(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		dir := t.TempDir()
		eng := &Engine{ConfigDir: dir, System: &fakeSystem{}}
		err := eng.RunProject(context.Background(), "x")
		wantExitError(t, err, 1, fmt.Sprintf(ProjectNotFound, "x", dir))
	})

	// not mounted -> calls Mount
	t.Run("not mounted calls mount", func(t *testing.T) {
		dir := t.TempDir()
		writeProject(t, dir, "demo", "demo-alias:/opt/app", "/mnt/x", "zed")
		rec := &recorder{}
		calls := 0
		sys := &fakeSystem{
			checkNet: func(ctx context.Context, alias string) (bool, string) { return true, "" },
			mount:    func(ctx context.Context, remotePath, localMount string, poll func(int)) error { return nil },
			isMounted: func(path string) bool {
				// first call (mount gate) false, all later true.
				calls++
				return calls > 1
			},
		}
		eng := &Engine{ConfigDir: dir, Spin: rec.recSpin, SpinDone: rec.recSpinDone, Out: rec.recOut, System: sys}

		if err := eng.RunProject(context.Background(), "demo"); err != nil {
			t.Fatalf("RunProject returned error: %v", err)
		}
		if len(sys.mountCalls) != 1 {
			t.Errorf("Mount calls = %d, want 1", len(sys.mountCalls))
		}
		wantSpinDone := []string{
			fmt.Sprintf(MountResolved, "demo-alias"),
			fmt.Sprintf(MountMounted, "demo-alias"),
			VFSReady,
		}
		if !reflect.DeepEqual(rec.spinDone, wantSpinDone) {
			t.Errorf("SpinDone = %v, want %v", rec.spinDone, wantSpinDone)
		}
		wantSpin := []string{fmt.Sprintf(MountResolving, "demo-alias")}
		if !reflect.DeepEqual(rec.spin, wantSpin) {
			t.Errorf("Spin = %v, want %v", rec.spin, wantSpin)
		}
		wantOut := []string{fmt.Sprintf(LaunchingEditor, "zed")}
		if !reflect.DeepEqual(rec.out, wantOut) {
			t.Errorf("Out = %v, want %v", rec.out, wantOut)
		}
		wantStart := [][]string{{"zed", "/mnt/x"}}
		if !reflect.DeepEqual(sys.startCalls, wantStart) {
			t.Errorf("Start calls = %v, want %v", sys.startCalls, wantStart)
		}
	})

	// VFS wait loop: briefly not mounted after mount, then ready.
	t.Run("vfs wait then ready", func(t *testing.T) {
		dir := t.TempDir()
		writeProject(t, dir, "demo", "demo-alias:/opt/app", "/mnt/x", "zed")
		rec := &recorder{}
		seq := []bool{false, false, false, true}
		sys := &fakeSystem{
			checkNet: func(ctx context.Context, alias string) (bool, string) { return true, "" },
			mount:    func(ctx context.Context, remotePath, localMount string, poll func(int)) error { return nil },
			isMounted: func(path string) bool {
				if len(seq) == 0 {
					return true
				}
				v := seq[0]
				seq = seq[1:]
				return v
			},
		}
		eng := &Engine{ConfigDir: dir, Spin: rec.recSpin, SpinDone: rec.recSpinDone, Out: rec.recOut, System: sys}

		if err := eng.RunProject(context.Background(), "demo"); err != nil {
			t.Fatalf("RunProject returned error: %v", err)
		}
		wantSpin := []string{
			fmt.Sprintf(MountResolving, "demo-alias"),
			fmt.Sprintf(VFSWait, "/mnt/x"),
			fmt.Sprintf(VFSWait, "/mnt/x"),
		}
		if !reflect.DeepEqual(rec.spin, wantSpin) {
			t.Errorf("Spin = %v, want %v", rec.spin, wantSpin)
		}
		wantSpinDone := []string{
			fmt.Sprintf(MountResolved, "demo-alias"),
			fmt.Sprintf(MountMounted, "demo-alias"),
			VFSReady,
		}
		if !reflect.DeepEqual(rec.spinDone, wantSpinDone) {
			t.Errorf("SpinDone = %v, want %v", rec.spinDone, wantSpinDone)
		}
	})

	// VFS timeout: never mounted.
	t.Run("vfs timeout", func(t *testing.T) {
		dir := t.TempDir()
		writeProject(t, dir, "demo", "demo-alias:/opt/app", "/mnt/x", "zed")
		rec := &recorder{}
		sys := &fakeSystem{
			checkNet:  func(ctx context.Context, alias string) (bool, string) { return true, "" },
			mount:     func(ctx context.Context, remotePath, localMount string, poll func(int)) error { return nil },
			isMounted: func(path string) bool { return false },
		}
		eng := &Engine{ConfigDir: dir, Spin: rec.recSpin, SpinDone: rec.recSpinDone, System: sys}

		err := eng.RunProject(context.Background(), "demo")
		wantExitError(t, err, 1, "")
		wantSpinDone := []string{
			fmt.Sprintf(MountResolved, "demo-alias"),
			fmt.Sprintf(MountMounted, "demo-alias"),
			VFSTimeout,
		}
		if !reflect.DeepEqual(rec.spinDone, wantSpinDone) {
			t.Errorf("SpinDone = %v, want %v", rec.spinDone, wantSpinDone)
		}
		// The first spin is the "Resolving <alias>" stage from MountProject; the
		// remaining 50 are VFS wait stages of the run loop.
		if len(rec.spin) != 51 {
			t.Errorf("Spin length = %d, want 51 (1 resolving + 50 VFS wait)", len(rec.spin))
		}
		for i, s := range rec.spin {
			if i == 0 && s != fmt.Sprintf(MountResolving, "demo-alias") {
				t.Errorf("Spin[0] = %q, want %q", s, fmt.Sprintf(MountResolving, "demo-alias"))
			}
			if i > 0 && s != fmt.Sprintf(VFSWait, "/mnt/x") {
				t.Errorf("Spin[%d] = %q, want %q", i, s, fmt.Sprintf(VFSWait, "/mnt/x"))
			}
		}
	})
}

func TestConnectProject(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		dir := t.TempDir()
		eng := &Engine{ConfigDir: dir, System: &fakeSystem{}}
		err := eng.ConnectProject(context.Background(), "x", "zed")
		wantExitError(t, err, 1, fmt.Sprintf(ProjectNotFound, "x", dir))
	})

	t.Run("check net fail", func(t *testing.T) {
		dir := t.TempDir()
		writeProject(t, dir, "demo", "demo-alias:/opt/app", "/mnt/x", "zed")
		rec := &recorder{}
		sys := &fakeSystem{checkNet: func(ctx context.Context, alias string) (bool, string) {
			return false, "down"
		}}
		eng := &Engine{ConfigDir: dir, Spin: rec.recSpin, SpinDone: rec.recSpinDone, System: sys}

		err := eng.ConnectProject(context.Background(), "demo", "zed")
		wantExitError(t, err, 1, "")
		wantSpinDone := []string{fmt.Sprintf(SpinFailed, "down")}
		if !reflect.DeepEqual(rec.spinDone, wantSpinDone) {
			t.Errorf("SpinDone = %v, want %v", rec.spinDone, wantSpinDone)
		}
	})

	t.Run("code remote", func(t *testing.T) {
		dir := t.TempDir()
		writeProject(t, dir, "demo", "demo-alias:/opt/app", "/mnt/x", "zed")
		rec := &recorder{}
		sys := &fakeSystem{
			checkNet: func(ctx context.Context, alias string) (bool, string) { return true, "" },
		}
		eng := &Engine{ConfigDir: dir, Spin: rec.recSpin, SpinDone: rec.recSpinDone, Out: rec.recOut, System: sys}

		if err := eng.ConnectProject(context.Background(), "demo", "code"); err != nil {
			t.Fatalf("ConnectProject returned error: %v", err)
		}
		wantSpinDone := []string{fmt.Sprintf(ConnectReachable, "demo-alias")}
		if !reflect.DeepEqual(rec.spinDone, wantSpinDone) {
			t.Errorf("SpinDone = %v, want %v", rec.spinDone, wantSpinDone)
		}
		wantOut := []string{fmt.Sprintf(CodeRemote, "demo-alias")}
		if !reflect.DeepEqual(rec.out, wantOut) {
			t.Errorf("Out = %v, want %v", rec.out, wantOut)
		}
		wantStart := [][]string{{"code", "--folder-uri", "vscode-remote://ssh-remote+demo-alias:/opt/app"}}
		if !reflect.DeepEqual(sys.startCalls, wantStart) {
			t.Errorf("Start calls = %v, want %v", sys.startCalls, wantStart)
		}
	})

	t.Run("native ssh", func(t *testing.T) {
		dir := t.TempDir()
		writeProject(t, dir, "demo", "demo-alias:/opt/app", "/mnt/x", "zed")
		rec := &recorder{}
		sys := &fakeSystem{
			checkNet: func(ctx context.Context, alias string) (bool, string) { return true, "" },
		}
		eng := &Engine{ConfigDir: dir, Spin: rec.recSpin, SpinDone: rec.recSpinDone, Out: rec.recOut, System: sys}

		if err := eng.ConnectProject(context.Background(), "demo", "zed"); err != nil {
			t.Fatalf("ConnectProject returned error: %v", err)
		}
		wantOut := []string{fmt.Sprintf(NativeSSH, "zed", "demo-alias")}
		if !reflect.DeepEqual(rec.out, wantOut) {
			t.Errorf("Out = %v, want %v", rec.out, wantOut)
		}
		wantStart := [][]string{{"zed", "ssh://demo-alias:/opt/app"}}
		if !reflect.DeepEqual(sys.startCalls, wantStart) {
			t.Errorf("Start calls = %v, want %v", sys.startCalls, wantStart)
		}
	})
}

func TestDeleteProject(t *testing.T) {
	t.Run("missing config", func(t *testing.T) {
		dir := t.TempDir()
		eng := &Engine{ConfigDir: dir, System: &fakeSystem{}}
		err := eng.DeleteProject(context.Background(), "x", true)
		wantExitError(t, err, 1, fmt.Sprintf(ProjectNotFound, "x", dir))
	})

	t.Run("mounted blocked", func(t *testing.T) {
		dir := t.TempDir()
		writeProject(t, dir, "demo", "demo-alias:/opt/app", "/mnt/x", "zed")
		sys := &fakeSystem{isMounted: func(path string) bool { return true }}
		eng := &Engine{ConfigDir: dir, System: sys}

		err := eng.DeleteProject(context.Background(), "demo", true)
		wantExitError(t, err, 1, config.ProjectMountedMessage)
	})

	t.Run("force deletes", func(t *testing.T) {
		dir := t.TempDir()
		writeProject(t, dir, "demo", "demo-alias:/opt/app", "/mnt/x", "zed")
		rec := &recorder{}
		sys := &fakeSystem{isMounted: func(path string) bool { return false }}
		eng := &Engine{ConfigDir: dir, Out: rec.recOut, System: sys}

		if err := eng.DeleteProject(context.Background(), "demo", true); err != nil {
			t.Fatalf("DeleteProject returned error: %v", err)
		}
		wantOut := []string{fmt.Sprintf(ProjectDeleted, "demo")}
		if !reflect.DeepEqual(rec.out, wantOut) {
			t.Errorf("Out = %v, want %v", rec.out, wantOut)
		}
		if _, statErr := os.Stat(filepath.Join(dir, "demo"+config.ConfigSuffix)); statErr == nil {
			t.Errorf("config file should have been deleted")
		}
	})

	t.Run("confirm true deletes", func(t *testing.T) {
		dir := t.TempDir()
		writeProject(t, dir, "demo", "demo-alias:/opt/app", "/mnt/x", "zed")
		rec := &recorder{}
		sys := &fakeSystem{isMounted: func(path string) bool { return false }}
		confirmed := false
		eng := &Engine{
			ConfigDir: dir,
			Out:       rec.recOut,
			Confirm: func(name string) (bool, error) {
				if name != "demo" {
					t.Errorf("Confirm name = %q, want demo", name)
				}
				confirmed = true
				return true, nil
			},
			System: sys,
		}

		if err := eng.DeleteProject(context.Background(), "demo", false); err != nil {
			t.Fatalf("DeleteProject returned error: %v", err)
		}
		if !confirmed {
			t.Errorf("Confirm was not invoked")
		}
		wantOut := []string{fmt.Sprintf(ProjectDeleted, "demo")}
		if !reflect.DeepEqual(rec.out, wantOut) {
			t.Errorf("Out = %v, want %v", rec.out, wantOut)
		}
		if _, statErr := os.Stat(filepath.Join(dir, "demo"+config.ConfigSuffix)); statErr == nil {
			t.Errorf("config file should have been deleted")
		}
	})

	t.Run("confirm false cancels, file remains", func(t *testing.T) {
		dir := t.TempDir()
		confPath := writeProject(t, dir, "demo", "demo-alias:/opt/app", "/mnt/x", "zed")
		rec := &recorder{}
		sys := &fakeSystem{isMounted: func(path string) bool { return false }}
		eng := &Engine{
			ConfigDir: dir,
			Out:       rec.recOut,
			Confirm:   func(name string) (bool, error) { return false, nil },
			System:    sys,
		}

		if err := eng.DeleteProject(context.Background(), "demo", false); err != nil {
			t.Fatalf("DeleteProject returned error (want nil): %v", err)
		}
		wantOut := []string{MountCancelled}
		if !reflect.DeepEqual(rec.out, wantOut) {
			t.Errorf("Out = %v, want %v", rec.out, wantOut)
		}
		if _, statErr := os.Stat(confPath); statErr != nil {
			t.Errorf("config file should remain after cancellation: %v", statErr)
		}
	})

	t.Run("confirm error propagates", func(t *testing.T) {
		dir := t.TempDir()
		writeProject(t, dir, "demo", "demo-alias:/opt/app", "/mnt/x", "zed")
		sys := &fakeSystem{isMounted: func(path string) bool { return false }}
		eng := &Engine{
			ConfigDir: dir,
			Confirm:   func(name string) (bool, error) { return false, errors.New("no input") },
			System:    sys,
		}

		err := eng.DeleteProject(context.Background(), "demo", false)
		if err == nil || err.Error() != "no input" {
			t.Fatalf("DeleteProject err = %v, want %q", err, "no input")
		}
	})
}
