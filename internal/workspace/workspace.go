// Package workspace executes the WSM use-cases (mount/unmount/run/connect/
// delete) on top of config discovery and the injected System adapters. All
// user-visible output flows through injected callbacks so the CLI and tests
// stay isolated from stdout/stderr and stdin; the package holds no stateful
// globals.
package workspace

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cy83rt00n/workspace-manager/internal/config"
	"github.com/cy83rt00n/workspace-manager/internal/system"
)

// Message constants (formats) used by the Engine. The values are byte-for-byte
// ports of the Python diagnostics; note that the project "not found" message
// uses SINGLE quotes around the name (matching Python repr), while "Config %q
// deleted." keeps double quotes (Python uses literal double quotes there).
//
// The names are capitalized so the catalog is exported for the CLI (BRIEF-005).
const (
	// MountResolving is the spinner stage while resolving the SSH alias.
	MountResolving = "Resolving %s"
	// MountResolved is the spinner completion once the alias is resolved.
	MountResolved = "Resolved %s"
	// MountSpinning is the spinner stage while mounting via sshfs.
	MountSpinning = "Mounting %s -> %s"
	// MountMounted is the spinner completion after a successful mount.
	MountMounted = "Mounted %s"
	// MountFailed is the spinner completion for a failed mount.
	MountFailed = "Mount FAILED (code %d)"
	// ConnectProbing is the spinner stage while probing an SSH alias.
	ConnectProbing = "Probing %s"
	// ConnectReachable is the spinner completion once the alias is reachable.
	ConnectReachable = "Reachable: %s"
	// SpinFailed is the spinner completion for a failed check.
	SpinFailed = "FAILED: %s"
	// VFSWait is the spinner stage while waiting for the VFS to become ready.
	VFSWait = "VFS wait %s"
	// VFSTimeout is the spinner completion when the VFS never becomes ready.
	VFSTimeout = "VFS TIMEOUT"
	// VFSReady is the spinner completion once the VFS is ready.
	VFSReady = "VFS ready"
	// LaunchingEditor announces the editor spawn in the run use-case.
	LaunchingEditor = "Launching %s..."
	// CodeRemote announces the VS Code Remote-SSH spawn in connect.
	CodeRemote = "Launching VS Code Remote-SSH on %s..."
	// NativeSSH announces the native SSH server spawn in connect.
	NativeSSH = "Launching %s native SSH Server on %s..."
	// MountNotMounted is printed when an unmount finds no mountpoint.
	MountNotMounted = "Project directory is not mounted."
	// MountDetached is printed after a successful unmount.
	MountDetached = "Workspace detached successfully."
	// MountCancelled is printed when a delete is declined.
	MountCancelled = "Cancelled."
	// ProjectDeleted is printed after a successful delete.
	ProjectDeleted = "Config %q deleted."
	// ProjectNotFound is the "project not found" diagnostic (single quotes).
	ProjectNotFound = "Project config '%s' not found in %s"
	// ErrorLine is the CLI error template applied to ExitError.Msg (BRIEF-005).
	ErrorLine = "Error: %s"
)

// ExitError is an operational failure with an intended process exit code and a
// user-visible Message. The CLI prints "Error: <Message>" to stderr and exits
// Code (see BRIEF-005).
//
// Convention: Msg does NOT carry the "Error: " prefix; the CLI adds it from the
// ErrorLine template.
type ExitError struct {
	Code int
	Msg  string
}

// Error implements error (returns Msg).
func (e *ExitError) Error() string {
	return e.Msg
}

// ExitCode returns e.Code, for consumption by the CLI.
func (e *ExitError) ExitCode() int {
	return e.Code
}

// System is the adapter surface the Engine depends on. It is satisfied by
// system.Default() and by the test fakes.
type System interface {
	CheckNet(ctx context.Context, alias string) (bool, string)
	IsMounted(path string) bool
	Mount(ctx context.Context, remotePath, localMount string, poll func(int)) error
	Unmount(ctx context.Context, localMount string, out func(string, ...any)) (bool, error)
	Start(ctx context.Context, name string, args ...string) (*system.Proc, error)
}

// Engine executes the workspace use-cases (mount/unmount/run/connect/delete) on
// top of config discovery and the injected System adapters. All user-visible
// output flows through injected callbacks so the CLI and tests stay isolated.
type Engine struct {
	// ConfigDir is the config directory (usually config.DefaultDir()).
	ConfigDir string
	// Out prints stdout lines (Python print).
	Out func(format string, args ...any)
	// ErrOut prints stderr lines (Python sys.stderr).
	ErrOut func(format string, args ...any)
	// Spin prints a spinner stage (Python spin()).
	Spin func(stage string)
	// SpinDone prints a spinner completion (Python spin_done()).
	SpinDone func(msg string)
	// Confirm asks for delete confirmation (CLI/BRIEF-005; no stdin here).
	Confirm func(name string) (bool, error)
	// System is the adapter surface (system.Default()).
	System System
}

// out writes a stdout line through the injected callback, nil-safe.
func (e *Engine) out(format string, args ...any) {
	if e.Out != nil {
		e.Out(format, args...)
	}
}

// spin writes a spinner stage through the injected callback, nil-safe.
func (e *Engine) spin(stage string) {
	if e.Spin != nil {
		e.Spin(stage)
	}
}

// spinDone writes a spinner completion through the injected callback, nil-safe.
func (e *Engine) spinDone(msg string) {
	if e.SpinDone != nil {
		e.SpinDone(msg)
	}
}

// MountProject mounts a project's remote path onto its local mount (port of
// cmd_mount). On success it returns nil; operational failures are reported as
// *ExitError (exit code 1 per ADR-001 D3). A failed CheckNet emits the
// "FAILED: <err>" spinner line and returns *ExitError with an empty Msg (the
// diagnostic was already shown; the CLI must not print a second "Error:" line).
func (e *Engine) MountProject(ctx context.Context, name string) error {
	proj := config.FindProject(e.ConfigDir, name)
	if proj == nil {
		return &ExitError{Code: 1, Msg: fmt.Sprintf(ProjectNotFound, name, e.ConfigDir)}
	}

	alias := strings.SplitN(proj.RemotePath, ":", 2)[0]
	e.spin(fmt.Sprintf(MountResolving, alias))
	ok, errStr := e.System.CheckNet(ctx, alias)
	if !ok {
		e.spinDone(fmt.Sprintf(SpinFailed, errStr))
		return &ExitError{Code: 1, Msg: ""}
	}
	e.spinDone(fmt.Sprintf(MountResolved, alias))

	err := e.System.Mount(ctx, proj.RemotePath, proj.LocalMount, func(int) {
		e.spin(fmt.Sprintf(MountSpinning, alias, proj.LocalMount))
	})
	if err == nil {
		e.spinDone(fmt.Sprintf(MountMounted, alias))
		return nil
	}

	code := 1
	if runErr, ok := err.(*system.RunError); ok {
		code = runErr.ExitCode
	}
	e.spinDone(fmt.Sprintf(MountFailed, code))
	return &ExitError{Code: 1, Msg: ""}
}

// UnmountProject unmounts a project's local mount (port of cmd_unmount). An
// already-unmounted path is not an error (exit 0). Failure of both umount
// attempts is reported as *ExitError (ADR-001 D4).
func (e *Engine) UnmountProject(ctx context.Context, name string) error {
	proj := config.FindProject(e.ConfigDir, name)
	if proj == nil {
		return &ExitError{Code: 1, Msg: fmt.Sprintf(ProjectNotFound, name, e.ConfigDir)}
	}

	already, err := e.System.Unmount(ctx, proj.LocalMount, e.Out)
	if already {
		e.out(MountNotMounted)
		return nil
	}
	if err != nil {
		return &ExitError{Code: 1, Msg: err.Error()}
	}
	e.out(MountDetached)
	return nil
}

// RunProject mounts a project if needed, waits for its VFS to become ready, then
// launches the configured editor on the local mount (port of cmd_run). A failed
// editor start is reported as *ExitError (exit code 1) instead of a raw error
// (the Python version fell back to a traceback).
func (e *Engine) RunProject(ctx context.Context, name string) error {
	proj := config.FindProject(e.ConfigDir, name)
	if proj == nil {
		return &ExitError{Code: 1, Msg: fmt.Sprintf(ProjectNotFound, name, e.ConfigDir)}
	}

	if !e.System.IsMounted(proj.LocalMount) {
		if err := e.MountProject(ctx, name); err != nil {
			return err
		}
	}

	for i := 0; i < 50; i++ {
		if e.System.IsMounted(proj.LocalMount) {
			break
		}
		e.spin(fmt.Sprintf(VFSWait, proj.LocalMount))
		time.Sleep(100 * time.Millisecond)
	}
	if !e.System.IsMounted(proj.LocalMount) {
		e.spinDone(VFSTimeout)
		return &ExitError{Code: 1, Msg: ""}
	}
	e.spinDone(VFSReady)

	e.out(fmt.Sprintf(LaunchingEditor, proj.EditorCmd))
	if _, err := e.System.Start(ctx, proj.EditorCmd, proj.LocalMount); err != nil {
		return &ExitError{Code: 1, Msg: err.Error()}
	}
	return nil
}

// ConnectProject launches the editor's native SSH remoting (bypassing FUSE) for
// a project (port of cmd_connect). editorCmd == "code" uses VS Code Remote-SSH;
// otherwise a native SSH server URI is used. A failed CheckNet emits the
// "FAILED: <err>" spinner line and returns *ExitError with an empty Msg; a
// failed editor start is reported as *ExitError (exit code 1).
func (e *Engine) ConnectProject(ctx context.Context, name, editorCmd string) error {
	proj := config.FindProject(e.ConfigDir, name)
	if proj == nil {
		return &ExitError{Code: 1, Msg: fmt.Sprintf(ProjectNotFound, name, e.ConfigDir)}
	}

	alias := strings.SplitN(proj.RemotePath, ":", 2)[0]
	e.spin(fmt.Sprintf(ConnectProbing, alias))
	ok, errStr := e.System.CheckNet(ctx, alias)
	if !ok {
		e.spinDone(fmt.Sprintf(SpinFailed, errStr))
		return &ExitError{Code: 1, Msg: ""}
	}
	e.spinDone(fmt.Sprintf(ConnectReachable, alias))

	remoteDir := proj.RemotePath[len(alias):]
	if editorCmd == "code" {
		uri := "vscode-remote://ssh-remote+" + alias + remoteDir
		e.out(fmt.Sprintf(CodeRemote, alias))
		if _, err := e.System.Start(ctx, "code", "--folder-uri", uri); err != nil {
			return &ExitError{Code: 1, Msg: err.Error()}
		}
		return nil
	}

	uri := "ssh://" + alias + remoteDir
	e.out(fmt.Sprintf(NativeSSH, editorCmd, alias))
	if _, err := e.System.Start(ctx, editorCmd, uri); err != nil {
		return &ExitError{Code: 1, Msg: err.Error()}
	}
	return nil
}

// DeleteProject deletes a project config after safety checks and, unless force,
// an interactive confirmation (port of cmd_delete). A mounted project is
// blocked; a missing config is reported with the project-not-found message (same
// as the other use-cases).
func (e *Engine) DeleteProject(ctx context.Context, name string, force bool) error {
	proj := config.FindProject(e.ConfigDir, name)
	if proj == nil {
		return &ExitError{Code: 1, Msg: fmt.Sprintf(ProjectNotFound, name, e.ConfigDir)}
	}

	if e.System.IsMounted(proj.LocalMount) {
		return &ExitError{Code: 1, Msg: config.ProjectMountedMessage}
	}

	if !force {
		ok, err := e.Confirm(proj.Name)
		if err != nil {
			return err
		}
		if !ok {
			e.out(MountCancelled)
			return nil
		}
	}

	if err := config.DeleteConfigFile(proj.ConfPath, false); err != nil {
		return &ExitError{Code: 1, Msg: err.Error()}
	}
	e.out(fmt.Sprintf(ProjectDeleted, proj.Name))
	return nil
}
