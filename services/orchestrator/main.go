package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/agents/conceptdesigner"
	"github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/api"
	"github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/database"
	"github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/modelrouter"
	"github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/promptloader"
	"github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/store"

	"github.com/joho/godotenv"
)

type HealthResponse struct {
	Status    string    `json:"status"`
	Service   string    `json:"service"`
	Timestamp time.Time `json:"timestamp"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := HealthResponse{
		Status:    "healthy",
		Service:   "studio-orchestrator",
		Timestamp: time.Now().UTC(),
	}

	json.NewEncoder(w).Encode(response)
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler)

	dbPath := filepath.Join("data", "studio.db")

	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		log.Fatal(err)
	}

	db, err := database.New(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		log.Fatal(err)
	}

	host := os.Getenv("OLLAMA_HOST")
	if host == "" {
		host = "http://localhost:11434"
	}

	modelClient, err := modelrouter.NewClient(host, os.Getenv("OLLAMA_MODEL"))
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Studio database connected")

	promptLoader := promptloader.New("prompts")

	conceptDesigner := conceptdesigner.NewConceptDesigner(modelClient, promptLoader)
	appStore := store.New(db)
	apiHandler := api.NewHandler(appStore, conceptDesigner)
	apiHandler.RegisterRoutes(mux)

	server := &http.Server{
		Addr:         "127.0.0.1:8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 4 * time.Minute,
	}

	log.Println("Studio orchestrator starting on :8080")

	log.Fatal(server.ListenAndServe())
}
