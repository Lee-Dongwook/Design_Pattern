package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Design_Pattern/services/control-plane/internal/adapters/outbound/memory"
	"github.com/Design_Pattern/services/control-plane/internal/applications"
)

func TestPipelineAndRunWorkflow(t *testing.T) {
	store := memory.NewStore()
	handler := NewHandler(applications.NewService(memory.Pipelines(store), memory.Runs(store), memory.Tasks(store)))

	pipeline := []byte(`{"id":"sample","name":"Sample","tasks":[{"id":"build","image":"alpine:3.20","command":["echo","build"]}]}`)
	response := request(handler, http.MethodPost, "/pipelines", pipeline)
	if response.Code != http.StatusCreated {
		t.Fatalf("save pipeline: expected 201, got %d: %s", response.Code, response.Body.String())
	}

	response = request(handler, http.MethodPost, "/runs", []byte(`{"pipelineId":"sample"}`))
	if response.Code != http.StatusCreated {
		t.Fatalf("create run: expected 201, got %d: %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"status":"pending"`)) {
		t.Fatalf("created run should be pending: %s", response.Body.String())
	}

	response = request(handler, http.MethodGet, "/runs", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("list runs: expected 200, got %d", response.Code)
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"pipelineId":"sample"`)) {
		t.Fatalf("run response omitted pipeline ID: %s", response.Body.String())
	}
}

func TestRunnerTaskEventsCompleteRunInDependencyOrder(t *testing.T) {
	store := memory.NewStore()
	handler := NewHandler(applications.NewService(memory.Pipelines(store), memory.Runs(store), memory.Tasks(store)))
	pipeline := []byte(`{"id":"ordered","name":"Ordered","tasks":[{"id":"build","image":"alpine","command":["true"]},{"id":"test","image":"alpine","command":["true"],"dependsOn":["build"]}]}`)
	if response := request(handler, http.MethodPost, "/pipelines", pipeline); response.Code != http.StatusCreated {
		t.Fatalf("save pipeline: %d", response.Code)
	}
	created := request(handler, http.MethodPost, "/runs", []byte(`{"pipelineId":"ordered"}`))
	var run struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &run); err != nil {
		t.Fatal(err)
	}

	first := claim(t, handler)
	report(t, handler, first.ID, "started")
	report(t, handler, first.ID, "succeeded")
	second := claim(t, handler)
	if second.Task.ID != "test" {
		t.Fatalf("expected dependent test task, got %q", second.Task.ID)
	}
	report(t, handler, second.ID, "started")
	report(t, handler, second.ID, "succeeded")

	result := request(handler, http.MethodGet, "/runs/"+run.ID, nil)
	if result.Code != http.StatusOK || !bytes.Contains(result.Body.Bytes(), []byte(`"status":"succeeded"`)) {
		t.Fatalf("run was not completed: %d %s", result.Code, result.Body.String())
	}
}

type claimedTask struct {
	ID   string `json:"id"`
	Task struct {
		ID string `json:"id"`
	} `json:"task"`
}

func claim(t *testing.T, handler http.Handler) claimedTask {
	t.Helper()
	response := request(handler, http.MethodPost, "/runner/tasks/claim", []byte(`{"runnerId":"runner-1"}`))
	if response.Code != http.StatusOK {
		t.Fatalf("claim task: %d %s", response.Code, response.Body.String())
	}
	var task claimedTask
	if err := json.Unmarshal(response.Body.Bytes(), &task); err != nil {
		t.Fatal(err)
	}
	return task
}
func report(t *testing.T, handler http.Handler, taskID, status string) {
	t.Helper()
	response := request(handler, http.MethodPost, "/runner/tasks/"+taskID+"/events", []byte(`{"status":"`+status+`"}`))
	if response.Code != http.StatusNoContent {
		t.Fatalf("report %s: %d %s", status, response.Code, response.Body.String())
	}
}

func TestPipelineValidationError(t *testing.T) {
	store := memory.NewStore()
	handler := NewHandler(applications.NewService(memory.Pipelines(store), memory.Runs(store), memory.Tasks(store)))
	response := request(handler, http.MethodPost, "/pipelines", []byte(`{"id":"invalid","name":"","tasks":[]}`))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
}

func request(handler http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
