package pipeline

import (
	"strings"
	"testing"
)

func TestPipelineValidate(t *testing.T) {
	task := func(id TaskID, dependencies ...TaskID) Task {
		return Task{
			ID:        id,
			Image:     "test-image",
			Command:   []string{"echo", "ok"},
			DependsOn: dependencies,
		}
	}

	tests := []struct {
		name    string
		tasks   []Task
		wantErr string
	}{
		{
			name:  "single task",
			tasks: []Task{task("build")},
		},
		{
			name: "parallel tasks with shared dependency",
			tasks: []Task{
				task("build"),
				task("unit-test", "build"),
				task("integration-test", "build"),
				task("deploy", "unit-test", "integration-test"),
			},
		},
		{
			name: "declaration order does not affect dependencies",
			tasks: []Task{
				task("deploy", "test"),
				task("test", "build"),
				task("build"),
			},
		},
		{
			name:    "no tasks",
			wantErr: "at least one task",
		},
		{
			name:    "duplicate task IDs",
			tasks:   []Task{task("build"), task("build")},
			wantErr: "duplicate task ID",
		},
		{
			name:    "unknown dependency",
			tasks:   []Task{task("deploy", "missing")},
			wantErr: "unknown dependency",
		},
		{
			name:    "self dependency",
			tasks:   []Task{task("build", "build")},
			wantErr: "cannot depend on itself",
		},
		{
			name: "duplicate dependency",
			tasks: []Task{
				task("build"),
				task("test", "build", "build"),
			},
			wantErr: "duplicate dependency",
		},
		{
			name: "cycle across multiple tasks",
			tasks: []Task{
				task("a", "c"),
				task("b", "a"),
				task("c", "b"),
			},
			wantErr: "dependency cycle",
		},
		{
			name: "cycle in disconnected component",
			tasks: []Task{
				task("build"),
				task("test", "build"),
				task("a", "b"),
				task("b", "a"),
			},
			wantErr: "dependency cycle",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Pipeline{
				Name:  "example",
				Tasks: tt.tasks,
			}

			err := p.Validate()

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected valid pipeline, got: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected error containing %q", tt.wantErr)
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}
