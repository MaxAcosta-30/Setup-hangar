// api/internal/rdb/client.go
package rdb

import (
	"context"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

// New crea y verifica un cliente de Redis listo para usar.
func New() (*redis.Client, error) {
	addr := os.Getenv("REDIS_URL")
	if addr == "" {
		addr = "localhost:6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis no responde en %s: %w", addr, err)
	}

	return client, nil
}

// LogChannel devuelve el nombre del canal Pub/Sub para un deployment.
// Tanto las actividades (al publicar) como el handler WS (al suscribirse)
// deben usar esta función para garantizar que el canal sea el mismo.
func LogChannel(deploymentID string) string {
	return fmt.Sprintf("hangar:logs:%s", deploymentID)
}
