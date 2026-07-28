package instances

import (
	"fmt"
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
