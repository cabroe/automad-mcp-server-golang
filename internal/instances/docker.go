package instances

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrDockerNotFound is returned by EnsureAvailable when the docker binary
// isn't in PATH.
var ErrDockerNotFound = errors.New("docker CLI not found in PATH; install Docker Desktop or Docker Engine")

// Runner executes docker CLI commands. All arguments are passed as a slice
// to os/exec (never through a shell), so there is no command-injection
// surface regardless of what a caller passes as an instance name, port, etc.
type Runner struct {
	bin string
}

// NewRunner creates a Runner that shells out to "docker" from PATH. This
// also means DOCKER_HOST / DOCKER_CONTEXT, if set in the server's
// environment, transparently point the server at a remote Docker daemon —
// no extra plumbing needed for "remote control" beyond what the docker CLI
// itself already supports.
func NewRunner() *Runner {
	return &Runner{bin: "docker"}
}

// run executes `docker <args...>` and returns stdout/stderr separately.
func (r *Runner) run(ctx context.Context, args ...string) (stdout, stderr string, err error) {
	cmd := exec.CommandContext(ctx, r.bin, args...)

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	runErr := cmd.Run()
	stdout = strings.TrimSpace(outBuf.String())
	stderr = strings.TrimSpace(errBuf.String())

	if runErr != nil {
		if errors.Is(runErr, exec.ErrNotFound) {
			return stdout, stderr, ErrDockerNotFound
		}
		msg := stderr
		if msg == "" {
			msg = runErr.Error()
		}
		return stdout, stderr, fmt.Errorf("docker %s: %s", strings.Join(args, " "), msg)
	}
	return stdout, stderr, nil
}

// EnsureAvailable checks that the docker CLI is installed and that the
// daemon it talks to is actually reachable, so tool handlers can fail fast
// with a clear message instead of a confusing raw CLI error.
func (r *Runner) EnsureAvailable(ctx context.Context) error {
	_, _, err := r.run(ctx, "info", "--format", "{{.ServerVersion}}")
	if err != nil {
		if errors.Is(err, ErrDockerNotFound) {
			return err
		}
		return fmt.Errorf("docker daemon not reachable (is Docker running?): %w", err)
	}
	return nil
}
