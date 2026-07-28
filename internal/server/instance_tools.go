// instance_tools.go registers MCP tools to create and remotely control
// Docker containers running Automad CMS, using the official
// automad/automad Docker image. See internal/instances for the safety
// model (label-scoped management, argument-slice exec, no arbitrary shell
// access, server-managed data directories).
package server

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cabroe/automad-mcp-server-golang/internal/instances"
)

// createInstanceInput is the input schema for the create_automad_instance tool.
type createInstanceInput struct {
	Name  string `json:"name" jsonschema:"Short name for the new instance (letters, digits, '-', '_'; max 63 chars). Used to derive the Docker container name and its data directory."`
	Port  int    `json:"port,omitempty" jsonschema:"Host port to map to the container's port 80. Leave empty/0 to auto-assign a free port."`
	Image string `json:"image,omitempty" jsonschema:"Docker image to use. Defaults to automad/automad:v2."`
}

// instanceNameInput is the input schema shared by tools that only need a name.
type instanceNameInput struct {
	Name string `json:"name" jsonschema:"The instance's name, as given to create_automad_instance or shown by list_automad_instances."`
}

// setInstanceStateInput is the input schema for set_automad_instance_state.
type setInstanceStateInput struct {
	Name  string `json:"name" jsonschema:"The instance's name."`
	State string `json:"state" jsonschema:"One of 'start', 'stop', 'restart'."`
}

// removeInstanceInput is the input schema for remove_automad_instance.
type removeInstanceInput struct {
	Name       string `json:"name" jsonschema:"The instance's name."`
	DeleteData bool   `json:"delete_data,omitempty" jsonschema:"If true, also permanently delete the instance's data directory (pages, config, uploads). Defaults to false: the container is removed but its data is kept, so it can be reattached later by creating a new instance pointed at the same data."`
}

// getInstanceLogsInput is the input schema for get_automad_instance_logs.
type getInstanceLogsInput struct {
	Name string `json:"name" jsonschema:"The instance's name."`
	Tail int    `json:"tail,omitempty" jsonschema:"Number of log lines to return, from the end. Defaults to 100; maximum 5000. Use this to find the auto-generated dashboard credentials right after create_automad_instance."`
}

// runConsoleCommandInput is the input schema for run_automad_console_command.
type runConsoleCommandInput struct {
	Name    string `json:"name" jsonschema:"The instance's name."`
	Command string `json:"command" jsonschema:"One of 'cache:clear', 'cache:purge', 'user:create', 'update' — see Automad's php automad/console CLI."`
}

// listInstancesInput is the (empty) input schema for list_automad_instances.
type listInstancesInput struct{}

// RegisterInstanceTools adds all Automad Docker instance management tools to the MCP server.
func RegisterInstanceTools(s *mcp.Server, svc *instances.Service) {
	registerCreateInstance(s, svc)
	registerListInstances(s, svc)
	registerGetInstance(s, svc)
	registerSetInstanceState(s, svc)
	registerRemoveInstance(s, svc)
	registerGetInstanceLogs(s, svc)
	registerRunConsoleCommand(s, svc)
}

func registerCreateInstance(s *mcp.Server, svc *instances.Service) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "create_automad_instance",
		Description: fmt.Sprintf(`Create and start a new Automad CMS instance as a Docker container, using the official %s image.
Requires Docker to be installed and running. Data persists in a server-managed directory keyed by the
instance name. The call waits for Automad's first-run installation to finish before returning. On first
start Automad auto-generates a dashboard user; retrieve the credentials with get_automad_instance_logs.`, instances.DefaultImage),
	}, func(ctx context.Context, req *mcp.CallToolRequest, input createInstanceInput) (*mcp.CallToolResult, any, error) {
		if strings.TrimSpace(input.Name) == "" {
			return toolError("name must not be empty"), nil, nil
		}

		inst, err := svc.Create(ctx, input.Name, input.Port, input.Image)
		if err != nil {
			var notReady *instances.NotReadyError
			if errors.As(err, &notReady) {
				return toolError(fmt.Sprintf("%v\n\nCurrent instance:\n%s\nUse get_automad_instance_logs to inspect initialization or remove_automad_instance to clean up.", err, formatInstance(*notReady.Instance, true))), nil, nil
			}
			return toolError(fmt.Sprintf("Failed to create instance %q: %v", input.Name, err)), nil, nil
		}

		return toolText(formatInstance(*inst, true)), nil, nil
	})
}

func registerListInstances(s *mcp.Server, svc *instances.Service) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_automad_instances",
		Description: `List every Automad Docker instance managed by this server, with status, port, and data directory.`,
	}, func(ctx context.Context, req *mcp.CallToolRequest, input listInstancesInput) (*mcp.CallToolResult, any, error) {
		list, err := svc.List(ctx)
		if err != nil {
			return toolError(fmt.Sprintf("Failed to list instances: %v", err)), nil, nil
		}
		if len(list) == 0 {
			return toolText("No Automad instances found. Use create_automad_instance to create one."), nil, nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d instance(s):\n\n", len(list)))
		for _, inst := range list {
			sb.WriteString(formatInstance(inst, false))
			sb.WriteString("\n")
		}
		return toolText(sb.String()), nil, nil
	})
}

func registerGetInstance(s *mcp.Server, svc *instances.Service) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_automad_instance",
		Description: `Get the current status, port, image, and data directory of a single Automad instance.`,
	}, func(ctx context.Context, req *mcp.CallToolRequest, input instanceNameInput) (*mcp.CallToolResult, any, error) {
		if strings.TrimSpace(input.Name) == "" {
			return toolError("name must not be empty"), nil, nil
		}
		inst, err := svc.Get(ctx, input.Name)
		if err != nil {
			return toolError(fmt.Sprintf("Failed to get instance %q: %v", input.Name, err)), nil, nil
		}
		return toolText(formatInstance(*inst, true)), nil, nil
	})
}

func registerSetInstanceState(s *mcp.Server, svc *instances.Service) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "set_automad_instance_state",
		Description: `Start, stop, or restart an Automad instance's container.
"state" must be one of 'start', 'stop', 'restart'. Only affects instances created by this server.`,
	}, func(ctx context.Context, req *mcp.CallToolRequest, input setInstanceStateInput) (*mcp.CallToolResult, any, error) {
		if strings.TrimSpace(input.Name) == "" {
			return toolError("name must not be empty"), nil, nil
		}
		state := instances.InstanceState(strings.ToLower(strings.TrimSpace(input.State)))
		if err := svc.SetState(ctx, input.Name, state); err != nil {
			return toolError(fmt.Sprintf("Failed to set instance %q to state %q: %v", input.Name, input.State, err)), nil, nil
		}

		inst, err := svc.Get(ctx, input.Name)
		if err != nil {
			return toolText(fmt.Sprintf("Instance %q is now %q.", input.Name, state)), nil, nil
		}
		return toolText(formatInstance(*inst, true)), nil, nil
	})
}

func registerRemoveInstance(s *mcp.Server, svc *instances.Service) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "remove_automad_instance",
		Description: `Stop and remove an Automad instance's container.
By default the instance's data directory is kept; pass delete_data=true to permanently delete it too.`,
	}, func(ctx context.Context, req *mcp.CallToolRequest, input removeInstanceInput) (*mcp.CallToolResult, any, error) {
		if strings.TrimSpace(input.Name) == "" {
			return toolError("name must not be empty"), nil, nil
		}
		result, err := svc.Remove(ctx, input.Name, input.DeleteData)
		if err != nil {
			return toolError(fmt.Sprintf("Failed to remove instance %q: %v", input.Name, err)), nil, nil
		}

		var parts []string
		if result.ContainerRemoved {
			parts = append(parts, fmt.Sprintf("Instance %q container removed.", input.Name))
		}
		if result.DataDeleted {
			parts = append(parts, "Its data directory was deleted.")
		} else if result.ContainerRemoved {
			parts = append(parts, "Its data directory was kept; call remove_automad_instance again with delete_data=true to delete it.")
		}
		return toolText(strings.Join(parts, " ")), nil, nil
	})
}

func registerGetInstanceLogs(s *mcp.Server, svc *instances.Service) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "get_automad_instance_logs",
		Description: `Fetch the recent container logs of an Automad instance.
Use this right after create_automad_instance to find the auto-generated dashboard username/password.`,
	}, func(ctx context.Context, req *mcp.CallToolRequest, input getInstanceLogsInput) (*mcp.CallToolResult, any, error) {
		if strings.TrimSpace(input.Name) == "" {
			return toolError("name must not be empty"), nil, nil
		}
		if input.Tail < 0 {
			return toolError("tail must not be negative"), nil, nil
		}
		if input.Tail > instances.MaxLogTail {
			return toolError(fmt.Sprintf("tail must not exceed %d lines", instances.MaxLogTail)), nil, nil
		}
		logs, err := svc.Logs(ctx, input.Name, input.Tail)
		if err != nil {
			return toolError(fmt.Sprintf("Failed to fetch logs for instance %q: %v", input.Name, err)), nil, nil
		}
		if strings.TrimSpace(logs) == "" {
			return toolText(fmt.Sprintf("No logs yet for instance %q.", input.Name)), nil, nil
		}
		return toolText(fmt.Sprintf("Logs for %q:\n\n%s", input.Name, markdownFence(logs))), nil, nil
	})
}

func registerRunConsoleCommand(s *mcp.Server, svc *instances.Service) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "run_automad_console_command",
		Description: fmt.Sprintf(`Run one of Automad's own CLI commands (php automad/console <command>) inside a running instance.
Allowed commands: %s. This is intentionally restricted to Automad's documented commands rather than
arbitrary shell execution.`, strings.Join(instances.AllowedConsoleCommands, ", ")),
	}, func(ctx context.Context, req *mcp.CallToolRequest, input runConsoleCommandInput) (*mcp.CallToolResult, any, error) {
		if strings.TrimSpace(input.Name) == "" {
			return toolError("name must not be empty"), nil, nil
		}
		if strings.TrimSpace(input.Command) == "" {
			return toolError("command must not be empty"), nil, nil
		}

		out, err := svc.RunConsoleCommand(ctx, input.Name, input.Command)
		if err != nil {
			return toolError(fmt.Sprintf("Failed to run %q on instance %q: %v", input.Command, input.Name, err)), nil, nil
		}

		msg := fmt.Sprintf("Ran %q on instance %q.", input.Command, input.Name)
		if strings.TrimSpace(out) != "" {
			msg += "\n\n" + markdownFence(out)
		}
		return toolText(msg), nil, nil
	})
}

func markdownFence(content string) string {
	fence := "```"
	for strings.Contains(content, fence) {
		fence += "`"
	}
	return fmt.Sprintf("%s\n%s\n%s", fence, content, fence)
}

// formatInstance renders an instance summary. If detailed is true, it
// includes the data directory and dashboard URL.
func formatInstance(inst instances.Instance, detailed bool) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**%s**\n", inst.Name))
	sb.WriteString(fmt.Sprintf("Container: %s (%s)\n", inst.ContainerName, shortID(inst.ContainerID)))
	sb.WriteString(fmt.Sprintf("Status: %s\n", inst.Status))
	if inst.Ports != "" {
		sb.WriteString(fmt.Sprintf("Ports: %s\n", inst.Ports))
	}
	if url := inst.DashboardURL(); url != "" {
		sb.WriteString(fmt.Sprintf("Dashboard: %s\n", url))
	}
	if detailed {
		sb.WriteString(fmt.Sprintf("Image: %s\n", inst.Image))
		sb.WriteString(fmt.Sprintf("Data directory: %s\n", inst.DataDir))
		sb.WriteString(fmt.Sprintf("Created: %s\n", inst.CreatedAt))
	}
	return sb.String()
}

// shortID truncates a Docker container ID to its conventional short form.
func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}
