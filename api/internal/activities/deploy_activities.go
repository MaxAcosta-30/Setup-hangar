// api/internal/activities/deploy_activities.go
package activities

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.temporal.io/sdk/activity"

	"github.com/tu-usuario/hangar/api/internal/rdb"
	"github.com/tu-usuario/hangar/api/internal/workflows"
)

const buildsDir = "/tmp/hangar-builds"

// DeployActivities agrupa todas las actividades del workflow de deploy.
type DeployActivities struct {
	db     *pgxpool.Pool
	docker *dockerclient.Client
	redis  *redis.Client
}

// NewDeployActivities crea el struct con sus dependencias inyectadas.
func NewDeployActivities(db *pgxpool.Pool, docker *dockerclient.Client, redis *redis.Client) *DeployActivities {
	return &DeployActivities{db: db, docker: docker, redis: redis}
}

// -- Activity 1 ------------------------------------------------------------

func (a *DeployActivities) UpdateDeployStatus(ctx context.Context, deploymentID, status string) error {
	activity.GetLogger(ctx).Info("actualizando status",
		"deployment_id", deploymentID, "status", status,
	)

	_, err := a.db.Exec(ctx,
		`UPDATE deployments SET status = $1, updated_at = NOW() WHERE id = $2`,
		status, deploymentID,
	)
	if err != nil {
		return fmt.Errorf("error actualizando status: %w", err)
	}

	a.log(ctx, deploymentID, fmt.Sprintf("[hangar] deployment status: %s", status))
	return nil
}

// -- Activity 2 ------------------------------------------------------------

func (a *DeployActivities) CloneRepository(ctx context.Context, input workflows.DeployInput) error {
	workDir := filepath.Join(buildsDir, input.AppID)

	if err := os.RemoveAll(workDir); err != nil {
		return fmt.Errorf("error limpiando directorio previo: %w", err)
	}
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return fmt.Errorf("error creando directorio de build: %w", err)
	}

	a.log(ctx, input.DeploymentID,
		fmt.Sprintf("[hangar] clonando %s...", input.GitURL),
	)

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth=1", input.GitURL, workDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		a.log(ctx, input.DeploymentID,
			fmt.Sprintf("[hangar] error en git clone: %s", string(output)),
		)
		return fmt.Errorf("git clone falló: %w - output: %s", err, string(output))
	}

	dockerfilePath := filepath.Join(workDir, "Dockerfile")
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		a.log(ctx, input.DeploymentID,
			"[hangar] error: no se encontró Dockerfile en la raíz del repositorio",
		)
		return fmt.Errorf("Dockerfile no encontrado en %s", input.GitURL)
	}

	a.log(ctx, input.DeploymentID, "[hangar] repositorio clonado [done]")
	return nil
}

// -- Activity 3 ------------------------------------------------------------

func (a *DeployActivities) BuildDockerImage(ctx context.Context, input workflows.DeployInput) error {
	workDir   := filepath.Join(buildsDir, input.AppID)
	imageName := fmt.Sprintf("hangar/%s:latest", input.AppID)

	a.log(ctx, input.DeploymentID, "[hangar] iniciando docker build...")

	buildCtx, err := createTarContext(workDir)
	if err != nil {
		return fmt.Errorf("error creando contexto de build: %w", err)
	}
	defer buildCtx.Close()

	resp, err := a.docker.ImageBuild(ctx, buildCtx, types.ImageBuildOptions{
		Tags:       []string{imageName},
		Dockerfile: "Dockerfile",
		Remove:     true,
	})
	if err != nil {
		return fmt.Errorf("error iniciando docker build: %w", err)
	}
	defer resp.Body.Close()

	if err := a.streamBuildLogs(ctx, input.DeploymentID, resp.Body); err != nil {
		return err
	}

	a.log(ctx, input.DeploymentID,
		fmt.Sprintf("[hangar] imagen construida: %s [done]", imageName),
	)
	return nil
}

// -- Activity 4 ------------------------------------------------------------

func (a *DeployActivities) RunContainer(ctx context.Context, input workflows.DeployInput) error {
	imageName     := fmt.Sprintf("hangar/%s:latest", input.AppID)
	containerName := fmt.Sprintf("hangar-%s", input.AppID)

	a.stopAndRemoveContainer(ctx, containerName)
	a.log(ctx, input.DeploymentID, "[hangar] iniciando contenedor...")

	traefikNetwork := os.Getenv("TRAEFIK_NETWORK")
	if traefikNetwork == "" {
		traefikNetwork = "infra_default"
	}

	routerName := fmt.Sprintf("hangar-%s", input.AppID)
	labels := map[string]string{
		"traefik.enable": "true",
		fmt.Sprintf("traefik.http.routers.%s.rule", routerName):
			fmt.Sprintf("Host(`%s.hangar.local`)", input.Subdomain),
		fmt.Sprintf("traefik.http.routers.%s.entrypoints", routerName): "web",
		"hangar.app_id":     input.AppID,
		"hangar.managed_by": "hangar",
	}

	createResp, err := a.docker.ContainerCreate(
		ctx,
		&container.Config{Image: imageName, Labels: labels},
		&container.HostConfig{
			Resources: container.Resources{
				Memory:    128 * 1024 * 1024,
				CPUPeriod: 100000,
				CPUQuota:  25000,
			},
			RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
		},
		&network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				traefikNetwork: {},
			},
		},
		nil,
		containerName,
	)
	if err != nil {
		return fmt.Errorf("error creando contenedor: %w", err)
	}

	if err := a.docker.ContainerStart(ctx, createResp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("error arrancando contenedor: %w", err)
	}

	url := fmt.Sprintf("http://%s.hangar.local", input.Subdomain)
	a.log(ctx, input.DeploymentID,
		fmt.Sprintf("[hangar] app disponible en %s [done]", url),
	)
	return nil
}

// -- Helpers privados ------------------------------------------------------

// log guarda la línea en DB Y la publica en Redis.
// El WebSocket handler escucha Redis y la reenvía al browser al instante.
func (a *DeployActivities) log(ctx context.Context, deploymentID, message string) {
	// Persiste en DB para historial
	_, _ = a.db.Exec(ctx,
		`INSERT INTO deploy_logs (deployment_id, message) VALUES ($1, $2)`,
		deploymentID, message,
	)

	// Publica en Redis para que el WebSocket lo reenvíe en tiempo real
	_ = a.redis.Publish(ctx, rdb.LogChannel(deploymentID), message)
}

func (a *DeployActivities) stopAndRemoveContainer(ctx context.Context, name string) {
	_ = a.docker.ContainerStop(ctx, name, container.StopOptions{})
	_ = a.docker.ContainerRemove(ctx, name, container.RemoveOptions{Force: true})
}

func (a *DeployActivities) streamBuildLogs(ctx context.Context, deploymentID string, reader io.Reader) error {
	decoder := json.NewDecoder(reader)
	for {
		var msg jsonmessage.JSONMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error leyendo logs del build: %w", err)
		}
		if msg.Error != nil {
			a.log(ctx, deploymentID, fmt.Sprintf("[build] error: %s", msg.Error.Message))
			return fmt.Errorf("docker build falló: %s", msg.Error.Message)
		}
		if line := strings.TrimSpace(msg.Stream); line != "" {
			a.log(ctx, deploymentID, "[build] "+line)
		}
	}
	return nil
}

func createTarContext(srcDir string) (io.ReadCloser, error) {
	pr, pw := io.Pipe()
	go func() {
		tw := tar.NewWriter(pw)
		err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if strings.Contains(path, "/.git") {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			relPath, err := filepath.Rel(srcDir, path)
			if err != nil {
				return err
			}
			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}
			header.Name = relPath
			if err := tw.WriteHeader(header); err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return nil
			}
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(tw, file)
			return err
		})
		tw.Close()
		pw.CloseWithError(err)
	}()
	return pr, nil
}
