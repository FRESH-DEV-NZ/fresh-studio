package conceptdesigner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/agents"
	"github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/modelrouter"
	"github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/promptloader"
)

var _ agent.Agent = (*ConceptDesigner)(nil)

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

func NewConceptDesigner(model *modelrouter.Client, prompts *promptloader.Loader) *ConceptDesigner {
	return &ConceptDesigner{
		model:   model,
		prompts: prompts,
	}
}

func (a *ConceptDesigner) Name() string {
	return "game-concept-designer"
}

func (a *ConceptDesigner) Run(
	ctx context.Context,
	task agent.Task,
) (*agent.Result, error) {

	if strings.TrimSpace(task.Objective) == "" {
		return nil, fmt.Errorf("task objective is required")
	}

	systemPrompt, err := a.prompts.Load(
		"concept_designer.md",
	)

	if err != nil {
		return nil, fmt.Errorf("load system prompt: %w", err)
	}

	userPrompt := fmt.Sprintf(`
		TASK OBJECTIVE:
		%s

		ADDITIONAL CONTEXT:
		%s

		CONSTRAINTS:
		%s

		Create a complete game concept that satisfies the task.
		`, task.Objective, task.Context, strings.Join(task.Constraints, "\n- "))

	response, err := a.model.ChatJSON(ctx, []modelrouter.Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: userPrompt,
		},
	})

	if err != nil {
		return nil, fmt.Errorf("concept designer model request: %w", err)
	}

	var concept GameConcept

	if err := json.Unmarshal(
		[]byte(response.Message.Content),
		&concept,
	); err != nil {
		return nil, fmt.Errorf("parse game concept JSON: %w", err)
	}

	if err := validateConcept(concept); err != nil {
		return nil, fmt.Errorf("invalid game concept: %w", err)
	}

	output, err := json.MarshalIndent(concept, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode game concept: %w", err)
	}

	return &agent.Result{
		TaskID: task.ID,
		Agent:  a.Name(),
		Status: "completed",
		Model:  response.Model,
		Output: output,
	}, nil
}

func validateConcept(c GameConcept) error {
	if strings.TrimSpace(c.Title) == "" {
		return fmt.Errorf("title is required")
	}

	if strings.TrimSpace(c.Genre) == "" {
		return fmt.Errorf("genre is required")
	}

	if strings.TrimSpace(c.CoreLoop) == "" {
		return fmt.Errorf("core loop is required")
	}

	if strings.TrimSpace(c.UniqueHook) == "" {
		return fmt.Errorf("unique hook is required")
	}

	if len(c.Mechanics) == 0 {
		return fmt.Errorf("at least one mechanic is required")
	}

	if c.TargetSessionMins <= 0 || c.TargetSessionMins > 120 {
		return fmt.Errorf("target session duration must be between 1 and 120 minutes")
	}

	return nil
}
