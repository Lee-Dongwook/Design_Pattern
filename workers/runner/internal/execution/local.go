package execution

import (
	"context"
	"fmt"
	"os/exec"
)

type Task struct {
	Image   string
	Command []string
}
type Executor interface {
	Execute(context.Context, Task) error
}

// LocalExecutor is for local MVP development only. It executes commands on
// the Runner host; a container adapter will enforce image isolation.
type LocalExecutor struct{}

func (LocalExecutor) Execute(ctx context.Context, task Task) error {
	if len(task.Command) == 0 {
		return fmt.Errorf("task command is required")
	}
	command := exec.CommandContext(ctx, task.Command[0], task.Command[1:]...)
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("command failed: %w: %s", err, output)
	}
	return nil
}
