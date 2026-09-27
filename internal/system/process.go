package system

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
)

// processRunner is the real Runner bound to os/exec. It launches external
// programs with a context and separate stdout/stderr buffers.
type processRunner struct{}

// Run runs name with args under ctx, capturing stdout and stderr into separate
// buffers returned verbatim. A non-zero exit code is reported as *RunError.
func (r *processRunner) Run(ctx context.Context, name string, args ...string) (stdout, stderr string, err error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return outBuf.String(), errBuf.String(), &RunError{ExitCode: exitErr.ExitCode()}
		}
		return outBuf.String(), errBuf.String(), err
	}
	return outBuf.String(), errBuf.String(), nil
}

// Start launches name with args under ctx without waiting, discarding its
// output, and returns a Proc that the caller may Wait on.
func (r *processRunner) Start(ctx context.Context, name string, args ...string) (*Proc, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &Proc{cmd: cmd}, nil
}
