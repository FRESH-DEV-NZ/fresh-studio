package conceptdesigner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"io"

	agent "github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/agents"
	"github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/modelrouter"
	"github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/promptloader"
)

type GameConcept struct {
	Title                 string     `json:"title"`
	Genre                 string     `json:"genre"`
	CoreGameplayLoop      []string   `json:"core_gameplay_loop"`
	TargetAudience        string     `json:"target_audience"`
	UniqueHook            string     `json:"unique_hook"`
	Mechanics             []Mechanic `json:"mechanics"`
	Progression           string     `json:"progression"`
	TargetSessionMinutes  int        `json:"target_session_minutes"`
	DevelopmentComplexity string     `json:"development_complexity"`
}

type Mechanic struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (c GameConcept) Validate() error {
	if strings.TrimSpace(c.Title) == "" {
		return fmt.Errorf("title is required")
	}

	if strings.TrimSpace(c.Genre) == "" {
		return fmt.Errorf("genre is required")
	}

	if len(c.CoreGameplayLoop) == 0 {
		return fmt.Errorf("core gameplay loop is required")
	}

	for i, step := range c.CoreGameplayLoop {
		if strings.TrimSpace(step) == "" {
			return fmt.Errorf("core gameplay loop step %d is empty", i)
		}
	}

	if strings.TrimSpace(c.TargetAudience) == "" {
		return fmt.Errorf("target audience is required")
	}

	if strings.TrimSpace(c.UniqueHook) == "" {
		return fmt.Errorf("unique hook is required")
	}

	if len(c.Mechanics) == 0 {
		return fmt.Errorf("at least one mechanic is required")
	}

	for i, mechanic := range c.Mechanics {
		if strings.TrimSpace(mechanic.Name) == "" {
			return fmt.Errorf("mechanic %d has no name", i)
		}

		if strings.TrimSpace(mechanic.Description) == "" {
			return fmt.Errorf("mechanic %d has no description", i)
		}
	}

	if strings.TrimSpace(c.Progression) == "" {
		return fmt.Errorf("progression is required")
	}

	if c.TargetSessionMinutes <= 0 {
		return fmt.Errorf("target session duration must be greater than zero")
	}

	if c.TargetSessionMinutes > 120 {
		return fmt.Errorf("target session duration cannot exceed 120 minutes")
	}

	complexity := strings.ToLower(strings.TrimSpace(c.DevelopmentComplexity))

	if complexity != "low" && complexity != "medium" && complexity != "high" {
		return fmt.Errorf("development complexity must be Low, Medium, or High")
	}

	return nil
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

		{
		"title": "string",
		"genre": "string",
		"core_gameplay_loop": ["string"],
		"target_audience": "string",
		"unique_hook": "string",
		"mechanics": [
			{
			"name": "string",
			"description": "string"
			}
		],
		"progression": "string",
		"target_session_minutes": 5,
		"development_complexity": "Low"
		}

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
		if ctx.Err() != nil {
			return nil, fmt.Errorf("generate game concept: request cancelled or timed out: %w", ctx.Err())
		}

		return nil, fmt.Errorf("generate game concept: Ollama request failed: %w", err)
	}

	if response == nil {
		return nil, fmt.Errorf("generate game concept: Ollama returned an empty response")
	}

	if strings.TrimSpace(response.Message.Content) == "" {
		return nil, fmt.Errorf("generate game concept: Ollama returned empty content")
	}

	if strings.TrimSpace(response.Model) == "" {
		return nil, fmt.Errorf("generate game concept: Ollama response did not identify a model")
	}

	// Parse and validate the model's JSON response.
	var concept GameConcept

	decoder := json.NewDecoder(
		strings.NewReader(response.Message.Content),
	)

	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&concept); err != nil {
		return nil, fmt.Errorf("decode game concept JSON: %w", err)
	}

	// Ensure the response contains exactly one JSON value.
	var extra interface{}

	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("decode game concept JSON: unexpected additional content")
	}

	if err := concept.Validate(); err != nil {
		return nil, fmt.Errorf("validate game concept: %w", err)
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
