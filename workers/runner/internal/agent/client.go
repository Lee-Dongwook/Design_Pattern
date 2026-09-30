package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Design_Pattern/workers/runner/internal/execution"
)

type Client struct {
	baseURL    string
	runnerID   string
	httpClient *http.Client
}
type assignment struct {
	ID   string `json:"id"`
	Task struct {
		Image   string   `json:"image"`
		Command []string `json:"command"`
	} `json:"task"`
}

func NewClient(baseURL, runnerID string) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), runnerID: runnerID, httpClient: http.DefaultClient}
}

func RunOnce(ctx context.Context, client *Client, executor execution.Executor) error {
	task, found, err := client.claim(ctx)
	if err != nil || !found {
		return err
	}
	if err := client.report(ctx, task.ID, "started", ""); err != nil {
		return err
	}
	result, err := executor.Execute(ctx, execution.Task{Image: task.Task.Image, Command: task.Task.Command})
	if err != nil {
		message := result.Output
		if message == "" {
			message = err.Error()
		}
		if reportErr := client.report(ctx, task.ID, "failed", message); reportErr != nil {
			return fmt.Errorf("execute task: %w; report failure: %v", err, reportErr)
		}
		return nil
	}
	return client.report(ctx, task.ID, "succeeded", result.Output)
}
func (c *Client) claim(ctx context.Context) (assignment, bool, error) {
	body, _ := json.Marshal(map[string]string{"runnerId": c.runnerID})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/runner/tasks/claim", bytes.NewReader(body))
	if err != nil {
		return assignment{}, false, err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return assignment{}, false, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNoContent {
		return assignment{}, false, nil
	}
	if response.StatusCode != http.StatusOK {
		return assignment{}, false, responseError(response)
	}
	var task assignment
	if err := json.NewDecoder(response.Body).Decode(&task); err != nil {
		return assignment{}, false, err
	}
	return task, true, nil
}
func (c *Client) report(ctx context.Context, taskID, status, message string) error {
	body, _ := json.Marshal(map[string]string{"status": status, "message": message})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/runner/tasks/"+taskID+"/events", bytes.NewReader(body))
	if err != nil {
		return err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		return responseError(response)
	}
	return nil
}
func responseError(response *http.Response) error {
	data, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	return fmt.Errorf("control plane returned %s: %s", response.Status, strings.TrimSpace(string(data)))
}
