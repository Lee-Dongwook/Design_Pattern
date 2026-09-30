package http

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Design_Pattern/services/control-plane/internal/adapters/outbound/memory"
	"github.com/Design_Pattern/services/control-plane/internal/applications"
)

func TestPipelineAndRunWorkflow(t *testing.T) {
	store := memory.NewStore()
	handler := NewHandler(applications.NewService(memory.Pipelines(store), memory.Runs(store)))

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

func TestPipelineValidationError(t *testing.T) {
	store := memory.NewStore()
	handler := NewHandler(applications.NewService(memory.Pipelines(store), memory.Runs(store)))
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
