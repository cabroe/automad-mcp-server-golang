package instances

import "fmt"

// NotFoundError indicates no container with the given name, managed by this
// server (see ManagedByLabel), currently exists.
type NotFoundError struct {
	Name string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("no Automad instance named %q managed by this server (use list_automad_instances to see what exists)", e.Name)
}

// AlreadyExistsError indicates an instance with the given name already exists.
type AlreadyExistsError struct {
	Name string
}

func (e *AlreadyExistsError) Error() string {
	return fmt.Sprintf("an Automad instance named %q already exists", e.Name)
}
