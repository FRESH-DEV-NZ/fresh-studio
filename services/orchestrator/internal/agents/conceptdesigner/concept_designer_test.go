package conceptdesigner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	agent "github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/agents"
	"github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/modelrouter"
	"github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/promptloader"
)

func TestRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request modelrouter.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if r.URL.Path != "/api/chat" || request.Format != "json" || len(request.Messages) != 2 {
			t.Errorf("unexpected request: %s %+v", r.URL.Path, request)
		}
		response := modelrouter.ChatResponse{
			Model: "test-model",
			Message: modelrouter.Message{
				Content: `{"title":"Test","genre":"Puzzle","core_loop":"Solve puzzles","target_audience":"Everyone","unique_hook":"Living puzzles","mechanics":["Rotate"],"progression":"New puzzles","target_session_minutes":5}`,
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer server.Close()

	root := t.TempDir()
	promptDir := filepath.Join(root, "prompts")

	if err := os.MkdirAll(promptDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(promptDir, "concept_designer.md"),
		[]byte("Design a game"),
		0600,
	); err != nil {
		t.Fatal(err)
	}
	client, err := modelrouter.NewClient(server.URL, "test-model")
	if err != nil {
		t.Fatal(err)
	}
	designer := NewConceptDesigner(client, promptloader.New(promptDir))

	result, err := designer.Run(context.Background(), agent.Task{ID: "task-1", Objective: "Make a puzzle"})
	if err != nil {
		t.Fatal(err)
	}
	if result.TaskID != "task-1" || result.Agent != designer.Name() || result.Status != "completed" || result.Model != "test-model" {
		t.Fatalf("unexpected result: %+v", result)
	}
	var concept GameConcept
	if err := json.Unmarshal(result.Output, &concept); err != nil {
		t.Fatal(err)
	}
	if concept.Title != "Test" || concept.TargetSessionMins != 5 {
		t.Fatalf("unexpected concept: %+v", concept)
	}
}

func TestRunRequiresObjective(t *testing.T) {
	designer := NewConceptDesigner(nil, nil)
	_, err := designer.Run(context.Background(), agent.Task{Objective: " "})
	if err == nil || !strings.Contains(err.Error(), "task objective is required") {
		t.Fatalf("expected objective error, got %v", err)
	}
}
