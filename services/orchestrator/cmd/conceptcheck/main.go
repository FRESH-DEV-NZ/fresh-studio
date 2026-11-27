package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	agent "github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/agents"
	"github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/agents/conceptdesigner"
	"github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/modelrouter"
	"github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/promptloader"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables.
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found; using environment variables")
	}

	// Configure Ollama.
	host := os.Getenv("OLLAMA_HOST")
	if host == "" {
		host = "http://localhost:11434"
	}

	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		log.Fatal("OLLAMA_MODEL must be configured")
	}

	modelClient, err := modelrouter.NewClient(host, model)
	if err != nil {
		log.Fatal(err)
	}

	// Load prompts from the repository's prompts directory.
	prompts := promptloader.New("prompts")

	// Create the Concept Designer agent.
	designer := conceptdesigner.NewConceptDesigner(
		modelClient,
		prompts,
	)

	// Define the task.
	task := agent.Task{
		ID:        "concept-check-001",
		Objective: "Create a creative 2D puzzle game where the player manipulates gravity to solve environmental challenges.",
		Context:   "The game should be suitable for browser-based play and have short, engaging sessions.",
		Constraints: []string{
			"Must be playable using keyboard controls",
			"Should be achievable as a small indie game",
			"Must have a distinctive central mechanic",
		},
	}

	fmt.Printf("Running agent: %s\n", designer.Name())
	fmt.Printf("Model: %s\n\n", modelClient.Model())

	// Execute the agent.
	result, err := designer.Run(context.Background(), task)
	if err != nil {
		log.Fatalf("Concept generation failed: %v", err)
	}

	// Pretty-print the generated JSON.
	var formatted json.RawMessage
	if err := json.Unmarshal(result.Output, &formatted); err != nil {
		log.Fatalf("Invalid result JSON: %v", err)
	}

	output, err := json.MarshalIndent(formatted, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Concept generation completed!")
	fmt.Printf("Task ID: %s\n", result.TaskID)
	fmt.Printf("Agent: %s\n", result.Agent)
	fmt.Printf("Status: %s\n\n", result.Status)
	fmt.Println(string(output))
}
