
package conceptdesigner

import (
	"context"
	"fmt"
	"strings"

	"github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/modelrouter"
	"github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/promptloader"
)

type ConceptRequest struct {
	Idea     string
	Platform string
	Audience string
}

type ConceptDesigner struct {
	model   *modelrouter.Client
	prompts *promptloader.Loader
}

func New(
	model *modelrouter.Client,
	prompts *promptloader.Loader,
) *ConceptDesigner {
	return &ConceptDesigner{
		model:   model,
		prompts: prompts,
	}
}

func (d *ConceptDesigner) Design(
	ctx context.Context,
	request ConceptRequest,
) (string, error) {

	if strings.TrimSpace(request.Idea) == "" {
		return "", fmt.Errorf("game idea cannot be empty")
	}

	systemPrompt, err := d.prompts.Load(
		"creative/concept_designer.md",
	)
	if err != nil {
		return "", fmt.Errorf("load concept designer prompt: %w", err)
	}

	userPrompt := fmt.Sprintf(
		`Develop a game concept based on the following idea.

Game idea:
%s

Target platform:
%s

Target audience:
%s

Follow your system instructions and return valid JSON.`,
		request.Idea,
		defaultValue(request.Platform, "Web"),
		defaultValue(request.Audience, "General audience"),
	)

	response, err := d.model.Chat(ctx, []modelrouter.Message{
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
		return "", fmt.Errorf("generate game concept: %w", err)
	}

	return response.Message.Content, nil
}

func defaultValue(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	return value
}
