package agent

import (
	"context"
	"encoding/json"
)

type Task struct {
	ID          string   `json:"id"`
	Objective   string   `json:"objective"`
	Context     string   `json:"context"`
	Constraints []string `json:"constraints"`
}

type Result struct {
	TaskID string          `json:"task_id"`
	Agent  string          `json:"agent"`
	Status string          `json:"status"`
	Model  string          `json:"model"`
	Output json.RawMessage `json:"output"`
}

type Agent interface {
	Name() string

	Run(
		ctx context.Context,
		task Task,
	) (*Result, error)
}
