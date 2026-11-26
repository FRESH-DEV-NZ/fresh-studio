package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Task struct {
	ID            string    `json:"id"`
	ProjectID     string    `json:"project_id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	AssignedAgent string    `json:"assigned_agent"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func newID() (string, error) {
	bytes := make([]byte, 16)

	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate ID: %w", err)
	}

	return hex.EncodeToString(bytes), nil
}

// Projects

func (s *Store) CreateProject(
	ctx context.Context,
	name string,
	description string,
) (*Project, error) {
	id, err := newID()
	if err != nil {
		return nil, err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO projects (id, name, description)
		VALUES (?, ?, ?)
	`, id, name, description)

	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}

	return s.GetProject(ctx, id)
}

func (s *Store) GetProject(
	ctx context.Context,
	id string,
) (*Project, error) {
	var p Project

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, status, created_at, updated_at
		FROM projects
		WHERE id = ?
	`, id).Scan(
		&p.ID,
		&p.Name,
		&p.Description,
		&p.Status,
		&p.CreatedAt,
		&p.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (s *Store) ListProjects(
	ctx context.Context,
) ([]Project, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, status, created_at, updated_at
		FROM projects
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []Project

	for rows.Next() {
		var p Project

		if err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Description,
			&p.Status,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, err
		}

		projects = append(projects, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return projects, nil
}

// Tasks

func (s *Store) CreateTask(
	ctx context.Context,
	projectID string,
	title string,
	description string,
) (*Task, error) {
	id, err := newID()
	if err != nil {
		return nil, err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO tasks (id, project_id, title, description)
		VALUES (?, ?, ?, ?)
	`, id, projectID, title, description)

	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	return s.GetTask(ctx, id)
}

func (s *Store) GetTask(
	ctx context.Context,
	id string,
) (*Task, error) {
	var t Task
	var assignedAgent sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, title, description,
		       assigned_agent, status, created_at, updated_at
		FROM tasks
		WHERE id = ?
	`, id).Scan(
		&t.ID,
		&t.ProjectID,
		&t.Title,
		&t.Description,
		&assignedAgent,
		&t.Status,
		&t.CreatedAt,
		&t.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if assignedAgent.Valid {
		t.AssignedAgent = assignedAgent.String
	}

	return &t, nil
}

func (s *Store) ListTasks(
	ctx context.Context,
	projectID string,
) ([]Task, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, project_id, title, description,
		       assigned_agent, status, created_at, updated_at
		FROM tasks
		WHERE project_id = ?
		ORDER BY created_at DESC
	`, projectID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task

	for rows.Next() {
		var t Task
		var assignedAgent sql.NullString

		if err := rows.Scan(
			&t.ID,
			&t.ProjectID,
			&t.Title,
			&t.Description,
			&assignedAgent,
			&t.Status,
			&t.CreatedAt,
			&t.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if assignedAgent.Valid {
			t.AssignedAgent = assignedAgent.String
		}

		tasks = append(tasks, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (s *Store) UpdateTaskStatus(
	ctx context.Context,
	id string,
	status string,
) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE tasks
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, status, id)

	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
