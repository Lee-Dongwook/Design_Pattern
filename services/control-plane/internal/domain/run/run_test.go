package run

import (
	"testing"
	"time"
)

func TestRunTransitions(t *testing.T) {
	base := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		from    Status
		to      Status
		allowed bool
	}{
		{"pending to running", StatusPending, StatusRunning, true},
		{"pending to succeeded", StatusPending, StatusSucceeded, false},
		{"pending to failed", StatusPending, StatusFailed, false},
		{"pending to canceled", StatusPending, StatusCanceled, true},

		{"running to running", StatusRunning, StatusRunning, false},
		{"running to succeeded", StatusRunning, StatusSucceeded, true},
		{"running to failed", StatusRunning, StatusFailed, true},
		{"running to canceled", StatusRunning, StatusCanceled, true},
	}

	// All terminal states must reject further transitions.
	for _, from := range []Status{
		StatusSucceeded,
		StatusFailed,
		StatusCanceled,
	} {
		for _, to := range []Status{
			StatusRunning,
			StatusSucceeded,
			StatusFailed,
			StatusCanceled,
		} {
			tests = append(tests, struct {
				name    string
				from    Status
				to      Status
				allowed bool
			}{
				name: string(from) + " to " + string(to),
				from: from,
				to:   to,
			})
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newRunAtStatus(t, tt.from, base)
			before := *r
			at := base.Add(3 * time.Minute)

			err := applyTransition(r, tt.to, at)

			if !tt.allowed {
				if err == nil {
					t.Fatal("expected transition to be rejected")
				}

				if *r != before {
					t.Fatal("rejected transition changed the run")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if r.Status() != tt.to {
				t.Fatalf("want status %q, got %q", tt.to, r.Status())
			}

			if r.ID() != before.ID() ||
				!r.CreatedAt().Equal(before.CreatedAt()) {
				t.Fatal("transition changed run identity or creation time")
			}

			if tt.to == StatusRunning {
				startedAt, ok := r.StartedAt()
				if !ok || !startedAt.Equal(at) {
					t.Fatal("start time was not recorded correctly")
				}

				if _, ok := r.EndedAt(); ok {
					t.Fatal("running run must not have an end time")
				}
			} else {
				endedAt, ok := r.EndedAt()
				if !ok || !endedAt.Equal(at) {
					t.Fatal("end time was not recorded correctly")
				}

				if r.startedAt != before.startedAt {
					t.Fatal("terminal transition changed the start time")
				}
			}
		})
	}
}

func TestRunRejectsInvalidTransitionTime(t *testing.T) {
	base := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		from Status
		to   Status
		at   time.Time
	}{
		{
			name: "zero time",
			from: StatusPending,
			to:   StatusRunning,
			at:   time.Time{},
		},
		{
			name: "start before creation",
			from: StatusPending,
			to:   StatusRunning,
			at:   base.Add(-time.Second),
		},
		{
			name: "finish before start",
			from: StatusRunning,
			to:   StatusSucceeded,
			at:   base.Add(30 * time.Second),
		},
		{
			name: "cancel before creation",
			from: StatusPending,
			to:   StatusCanceled,
			at:   base.Add(-time.Second),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newRunAtStatus(t, tt.from, base)
			before := *r

			if err := applyTransition(r, tt.to, tt.at); err == nil {
				t.Fatal("expected invalid time to be rejected")
			}

			if *r != before {
				t.Fatal("rejected transition changed the run")
			}
		})
	}
}

func newRunAtStatus(t *testing.T, status Status, base time.Time) *Run {
	t.Helper()

	r, err := New("run-001", base)
	if err != nil {
		t.Fatal(err)
	}

	if status == StatusPending {
		return r
	}

	if err := r.Start(base.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	if status != StatusRunning {
		if err := applyTransition(r, status, base.Add(2*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}

	return r
}

func applyTransition(r *Run, to Status, at time.Time) error {
	switch to {
	case StatusRunning:
		return r.Start(at)
	case StatusSucceeded:
		return r.Succeed(at)
	case StatusFailed:
		return r.Fail(at)
	case StatusCanceled:
		return r.Cancel(at)
	default:
		panic("unsupported test status: " + string(to))
	}
}
