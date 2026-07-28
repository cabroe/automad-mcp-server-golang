package instances

import (
	"fmt"
	"net"
	"regexp"
)

// nameRE mirrors Docker's own container/volume naming rules closely enough
// for our purposes, while being strict about it: this string ends up in a
// container name, a label value, and a directory name, so keeping it to a
// conservative charset avoids any ambiguity there.
var nameRE = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}$`)

// ValidateName reports whether name is a safe, valid instance name.
func ValidateName(name string) error {
	if !nameRE.MatchString(name) {
		return fmt.Errorf(
			"invalid instance name %q: must start with a letter or digit and contain only letters, digits, '-' and '_' (max 63 characters)",
			name,
		)
	}
	return nil
}

// AllowedConsoleCommands are Automad v2's documented, argument-free
// `php automad/console <command>` subcommands. run_automad_console_command
// only allows these — not arbitrary shell execution — so this tool can't be
// used as a generic "run anything in the container" backdoor.
var AllowedConsoleCommands = []string{"cache:clear", "cache:purge", "user:create", "update"}

// ValidateConsoleCommand reports whether command is one of AllowedConsoleCommands.
func ValidateConsoleCommand(command string) error {
	for _, c := range AllowedConsoleCommands {
		if c == command {
			return nil
		}
	}
	return fmt.Errorf("unknown console command %q; allowed: %v", command, AllowedConsoleCommands)
}

// findFreePort asks the OS for an unused TCP port by briefly binding to
// port 0 and immediately releasing it. There's an inherent (tiny) race
// between releasing the port here and `docker run` binding it moments
// later, but that's an acceptable trade-off for a local dev tool and
// mirrors how most "give me a free port" helpers work in the absence of a
// reservation API.
func findFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("finding a free port: %w", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
