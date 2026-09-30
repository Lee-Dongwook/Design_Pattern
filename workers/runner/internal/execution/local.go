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
type Result struct{ Output string }
type Executor interface {
	Execute(context.Context, Task) (Result, error)
}

// LocalExecutor is for local MVP development only. It executes commands on
// the Runner host; a container adapter will enforce image isolation.
type LocalExecutor struct{}

func (LocalExecutor) Execute(ctx context.Context, task Task) (Result, error) {
	if len(task.Command) == 0 {
		return Result{}, fmt.Errorf("task command is required")
	}
	command := exec.CommandContext(ctx, task.Command[0], task.Command[1:]...)
	output, err := command.CombinedOutput()
	if err != nil {
		return Result{Output: string(output)}, fmt.Errorf("command failed: %w", err)
	}
	return Result{Output: string(output)}, nil
}
