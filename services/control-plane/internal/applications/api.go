// Package applications contains use cases composed from domain objects and ports.
package applications

import (
	"context"
	"time"

	"github.com/Design_Pattern/services/control-plane/internal/domain/pipeline"
	"github.com/Design_Pattern/services/control-plane/internal/domain/run"
)

type Pipeline struct {
	ID         string
	Definition pipeline.Pipeline
}

type Run struct {
	ID         string
	PipelineID string
	Status     run.Status
	CreatedAt  time.Time
	StartedAt  *time.Time
	EndedAt    *time.Time
}

type TaskAssignment struct {
	ID    string
	RunID string
	Task  pipeline.Task
}
type TaskExecution struct{ ID, TaskID, Status, RunnerID, Log string }

type TaskEventStatus string

const (
	TaskEventStarted   TaskEventStatus = "started"
	TaskEventSucceeded TaskEventStatus = "succeeded"
	TaskEventFailed    TaskEventStatus = "failed"
)

// API is the inbound port consumed by the HTTP adapter.
// Storage and worker-delivery technologies remain behind outbound adapters.
type API interface {
	ListPipelines(context.Context) ([]Pipeline, error)
	SavePipeline(context.Context, Pipeline) (Pipeline, error)
	CreateRun(context.Context, string) (Run, error)
	ListRuns(context.Context) ([]Run, error)
	GetRun(context.Context, string) (Run, bool, error)
	ListRunTasks(context.Context, string) ([]TaskExecution, bool, error)
	ClaimTask(context.Context, string) (TaskAssignment, bool, error)
	ReportTaskEvent(context.Context, string, TaskEventStatus, string) (bool, error)
}
