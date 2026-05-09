package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/MaxAcosta-30/hangar/api/internal/models"
)

// DeploymentRepository maneja queries de deployments
type DeploymentRepository struct {
	pool *pgxpool.Pool
}

// NewDeploymentRepository crea un nuevo repositorio de deployments
func NewDeploymentRepository(pool *pgxpool.Pool) *DeploymentRepository {
	return &DeploymentRepository{pool: pool}
}

// Create inserta un nuevo deployment
func (r *DeploymentRepository) Create(ctx context.Context, appID, commitSha string) (*models.Deployment, error) {
	query := `
		INSERT INTO deployments (app_id, commit_sha)
		VALUES ($1, $2)
		RETURNING id, app_id, commit_sha, status, created_at, updated_at
	`

	var d models.Deployment
	err := r.pool.QueryRow(ctx, query, appID, commitSha).Scan(
		&d.ID,
		&d.AppID,
		&d.CommitSha,
		&d.Status,
		&d.CreatedAt,
		&d.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("error creando deployment: %w", err)
	}

	return &d, nil
}

// GetByID obtiene un deployment por su UUID
func (r *DeploymentRepository) GetByID(ctx context.Context, id string) (*models.Deployment, error) {
	query := `
		SELECT id, app_id, commit_sha, status, created_at, updated_at
		FROM deployments
		WHERE id = $1
	`

	var d models.Deployment
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&d.ID,
		&d.AppID,
		&d.CommitSha,
		&d.Status,
		&d.CreatedAt,
		&d.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("deployment no encontrado: %w", err)
	}

	return &d, nil
}

// ListByApp devuelve todos los deployments de una app
func (r *DeploymentRepository) ListByApp(ctx context.Context, appID string) ([]models.Deployment, error) {
	query := `
		SELECT id, app_id, commit_sha, status, created_at, updated_at
		FROM deployments
		WHERE app_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, appID)
	if err != nil {
		return nil, fmt.Errorf("error listando deployments: %w", err)
	}
	defer rows.Close()

	var deployments []models.Deployment
	for rows.Next() {
		var d models.Deployment
		if err := rows.Scan(
			&d.ID,
			&d.AppID,
			&d.CommitSha,
			&d.Status,
			&d.CreatedAt,
			&d.UpdatedAt,
		); err != nil {
			return nil, err
		}
		deployments = append(deployments, d)
	}

	return deployments, nil
}

// UpdateStatus actualiza el status de un deployment
func (r *DeploymentRepository) UpdateStatus(ctx context.Context, id string, status models.DeploymentStatus) error {
	query := `
		UPDATE deployments
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`

	_, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("error actualizando status: %w", err)
	}

	return nil
}

// InsertLog guarda una línea de log de un deployment
func (r *DeploymentRepository) InsertLog(ctx context.Context, deploymentID, message string) error {
	query := `
		INSERT INTO deploy_logs (deployment_id, message)
		VALUES ($1, $2)
	`

	_, err := r.pool.Exec(ctx, query, deploymentID, message)
	if err != nil {
		return fmt.Errorf("error guardando log: %w", err)
	}

	return nil
}

// GetLogs devuelve los logs de un deployment
func (r *DeploymentRepository) GetLogs(ctx context.Context, deploymentID string) ([]models.DeployLog, error) {
	query := `
		SELECT id, deployment_id, message, logged_at
		FROM deploy_logs
		WHERE deployment_id = $1
		ORDER BY logged_at ASC
	`

	rows, err := r.pool.Query(ctx, query, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo logs: %w", err)
	}
	defer rows.Close()

	var logs []models.DeployLog
	for rows.Next() {
		var l models.DeployLog
		if err := rows.Scan(&l.ID, &l.DeploymentID, &l.Message, &l.LoggedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}

	return logs, nil
}
