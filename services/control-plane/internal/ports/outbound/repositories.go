// Package outbound defines infrastructure-facing ports used by application use cases.
package outbound

import (
	"context"
	"time"

	"github.com/Design_Pattern/services/control-plane/internal/domain/pipeline"
	"github.com/Design_Pattern/services/control-plane/internal/domain/run"
)

type StoredPipeline struct {
	ID         string
	Definition pipeline.Pipeline
}

type StoredRun struct {
	ID         string
	PipelineID string
	Definition pipeline.Pipeline
	Status     run.Status
	CreatedAt  time.Time
	StartedAt  *time.Time
	EndedAt    *time.Time
}

type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskClaimed   TaskStatus = "claimed"
	TaskRunning   TaskStatus = "running"
	TaskSucceeded TaskStatus = "succeeded"
	TaskFailed    TaskStatus = "failed"
	TaskCanceled  TaskStatus = "canceled"
)

type StoredTask struct {
	ID       string
	RunID    string
	Task     pipeline.Task
	Status   TaskStatus
	RunnerID string
	Log      string
}

type PipelineRepository interface {
	Save(context.Context, StoredPipeline) error
	List(context.Context) ([]StoredPipeline, error)
	Get(context.Context, string) (StoredPipeline, bool, error)
}

type RunRepository interface {
	Save(context.Context, StoredRun) error
	List(context.Context) ([]StoredRun, error)
	Get(context.Context, string) (StoredRun, bool, error)
}
type TaskRepository interface {
	Enqueue(context.Context, []StoredTask) error
	ClaimNextRunnable(context.Context, string) (StoredTask, bool, error)
	Get(context.Context, string) (StoredTask, bool, error)
	Save(context.Context, StoredTask) error
	ListByRun(context.Context, string) ([]StoredTask, error)
}
