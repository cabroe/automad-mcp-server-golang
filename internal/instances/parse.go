package instances

import (
	"regexp"
	"strconv"
	"strings"
)

// psFieldSep separates fields in the custom `docker ps --format` template
// below. It's the ASCII "unit separator" control character, chosen because
// it will never legitimately appear in an image name, status string, or
// port mapping, unlike a comma, tab, or space.
const psFieldSep = "\x1f"

// psFormat is the Go template passed to `docker ps --format`. Field order
// must match parsePsLine.
var psFormat = strings.Join([]string{
	"{{.ID}}",
	"{{.Image}}",
	"{{.Status}}",
	"{{.Ports}}",
	"{{.CreatedAt}}",
	`{{.Label "` + nameLabelKey + `"}}`,
	"{{.Names}}",
}, psFieldSep)

// hostPortRE extracts the host-side port of a "->80/tcp" mapping out of
// Docker's human-readable Ports string, e.g. "0.0.0.0:8080->80/tcp, :::8080->80/tcp".
var hostPortRE = regexp.MustCompile(`:(\d+)->` + containerPort + `/tcp`)

// parsePsLine parses one line of `docker ps --format psFormat` output into
// an Instance. Returns false if the line doesn't have the expected number
// of fields (e.g. a stray blank line).
func parsePsLine(line string) (Instance, bool) {
	fields := strings.Split(line, psFieldSep)
	if len(fields) != 7 {
		return Instance{}, false
	}

	inst := Instance{
		ContainerID:   fields[0],
		Image:         fields[1],
		Status:        fields[2],
		Ports:         fields[3],
		CreatedAt:     fields[4],
		Name:          fields[5],
		ContainerName: strings.TrimPrefix(fields[6], "/"),
	}
	inst.Running = strings.HasPrefix(inst.Status, "Up")

	if m := hostPortRE.FindStringSubmatch(inst.Ports); m != nil {
		if p, err := strconv.Atoi(m[1]); err == nil {
			inst.HostPort = p
		}
	}

	// The name label is only missing for containers not created by this
	// package, which shouldn't appear here since every call filters by
	// ManagedByLabel — but fall back to stripping the prefix just in case.
	if inst.Name == "" {
		inst.Name = strings.TrimPrefix(inst.ContainerName, ContainerPrefix)
	}

	return inst, true
}

// parsePsOutput parses the full (possibly multi-line) output of
// `docker ps --format psFormat`.
func parsePsOutput(output string) []Instance {
	var out []Instance
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if inst, ok := parsePsLine(line); ok {
			out = append(out, inst)
		}
	}
	return out
}
