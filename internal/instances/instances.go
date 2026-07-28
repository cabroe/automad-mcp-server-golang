// Package instances lets MCP tools create and remotely control Docker
// containers running Automad CMS (https://automad.org), using the official
// automad/automad Docker image.
//
// Design principles, since this package can start, stop, and remove Docker
// containers on the user's machine:
//
//   - Every container this package creates is tagged with the ManagedByLabel
//     label. Every lifecycle operation (start/stop/restart/remove/logs/exec)
//     first verifies the target container carries that label before acting
//     on it, so this server can never affect a container it didn't create —
//     even if a user names an unrelated container the same thing.
//   - The Docker CLI is always invoked with an argument slice
//     (os/exec, never a shell string), so there is no command-injection
//     surface from instance names, ports, or other parameters.
//   - Console command execution (run_automad_console_command) is restricted
//     to Automad's own documented, argument-free CLI commands
//     (cache:clear, cache:purge, user:create, update) rather than exposing arbitrary
//     shell execution inside the container.
//   - Instance data directories live under a single server-managed base
//     directory (AUTOMAD_INSTANCES_DIR, default ~/.automad-mcp-server/instances)
//     rather than accepting an arbitrary host path from a tool call, so a
//     tool call can't be used to bind-mount a sensitive host directory into
//     a container.
package instances

import "time"

const (
	// DefaultImage is the official Automad v2 Docker image used unless a
	// tool call specifies a different one.
	DefaultImage = "automad/automad:v2"

	// ContainerPrefix is prepended to every instance name to form the actual
	// Docker container name, namespacing containers created by this server
	// away from unrelated ones in `docker ps`.
	ContainerPrefix = "automad-mcp-"

	// ManagedByLabel is set on every container this package creates
	// (as a ready-to-use "key=value" Docker label/filter string), and is
	// required to be present before any lifecycle operation is allowed to
	// act on a container.
	ManagedByLabel = "managed-by=automad-mcp-server"

	// nameLabelKey stores the tool-facing instance name (as opposed to the
	// prefixed container name) on the container, so it can be recovered
	// without string-parsing the container name.
	nameLabelKey = "automad-mcp-name"

	// containerPort is the port Automad listens on inside the container,
	// per the official image.
	containerPort = "80"

	// dockerAvailabilityTimeout bounds the `docker info` check that runs
	// before every tool call.
	dockerAvailabilityTimeout = 5 * time.Second

	// defaultCommandTimeout bounds most Docker CLI calls (start/stop/logs/etc).
	defaultCommandTimeout = 20 * time.Second

	// instanceReadyTimeout bounds the wait for Automad's first-run Composer
	// installation and a successful console readiness probe.
	instanceReadyTimeout = 2 * time.Minute

	// instanceReadyPollInterval controls how often readiness is checked.
	instanceReadyPollInterval = 500 * time.Millisecond

	// createTimeout is longer, since `docker run` may need to pull the
	// image on first use.
	createTimeout = 5 * time.Minute
)
