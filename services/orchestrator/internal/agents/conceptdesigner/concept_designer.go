package conceptdesigner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	agent "github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/agents"
	"github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/modelrouter"
	"github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/promptloader"
)

type GameConcept struct {
	Title             string   `json:"title"`
	Genre             string   `json:"genre"`
	CoreLoop          string   `json:"core_loop"`
	TargetAudience    string   `json:"target_audience"`
	UniqueHook        string   `json:"unique_hook"`
	Mechanics         []string `json:"mechanics"`
	Progression       string   `json:"progression"`
	TargetSessionMins int      `json:"target_session_minutes"`
}

type ConceptDesigner struct {
	model   *modelrouter.Client
	prompts *promptloader.Loader
}

func NewConceptDesigner(
	model *modelrouter.Client,
	prompts *promptloader.Loader,
) *ConceptDesigner {
	return &ConceptDesigner{
		model:   model,
		prompts: prompts,
	}
}

func (d *ConceptDesigner) Name() string {
	return "concept_designer"
}

func (d *ConceptDesigner) Run(
	ctx context.Context,
	task agent.Task,
) (*agent.Result, error) {

	// Validate the task before attempting to use dependencies.
	if strings.TrimSpace(task.Objective) == "" {
		return nil, fmt.Errorf("task objective is required")
	}

	if d.model == nil {
		return nil, fmt.Errorf("concept designer model client is nil")
	}

	if d.prompts == nil {
		return nil, fmt.Errorf("concept designer prompt loader is nil")
	}

	// Load the system prompt.
	systemPrompt, err := d.prompts.Load(
		"concept_designer.md",
	)
	if err != nil {
		return nil, fmt.Errorf("load concept designer prompt: %w", err)
	}

	// Convert constraints into readable prompt text.
	constraints := "None specified"
	if len(task.Constraints) > 0 {
		constraints = "- " + strings.Join(task.Constraints, "\n- ")
	}

	userPrompt := fmt.Sprintf(
		`Develop a detailed game concept based on this task.

Objective:
%s

Additional context:
%s

Constraints:
%s

Return a valid JSON object containing exactly these fields:
- title: string
- genre: string
- core_loop: string
- target_audience: string
- unique_hook: string
- mechanics: array of strings
- progression: string
- target_session_minutes: integer

Do not include Markdown code fences or any text outside the JSON object.`,
		task.Objective,
		defaultValue(task.Context, "None provided"),
		constraints,
	)

	messages := []modelrouter.Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: userPrompt,
		},
	}

	// Request structured JSON from Ollama.
	response, err := d.model.ChatJSON(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("generate game concept: %w", err)
	}

	// Parse and validate the model's JSON response.
	var concept GameConcept

	if err := json.Unmarshal(
		[]byte(response.Message.Content),
		&concept,
	); err != nil {
		return nil, fmt.Errorf("decode game concept JSON: %w", err)
	}

	// Convert the typed concept back into JSON for the agent result.
	output, err := json.Marshal(concept)
	if err != nil {
		return nil, fmt.Errorf("encode game concept: %w", err)
	}

	return &agent.Result{
		TaskID: task.ID,
		Agent:  d.Name(),
		Status: "completed",
		Model:  response.Model,
		Output: output,
	}, nil
}

func defaultValue(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	return value
}
