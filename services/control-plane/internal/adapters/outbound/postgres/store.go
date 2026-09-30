// Package postgres provides the production persistence adapter for Control Plane data.
package postgres

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/Design_Pattern/services/control-plane/internal/ports/outbound"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schema string

type Store struct{ pool *pgxpool.Pool }

func NewStore(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	if _, err := pool.Exec(ctx, schema); err != nil {
		pool.Close()
		return nil, fmt.Errorf("apply PostgreSQL schema: %w", err)
	}
	return &Store{pool: pool}, nil
}
func (s *Store) Close() { s.pool.Close() }

type pipelineRepository struct{ store *Store }
type runRepository struct{ store *Store }
type taskRepository struct{ store *Store }

func Pipelines(store *Store) *pipelineRepository { return &pipelineRepository{store} }
func Runs(store *Store) *runRepository           { return &runRepository{store} }
func Tasks(store *Store) *taskRepository         { return &taskRepository{store} }

func (r *pipelineRepository) Save(ctx context.Context, value outbound.StoredPipeline) error {
	tasks, err := json.Marshal(value.Definition.Tasks)
	if err != nil {
		return err
	}
	_, err = r.store.pool.Exec(ctx, `INSERT INTO pipelines (id, name, tasks) VALUES ($1, $2, $3) ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, tasks = EXCLUDED.tasks, updated_at = NOW()`, value.ID, value.Definition.Name, tasks)
	return err
}
func (r *pipelineRepository) List(ctx context.Context) ([]outbound.StoredPipeline, error) {
	rows, err := r.store.pool.Query(ctx, `SELECT id, name, tasks FROM pipelines ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]outbound.StoredPipeline, 0)
	for rows.Next() {
		value, err := scanPipeline(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}
func (r *pipelineRepository) Get(ctx context.Context, id string) (outbound.StoredPipeline, bool, error) {
	value, err := scanPipeline(r.store.pool.QueryRow(ctx, `SELECT id, name, tasks FROM pipelines WHERE id = $1`, id))
	if err == pgx.ErrNoRows {
		return outbound.StoredPipeline{}, false, nil
	}
	return value, err == nil, err
}

func (r *runRepository) Save(ctx context.Context, value outbound.StoredRun) error {
	definition, err := json.Marshal(value.Definition)
	if err != nil {
		return err
	}
	_, err = r.store.pool.Exec(ctx, `INSERT INTO runs (id, pipeline_id, definition, status, created_at, started_at, ended_at) VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, started_at = EXCLUDED.started_at, ended_at = EXCLUDED.ended_at`, value.ID, value.PipelineID, definition, value.Status, value.CreatedAt, value.StartedAt, value.EndedAt)
	return err
}
func (r *runRepository) List(ctx context.Context) ([]outbound.StoredRun, error) {
	rows, err := r.store.pool.Query(ctx, `SELECT id, pipeline_id, definition, status, created_at, started_at, ended_at FROM runs ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]outbound.StoredRun, 0)
	for rows.Next() {
		value, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}
func (r *runRepository) Get(ctx context.Context, id string) (outbound.StoredRun, bool, error) {
	value, err := scanRun(r.store.pool.QueryRow(ctx, `SELECT id, pipeline_id, definition, status, created_at, started_at, ended_at FROM runs WHERE id = $1`, id))
	if err == pgx.ErrNoRows {
		return outbound.StoredRun{}, false, nil
	}
	return value, err == nil, err
}

func (r *taskRepository) Enqueue(ctx context.Context, values []outbound.StoredTask) error {
	batch := &pgx.Batch{}
	for _, value := range values {
		task, err := json.Marshal(value.Task)
		if err != nil {
			return err
		}
		batch.Queue(`INSERT INTO tasks (id, run_id, task, status, runner_id) VALUES ($1,$2,$3,$4,$5)`, value.ID, value.RunID, task, value.Status, value.RunnerID)
	}
	results := r.store.pool.SendBatch(ctx, batch)
	defer results.Close()
	for range values {
		if _, err := results.Exec(); err != nil {
			return err
		}
	}
	return nil
}
func (r *taskRepository) ClaimNextRunnable(ctx context.Context, runnerID string) (outbound.StoredTask, bool, error) {
	row := r.store.pool.QueryRow(ctx, `WITH candidate AS (SELECT t.id FROM tasks t WHERE t.status = 'pending' AND NOT EXISTS (SELECT 1 FROM jsonb_array_elements_text(COALESCE(t.task->'DependsOn', '[]'::jsonb)) AS dependency(task_id) JOIN tasks prerequisite ON prerequisite.run_id = t.run_id AND prerequisite.task->>'ID' = dependency.task_id WHERE prerequisite.status <> 'succeeded') ORDER BY t.created_at FOR UPDATE SKIP LOCKED LIMIT 1) UPDATE tasks t SET status = 'claimed', runner_id = $1 FROM candidate WHERE t.id = candidate.id RETURNING t.id, t.run_id, t.task, t.status, t.runner_id`, runnerID)
	value, err := scanTask(row)
	if err == pgx.ErrNoRows {
		return outbound.StoredTask{}, false, nil
	}
	return value, err == nil, err
}
func (r *taskRepository) Get(ctx context.Context, id string) (outbound.StoredTask, bool, error) {
	value, err := scanTask(r.store.pool.QueryRow(ctx, `SELECT id, run_id, task, status, runner_id FROM tasks WHERE id = $1`, id))
	if err == pgx.ErrNoRows {
		return outbound.StoredTask{}, false, nil
	}
	return value, err == nil, err
}
func (r *taskRepository) Save(ctx context.Context, value outbound.StoredTask) error {
	task, err := json.Marshal(value.Task)
	if err != nil {
		return err
	}
	_, err = r.store.pool.Exec(ctx, `UPDATE tasks SET task = $2, status = $3, runner_id = $4 WHERE id = $1`, value.ID, task, value.Status, value.RunnerID)
	return err
}
func (r *taskRepository) ListByRun(ctx context.Context, runID string) ([]outbound.StoredTask, error) {
	rows, err := r.store.pool.Query(ctx, `SELECT id, run_id, task, status, runner_id FROM tasks WHERE run_id = $1 ORDER BY id`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]outbound.StoredTask, 0)
	for rows.Next() {
		value, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

type rowScanner interface{ Scan(...any) error }

func scanPipeline(row rowScanner) (outbound.StoredPipeline, error) {
	var value outbound.StoredPipeline
	var tasks []byte
	if err := row.Scan(&value.ID, &value.Definition.Name, &tasks); err != nil {
		return value, err
	}
	if err := json.Unmarshal(tasks, &value.Definition.Tasks); err != nil {
		return value, err
	}
	return value, nil
}
func scanRun(row rowScanner) (outbound.StoredRun, error) {
	var value outbound.StoredRun
	var definition []byte
	if err := row.Scan(&value.ID, &value.PipelineID, &definition, &value.Status, &value.CreatedAt, &value.StartedAt, &value.EndedAt); err != nil {
		return value, err
	}
	if err := json.Unmarshal(definition, &value.Definition); err != nil {
		return value, err
	}
	return value, nil
}
func scanTask(row rowScanner) (outbound.StoredTask, error) {
	var value outbound.StoredTask
	var task []byte
	if err := row.Scan(&value.ID, &value.RunID, &task, &value.Status, &value.RunnerID); err != nil {
		return value, err
	}
	if err := json.Unmarshal(task, &value.Task); err != nil {
		return value, err
	}
	return value, nil
}

var _ outbound.PipelineRepository = (*pipelineRepository)(nil)
var _ outbound.RunRepository = (*runRepository)(nil)
var _ outbound.TaskRepository = (*taskRepository)(nil)
