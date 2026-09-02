package common

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"time"
)

type AgentTask struct {
	ID         uint   `json:"id"`
	Type       string `json:"type"`
	Command    string `json:"command"`
	Status     string `json:"status"`
	TimeoutSec int    `json:"timeout_sec"`
}

type TaskResult struct {
	Status string `json:"status"`
	Result string `json:"result"`
}

func doRequestWithToken(method, url, token string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	return client.Do(req)
}

func FetchNextTask(serverURL, uuid, token string) (*AgentTask, error) {
	resp, err := doRequestWithToken("GET", serverURL+"/api/agents/tasks/next?host_uuid="+uuid, token, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch task status %d", resp.StatusCode)
	}
	var task AgentTask
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, err
	}
	return &task, nil
}

func ReportTaskResult(serverURL string, taskID uint, token, status, result string) error {
	body, _ := json.Marshal(TaskResult{Status: status, Result: result})
	resp, err := doRequestWithToken("POST", fmt.Sprintf("%s/api/agents/tasks/%d/result", serverURL, taskID), token, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func ExecuteCommand(command string, timeout int) (string, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("/bin/sh", "-c", command)
	}

	if timeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
		defer cancel()
		cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
		if runtime.GOOS == "windows" {
			cmd = exec.CommandContext(ctx, "cmd", "/c", command)
		} else {
			cmd = exec.CommandContext(ctx, "/bin/sh", "-c", command)
		}
	}

	output, err := cmd.CombinedOutput()
	return string(output), err
}
