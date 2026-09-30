package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Design_Pattern/workers/runner/internal/adapters/container"
	"github.com/Design_Pattern/workers/runner/internal/agent"
	"github.com/Design_Pattern/workers/runner/internal/execution"
)

func main() {
	baseURL := os.Getenv("CONTROL_PLANE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	runnerID := os.Getenv("RUNNER_ID")
	if runnerID == "" {
		runnerID = "local-runner"
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	client := agent.NewClient(baseURL, runnerID)
	var executor execution.Executor
	switch os.Getenv("RUNNER_EXECUTOR") {
	case "", "docker":
		executor = container.NewDockerExecutor()
		log.Print("using Docker task executor")
	case "local":
		executor = execution.LocalExecutor{}
		log.Print("using local task executor; do not use this mode in shared environments")
	default:
		log.Fatalf("unsupported RUNNER_EXECUTOR %q", os.Getenv("RUNNER_EXECUTOR"))
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if err := agent.RunOnce(ctx, client, executor); err != nil && ctx.Err() == nil {
			log.Printf("runner cycle failed: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
