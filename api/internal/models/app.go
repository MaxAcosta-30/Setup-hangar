package models

import (
	"time"
)

// AppStatus representa el estado actual de una app
type AppStatus string

const (
	AppStatusIdle     AppStatus = "idle"
	AppStatusBuilding AppStatus = "building"
	AppStatusRunning  AppStatus = "running"
	AppStatusFailed   AppStatus = "failed"
	AppStatusStopped  AppStatus = "stopped"
)

// App representa una aplicación registrada en Hangar
type App struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	GitURL    string    `json:"git_url" db:"git_url"`
	Subdomain string    `json:"subdomain" db:"subdomain"`
	Status    AppStatus `json:"status" db:"status"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// CreateAppRequest es el body esperado en POST /apps
type CreateAppRequest struct {
	Name   string `json:"name" validate:"required,min=2,max=100"`
	GitURL string `json:"git_url" validate:"required,url"`
}

// CreateAppResponse es lo que devuelve POST /apps
type CreateAppResponse struct {
	App     App    `json:"app"`
	Message string `json:"message"`
}
