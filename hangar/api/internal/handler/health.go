package handler

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthHandler maneja el endpoint de health check
type HealthHandler struct {
	pool *pgxpool.Pool
}

// NewHealthHandler crea un nuevo health handler
func NewHealthHandler(pool *pgxpool.Pool) *HealthHandler {
	return &HealthHandler{pool: pool}
}

// Register registra la ruta de health
func (h *HealthHandler) Register(router fiber.Router) {
	router.Get("/health", h.Check)
}

// Check GET /health
func (h *HealthHandler) Check(c *fiber.Ctx) error {
	// Verifica que Postgres responde
	ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
	defer cancel()

	dbStatus := "ok"
	if err := h.pool.Ping(ctx); err != nil {
		dbStatus = "error: " + err.Error()
	}

	return c.JSON(fiber.Map{
		"status":   "ok",
		"postgres": dbStatus,
		"version":  "0.1.0",
	})
}
