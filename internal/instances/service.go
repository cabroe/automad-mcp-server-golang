package instances

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Service is the main entry point for creating and controlling Automad
// Docker instances. It wraps a Runner (the docker CLI) with the naming,
// labeling, and safety-check conventions described in the instances.go
// package doc comment.
type Service struct {
	runner  *Runner
	baseDir string
	image   string
}

// NewService creates a Service using default settings, configurable via:
//   - AUTOMAD_INSTANCES_DIR: base directory for instance data
//     (default: ~/.automad-mcp-server/instances)
//   - AUTOMAD_DOCKER_IMAGE: default image for new instances
//     (default: automad/automad:v2)
func NewService() *Service {
	baseDir := os.Getenv("AUTOMAD_INSTANCES_DIR")
	if baseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		baseDir = filepath.Join(home, ".automad-mcp-server", "instances")
	}

	image := os.Getenv("AUTOMAD_DOCKER_IMAGE")
	if image == "" {
		image = DefaultImage
	}

	return &Service{
		runner:  NewRunner(),
		baseDir: baseDir,
		image:   image,
	}
}

// BaseDir returns the directory under which every instance's data lives
// (each instance gets its own subdirectory named after it).
func (s *Service) BaseDir() string { return s.baseDir }

// DefaultImageTag returns the Docker image used for new instances unless a
// tool call overrides it.
func (s *Service) DefaultImageTag() string { return s.image }

// EnsureAvailable checks that Docker is installed and its daemon is reachable.
func (s *Service) EnsureAvailable(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, dockerAvailabilityTimeout)
	defer cancel()
	return s.runner.EnsureAvailable(ctx)
}

func (s *Service) dataDir(name string) string {
	return filepath.Join(s.baseDir, name)
}

func (s *Service) containerName(name string) string {
	return ContainerPrefix + name
}

func (s *Service) getUnchecked(ctx context.Context, name string) (*Instance, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	runCtx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	args := []string{
		"ps", "-a",
		"--filter", "label=" + ManagedByLabel,
		"--filter", "name=^/" + s.containerName(name) + "$",
		"--format", psFormat,
	}
	out, _, err := s.runner.run(runCtx, args...)
	if err != nil {
		return nil, fmt.Errorf("looking up instance %q: %w", name, err)
	}
	found := parsePsOutput(out)
	if len(found) == 0 {
		return nil, &NotFoundError{Name: name}
	}
	inst := found[0]
	inst.DataDir = s.dataDir(inst.Name)
	return &inst, nil
}

// Create starts a new Automad Docker instance. port, if zero, is
// auto-assigned to a free host port. image, if empty, defaults to the
// service's DefaultImageTag.
func (s *Service) Create(ctx context.Context, name string, port int, image string) (*Instance, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	if port < 0 || port > 65535 {
		return nil, fmt.Errorf("invalid port %d: must be between 1 and 65535, or 0 for automatic assignment", port)
	}
	if err := s.EnsureAvailable(ctx); err != nil {
		return nil, err
	}

	if _, err := s.getUnchecked(ctx, name); err == nil {
		return nil, &AlreadyExistsError{Name: name}
	} else {
		var notFound *NotFoundError
		if !errors.As(err, &notFound) {
			return nil, fmt.Errorf("checking whether instance %q exists: %w", name, err)
		}
	}
	if err := s.ensureContainerNameAvailable(ctx, name); err != nil {
		return nil, err
	}

	if port == 0 {
		p, err := findFreePort()
		if err != nil {
			return nil, err
		}
		port = p
	}
	if image == "" {
		image = s.image
	}
	if image != DefaultImage && image != s.image {
		return nil, fmt.Errorf("unsupported image %q: instance readiness is guaranteed only for configured image %q", image, s.image)
	}

	dir := s.dataDir(name)
	_, statErr := os.Stat(dir)
	dirExisted := statErr == nil
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return nil, fmt.Errorf("checking data directory %s: %w", dir, statErr)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating data directory %s: %w", dir, err)
	}

	args := []string{
		"run", "-d",
		"--name", s.containerName(name),
		"--label", ManagedByLabel,
		"--label", nameLabelKey + "=" + name,
		"-p", fmt.Sprintf("127.0.0.1:%d:%s", port, containerPort),
		"-v", dir + ":/app",
		image,
	}

	runCtx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()
	if _, _, err := s.runner.run(runCtx, args...); err != nil {
		// Clean up the directory we just created if the container failed to
		// start, so a retry with the same name doesn't leave orphaned state.
		if !dirExisted {
			_ = os.Remove(dir)
		}
		return nil, fmt.Errorf("creating instance %q: %w", name, err)
	}

	if err := s.waitReady(ctx, name); err != nil {
		inst, getErr := s.getUnchecked(ctx, name)
		if getErr != nil {
			return nil, err
		}
		return nil, &NotReadyError{Instance: inst, Cause: err}
	}
	return s.getUnchecked(ctx, name)
}

// ensureContainerNameAvailable detects collisions with containers not managed
// by this server before docker run returns a lower-level name-conflict error.
func (s *Service) ensureContainerNameAvailable(ctx context.Context, name string) error {
	runCtx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()
	out, _, err := s.runner.run(runCtx, "ps", "-a", "--filter", "name=^/"+s.containerName(name)+"$", "--format", "{{.ID}}")
	if err != nil {
		return fmt.Errorf("checking Docker container name for instance %q: %w", name, err)
	}
	if strings.TrimSpace(out) != "" {
		return fmt.Errorf("Docker container name %q is already used by a container not managed as instance %q", s.containerName(name), name)
	}
	return nil
}

// waitReady waits until the first-run image initialization has created the
// Automad console. A container being "running" only means its entrypoint is
// active; Composer may still be installing Automad into the mounted /app.
func (s *Service) waitReady(ctx context.Context, name string) error {
	readyCtx, cancel := context.WithTimeout(ctx, instanceReadyTimeout)
	defer cancel()

	ticker := time.NewTicker(instanceReadyPollInterval)
	defer ticker.Stop()

	var lastErr error
	for {
		checkCtx, checkCancel := context.WithTimeout(readyCtx, defaultCommandTimeout)
		_, _, err := s.runner.run(checkCtx, "exec", s.containerName(name), "php", "automad/console", "log:path")
		checkCancel()
		if err == nil {
			return nil
		}
		lastErr = err

		inst, getErr := s.getUnchecked(readyCtx, name)
		if getErr != nil {
			return fmt.Errorf("checking readiness of instance %q: %w", name, getErr)
		}
		if !inst.Running {
			return fmt.Errorf("instance %q stopped before becoming ready (status: %s): %w", name, inst.Status, lastErr)
		}

		select {
		case <-readyCtx.Done():
			return fmt.Errorf("instance %q started but Automad did not become ready within %s; last probe error: %v: %w", name, instanceReadyTimeout, lastErr, readyCtx.Err())
		case <-ticker.C:
		}
	}
}

// List returns every Automad instance managed by this server.
func (s *Service) List(ctx context.Context) ([]Instance, error) {
	if err := s.EnsureAvailable(ctx); err != nil {
		return nil, err
	}

	runCtx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	args := []string{"ps", "-a", "--filter", "label=" + ManagedByLabel, "--format", psFormat}
	out, _, err := s.runner.run(runCtx, args...)
	if err != nil {
		return nil, fmt.Errorf("listing instances: %w", err)
	}

	instances := parsePsOutput(out)
	for i := range instances {
		instances[i].DataDir = s.dataDir(instances[i].Name)
	}
	return instances, nil
}

// Get returns a single instance by name, or a *NotFoundError if no such
// instance (managed by this server) exists.
func (s *Service) Get(ctx context.Context, name string) (*Instance, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	if err := s.EnsureAvailable(ctx); err != nil {
		return nil, err
	}

	return s.getUnchecked(ctx, name)
}

// InstanceState is the set of lifecycle actions SetState accepts.
type InstanceState string

const (
	StateStart   InstanceState = "start"
	StateStop    InstanceState = "stop"
	StateRestart InstanceState = "restart"
)

// ValidStates lists every InstanceState SetState accepts, for building
// tool descriptions/errors.
var ValidStates = []InstanceState{StateStart, StateStop, StateRestart}

func isValidState(state InstanceState) bool {
	for _, s := range ValidStates {
		if s == state {
			return true
		}
	}
	return false
}

// SetState starts, stops, or restarts an existing instance.
func (s *Service) SetState(ctx context.Context, name string, state InstanceState) error {
	if !isValidState(state) {
		return fmt.Errorf("invalid state %q; must be one of %v", state, ValidStates)
	}
	inst, err := s.Get(ctx, name)
	if err != nil {
		return err
	}
	if state == StateStop && !inst.Running {
		return nil
	}
	if state == StateStart && inst.Running {
		return nil
	}

	runCtx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	if _, _, err := s.runner.run(runCtx, string(state), s.containerName(name)); err != nil {
		return fmt.Errorf("%s instance %q: %w", state, name, err)
	}
	if state == StateStart || state == StateRestart {
		if err := s.waitReady(ctx, name); err != nil {
			return err
		}
	}
	return nil
}

// RemoveResult describes what Remove actually changed.
type RemoveResult struct {
	ContainerRemoved bool
	DataDeleted      bool
}

// Remove stops (if needed) and deletes an instance's container. If
// deleteData is true, its data directory is also permanently deleted.
func (s *Service) Remove(ctx context.Context, name string, deleteData bool) (RemoveResult, error) {
	var result RemoveResult
	if err := ValidateName(name); err != nil {
		return result, err
	}
	_, err := s.Get(ctx, name)
	if err != nil {
		var notFound *NotFoundError
		if deleteData && errors.As(err, &notFound) {
			if _, statErr := os.Stat(s.dataDir(name)); errors.Is(statErr, os.ErrNotExist) {
				return result, fmt.Errorf("no retained data found for instance %q", name)
			} else if statErr != nil {
				return result, fmt.Errorf("checking retained data for instance %q: %w", name, statErr)
			}
			if err := os.RemoveAll(s.dataDir(name)); err != nil {
				return result, err
			}
			result.DataDeleted = true
			return result, nil
		}
		return result, err
	}

	runCtx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	if _, _, err := s.runner.run(runCtx, "rm", "-f", s.containerName(name)); err != nil {
		return result, fmt.Errorf("removing instance %q: %w", name, err)
	}
	result.ContainerRemoved = true

	if deleteData {
		if err := os.RemoveAll(s.dataDir(name)); err != nil {
			return result, fmt.Errorf("instance %q removed, but deleting its data directory failed: %w", name, err)
		}
		result.DataDeleted = true
	}
	return result, nil
}

// MaxLogTail is the maximum number of log lines returned by Logs.
const MaxLogTail = maxLogTail

// Logs returns the last `tail` lines of an instance's container logs
// (stdout and stderr combined, stderr labeled separately if present). This
// is also how to retrieve the dashboard credentials Automad auto-generates
// on first start.
func (s *Service) Logs(ctx context.Context, name string, tail int) (string, error) {
	if _, err := s.Get(ctx, name); err != nil {
		return "", err
	}
	if tail < 0 {
		return "", fmt.Errorf("tail must not be negative")
	}
	if tail == 0 {
		tail = 100
	}
	if tail > MaxLogTail {
		return "", fmt.Errorf("tail must not exceed %d lines", MaxLogTail)
	}

	runCtx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	stdout, stderr, err := s.runner.run(runCtx, "logs", "--tail", strconv.Itoa(tail), s.containerName(name))
	if err != nil {
		return "", fmt.Errorf("fetching logs for instance %q: %w", name, err)
	}

	if stderr == "" {
		return stdout, nil
	}
	return strings.TrimSpace(stdout + "\n" + stderr), nil
}

// RunConsoleCommand runs one of Automad's own `php automad/console <command>`
// CLI commands inside a running instance. command must be one of
// AllowedConsoleCommands.
func (s *Service) RunConsoleCommand(ctx context.Context, name, command string) (string, error) {
	if err := ValidateConsoleCommand(command); err != nil {
		return "", err
	}

	inst, err := s.Get(ctx, name)
	if err != nil {
		return "", err
	}
	if !inst.Running {
		return "", fmt.Errorf("instance %q is not running (status: %s); start it first", name, inst.Status)
	}
	// Execute the requested command directly. Readiness is established after
	// create/start/restart; any later failure should report this command's own error.
	runCtx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	stdout, stderr, err := s.runner.run(runCtx, "exec", s.containerName(name), "php", "automad/console", command)
	if err != nil {
		return "", fmt.Errorf("running console command %q on instance %q: %w", command, name, err)
	}

	if stderr == "" {
		return stdout, nil
	}
	return strings.TrimSpace(stdout + "\n" + stderr), nil
}
