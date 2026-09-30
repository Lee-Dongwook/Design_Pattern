package applications

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/Design_Pattern/services/control-plane/internal/domain/run"
	"github.com/Design_Pattern/services/control-plane/internal/ports/outbound"
)

type Service struct {
	pipelines outbound.PipelineRepository
	runs      outbound.RunRepository
	tasks     outbound.TaskRepository
	now       func() time.Time
}

func NewService(pipelines outbound.PipelineRepository, runs outbound.RunRepository, tasks outbound.TaskRepository) *Service {
	return &Service{pipelines: pipelines, runs: runs, tasks: tasks, now: time.Now}
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
	id, err := newRunID(now)
	if err != nil {
		return Run{}, err
	}
	domainRun, err := run.New(run.ID(id), storedPipeline.Definition, now)
	if err != nil {
		return Run{}, err
	}
	result := toRun(id, pipelineID, domainRun)
	storedRun := toStoredRun(result)
	storedRun.Definition = storedPipeline.Definition
	if err := s.runs.Save(ctx, storedRun); err != nil {
		return Run{}, err
	}
	queued := make([]outbound.StoredTask, 0, len(storedPipeline.Definition.Tasks))
	for _, task := range storedPipeline.Definition.Tasks {
		queued = append(queued, outbound.StoredTask{ID: id + ":" + string(task.ID), RunID: id, Task: task, Status: outbound.TaskPending})
	}
	if err := s.tasks.Enqueue(ctx, queued); err != nil {
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
func (s *Service) ListRunTasks(ctx context.Context, runID string) ([]TaskExecution, bool, error) {
	if _, found, err := s.runs.Get(ctx, runID); err != nil || !found {
		return nil, found, err
	}
	tasks, err := s.tasks.ListByRun(ctx, runID)
	if err != nil {
		return nil, true, err
	}
	result := make([]TaskExecution, 0, len(tasks))
	for _, task := range tasks {
		result = append(result, TaskExecution{ID: task.ID, TaskID: string(task.Task.ID), Status: string(task.Status), RunnerID: task.RunnerID, Log: task.Log})
	}
	return result, true, nil
}

func (s *Service) ClaimTask(ctx context.Context, runnerID string) (TaskAssignment, bool, error) {
	if strings.TrimSpace(runnerID) == "" {
		return TaskAssignment{}, false, fmt.Errorf("runner ID is required")
	}
	claimed, found, err := s.tasks.ClaimNextRunnable(ctx, runnerID)
	if err != nil || !found {
		return TaskAssignment{}, found, err
	}
	return TaskAssignment{ID: claimed.ID, RunID: claimed.RunID, Task: claimed.Task}, true, nil
}
func (s *Service) ReportTaskEvent(ctx context.Context, taskID string, status TaskEventStatus, message string) (bool, error) {
	task, found, err := s.tasks.Get(ctx, taskID)
	if err != nil || !found {
		return found, err
	}
	switch status {
	case TaskEventStarted:
		if task.Status != outbound.TaskClaimed {
			return true, fmt.Errorf("task %q cannot start from %q", taskID, task.Status)
		}
		task.Status = outbound.TaskRunning
		if err := s.tasks.Save(ctx, task); err != nil {
			return true, err
		}
		return true, s.transitionRun(ctx, task.RunID, run.StatusRunning)
	case TaskEventSucceeded:
		if task.Status != outbound.TaskRunning {
			return true, fmt.Errorf("task %q cannot succeed from %q", taskID, task.Status)
		}
		task.Status = outbound.TaskSucceeded
		task.Log = appendLog(task.Log, message)
		if err := s.tasks.Save(ctx, task); err != nil {
			return true, err
		}
		tasks, err := s.tasks.ListByRun(ctx, task.RunID)
		if err != nil {
			return true, err
		}
		for _, item := range tasks {
			if item.Status != outbound.TaskSucceeded {
				return true, nil
			}
		}
		return true, s.transitionRun(ctx, task.RunID, run.StatusSucceeded)
	case TaskEventFailed:
		if task.Status != outbound.TaskRunning {
			return true, fmt.Errorf("task %q cannot fail from %q", taskID, task.Status)
		}
		task.Status = outbound.TaskFailed
		task.Log = appendLog(task.Log, message)
		if err := s.tasks.Save(ctx, task); err != nil {
			return true, err
		}
		return true, s.transitionRun(ctx, task.RunID, run.StatusFailed)
	default:
		return true, fmt.Errorf("unsupported task event status %q", status)
	}
}
func appendLog(current, message string) string {
	if strings.TrimSpace(message) == "" {
		return current
	}
	if current == "" {
		return message
	}
	return current + "\n" + message
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

func newRunID(now time.Time) (string, error) {
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("generate run ID: %w", err)
	}
	return fmt.Sprintf("run-%d-%s", now.UnixNano(), hex.EncodeToString(random)), nil
}

func (s *Service) transitionRun(ctx context.Context, id string, next run.Status) error {
	stored, found, err := s.runs.Get(ctx, id)
	if err != nil || !found {
		if err != nil {
			return err
		}
		return fmt.Errorf("run %q not found", id)
	}
	if stored.Status == next {
		return nil
	}
	if stored.Status != run.StatusPending && stored.Status != run.StatusRunning {
		return fmt.Errorf("run %q is already finished", id)
	}
	now := s.now().UTC()
	stored.Status = next
	if next == run.StatusRunning {
		stored.StartedAt = &now
	} else {
		stored.EndedAt = &now
	}
	return s.runs.Save(ctx, stored)
}
