package instances

import "strconv"

// Instance describes a single Automad Docker container managed by this server.
type Instance struct {
	// Name is the short, tool-facing instance name (without ContainerPrefix).
	Name string
	// ContainerName is the actual Docker container name (ContainerPrefix + Name).
	ContainerName string
	// ContainerID is the Docker container ID.
	ContainerID string
	// Image is the Docker image the container was created from.
	Image string
	// Status is Docker's human-readable status string, e.g. "Up 3 minutes"
	// or "Exited (0) 2 hours ago".
	Status string
	// Running reports whether the container is currently running.
	Running bool
	// Ports is Docker's human-readable port mapping, e.g. "0.0.0.0:8080->80/tcp".
	Ports string
	// HostPort is the host port mapped to the container's HTTP port, parsed
	// out of Ports when possible. Zero if it couldn't be determined (e.g.
	// the container is stopped and has no active port mapping).
	HostPort int
	// DataDir is the host directory mounted into the container's /app.
	DataDir string
	// CreatedAt is Docker's human-readable creation timestamp.
	CreatedAt string
}

// DashboardURL returns the Automad dashboard URL for the instance, or an
// empty string if the host port is unknown (e.g. the container is stopped).
func (i Instance) DashboardURL() string {
	if i.HostPort == 0 {
		return ""
	}
	return "http://localhost:" + strconv.Itoa(i.HostPort) + "/dashboard"
}
