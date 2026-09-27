package system

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Message constants emitted by the network adapter (port of check_net). They
// use single quotes around the alias, matching the Python repr() of the
// diagnostic (per the primary redefinition).
const (
	// SSHAliasNotFound reports that `ssh -G` could not resolve the alias.
	SSHAliasNotFound = "SSH alias '%s' not found in ~/.ssh/config"
	// CannotResolveHostname reports that `ssh -G` produced hostname-less output.
	CannotResolveHostname = "Cannot resolve hostname for '%s'"
	// HostUnreachable reports a failed reachability probe.
	HostUnreachable = "Host %s:%s is unreachable. Check network or VPN."
)

// networkChecker is the real NetworkChecker: it resolves an SSH alias via
// `ssh -G` and verifies reachability via `nc`.
type networkChecker struct {
	runner Runner
}

// CheckNet resolves an SSH alias via `ssh -G` and verifies reachability via
// nc. It returns (ok bool, msg string); msg is an exact diagnostic when !ok.
func (n *networkChecker) CheckNet(ctx context.Context, alias string) (bool, string) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	stdout, _, err := n.runner.Run(ctx, "ssh", "-G", alias)
	if err != nil {
		// Any launch failure, non-zero rc or timeout is treated as unresolved
		// (ADR-001 D8: the rc is now checked).
		return false, fmt.Sprintf(SSHAliasNotFound, alias)
	}

	host := ""
	port := "22"
	for _, line := range strings.Split(stdout, "\n") {
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "hostname ") {
			host = strings.TrimSpace(line[len("hostname"):])
		}
		if strings.HasPrefix(lower, "port ") {
			port = strings.TrimSpace(line[len("port"):])
		}
	}

	if host == "" {
		return false, fmt.Sprintf(CannotResolveHostname, alias)
	}

	if _, _, err := n.runner.Run(ctx, "nc", "-z", "-w", "2", host, port); err != nil {
		return false, fmt.Sprintf(HostUnreachable, host, port)
	}
	return true, ""
}
