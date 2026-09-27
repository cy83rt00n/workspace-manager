// Package smoketest contains safe, real smoke tests that exercise the
// production system.Default() adapters and the workspace.Engine use-cases
// against real external binaries (mountpoint, ssh, nc) without ever performing
// a real mount, unmount of a live mount, delete or editor launch. The tests
// deliberately are not fakes: they run the real adapters, but the scenarios are
// designed so that sshfs never starts and no filesystem path outside
// t.TempDir() is touched.
package smoketest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cy83rt00n/workspace-manager/internal/config"
	"github.com/cy83rt00n/workspace-manager/internal/system"
	"github.com/cy83rt00n/workspace-manager/internal/workspace"
)

// writeSmokeConfig writes a single project config file
// <dir>/<name>.config.toml with the three recognised keys and returns its
// absolute path. The formatting follows the CLI's writeConfig helper
// (remote_path/local_mount/editor_cmd, one per line), and the file is written
// with mode 0o644.
func writeSmokeConfig(t *testing.T, dir, name, remote, local, editor string) string {
	t.Helper()
	content := fmt.Sprintf("remote_path = %q\nlocal_mount = %q\neditor_cmd = %q\n", remote, local, editor)
	path := filepath.Join(dir, name+config.ConfigSuffix)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing smoke config: %v", err)
	}
	return path
}

// containsLine reports whether lines contains an element exactly equal to want
// (exact string equality, not a substring match).
func containsLine(lines []string, want string) bool {
	for _, line := range lines {
		if line == want {
			return true
		}
	}
	return false
}

// newCaptureEngine builds a workspace.Engine wired to system.Default() (the real
// adapters) with injected output callbacks: Out and ErrOut append the
// normalised line (fmt.Sprintf of the format+args) to the provided slices, and
// Spin/SpinDone use the provided callbacks. The System is the real adapter
// surface, so subprocesses invoked by the use-case are real; the Engine merely
// runs the use-case against that System.
func newCaptureEngine(cfgDir string, outLines, errLines *[]string, spin func(string), spinDone func(string)) *workspace.Engine {
	return &workspace.Engine{
		ConfigDir: cfgDir,
		System:    system.Default(),
		Out: func(format string, args ...any) {
			*outLines = append(*outLines, fmt.Sprintf(format, args...))
		},
		ErrOut: func(format string, args ...any) {
			*errLines = append(*errLines, fmt.Sprintf(format, args...))
		},
		Spin:     spin,
		SpinDone: spinDone,
	}
}

// TestSmokeUnmountNotMounted is a real smoke test that un-mounts a project
// whose local_mount path does not exist. The production mountpoint -q check is
// used (via system.Default), so a project that is not mounted is reported as
// "already unmounted": the use-case returns nil, emits
// workspace.MountNotMounted exactly once, and does NOT create the local mount
// directory.
func TestSmokeUnmountNotMounted(t *testing.T) {
	cfgDir := t.TempDir()
	mountPath := filepath.Join(cfgDir, "mountpoint")

	writeSmokeConfig(t, cfgDir, "wsmsmoke", "wsmsmoke:/srv/x", mountPath, "true")

	var outLines, errLines []string
	noop := func(string) {}
	engine := newCaptureEngine(cfgDir, &outLines, &errLines, noop, noop)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := engine.UnmountProject(ctx, "wsmsmoke")
	if err != nil {
		t.Fatalf("UnmountProject error = %v, want nil", err)
	}

	if !containsLine(outLines, workspace.MountNotMounted) {
		t.Errorf("Out lines %q do not contain exactly %q", outLines, workspace.MountNotMounted)
	}

	if _, statErr := os.Stat(mountPath); !os.IsNotExist(statErr) {
		t.Errorf("local_mount %q should not have been created by unmount, stat error = %v", mountPath, statErr)
	}
}

// TestSmokeNetworkCheckUnreachable is a real smoke test that calls
// system.Default().CheckNet directly with a deterministically-unresolvable
// alias. It verifies that CheckNet returns ok=false and one of the allowed
// unreachable diagnostics. The test deliberately does not build an Engine, does
// not create a mount path, and does not touch the host ~/.ssh: the alias
// resolves unpredictably across environments (an unresolvable hostname, an
// unresolved SSH alias, or a closed 127.0.0.1/port-22 probe), so any of the
// three allowed diagnostics is accepted.
func TestSmokeNetworkCheckUnreachable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	alias := "wsmsmoke-missing.invalid"
	ok, msg := system.Default().CheckNet(ctx, alias)

	if ok {
		t.Fatalf("CheckNet ok = true for unresolvable alias %q, want false; msg = %q", alias, msg)
	}
	if !isAllowedUnreachableDiagnostic(alias, msg) {
		t.Errorf("CheckNet msg = %q, want one of the allowed unreachable diagnostics for alias %q", msg, alias)
	}
}

// allowedUnreachableDiagnostics returns the exact set of diagnostics that
// CheckNet may return for a deterministically-unresolvable alias, depending on
// the environment (SSH config resolution and DNS behaviour). Each is built from
// the exported system message constants.
func allowedUnreachableDiagnostics(alias string) []string {
	return []string{
		fmt.Sprintf(system.SSHAliasNotFound, alias),
		fmt.Sprintf(system.CannotResolveHostname, alias),
		fmt.Sprintf(system.HostUnreachable, alias, "22"),
	}
}

// isAllowedUnreachableDiagnostic reports whether msg exactly matches one of the
// diagnostics produced by CheckNet for an unreachable alias.
func isAllowedUnreachableDiagnostic(alias, msg string) bool {
	for _, want := range allowedUnreachableDiagnostics(alias) {
		if msg == want {
			return true
		}
	}
	return false
}
