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
	containertypes "github.com/docker/docker/api/types/container"
	networktypes "github.com/docker/docker/api/types/network"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.temporal.io/sdk/activity"

	"github.com/MaxAcosta-30/hangar/api/internal/rdb"
	"github.com/MaxAcosta-30/hangar/api/internal/workflows"
)

const buildsDir = "/tmp/hangar-builds"

type DeployActivities struct {
	db     *pgxpool.Pool
	docker *dockerclient.Client
	redis  *redis.Client
}

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

	a.log(ctx, input.DeploymentID, fmt.Sprintf("[hangar] clonando %s...", input.GitURL))

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth=1", input.GitURL, workDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		a.log(ctx, input.DeploymentID, fmt.Sprintf("[hangar] error en git clone: %s", string(output)))
		return fmt.Errorf("git clone falló: %w", err)
	}

	if _, err := os.Stat(filepath.Join(workDir, "Dockerfile")); os.IsNotExist(err) {
		a.log(ctx, input.DeploymentID, "[hangar] error: Dockerfile no encontrado en la raíz")
		return fmt.Errorf("Dockerfile no encontrado en %s", input.GitURL)
	}

	a.log(ctx, input.DeploymentID, "[hangar] repositorio clonado [DONE]")
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

	a.log(ctx, input.DeploymentID, fmt.Sprintf("[hangar] imagen construida: %s [DONE]", imageName))
	return nil
}

// -- Activity 4 ------------------------------------------------------------

// RunContainer crea una red aislada por app, arranca el contenedor en esa red,
// y luego lo conecta a la red de Traefik para que el routing funcione.
//
// Por qué dos redes y no una:
//   - Red app (hangar-app-{id}): aislada por app -- App A no puede hacer
//     requests directos a App B aunque conozca su IP interna.
//   - Red Traefik (infra_default): compartida solo con el proxy --
//     Traefik puede alcanzar el contenedor para hacer el routing.
//
// El resultado: el único camino entre apps es a través de sus subdominios
// públicos (HTTP via Traefik), nunca por red interna directa.
func (a *DeployActivities) RunContainer(ctx context.Context, input workflows.DeployInput) error {
	imageName     := fmt.Sprintf("hangar/%s:latest", input.AppID)
	containerName := fmt.Sprintf("hangar-%s", input.AppID)
	appNetwork    := appNetworkName(input.AppID)

	traefikNetwork := os.Getenv("TRAEFIK_NETWORK")
	if traefikNetwork == "" {
		traefikNetwork = "infra_default"
	}

	// Limpia contenedor anterior si existe (re-deploy)
	a.stopAndRemoveContainer(ctx, containerName)

	// -- Crea la red aislada de la app ---------------------------------
	// Si ya existe (deploy anterior fallido a medias), la reutilizamos
	a.log(ctx, input.DeploymentID,
		fmt.Sprintf("[hangar] creando red aislada: %s...", appNetwork),
	)
	_, err := a.docker.NetworkCreate(ctx, appNetwork, networktypes.CreateOptions{
		Driver:     "bridge",
		// Internal: false -- la app SÍ puede salir a internet.
		// El aislamiento es lateral (entre apps), no vertical (hacia internet).
		Internal:   false,
		Labels: map[string]string{
			"hangar.managed_by": "hangar",
			"hangar.app_id":     input.AppID,
		},
	})
	// Si la red ya existe, NetworkCreate devuelve error -- lo ignoramos
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("error creando red de app: %w", err)
	}

	// -- Labels para que Traefik detecte automáticamente el routing -----
	routerName := fmt.Sprintf("hangar-%s", input.AppID)
	labels := map[string]string{
		"traefik.enable": "true",
		fmt.Sprintf("traefik.http.routers.%s.rule", routerName):
			fmt.Sprintf("Host(`%s.hangar.local`)", input.Subdomain),
		fmt.Sprintf("traefik.http.routers.%s.entrypoints", routerName): "web",
		"hangar.app_id":     input.AppID,
		"hangar.managed_by": "hangar",
	}

	// -- Crea el contenedor conectado SOLO a la red de app -------------
	// Traefik se conectará en el siguiente paso via NetworkConnect
	a.log(ctx, input.DeploymentID, "[hangar] iniciando contenedor...")
	createResp, err := a.docker.ContainerCreate(
		ctx,
		&containertypes.Config{
			Image:  imageName,
			Labels: labels,
		},
		&containertypes.HostConfig{
			Resources: containertypes.Resources{
				Memory:    128 * 1024 * 1024, // 128 MB
				CPUPeriod: 100000,
				CPUQuota:  25000,             // 25% de un core
			},
			RestartPolicy: containertypes.RestartPolicy{Name: "unless-stopped"},
		},
		&networktypes.NetworkingConfig{
			EndpointsConfig: map[string]*networktypes.EndpointSettings{
				appNetwork: {},
			},
		},
		nil,
		containerName,
	)
	if err != nil {
		return fmt.Errorf("error creando contenedor: %w", err)
	}

	// -- Arranca el contenedor ------------------------------------------
	if err := a.docker.ContainerStart(ctx, createResp.ID, containertypes.StartOptions{}); err != nil {
		return fmt.Errorf("error arrancando contenedor: %w", err)
	}

	// -- Conecta el contenedor a la red de Traefik ---------------------
	// Se hace POST-start porque ContainerCreate solo acepta una red.
	// Traefik detecta el nuevo contenedor en su red y crea el routing automáticamente.
	if err := a.docker.NetworkConnect(ctx, traefikNetwork, createResp.ID, nil); err != nil {
		// No es fatal -- el contenedor corre, solo falla el routing público
		a.log(ctx, input.DeploymentID,
			fmt.Sprintf("[hangar] advertencia: no se pudo conectar a red Traefik: %v", err),
		)
	}

	url := fmt.Sprintf("http://%s.hangar.local", input.Subdomain)
	a.log(ctx, input.DeploymentID,
		fmt.Sprintf("[hangar] app disponible en %s [DONE]", url),
	)
	return nil
}

// -- Activity 5 (nueva): CleanupApp ---------------------------------------

// CleanupApp detiene y elimina el contenedor y la red aislada de una app.
// Se llama cuando el usuario elimina una app desde el dashboard.
func (a *DeployActivities) CleanupApp(ctx context.Context, appID string) error {
	containerName := fmt.Sprintf("hangar-%s", appID)
	appNetwork    := appNetworkName(appID)

	a.stopAndRemoveContainer(ctx, containerName)

	// Elimina la red aislada de la app
	if err := a.docker.NetworkRemove(ctx, appNetwork); err != nil {
		if !strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("error eliminando red %s: %w", appNetwork, err)
		}
	}

	return nil
}

// -- Helpers privados ------------------------------------------------------

func (a *DeployActivities) log(ctx context.Context, deploymentID, message string) {
	_, _ = a.db.Exec(ctx,
		`INSERT INTO deploy_logs (deployment_id, message) VALUES ($1, $2)`,
		deploymentID, message,
	)
	_ = a.redis.Publish(ctx, rdb.LogChannel(deploymentID), message)
}

func (a *DeployActivities) stopAndRemoveContainer(ctx context.Context, name string) {
	_ = a.docker.ContainerStop(ctx, name, containertypes.StopOptions{})
	_ = a.docker.ContainerRemove(ctx, name, containertypes.RemoveOptions{Force: true})
}

func (a *DeployActivities) streamBuildLogs(ctx context.Context, deploymentID string, reader io.Reader) error {
	decoder := json.NewDecoder(reader)
	for {
		var msg jsonmessage.JSONMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error leyendo logs: %w", err)
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

// appNetworkName genera el nombre de red aislada para una app.
// Centralizado aquí para que CleanupApp y RunContainer usen siempre el mismo formato.
func appNetworkName(appID string) string {
	return fmt.Sprintf("hangar-app-%s", appID)
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
