package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/agents"
	"github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/agents/conceptdesigner"
	"github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/modelrouter"
	"github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/promptloader"
)

type testAgent struct {
	run func(context.Context, agent.Task) (*agent.Result, error)
}

func (a testAgent) Name() string { return "test" }

func (a testAgent) Run(ctx context.Context, task agent.Task) (*agent.Result, error) {
	return a.run(ctx, task)
}

func testMux(designer agent.Agent) *http.ServeMux {
	mux := http.NewServeMux()
	NewHandler(nil, designer).RegisterRoutes(mux)
	return mux
}

func TestCreateConceptEndToEnd(t *testing.T) {
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request modelrouter.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode model request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if r.URL.Path != "/api/chat" || request.Format != "json" || len(request.Messages) != 2 || !strings.Contains(request.Messages[1].Content, "puzzle") {
			t.Errorf("unexpected model request: %s %+v", r.URL.Path, request)
		}
		response := modelrouter.ChatResponse{Model: "test-model", Message: modelrouter.Message{
			Content: `{"title":"Test","genre":"Puzzle","core_gameplay_loop":["Solve"],"target_audience":"Everyone","unique_hook":"Living puzzles","mechanics":[{"name":"Rotate","description":"Rotate puzzle pieces"}],"progression":"New puzzles","target_session_minutes":5,"development_complexity":"Low"}`,
		}}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer ollama.Close()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "concept_designer.md"), []byte("Design a game"), 0600); err != nil {
		t.Fatal(err)
	}
	client, err := modelrouter.NewClient(ollama.URL, "test-model")
	if err != nil {
		t.Fatal(err)
	}
	mux := testMux(conceptdesigner.NewConceptDesigner(client, promptloader.New(root)))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/concepts", strings.NewReader(`{"id":"task-1","objective":"Make a puzzle"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var result agent.Result
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	var concept conceptdesigner.GameConcept
	if err := json.Unmarshal(result.Output, &concept); err != nil {
		t.Fatal(err)
	}
	if result.TaskID != "task-1" || result.Status != "completed" || result.Model != "test-model" || concept.Title != "Test" {
		t.Fatalf("unexpected result: %+v, concept: %+v", result, concept)
	}
}

func TestCreateConceptErrors(t *testing.T) {
	for _, tt := range []struct {
		name   string
		body   string
		err    error
		status int
	}{
		{"invalid JSON", `{`, nil, http.StatusBadRequest},
		{"unknown field", `{"objective":"test","extra":true}`, nil, http.StatusBadRequest},
		{"missing objective", `{"objective":" "}`, nil, http.StatusBadRequest},
		{"agent failure", `{"objective":"test"}`, errors.New("model unavailable"), http.StatusInternalServerError},
		{"timeout", `{"objective":"test"}`, context.DeadlineExceeded, http.StatusGatewayTimeout},
	} {
		t.Run(tt.name, func(t *testing.T) {
			designer := testAgent{run: func(ctx context.Context, task agent.Task) (*agent.Result, error) {
				if tt.err == nil {
					t.Error("agent called for invalid request")
				}
				return nil, tt.err
			}}
			rec := httptest.NewRecorder()
			testMux(designer).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/concepts", strings.NewReader(tt.body)))
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, tt.status, rec.Body.String())
			}
		})
	}
}

func TestCreateConceptLimitsConcurrency(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	designer := testAgent{run: func(ctx context.Context, task agent.Task) (*agent.Result, error) {
		close(started)
		<-release
		return &agent.Result{Status: "completed", Output: json.RawMessage(`{}`)}, nil
	}}
	mux := testMux(designer)
	done := make(chan struct{})
	go func() {
		defer close(done)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/concepts", strings.NewReader(`{"objective":"first"}`)))
	}()
	<-started
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/concepts", strings.NewReader(`{"objective":"second"}`)))
	close(release)
	<-done
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusTooManyRequests)
	}
}
