// Package memory provides an in-process adapter for local MVP development.
// Its data is intentionally discarded when the Control Plane restarts.
package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/Design_Pattern/services/control-plane/internal/domain/pipeline"
	"github.com/Design_Pattern/services/control-plane/internal/ports/outbound"
)

type Store struct {
	mu        sync.RWMutex
	pipelines map[string]outbound.StoredPipeline
	runs      map[string]outbound.StoredRun
	tasks     map[string]outbound.StoredTask
}

func NewStore() *Store {
	return &Store{pipelines: make(map[string]outbound.StoredPipeline), runs: make(map[string]outbound.StoredRun), tasks: make(map[string]outbound.StoredTask)}
}

func (s *Store) Save(_ context.Context, value outbound.StoredPipeline) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pipelines[value.ID] = clonePipeline(value)
	return nil
}

func (s *Store) List(_ context.Context) ([]outbound.StoredPipeline, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]outbound.StoredPipeline, 0, len(s.pipelines))
	for _, value := range s.pipelines {
		result = append(result, clonePipeline(value))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func (s *Store) Get(_ context.Context, id string) (outbound.StoredPipeline, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.pipelines[id]
	return clonePipeline(value), ok, nil
}

func (s *Store) SaveRun(_ context.Context, value outbound.StoredRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs[value.ID] = cloneRun(value)
	return nil
}

func (s *Store) ListRuns(_ context.Context) ([]outbound.StoredRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]outbound.StoredRun, 0, len(s.runs))
	for _, value := range s.runs {
		result = append(result, cloneRun(value))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.Before(result[j].CreatedAt) })
	return result, nil
}

func (s *Store) GetRun(_ context.Context, id string) (outbound.StoredRun, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.runs[id]
	return cloneRun(value), ok, nil
}

// PipelineRepository and RunRepository have intentionally distinct method
// names on Store to keep the Go adapter implementation explicit.
type pipelineRepository struct{ store *Store }
type runRepository struct{ store *Store }
type taskRepository struct{ store *Store }

func Pipelines(store *Store) *pipelineRepository { return &pipelineRepository{store} }
func Runs(store *Store) *runRepository           { return &runRepository{store} }
func Tasks(store *Store) *taskRepository         { return &taskRepository{store} }
func (r *pipelineRepository) Save(ctx context.Context, value outbound.StoredPipeline) error {
	return r.store.Save(ctx, value)
}
func (r *pipelineRepository) List(ctx context.Context) ([]outbound.StoredPipeline, error) {
	return r.store.List(ctx)
}
func (r *pipelineRepository) Get(ctx context.Context, id string) (outbound.StoredPipeline, bool, error) {
	return r.store.Get(ctx, id)
}
func (r *runRepository) Save(ctx context.Context, value outbound.StoredRun) error {
	return r.store.SaveRun(ctx, value)
}
func (r *runRepository) List(ctx context.Context) ([]outbound.StoredRun, error) {
	return r.store.ListRuns(ctx)
}
func (r *runRepository) Get(ctx context.Context, id string) (outbound.StoredRun, bool, error) {
	return r.store.GetRun(ctx, id)
}
func (r *taskRepository) Enqueue(_ context.Context, values []outbound.StoredTask) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	for _, value := range values {
		r.store.tasks[value.ID] = cloneTask(value)
	}
	return nil
}
func (r *taskRepository) ClaimNextRunnable(_ context.Context, runnerID string) (outbound.StoredTask, bool, error) {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	ids := make([]string, 0, len(r.store.tasks))
	for id := range r.store.tasks {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		value := r.store.tasks[id]
		if value.Status != outbound.TaskPending || !dependenciesSucceeded(r.store.tasks, value) {
			continue
		}
		value.Status, value.RunnerID = outbound.TaskClaimed, runnerID
		r.store.tasks[id] = value
		return cloneTask(value), true, nil
	}
	return outbound.StoredTask{}, false, nil
}
func (r *taskRepository) Get(_ context.Context, id string) (outbound.StoredTask, bool, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	value, found := r.store.tasks[id]
	return cloneTask(value), found, nil
}
func (r *taskRepository) Save(_ context.Context, value outbound.StoredTask) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	r.store.tasks[value.ID] = cloneTask(value)
	return nil
}
func (r *taskRepository) ListByRun(_ context.Context, runID string) ([]outbound.StoredTask, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	result := make([]outbound.StoredTask, 0)
	for _, value := range r.store.tasks {
		if value.RunID == runID {
			result = append(result, cloneTask(value))
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func clonePipeline(value outbound.StoredPipeline) outbound.StoredPipeline {
	tasks := value.Definition.Tasks
	copied := make([]pipeline.Task, len(tasks))
	for i, task := range tasks {
		copied[i] = task
		copied[i].Command = append([]string(nil), task.Command...)
		copied[i].DependsOn = append([]pipeline.TaskID(nil), task.DependsOn...)
	}
	value.Definition.Tasks = copied
	return value
}
func cloneRun(value outbound.StoredRun) outbound.StoredRun {
	if value.StartedAt != nil {
		copied := *value.StartedAt
		value.StartedAt = &copied
	}
	if value.EndedAt != nil {
		copied := *value.EndedAt
		value.EndedAt = &copied
	}
	return value
}
func cloneTask(value outbound.StoredTask) outbound.StoredTask {
	value.Task.Command = append([]string(nil), value.Task.Command...)
	value.Task.DependsOn = append([]pipeline.TaskID(nil), value.Task.DependsOn...)
	return value
}
func dependenciesSucceeded(tasks map[string]outbound.StoredTask, value outbound.StoredTask) bool {
	for _, dependency := range value.Task.DependsOn {
		dependencyTask, found := tasks[value.RunID+":"+string(dependency)]
		if !found || dependencyTask.Status != outbound.TaskSucceeded {
			return false
		}
	}
	return true
}
