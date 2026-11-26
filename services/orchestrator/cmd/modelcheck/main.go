package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/modelrouter"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Could not load .env: %v (using process environment)", err)
	}

	host := os.Getenv("OLLAMA_HOST")
	if host == "" {
		host = "http://localhost:11434"
	}

	model := strings.TrimSpace(os.Getenv("OLLAMA_MODEL"))
	if model == "" {
		log.Fatal("OLLAMA_MODEL must be set to an installed Ollama model")
	}

	client, err := modelrouter.NewClient(host, model)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		3*time.Minute,
	)
	defer cancel()

	fmt.Println("Sending test prompt to:", client.Model())

	response, err := client.Chat(ctx, []modelrouter.Message{
		{
			Role:    "system",
			Content: "You are a helpful game design assistant. Be concise.",
		},
		{
			Role:    "user",
			Content: "Suggest one original casual mobile game concept in two sentences.",
		},
	})

	if err != nil {
		log.Fatal("Model request failed: ", err)
	}

	fmt.Println("\nModel:", response.Model)
	fmt.Println("\nResponse:")
	fmt.Println(response.Message.Content)
}
