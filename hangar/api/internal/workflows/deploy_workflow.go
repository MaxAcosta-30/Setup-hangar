// api/internal/workflows/deploy_workflow.go
package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// DeployInput contiene todo lo que el workflow necesita saber
type DeployInput struct {
	DeploymentID string
	AppID        string
	AppName      string
	GitURL       string
	Subdomain    string
	CommitSha    string
}

// DeployWorkflow orquesta el ciclo de vida completo de un deploy.
//
// REGLA CRÍTICA DE TEMPORAL: este archivo no puede tener I/O,
// llamadas de red, time.Now(), ni rand. Todo eso va en Activities.
// El workflow solo orquesta - llama actividades en orden y maneja errores.
func DeployWorkflow(ctx workflow.Context, input DeployInput) error {
	log := workflow.GetLogger(ctx)
	log.Info("deploy iniciado",
		"app", input.AppName,
		"deployment_id", input.DeploymentID,
	)

	// Opciones que aplican a todas las actividades
	// StartToCloseTimeout: tiempo máximo que puede tardar UNA actividad
	// RetryPolicy: si falla, reintenta con backoff exponencial
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts:    3,
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// -- Activity 1: Marca el deployment como "running" en DB ----------
	if err := workflow.ExecuteActivity(ctx, UpdateDeployStatus,
		input.DeploymentID, "running",
	).Get(ctx, nil); err != nil {
		return err
	}

	// -- Activity 2: Clona el repositorio ------------------------------
	if err := workflow.ExecuteActivity(ctx, CloneRepository,
		input,
	).Get(ctx, nil); err != nil {
		// Si falla, marcamos el deployment como fallido antes de salir
		_ = workflow.ExecuteActivity(ctx, UpdateDeployStatus,
			input.DeploymentID, "failed",
		).Get(ctx, nil)
		return err
	}

	// -- Activity 3: Construye la imagen Docker ------------------------─
	if err := workflow.ExecuteActivity(ctx, BuildDockerImage,
		input,
	).Get(ctx, nil); err != nil {
		_ = workflow.ExecuteActivity(ctx, UpdateDeployStatus,
			input.DeploymentID, "failed",
		).Get(ctx, nil)
		return err
	}

	// -- Activity 4: Corre el contenedor con labels de Traefik --------─
	if err := workflow.ExecuteActivity(ctx, RunContainer,
		input,
	).Get(ctx, nil); err != nil {
		_ = workflow.ExecuteActivity(ctx, UpdateDeployStatus,
			input.DeploymentID, "failed",
		).Get(ctx, nil)
		return err
	}

	// -- Activity 5: Marca como exitoso --------------------------------
	_ = workflow.ExecuteActivity(ctx, UpdateDeployStatus,
		input.DeploymentID, "success",
	).Get(ctx, nil)

	log.Info("deploy completado",
		"app", input.AppName,
		"url", input.Subdomain+".hangar.local",
	)
	return nil
}

// Estas son las firmas de las actividades que el workflow llama.
// Las implementaciones reales están en activities/deploy_activities.go
// Temporal usa estas firmas para el registro - deben coincidir exactamente.
var (
	UpdateDeployStatus = "UpdateDeployStatus"
	CloneRepository    = "CloneRepository"
	BuildDockerImage   = "BuildDockerImage"
	RunContainer       = "RunContainer"
)
