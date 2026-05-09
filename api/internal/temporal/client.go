// api/internal/temporal/client.go
package temporalclient

import (
	"fmt"
	"os"

	"go.temporal.io/sdk/client"
)

// TaskQueue es el nombre de la cola — worker y API deben usar el mismo
const TaskQueue = "hangar-deploy"

// New crea y devuelve un cliente de Temporal listo para usar
func New() (client.Client, error) {
	host := os.Getenv("TEMPORAL_HOST")
	if host == "" {
		host = "localhost:7233"
	}

	c, err := client.Dial(client.Options{
		HostPort: host,
	})
	if err != nil {
		return nil, fmt.Errorf("no se pudo conectar a Temporal: %w", err)
	}

	return c, nil
}
