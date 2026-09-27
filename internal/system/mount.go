package system

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// Message constants emitted by the mount/unmount adapters. They are printed
// through the Unmount callback (see Unmount) and are exact byte-for-byte ports
// of the Python diagnostics.
const (
	// FuserKill announces the fuser cleanup phase.
	FuserKill = "Terminating locking file descriptors via fuser..."
	// Unmounting announces the umount attempt for the given mount path.
	Unmounting = "Unmounting %s..."
	// LazyUnmount announces the safe lazy-unmount fallback after a blocked
	// standard umount.
	LazyUnmount = "Standard unmount blocked. Applying safe lazy-unmount..."
)

// mountChecker is the real MountChecker: it runs `mountpoint -q <path>` via the
// injected Runner. Any error or non-zero exit yields false (never an error), and
// an empty path yields false.
type mountChecker struct {
	runner Runner
}

// IsMounted reports whether path is a mountpoint. An empty path is treated as
// not mounted. Failures evaluate to false.
func (m *mountChecker) IsMounted(path string) bool {
	if path == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, _, err := m.runner.Run(ctx, "mountpoint", "-q", path)
	return err == nil
}

// Mount mounts remotePath onto localMount via sshfs, invoking poll(iteration) on
// each wait iteration. It is a port of cmd_mount: MkdirAll(0o755), then Start
// sshfs with the exact argument vector below, then polling the parent sshfs
// process until it exits. Exit 0 (sshfs daemonizes) => nil; any other exit =>
// an error carrying the exit code (ADR-001 D3). The caller's ctx bounds the
// whole operation (the CLI passes 300s).
func (s *System) Mount(ctx context.Context, remotePath, localMount string, poll func(int)) error {
	if err := os.MkdirAll(localMount, 0o755); err != nil {
		return err
	}

	proc, err := s.Start(ctx, "sshfs", remotePath, localMount,
		"-o", "cache=yes,cache_stat_timeout=1200,cache_dir_timeout=1200,cache_link_timeout=1200",
		"-o", "compression=yes,reconnect,ServerAliveInterval=15",
		"-o", "kernel_cache,noauto_cache",
	)
	if err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() { done <- proc.Wait() }()

	for i := 0; ; i++ {
		poll(i)
		select {
		case waitErr := <-done:
			if waitErr != nil {
				return &RunError{ExitCode: commandExitCode(waitErr)}
			}
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// Unmount safely unmounts localMount: fuser cleanup, umount, lazy-umount
// fallback (ADR-001 D4). It emits progress lines through out (nil-safe). It
// returns alreadyUnmounted=true with nil error when the path is not mounted
// (not an error, exit 0). A failure of BOTH umount and umount -l returns an
// error (D4).
func (s *System) Unmount(ctx context.Context, localMount string, out func(format string, args ...any)) (alreadyUnmounted bool, err error) {
	if out == nil {
		out = func(string, ...any) {}
	}

	if !s.IsMounted(localMount) {
		return true, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	out(FuserKill)
	s.Run(ctx, "fuser", "-k", "-M", localMount)

	out(fmt.Sprintf(Unmounting, localMount))
	_, _, e1 := s.Run(ctx, "umount", localMount)
	if e1 != nil {
		out(LazyUnmount)
		_, _, e2 := s.Run(ctx, "umount", "-l", localMount)
		if e2 != nil {
			return false, e2
		}
	}
	return false, nil
}

// commandExitCode extracts the intended process exit code from a Wait error,
// understanding both *RunError and *exec.ExitError; it defaults to 1 for any
// other (non-exit) error.
func commandExitCode(err error) int {
	if err == nil {
		return 0
	}
	var runErr *RunError
	if errors.As(err, &runErr) {
		return runErr.ExitCode
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return 1
}
