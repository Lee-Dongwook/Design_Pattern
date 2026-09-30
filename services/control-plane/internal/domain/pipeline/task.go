package pipeline

type TaskID string

type Task struct {
	ID        TaskID
	Image     string
	Command   []string
	DependsOn []TaskID
}
