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
	Status     run.Status
	CreatedAt  time.Time
	StartedAt  *time.Time
	EndedAt    *time.Time
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
