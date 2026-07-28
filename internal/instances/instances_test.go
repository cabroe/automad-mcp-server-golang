package instances_test

import (
	"strings"
	"testing"

	"github.com/cabroe/automad-mcp-server/internal/instances"
)

func TestValidateName_Valid(t *testing.T) {
	for _, name := range []string{"mysite", "my-site", "my_site", "a", "Site123", "x123456789"} {
		if err := instances.ValidateName(name); err != nil {
			t.Errorf("ValidateName(%q) = %v, want nil", name, err)
		}
	}
}

func TestValidateName_Invalid(t *testing.T) {
	for _, name := range []string{
		"",
		"-starts-with-dash",
		"_starts-with-underscore",
		"has space",
		"has/slash",
		"has.dot",
		"has;semicolon",
		"$(rm -rf /)",
		"../../etc",
		strings.Repeat("a", 64), // too long
	} {
		if err := instances.ValidateName(name); err == nil {
			t.Errorf("ValidateName(%q) = nil, want an error", name)
		}
	}
}

func TestValidateConsoleCommand_Allowed(t *testing.T) {
	for _, cmd := range instances.AllowedConsoleCommands {
		if err := instances.ValidateConsoleCommand(cmd); err != nil {
			t.Errorf("ValidateConsoleCommand(%q) = %v, want nil", cmd, err)
		}
	}
}

func TestValidateConsoleCommand_Rejected(t *testing.T) {
	for _, cmd := range []string{"", "rm -rf /", "migrate", "shell", "clearcache; purge"} {
		if err := instances.ValidateConsoleCommand(cmd); err == nil {
			t.Errorf("ValidateConsoleCommand(%q) = nil, want an error", cmd)
		}
	}
}

func TestInstance_DashboardURL(t *testing.T) {
	withPort := instances.Instance{HostPort: 8080}
	if got, want := withPort.DashboardURL(), "http://localhost:8080/dashboard"; got != want {
		t.Errorf("DashboardURL() = %q, want %q", got, want)
	}

	withoutPort := instances.Instance{}
	if got := withoutPort.DashboardURL(); got != "" {
		t.Errorf("DashboardURL() = %q, want empty string when HostPort is 0", got)
	}
}

func TestNotFoundError_MentionsName(t *testing.T) {
	err := &instances.NotFoundError{Name: "mysite"}
	if !strings.Contains(err.Error(), "mysite") {
		t.Errorf("error message %q does not mention the instance name", err.Error())
	}
}

func TestAlreadyExistsError_MentionsName(t *testing.T) {
	err := &instances.AlreadyExistsError{Name: "mysite"}
	if !strings.Contains(err.Error(), "mysite") {
		t.Errorf("error message %q does not mention the instance name", err.Error())
	}
}
