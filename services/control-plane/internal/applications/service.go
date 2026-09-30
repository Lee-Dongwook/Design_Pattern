package applications

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Design_Pattern/services/control-plane/internal/domain/run"
	"github.com/Design_Pattern/services/control-plane/internal/ports/outbound"
)

type Service struct {
	pipelines outbound.PipelineRepository
	runs      outbound.RunRepository
	now       func() time.Time
	sequence  atomic.Uint64
}

func NewService(pipelines outbound.PipelineRepository, runs outbound.RunRepository) *Service {
	return &Service{pipelines: pipelines, runs: runs, now: time.Now}
}

func (s *Service) ListPipelines(ctx context.Context) ([]Pipeline, error) {
	stored, err := s.pipelines.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]Pipeline, 0, len(stored))
	for _, value := range stored {
		result = append(result, Pipeline{ID: value.ID, Definition: value.Definition})
	}
	return result, nil
}

func (s *Service) SavePipeline(ctx context.Context, value Pipeline) (Pipeline, error) {
	if strings.TrimSpace(value.ID) == "" {
		return Pipeline{}, fmt.Errorf("pipeline ID is required")
	}
	if err := value.Definition.Validate(); err != nil {
		return Pipeline{}, err
	}
	if err := s.pipelines.Save(ctx, outbound.StoredPipeline{ID: value.ID, Definition: value.Definition}); err != nil {
		return Pipeline{}, err
	}
	return value, nil
}

func (s *Service) CreateRun(ctx context.Context, pipelineID string) (Run, error) {
	if strings.TrimSpace(pipelineID) == "" {
		return Run{}, fmt.Errorf("pipeline ID is required")
	}
	storedPipeline, found, err := s.pipelines.Get(ctx, pipelineID)
	if err != nil {
		return Run{}, err
	}
	if !found {
		return Run{}, fmt.Errorf("pipeline %q not found", pipelineID)
	}

	now := s.now().UTC()
	id := fmt.Sprintf("run-%06d", s.sequence.Add(1))
	domainRun, err := run.New(run.ID(id), storedPipeline.Definition, now)
	if err != nil {
		return Run{}, err
	}
	result := toRun(id, pipelineID, domainRun)
	if err := s.runs.Save(ctx, toStoredRun(result)); err != nil {
		return Run{}, err
	}
	return result, nil
}

func (s *Service) ListRuns(ctx context.Context) ([]Run, error) {
	stored, err := s.runs.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]Run, 0, len(stored))
	for _, value := range stored {
		result = append(result, fromStoredRun(value))
	}
	return result, nil
}
func (s *Service) GetRun(ctx context.Context, id string) (Run, bool, error) {
	stored, found, err := s.runs.Get(ctx, id)
	return fromStoredRun(stored), found, err
}

// Runner dispatch is added in the next MVP slice. These methods keep the HTTP
// inbound port stable while no task dispatcher has been configured yet.
func (s *Service) ClaimTask(context.Context, string) (TaskAssignment, bool, error) {
	return TaskAssignment{}, false, nil
}
func (s *Service) ReportTaskEvent(context.Context, string, TaskEventStatus, string) (bool, error) {
	return false, nil
}

func toRun(id, pipelineID string, value *run.Run) Run {
	result := Run{ID: id, PipelineID: pipelineID, Status: value.Status(), CreatedAt: value.CreatedAt()}
	if startedAt, ok := value.StartedAt(); ok {
		result.StartedAt = &startedAt
	}
	if endedAt, ok := value.EndedAt(); ok {
		result.EndedAt = &endedAt
	}
	return result
}

func toStoredRun(value Run) outbound.StoredRun {
	return outbound.StoredRun{ID: value.ID, PipelineID: value.PipelineID, Status: value.Status, CreatedAt: value.CreatedAt, StartedAt: value.StartedAt, EndedAt: value.EndedAt}
}
func fromStoredRun(value outbound.StoredRun) Run {
	return Run{ID: value.ID, PipelineID: value.PipelineID, Status: value.Status, CreatedAt: value.CreatedAt, StartedAt: value.StartedAt, EndedAt: value.EndedAt}
}

var _ API = (*Service)(nil)
