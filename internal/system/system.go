// Package system provides the typed adapters that the workspace use-cases
// depend on for their side-effecting operations: process execution, mount-state
// checks, SSH alias resolution/reachability and key generation. The real
// implementations live behind small interfaces (Runner, MountChecker,
// NetworkChecker) so that they can be replaced by test fakes without a real
// sshfs, network or terminal. os/exec is confined to this package and to the
// concrete adapters (process.go, mount.go, network.go, keys.go); the package
// holds no stateful package globals.
package system

import (
	"context"
	"fmt"
	"os/exec"
)

// RunError is returned by Runner.Run when a command exits with a non-zero
// status. ExitCode is the command's exit code.
type RunError struct {
	ExitCode int
}

// Error implements error, reporting the non-zero exit status.
func (e *RunError) Error() string {
	return fmt.Sprintf("command exited with status %d", e.ExitCode)
}

// Proc is a started (fire-and-forget) process that has not yet been awaited.
//
// It wraps the underlying *exec.Cmd. An optional overridable wait lets tests
// inject a scripted result without launching a real process; the zero value
// (nil wait) delegates to the real cmd.Wait.
type Proc struct {
	cmd *exec.Cmd

	// wait, when non-nil, is used by Wait instead of cmd.Wait. It is a
	// test-only seam; the real processRunner.Start leaves it nil.
	wait func() error
}

// Wait blocks until the process exits and returns its error, if any. A
// successful (exit 0) run returns nil.
func (p *Proc) Wait() error {
	if p.wait != nil {
		return p.wait()
	}
	return p.cmd.Wait()
}

// Runner executes external programs with a context and separate stdout/stderr
// buffers. Run waits for completion; Start launches without waiting.
type Runner interface {
	// Run executes name with args, writing stdout and stderr into separate
	// buffers that are returned verbatim. On a non-zero exit code it returns
	// *RunError.
	Run(ctx context.Context, name string, args ...string) (stdout, stderr string, err error)
	// Start launches name with args without waiting and returns a Proc that the
	// caller may later Wait on.
	Start(ctx context.Context, name string, args ...string) (*Proc, error)
}

// MountChecker reports whether a path is a mountpoint. It must never return an
// error: failures evaluate to false (Python `mountpoint -q` parity).
type MountChecker interface {
	IsMounted(path string) bool
}

// NetworkChecker resolves an SSH alias and verifies host reachability.
type NetworkChecker interface {
	// CheckNet resolves an SSH alias via `ssh -G` and probes the resolved host
	// via `nc`. It returns (ok, msg); ok=false carries an exact diagnostic.
	CheckNet(ctx context.Context, alias string) (bool, string)
}

// System bundles the injected adapters consumed by the workspace layer. It is
// constructed with Default; each dependency is replaceable for tests.
//
// The three interfaces are embedded, so the caller directly sees Run, Start,
// IsMounted and CheckNet plus the operation methods Mount, Unmount and
// GenerateKeypair defined on *System.
type System struct {
	MountChecker   // IsMounted
	NetworkChecker // CheckNet
	Runner         // Run, Start
}

// Default returns a System wired to the REAL implementations: the process
// runner (os/exec, behind Runner) and the mountpoint -q check and ssh/nc
// network check. os/exec lives only here and in the concrete adapters.
//
// Unlike the source reference (which returns a value), Default returns a
// *System so that the returned value satisfies the workspace System interface
// (Mount/Unmount/GenerateKeypair are pointer-receiver methods).
func Default() *System {
	runner := &processRunner{}
	return &System{
		Runner:         runner,
		MountChecker:   &mountChecker{runner: runner},
		NetworkChecker: &networkChecker{runner: runner},
	}
}
