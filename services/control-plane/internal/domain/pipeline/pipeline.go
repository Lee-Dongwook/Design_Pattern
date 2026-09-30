package pipeline

import (
	"fmt"
	"strings"
)

type Pipeline struct {
	Name  string
	Tasks []Task
}

func (p Pipeline) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("pipeline name is required")
	}

	if len(p.Tasks) == 0 {
		return fmt.Errorf("pipeline must contain at least one task")
	}

	tasks := make(map[TaskID]Task, len(p.Tasks))

	for _, task := range p.Tasks {
		if strings.TrimSpace(string(task.ID)) == "" {
			return fmt.Errorf("task ID is required")
		}

		if _, exists := tasks[task.ID]; exists {
			return fmt.Errorf("duplicate task ID %q", task.ID)
		}

		if strings.TrimSpace(task.Image) == "" {
			return fmt.Errorf("task %q: image is required", task.ID)
		}

		if len(task.Command) == 0 ||
			strings.TrimSpace(task.Command[0]) == "" {
			return fmt.Errorf("task %q: command is required", task.ID)
		}

		tasks[task.ID] = task
	}

	for _, task := range p.Tasks {
		seen := make(map[TaskID]bool, len(task.DependsOn))

		for _, dependency := range task.DependsOn {
			if dependency == task.ID {
				return fmt.Errorf("task %q cannot depend on itself", task.ID)
			}

			if _, exists := tasks[dependency]; !exists {
				return fmt.Errorf(
					"task %q: unknown dependency %q",
					task.ID,
					dependency,
				)
			}

			if seen[dependency] {
				return fmt.Errorf(
					"task %q: duplicate dependency %q",
					task.ID,
					dependency,
				)
			}

			seen[dependency] = true
		}
	}

	return p.validateAcyclic(tasks)
}

func (p Pipeline) validateAcyclic(tasks map[TaskID]Task) error {
	type visitState uint8

	const (
		unvisited visitState = iota
		visiting
		visited
	)

	states := make(map[TaskID]visitState, len(tasks))

	var visit func(TaskID) error

	visit = func(id TaskID) error {
		switch states[id] {
		case visiting:
			return fmt.Errorf("dependency cycle detected at task %q", id)
		case visited:
			return nil
		}

		states[id] = visiting

		for _, dependency := range tasks[id].DependsOn {
			if err := visit(dependency); err != nil {
				return err
			}
		}

		states[id] = visited
		return nil
	}

	for _, task := range p.Tasks {
		if err := visit(task.ID); err != nil {
			return err
		}
	}

	return nil
}
