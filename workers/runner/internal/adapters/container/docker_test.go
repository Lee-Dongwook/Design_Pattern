package container

import (
	"context"
	"reflect"
	"testing"

	"github.com/Design_Pattern/workers/runner/internal/execution"
)

func TestDockerExecutorUsesRestrictedContainerOptions(t *testing.T) {
	var command string
	var arguments []string
	executor := DockerExecutor{run: func(_ context.Context, gotCommand string, gotArguments ...string) ([]byte, error) {
		command = gotCommand
		arguments = gotArguments
		return nil, nil
	}}
	if err := executor.Execute(context.Background(), execution.Task{Image: "alpine:3.20", Command: []string{"echo", "ok"}}); err != nil {
		t.Fatal(err)
	}
	want := []string{"run", "--rm", "--network", "none", "--read-only", "--cap-drop", "ALL", "--pids-limit", "256", "--memory", "512m", "--cpus", "1", "alpine:3.20", "echo", "ok"}
	if command != "docker" || !reflect.DeepEqual(arguments, want) {
		t.Fatalf("unexpected docker invocation: %s %v", command, arguments)
	}
}
