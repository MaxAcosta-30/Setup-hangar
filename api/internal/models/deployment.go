package models

import (
	"time"
)

// DeploymentStatus representa el estado de un deployment
type DeploymentStatus string

const (
	DeploymentStatusPending  DeploymentStatus = "pending"
	DeploymentStatusRunning  DeploymentStatus = "running"
	DeploymentStatusSuccess  DeploymentStatus = "success"
	DeploymentStatusFailed   DeploymentStatus = "failed"
	DeploymentStatusCanceled DeploymentStatus = "canceled"
)

// Deployment representa un intento de deploy de una app
type Deployment struct {
	ID        string           `json:"id" db:"id"`
	AppID     string           `json:"app_id" db:"app_id"`
	CommitSha string           `json:"commit_sha" db:"commit_sha"`
	Status    DeploymentStatus `json:"status" db:"status"`
	CreatedAt time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt time.Time        `json:"updated_at" db:"updated_at"`
}

// DeployLog es una línea de log de un deployment
type DeployLog struct {
	ID           int64     `json:"id" db:"id"`
	DeploymentID string    `json:"deployment_id" db:"deployment_id"`
	Message      string    `json:"message" db:"message"`
	LoggedAt     time.Time `json:"logged_at" db:"logged_at"`
}

// TriggerDeployRequest es el body esperado en POST /apps/:id/deploy
type TriggerDeployRequest struct {
	CommitSha string `json:"commit_sha"`
}
