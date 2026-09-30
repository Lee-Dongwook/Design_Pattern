// Package container executes runner tasks in short-lived Docker containers.
package container

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/Design_Pattern/workers/runner/internal/execution"
)

type DockerExecutor struct {
	run func(context.Context, string, ...string) ([]byte, error)
}

func NewDockerExecutor() DockerExecutor {
	return DockerExecutor{run: func(ctx context.Context, command string, arguments ...string) ([]byte, error) {
		return exec.CommandContext(ctx, command, arguments...).CombinedOutput()
	}}
}

func (e DockerExecutor) Execute(ctx context.Context, task execution.Task) error {
	if strings.TrimSpace(task.Image) == "" {
		return fmt.Errorf("task image is required")
	}
	if len(task.Command) == 0 {
		return fmt.Errorf("task command is required")
	}
	arguments := []string{"run", "--rm", "--network", "none", "--read-only", "--cap-drop", "ALL", "--pids-limit", "256", "--memory", "512m", "--cpus", "1", task.Image}
	arguments = append(arguments, task.Command...)
	output, err := e.run(ctx, "docker", arguments...)
	if err != nil {
		return fmt.Errorf("container task failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

var _ execution.Executor = DockerExecutor{}
