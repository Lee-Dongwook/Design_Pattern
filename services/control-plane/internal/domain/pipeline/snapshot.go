package pipeline

type Snapshot struct {
	name  string
	tasks []Task
}

func (p Pipeline) Snapshot() (Snapshot, error) {
	if err := p.Validate(); err != nil {
		return Snapshot{}, err
	}

	return Snapshot{
		name:  p.Name,
		tasks: cloneTasks(p.Tasks),
	}, nil
}

func (s Snapshot) Name() string {
	return s.name
}

func (s Snapshot) Tasks() []Task {
	return cloneTasks(s.tasks)
}

func cloneTasks(tasks []Task) []Task {
	result := make([]Task, len(tasks))

	for i, task := range tasks {
		result[i] = task
		result[i].Command = append([]string(nil), task.Command...)
		result[i].DependsOn = append([]TaskID(nil), task.DependsOn...)
	}

	return result
}
