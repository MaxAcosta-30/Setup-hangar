package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// Logger es middleware que registra cada request con zap
func Logger(log *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Procesa el request
		err := c.Next()

		// Loguea el resultado
		log.Info("request",
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Int("status", c.Response().StatusCode()),
			zap.Duration("latency", time.Since(start)),
			zap.String("ip", c.IP()),
		)

		return err
	}
}

// ErrorHandler maneja errores no capturados devolviendo JSON limpio
func ErrorHandler(log *zap.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError
		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
		}

		log.Error("error no manejado",
			zap.Error(err),
			zap.String("path", c.Path()),
		)

		return c.Status(code).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
}
