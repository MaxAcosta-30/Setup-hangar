package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/MaxAcosta-30/Setup-hangar/api/internal/models"
)

// AppRepository maneja todas las queries relacionadas a apps
type AppRepository struct {
	pool *pgxpool.Pool
}

// NewAppRepository crea un nuevo repositorio de apps
func NewAppRepository(pool *pgxpool.Pool) *AppRepository {
	return &AppRepository{pool: pool}
}

// Create inserta una nueva app en la base de datos
func (r *AppRepository) Create(ctx context.Context, name, gitURL, subdomain string) (*models.App, error) {
	query := `
		INSERT INTO apps (name, git_url, subdomain)
		VALUES ($1, $2, $3)
		RETURNING id, name, git_url, subdomain, status, created_at, updated_at
	`

	var app models.App
	err := r.pool.QueryRow(ctx, query, name, gitURL, subdomain).Scan(
		&app.ID,
		&app.Name,
		&app.GitURL,
		&app.Subdomain,
		&app.Status,
		&app.CreatedAt,
		&app.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("error creando app: %w", err)
	}

	return &app, nil
}

// GetByID obtiene una app por su UUID
func (r *AppRepository) GetByID(ctx context.Context, id string) (*models.App, error) {
	query := `
		SELECT id, name, git_url, subdomain, status, created_at, updated_at
		FROM apps
		WHERE id = $1
	`

	var app models.App
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&app.ID,
		&app.Name,
		&app.GitURL,
		&app.Subdomain,
		&app.Status,
		&app.CreatedAt,
		&app.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("app no encontrada: %w", err)
	}

	return &app, nil
}

// List devuelve todas las apps registradas
func (r *AppRepository) List(ctx context.Context) ([]models.App, error) {
	query := `
		SELECT id, name, git_url, subdomain, status, created_at, updated_at
		FROM apps
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error listando apps: %w", err)
	}
	defer rows.Close()

	var apps []models.App
	for rows.Next() {
		var app models.App
		if err := rows.Scan(
			&app.ID,
			&app.Name,
			&app.GitURL,
			&app.Subdomain,
			&app.Status,
			&app.CreatedAt,
			&app.UpdatedAt,
		); err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}

	return apps, nil
}

// UpdateStatus actualiza el status de una app
func (r *AppRepository) UpdateStatus(ctx context.Context, id string, status models.AppStatus) error {
	query := `
		UPDATE apps
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`

	_, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("error actualizando status: %w", err)
	}

	return nil
}

// SubdomainExists verifica si un subdominio ya está tomado
func (r *AppRepository) SubdomainExists(ctx context.Context, subdomain string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM apps WHERE subdomain = $1)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, subdomain).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// Pool expone el pool para uso en handlers que necesitan dependencias directas
func (r *AppRepository) Pool() *pgxpool.Pool {
    return r.pool
}

// Delete elimina una app y en cascada sus deployments y logs
func (r *AppRepository) Delete(ctx context.Context, id string) error {
    _, err := r.pool.Exec(ctx, `DELETE FROM apps WHERE id = $1`, id)
    return err
}
