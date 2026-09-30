package run

import (
	"fmt"
	"strings"
	"time"

	"github.com/Design_Pattern/services/control-plane/internal/domain/pipeline"
)

type ID string

type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusCanceled  Status = "canceled"
)

type Run struct {
	id        ID
	status    Status
	snapshot  pipeline.Snapshot
	createdAt time.Time
	startedAt time.Time
	endedAt   time.Time
}

func New(id ID, definition pipeline.Pipeline, now time.Time) (*Run, error) {
	if strings.TrimSpace(string(id)) == "" {
		return nil, fmt.Errorf("run ID is required")
	}

	if now.IsZero() {
		return nil, fmt.Errorf("creation time is required")
	}

	snapshot, err := definition.Snapshot()
	if err != nil {
		return nil, fmt.Errorf("invalid pipeline definition: %w", err)
	}

	return &Run{
		id:        id,
		status:    StatusPending,
		snapshot:  snapshot,
		createdAt: now.UTC(),
	}, nil
}

func (r *Run) ID() ID {
	return r.id
}

func (r *Run) Status() Status {
	return r.status
}

func (r *Run) CreatedAt() time.Time {
	return r.createdAt
}

func (r *Run) StartedAt() (time.Time, bool) {
	return r.startedAt, !r.startedAt.IsZero()
}

func (r *Run) EndedAt() (time.Time, bool) {
	return r.endedAt, !r.endedAt.IsZero()
}

func (r *Run) Start(now time.Time) error {
	return r.transition(StatusRunning, now)
}

func (r *Run) Succeed(now time.Time) error {
	return r.transition(StatusSucceeded, now)
}

func (r *Run) Fail(now time.Time) error {
	return r.transition(StatusFailed, now)
}

func (r *Run) Cancel(now time.Time) error {
	return r.transition(StatusCanceled, now)
}

func (r *Run) Definition() pipeline.Snapshot {
	return r.snapshot
}

func (r *Run) transition(next Status, now time.Time) error {
	if !canTransition(r.status, next) {
		return fmt.Errorf(
			"invalid run transition: %q -> %q",
			r.status,
			next,
		)
	}

	if now.IsZero() {
		return fmt.Errorf("transition time is required")
	}

	latest := r.createdAt
	if !r.startedAt.IsZero() {
		latest = r.startedAt
	}

	if now.Before(latest) {
		return fmt.Errorf("transition time cannot precede the previous state")
	}

	r.status = next

	if next == StatusRunning {
		r.startedAt = now.UTC()
	} else {
		r.endedAt = now.UTC()
	}

	return nil
}

func canTransition(current, next Status) bool {
	switch current {
	case StatusPending:
		return next == StatusRunning || next == StatusCanceled
	case StatusRunning:
		return next == StatusSucceeded ||
			next == StatusFailed ||
			next == StatusCanceled
	default:
		return false
	}
}
