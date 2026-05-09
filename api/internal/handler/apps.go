// api/internal/handler/apps.go
// — agrega DELETE /apps/:id al final, todo lo demás igual —
package handler

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"

	"github.com/MaxAcosta-30/hangar/api/internal/activities"
	"github.com/MaxAcosta-30/hangar/api/internal/db"
	dockerfactory "github.com/MaxAcosta-30/hangar/api/internal/docker"
	"github.com/MaxAcosta-30/hangar/api/internal/models"
	"github.com/MaxAcosta-30/hangar/api/internal/rdb"
	"github.com/MaxAcosta-30/hangar/api/internal/workflows"

	dockerclient "github.com/docker/docker/client"
	"github.com/redis/go-redis/v9"
)

type AppHandler struct {
	apps     *db.AppRepository
	deploys  *db.DeploymentRepository
	temporal client.Client
	docker   *dockerclient.Client
	redis    *redis.Client
	log      *zap.Logger
}

func NewAppHandler(
	apps *db.AppRepository,
	deploys *db.DeploymentRepository,
	temporal client.Client,
	docker *dockerclient.Client,
	redis *redis.Client,
	log *zap.Logger,
) *AppHandler {
	return &AppHandler{
		apps:     apps,
		deploys:  deploys,
		temporal: temporal,
		docker:   docker,
		redis:    redis,
		log:      log,
	}
}

func (h *AppHandler) Register(router fiber.Router) {
	router.Get("/apps", h.ListApps)
	router.Post("/apps", h.CreateApp)
	router.Get("/apps/:id", h.GetApp)
	router.Delete("/apps/:id", h.DeleteApp)
	router.Post("/apps/:id/deploy", h.TriggerDeploy)
	router.Get("/apps/:id/deployments", h.ListDeployments)
	router.Get("/deployments/:id/logs", h.GetDeployLogs)
}

func (h *AppHandler) ListApps(c *fiber.Ctx) error {
	apps, err := h.apps.List(c.Context())
	if err != nil {
		h.log.Error("error listando apps", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{"error": "error interno"})
	}
	return c.JSON(fiber.Map{"apps": apps, "total": len(apps)})
}

func (h *AppHandler) CreateApp(c *fiber.Ctx) error {
	var req models.CreateAppRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "body inválido"})
	}
	if req.Name == "" || req.GitURL == "" {
		return c.Status(400).JSON(fiber.Map{"error": "name y git_url son requeridos"})
	}

	subdomain := generateSubdomain(req.Name)
	exists, err := h.apps.SubdomainExists(c.Context(), subdomain)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "error interno"})
	}
	if exists {
		subdomain = fmt.Sprintf("%s-%s", subdomain, randomSuffix(4))
	}

	app, err := h.apps.Create(c.Context(), req.Name, req.GitURL, subdomain)
	if err != nil {
		h.log.Error("error creando app", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{"error": "error creando app"})
	}

	h.log.Info("app creada", zap.String("id", app.ID), zap.String("subdomain", app.Subdomain))
	return c.Status(201).JSON(models.CreateAppResponse{
		App:     *app,
		Message: fmt.Sprintf("App lista. Despliega con POST /api/v1/apps/%s/deploy", app.ID),
	})
}

func (h *AppHandler) GetApp(c *fiber.Ctx) error {
	app, err := h.apps.GetByID(c.Context(), c.Params("id"))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "app no encontrada"})
	}
	return c.JSON(app)
}

// DeleteApp DELETE /api/v1/apps/:id
// Detiene el contenedor, elimina la red aislada y borra el registro de DB.
func (h *AppHandler) DeleteApp(c *fiber.Ctx) error {
	id := c.Params("id")

	app, err := h.apps.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "app no encontrada"})
	}

	// Usa CleanupApp directamente (sin workflow — es una operación sincrónica simple)
	rdbClient, _ := rdb.New()
	defer rdbClient.Close()

	dockerClient, _ := dockerfactory.New()
	defer dockerClient.Close()

	pool := h.apps.Pool() // expón el pool en el repositorio
	act := activities.NewDeployActivities(pool, dockerClient, rdbClient)

	if err := act.CleanupApp(c.Context(), app.ID); err != nil {
		h.log.Error("error en cleanup", zap.String("app_id", app.ID), zap.Error(err))
		// No es fatal — el registro se borra aunque falle el cleanup de Docker
	}

	// Borra de DB (cascade elimina deployments y logs)
	if err := h.apps.Delete(c.Context(), id); err != nil {
		h.log.Error("error borrando app de DB", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{"error": "error eliminando app"})
	}

	h.log.Info("app eliminada", zap.String("id", id), zap.String("name", app.Name))
	return c.JSON(fiber.Map{"message": fmt.Sprintf("app '%s' eliminada", app.Name)})
}

func (h *AppHandler) TriggerDeploy(c *fiber.Ctx) error {
	id := c.Params("id")

	app, err := h.apps.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "app no encontrada"})
	}

	var req models.TriggerDeployRequest
	c.BodyParser(&req)

	deployment, err := h.deploys.Create(c.Context(), app.ID, req.CommitSha)
	if err != nil {
		h.log.Error("error creando deployment", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{"error": "error iniciando deploy"})
	}

	input := workflows.DeployInput{
		DeploymentID: deployment.ID,
		AppID:        app.ID,
		AppName:      app.Name,
		GitURL:       app.GitURL,
		Subdomain:    app.Subdomain,
		CommitSha:    req.CommitSha,
	}

	we, err := h.temporal.ExecuteWorkflow(
		c.Context(),
		client.StartWorkflowOptions{
			ID:        fmt.Sprintf("deploy-%s-%s", app.ID, deployment.ID),
			TaskQueue: "hangar-deploy",
		},
		workflows.DeployWorkflow,
		input,
	)
	if err != nil {
		h.log.Error("error disparando workflow", zap.Error(err))
		_ = h.deploys.UpdateStatus(c.Context(), deployment.ID, models.DeploymentStatusFailed)
		return c.Status(500).JSON(fiber.Map{"error": "error iniciando workflow"})
	}

	h.log.Info("workflow iniciado",
		zap.String("workflow_id", we.GetID()),
		zap.String("app", app.Name),
	)

	return c.Status(202).JSON(fiber.Map{
		"deployment":  deployment,
		"workflow_id": we.GetID(),
		"run_id":      we.GetRunID(),
		"message":     "Deploy iniciado — sigue el progreso en http://localhost:8088",
	})
}

func (h *AppHandler) ListDeployments(c *fiber.Ctx) error {
	id := c.Params("id")
	if _, err := h.apps.GetByID(c.Context(), id); err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "app no encontrada"})
	}
	deployments, err := h.deploys.ListByApp(c.Context(), id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "error interno"})
	}
	return c.JSON(fiber.Map{"deployments": deployments, "total": len(deployments)})
}

func (h *AppHandler) GetDeployLogs(c *fiber.Ctx) error {
	logs, err := h.deploys.GetLogs(c.Context(), c.Params("id"))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "deployment no encontrado"})
	}
	return c.JSON(fiber.Map{"logs": logs, "total": len(logs)})
}

func generateSubdomain(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, " ", "-")
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	return strings.Trim(result.String(), "-")
}

func randomSuffix(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[i%len(chars)]
	}
	return string(b)
}
