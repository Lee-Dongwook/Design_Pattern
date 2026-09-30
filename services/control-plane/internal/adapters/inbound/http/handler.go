// Package http adapts the public HTTP contract to the Control Plane inbound port.
package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Design_Pattern/services/control-plane/internal/applications"
	"github.com/Design_Pattern/services/control-plane/internal/domain/pipeline"
)

func NewUnavailableHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusServiceUnavailable, "control-plane dependencies are not configured")
	})
	return mux
}

func NewHandler(api applications.API) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /pipelines", func(w http.ResponseWriter, r *http.Request) {
		pipelines, err := api.ListPipelines(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		response := make([]pipelineResponse, 0, len(pipelines))
		for _, item := range pipelines {
			response = append(response, toPipelineResponse(item))
		}
		writeJSON(w, http.StatusOK, map[string]any{"pipelines": response})
	})
	mux.HandleFunc("POST /pipelines", func(w http.ResponseWriter, r *http.Request) {
		var request pipelineResponse
		if !decodeJSON(w, r, &request) {
			return
		}
		saved, err := api.SavePipeline(r.Context(), toApplicationPipeline(request))
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, toPipelineResponse(saved))
	})
	mux.HandleFunc("GET /runs", func(w http.ResponseWriter, r *http.Request) {
		runs, err := api.ListRuns(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		response := make([]runResponse, 0, len(runs))
		for _, item := range runs {
			response = append(response, toRunResponse(item))
		}
		writeJSON(w, http.StatusOK, map[string]any{"runs": response})
	})
	mux.HandleFunc("POST /runs", func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			PipelineID string `json:"pipelineId"`
		}
		if !decodeJSON(w, r, &request) {
			return
		}
		created, err := api.CreateRun(r.Context(), request.PipelineID)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, toRunResponse(created))
	})
	mux.HandleFunc("GET /runs/{runId}", func(w http.ResponseWriter, r *http.Request) {
		found, ok, err := api.GetRun(r.Context(), r.PathValue("runId"))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(w, http.StatusNotFound, "run not found")
			return
		}
		writeJSON(w, http.StatusOK, toRunResponse(found))
	})
	mux.HandleFunc("GET /runs/{runId}/tasks", func(w http.ResponseWriter, r *http.Request) {
		tasks, found, err := api.ListRunTasks(r.Context(), r.PathValue("runId"))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(w, http.StatusNotFound, "run not found")
			return
		}
		response := make([]taskExecutionResponse, 0, len(tasks))
		for _, task := range tasks {
			response = append(response, taskExecutionResponse{ID: task.ID, TaskID: task.TaskID, Status: task.Status, RunnerID: task.RunnerID, Log: task.Log})
		}
		writeJSON(w, http.StatusOK, map[string]any{"tasks": response})
	})
	mux.HandleFunc("POST /runner/tasks/claim", func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			RunnerID string `json:"runnerId"`
		}
		if !decodeJSON(w, r, &request) {
			return
		}
		assignment, ok, err := api.ClaimTask(r.Context(), request.RunnerID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeJSON(w, http.StatusOK, taskAssignmentResponse{ID: assignment.ID, RunID: assignment.RunID, Task: toTaskResponse(assignment.Task)})
	})
	mux.HandleFunc("POST /runner/tasks/{taskId}/events", func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Status  applications.TaskEventStatus `json:"status"`
			Message string                       `json:"message"`
		}
		if !decodeJSON(w, r, &request) {
			return
		}
		found, err := api.ReportTaskEvent(r.Context(), r.PathValue("taskId"), request.Status, request.Message)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if !found {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	return mux
}

type pipelineResponse struct {
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Tasks []taskResponse `json:"tasks"`
}
type taskResponse struct {
	ID        string   `json:"id"`
	Image     string   `json:"image"`
	Command   []string `json:"command"`
	DependsOn []string `json:"dependsOn,omitempty"`
}
type runResponse struct {
	ID         string     `json:"id"`
	PipelineID string     `json:"pipelineId"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"createdAt"`
	StartedAt  *time.Time `json:"startedAt"`
	EndedAt    *time.Time `json:"endedAt"`
}
type taskAssignmentResponse struct {
	ID    string       `json:"id"`
	RunID string       `json:"runId"`
	Task  taskResponse `json:"task"`
}
type taskExecutionResponse struct {
	ID       string `json:"id"`
	TaskID   string `json:"taskId"`
	Status   string `json:"status"`
	RunnerID string `json:"runnerId"`
	Log      string `json:"log"`
}

func toApplicationPipeline(value pipelineResponse) applications.Pipeline {
	tasks := make([]pipeline.Task, 0, len(value.Tasks))
	for _, task := range value.Tasks {
		tasks = append(tasks, pipeline.Task{ID: pipeline.TaskID(task.ID), Image: task.Image, Command: task.Command, DependsOn: toTaskIDs(task.DependsOn)})
	}
	return applications.Pipeline{ID: value.ID, Definition: pipeline.Pipeline{Name: value.Name, Tasks: tasks}}
}
func toPipelineResponse(value applications.Pipeline) pipelineResponse {
	tasks := value.Definition.Tasks
	result := make([]taskResponse, 0, len(tasks))
	for _, task := range tasks {
		result = append(result, toTaskResponse(task))
	}
	return pipelineResponse{ID: value.ID, Name: value.Definition.Name, Tasks: result}
}
func toTaskResponse(value pipeline.Task) taskResponse {
	dependencies := make([]string, 0, len(value.DependsOn))
	for _, id := range value.DependsOn {
		dependencies = append(dependencies, string(id))
	}
	return taskResponse{ID: string(value.ID), Image: value.Image, Command: append([]string(nil), value.Command...), DependsOn: dependencies}
}
func toTaskIDs(values []string) []pipeline.TaskID {
	result := make([]pipeline.TaskID, 0, len(values))
	for _, value := range values {
		result = append(result, pipeline.TaskID(value))
	}
	return result
}
func toRunResponse(value applications.Run) runResponse {
	return runResponse{ID: value.ID, PipelineID: value.PipelineID, Status: string(value.Status), CreatedAt: value.CreatedAt, StartedAt: value.StartedAt, EndedAt: value.EndedAt}
}
func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if r.Body == nil {
		writeError(w, http.StatusBadRequest, "request body is required")
		return false
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request: "+err.Error())
		return false
	}
	return true
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"message": message})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
