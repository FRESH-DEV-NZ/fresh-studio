package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/agents"
	"github.com/FRESH-DEV-NZ/fresh-studio/services/orchestrator/internal/store"
)

type Handler struct {
	store           *store.Store
	conceptDesigner agent.Agent
	conceptSlot     chan struct{}
}

func NewHandler(s *store.Store, conceptDesigner agent.Agent) *Handler {
	return &Handler{
		store:           s,
		conceptDesigner: conceptDesigner,
		conceptSlot:     make(chan struct{}, 1),
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /concepts", h.createConcept)
	mux.HandleFunc("POST /projects", h.createProject)
	mux.HandleFunc("GET /projects", h.listProjects)
	mux.HandleFunc("GET /projects/{id}", h.getProject)

	mux.HandleFunc("POST /projects/{id}/tasks", h.createTask)
	mux.HandleFunc("GET /projects/{id}/tasks", h.listTasks)

	mux.HandleFunc("PATCH /tasks/{id}", h.updateTask)
}

func (h *Handler) createConcept(w http.ResponseWriter, r *http.Request) {
	var task agent.Task
	if err := decodeJSON(w, r, &task); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON request")
		return
	}
	if strings.TrimSpace(task.Objective) == "" {
		writeError(w, http.StatusBadRequest, "Task objective is required")
		return
	}

	select {
	case h.conceptSlot <- struct{}{}:
		defer func() { <-h.conceptSlot }()
	default:
		writeError(w, http.StatusTooManyRequests, "Concept designer is busy")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()
	log.Printf("Concept design started: task_id=%q", task.ID)
	result, err := h.conceptDesigner.Run(ctx, task)
	if err != nil {
		log.Printf("Concept design failed: task_id=%q error=%v", task.ID, err)
		if errors.Is(err, context.DeadlineExceeded) {
			writeError(w, http.StatusGatewayTimeout, "Concept design timed out")
			return
		}
		writeError(w, http.StatusInternalServerError, "Could not generate concept")
		return
	}
	log.Printf("Concept design completed: task_id=%q model=%q", task.ID, result.Model)
	writeJSON(w, http.StatusOK, result)
}

type createProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type createTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type updateTaskRequest struct {
	Status string `json:"status"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return err
	}

	// Reject requests containing multiple JSON values.
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("request must contain exactly one JSON object")
	}

	return nil
}

// POST /projects
func (h *Handler) createProject(w http.ResponseWriter, r *http.Request) {
	var req createProjectRequest

	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON request")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "Project name is required")
		return
	}

	project, err := h.store.CreateProject(
		r.Context(),
		req.Name,
		req.Description,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not create project")
		return
	}

	writeJSON(w, http.StatusCreated, project)
}

// GET /projects
func (h *Handler) listProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.store.ListProjects(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not retrieve projects")
		return
	}

	writeJSON(w, http.StatusOK, projects)
}

// GET /projects/{id}
func (h *Handler) getProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	project, err := h.store.GetProject(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Project not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not retrieve project")
		return
	}

	writeJSON(w, http.StatusOK, project)
}

// POST /projects/{id}/tasks
func (h *Handler) createTask(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	// Confirm that the parent project exists.
	_, err := h.store.GetProject(r.Context(), projectID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Project not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not retrieve project")
		return
	}

	var req createTaskRequest

	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON request")
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)

	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "Task title is required")
		return
	}

	task, err := h.store.CreateTask(
		r.Context(),
		projectID,
		req.Title,
		req.Description,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not create task")
		return
	}

	writeJSON(w, http.StatusCreated, task)
}

// GET /projects/{id}/tasks
func (h *Handler) listTasks(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	_, err := h.store.GetProject(r.Context(), projectID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Project not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not retrieve project")
		return
	}

	tasks, err := h.store.ListTasks(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not retrieve tasks")
		return
	}

	writeJSON(w, http.StatusOK, tasks)
}

// PATCH /tasks/{id}
func (h *Handler) updateTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")

	var req updateTaskRequest

	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON request")
		return
	}

	req.Status = strings.TrimSpace(req.Status)

	validStatuses := map[string]bool{
		"pending":     true,
		"in_progress": true,
		"completed":   true,
		"failed":      true,
		"cancelled":   true,
	}

	if !validStatuses[req.Status] {
		writeError(w, http.StatusBadRequest, "Invalid task status")
		return
	}

	err := h.store.UpdateTaskStatus(r.Context(), taskID, req.Status)

	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Task not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not update task")
		return
	}

	task, err := h.store.GetTask(r.Context(), taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not retrieve updated task")
		return
	}

	writeJSON(w, http.StatusOK, task)
}
