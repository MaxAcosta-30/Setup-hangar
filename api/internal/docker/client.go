// api/internal/docker/client.go
package docker

import (
	"fmt"

	"github.com/docker/docker/client"
)

// New crea un cliente de Docker Engine conectado al socket local.
// Por defecto usa /var/run/docker.sock - el mismo que usa Traefik.
func New() (*client.Client, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(), // negocia la versión automáticamente
	)
	if err != nil {
		return nil, fmt.Errorf("no se pudo conectar a Docker Engine: %w", err)
	}

	return cli, nil
}
